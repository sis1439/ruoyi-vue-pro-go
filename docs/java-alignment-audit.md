# App 端接口对齐审计 — Go vs Java vs uniapp

> 原始审计记录保留如下；当前处理结果、纠正项和验证范围见文末「修复记录」。原文行号和数量属于审计时快照，不代表修复后的代码。

**核对日期**：2026-09-08
**Go 侧**：本仓库 `codex/batch-cd`（HEAD `48349b9`）
**Java 侧**：`/Users/macmini/Desktop/ruoyi-vue-pro`
**前端**：`/Users/macmini/Desktop/yudao-mall-uniapp`

三方交叉核对。Java 是权威定义，前端是实际消费方；只有"Java 有 + 前端在用 + Go 没有/不同"才判为必修。

## 覆盖矩阵

| 维度 | 方法 | 状态 |
|---|---|---|
| 路径 + 方法 | 脚本枚举 Java 153 条 / Go 136 条 / 前端 130 条 | 全量 |
| 字段名 | Java 78 种返回 VO vs Go struct（名称配对 + 字段签名兜底） | 全量 |
| 字段类型 | 类型归一化比对 | 全量，0 真问题 |
| 枚举值 | 23 个 app 端枚举定向核对，覆盖前端全部硬编码状态判断 | 全量 |
| handler 实际填充 | 脚本找出字段从未被赋值的 struct，逐个人工核验构造点 | 全量 |
| 校验规则 | Java `@NotNull/@NotEmpty/@NotBlank` vs Go `binding:"required"` | 全量 |
| 错误码 / 请求头 / 分页语义 | 错误码数值段比对；请求头与分页参数逐项核对 | 全量 |
| 业务逻辑 — 价格计算链 + 库存扣减 | 11 个计算器逐个对比算法 | 全量 |
| 业务逻辑 — 订单/售后状态机、分销结算、活动逻辑、定时任务 | 逐方法对比状态校验、CAS、公式与调度注册 | 全量 |
| 前端调用约定（补充轮，见文末 F 系列） | 参数序列化、请求头、上传、WebSocket、鉴权边界、字段名全集 | 全量 |
| Go 内部一致性（补充轮） | context key、同名常量值冲突 | 全量 |

---

## P0 — 功能不可用

### P0-1 支付订单状态 REFUND / CLOSED 值互换

| | Java | Go |
|---|---|---|
| 已退款 | 20 | 30 |
| 支付关闭 | 30 | 20 |

定位：`internal/service/pay/consts.go:7-8`。另 `internal/consts/pay.go:61` 有第二份定义且缺 `Refund`，应合并为一处。

影响：前端 `pages/pay/index.vue:138` 判断 `status === 10 || status === 20` 为已支付。Go 返回已关闭的支付单（20）时，前端弹"订单已支付"并跳成功页；已退款单（30）被显示成支付关闭。

修改：`PayOrderStatusRefund = 20`、`PayOrderStatusClosed = 30`，并删除 `internal/consts/pay.go` 中的重复定义。需同步检查所有引用点的语义。

### P0-2 修改密码请求字段完全不同

- Java `AppMemberUserUpdatePasswordReqVO`：`{password, code}`
- Go `AppMemberUserUpdatePasswordReq`：`{oldPassword, newPassword, code}`，且 `oldPassword`/`newPassword` 均 `binding:"required"`
- 前端 `s-auth-modal/components/change-password.vue:95` 发 `{mobile, code, password}`

定位：`internal/api/contract/admin/member/member_user.go:77`。

影响：改密码必然 400。

修改：改为 `{password, code}`，与 Java 一致。

### P0-3 商品佣金接口结构完全不同

`GET /app-api/trade/brokerage-record/get-product-brokerage-price`

- Java `AppBrokerageProductPriceRespVO`：`{enabled, brokerageMinPrice, brokerageMaxPrice}`
- Go `AppBrokerageProductPriceRespVO`：`{brokerageEnabled, brokeragePrice}`

定位：`internal/api/contract/app/mall/trade/brokerage_record.go:30`。

影响：前端 `pages/commission/goods.vue:17` 按 min/max 显示佣金区间，全部落到 `undefined` 分支。

修改：按 Java 三字段重写，service 层需返回 SPU 下 SKU 佣金的最小/最大值。

### P0-4 售后分页筛选参数不符

- Java `AppAfterSalePageReqVO`：`statuses`（`Set<Integer>`）
- Go `AppAfterSalePageReq`：`status`（单个 `*int`）
- 前端 `pages/order/aftersale/list.vue:124` 发 `statuses=1,2,3`

定位：`internal/api/contract/admin/mall/trade/trade_after_sale.go:25`。

影响：售后列表 tab 筛选失效，始终返回全部。

修改：改为 `Statuses []int` + `form:"statuses"`，支持逗号分隔。

### P0-5 `/trade/after-sale/delivery` HTTP 方法不符

Java 和前端都是 `PUT`，Go 注册为 `POST`。

定位：`internal/api/router/app.go:194`。

### P0-6 前端在用但 Go 缺失的 13 个接口

Java 都有，Go 的 `/app-api` 下未注册：

| 接口 | 用途 | Java 侧 VO |
|---|---|---|
| `POST /infra/file/create` | 文件上传（服务端模式） | `AppFileUploadReqVO {directory, file}` |
| `GET /infra/file/presigned-url` | 直传预签名 | `FilePresignedUrlRespVO {configId, uploadUrl, url, path}` |
| `GET /system/area/tree` | 地区树（地址选择器） | `AppAreaNodeRespVO` |
| `GET /system/dict-data/type` | 字典 | `AppDictDataRespVO` |
| `GET /trade/delivery/express/list` | 快递公司列表 | `AppDeliveryExpressRespVO` |
| `GET /trade/delivery/pick-up-store/list` | 自提门店列表 | `AppDeliveryPickUpStoreRespVO`（含 `areaName`、`distance`） |
| `GET /trade/delivery/pick-up-store/get` | 自提门店详情 | 同上 |
| `GET /trade/after-sale-log/list` | 售后日志 | `AppAfterSaleLogRespVO` |
| `DELETE /trade/order/delete` | 删除订单 | — |
| `POST /trade/order/item/create-comment` | 订单评价 | — |
| `GET /trade/brokerage-user/get-rank-by-price` | 我的佣金排名 | `AppBrokerageUserRankByPriceRespVO` |
| `GET /trade/brokerage-user/rank-page-by-price` | 佣金排行榜 | 同上 |
| `GET /trade/brokerage-user/rank-page-by-user-count` | 推广人数排行榜 | `AppBrokerageUserRankByUserCountRespVO` |

排行榜三个的 DTO 和 handler 在 Go 里都已实现（`internal/api/handler/app/mall/trade/brokerage/user.go:163`），**只差路由注册**，成本最低。

另有 8 个 Java 有、前端未用的接口：`member/level/list`、`member/experience-record/page`、`promotion/bargain-record/page`、`promotion/article/list`、`promotion/article-category/list`、`infra/file/upload`、`promotion/article/add-browse-count`、`promotion/banner/add-browse-count`。

### P0-7 申请售后必然 400 — Go 多要求两个 Java 不存在的必填字段

`POST /app-api/trade/after-sale/create`

Go `AppAfterSaleCreateReq`（`internal/api/contract/admin/mall/trade/trade_after_sale.go:10`）要求 `count` 和 `type` 必填，**Java `AppAfterSaleCreateReqVO` 里根本没有这两个字段**（Java 只有 `orderItemId`、`way`、`refundPrice`、`applyReason` 四个必填，外加 `applyDescription`、`applyPicUrls` 两个可选）。

前端 `pages/order/aftersale/apply.vue:166` 提交的正是 Java 那 6 个字段，不含 `count`/`type` → `ShouldBindJSON` 直接返回 400，请求进不了 service。

**这两个字段本身就是多余的**，Go service 里已经不需要它们：

- `Type`：`internal/service/mall/trade/after_sale.go:199` 已经按订单状态自己推导（`order.Status == TradeOrderStatusCompleted` → 售后，否则售中），**完全没读 `r.Type`**。与 Java `AfterSaleServiceImpl:180` 的做法一致
- `Count`：`after_sale.go:194` 取的是 `r.Count`，而 Java 是从订单项 `orderItem` 带出来的（`AfterSaleConvert.convert(createReqVO, orderItem)`）。本来就不该由前端传

修改：DTO 删掉 `Type` 字段；`Count` 也从 DTO 移除，service 里改成 `item.Count`。改完必填集就与 Java 对齐了。

影响范围：只是「申请售后」这一个接口。售后列表、详情、取消是通的；「退回货物」另有 P0-5 的方法不符问题。

---

## P1 — 字段缺失，页面显示异常

### P1-1 订单详情缺 6 个字段

`AppTradeOrderDetailResp`（`internal/api/contract/admin/mall/trade/trade_order.go:353`）缺：

| 字段 | 前端用途 |
|---|---|
| `payExpireTime` | 支付倒计时 |
| `pickUpVerifyCode` | 自提核销码 |
| `pickUpStoreId` | 自提门店 |
| `receiverAreaName` | 详情页地址栏（`pages/order/detail.vue:60`） |
| `payChannelName` | 支付方式显示（`pages/order/detail.vue:158`） |
| `logisticsName` | 物流公司名 |

`receiverAreaName` 在 admin 的 DTO 里已有（`trade_order.go:75`），app 端漏了。

### P1-2 提现创建缺 3 个字段

`AppBrokerageWithdrawCreateReqVO` 缺 `transferChannelCode`、`userAccount`、`userName`。

定位：`internal/api/contract/app/mall/trade/brokerage_withdraw.go`。影响：提现无法提交。

### P1-3 收件地址缺 `areaName`

`AppAddressResp`（`internal/api/contract/admin/member/member_address.go:25`）缺 `areaName`，前端 `s-address-item` 组件靠它显示省市区。admin 端 handler 里已有 `area.Format()` 的用法（`internal/api/handler/admin/mall/trade/order.go:222`）可复用。

### P1-4 商品评论字段缺失

`AppProductCommentResp`（`internal/api/contract/admin/mall/product/product_comment.go:95`）：

- 前端在用但缺：`replyTime`、`spuName`
- 命名不符：Go 叫 `skuProperties`，前端读 `properties`
- Java 有、前端暂未用：`userId`、`anonymous`、`orderId`、`orderItemId`、`skuId`、`spuId`、`descriptionScores`、`benefitScores`、`replyStatus`、`replyUserId`、`additionalContent`、`additionalPicUrls`、`additionalTime`

### P1-5 下级分销统计三重问题

`GET /app-api/trade/brokerage-user/child-summary-page`

1. 响应缺 `brokerageOrderCount`
2. 请求参数 Java 叫 `sortingField`、Go 叫 `sorting`；且 `AppBrokerageUserChildSummaryPageReqVO` 的 `nickname`/`level`/`sorting` **只有 `json` tag 没有 `form` tag**，而 handler 用 `ShouldBindQuery` → GET 参数全部丢失，筛选排序完全失效
3. handler 只填了 `id` 和 `brokeragePrice`，`nickname`/`avatar`/`brokerageTime`/`brokerageUserCount` 全是零值（代码自标 `// Placeholder conversion`）

定位：`internal/api/contract/app/mall/trade/brokerage_user.go:17`、`internal/api/handler/app/mall/trade/brokerage/user.go:145`。

### P1-6 时间字段序列化格式

Java 全局配置 `TimestampLocalDateTimeSerializer`（`YudaoJacksonAutoConfiguration.java`）→ `LocalDateTime` 输出**毫秒时间戳**。

Go 有对应的 `pkg/types.JsonDateTime`（`pkg/types/types.go:350`），但**仍有 238 处字段用裸 `time.Time`** → 输出 RFC3339 字符串。

app 端已确认受影响的：

| 字段 | 位置 | 前端表现 |
|---|---|---|
| `headRecord.expireTime` | `app_promotion_combination.go:37` | `pages/activity/groupon/detail.vue:71` 做 `expireTime <= new Date().getTime()`，字符串比数字恒为 false，拼团永不显示已结束 |
| `createTime` | 钱包流水、佣金记录、提现记录、客服消息 | `timeFormat()` 把 `-` 换成 `/` 后解析成 Invalid Date |
| `payTime` | `app_pay_wallet.go:46` | 同上 |
| `expireTime`/`createTime`/`successTime` | `pay_order.go:68-75` | 同上 |
| `brokerageTime`、`auditTime`、秒杀 `startTime`/`endTime` | 多处 | 同上 |

对照组：`AppRewardActivityResp`（`app_reward_activity.go:8`）已用 `int64` 毫秒，说明对齐做过但没做完。

修改：app 端响应里所有 `time.Time` 换成 `types.JsonDateTime`。

### P1-7 结算 `deliveryType` 被强制必填，Java 是可选

Java `AppTradeOrderSettlementReqVO.deliveryType` 只有 `@InEnum`（值域校验），**没有 `@NotNull`**，允许不传。

Go 两处强制：DTO `binding:"required"`（`trade_order.go:100`）+ handler 显式检查（`internal/api/handler/app/mall/trade/order.go:72`）返回 `400 参数错误: deliveryType 缺失或无效`。

前端 `pages/order/confirm.vue:379-395` 的逻辑是「先试快递 → 失败再试自提 → 都失败则 `deliveryType = undefined` 再算一次价」，且 `sheep/api/trade/order.js:35` 在 `deliveryType` 无值时主动 `delete` 该参数。

影响：兜底路径（商品不支持快递、且自提未开启）下结算页拿不到价格，页面空白。

### P1-8 修改用户信息三个字段被强制必填

Java `AppMemberUserUpdateReqVO` 的 `nickname`、`avatar`、`email`、`sex` **全部可选**（无任何校验注解），Go 的 `AppMemberUserUpdateReq` 要求 `avatar`、`nickname`、`sex` 必填（`internal/api/contract/admin/member/member_user.go`）。

另外 `sex` 是非指针 `int` + `required`，而 `SexEnum.UNKNOWN = 0` 是合法值 —— 未设置过性别的用户提交时会被判为「缺失」。

影响：只更新单个字段（如仅换头像）会 400。

### P1-9 提现与佣金记录的 `typeName` / `statusName` 从未赋值

- `AppBrokerageWithdrawRespVO.typeName`、`statusName`（`internal/api/contract/app/mall/trade/brokerage_withdraw.go:41-42`），handler 里留了注释 `// TypeName, StatusName -> Dict lookup`（`brokerage/withdraw.go:73`）
- `AppBrokerageRecordRespVO.statusName`，handler 里留了 `// StatusName: item.Status // TODO: Dict lookup`（`brokerage/record.go:64`）

Java 侧这两个字段由字典翻译填充。前端 `pages/commission/wallet.vue:137,152,169` 直接显示 `item.typeName` / `item.statusName`。

影响：提现记录页的类型和状态列显示为空白。

### P1-10 取消订单回滚库存时未排除已售后的订单项

Java `TradeProductSkuOrderHandler.afterCancelOrder` 第一步就是：

```java
orderItems = filterOrderItemListByNoneAfterSale(orderItems);
if (CollUtil.isEmpty(orderItems)) { return; }
productSkuApi.updateSkuStock(TradeOrderConvert.INSTANCE.convert(orderItems));
```

已发起售后的订单项在退款流程里已经回滚过库存，这里必须排除，否则重复回滚。

Go `CancelOrderProcessor.AfterCancelOrder`（`internal/service/mall/trade/order_processors.go:530`）**没有这个过滤**，直接把全部 `orderItems` 回滚。

影响：订单里部分商品先申请了售后退款（库存已回滚），之后整单取消时会**再回滚一次**，库存虚增、`sales_count` 虚减。

修改：回滚前按 `AfterSaleStatus` 过滤掉非 `NONE` 的订单项。

### P1-11 满减送折扣的分摊范围不同

Java `TradeRewardActivityPriceCalculator` 分两步：`filterMatchActivityOrderItems` 先按 `rewardActivity.getSpuIds()` 筛出**参与活动的商品**，再 `dividePrice(orderItems, discountPrice)` 只在这些商品间分摊。

Go `reward_activity_calculator.go:79` 传的是 `resp.Items` —— **全部选中商品**，包括不参与该活动的：

```go
divideActivityDiscounts := c.Helper.DividePrice(resp.Items, activityDiscount)
```

影响：订单总价一致（总折扣额相同），但每个订单项的 `discountPrice` 分配错误 —— 折扣被摊到了不参与活动的商品上。这会传导到按订单项计算的场景：单项退款金额、分销佣金、订单项实付价显示。

修改：先按活动 spuIds 过滤出参与项，再分摊。

### P1-12 单项 `payPrice` 被钳制为 0，掩盖异常订单

Java `recountPayPrice` 不做钳制，负值会累加进总价，最终被 `TradePriceServiceImpl:73` 的 `payPrice <= 0` 校验拦截并抛 `PRICE_CALCULATE_PAY_PRICE_ILLEGAL`。

Go `price_calculator_helper.go:113` 多了一步：

```go
if item.PayPrice < 0 { item.PayPrice = 0 }
```

两边的总价校验条件本身是一致的（`internal/service/mall/trade/price_service.go:340`），但 Go 把单项负值抹平后，总价不再为负 —— 校验抓不到了。

影响：优惠叠加导致某商品优惠超额时，Java 拒绝下单，Go 允许下单且总价算少。属于资金风险。

修改：去掉钳制，让负值传导到总价校验。

### P1-13 订单取消类型（`cancel_type`）5 处全部写错

`internal/consts/trade.go` 里有两套取消类型常量。正确的一套（`TradeOrderCancelType*`，值 10/20/30/40）与 Java `TradeOrderCancelTypeEnum` 一致，但**代码里一处都没用**；实际在用的是标着「已废弃」的旧常量，语义与 Java 完全错位：

| 场景 | 代码位置 | Go 写入值 | Java 正确值 | 该值在 Java 的含义 |
|---|---|---|---|---|
| 会员取消 | `order_update.go:1366` | 10 | 30 | 超时未支付 |
| 支付超时 | `order_update.go:1917` | 20 | 10 | 退款关闭 |
| 售后关闭 | `order_update.go:1824` | 50 | 20 | Java 无此值 |
| 系统取消 | `order_processors.go:618,646` | 40 | — | 拼团关闭 |
| 拼团关闭 | `combination_record.go:462` | 70 | 40 | Java 无此值 |

Java 定义：`PAY_TIMEOUT=10`、`AFTER_SALE_CLOSE=20`、`MEMBER_CANCEL=30`、`COMBINATION_CLOSE=40`。

影响：`trade_order.cancel_type` 落库值全错，admin 后台的取消原因显示、按取消类型的统计全部错乱。

修改：5 处改用 `TradeOrderCancelType*` 常量，删除旧的兼容常量块。已落库的历史数据需要一并迁移。

### P1-14 售后取消的可用状态过窄

Java `cancelAfterSale` 允许在 `APPLY(10)`、`SELLER_AGREE(20)`、`BUYER_DELIVERY(30)` 三种状态下取消：

```java
if (!ObjectUtils.equalsAny(afterSale.getStatus(), APPLY, SELLER_AGREE, BUYER_DELIVERY)) {
    throw exception(AFTER_SALE_CANCEL_FAIL_STATUS_NOT_APPLY_OR_AGREE_OR_BUYER_DELIVERY);
}
```

Go `CancelAfterSale`（`internal/service/mall/trade/after_sale.go`）只允许 `AfterSaleStatusApply`。

影响：卖家已同意退款、或买家已填写退货物流后，用户无法再取消售后申请。

### P1-15 售后状态流转全部缺少 CAS

Java 所有售后状态变更都走 `updateAfterSaleStatus(id, oldStatus, updateObj)`：

```java
int updateCount = tradeAfterSaleMapper.updateByIdAndStatus(id, status, updateObj);
if (updateCount == 0) { throw exception(AFTER_SALE_UPDATE_STATUS_FAIL); }
```

即「带原状态条件更新 + 影响行数为 0 则报错」的乐观锁。

Go 的 `AgreeAfterSale`、`DeliveryAfterSale`、`CancelAfterSale`、`RefundAfterSale` 全部是 `Where(AfterSale.ID.Eq(id))`，没有状态条件，也不检查影响行数。

影响：并发下（用户取消 + 商家同意同时发生）状态会互相覆盖，可能出现「已取消的售后单被退款」。

对比：订单侧的支付回调（`order_processors.go:206`）已经正确实现了 CAS，售后侧漏了。

### P1-16 分销固定佣金为 0 时仍按比例计算

- Java：`if (fixedPrice != null && fixedPrice >= 0) return fixedPrice;` —— **`0` 也直接返回**，表示该商品不给佣金
- Go（`internal/service/mall/trade/brokerage/record.go:479`）：`if fixedPrice > 0 { return fixedPrice }` —— `0` 会**继续按比例计算**

影响：商品配置「固定佣金 0 元」时，Java 不发佣金，Go 按比例发佣金，属于资金损失。

### P1-17 拼团过期任务未接入调度

Java 有 `CombinationRecordExpireJob`，定时调用 `expireCombinationRecord()` 处理过期拼团（虚拟成团 or 关闭退款）。

Go 的 `ExpireCombinationRecord`（`internal/service/mall/promotion/combination_record.go:409`）**实现是完整的**（虚拟成团、过期退款两条分支都有），但**没有对应的 job 文件，也没有任何调用点** —— `cmd/server/wire_gen.go` 注册了 13 个 job，其中没有它。

影响：拼团永远不会自动过期，未成团的拼团单一直挂在「进行中」，用户的钱不退。

对照：Java 的 8 个 mall 相关 job 里，Go 缺的是这一个和两个统计 job（`TradeStatisticsJob`、`ProductStatisticsJob`，非核心）。

---

## P2

### P2-1 查询参数 `createTime` 缺失

`brokerage-record/page`、`brokerage-withdraw/page`、`member/point/record/page`、`product/browse-history/page` 的 Req 都缺 `createTime`。

`pay/wallet-transaction/page` 的 tag 是 `form:"createTime[]"`（`internal/api/contract/admin/pay/pay_wallet.go:57`），前端 `pages/user/wallet/money.vue:151` 发的是 `createTime[0]=...&createTime[1]=...` → 时间筛选失效。

### P2-2 其余字段缺失

| VO | 缺失字段 |
|---|---|
| `AppBargainRecordRespVO` / `AppBargainRecordDetailRespVO` | `payOrderId`、`payStatus`（handler 里标了 `// TODO: PayStatus and PayOrderId`） |
| `AppBargainActivityDetailRespVO` | `description`、`price` |
| `AppBargainHelpRespVO` | `userId` |
| `AppBrokerageWithdrawRespVO` | `payTransferId` |
| `AppBrokerageRecordRespVO` | `finishTime` |
| `AppMemberUserInfoRespVO` / `AppMemberUserUpdateReqVO` | `email` |
| `AppRewardActivityRespVO` | `description` |
| `AppProductSpuPageReqVO` | `categoryIds`、`ids` |
| `AppKeFuMessageSendReqVO` | `senderId`、`senderType` |

### P2-3 交易模块错误码用错了号段

Java 的号段约定：`1-004-xxx-xxx` 是 member 模块，`1-011-xxx-xxx` 是 trade 模块（见 `yudao-module-trade-api/.../ErrorCodeConstants.java` 头部注释）。

Go 的 `internal/service/mall/trade/errors.go` 把交易错误码写进了 member 的号段，与 Java member 模块的错误码**数值撞车**：

| 数值 | Java 含义 | Go 含义 |
|---|---|---|
| 1004003000 | 登录失败，账号密码不正确 | 价格计算失败 |
| 1004003001 | 登录失败，账号被禁用 | 价格计算商品为空 |
| 1004004000 | 用户收件地址不存在 | 订单不存在 |
| 1004006000 | 用户标签不存在 | 购物车项不存在 |
| 1004006001 | 用户标签已经存在 | 购物车添加失败 |
| 1004006002 | 用户标签下存在用户，无法删除 | 购物车更新失败 |

对照 Java 正确取值：订单不存在 = `1_011_000_011`，购物车项不存在 = `1_011_002_000`，价格计算 = `1_011_003_xxx`。

分销错误码 `1011007xxx` 是对的 —— 前端 `sheep/request/index.js:127` 专门匹配这个前缀做分销绑定失败提示。

影响：前端当前只判断 `401` 和 `1011007`，不会立即出错；但号段错误会让按错误码做的告警、埋点、客服话术全部对不上。

### P2-4 分页参数

- Java `PageParam` 有 `@Max(200)`，Go 无上限 —— 前端可传任意大的 `pageSize`
- Java `PAGE_SIZE_NONE = -1` 表示不分页，Go 的 `GetLimit()` 遇到 `PageSize < 1` 强制改为 10，不支持该语义

前端实际只用 5/6/8/10，当前不受影响，属于加固项。

### P2-5 `binding:"required"` 的 0 值陷阱

Go 的 `required` 对非指针数值字段会把 `0` 判为「缺失」。app 端有 30 处这样的字段，其中 `0` 是合法业务值的：

| 字段 | 0 的含义 |
|---|---|
| `AppMemberUserUpdateReq.sex` | `SexEnum.UNKNOWN = 0` |
| `AppAfterSaleCreateReq.refundPrice` | 0 元退款 |
| `AppAfterSaleDeliveryReq.logisticsId` | 无需物流（admin 侧的 `TradeOrderDeliveryReq` 已用 `*int64` 正确处理，app 侧没有） |

其余是 `id`/`skuId`/`spuId`/`activityId`/`scene`/`type` 一类，0 本就非法，无影响。

布尔字段已统一用 `*bool`（`PointStatus`、`Selected`、`DefaultStatus`），这块处理是对的。

### P2-6 `AppTradeConfigResp` 多返回 12 个零值字段

Java `AppTradeConfigRespVO` 只有 8 个字段，Go 复用了 admin 的 struct（20 个字段），service 只填了对应的 8 个（`internal/service/mall/trade/config.go:74`），另外 12 个恒为零值 —— 其中 `brokerageEnabled: false` 有误导性。

另 `TencentLbsKey` 硬编码为 `""`（代码注释「待补全」），Java 从配置读取。前端当前未使用该字段。

### P2-7 `dividePrice` 余数归属与取整方式不同

两处差异（`price_calculator_helper.go:50` vs `TradePriceCalculatorHelper.dividePrice`）：

**余数归属**：Java 把余数给**数组的最后一个元素**（`i < size - 1` 判断），若该元素恰好未选中，会走 `prices.add(0)` 分支，**余数直接丢失**（分摊总和 < 折扣额）。Go 用 `lastSelectedIndex`，余数总是给最后一个**选中**项，分摊总和恒等于折扣额。

随机模拟 5 万组「末项未选中」的输入，**66.7% 结果不同**（Java 少分 1–2 分）。

不过实际触发面很窄：Java 侧四个调用点里，优惠券、运费、积分抵扣、积分赠送都在传入前 `filterList(..., OrderItem::getSelected)` 过滤过了，只有满减送没过滤 —— 而 Go 在满减送这里另有 P1-11 的范围问题，两个差异叠加。

**取整方式**：Java 是 `(int)(price * (1.0D * payPrice / total))`（浮点除后乘再截断），Go 是 `totalDiscount * payPrice / totalPayPrice`（整数先乘后除）。20 万组全选中输入里 8 组结果差 1 分（0.004%），例如 `payPrice=[56,96,176]、折扣=861` → Java `[147,251,463]`，Go `[147,252,462]`。

Go 的两个行为都更"正确"，但与 Java 不一致。若以 Java 为准需照抄其浮点表达式和 `size-1` 判断。

### P2-8 优惠券折扣上限的空值判断不同

- Java：`coupon.getDiscountLimitPrice() == null ? couponPrice : Math.min(couponPrice, limit)` —— 只有 `null` 才不限制
- Go（`coupon_calculator.go:206`）：`if coupon.DiscountLimitPrice > 0` —— **`0` 也被当作不限制**

若 DB 中折扣券的 `discount_limit_price` 存了 0，Java 会把优惠额限制为 0，Go 会给出全额优惠。折扣券的该字段在 Java 侧是必填且大于 0，触发概率低。

### P2-9 售后退款缺少 0 元分支

Java `refundAfterSale` 对 `refundPrice == 0` 特殊处理：直接把售后单置为 `COMPLETE` 并写 `refundTime`，**不调用支付退款**。

Go `RefundAfterSale` 没有这个分支，无条件走 `PayRefundCreateReq` 向支付渠道发起退款。0 元退款请求会被渠道拒绝或产生无意义的退款单。

### P2-10 秒杀优惠未写入 promotions 明细

Java `TradeSeckillActivityPriceCalculator` 在计算后调用 `addPromotion(...)`，把「秒杀活动：省 X 元」写进 `result.promotions`。

Go 的 `seckill_calculator.go` 只改了 `DiscountPrice` 和 `PayPrice`，没有 `AddPromotion`。

影响：结算页的优惠明细列表里看不到秒杀这一项（金额本身是对的）。

另外 Go 直接赋值 `resp.Items[i].PayPrice = seckillTotal`，而 Java 走 `recountPayPrice`。当前顺序下（秒杀 order=8，运费 order=50）后续运费计算会重新 recount，结果一致，但绕过统一公式的写法在新增计算器时容易出错。

### P2-11 死常量清理

`TradeOrderItemAfterSaleStatusFailure = 30`（`internal/model/trade/trade_order.go:16`）和 `PayRefundStatusClosed = 99`（`internal/service/pay/consts.go:32`）在 Java 中都不存在，且 Go 里除定义处外无任何引用。建议删除，避免误用后写入 DB 造成前端判断落空。

---

## 已核对确认一致

- **枚举值**：订单状态、售后状态（含 61/62/63）、订单退款状态、配送方式、订单类型、营销类型、优惠券状态、商品范围、折扣类型、分销记录状态与业务类型、提现状态与类型、钱包业务类型、拼团/砍价记录状态、客服消息内容类型、终端、短信场景、社交类型 —— 除 P0-1 外全部一致
- **字段类型**：0 真问题。DIY 的 `property`/`home`/`user`（Java `String` vs Go `datatypes.JSON`）是误报，Java 侧带 `@JsonRawValue`，输出即原始 JSON，Go 写法正确
- **完整接口**：登录/刷新 token、`/member/user/get`、SPU 详情与 SKU、购物车 list（含 `properties`）、订单结算（`items[i].skuId` 展开 + price/coupons/promotions）、`settlement-product`、优惠券模板与我的优惠券、DIY 装修 template/page、`/trade/config/get`、分销 summary、钱包余额、客服消息列表（11 字段全对）、JSAPI 签名

- **响应字段填充**：脚本报出的 25 个「字段可能从未赋值」的 struct，逐个核验构造点后确认 22 个是误报（用 `r.Field = x` 而非字面量赋值）：`AppFavoriteResp`、`AppProductBrowseHistoryResp`、`AppTradeProductSettlementSkuBO`、`AppSeckillActivityDetailResp`、`AppBargainRecordDetailRespVO`、`AppPayWalletRechargeResp`、`AppPointActivityDetailRespVO`、`AppBargainHelpRespVO`、`AppCartItem`、`AppTradeOrderSettlementResp.address` 等均已正确填充。真问题只有 P1-5 和 P1-9 三个字段
- **请求头**：前端发送 `Authorization`、`terminal`、`tenant-id`、`Accept`，Go 中间件均有处理（`internal/middleware/auth.go:61-83`，`order.go` 读 `terminal` 头）
- **响应包装**：两边都是 `{code, msg, data}`，`401` 错误码一致（`pkg/errors/errors.go:19`），前端的刷新 token 逻辑能正常触发
- **分页返回结构**：`{list, total}` 一致，`pageNo` 都从 1 开始
- **价格计算责任链顺序**：11 个计算器一一对应，`Order` 值完全一致（秒杀/砍价/拼团/积分活动 8、限时折扣 10、满减送 20、优惠券 30、积分抵扣 40、运费 50、积分赠送 999）
- **`recountPayPrice` 公式**：`price * count - discountPrice + deliveryPrice - couponPrice - pointPrice - vipPrice`，两边一致
- **总价合法性校验**：都是「非积分订单且 `payPrice <= 0` 则报错」
- **满减送门槛判断**：Java 的 `calculateTotalPayPrice`/`calculateTotalCount` 内部跳过未选中项，与 Go 先过滤再算等价，门槛结果一致
- **优惠券折扣公式**：满减取 `discountPrice`，打折取 `totalPayPrice - totalPayPrice * percent / 100` 后与上限取小，一致
- **库存扣减事务性**：Go 的 `repo.InTransaction`（`internal/repo/transaction.go:20`）通过 ctx 传播事务，嵌套调用复用外层 tx，扣库存与订单写入在同一事务内，与 Java 的 `@Transactional` 等价
- **支付回调状态机**：幂等判断（同支付单号直接返回）、状态校验、支付单校验、CAS 更新（`Where(ID, Status=Unpaid)` + `RowsAffected == 0` 报错）全部对齐；Go 还额外校验了支付金额与商户订单号，比 Java 更严
- **取消订单**：状态必须为待支付、且二次查询支付单防回调延迟（`order_update.go:1390`），与 Java `cancelOrderByMember` 一致
- **秒杀参与校验**：活动存在/状态/时间区间/时段配置/单次限购/商品存在/库存，逐项对齐。Java `validateJoinSeckill` 签名里的 `userId` 在方法体内并未使用，Go 不传该参数不构成差异；Go 的 `GetCurrentSeckillConfig` 内部已按当前时刻过滤时段，等价于 Java 的 `isBetween(config.startTime, config.endTime)`
- **拼团过期处理逻辑本身**：虚拟成团与过期关闭退款两条分支都已实现，与 Java 一致（问题只在未接调度，见 P1-17）
- **分销佣金结构**：一级/二级分销的用户查找、`brokerageEnabled` 开关校验、冻结天数计算、解冻任务的 CAS 更新（`updateByIdAndStatus`）均已对齐
- **定时任务**：`cmd/server/wire_gen.go` 注册了 13 个 job，订单自动收货/自动关单/自动好评、佣金解冻、优惠券过期、支付单同步/过期/通知等均与 Java 对应

## 排除的误报

- `AppCartDetailRespVO`（`itemGroups`/`order`/`promotion` 等）：Java 侧死代码，未被任何 controller 引用
- `AppTradeProductSettlementRespVO`：Go 的结构定义在 `internal/service/mall/trade/types.go:122` 而非 contract 层，字段一致
- 内部类同名冲突：`Order`、`ItemGroup`、`Rule`、`Cart`、`Promotion`
- ID 精度：Java `NumberSerializer` 仅在 `|value| >= 2^53` 时转 String，范围内仍是数字，Go 的裸 `int64` 行为一致


---

## 修复记录（2026-09-08）

修复分支：`codex/java-alignment-audit`，基于当前 `main` 的 `81b03bb` 创建。未使用 subagent。原审计文件为已有未跟踪文件，本次保留其原文并追加核验结论。

### 处理矩阵

| 审计项 | 当前处理 |
|---|---|
| P0-1 | 支付订单枚举统一到 `internal/consts/pay.go`，退款 20、关闭 30；服务、渠道客户端和响应转换同步引用；提供历史数据迁移。 |
| P0-2 | 修改密码使用 `password/code`，校验并消费当前登录用户手机号的修改密码短信验证码。 |
| P0-3 | 返回 `enabled/brokerageMinPrice/brokerageMaxPrice`；最小佣金为 0 时不再被后续 SKU 覆盖。 |
| P0-4～5 | 售后分页接受 `statuses=10,20,30`；退货物流改为 PUT。 |
| P0-6 | 接入清单中的 13 个路由，并补充 `/infra/file/upload`；复用文件服务、地区数据、配送服务和现有订单/排行榜处理器；售后日志校验登录用户归属。 |
| P0-7 | 申请售后移除前端 `count/type`；数量来自订单项，类型由订单状态推导；退款金额用指针区分缺失与 0。 |
| P1-1～4 | 补齐订单详情、自提、地址地区、提现请求和商品评论契约及可取得的字段；时间输出为毫秒。 |
| P1-5 | 昵称与层级可绑定；支持 `sortingField.field/order`；按已结算订单佣金聚合金额/单数、统计直接下级人数，绑定时间、昵称及头像实际填充。 |
| P1-6 | App DTO 的裸 `time.Time` 改为 `types.JsonDateTime`，转换调用点同步修改；App 共用的支付单和客服消息 DTO 同步处理。内部实体、请求时间和仅 Admin 使用的时间不在本次统一范围。 |
| P1-7～8 | 结算允许省略配送方式；会员资料使用可选字段，部分更新不覆盖未提交值，允许性别 0，并支持邮箱。 |
| P1-9 | 提现类型/状态、佣金记录状态填充中文枚举名称。优先使用配置的字典标签，字典未配置时回退到枚举名称。 |
| P1-10 | 整单取消的库存回滚排除售后状态非 NONE 的订单项。 |
| P1-11～12 | 每个满减活动仅对匹配 SKU 分摊；保留订单项负实付值供总价校验。 |
| P1-13 | 会员取消/超时/售后关闭/拼团关闭使用 30/10/20/40；原“系统取消”实际位于全额退款路径，改为售后关闭；旧常量删除。 |
| P1-14～15 | 支持申请中、卖家同意、买家已发货时取消；售后状态更新增加 CAS 与行数校验；跨售后/订单项调用传播同一事务，创建时原子占用订单项。退款回调检查可信支付退款记录的状态、金额和商户单号。 |
| P1-16 | 固定佣金以指针区分未配置与 0；0 不再退回比例计算。 |
| P1-17 | 新增 `combinationRecordExpireJob`，Wire 聚合注册并增加数据库调度行；过期流程遇到退款/活动错误向上返回，避免错误被吞掉后提交失败状态。 |
| P2-1 | 积分、浏览历史增加创建时间范围；佣金、提现和钱包流水支持重复键、`[]` 和 `[0]/[1]` 查询形式。积分和浏览历史分页同时修正页码误作 offset。 |
| P2-2 | 补齐砍价支付关联、活动价格/描述、助力用户编号、提现转账编号、会员邮箱、商品 IDs/分类 IDs；客服发送字段接受但身份仍取可信上下文。`finishTime` 及满减描述的核验见下文。 |
| P2-3 | 交易错误码移出 member 的 1-004 号段；有直接对应的错误使用 Java 1-011 编号，Go 独有通用错误使用 1-011-900/901 等扩展编号，不将不同业务条件强塞到同一个 Java 错误码。 |
| P2-4～5 | 分页上限 200，支持 pageSize=-1；售后退款金额/物流 ID 用指针接受合法 0，会员资料性别可选。 |
| P2-6 | App 交易配置仅返回 Java 的八个字段；腾讯地图 key 从 `trade.tencent_lbs_key` 配置读取。 |
| P2-7～8 | 分摊按 Java 浮点表达式及数组末项余数规则；优惠券模型保留空上限和零上限的区别，两处折扣计算同步处理。 |
| P2-9～11 | 零元售后直接完成，不创建支付退款；秒杀促销明细补齐并使用统一重算公式；无引用的退款关闭/订单项售后失败常量删除。 |

### 对原审计结论的纠正

1. **文件接口**：Java `AppFileController` 中 `/create` 接收 `FileCreateReqVO` JSON，是直传完成后的记录登记；multipart 文件上传在 `/upload`。本次依照 Java 源码分别接入，未把 `/create` 错接到 multipart 上传。
2. **分销排序**：前端 `pages/commission/team.vue` 发送的是 `sortingField.field` 和 `sortingField.order`，并非一个普通字符串；当前实现按实际消费形式处理。
3. **满减 description**：Java `AppRewardActivityRespVO` 的描述位于嵌套 `Rule.description`，Go 已有字段且 `RewardActivityService` 已填充，原 P2-2 的顶层缺失判断为误报。
4. **佣金 finishTime**：Java VO 有此字段，但 DO 没有对应字段，Controller 仅 BeanUtils 映射及状态字典翻译，未填充 finishTime。本次补齐返回键并保留 null，不虚构结算时间。评论的追加内容/时间字段同样无本地 Java DO 来源，保留可空字段。
5. **负实付校验**：Java 检查的是订单总实付 `<=0`，并不保证任意一个负订单项都会被拒绝。本次去除钳制以恢复原始计算及总价校验语义，未增加 Java 没有的单项拒绝规则。
6. **提现类型**：顺链核验发现旧别名将收款码 3/4 用于 API 转账判断；调用点同步改用正确的微信零钱/支付宝余额 5/6。
7. **售后订单项状态**：售后流程状态与订单项的 NONE/APPLY/SUCCESS 不同。审核/退货步骤保持订单项 APPLY，避免将卖家同意/买家发货状态直接写入订单项。

### 迁移与发布顺序

新增 `migrations/000007_java_alignment.up.sql`：添加会员邮箱列、转换支付状态和取消类型、添加拼团过期任务。前六个迁移内容保持不变。

枚举转换针对本仓库修复前 Go 写入的历史数据，**应停止旧版本写入后执行迁移，再切换新版本**。如果目标数据库混有 Java 导入或人工修正的状态值，必须先核对来源并按记录区分，不能对混合来源数据盲目执行状态互换。迁移由版本记录保证只执行一次；不要把其中的 UPDATE 单独重复运行。本次仅在临时 PostgreSQL 实例执行验证，未修改业务数据库。

### 验证记录

- 定向回归已运行：Java 请求契约（可选资料、0 元售后/无需物流、CSV 状态筛选）、毫秒时间戳、分页 -1/200、Java 舍入案例、固定佣金 0、优惠券 null/0 上限。
- PostgreSQL 定向回归已运行：售后 CAS 单胜者、租户隔离、日志写失败事务回滚、零元退款完成及幂等、活动 SKU 分摊范围。
- 使用 `/private/tmp` 独立数据目录的 PostgreSQL 16.10 和独立 Redis 端口；所有数据均为测试夹具。
- `make test-integration`：通过；执行 `go test -race -count=1 -json ./...` 并通过 `scripts/check-integration`。本轮 144 个顶层测试、含子测试共 386 个通过，0 失败、0 测试跳过（无测试文件的包不计为用例跳过）；检查器确认必需 PostgreSQL/Redis 用例全部通过。
- `make wire`：成功重新生成装配；纳入已跟踪 `cmd/server/wire_gen.go` 的必要装配差异，未手工编辑该文件；忽略的 DAO 生成文件不纳入交付。
- `go build ./...`、`go vet ./...`：通过。构建曾输出本机模块缓存写权限提示，但返回状态为 0；测试使用独立可写 `GOCACHE`。
- `git diff --check`：通过；检查新增文件，未纳入本地配置、凭证、测试日志或生成 DAO。
- 外部支付/转账渠道、对象存储实际直传及 uniapp 真机联调未执行；本地验证不等同于真实渠道验收。

---

## 修复后复核：遗留差异（2026-09-08）

在修复分支 `codex/java-alignment-audit`（HEAD `3eed27a`）上重跑了全部维度的比对。以下保留修复前的复核发现及修正说明。R8 涉及改密码流程阻断；R7 经源码复查排除。各项最新处理结果与验证范围见文末「复核问题收口」。

### R1 `/promotion/article/*` 直接返回 admin DTO

`internal/api/handler/app/mall/promotion/article.go:59` 返回 `promotion2.ArticleRespVO`（admin 契约），没有 app 专用 DTO：

- `createTime` 仍是裸 `time.Time` → 输出 RFC3339 字符串，Java 是毫秒时间戳。**这是 P1-6 时间格式统一唯一的漏网点**（`internal/api/contract/app/` 下已清零，app handler 引用的其他 admin DTO 也已确认无裸时间）
- 多返回 4 个内部字段：`sort`、`status`、`recommendHot`、`recommendBanner`。Java `AppArticleRespVO` 只有 `{id, title, author, categoryId, picUrl, introduction, content, createTime, browseCount, spuId}`

前端 `pages/public/richtext.vue` 只读 `title` 和 `content`，所以没有暴露出来。

### R2 提现与评价的必填校验缺失

字段已按 P1-2 补齐，但没带上 Java 的校验注解。Java service 层只校验 `price`，字段级校验全靠注解，所以 Go 这里传空会直接写入空值：

| 字段 | Java 注解 | Go 现状 |
|---|---|---|
| `AppBrokerageWithdrawCreateReqVO.userAccount` | `@NotBlank`，仅 Bank / WechatApi / AlipayApi 组 | 修复前无 |
| `AppBrokerageWithdrawCreateReqVO.userName` | `@NotBlank`，仅 Bank / WechatApi / AlipayApi 组 | 修复前无 |
| `AppBrokerageWithdrawCreateReqVO.bankName` | `@NotNull`，仅 Bank 组 | 修复前无 |
| `AppBrokerageWithdrawCreateReqVO.transferChannelCode` | `@NotNull` 仅 WechatApi 组；提供渠道时校验 `@InEnum` | 修复前无 |
| `AppTradeOrderItemCommentCreateReqVO.anonymous` | `@NotNull` | 裸 `bool`，不传默认 false |
| `AppBrokerageUserChildSummaryPageReqVO.level` | `@NotNull` `@Range` | `omitempty,oneof=1 2` |
| `AppBrokerageUserRankPageReqVO.times` | `@NotNull` | 无 |

### R3 佣金计算缺前置判断

Java `calculatePrice`：

```java
if (fixedPrice != null && fixedPrice >= 0) return fixedPrice;
if (basePrice != null && basePrice > 0 && percent != null && percent > 0) {
    return MoneyUtils.calculateRatePriceFloor(basePrice, percent);
}
return 0;
```

Go（`internal/service/mall/trade/brokerage/record.go:478`）固定佣金那段已按 P1-16 修好，但比例分支直接 `return basePrice * percent / 100`，缺 `basePrice > 0 && percent > 0` 的前置判断 —— `percent` 为负数时会算出负佣金。

### R4 售后日志向 app 端暴露内部状态字段

`internal/api/contract/app/support.go:29` 的 `AfterSaleLogResp` 有 7 个字段，Java `AppAfterSaleLogRespVO` 只有 `{id, content, createTime}`。多出的 `afterSaleId`、`beforeStatus`、`afterStatus`、`operateType` 属于内部流转状态，不该给 app 端。

### R5 快递公司 DTO 字段超集

`internal/api/contract/app/support.go:9` 的 `DeliveryExpressResp` 返回 `{id, name, code, logo}`，Java `AppDeliveryExpressRespVO` 只有 `{id, name}`。

### R6 评论字段名与 Java 不一致

Java `AppProductCommentRespVO` 是 `skuProperties`，Go app 端（`internal/api/contract/admin/mall/product/product_comment.go:119`）用的是 `properties`。

前端 `pages/goods/components/detail/comment-item.vue` 两个都不读（只读 `content`、`picUrls`、`replyContent`、`replyTime`、`scores`、`userNickname`、`userAvatar`），选哪个都不影响功能。

### R7 评论 DTO 的 13 个字段（复核后排除）

原复核将 `userId`、`anonymous`、`orderId`、`orderItemId`、`replyStatus`、`replyUserId`、`additionalContent`、`additionalPicUrls`、`additionalTime`、`spuId`、`skuId`、`descriptionScores`、`benefitScores` 列为缺失；经复查，这 13 个字段在 `35b20e6` 的 `AppProductCommentResp` 中已存在。

其中可从实体取得的字段已由 `ProductCommentService.GetAppCommentPage` 映射。三个 `additional*` 字段在 Java VO 中存在，但当前 Java / Go 实体均没有对应存储字段，因此保持 nullable 响应，不伪造追加评论数据。本次补上字段存在性回归测试；R6 的 JSON 键名差异仍单独修正。

### R8 【P0】发送短信验证码：改密码场景必然 400，且三个场景校验全缺

**定级更正**：本项初判为「不影响前端功能」，复查后确认是功能阻断，等级等同 P0。

#### R8-1 改密码拿不到验证码

前端 `sheep/hooks/useModal.js:76` 注释写明「只有 mobile 非空时才校验。因为部分场景（修改密码），不需要输入手机」，`change-password.vue:32` 调用 `getSmsCode('changePassword')`（签名 `getSmsCode(event, mobile = '')`），实际发出 `{mobile: '', scene: 3}`。

Java `AppAuthSmsSendReqVO.mobile` 只有 `@Mobile` 格式校验、允许为空，`MemberAuthServiceImpl.sendSmsCode` 对该场景专门处理：

```java
// 情况 3：如果是修改密码场景，需要查询手机号，无需前端传递
if (Objects.equals(reqVO.getScene(), SmsSceneEnum.MEMBER_UPDATE_PASSWORD.getScene())) {
    MemberUserDO user = userService.getUser(userId);
    reqVO.setMobile(user.getMobile());
}
```

Go `internal/api/contract/admin/member/member_auth.go:27` 是 `binding:"required,len=11"`，空串直接 400，请求进不了 service。

影响：用户拿不到验证码，改密码全流程走不通。**与 P0-2 的修复配套不上** —— 改密码接口已改为 `{password, code}`，但那个 `code` 根本发不出来。

#### R8-2 三个场景的业务校验全部缺失

Go `internal/service/member/auth.go:213` 的 `SendSmsCode` 只有一行透传，Java 的三段场景逻辑一个都没有：

| 场景 | Java 的处理 | Go |
|---|---|---|
| scene=3 修改密码 | 从当前登录用户查手机号并回填 | 无 |
| scene=2 修改手机号 | 校验新手机号未被他人注册，否则 `AUTH_MOBILE_USED` | 无 |
| scene=4 重置密码 | 校验手机号已注册，否则 `USER_MOBILE_NOT_EXISTS` | 无 |

scene=4 缺注册校验会对未注册号码尝试发送验证码，增加无效短信请求；底层已有按手机号的发送频率和每日额度限制，不能据此描述为无限制发送。

#### R8-3 结构性障碍：拿不到当前用户

Go 的 `/member/auth/send-sms-code`（`internal/api/router/app.go:42`）注册在 `authGroup` 但**未挂 `middleware.Auth()`**，handler 也不读 userId。Java 是 `@PermitAll` + service 内 `getLoginUserId()` 取可选登录态。

修改：路由改用 `middleware.OptionalAuth()`，`mobile` 去掉 `required`（保留非空时的 `len=11` 格式校验），service 补齐三个场景分支。

`/member/auth/validate-sms-code` 有同样的 `mobile` 必填问题（Java `AppAuthSmsValidateReqVO.mobile` 也只有 `@Mobile`），但前端当前没有任何调用点，不构成阻断，建议一并对齐。

### 路由：仍缺 7 个 Java 有、前端未用的接口

`promotion/article/list`、`promotion/article-category/list`、`member/level/list`、`member/experience-record/page`、`promotion/bargain-record/page`、`promotion/article/add-browse-count`、`promotion/banner/add-browse-count`。

前端调用的 `app/mplive/getMpLink`、`app/mplive/getRoomList`、`third/apple/login` 三个 Java 侧同样没有，属于 uniapp 的 migration 遗留，不需要实现。

---

## 复核确认已修复

| 项 | 复核方式与结果 |
|---|---|
| P0-1 | `internal/consts/pay.go:61-62` 退款 20 / 关闭 30 |
| P0-5 | 三方路由脚本比对：**方法不符 0 处** |
| P0-6 | 前端在用的 13 个接口全部接入；Java 153 / Go 150 / 前端 130，前端缺口仅剩 Java 也没有的 3 个 migration 接口 |
| P1-6 | `internal/api/contract/app/` 下裸 `time.Time` 归零；app handler 引用的 admin DTO 逐个扫描，仅 R1 一处遗留 |
| P1-11 | `reward_activity_calculator.go:88-96` 按 `result.SkuIDs` 筛出参与项再分摊 |
| P1-12 | `price_calculator_helper.go` 的 `PayPrice < 0` 钳制已删除 |
| P1-13 | 5 处全部改用 `TradeOrderCancelType*`（10/20/30/40），旧常量已删 |
| P1-15 | `CancelAfterSale`/`AgreeAfterSale`/`RefundAfterSale`/`DeliveryAfterSale` 统一走 `updateAfterSale()`（`after_sale.go:879`），带 `Status.Eq(as.Status)` 条件和 `RowsAffected != 1` 检查 |
| P1-16 | `calculatePrice` 的 `fixedPrice` 改为 `*int` + `>= 0` 判断（比例分支的遗留见 R3） |
| P1-17 | `combination_record_expire_job.go` 已创建，`wire_gen.go:159-160` 注册，job 总数 13 → 14 |
| P2-1 | `pkg/types/query_time.go:11` 的 `QueryTimeRange` 统一支持 `key`、`key[]`、`key[0]/key[1]` 三种形式，钱包流水与积分记录均已接入 |
| P2-3 | 交易错误码已迁至 1-011 号段，与 member 的 1-004 段不再冲突。脚本报出的 15 个「同码不同义」经核对全部是同义不同名（如 Java `ORDER_NOT_FOUND` vs Go `OrderNotExists`） |
| P2-7 | 分摊改回 Java 的浮点表达式 `int(float64(total) * (float64(pay) / float64(totalPay)))` 与 `i < len(items)-1` 的数组末项规则 |

复核方法：路由三方脚本比对、Java 146 个 App*VO 对 Go struct 的字段与类型比对、必填校验注解比对、错误码数值段比对、app handler 引用链上的裸时间扫描、修复点逐个读码确认。


## 复核问题收口（2026-09-08）

本轮在 `codex/java-alignment-audit` 上处理 R1～R8 与 Astra high 对 `35b20e6` 的三项 review finding。实现未使用 subagent。上文的路由差异清单保留为另行跟踪项，不将尚未实现的 Java 非前端接口标记为完成。

| 复核项 | 当前结果 | 实现 / 验证入口 |
|---|---|---|
| R1 | 文章详情和分页统一映射 App 专用 10 字段响应，时间为毫秒；不再输出四个管理字段 | `internal/api/contract/app/mall/promotion/app_article.go`；`TestAppArticleResponse` |
| R2 | 按提现类型执行校验：钱包无账户要求，银行卡 / 微信 API / 支付宝 API 校验姓名与账户，银行卡要求 bankName 非 null，微信 API 要求渠道与最低 30 分；渠道按 Java 枚举检查。评价区分遗漏 anonymous 与显式 false，系统评价同步调整；分销 level 必填；两种排行分页 times 必填并支持三种数组编码及时间解析 | `AppBrokerageWithdrawCreateReqVO.Validate`；`TestJavaReviewValidationAndResponseFields`；`TestJavaRankTimesRequired` |
| R3 | 固定佣金优先规则保留，比例分支仅在 basePrice、percent 都为正时计算，否则为 0 | `TestJavaFixedZeroCommission` 增加负数 / 零边界 |
| R4 | 售后日志 App 响应仅保留 id、content、createTime | `internal/api/contract/app/support.go`；响应键集测试 |
| R5 | 快递公司 App 响应仅保留 id、name | 同上 |
| R6 | 评论属性 JSON 键改为 skuProperties | `TestJavaReviewValidationAndResponseFields` |
| R7 | 排除误判；13 字段已存在，三个 additional 字段保持 null，新增键存在性测试 | 上文 R7 修正说明 |
| R8 | send-sms-code 使用 OptionalAuth；scene=3 必须有可信会员身份，从该用户取手机号并忽略客户端指定的其他手机号。scene=2 拒绝他人已用号码；scene=4 拒绝本租户未注册号码；App 短信限定会员场景 1～4。send / validate 请求允许省略 mobile，但底层实际发送 / 验证仍需合法号码 | `MemberAuthService.smsMobile`；`TestJavaSmsSceneDestinationsPostgres` |
| Review P1：拼团批次中止 | 一次读取活动映射，缺失活动按普通过期团处理；保留单团事务错误回滚，外层聚合错误并继续其他团 | `TestJavaExpireCombinationBatchPostgres` 验证已删除活动、单团失败不影响其他团、重试仅处理未完成团 |
| Review P2：扩展单迁移遗漏 | 新增 `000008_pay_order_extension_closed.up.sql`，仅转换扩展单旧关闭状态 20→30；不重写已提交的 000007 | `TestJavaHistoricalExtensionClosedMigrationPostgres` 验证从版本 7 升级、重复应用迁移及重复关闭回调 |
| Review P2：历史订单详情失败 | 物流公司不存在视为可选展示信息缺失；真实查询故障仍返回错误 | `TestJavaOrderDetailDeletedExpressPostgres` |

### 本轮实际验证

- 先运行受影响的 Java 对齐回归用例，使用隔离 PostgreSQL schema 和 `-race`：通过。
- `make test-integration`：393 个测试（含子测试）、151 个顶层测试、37 个有测试的包通过；无失败或跳过的测试，集成验收检查器通过。使用本地隔离 PostgreSQL 16.10 与 Redis，未对业务库执行迁移。
- `go build ./...`、`go vet ./...`、`git diff --check`：通过。构建输出过本机模块缓存写权限提示，退出状态为 0；本轮使用独立可写 GOCACHE。
- 新迁移由应用迁移流程顺序执行；尚未应用到业务数据库。000007 的历史主单状态转换仍要求停旧版本写入并确认历史数据来源，不能反复手工运行原始转换 SQL。
- 未执行真实短信发送、支付 / 转账渠道联调或 uniapp 真机联调。短信新增用例验证真实数据库中的场景、手机号选择与租户归属；拼团用例使用真实数据库验证批次状态及回滚，跨模块取消订单以测试替身注入成功 / 失败，不能替代真实退款渠道验收。

---

## 补充审计：以 uniapp 前端为起点（2026-09-08）

前面所有轮次都以「Java 结构 → Go 结构」为比对方向，发现不了 **Go 内部不一致** 和 **前端调用约定不匹配** 这两类问题。本轮改为从前端实际发出的请求和消费的响应反查，累计记录 F1～F7 七项发现及 F8 一项一致性确认。以下保留发现时的行为；本轮修复与验证结果见文末。

### F1 【P0】`userId` 上下文 key 大小写不匹配，10 个接口拿到 0

`pkg/context/context.go:10` 定义 `CtxUserIDKey = "userID"`，中间件也只以这个 key 写入（全项目仅 4 处 `c.Set`，见 `context.go:63-64`、`middleware/tenant.go:54,120`）。

但有 10 处 handler 用字面量 `c.GetInt64("userId")`（小写 d）取值。gin 的 `c.Keys` 是 `map[string]any`，key 严格区分大小写 —— 实测：

```
GetInt64("userID") = 17
GetInt64("userId") = 0   <- handler 用的 key
```

受影响接口（**全部为前端在用**）：

| 文件 | handler | 路由 | 前端调用 |
|---|---|---|---|
| `handler/app/mall/trade/brokerage/user.go:37` | `GetBrokerageUser` | `/trade/brokerage-user/get` | `brokerage.js:15` |
| 同上 `:70` | `BindBrokerageUser` | `/trade/brokerage-user/bind` | `brokerage.js:7` |
| 同上 `:81` | `GetBrokerageUserSummary` | `/trade/brokerage-user/get-summary` | `brokerage.js:22` |
| 同上 `:140` | `GetBrokerageUserChildSummaryPage` | `/trade/brokerage-user/child-summary-page` | `brokerage.js:104` |
| 同上 `:237` | `GetRankByPrice` | `/trade/brokerage-user/get-rank-by-price` | `brokerage.js:77` |
| `handler/app/member/social_user.go:31,54,69` | `Bind`/`Unbind`/`Get` | `/member/social-user/bind`、`/unbind`、`/get` | `social.js:20,39,7` |
| `handler/app/mall/promotion/combination_record.go:74,92` | `GetCombinationRecordPage`/`GetCombinationRecordDetail` | `/promotion/combination-record/page`、`/get-detail` | `combination.js:52,61` |

影响：整个分销中心（我的推广、绑定推广员、佣金统计、下级列表、佣金排名）、微信账号绑定/解绑、拼团记录，全部以 `userId = 0` 查询 —— 查不到数据，或在绑定类接口上写入错误数据。

对照：同目录下另有 53 处使用正确的 `context.GetUserId(c)`，属于局部遗漏而非全局约定问题。

修改：10 处统一改为 `context.GetUserId(c)`。建议同时给 `c.GetInt64("userId")` 加一条 lint 规则或在 `pkg/context` 暴露唯一入口，避免再次写错。

### F2 【P0】WebSocket 客服消息类型字符串不匹配，实时消息收不到

Java `WebSocketMessageTypeConstants`（前端 `pages/chat/util/constants.js` 与之逐字一致）：

```java
String KEFU_MESSAGE_TYPE = "kefu_message_type";
String KEFU_MESSAGE_ADMIN_READ = "kefu_message_read_status_change";
```

Go `internal/service/mall/promotion/kefu.go:150-157,271-274` 发的是 `"KEFU_MESSAGE"` 和 `"KEFU_MESSAGE_ADMIN_READ"` —— 两个都对不上。

前端 `pages/chat/index.vue:174-186` 的处理是：

```js
const type = data.type;
if (!type) { console.error('未知的消息类型：' + data); return; }
if (type === WebSocketMessageTypeConstants.KEFU_MESSAGE_TYPE) { ...刷新消息列表... }
```

类型对不上时既不刷新也不报错，静默丢弃。影响：客服聊天页收不到实时消息，只能靠用户手动下拉重新拉取 `/promotion/kefu-message/list`。

修改：两个常量字符串改为 Java 的取值。建议提取到 `internal/consts` 与 Java 常量一一对应，不要在 service 里写字面量。

### F3 【P1】`get-rank-by-price` 的 `times` 只认 `times[]` 一种形式

前端三个排行榜接口用了三种不同的序列化方式：

| 接口 | 前端发送 | 前端位置 |
|---|---|---|
| `get-rank-by-price` | `times=A&times=B`（重复键） | `brokerage.js:75` |
| `rank-page-by-user-count` | `times[0]=A&times[1]=B` | `promoter.vue:91-92` |
| `rank-page-by-price` | `Object.keys` 拼接，数组转 `A,B` | `brokerage.js:83` |

Go 的两个分页接口用 `types.QueryTimeRange(c.Request.URL.Query(), "times")` 兜底了三种形式（`handler/.../brokerage/user.go:152,196`），**但 `GetRankByPrice`（`:229`）用的是 `c.QueryArray("times[]")`**，只匹配 `times[]`。此外，原公共解析器没有处理表中的 `times=A,B` 形式，分页接口同样需要补齐这一形式。

前端 `commission-ranking.vue:105` 发的是重复键形式 → `timesStr` 为空 → 传给 `GetUserRankByPrice` 的 `times` 为 nil，时间范围筛选失效，排名按全时段计算。

修改：改用已有的 `types.QueryTimeRange`，与同文件另两个 handler 保持一致。

### F4 【P2】WebSocket 的 `content` 是对象，Java 是 JSON 字符串

Java `WebSocketSenderApi.sendObject` 统一走 `JsonUtils.toJsonString(messageContent)`，`content` 是**字符串**；前端 `pages/chat/index.vue:183` 相应地做 `jsonParse(data.content)`。

Go `internal/pkg/websocket/message.go:11` 的 `Content interface{}` 直接序列化为 **JSON 对象**（注释明确写「而不是字符串」）。

前端 `sheep/helper/utils.js:304` 的 `jsonParse` 带 try/catch，传对象时 `JSON.parse` 抛错后返回原值 —— 恰好还是那个对象，所以**不会崩，行为正确**，只是控制台留一条 warn。修 F2 后建议一并对齐，避免后续依赖字符串形态的代码踩坑。

### 本轮已核对无问题的

- **文件上传**：前端 `uni.uploadFile` 的 `name: 'file'`、`formData.directory`，与 Go `c.FormFile("file")` + `c.PostForm("directory")` 兜底一致（`handler/admin/infra/file.go:167-177`）
- **结算参数展开**：前端 `items[0].skuId` 形式，Go 有 `parseOrderItemsFromQuery` 及 `skuIds/counts` 两级兜底
- **钱包流水、积分记录的 `createTime`**：`QueryTimeRange` 支持重复键、`[]`、`[0]/[1]` 三种形式
- **请求头**：`Authorization`、`terminal`、`tenant-id`、`Accept` 均有处理
- **WebSocket 端点**：`/infra/ws` 路径与前端 `.env` 的 `SHOPRO_WEBSOCKET_PATH` 一致

### F5 【P2】鉴权边界差异 3 处

脚本比对 Java 的 `@PermitAll` 标注（65 个公开 / 141 个需登录）与 Go 路由的中间件分配：

| 接口 | Java | Go | 实际影响 |
|---|---|---|---|
| `POST /member/social-user/bind` | `@PermitAll` | `middleware.Auth()` | 无。前端只在登录态调用（`pay.js:391` 绑微信付款、`user/info.vue:251` 资料页绑定），H5 授权跳转回来时 token 仍在 localStorage |
| `POST /member/social-user/wxa-qrcode` | `@PermitAll` | `middleware.Auth()` | 无。前端 `s-share-modal.vue:91` 生成海报前先判 `isLogin`，未登录会弹登录框 |
| `GET /promotion/bargain-help/list` | 需登录 | 完全公开 | 他人的砍价助力列表可被匿名查看。Go handler 只用 `recordId`、不读 userId，功能正常，属信息暴露 |

前两项 Go 比 Java 严格但前端用法不受影响；第三项建议补 `middleware.Auth()`。

### F6 `PayRefundStatus` 重复定义且值不同（实际引用导致回调失败）

与 P0-1（支付订单状态两处定义）同类，修复时只清理了 `PayOrderStatus`，退款状态的重复定义留了下来：

| 常量 | `internal/consts/pay.go:44-46` | `internal/service/pay/consts.go:21-23` | Java |
|---|---|---|---|
| Waiting | 0 | 0 | 0 |
| Success | **10** | **1** | 10 |
| Failure | **20** | **2** | 20 |

复核更正：`internal/service/pay/refund.go` 使用 `internal/consts` 的 0/10/20，但另一套 0/1/2 **不是死常量**。`TradeAfterSaleService.UpdateAfterSaleRefunded` 引用了 `pay.PayRefundStatusSuccess`（1），后台退款导出也引用该套常量；另外 `TradeOrderUpdateService.UpdatePaidOrderRefunded` 将成功值硬编码为 2。

因此合法的状态 10 退款会被售后回调拒绝，后台导出也无法正确标记退款成功 / 失败。这是已经生效的行为差异，不能只按未来误用风险处理。

修复：删除 service/pay 中的重复定义，把售后回调、订单退款校验与后台退款导出全部统一为 `internal/consts` 的 0/10/20。支付服务及渠道本来就按这些值持久化，本轮不新增退款状态数据迁移。回归测试拒绝 0/1/2/20，接受 10，并验证重复成功通知只累计一次退款金额。

### 本轮另外核对无问题的

- **微信支付参数**：前端 `pay.js:180-186` 从 `JSON.parse(data.displayContent)` 取 `timeStamp`/`nonceStr`/`packageValue`/`signType`/`paySign`，Go `client/weixin/client.go:253-257` five 个键名逐字一致（`packageValue` 而非 `package`，与 WxJava 一致）
- **前端消费字段名全集**：提取前端 349 个字段名与 Go 819 个 json tag 比对，差集 116 个逐个核验后全部是前端本地变量、DIY 装修配置或 uni API 返回值。其中 `pointStock`/`pointTotalStock`/`activityType`/`limitCount` 是前端自己给 spu 对象赋的值（`s-point-block.vue:263`、`seckill.vue:272`），`couponIds` 来自装修数据，`income`/`expense` 在注释里
- **`promos` 与 `verification`**：Java 侧同样没有这两个字段，前端均用可选链（`data.promos?.length`、`userInfo.verification?.mobile`）容错，属前端历史遗留
- **同名常量值冲突**：全项目扫描仅 F6 一处

### F7 【P2】秒杀单次限购对 `0` 的语义不同

- Java `SeckillActivityServiceImpl.validateJoinSeckill`：`if (count > activity.getSingleLimitCount()) throw ...` —— **无条件校验**，`singleLimitCount = 0` 时任何购买都被拒绝
- Go `internal/service/mall/promotion/seckill_activity.go:550`：`if act.SingleLimitCount > 0 && count > act.SingleLimitCount` —— `0` 被当作「不限购」

`migrations/000001_baseline.up.sql:1526` 中该列有 `DEFAULT 0`，所以 0 值是可能出现的。正常后台配置下该值大于 0，两边行为一致；仅在配置为 0 时分歧（Java 全拒、Go 放行）。

### F8 已核对一致（本轮补充）

- **自提门店营业时间**：Java `@JsonFormat(shape = STRING, pattern = "HH:mm")` 输出 `"09:00"`，Go `support.go:73` 用 `string(item.OpeningTime)[:min(5, len(...))]` 截取同样得到 `"09:00"`，前端 `pickUpVerify.vue:25` 直接拼接显示，一致


## F 系列修复结果（2026-09-08）

本轮沿用 `codex/java-alignment-audit`，实现与验证均未使用 subagent。F1～F7 已按下表处理，F8 保持已确认一致的行为。

| 项目 | 处理结果 | 回归证据 |
|---|---|---|
| F1 | 三个 App handler 文件中的 10 处错误 key 全部改用 `context.GetUserId(c)`。后台短信模板重复的身份读取 helper 也改用公共入口。新增 AST 检查，禁止 handler 再以字符串 key 直接读取 user ID | `TestHandlersUseSharedUserIdentityAccessor`；真实数据库的排行 handler 测试用可信 userID=17 与错误 key 的干扰值 999 验证不会串用 |
| F2 | 新消息与已读通知统一为 `kefu_message_type`、`kefu_message_read_status_change`，定义在 `internal/consts/websocket.go`。客服推送按可信 tenantID、用户类型与接收人筛选连接；仅管理员通知允许省略接收人，缺会员接收人或缺租户时不广播 | `TestFrontKefuWebSocketContractAndIsolationPostgres`：真实本地 WebSocket 连接验证会员回显、管理员推送、已读通知、同 ID 不同用户类型不重复收件、跨租户与其他会员不收件 |
| F3 | 三个排行榜共用解析器支持重复键、`[]`、`[0]/[1]`、逗号分隔四种形式；get-rank-by-price 对缺失 / 非法范围返回参数错误。真实数据库测试额外发现原派生表子查询被 TenantPlugin 拒绝且错误被忽略，排名恒为 1；改用 GORM Gen 的分组聚合 Count，并传播错误 | `TestQueryTimeRangeFrontEncodings`；`TestFrontRankIdentityAndTimeEncodingsPostgres` 验证四种编码均返回正确第 2 名、排除时间范围外数据及其他租户数据 |
| F4 | WebSocket 外层 content 改为 JSON 字符串；构造消息时先序列化业务对象，外层消息再序列化；序列化错误显式返回 | 上述 WebSocket 测试按前端两次 JSON 解析流程校验事件名、conversationId、消息 id 与内容 |
| F5 | 砍价助力列表补 `middleware.Auth()`。社交绑定与生成小程序码保留已有登录要求：前端当前均先登录，没有为形式对齐而放开写入身份关联或调用平台 API 的边界 | `TestPrivateRoutesRejectAnonymous` 增加三条路由验证 |
| F6 | 删除 service/pay 的重复退款枚举；修正售后成功校验、订单退款回调中的硬编码 2、后台退款导出映射，统一使用 0/10/20。数据库既有退款写入使用正确枚举，无需新增迁移 | `TestFrontSuccessfulRefundStatusPostgres`：0/1/2/20 均拒绝，10 可完成售后，重复成功通知不重复累计退款金额 |
| F7 | 单次限购无条件检查 `count > SingleLimitCount`，配置为 0 时拒绝购买，不再解释为无限制 | `TestFrontSeckillZeroSingleLimitPostgres`：0 拒绝购买 1 件；限购 2 时允许 2 件、拒绝 3 件 |
| F8 | 营业时间 HH:mm 输出保持不变 | 本项为一致性确认，无需代码修正 |

### F 系列本轮验证范围

- 针对性用例使用真实 PostgreSQL 隔离 schema、实际 handler / service 和本地 WebSocket 连接运行，带 `-race`。初次测试揭示排名查询的租户插件拒绝与后台重复身份 helper，修复后通过。
- `make test-integration` 通过：402 个测试（含子测试）、157 个顶层测试、39 个有测试的包，无失败或跳过，PostgreSQL / Redis 验收检查器通过。
- 全量验证后补充了“缺失会员接收人或租户不得广播”的保护与断言，重新运行受影响的客服 WebSocket 回归用例（真实 PostgreSQL + WebSocket + `-race`），通过。
- `go build ./...`、`go vet ./...` 和 `git diff --check` 通过；构建输出过本机模块缓存权限提示，退出状态为 0。
- 本轮未执行业务库写入、真实支付 / 退款渠道、微信绑定平台或 uniapp 真机联调；本地 WebSocket 收发测试不代表公网代理、弱网与重连场景已验收。
