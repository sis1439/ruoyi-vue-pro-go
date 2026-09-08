package member

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

// AppAddressCreateReq 创建收件地址请求
type AppAddressCreateReq struct {
	Name          string `json:"name" binding:"required"`
	Mobile        string `json:"mobile" binding:"required,len=11"`
	AreaID        int64  `json:"areaId" binding:"required"`
	DetailAddress string `json:"detailAddress" binding:"required"`
	DefaultStatus *bool  `json:"defaultStatus" binding:"required"`
}

// AppAddressUpdateReq 更新收件地址请求
type AppAddressUpdateReq struct {
	ID            int64  `json:"id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Mobile        string `json:"mobile" binding:"required,len=11"`
	AreaID        int64  `json:"areaId" binding:"required"`
	DetailAddress string `json:"detailAddress" binding:"required"`
	DefaultStatus *bool  `json:"defaultStatus" binding:"required"`
}

// AppAddressResp 收件地址响应
type AppAddressResp struct {
	AreaName      string             `json:"areaName"`
	ID            int64              `json:"id"`
	Name          string             `json:"name"`
	Mobile        string             `json:"mobile"`
	AreaID        int64              `json:"areaId"`
	DetailAddress string             `json:"detailAddress"`
	DefaultStatus bool               `json:"defaultStatus"`
	CreateTime    types.JsonDateTime `json:"createTime"`
}
