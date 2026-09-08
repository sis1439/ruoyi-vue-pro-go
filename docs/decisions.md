# decisions.md — 架构决策记录（批次 C / D）

> 仅记录批次 C/D 实际做出的决策。批次 A/B 相关决策（数据库版本、迁移器、
> 事务传播、认证整合、租户策略）**尚未做出**，见 `docs/blockers.md`。

## ADR-001 支付渠道客户端的构造与生命周期

**背景**：`ClientCreator` 签名不含渠道编码，导致所有渠道实例持有 `wx_unknown` /
`alipay_unknown`；客户端只存在于内存 map，进程重启后全部丢失。

**决策**：
1. `ClientCreator` 签名改为 `func(channelID int64, channelCode, config string)`，
   渠道编码由注册表透传到实例。
2. 取客户端的唯一入口是 `PayChannelService.GetPayClient(ctx, channelID)`：
   缓存未命中时先 `ValidPayChannel`（存在 + 启用），再从数据库配置重建。
3. 渠道 `Update`/`Delete` 时 `RemovePayClient` 失效缓存。
4. 未注册的渠道编码一律报错，**不回退到 Mock**。

**被否决的方案**：在每个调用点补 `CreateOrUpdatePayClient`（原 `SubmitOrder` 的写法）。
调用点有 8 处，漏掉任何一处就是一条静默失效路径；集中在取客户端的那一层修更小也更安全。

**代价**：缓存未命中时多一次数据库查询。渠道数量是个位数，可接受。

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
4. 任务执行使用 `context.WithoutCancel(ctx)`，HTTP 请求/Job 结束不取消在途通知；
   单次请求另有 10s 超时。
5. 并发上限 8，`WaitGroup` 等待本轮结束再返回计数。

**未采用 Outbox**：支付结果落库与通知任务创建已在同一个
`s.q.Transaction` 内（`notifyOrderSuccessTx`），满足"原子提交"要求。
引入 Outbox 表会多一套投递与清理逻辑，当前收益不足。
**若将来通知任务创建移出该事务，必须补 Outbox。**

---

## ADR-004 支付中心 → 商城的信任边界

**决策**：双层防护，两层都不可省。
1. **令牌层**：共享令牌 `pay.notify_token`，
   发送端置于 `X-Pay-Notify-Token` 头，接收端 `middleware.PayNotifyToken()`
   常量时间比较。未配置时放行（本地开发），非 local 环境由 `Validate()` 强制要求。
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
* 频率闸门放在 Redis（`SetNX` + TTL 3s，按订单粒度）。
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
