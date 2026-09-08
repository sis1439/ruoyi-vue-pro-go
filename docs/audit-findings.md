# audit-findings.md — 批次 C / D 复核与修复记录

**基线 commit**：`4a240b8b15d54f77d439b631417d35630c27ad4a`
**参照前端**：`yudao-mall-uniapp`，本地快照 `/Users/macmini/Desktop/yudao-mall-uniapp`
**执行日期**：2026-09-07
**证据等级**：A = 固定源码可直接看见；B = 由调用链推导、需运行复现；C = 为工程化新增的验收要求

> 说明：本文只覆盖批次 C（T05–T08）与批次 D（T11/T13 + T01/T12 收口）。
> 批次 A（T00–T03）与批次 B（T09/T10/T04）**尚未在本仓库执行**，相关缺口在
> `docs/blockers.md` 中列出，未在此声称已修复。

---

## T05 微信支付调用与客户端参数

### F-05-01 渠道标识丢失（A · 已修复）
* **现状复核**：`internal/service/pay/client/weixin/client.go:41` 的
  `NewWxPayClientAsClient` 把渠道写死为 `"wx_unknown"`；支付宝
  `internal/service/pay/client/alipay/client.go:25` 同样写死 `"alipay_unknown"`。
* **最小失败场景**：任意微信/支付宝渠道下单 → `UnifiedOrder` 的 switch 落到
  `default` → 返回 `暂不支持的微信支付渠道: wx_unknown`。**全部渠道下单 100% 失败。**
* **根因**：`ClientCreator` 签名 `func(channelID int64, config string)` 不含渠道编码，
  注册表按编码注册却无法把编码传回构造函数。
* **修复**：改 `ClientCreator` 签名为 `func(channelID int64, channelCode, config string)`
  （`internal/service/pay/client/factory.go`），两个渠道实现同步透传。
* **回归**：`TestCreatorReceivesChannelCode`、`TestCreatorKeepsChannelCode`。

### F-05-02 openid 未透传（A · 已修复）
* **现状复核**：`internal/service/pay/order.go` 的 `SubmitOrder` 构造
  `client.UnifiedOrderReq` 时未填 `ChannelExtras`，而 `jsapiOrder` 从
  `req.ChannelExtras["openid"]` 取 openid。
* **最小失败场景**：小程序 `wx_lite` 下单 → `JSAPI 支付需要 openid`。
  固定版前端 `sheep/platform/pay.js:102` 确实传了 `channelExtras.openid`。
* **修复**：`SubmitOrder` 透传 `reqVO.ChannelExtras`。
* **回归**：`TestJsapiRequiresOpenid` 锁定缺失/空 openid 必须报错而非带空值调渠道。

### F-05-03 displayContent 不是支付参数对象（A · 已修复）
* **现状复核**：`jsapiOrder`/`appOrder` 把 `*resp.PrepayId` 直接作为 `DisplayContent`。
* **前端契约（固定快照实测读取）**：
  * `sheep/platform/pay.js:179` `wechatMiniProgramPay` → `JSON.parse(displayContent)`，
    读 `timeStamp` / `nonceStr` / `packageValue` / `signType` / `paySign`。
  * `sheep/libs/sdk-h5-weixin.js:181` `wxpay` → 同一组键名（`wx_pub` 公众号 JSSDK）。
  * `sheep/platform/pay.js:268` `wechatAppPay` → 读全小写
    `appid`/`partnerid`/`prepayid`/`package`/`noncestr`/`timestamp`/`sign`。
* **最小失败场景**：小程序拿到裸 prepayId 字符串 → `JSON.parse` 抛异常，支付无法拉起。
* **修复**：改用官方 SDK 的 `PrepayWithRequestPayment`（签名由商户私钥在服务端完成，
  私钥不下发前端），再按上述键名序列化。JSAPI 用 `packageValue`（SDK 的 JSON tag 是
  `package`，与前端不符，故手工组装 map）。
* **残留**：`wx_pub` 与 `wx_app` 未做终端实测，见 `blockers.md`。

### F-05-04 渠道枚举不一致（A · 已修复）
* **现状复核**：`init()` 注册 `wx_wap`，`UnifiedOrder` 的 switch 只认 `wx_h5`；
  `wx_bar`（付款码）注册了但无任何分派分支。
* **修复**：抽出 `payMethodOf(channelCode)`，`wx_wap` 为正式编码、`wx_h5` 作为历史别名。
  `wx_bar` 保持未实现并显式报错（不静默降级）。
* **回归**：`TestEveryRegisteredChannelIsDispatchable`。

### F-05-05 客户端缓存重启后为空导致静默失效（A · 已修复）
* **现状复核**：`PayClientFactory.GetPayClient` 只读内存 map；除
  `SubmitOrder` 外，回调处理、主动查单、退款、转账全部直接取缓存，取不到就
  **静默返回 nil / false**。进程重启后第一次支付回调即被丢弃。
* **修复**：`PayChannelService.GetPayClient(ctx, channelID)` 缓存未命中时
  校验渠道有效性并从数据库重建；渠道停用/跨租户一律报错，不回退 Mock。
  渠道更新/删除时 `RemovePayClient` 失效缓存。
* **残留（B）**：跨租户拒绝依赖 `PayChannel` 的租户字段，属批次 B（T10）范围，未验收。

---

## T06 支付通知、订单标识与可靠补偿

### F-06-01 通知请求体是 `{}`（A · 已修复）
* **现状复核**：`internal/service/pay/notify.go:170` `reqBody := []byte("{}")`，
  注释里明确写着 `TODO: Build actual payload`。
* **最小失败场景**：支付成功 → 商城 `/trade/order/update-paid` 收到 `{}` →
  `ShouldBindJSON` 因 `binding:"required"` 失败 → 返回参数错误 → 通知重试至耗尽。
  **订单永远停留在待支付。**
* **修复**：新增 `buildNotifyBody(task)`，按类型构造
  `PayOrderNotifyReq` / `PayRefundNotifyReqDTO` / 转账 DTO，与接收端契约共享。
* **回归**：`TestBuildNotifyBody`。

### F-06-02 成功判定依赖裸字符串（A · 已修复）
* **现状复核**：`responseBody == "success" || responseBody == "SUCCESS"`，
  而商城接口返回 `{"code":0,"msg":"","data":true}`。
* **后果**：即便请求成功也永远判为失败，任务重试到耗尽。
* **修复**：`isNotifySuccess` 解析统一 JSON 业务码，要求 `code == 0`；
  HTTP 200 但业务码非 0 记为 `RequestSuccess`（请求成功、结果失败）继续重试。
* **回归**：`TestIsNotifySuccess`（含旧实现会误判的 `success`/`SUCCESS` 用例）。

### F-06-03 商户订单号语义冲突（A · 已修复）
* **现状复核**：
  * 创建支付单：`order_update.go:910/1006` 用 `order.No`（20 位时间戳串）。
  * 接收回调：`handler/app/mall/trade/order.go:130` `utils.ParseInt64(r.MerchantOrderId)`
    当作数据库主键查单。`order.No` 由 `generateOrderNo()` 生成 20 位十进制，
    **超过 int64 上限（9.22e18）**，解析必然出错。
  * 售后退款：`after_sale.go:504` 却用 `strconv.FormatInt(as.OrderID, 10)`（主键）。
* **最小失败场景**：
  1. 支付成功回调 → `ParseInt64` 得到 0 或错误值 → 参数错误，订单不更新。
  2. 售后发起退款 → 支付侧按 `MerchantOrderId = 订单主键` 查支付单，
     而支付单是用 `order.No` 建的 → **支付订单不存在，售后退款完全无法发起。**
* **决策**：统一为 **交易订单主键的十进制字符串**，对齐 Java 基线
  （`String.valueOf(order.getId())`），见 `docs/decisions.md` ADR-002。
* **修复**：`order_update.go` 三处 + `order_processors.go` 的一致性校验同步改为主键；
  `after_sale.go` 本就是主键，现在四处一致。

### F-06-04 取消退款的退款标识不匹配分派规则（A · 已修复）
* **现状复核**：取消已支付订单时 `MerchantRefundId: refundNo`（`R` 前缀），
  但 `TradeAfterSaleService.UpdateRefunded` 按 `order-` 前缀分派订单级退款，
  否则 `ParseInt64` 当售后单 ID → `R2026…` 解析失败。
* **后果**：取消订单的退款回调无法归位，订单退款状态永远不更新。
* **修复**：改为 `"order-" + 订单主键`，同时天然唯一，重复取消被
  `validatePayRefundExist` 拒绝。

### F-06-05 通知任务依赖可能失效的 Context / 无界并发（A · 已修复）
* **现状复核**：`ExecuteNotify` 对每个任务 `go func` 无上限，且直接复用调用方 ctx；
  `Save`/`Create` 的错误被丢弃（`s.q.PayNotifyTask...Save(task)` 无返回值检查）。
* **修复**：`context.WithoutCancel` 脱离调用方取消信号；`notifyConcurrency = 8`
  有界并发并 `WaitGroup` 等待；`url.ParseRequestURI` 校验回调地址；
  请求构造/读取失败显式记为 `RequestFailure`；任务状态写入失败返回错误并记日志；
  重试耗尽时打 Error 级日志（告警接入点）。
* **残留（C）**：告警只到日志级别，未接告警系统，见 `blockers.md`。

### F-06-06 通知锁会误删他人持有的锁（A · 已修复）
* **现状复核**：`notify_lock.go` `SetNX` 写固定值 `"1"`，`defer` 无条件 `Del`。
  任务超过 120s 锁过期被他人获取后，本进程结束仍会删掉对方的锁。
* **修复**：锁值改为随机 token，释放用 Lua CAS 只删自己的；
  释放使用 `context.WithoutCancel` 独立 Context，业务 ctx 取消时锁仍能归还。

### F-06-07 支付中心 → 商城无信任边界（A · 已修复）
* **现状复核**：`/app-api/trade/order/update-paid` 挂在 `OptionalAuth` 分组下，
  `/admin-api/trade/after-sale/update-refunded` 注释直书 `No Auth`，均可公开 POST。
* **现有防线**：接收端确实会重新查询支付单校验状态、金额与商户订单号
  （`order_processors.go` 4.1–4.4），伪造请求无法把未支付订单改成已支付。
* **修复（纵深防御）**：新增共享令牌 `pay.notify_token` +
  `middleware.PayNotifyToken()`（常量时间比较）。发送端在
  `X-Pay-Notify-Token` 头携带。未配置时放行（本地开发），
  `config.Validate` 保证非 local 环境必须配置。

### F-06-08 渠道回调缺少商户/金额校验（A · 已修复）
* **现状复核**：`ParseOrderNotify` 只验签解密，不校验 `mchid`/`appid`；
  `OrderResp` 无金额字段，`updateOrderSuccessTx` 也就无从校验实收金额。
* **修复**：`OrderResp` 增加 `Price`；微信 `ParseOrderNotify`/`GetOrder` 填充，
  并校验 `mchid`/`appid` 归属；`updateOrderSuccessTx` 在置为已支付前
  比对渠道实收金额与支付单金额。

---

## T07 订单状态机与支付同步

### F-07-01 支付更新未纳入预期原状态（A · 已修复）
* **现状复核**：`order_processors.go` 的 `PayOrderProcessor` 先读状态再
  `Where(ID.Eq(orderID)).Updates(...)`，无状态条件、不检查影响行数。
* **最小失败场景**：读到"待支付"后、更新前订单被取消 → 更新仍成功 →
  已取消订单被"复活"成待发货，且库存已在取消时释放。
* **修复**：加 `Status.Eq(TradeOrderStatusUnpaid)` 条件，`RowsAffected == 0` 报错。

### F-07-02 取消未检查影响行数（A · 已修复）
* **现状复核**：`CancelOrder` 有状态条件但丢弃 `ResultInfo`；
  更严重的是 `CancelOrderProcessor`（超时取消任务走这条）**完全没有状态条件**。
* **最小失败场景**：超时取消任务查出待支付订单后，用户完成支付；
  无条件更新把已支付订单改成已取消，后置处理照常恢复库存/返券 →
  **已收款却释放库存。**
* **修复**：两处都加条件状态转换 + `RowsAffected` 检查；未更新到行时中止事务，
  绝不继续执行资源恢复。

### F-07-03 `get-detail?sync=true` 未实现（A · 已修复）
* **现状复核**：`GetOrderDetail` 只读 `id`，忽略 `sync`。
  固定版前端在 `sheep/api/trade/order.js:83` 声明了该参数
  （当前页面主要在 `/pay/order/get?sync=true` 使用，见 `api-compatibility.csv`）。
* **修复**：新增 `TradeOrderUpdateService.SyncOrderPayStatus`：
  1. 归属校验（`user_id` 必须匹配）；
  2. Redis 频率闸门 `AcquireSyncSlot`，同订单 3s 一次，Redis 不可用时**不放行**；
  3. 通过 `ValidateOrderActuallyPaid` 向渠道查单；查询失败/结果未知时保持原状态，
     不当作"确定未支付"；
  4. 确认成功后走 `UpdateOrderPaid` 这个统一可信入口更新，
     由它重新校验支付单状态、金额、商户订单号。
     **前端参数与前端"支付成功"回调都不能直接把订单改成已支付。**
* **同时**：`/pay/order/get?sync=true` 加 `SyncOrderQuietlyThrottled` 频率闸门。

---

## T08 退款与售后资金闭环

### F-08-01 退款金额未校验下界（A · 已修复）
* **现状复核**：`validatePayOrderCanRefund` 只校验上界
  （`RefundPrice + Price > payOrder.Price`），未拒绝 0 或负数。
* **修复**：加 `reqDTO.Price <= 0` 拒绝。

### F-08-02 售后退款完成不幂等（A · 已修复）
* **现状复核**：`UpdateAfterSaleRefunded` 事务外做 `as.Status == Complete` 预检查，
  事务内无条件 `Updates`。重复/乱序回调会重复执行
  `UpdateOrderItemWhenAfterSaleSuccess`，**重复累加订单退款金额**。
* **修复**：改为条件状态转换 + `RowsAffected == 0` 时幂等返回。

### F-08-03 支付侧退款状态机复核（A · 无需修改）
* `notifyRefundSuccessTx` / `notifyRefundFailureTx` 已使用条件状态转换 +
  `RowsAffected` 检查，"退款受理（Waiting）"与"退款成功（Success）"分离，
  与 Java 基线一致。本次未改动。
* **残留（B）**：`validateNoRefundingOrder` 是先查后写，存在 TOCTOU 窗口；
  真正的唯一性防线是 `MerchantRefundId` 的唯一约束，
  **而该约束依赖批次 A（T02）的 Schema 尚未建立**，见 `blockers.md`。

---

## T11 首期必需服务与配置

### F-11-01 关键配置缺失时静默启动（A · 已修复）
* **现状复核**：`config.Load` 只做 Unmarshal，不校验。
  `pay.order_notify_url` 为空时 `genChannelOrderNotifyUrl` 生成 `/3` 这种非法地址，
  渠道回调永远收不到，且没有任何启动期提示。
* **修复**：新增 `Config.Validate()`，缺 `mysql.dsn`/`redis.addr`/
  两个支付回调地址时拒绝启动并给出可操作错误；`Load()` 末尾调用。
* **回归**：`TestValidateRejectsMissingConfig`。

### F-11-02 JWT 密钥硬编码 + 未固定签名算法（A · 已修复）
* **现状复核**：`pkg/utils/jwt.go:10` `[]byte("yudao-backend-go-secret")`，
  注释 `TODO: Move to config`；`ParseToken` 未限制 `alg`。
* **风险**：默认密钥公开可得，任何人可签发管理员 Token；
  未限制 `alg` 存在算法混淆面。
* **修复**：密钥改为可配置（`app.jwt_secret`），`main` 启动时注入；
  `ParseToken` 加 `jwt.WithValidMethods(["HS256"])`；
  `Validate()` 在非 local 环境拒绝空密钥。
* **回归**：`TestValidateRejectsWeakProdConfig`。
* **注意**：这只是 A25 的一部分。**完整的 T09/T10 认证与租户验收属批次 B，未执行。**

---

## T13 构建、CI 与最低生产运维

### F-13-01 生成代码未入库且无 CI（A · 已修复）
* **现状复核**：`.gitignore` 忽略 `/internal/repo/query/`，
  未执行 `make gen` 时 `go build ./...` 直接失败；仓库无任何 CI 配置。
* **修复**：`.github/workflows/ci.yml`（固定 Go 1.25.4）：
  `make gen` → build → vet → 单元测试 → `govulncheck@v1.1.4`（固定版本）；
  另起 `integration` job 挂 `postgres:16` + `redis:7` 固定镜像，
  跑 `-tags=integration`。Makefile 增加 `test` / `test-integration` / `vet` /
  `lint`（golangci-lint 固定 v1.64.8）/ `ci`。

### F-13-02 无优雅退出（A · 已修复）
* **现状复核**：`main.go` 用 `engine.Run(addr)`，收到 SIGTERM 直接退出，
  在途请求被切断。
* **修复**：改为 `http.Server` + `signal.Notify` + 15s `Shutdown`。
  未完成的通知任务由 `pay_notify_task` 表持久化，重启后由定时任务继续。

---

## 未修改项（明确记录，不冒充已完成）

| 项 | 原因 |
|---|---|
| 前端 `yudao-mall-uniapp` | 按 T01 原则，差异全部在 Go 侧修正，**未改动任何前端文件** |
| PostgreSQL 适配（T03） | 属批次 A；当前 `pkg/config` 仍只有 `mysql.dsn`，见 blockers |
| Schema / 迁移（T02） | 属批次 A；唯一约束缺失影响 F-08-03 的最终结论 |
| 认证与租户（T09/T10） | 属批次 B；本次仅顺带修了 JWT 密钥与 alg 两点 |
| 订单/库存/优惠券事务（T04） | 属批次 B；本次只处理了状态转换的条件与影响行数 |
| 支付宝、钱包、转账渠道 | 只做了签名一致性修正（渠道编码透传），未做终端验收 |
