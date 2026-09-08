package app

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

type DictDataResp struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type DeliveryExpressResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
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
	ID         int64              `json:"id"`
	Content    string             `json:"content"`
	CreateTime types.JsonDateTime `json:"createTime"`
}
