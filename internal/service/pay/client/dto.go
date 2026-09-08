package client

import (
	"time"
)

// ============ Constants ============
const (
	DisplayModeUrl    = "url"
	DisplayModeQrCode = "qr_code"
	DisplayModeApp    = "app"
	DisplayModeForm   = "form"
)

// ============ Order DTOs ============

// UnifiedOrderReq 统一下单 Request DTO
type UnifiedOrderReq struct {
	UserIP        string            `json:"userIp"`        // 用户 IP
	OutTradeNo    string            `json:"outTradeNo"`    // 外部订单号
	Subject       string            `json:"subject"`       // 商品标题
	Body          string            `json:"body"`          // 商品描述信息
	NotifyURL     string            `json:"notifyUrl"`     // 支付结果的 notify 回调地址
	ReturnURL     string            `json:"returnUrl"`     // 支付结果的 return 回调地址
	Price         int               `json:"price"`         // 支付金额，单位：分
	ExpireTime    time.Time         `json:"expireTime"`    // 支付过期时间
	ChannelExtras map[string]string `json:"channelExtras"` // 支付渠道的额外参数
	DisplayMode   string            `json:"displayMode"`   // 展示模式
}

// OrderResp 渠道支付订单 Response DTO
type OrderResp struct {
	Status           int         `json:"status"`           // 支付状态
	OutTradeNo       string      `json:"outTradeNo"`       // 外部订单号
	ChannelOrderNo   string      `json:"channelOrderNo"`   // 支付渠道编号
	ChannelUserID    string      `json:"channelUserId"`    // 支付渠道用户编号
	Price            int         `json:"price"`            // 渠道订单金额，单位：分；成功结果必须为正且与本地一致
	SuccessTime      time.Time   `json:"successTime"`      // 支付成功时间
	RawData          interface{} `json:"rawData"`          // 原始的同步/异步通知结果
	DisplayMode      string      `json:"displayMode"`      // 展示模式
	DisplayContent   string      `json:"displayContent"`   // 展示内容
	ChannelErrorCode string      `json:"channelErrorCode"` // 调用渠道的错误码
	ChannelErrorMsg  string      `json:"channelErrorMsg"`  // 调用渠道报错时，错误信息
}

// ============ Refund DTOs ============

// UnifiedRefundReq 统一退款 Request DTO
type UnifiedRefundReq struct {
	OutTradeNo  string `json:"outTradeNo"`  // 外部订单号
	OutRefundNo string `json:"outRefundNo"` // 外部退款号
	Reason      string `json:"reason"`      // 退款原因
	PayPrice    int    `json:"payPrice"`    // 支付金额
	RefundPrice int    `json:"refundPrice"` // 退款金额
	NotifyURL   string `json:"notifyUrl"`   // 退款结果的 notify 回调地址
}

// RefundResp 渠道退款订单 Response DTO
type RefundResp struct {
	Status           int         `json:"status"`           // 退款状态
	OutTradeNo       string      `json:"outTradeNo"`       // 外部订单号
	OutRefundNo      string      `json:"outRefundNo"`      // 外部退款号
	ChannelRefundNo  string      `json:"channelRefundNo"`  // 渠道退款单号
	SuccessTime      time.Time   `json:"successTime"`      // 退款成功时间
	RawData          interface{} `json:"rawData"`          // 原始的同步/异步通知结果
	ChannelErrorCode string      `json:"channelErrorCode"` // 调用渠道的错误码
	ChannelErrorMsg  string      `json:"channelErrorMsg"`  // 调用渠道报错时，错误信息
}

// ============ Transfer DTOs ============

// UnifiedTransferReq 统一转账 Request DTO
type UnifiedTransferReq struct {
	OutTradeNo    string            `json:"outTradeNo"`    // 外部订单号
	Subject       string            `json:"subject"`       // 转账标题
	Price         int               `json:"price"`         // 转账金额，单位：分
	ChannelExtras map[string]string `json:"channelExtras"` // 渠道的额外参数
	UserIP        string            `json:"userIp"`        // 用户 IP
	ChannelUserID string            `json:"channelUserId"` // 渠道用户编号
	UserName      string            `json:"userName"`      // 收款人姓名
	UserAccount   string            `json:"userAccount"`   // 收款人账号 (Alipay need)
}

// TransferResp 渠道转账 Response DTO
type TransferResp struct {
	Status             int               `json:"status"`             // 转账状态
	OutTradeNo         string            `json:"outTradeNo"`         // 外部订单号
	ChannelTransferNo  string            `json:"channelTransferNo"`  // 渠道转账单号
	SuccessTime        time.Time         `json:"successTime"`        // 转账成功时间
	RawData            interface{}       `json:"rawData"`            // 原始的同步/异步通知结果
	ChannelErrorCode   string            `json:"channelErrorCode"`   // 调用渠道的错误码
	ChannelErrorMsg    string            `json:"channelErrorMsg"`    // 调用渠道报错时，错误信息
	ChannelNotifyData  string            `json:"channelNotifyData"`  // 渠道的同步/异步通知的内容
	ChannelPackageInfo string            `json:"channelPackageInfo"` // 渠道 package 信息 (WeChat)
	ChannelExtras      map[string]string `json:"channelExtras"`      // 渠道额外参数 (AppId, MchId etc)
}
