package product

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

// ProductFavoritePageReq (Admin)
type ProductFavoritePageReq struct {
	pagination.PageParam
	UserId int64 `form:"userId"` // 用户编号
	SpuId  int64 `form:"spuId"`  // 商品 SPU 编号
}

// AppFavoritePageReq (App)
type AppFavoritePageReq struct {
	pagination.PageParam
}

// AppFavoriteReq (App) - For Create, Delete, Exists
type AppFavoriteReq struct {
	SpuId int64 `json:"spuId" form:"spuId" binding:"required"` // 商品 SPU 编号
}

type AppFavoriteCreateReq struct {
	SpuId int64 `json:"spuId" binding:"required"` // 商品 SPU 编号
}

// ProductFavoriteResp 商品收藏响应 (Admin)
type ProductFavoriteResp struct {
	ID                 int64             `json:"id"`
	UserID             int64             `json:"userId"`
	SpuID              int64             `json:"spuId"`
	CreateTime         time.Time         `json:"createTime"`
	Name               string            `json:"name"`
	Keyword            string            `json:"keyword"`
	Introduction       string            `json:"introduction"`
	Description        string            `json:"description"`
	CategoryID         int64             `json:"categoryId"`
	BrandID            int64             `json:"brandId"`
	PicURL             string            `json:"picUrl"`
	SliderPicURLs      []string          `json:"sliderPicUrls"`
	Sort               int               `json:"sort"`
	Status             int               `json:"status"`
	SpecType           bool              `json:"specType"`
	Price              int               `json:"price"`
	MarketPrice        int               `json:"marketPrice"`
	CostPrice          int               `json:"costPrice"`
	Stock              int               `json:"stock"`
	DeliveryTypes      []int             `json:"deliveryTypes"`
	DeliveryTemplateID int64             `json:"deliveryTemplateId"`
	GiveIntegral       int               `json:"giveIntegral"`
	SubCommissionType  bool              `json:"subCommissionType"`
	SalesCount         int               `json:"salesCount"`
	VirtualSalesCount  int               `json:"virtualSalesCount"`
	BrowseCount        int               `json:"browseCount"`
	Skus               []*ProductSkuResp `json:"skus"`
}

// AppFavoriteResp App 商品收藏响应
type AppFavoriteResp struct {
	ID         int64              `json:"id"`
	SpuID      int64              `json:"spuId"`
	CreateTime types.JsonDateTime `json:"createTime"`
	SpuName    string             `json:"spuName"`
	PicURL     string             `json:"picUrl"`
	Price      int64              `json:"price"`
}
