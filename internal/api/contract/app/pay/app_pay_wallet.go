package pay

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

type AppPayWalletRechargeCreateReq struct {
	PayPrice  *int   `json:"payPrice"`
	PackageID *int64 `json:"packageId"`
}

// AppPayWalletPackageResp 对齐 Java AppPayWalletPackageRespVO
type AppPayWalletPackageResp struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PayPrice   int    `json:"payPrice"`
	BonusPrice int    `json:"bonusPrice"`
}

// AppPayWalletTransactionResp 对齐 Java AppPayWalletTransactionRespVO
type AppPayWalletTransactionResp struct {
	BizType    int                `json:"bizType"`
	Price      int64              `json:"price"`
	Title      string             `json:"title"`
	CreateTime types.JsonDateTime `json:"createTime"`
}

type AppPayWalletResp struct {
	Balance       int `json:"balance"`
	TotalExpense  int `json:"totalExpense"`
	TotalRecharge int `json:"totalRecharge"`
}

type AppPayWalletRechargeCreateResp struct {
	ID         int64 `json:"id"`
	PayOrderID int64 `json:"payOrderId"`
}

type AppPayWalletRechargeResp struct {
	ID                     int64               `json:"id"`
	TotalPrice             int                 `json:"totalPrice"`
	PayPrice               int                 `json:"payPrice"`
	BonusPrice             int                 `json:"bonusPrice"`
	PayChannelCode         string              `json:"payChannelCode"`
	PayChannelName         string              `json:"payChannelName"`
	PayOrderID             int64               `json:"payOrderId"`
	PayOrderChannelOrderNo string              `json:"payOrderChannelOrderNo"`
	PayTime                *types.JsonDateTime `json:"payTime"`
	RefundStatus           int                 `json:"refundStatus"`
}

type AppPayWalletTransactionSummaryResp struct {
	TotalExpense int `json:"totalExpense"`
	TotalIncome  int `json:"totalIncome"`
}
