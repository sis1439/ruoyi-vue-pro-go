package consts

// PayTransfer 转账状态枚举
// 对齐 Java: PayTransferStatusEnum
const (
	PayTransferStatusWaiting    = 0  // 等待转账
	PayTransferStatusProcessing = 5  // 转账进行中
	PayTransferStatusSuccess    = 10 // 转账成功
	PayTransferStatusClosed     = 20 // 转账关闭
)

// IsPayTransferStatusSuccess 判断是否转账成功
func IsPayTransferStatusSuccess(status int) bool {
	return status == PayTransferStatusSuccess
}

// IsPayTransferStatusClosed 判断是否转账关闭
func IsPayTransferStatusClosed(status int) bool {
	return status == PayTransferStatusClosed
}

// IsPayTransferStatusWaiting 判断是否等待转账
func IsPayTransferStatusWaiting(status int) bool {
	return status == PayTransferStatusWaiting
}

// IsPayTransferStatusProcessing 判断是否转账进行中
func IsPayTransferStatusProcessing(status int) bool {
	return status == PayTransferStatusProcessing
}

// IsPayTransferStatusWaitingOrProcessing 判断是否处于待转账或转账中状态
func IsPayTransferStatusWaitingOrProcessing(status int) bool {
	return IsPayTransferStatusWaiting(status) || IsPayTransferStatusProcessing(status)
}

// IsPayTransferStatusSuccessOrClosed 判断是否处于成功或关闭状态
func IsPayTransferStatusSuccessOrClosed(status int) bool {
	return IsPayTransferStatusSuccess(status) || IsPayTransferStatusClosed(status)
}

// PayRefundStatus 退款状态
const (
	PayRefundStatusWaiting = 0  // 未退款
	PayRefundStatusSuccess = 10 // 退款成功
	PayRefundStatusFailure = 20 // 退款失败
)

// PayTransferType 转账类型
const (
	PayTransferTypeAlipayBalance = 1 // 支付宝 - 余额
	PayTransferTypeWxBalance     = 2 // 微信 - 余额
	PayTransferTypeBankCard      = 3 // 银行卡
	PayTransferTypeWallet        = 4 // 钱包余额
)

// PayOrderStatus 支付订单状态
const (
	PayOrderStatusWaiting = 0  // 等待支付
	PayOrderStatusSuccess = 10 // 支付成功
	PayOrderStatusRefund  = 20 // 已退款
	PayOrderStatusClosed  = 30 // 支付关闭
)

// PayWalletBizType 钱包业务类型 (对齐 Java: PayWalletBizTypeEnum)
const (
	PayWalletBizTypeRecharge       = 1 // 充值
	PayWalletBizTypeRechargeRefund = 2 // 充值退款
	PayWalletBizTypePayment        = 3 // 支付
	PayWalletBizTypePaymentRefund  = 4 // 支付退款
	PayWalletBizTypeUpdateBalance  = 5 // 更新余额 (Admin)
	PayWalletBizTypeTransfer       = 6 // 转账
)

// PayChannel 支付渠道编码 (对齐 Java: PayChannelEnum)
// 参考 https://www.pingxx.com/api/支付渠道属性值.html
const (
	PayChannelWXPub     = "wx_pub"     // 微信 JSAPI 支付 - 公众号网页
	PayChannelWXLite    = "wx_lite"    // 微信小程序支付
	PayChannelWXApp     = "wx_app"     // 微信 App 支付
	PayChannelWXNative  = "wx_native"  // 微信 Native 支付
	PayChannelWXWap     = "wx_wap"     // 微信 Wap 网站支付 - H5 网页
	PayChannelWXBar     = "wx_bar"     // 微信付款码支付
	PayChannelAlipayPC  = "alipay_pc"  // 支付宝 PC 网站支付
	PayChannelAlipayWap = "alipay_wap" // 支付宝 Wap 网站支付
	PayChannelAlipayApp = "alipay_app" // 支付宝 App 支付
	PayChannelAlipayQR  = "alipay_qr"  // 支付宝扫码支付
	PayChannelAlipayBar = "alipay_bar" // 支付宝条码支付
	PayChannelMock      = "mock"       // 模拟支付
	PayChannelWallet    = "wallet"     // 钱包支付
)

// IsPayChannelAlipay 判断是否为支付宝渠道
func IsPayChannelAlipay(channelCode string) bool {
	return len(channelCode) >= 7 && channelCode[:7] == "alipay_"
}

// IsPayChannelWeixin 判断是否为微信渠道
func IsPayChannelWeixin(channelCode string) bool {
	return len(channelCode) >= 3 && channelCode[:3] == "wx_"
}

// PayNotifyStatus 支付通知状态 (对齐 Java: PayNotifyStatusEnum)
const (
	// PayNotifyStatusWaiting 等待通知
	PayNotifyStatusWaiting = 0
	// PayNotifyStatusSuccess 通知成功
	PayNotifyStatusSuccess = 10
	// PayNotifyStatusFailure 通知失败
	PayNotifyStatusFailure = 20
	// PayNotifyStatusRequestSuccess 请求成功，但是结果失败
	PayNotifyStatusRequestSuccess = 21
	// PayNotifyStatusRequestFailure 请求失败
	PayNotifyStatusRequestFailure = 22
)

// PayNotifyType 支付通知类型 (对齐 Java: PayNotifyTypeEnum)
const (
	// PayNotifyTypeOrder 支付单
	PayNotifyTypeOrder = 1
	// PayNotifyTypeRefund 退款单
	PayNotifyTypeRefund = 2
	// PayNotifyTypeTransfer 转账单
	PayNotifyTypeTransfer = 3
)

func PayChannelName(code string) string {
	return map[string]string{
		"wx_pub": "微信 JSAPI 支付", "wx_lite": "微信小程序支付", "wx_app": "微信 App 支付", "wx_native": "微信 Native 支付", "wx_wap": "微信 H5 支付", "wx_bar": "微信付款码支付", "alipay_pc": "支付宝 PC 网站支付", "alipay_wap": "支付宝 Wap 网站支付", "alipay_app": "支付宝 App 支付", "alipay_qr": "支付宝扫码支付", "alipay_bar": "支付宝条码支付", "mock": "模拟支付", "wallet": "钱包支付",
	}[code]
}
