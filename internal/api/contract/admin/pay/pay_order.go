package pay

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type PayOrderPageReq struct {
	pagination.PageParam
	AppID           int64    `form:"appId"`
	ChannelCode     string   `form:"channelCode"`
	MerchantOrderId string   `form:"merchantOrderId"`
	Subject         string   `form:"subject"`
	No              string   `form:"no"`
	Status          *int     `form:"status"`
	CreateTime      []string `form:"createTime[]"` // time range
}

type PayOrderExportReq struct {
	AppID           int64    `form:"appId"`
	ChannelCode     string   `form:"channelCode"`
	MerchantOrderId string   `form:"merchantOrderId"`
	Subject         string   `form:"subject"`
	No              string   `form:"no"`
	Status          *int     `form:"status"`
	CreateTime      []string `form:"createTime[]"`
}

type PayOrderSubmitReq struct {
	ID            int64             `json:"id" binding:"required"`
	ChannelCode   string            `json:"channelCode" binding:"required"`
	ChannelExtras map[string]string `json:"channelExtras"`
	DisplayMode   string            `json:"displayMode"` // PayOrderDisplayModeEnum
	ReturnUrl     string            `json:"returnUrl"`
}

type PayOrderCreateReq struct {
	AppKey          string    `json:"appKey" binding:"required"`
	UserIP          string    `json:"userIp" binding:"required"`
	MerchantOrderId string    `json:"merchantOrderId" binding:"required"`
	Subject         string    `json:"subject" binding:"required"`
	Body            string    `json:"body"`
	Price           int       `json:"price" binding:"required,min=0"`
	ExpireTime      time.Time `json:"expireTime" binding:"required"`
}

type PayOrderNotifyReq struct {
	MerchantOrderId string `json:"merchantOrderId" binding:"required"`
	PayOrderID      int64  `json:"payOrderId" binding:"required"`
}

type PayOrderResp struct {
	ID              int64               `json:"id"`
	AppID           int64               `json:"appId"`
	AppName         string              `json:"appName"` // From PayApp
	ChannelID       int64               `json:"channelId"`
	ChannelCode     string              `json:"channelCode"`
	MerchantOrderId string              `json:"merchantOrderId"`
	Subject         string              `json:"subject"`
	Body            string              `json:"body"`
	NotifyURL       string              `json:"notifyUrl"`
	Price           int64               `json:"price"` // 改为 int64
	ChannelFeeRate  float64             `json:"channelFeeRate"`
	ChannelFeePrice int                 `json:"channelFeePrice"`
	Status          int                 `json:"status"`
	UserIP          string              `json:"userIp"`
	ExpireTime      types.JsonDateTime  `json:"expireTime"`
	SuccessTime     *types.JsonDateTime `json:"successTime"`
	ExtensionID     int64               `json:"extensionId"`
	No              string              `json:"no"`
	RefundPrice     int64               `json:"refundPrice"` // 改为 int64
	ChannelUserID   string              `json:"channelUserId"`
	ChannelOrderNo  string              `json:"channelOrderNo"`
	CreateTime      types.JsonDateTime  `json:"createTime"`
	UpdateTime      types.JsonDateTime  `json:"updateTime"`
	Creator         string              `json:"creator"`
	Updater         string              `json:"updater"`
}

type PayOrderDetailsResp struct {
	PayOrderResp
	Extension *PayOrderExtensionResp `json:"extension"`
	App       *PayAppResp            `json:"app"`
}

type PayOrderExtensionResp struct {
	ID                int64     `json:"id"`
	No                string    `json:"no"`
	OrderID           int64     `json:"orderId"`
	ChannelID         int64     `json:"channelId"`
	ChannelCode       string    `json:"channelCode"`
	UserIP            string    `json:"userIp"`
	Status            int       `json:"status"`
	ChannelExtras     string    `json:"channelExtras"`
	ChannelErrorCode  string    `json:"channelErrorCode"`
	ChannelErrorMsg   string    `json:"channelErrorMsg"`
	ChannelNotifyData string    `json:"channelNotifyData"`
	CreateTime        time.Time `json:"createTime"`
}

type PayOrderSubmitResp struct {
	Status         int    `json:"status"`
	DisplayMode    string `json:"displayMode"`
	DisplayContent string `json:"displayContent"`
}
