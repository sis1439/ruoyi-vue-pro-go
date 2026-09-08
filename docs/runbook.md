# runbook.md — 初始化、升级、恢复与排障

> 适用范围：批次 C/D 交付后的状态。
> **数据库仍是 MySQL**（PostgreSQL 适配属批次 A，见 `blockers.md` PRE-02），
> 本手册中涉及 PostgreSQL 的步骤为 CI 的集成测试环境，不是应用运行环境。

## 1. 构建与本地启动

```bash
make gen        # 必须：internal/repo/query 未入库
make build      # 产物 ./server
make test       # 单元测试，不依赖数据库/Redis
make vet
make ci         # gen + build + vet + test
```

配置文件放在 `config/config.<env>.yaml`（`GO_ENV` 决定，默认 `local`）。
`config/*.local.yaml` 已被 `.gitignore` 忽略。

### 必填配置（缺任一项启动即失败）

```yaml
app:
  env: local              # 非 local 时，jwt_secret 与 pay.notify_token 变为必填
  jwt_secret: ""          # 非 local 必填；留空时用内置默认值，仅限本地
mysql:
  dsn: "user:pass@tcp(127.0.0.1:3306)/yudao?parseTime=true"
redis:
  addr: "127.0.0.1:6379"
pay:
  order_notify_url:  "https://api.example.com/admin-api/pay/notify/order"
  refund_notify_url: "https://api.example.com/admin-api/pay/notify/refund"
  notify_token: ""        # 非 local 必填：支付中心 → 商城的内部调用令牌
```

启动期校验在 `pkg/config/config.go` 的 `Config.Validate()`。
非 local 环境额外拒绝：空 `jwt_secret`、空 `notify_token`、`http://` 的回调地址。

## 2. 升级与回滚

* **迁移**：尚无迁移器（批次 A）。当前升级 = 替换二进制 + 重启。
* **回滚**：本次改动中 **ADR-002 改变了 `pay_order.merchant_order_id` 的语义**。
  升级后新建的支付单用交易主键，回滚到旧版本会导致这些支付单的回调无法归位。
  **回滚前须确认没有处于"已创建未完成"的支付单**，或先手工转换该字段。
* **优雅退出**：进程收到 SIGTERM 后停止接收新请求，等待在途请求最多 15s。
  未完成的通知任务由 `pay_notify_task` 表持久化，重启后由定时任务继续。

## 3. 排障

### 支付拉不起来
1. 看日志有无 `暂不支持的微信支付渠道: <code>` → 渠道编码未在 `payMethodOf` 中，
   或数据库里的 `pay_channel.code` 拼错。
2. `JSAPI 支付需要 openid` → 前端未传 `channelExtras.openid`，
   或用户未绑定微信。前端在 `sheep/platform/pay.js` 会引导绑定。
3. 前端 `JSON.parse` 报错 → 检查 `displayContent` 是否为对象字符串。
   `wx_lite`/`wx_pub` 应含 `timeStamp`/`nonceStr`/`packageValue`/`signType`/`paySign`。

### 支付成功但订单还是待支付
按顺序排查：
1. **渠道回调是否到达**：查 `/admin-api/pay/notify/order/:channelId` 的访问日志。
   没到 → 检查 `pay.order_notify_url` 与回调域名备案。
2. **回调是否被拒**：日志 `渠道编号找不到对应的支付客户端` → 渠道被停用或配置损坏；
   `回调商户号不匹配` / `回调 AppID 不匹配` → 回调来自其它商户；
   `支付金额不匹配` → 渠道实收与支付单金额不一致，**须人工核对，不要直接改库**。
3. **支付单是否已成功**：`select status from pay_order where id = ?`，10 = 成功。
4. **业务通知任务**：`select * from pay_notify_task where data_id = ? and type = 1`。
   * `status = 0` 且 `next_notify_time` 在未来 → 正在退避重试，等待即可。
   * `status = 20` → 重试耗尽，见"人工重放"。
   * `notify_times` 一直不涨 → 检查定时任务是否注册、Redis 锁是否可用。
5. **通知日志**：`select * from pay_notify_log where task_id = ? order by id desc`。
   `response` 里是商城的原始响应，`{"code":非0}` 说明商城侧拒绝了，看 msg。

### 人工重放通知任务
```sql
-- 确认任务与目标订单，再执行
UPDATE pay_notify_task
SET status = 0, notify_times = 0, next_notify_time = NOW()
WHERE id = ?;
```
重放是安全的：接收端对同一订单的重复通知幂等
（`PayOrderProcessor` 的条件状态转换 + `RowsAffected` 检查）。

### 主动查单
前端 `/pay/order/get?id=X&sync=true` 与 `/trade/order/get-detail?id=X&sync=true`
都会触发一次渠道查单，**每个订单 3 秒一次**（Redis 闸门）。
Redis 不可用时闸门不放行，此时只能等渠道回调。

## 4. 需要告警的场景（T13 最低覆盖）

当前实现状态：**均为日志级别，未接告警系统**（见 `blockers.md` TODO-02）。

| 场景 | 日志特征 | 建议阈值 |
|---|---|---|
| 已收款但商城未更新 | `pay_order.status=10` 且对应 `trade_order.status=0` 超过 5 分钟 | 任意 1 单 |
| 通知重试耗尽 | `pay notify retries exhausted, manual replay required` | 任意 1 条 |
| 待付款异常占库 | `trade_order.status=0` 且 `create_time` 超过支付超时 2 倍 | 需查询 |
| 退款长时间处理中 | `pay_refund.status=0` 超过 24 小时 | 任意 1 单 |
| 任务堆积 | `pay_notify_task.status=0 and next_notify_time < now()` 数量 | > 100 |
| 认证依赖故障 | Redis 连接错误日志 | 任意 |
| 数据库错误 | `save pay notify task failed` 等 | 任意 |

日志已包含可关联标识：`taskId` / `orderId` / `payOrderId` / `merchantOrderId`。
**未输出商户证书、AppSecret、完整 Token。**

## 5. 备份恢复演练（尚未执行）

依赖 `blockers.md` BLK-02/BLK-03。步骤草案：
1. 全量下单至支付成功，记录 `trade_order` / `pay_order` / `pay_notify_task` 快照。
2. 备份数据库与 Redis。
3. 制造故障（停库），恢复备份。
4. 断言：订单与支付记录一致；`pay_notify_task` 中未完成任务在重启后继续执行；
   重复执行不产生重复库存恢复或重复退款。
