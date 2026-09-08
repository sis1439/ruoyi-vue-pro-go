package product

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

// AppProductSpuPageReq 商品 SPU 分页 Request VO
type AppProductSpuPageReq struct {
	CategoryIDs []int64 `form:"categoryIds" collection_format:"csv"`
	IDs         []int64 `form:"ids" collection_format:"csv"`
	pagination.PageParam
	// 分类编号
	CategoryID *int64 `form:"categoryId"`
	// 关键字
	Keyword *string `form:"keyword"`
	// 排序字段
	SortField *string `form:"sortField"` // price, sales_count
	// 排序方式
	SortAsc *bool `form:"sortAsc"`
	// 推荐类型
	RecommendType *string `form:"recommendType"` // new, hot, best
}

// AppProductSpuResp 用户 APP - 商品 SPU Response VO (List) - 对齐Java版本AppProductSpuRespVO
type AppProductSpuResp struct {
	ID            int64    `json:"id"`            // 商品SPU编号
	Name          string   `json:"name"`          // 商品名称
	Introduction  string   `json:"introduction"`  // 商品简介 - 对齐Java版本
	CategoryID    int64    `json:"categoryId"`    // 分类编号 - 对齐Java版本
	PicURL        string   `json:"picUrl"`        // 商品封面图 - 对齐Java版本
	SliderPicURLs []string `json:"sliderPicUrls"` // 商品轮播图 - 对齐Java版本
	SpecType      bool     `json:"specType"`      // 规格类型 - 对齐Java版本
	Price         int      `json:"price"`         // 商品价格，单位：分 - 对齐Java版本
	MarketPrice   int      `json:"marketPrice"`   // 市场价，单位：分 - 对齐Java版本
	Stock         int      `json:"stock"`         // 库存 - 对齐Java版本
	SalesCount    int      `json:"salesCount"`    // 商品销量（实际销量+虚拟销量）- 对齐Java版本
	DeliveryTypes []int    `json:"deliveryTypes"` // 配送方式数组 - 对齐Java版本
}
