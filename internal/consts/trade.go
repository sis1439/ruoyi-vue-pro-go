package consts

// 订单状态常量
const (
	// TradeOrderStatusUnpaid 待支付
	TradeOrderStatusUnpaid = 0
	// TradeOrderStatusUndelivered 待发货
	TradeOrderStatusUndelivered = 10
	// TradeOrderStatusDelivered 待收货
	TradeOrderStatusDelivered = 20
	// TradeOrderStatusCompleted 完成
	TradeOrderStatusCompleted = 30
	// TradeOrderStatusCanceled 取消
	TradeOrderStatusCanceled = 40
)

// 配送方式常量 (对齐 Java: DeliveryTypeEnum)
const (
	// DeliveryTypeExpress 快递发货
	DeliveryTypeExpress = 1
	// DeliveryTypePickUp 用户自提
	DeliveryTypePickUp = 2
)

// 订单类型常量
const (
	// OrderTypeNormal 普通订单
	OrderTypeNormal = 1
)

// 交易订单类型常量
const (
	// TradeOrderTypeNormal 普通订单
	TradeOrderTypeNormal = 0
	// TradeOrderTypeSeckill 秒杀订单
	TradeOrderTypeSeckill = 1
	// TradeOrderTypeBargain 砍价订单
	TradeOrderTypeBargain = 2
	// TradeOrderTypeCombination 拼团订单
	TradeOrderTypeCombination = 3
	// TradeOrderTypePoint 积分订单
	TradeOrderTypePoint = 4
)

// 价格计算器优先级常量
// 数字越小优先级越高
const (
	// OrderSeckillActivity 秒杀活动计算器优先级
	OrderSeckillActivity = 8
	// OrderBargainActivity 砍价活动计算器优先级
	OrderBargainActivity = 8
	// OrderCombinationActivity 拼团活动计算器优先级
	OrderCombinationActivity = 8
	// OrderPointActivity 积分商城活动计算器优先级
	OrderPointActivity = 8
	// OrderDiscountActivity 限时折扣活动计算器优先级
	OrderDiscountActivity = 10
	// OrderRewardActivity 满减送活动计算器优先级
	OrderRewardActivity = 20
	// OrderCoupon 优惠券计算器优先级
	OrderCoupon = 30
	// OrderPointUse 积分抵扣计算器优先级
	OrderPointUse = 40
	// OrderDelivery 运费计算器优先级
	OrderDelivery = 50
	// OrderPointGive 积分赠送计算器优先级
	OrderPointGive = 999
)

// 计算器名称常量
const (
	// CalculatorNameSeckill 秒杀活动价格计算器
	CalculatorNameSeckill = "秒杀活动价格计算器"
	// CalculatorNameBargain 砍价活动价格计算器
	CalculatorNameBargain = "砍价活动价格计算器"
	// CalculatorNameCombination 拼团活动价格计算器
	CalculatorNameCombination = "拼团活动价格计算器"
	// CalculatorNamePoint 积分商城价格计算器
	CalculatorNamePoint = "积分商城价格计算器"
	// CalculatorNameDiscount 限时折扣活动价格计算器
	CalculatorNameDiscount = "限时折扣活动价格计算器"
	// CalculatorNameReward 满减送活动价格计算器
	CalculatorNameReward = "满减送活动价格计算器"
	// CalculatorNameCoupon 优惠券价格计算器
	CalculatorNameCoupon = "优惠券价格计算器"
	// CalculatorNamePointUse 积分抵扣价格计算器
	CalculatorNamePointUse = "积分抵扣价格计算器"
	// CalculatorNameDelivery 运费计算器
	CalculatorNameDelivery = "运费计算器"
	// CalculatorNamePointGive 积分赠送计算器
	CalculatorNamePointGive = "积分赠送计算器"
)

// 订单操作类型常量 (对齐 Java: TradeOrderOperateTypeEnum)
const (
	// TradeOrderOperateTypeMemberCreate 用户下单
	TradeOrderOperateTypeMemberCreate = 1
	// TradeOrderOperateTypeAdminUpdatePrice 订单价格修改
	TradeOrderOperateTypeAdminUpdatePrice = 2
	// TradeOrderOperateTypeMemberPay 用户付款成功
	TradeOrderOperateTypeMemberPay = 10
	// TradeOrderOperateTypeAdminUpdateAddress 收货地址修改
	TradeOrderOperateTypeAdminUpdateAddress = 11
	// TradeOrderOperateTypeAdminDelivery 已发货
	TradeOrderOperateTypeAdminDelivery = 20
	// TradeOrderOperateTypeMemberReceive 用户已收货
	TradeOrderOperateTypeMemberReceive = 30
	// TradeOrderOperateTypeSystemReceive 到期未收货，系统自动确认收货
	TradeOrderOperateTypeSystemReceive = 31
	// TradeOrderOperateTypeAdminPickUpReceive 管理员自提收货
	TradeOrderOperateTypeAdminPickUpReceive = 32
	// TradeOrderOperateTypeMemberComment 用户评价
	TradeOrderOperateTypeMemberComment = 33
	// TradeOrderOperateTypeSystemComment 到期未评价，系统自动评价
	TradeOrderOperateTypeSystemComment = 34
	// TradeOrderOperateTypeMemberCancel 取消订单
	TradeOrderOperateTypeMemberCancel = 40
	// TradeOrderOperateTypeSystemCancel 到期未支付，系统自动取消订单
	TradeOrderOperateTypeSystemCancel = 41
	// TradeOrderOperateTypeAdminCancelAfterSale 订单全部售后，管理员自动取消订单
	TradeOrderOperateTypeAdminCancelAfterSale = 43
	// TradeOrderOperateTypeMemberDelete 删除订单
	TradeOrderOperateTypeMemberDelete = 49
)

// 兼容旧的常量名（保留向后兼容）
const (
	// OrderOperateTypeCreate 创建订单
	OrderOperateTypeCreate = 10 // 已废弃，使用 TradeOrderOperateTypeMemberCreate
	// OrderOperateTypePay 支付
	OrderOperateTypePay = 20 // 已废弃，使用 TradeOrderOperateTypeMemberPay
	// OrderOperateTypeDelivery 发货
	OrderOperateTypeDelivery = 30 // 已废弃，使用 TradeOrderOperateTypeAdminDelivery
	// OrderOperateTypeReceive 确认收货
	OrderOperateTypeReceive = 40 // 已废弃，使用 TradeOrderOperateTypeMemberReceive
	// OrderOperateTypePickUp 自提核销
	OrderOperateTypePickUp = 50 // 已废弃，使用 TradeOrderOperateTypeAdminPickUpReceive
	// OrderOperateTypeCancel 取消订单
	OrderOperateTypeCancel = 41 // 已废弃，使用 TradeOrderOperateTypeSystemCancel
	// OrderOperateTypeRefund 退款
	OrderOperateTypeRefund = 60 // 已废弃，使用 TradeOrderOperateTypeAdminRefund
)

// 订单取消类型常量 (对齐 Java: TradeOrderCancelTypeEnum)
const (
	// TradeOrderCancelTypePayTimeout 超时未支付
	TradeOrderCancelTypePayTimeout = 10
	// TradeOrderCancelTypeAfterSaleClose 退款关闭
	TradeOrderCancelTypeAfterSaleClose = 20
	// TradeOrderCancelTypeMemberCancel 买家取消
	TradeOrderCancelTypeMemberCancel = 30
	// TradeOrderCancelTypeCombinationClose 拼团关闭
	TradeOrderCancelTypeCombinationClose = 40
)

// 兼容旧的常量名（保留向后兼容）

// 订单退款状态常量 (对齐 Java: TradeOrderRefundStatusEnum)
const (
	// TradeOrderRefundStatusNone 未退款
	TradeOrderRefundStatusNone = 0
	// TradeOrderRefundStatusPart 部分退款
	TradeOrderRefundStatusPart = 10
	// TradeOrderRefundStatusAll 全部退款
	TradeOrderRefundStatusAll = 20
)

// 兼容旧的常量名（保留向后兼容）
const (
	// OrderRefundStatusNone 无退款
	OrderRefundStatusNone = 0 // 已废弃，使用 TradeOrderRefundStatusNone
	// OrderRefundStatusApply 申请退款
	OrderRefundStatusApply = 10 // 已废弃
	// OrderRefundStatusAuditing 审核中
	OrderRefundStatusAuditing = 20 // 已废弃
	// OrderRefundStatusRefunded 已退款
	OrderRefundStatusRefunded = 30 // 已废弃，使用 TradeOrderRefundStatusAll
)

// 自提核销码常量
const (
	// PickUpVerifyCodeLength 核销码长度
	PickUpVerifyCodeLength = 8
)

// 配送状态常量
const (
	// DeliveryStatusEnabled 启用状态
	DeliveryStatusEnabled = 1
	// DeliveryStatusDisabled 禁用状态
	DeliveryStatusDisabled = 0
)

// 售后状态常量
const (
	// AfterSaleStatusNone 无售后申请 (Go 扩展，Java 无此状态)
	AfterSaleStatusNone = 0
	// AfterSaleStatusApply 申请中
	AfterSaleStatusApply = 10
	// AfterSaleStatusSellerAgree 卖家通过，等待买家退货
	AfterSaleStatusSellerAgree = 20
	// AfterSaleStatusBuyerDelivery 待卖家收货
	AfterSaleStatusBuyerDelivery = 30
	// AfterSaleStatusWaitRefund 等待平台退款
	AfterSaleStatusWaitRefund = 40
	// AfterSaleStatusComplete 完成
	AfterSaleStatusComplete = 50
	// AfterSaleStatusBuyerCancel 买家取消售后
	AfterSaleStatusBuyerCancel = 61
	// AfterSaleStatusSellerDisagree 卖家拒绝
	AfterSaleStatusSellerDisagree = 62
	// AfterSaleStatusSellerRefuse 卖家拒绝收货
	AfterSaleStatusSellerRefuse = 63
)

// 售后方式常量
const (
	// AfterSaleWayRefund 仅退款
	AfterSaleWayRefund = 10
	// AfterSaleWayReturnAndRefund 退货退款
	AfterSaleWayReturnAndRefund = 20
)

// 售后类型常量
const (
	// AfterSaleTypeInSale 售中
	AfterSaleTypeInSale = 10
	// AfterSaleTypeAfterSale 售后
	AfterSaleTypeAfterSale = 20
)

// 售后操作类型常量 (对齐 Java: AfterSaleOperateTypeEnum)
const (
	// AfterSaleOperateTypeMemberCreate 会员申请退款
	AfterSaleOperateTypeMemberCreate = 10
	// AfterSaleOperateTypeAdminAgreeApply 商家同意退款
	AfterSaleOperateTypeAdminAgreeApply = 11
	// AfterSaleOperateTypeAdminDisagreeApply 商家拒绝退款
	AfterSaleOperateTypeAdminDisagreeApply = 12
	// AfterSaleOperateTypeMemberDelivery 会员填写退货物流信息
	AfterSaleOperateTypeMemberDelivery = 20
	// AfterSaleOperateTypeAdminAgreeReceive 商家收货
	AfterSaleOperateTypeAdminAgreeReceive = 21
	// AfterSaleOperateTypeAdminDisagreeReceive 商家拒绝收货
	AfterSaleOperateTypeAdminDisagreeReceive = 22
	// AfterSaleOperateTypeAdminRefund 商家发起退款
	AfterSaleOperateTypeAdminRefund = 30
	// AfterSaleOperateTypeSystemRefundSuccess 退款成功
	AfterSaleOperateTypeSystemRefundSuccess = 31
	// AfterSaleOperateTypeSystemRefundFail 退款失败
	AfterSaleOperateTypeSystemRefundFail = 32
	// AfterSaleOperateTypeMemberCancel 会员取消退款
	AfterSaleOperateTypeMemberCancel = 40
)

// 分销佣金提现状态常量
const (
	// BrokerageWithdrawStatusAuditing 审核中
	BrokerageWithdrawStatusAuditing = 0
	// BrokerageWithdrawStatusAuditSuccess 审核通过
	BrokerageWithdrawStatusAuditSuccess = 10
	// BrokerageWithdrawStatusWithdrawSuccess 提现成功
	BrokerageWithdrawStatusWithdrawSuccess = 11
	// BrokerageWithdrawStatusAuditFail 审核不通过
	BrokerageWithdrawStatusAuditFail = 20
	// BrokerageWithdrawStatusWithdrawFail 提现失败
	BrokerageWithdrawStatusWithdrawFail = 21
)

// 分销佣金提现类型常量 (对齐 Java: BrokerageWithdrawTypeEnum)
const (
	// BrokerageWithdrawTypeWallet 钱包
	BrokerageWithdrawTypeWallet = 1
	// BrokerageWithdrawTypeBank 银行卡
	BrokerageWithdrawTypeBank = 2
	// BrokerageWithdrawTypeWechatQR 微信收款码
	BrokerageWithdrawTypeWechatQR = 3
	// BrokerageWithdrawTypeAlipayQR 支付宝收款码
	BrokerageWithdrawTypeAlipayQR = 4
	// BrokerageWithdrawTypeWechatAPI 微信零钱
	BrokerageWithdrawTypeWechatAPI = 5
	// BrokerageWithdrawTypeAlipayAPI 支付宝余额
	BrokerageWithdrawTypeAlipayAPI = 6
)

// 兼容旧的常量名（保留向后兼容）
const (
	// BrokerageWithdrawTypeWechat 微信 API (已废弃)
	BrokerageWithdrawTypeWechat = 3 // 已废弃，使用 BrokerageWithdrawTypeWechatQR
	// BrokerageWithdrawTypeAlipay 支付宝 API (已废弃)
	BrokerageWithdrawTypeAlipay = 4 // 已废弃，使用 BrokerageWithdrawTypeAlipayQR
)

// 分销佣金记录业务类型常量
const (
	// BrokerageRecordBizTypeOrder 分销订单
	BrokerageRecordBizTypeOrder = 1
	// BrokerageRecordBizTypeWithdraw 佣金提现
	BrokerageRecordBizTypeWithdraw = 2
	// BrokerageRecordBizTypeWithdrawReject 提现驳回
	BrokerageRecordBizTypeWithdrawReject = 3
)

// 分销佣金记录状态常量
const (
	// BrokerageRecordStatusWait 待结算
	BrokerageRecordStatusWait = 0
	// BrokerageRecordStatusSettlement 已结算
	BrokerageRecordStatusSettlement = 1
	// BrokerageRecordStatusCancel 已取消
	BrokerageRecordStatusCancel = 2
)

// 分销用户等级常量
const (
	// BrokerageUserLevelOne 一级
	BrokerageUserLevelOne = 1
	// BrokerageUserLevelTwo 二级
	BrokerageUserLevelTwo = 2
)

// 分销绑定模式常量 (对齐 Java: BrokerageBindModeEnum)
const (
	// BrokerageBindModeAnytime 首次绑定 - 只要用户没有推广人，随时都可以绑定分销关系
	BrokerageBindModeAnytime = 1
	// BrokerageBindModeRegister 注册绑定 - 仅新用户注册时才能绑定推广关系
	BrokerageBindModeRegister = 2
	// BrokerageBindModeOverride 覆盖绑定 - 每次扫码都覆盖
	BrokerageBindModeOverride = 3
)

// 分销启用条件常量 (对齐 Java: BrokerageEnabledConditionEnum)
const (
	// BrokerageEnabledConditionAll 人人分销 - 所有用户都可以分销
	BrokerageEnabledConditionAll = 1
	// BrokerageEnabledConditionAdmin 指定分销 - 仅可后台手动设置推广员
	BrokerageEnabledConditionAdmin = 2
)

// 配送计费方式常量 (对齐 Java: DeliveryExpressChargeModeEnum)
const (
	// DeliveryExpressChargeModeCount 按件
	DeliveryExpressChargeModeCount = 1
	// DeliveryExpressChargeModeWeight 按重量
	DeliveryExpressChargeModeWeight = 2
	// DeliveryExpressChargeModeVolume 按体积
	DeliveryExpressChargeModeVolume = 3
)

// 订单项售后状态常量 (对齐 Java: TradeOrderItemAfterSaleStatusEnum)
const (
	// TradeOrderItemAfterSaleStatusNone 无
	TradeOrderItemAfterSaleStatusNone = 0
	// TradeOrderItemAfterSaleStatusApply 申请中
	TradeOrderItemAfterSaleStatusApply = 10
	// TradeOrderItemAfterSaleStatusSuccess 成功
	TradeOrderItemAfterSaleStatusSuccess = 20
)

// 快递查询渠道常量 (对齐 Java: ExpressClientEnum)
const (
	// ExpressClientKuaidi100 快递100
	ExpressClientKuaidi100 = "kuaidi100"
	// ExpressClientBird 快递鸟
	ExpressClientBird = "bird"
)

// 交易配置默认值常量
const (
	// DefaultPayTimeoutMinutes 默认支付超时时间（分钟）
	DefaultPayTimeoutMinutes = 120
	// DefaultAfterSaleDeadlineDays 默认售后期限（天）
	DefaultAfterSaleDeadlineDays = 7
	// DefaultAutoReceiveDays 默认自动收货天数
	DefaultAutoReceiveDays = 7
	// DefaultAutoCommentDays 默认自动评价天数
	DefaultAutoCommentDays = 7
)

// 分页默认值常量
const (
	// DefaultPageNo 默认页码
	DefaultPageNo = 1
)

func BrokerageRecordStatusName(status int) string {
	return map[int]string{0: "待结算", 1: "已结算", 2: "已取消"}[status]
}
func BrokerageWithdrawStatusName(status int) string {
	return map[int]string{0: "审核中", 10: "审核通过", 11: "提现成功", 20: "审核不通过", 21: "提现失败"}[status]
}
func BrokerageWithdrawTypeName(kind int) string {
	return map[int]string{1: "钱包", 2: "银行卡", 3: "微信收款码", 4: "支付宝收款码", 5: "微信零钱", 6: "支付宝余额"}[kind]
}
