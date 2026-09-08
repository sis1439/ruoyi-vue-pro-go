# decisions.md — 架构决策记录（批次 C / D）

> 本分支基于 A/B `6548c9e`；数据库、事务、认证与租户决策沿用 `docs/batch-ab/decisions.md`。C/D 迁入提交为 `49f1111`，以下内容已按 review 修复更新。

## ADR-001 支付渠道客户端的构造与生命周期

**背景**：`ClientCreator` 签名不含渠道编码，导致所有渠道实例持有 `wx_unknown` /
`alipay_unknown`；客户端只存在于内存 map，进程重启后全部丢失。

**决策**：
1. `ClientCreator` 签名改为 `func(channelID int64, channelCode, config string)`，
   渠道编码由注册表透传到实例。
2. 取客户端的唯一入口是 `PayChannelService.GetPayClient(ctx, channelID)`：
   每次读取缓存前校验可信租户、渠道存在且启用、支付应用存在且启用；未命中时从数据库配置重建。
3. 渠道 `Update`/`Delete` 时 `RemovePayClient` 失效缓存。
4. 未注册的渠道编码一律报错，**不回退到 Mock**。

**被否决的方案**：在每个调用点补 `CreateOrUpdatePayClient`（原 `SubmitOrder` 的写法）。
调用点有 8 处，漏掉任何一处就是一条静默失效路径；集中在取客户端的那一层修更小也更安全。

**代价**：每次访问读取渠道与应用状态，保证其他实例禁用渠道后缓存不能绕过校验。

---

## ADR-002 订单标识关系表（本次的核心决策）

| 标识 | 含义 | 唯一范围 | 生成处 |
|---|---|---|---|
| `trade_order.id` | 交易订单主键 | 全局 | 数据库自增 |
| `trade_order.no` | 展示用订单号，20 位十进制 | 全局（Redis 序列） | `generateOrderNo()` |
| `pay_order.id` | 支付单主键 | 全局 | 数据库自增 |
| `pay_order.no` | 支付单号，`P` + 时间 + 序列 | 全局 | `PayNoRedisDAO.Generate("P")` |
| `pay_order_extension.no` | **渠道侧的 `out_trade_no`** | 全局 | 同上（每次提交一条） |
| `pay_order.merchant_order_id` | **商户订单号** | 每个 `pay_app` 内唯一 | 见下 |
| `pay_refund.no` | 渠道侧 `out_refund_no` | 全局 | `PayNoRedisDAO.Generate("R")` |
| `pay_refund.merchant_refund_id` | 商户退款号 | 每个 `pay_app` 内唯一 | 见下 |
| `channel_order_no` | 渠道单号（微信 `transaction_id`） | 渠道侧 | 渠道返回 |

**决策**：`merchant_order_id` = **`strconv.FormatInt(trade_order.id, 10)`**。

**理由**：
1. 对齐 Java 基线（`String.valueOf(order.getId())`）。
2. `trade_order.no` 是 20 位十进制，**超过 int64 上限**，接收端的
   `ParseInt64` 必然出错——保留 `No` 方案必须同时改接收端按 `No` 查询，
   而仓库里已有一半代码（售后 `after_sale.go:504`）用的是主键，
   统一到主键的改动面更小、与 Java 更一致。
3. 商户订单号不参与对外展示，前端展示用的是 `trade_order.no`，契约不受影响。

**`merchant_refund_id` 的分派规则**（`TradeAfterSaleService.UpdateRefunded` 依赖）：
* `order-<订单主键>` → 订单级退款（取消已支付订单）
* 纯数字 → 售后单主键

**数据迁移影响**：基线上若已存在用 `order.No` 建的 `pay_order` 行，
其 `merchant_order_id` 与新规则不符。当前无生产数据，**未编写迁移脚本**；
若需带数据升级，须在批次 A 的迁移体系里补一条转换。已记入 blockers。

---

## ADR-003 支付通知的契约与可靠性

**决策**：
1. 请求体由 `buildNotifyBody(task)` 按类型构造，与接收端 DTO 同源
   （`PayOrderNotifyReq` / `PayRefundNotifyReqDTO`），不再发 `{}`。
2. 成功判定 = HTTP 200 **且** 响应 JSON `code == 0`。
   HTTP 200 但业务码非 0 → `RequestSuccess`，继续按退避表重试。
3. 重试表沿用 Java 的 `{15,15,30,180,1800,1800,1800,3600}` 秒，
   `MaxNotifyTimes = len + 1`；耗尽后置 `Failure` 并打 Error 日志。
4. 沿用 A/B 的有截止时间租户上下文，最多顺序执行 20 条；调用方取消后停止处理，单次 HTTP 最长 10 秒。
5. 支付/退款/转账的本地状态与通知任务通过 `repo.InTransaction`、`repo.QueryFromContext` 使用同一事务；通知插入错误必须向上传播。
6. 每次投递后的任务状态和日志也在同一事务中提交。网络请求发生在本地事务之外；请求成功后进程崩溃仍可能重放，接收端必须幂等。

现有 `pay_notify_task` 就是持久投递记录，无需新增第二张 Outbox 表。旧 C/D 版本只有外层事务，下游使用根连接且忽略插入错误，并不原子；本次以真实 PostgreSQL 故障注入验证修正后的事务连接与回滚。

---

## ADR-004 支付中心 → 商城的信任边界

**决策**：双层防护，两层都不可省。
1. **令牌层**：共享令牌 `pay.notify_token`，
   发送端置于 `X-Pay-Notify-Token` 头，接收端 `middleware.PayNotifyToken()`
   常量时间比较，未配置时拒绝。令牌必须与 JWT 密钥独立；生产类环境要求强令牌及 `pay.trusted_notify_urls` 完整 URL 白名单。发送端仅向白名单地址附加令牌，禁止重定向。
2. **重校验层**：接收端**必须**重新查询可信支付记录，校验
   支付单状态 = 成功、金额一致、商户订单号一致、订单归属
   （`order_processors.go` 4.1–4.4）。

**理由**：令牌一旦泄漏，重校验层仍能挡住伪造支付；
只有重校验层时，端点仍可被匿名探测与刷。两层职责不同，不互相替代。

---

## ADR-005 支付状态主动同步（`sync=true`）

**决策**：
* 状态变更的唯一路径是 `UpdateOrderPaid` → `PayOrderProcessor`，
  由它重新校验支付单。**前端参数与前端"支付成功"回调都不是付款凭证。**
* 频率闸门放在 Redis（`SetNX` + TTL 3s，按可信租户和订单粒度）。
  **Redis 不可用时不放行**——宁可不同步（回调仍会到），也不退化成无限制查单。
* 渠道查询失败或结果未知时保持原状态，不写"未支付"。

---

## ADR-006 首期启用与延期范围（本次未扩大）

**启用**：微信小程序支付（`wx_lite`）为首期必达；`wx_pub`/`wx_native`/`wx_wap`/
`wx_app` 代码路径已修正但未做终端验收，按渠道单独记录。

**明确未实现且显式报错**：`wx_bar`（付款码支付）——注册表保留但
`payMethodOf` 返回 false，下单直接报错，不静默降级成其它渠道。

**未纳入**：秒杀、拼团、砍价、分销、钱包充值、支付宝各终端的真实验收。
本次仅修正了它们共用的渠道编码透传缺陷，未做功能验收。

## ADR-007 退款预留与迟到付款

创建退款前先锁定支付单行，在该事务内重新检查已退金额和待处理退款，再插入待处理退款单；外部请求在提交之后发送。当前仍沿用“一单同时只允许一笔待处理退款”的规则。未知网络结果和微信 `ABNORMAL` 保留待处理状态及额度，确定 `CLOSED` 才释放；同一退款号查单继续恢复。

已取消且未标记支付的交易订单收到支付通知时，返回失败并保留持久通知任务，不能把相同支付单号误判成“已经支付”。订单不复活、资源不重复扣回。重试耗尽转为失败任务并记录错误日志，进入人工对账/退款流程；本次没有自动退款，也没有外接告警系统。

## ADR-008 金额与渠道协议

成功付款必须携带正整数分且等于本地金额；重复通知同样校验金额及渠道/应用绑定。支付宝金额用十进制字符串精确解析，禁止浮点舍入。尚未实现的支付宝转账明确报错。

微信退款通知单独映射 `refund_status`，不复用查询接口的 `status` JSON 字段。渠道回调只有本地处理提交成功才返回 HTTP 200；验签、解析或数据库处理失败返回 HTTP 500，允许渠道重试。协议依据：https://pay.wechatpay.cn/doc/v3/merchant/4012268885 。
