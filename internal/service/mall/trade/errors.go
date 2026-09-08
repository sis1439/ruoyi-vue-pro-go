package trade

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
)

// 交易模块错误码定义 (对齐 Java ErrorCodeConstants)
// Java 已定义的业务条件沿用 1-011 对应错误码。
// Go 独有的通用错误使用 1-011-900-xxx 扩展段，避免与 Java 具体业务条件撞码。

const (
	// ========== 价格计算相关错误码 ==========

	// 价格计算基础错误 (1011900000-1011904099)
	ErrorCodePriceCalculateError     = 1011900000 // 价格计算失败
	ErrorCodePriceCalculateItemEmpty = 1011900001 // 价格计算商品为空
	ErrorCodePriceCalculateItemError = 1011900002 // 价格计算商品错误
	ErrorCodePriceCalculateUserError = 1011900003 // 价格计算用户错误

	// 商品相关错误 (1011900004-1011904199)
	ErrorCodeProductNotExists      = 1011900004 // 商品不存在
	ErrorCodeProductNotEnable      = 1011900005 // 商品未启用
	ErrorCodeProductStockNotEnough = 1011900006 // 商品库存不足
	ErrorCodeProductSkuNotExists   = 1011900007 // 商品SKU不存在
	ErrorCodeProductSkuNotEnable   = 1011900008 // 商品SKU未启用

	// 优惠券相关错误 (1011900009-1011904299)
	ErrorCodeCouponNotExists = 1011900009 // 优惠券不存在
	ErrorCodeCouponNotMatch  = 1011900010 // 优惠券不匹配
	ErrorCodeCouponUsed      = 1011900011 // 优惠券已使用
	ErrorCodeCouponExpired   = 1011900012 // 优惠券已过期
	ErrorCodeCouponNotStart  = 1011900013 // 优惠券未开始
	ErrorCodeCouponNotEnough = 1011900014 // 优惠券数量不足

	// 积分相关错误 (1011000038-1011904399)
	ErrorCodePointNotEnough      = 1011000038 // 积分不足
	ErrorCodePointCalculateError = 1011900015 // 积分计算错误

	// 活动相关错误 (1011900016-1011904499)
	ErrorCodeActivityNotExists      = 1011900016 // 活动不存在
	ErrorCodeActivityNotStart       = 1011900017 // 活动未开始
	ErrorCodeActivityExpired        = 1011900018 // 活动已结束
	ErrorCodeActivityNotMatch       = 1011900019 // 活动不匹配
	ErrorCodeActivityStockNotEnough = 1011900020 // 活动库存不足

	// 运费相关错误 (1011003005-1011904599)
	ErrorCodeDeliveryNotSupport        = 1011003005 // 不支持配送
	ErrorCodeDeliveryTemplateNotExists = 1011003001 // 运费模板不存在
	ErrorCodeDeliveryCalculateError    = 1011900021 // 运费计算错误

	// ========== 订单操作相关错误码 ==========

	// 订单基础错误 (1011000011-1011905099)
	ErrorCodeOrderNotExists    = 1011000011 // 订单不存在
	ErrorCodeOrderStatusError  = 1011900022 // 订单状态错误
	ErrorCodeOrderUserNotMatch = 1011900023 // 订单用户不匹配
	ErrorCodeOrderCreateError  = 1011900024 // 订单创建失败
	ErrorCodeOrderUpdateError  = 1011900025 // 订单更新失败
	ErrorCodeOrderDeleteError  = 1011000029 // 订单删除失败

	// 订单支付相关错误 (1011900026-1011905199)
	ErrorCodeOrderNotPaid        = 1011900026 // 订单未支付
	ErrorCodeOrderAlreadyPaid    = 1011900027 // 订单已支付
	ErrorCodeOrderPayError       = 1011900028 // 订单支付失败
	ErrorCodeOrderPayTimeout     = 1011900029 // 订单支付超时
	ErrorCodeOrderPayAmountError = 1011000016 // 订单支付金额错误

	// 订单发货相关错误 (1011000018-1011905299)
	ErrorCodeOrderNotDelivered     = 1011000018 // 订单未发货
	ErrorCodeOrderAlreadyDelivered = 1011900030 // 订单已发货
	ErrorCodeOrderDeliveryError    = 1011900031 // 订单发货失败
	ErrorCodeOrderLogisticsError   = 1011900032 // 物流信息错误

	// 订单收货相关错误 (1011900033-1011905399)
	ErrorCodeOrderNotReceived     = 1011900033 // 订单未收货
	ErrorCodeOrderAlreadyReceived = 1011900034 // 订单已收货
	ErrorCodeOrderReceiveError    = 1011900035 // 订单收货失败

	// 订单取消相关错误 (1011900036-1011905499)
	ErrorCodeOrderNotCanceled     = 1011900036 // 订单未取消
	ErrorCodeOrderAlreadyCanceled = 1011900037 // 订单已取消
	ErrorCodeOrderCancelError     = 1011900038 // 订单取消失败
	ErrorCodeOrderCancelNotAllow  = 1011000025 // 订单不允许取消

	// 订单退款相关错误 (1011900039-1011905599)
	ErrorCodeOrderRefundError       = 1011900039 // 订单退款失败
	ErrorCodeOrderRefundAmountError = 1011900040 // 退款金额错误
	ErrorCodeOrderRefundNotAllow    = 1011900041 // 订单不允许退款

	// 订单核销相关错误 (1011900042-1011905699)
	ErrorCodeOrderPickUpError     = 1011900042 // 订单核销失败
	ErrorCodeOrderNotPickUp       = 1011000030 // 非自提订单
	ErrorCodeOrderPickUpCodeError = 1011900043 // 核销码错误
	ErrorCodeOrderAlreadyPickUp   = 1011900044 // 订单已核销

	// 订单评价相关错误 (1011900045-1011905799)
	ErrorCodeOrderCommentError    = 1011900045 // 订单评价失败
	ErrorCodeOrderAlreadyComment  = 1011000020 // 订单已评价
	ErrorCodeOrderCommentNotAllow = 1011000019 // 订单不允许评价

	// ========== 售后相关错误码 ==========

	// 售后基础错误 (1011000100-1011906099)
	ErrorCodeAfterSaleNotExists   = 1011000100 // 售后单不存在
	ErrorCodeAfterSaleStatusError = 1011900046 // 售后单状态错误
	ErrorCodeAfterSaleCreateError = 1011900047 // 售后单创建失败
	ErrorCodeAfterSaleUpdateError = 1011000107 // 售后单更新失败

	// ========== 购物车相关错误码 ==========

	// 购物车基础错误 (1011002000-1011907099)
	ErrorCodeCartNotExists   = 1011002000 // 购物车项不存在
	ErrorCodeCartAddError    = 1011900048 // 添加购物车失败
	ErrorCodeCartUpdateError = 1011900049 // 更新购物车失败
	ErrorCodeCartDeleteError = 1011900050 // 删除购物车失败
	ErrorCodeCartCountError  = 1011900051 // 购物车数量错误

	// ========== 分销相关错误码 (1011007xxx) ==========
	// 对齐 Java: ErrorCodeConstants.BROKERAGE_* (1_011_007_xxx)

	// 分销用户相关错误 (1011007000-1011007099)
	ErrorCodeBrokerageUserNotExists            = 1011007000 // 分销用户不存在
	ErrorCodeBrokerageUserFrozenPriceNotEnough = 1011007001 // 用户冻结佣金数量不足
	ErrorCodeBrokerageBindSelf                 = 1011007002 // 不能绑定自己
	ErrorCodeBrokerageBindUserNotEnabled       = 1011007003 // 绑定用户没有推广资格
	ErrorCodeBrokerageBindConditionAdmin       = 1011007004 // 仅可在后台绑定推广员
	ErrorCodeBrokerageBindModeRegister         = 1011007005 // 只有在注册时可以绑定
	ErrorCodeBrokerageBindOverride             = 1011007006 // 已绑定了推广人
	ErrorCodeBrokerageBindLoop                 = 1011007007 // 下级不能绑定自己的上级
	ErrorCodeBrokerageUserLevelNotSupport      = 1011007008 // 目前只支持 level 小于等于 2
	ErrorCodeBrokerageUserExists               = 1011007009 // 分销用户已存在

	// 分销提现相关错误 (1011008000-1011008099)
	ErrorCodeBrokerageWithdrawNotExists = 1011008000 // 佣金提现记录不存在
)

// 错误消息映射表 (对齐 Java 版本的错误消息)
var errorMessages = map[int]string{
	// 价格计算相关错误消息
	ErrorCodePriceCalculateError:     "价格计算失败",
	ErrorCodePriceCalculateItemEmpty: "价格计算商品为空",
	ErrorCodePriceCalculateItemError: "价格计算商品错误",
	ErrorCodePriceCalculateUserError: "价格计算用户错误",

	ErrorCodeProductNotExists:      "商品不存在",
	ErrorCodeProductNotEnable:      "商品未启用",
	ErrorCodeProductStockNotEnough: "商品库存不足",
	ErrorCodeProductSkuNotExists:   "商品SKU不存在",
	ErrorCodeProductSkuNotEnable:   "商品SKU未启用",

	ErrorCodeCouponNotExists: "优惠券不存在",
	ErrorCodeCouponNotMatch:  "优惠券不匹配",
	ErrorCodeCouponUsed:      "优惠券已使用",
	ErrorCodeCouponExpired:   "优惠券已过期",
	ErrorCodeCouponNotStart:  "优惠券未开始",
	ErrorCodeCouponNotEnough: "优惠券数量不足",

	ErrorCodePointNotEnough:      "积分不足",
	ErrorCodePointCalculateError: "积分计算错误",

	ErrorCodeActivityNotExists:      "活动不存在",
	ErrorCodeActivityNotStart:       "活动未开始",
	ErrorCodeActivityExpired:        "活动已结束",
	ErrorCodeActivityNotMatch:       "活动不匹配",
	ErrorCodeActivityStockNotEnough: "活动库存不足",

	ErrorCodeDeliveryNotSupport:        "不支持配送",
	ErrorCodeDeliveryTemplateNotExists: "运费模板不存在",
	ErrorCodeDeliveryCalculateError:    "运费计算错误",

	// 订单操作相关错误消息
	ErrorCodeOrderNotExists:    "订单不存在",
	ErrorCodeOrderStatusError:  "订单状态错误",
	ErrorCodeOrderUserNotMatch: "订单用户不匹配",
	ErrorCodeOrderCreateError:  "订单创建失败",
	ErrorCodeOrderUpdateError:  "订单更新失败",
	ErrorCodeOrderDeleteError:  "订单删除失败",

	ErrorCodeOrderNotPaid:        "订单未支付",
	ErrorCodeOrderAlreadyPaid:    "订单已支付",
	ErrorCodeOrderPayError:       "订单支付失败",
	ErrorCodeOrderPayTimeout:     "订单支付超时",
	ErrorCodeOrderPayAmountError: "订单支付金额错误",

	ErrorCodeOrderNotDelivered:     "订单未发货",
	ErrorCodeOrderAlreadyDelivered: "订单已发货",
	ErrorCodeOrderDeliveryError:    "订单发货失败",
	ErrorCodeOrderLogisticsError:   "物流信息错误",

	ErrorCodeOrderNotReceived:     "订单未收货",
	ErrorCodeOrderAlreadyReceived: "订单已收货",
	ErrorCodeOrderReceiveError:    "订单收货失败",

	ErrorCodeOrderNotCanceled:     "订单未取消",
	ErrorCodeOrderAlreadyCanceled: "订单已取消",
	ErrorCodeOrderCancelError:     "订单取消失败",
	ErrorCodeOrderCancelNotAllow:  "订单不允许取消",

	ErrorCodeOrderRefundError:       "订单退款失败",
	ErrorCodeOrderRefundAmountError: "退款金额错误",
	ErrorCodeOrderRefundNotAllow:    "订单不允许退款",

	ErrorCodeOrderPickUpError:     "订单核销失败",
	ErrorCodeOrderNotPickUp:       "非自提订单",
	ErrorCodeOrderPickUpCodeError: "核销码错误",
	ErrorCodeOrderAlreadyPickUp:   "订单已核销",

	ErrorCodeOrderCommentError:    "订单评价失败",
	ErrorCodeOrderAlreadyComment:  "订单已评价",
	ErrorCodeOrderCommentNotAllow: "订单不允许评价",

	// 售后相关错误消息
	ErrorCodeAfterSaleNotExists:   "售后单不存在",
	ErrorCodeAfterSaleStatusError: "售后单状态错误",
	ErrorCodeAfterSaleCreateError: "售后单创建失败",
	ErrorCodeAfterSaleUpdateError: "售后单更新失败",

	// 购物车相关错误消息
	ErrorCodeCartNotExists:   "购物车项不存在",
	ErrorCodeCartAddError:    "添加购物车失败",
	ErrorCodeCartUpdateError: "更新购物车失败",
	ErrorCodeCartDeleteError: "删除购物车失败",
	ErrorCodeCartCountError:  "购物车数量错误",

	// 分销相关错误消息
	ErrorCodeBrokerageUserNotExists:            "分销用户不存在",
	ErrorCodeBrokerageUserFrozenPriceNotEnough: "用户冻结佣金数量不足",
	ErrorCodeBrokerageBindSelf:                 "不能绑定自己",
	ErrorCodeBrokerageBindUserNotEnabled:       "绑定用户没有推广资格",
	ErrorCodeBrokerageBindConditionAdmin:       "仅可在后台绑定推广员",
	ErrorCodeBrokerageBindModeRegister:         "只有在注册时可以绑定",
	ErrorCodeBrokerageBindOverride:             "已绑定了推广人",
	ErrorCodeBrokerageBindLoop:                 "下级不能绑定自己的上级",
	ErrorCodeBrokerageUserLevelNotSupport:      "目前只支持 level 小于等于 2",
	ErrorCodeBrokerageUserExists:               "分销用户已存在",
	ErrorCodeBrokerageWithdrawNotExists:        "佣金提现记录不存在",
}

// NewTradeError 创建交易模块业务错误
func NewTradeError(code int) error {
	if msg, exists := errorMessages[code]; exists {
		return errors.NewBizError(code, msg)
	}
	return errors.NewBizError(code, "未知错误")
}

// NewTradeErrorWithMsg 创建交易模块业务错误（自定义消息）
func NewTradeErrorWithMsg(code int, msg string) error {
	return errors.NewBizError(code, msg)
}

// 便捷的错误创建函数

// 价格计算相关错误
func ErrPriceCalculateError() error     { return NewTradeError(ErrorCodePriceCalculateError) }
func ErrPriceCalculateItemEmpty() error { return NewTradeError(ErrorCodePriceCalculateItemEmpty) }
func ErrProductNotExists() error        { return NewTradeError(ErrorCodeProductNotExists) }
func ErrProductStockNotEnough() error   { return NewTradeError(ErrorCodeProductStockNotEnough) }
func ErrCouponNotExists() error         { return NewTradeError(ErrorCodeCouponNotExists) }
func ErrCouponNotMatch() error          { return NewTradeError(ErrorCodeCouponNotMatch) }
func ErrPointNotEnough() error          { return NewTradeError(ErrorCodePointNotEnough) }
func ErrActivityNotExists() error       { return NewTradeError(ErrorCodeActivityNotExists) }
func ErrDeliveryNotSupport() error      { return NewTradeError(ErrorCodeDeliveryNotSupport) }

// 订单操作相关错误
func ErrOrderNotExists() error        { return NewTradeError(ErrorCodeOrderNotExists) }
func ErrOrderStatusError() error      { return NewTradeError(ErrorCodeOrderStatusError) }
func ErrOrderUserNotMatch() error     { return NewTradeError(ErrorCodeOrderUserNotMatch) }
func ErrOrderAlreadyPaid() error      { return NewTradeError(ErrorCodeOrderAlreadyPaid) }
func ErrOrderNotDelivered() error     { return NewTradeError(ErrorCodeOrderNotDelivered) }
func ErrOrderAlreadyDelivered() error { return NewTradeError(ErrorCodeOrderAlreadyDelivered) }
func ErrOrderNotReceived() error      { return NewTradeError(ErrorCodeOrderNotReceived) }
func ErrOrderAlreadyReceived() error  { return NewTradeError(ErrorCodeOrderAlreadyReceived) }
func ErrOrderAlreadyCanceled() error  { return NewTradeError(ErrorCodeOrderAlreadyCanceled) }
func ErrOrderCancelNotAllow() error   { return NewTradeError(ErrorCodeOrderCancelNotAllow) }
func ErrOrderRefundNotAllow() error   { return NewTradeError(ErrorCodeOrderRefundNotAllow) }
func ErrOrderNotPickUp() error        { return NewTradeError(ErrorCodeOrderNotPickUp) }
func ErrOrderPickUpCodeError() error  { return NewTradeError(ErrorCodeOrderPickUpCodeError) }
func ErrOrderAlreadyPickUp() error    { return NewTradeError(ErrorCodeOrderAlreadyPickUp) }

// 分销相关错误
func ErrBrokerageUserNotExists() error { return NewTradeError(ErrorCodeBrokerageUserNotExists) }
func ErrBrokerageUserFrozenPriceNotEnough() error {
	return NewTradeError(ErrorCodeBrokerageUserFrozenPriceNotEnough)
}
func ErrBrokerageBindSelf() error { return NewTradeError(ErrorCodeBrokerageBindSelf) }
func ErrBrokerageBindUserNotEnabled() error {
	return NewTradeError(ErrorCodeBrokerageBindUserNotEnabled)
}
func ErrBrokerageBindConditionAdmin() error {
	return NewTradeError(ErrorCodeBrokerageBindConditionAdmin)
}
func ErrBrokerageBindModeRegister() error { return NewTradeError(ErrorCodeBrokerageBindModeRegister) }
func ErrBrokerageBindOverride() error     { return NewTradeError(ErrorCodeBrokerageBindOverride) }
func ErrBrokerageBindLoop() error         { return NewTradeError(ErrorCodeBrokerageBindLoop) }
func ErrBrokerageUserLevelNotSupport() error {
	return NewTradeError(ErrorCodeBrokerageUserLevelNotSupport)
}
func ErrBrokerageUserExists() error        { return NewTradeError(ErrorCodeBrokerageUserExists) }
func ErrBrokerageWithdrawNotExists() error { return NewTradeError(ErrorCodeBrokerageWithdrawNotExists) }
