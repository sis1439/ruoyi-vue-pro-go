package product

import (
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

// ProductBrowseHistoryPageReq (Admin)
type ProductBrowseHistoryPageReq struct {
	pagination.PageParam
	UserId      int64       `form:"userId"`      // 用户编号
	UserDeleted *bool       `form:"userDeleted"` // 用户是否删除
	SpuId       int64       `form:"spuId"`       // 商品 SPU 编号
	CreateTime  []time.Time `form:"createTime"`  // 创建时间
}

// AppProductBrowseHistoryPageReq (App)
type AppProductBrowseHistoryPageReq struct {
	CreateTime []string `form:"createTime[]"`
	pagination.PageParam
}

// AppProductBrowseHistoryDeleteReq (App)
type AppProductBrowseHistoryDeleteReq struct {
	SpuIds []int64 `json:"spuIds" binding:"required"` // 商品 SPU 编号数组
}

type ProductBrowseHistoryResp struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"userId"`
	SpuID      int64  `json:"spuId"`
	SpuName    string `json:"spuName"`
	PicURL     string `json:"picUrl"`
	Price      int64  `json:"price"`
	SalesCount int    `json:"salesCount"`
	Stock      int    `json:"stock"`
}

type AppProductBrowseHistoryResp struct {
	ID         int64  `json:"id"`
	SpuID      int64  `json:"spuId"`
	SpuName    string `json:"spuName"`
	PicURL     string `json:"picUrl"`
	Price      int64  `json:"price"`
	SalesCount int    `json:"salesCount"`
	Stock      int    `json:"stock"`
}
