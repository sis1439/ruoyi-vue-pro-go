package app

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

type DictDataResp struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type DeliveryExpressResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
	Logo string `json:"logo"`
}
type DeliveryPickUpStoreResp struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Logo          string   `json:"logo"`
	Phone         string   `json:"phone"`
	AreaID        int      `json:"areaId"`
	AreaName      string   `json:"areaName"`
	DetailAddress string   `json:"detailAddress"`
	OpeningTime   string   `json:"openingTime"`
	ClosingTime   string   `json:"closingTime"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	Distance      *float64 `json:"distance"`
}
type AfterSaleLogResp struct {
	ID           int64              `json:"id"`
	AfterSaleID  int64              `json:"afterSaleId"`
	BeforeStatus int                `json:"beforeStatus"`
	AfterStatus  int                `json:"afterStatus"`
	OperateType  int                `json:"operateType"`
	Content      string             `json:"content"`
	CreateTime   types.JsonDateTime `json:"createTime"`
}
