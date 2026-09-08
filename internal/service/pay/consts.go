package pay

// PayNotifyTypeEnum 支付通知类型
const (
	PayNotifyTypeOrder    = 1 // 支付单
	PayNotifyTypeRefund   = 2 // 退款单
	PayNotifyTypeTransfer = 3 // 转账单
)

// PayNotifyStatusEnum 支付通知状态
const (
	PayNotifyStatusWaiting        = 0  // 等待通知
	PayNotifyStatusSuccess        = 10 // 通知成功
	PayNotifyStatusFailure        = 20 // 通知失败 (多次尝试，彻底失败)
	PayNotifyStatusRequestSuccess = 21 // 请求成功，但是结果失败
	PayNotifyStatusRequestFailure = 22 // 请求失败
)

// PayTransferStatusEnum 转账状态 (对齐 Java)
const (
	PayTransferStatusWaiting    = 0  // 等待转账
	PayTransferStatusProcessing = 5  // 转账进行中
	PayTransferStatusSuccess    = 10 // 转账成功
	PayTransferStatusClosed     = 20 // 转账关闭
)
