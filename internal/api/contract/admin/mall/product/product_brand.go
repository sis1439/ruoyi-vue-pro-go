package product

import "time"

// ProductBrandCreateReq 创建品牌 Request
type ProductBrandCreateReq struct {
	Name        string `json:"name" binding:"required"`
	PicURL      string `json:"picUrl" binding:"required"`
	Sort        *int   `json:"sort" binding:"required,min=0"`
	Description string `json:"description"`
	Status      int    `json:"status" binding:"oneof=0 1"` // 0: 开启, 1: 关闭
}

// ProductBrandUpdateReq 更新品牌 Request
type ProductBrandUpdateReq struct {
	ID          int64  `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	PicURL      string `json:"picUrl" binding:"required"`
	Sort        *int   `json:"sort" binding:"required,min=0"`
	Description string `json:"description"`
	Status      int    `json:"status" binding:"oneof=0 1"`
}

// ProductBrandPageReq 分页查询品牌 Request
type ProductBrandPageReq struct {
	PageNo     int      `form:"pageNo" binding:"required,min=1"`
	PageSize   int      `form:"pageSize" binding:"required,min=1,max=100"`
	Name       string   `form:"name"`
	Status     *int     `form:"status"`
	CreateTime []string `form:"createTime[]"` // 时间范围查询
}

// ProductBrandListReq 列表查询品牌 Request
type ProductBrandListReq struct {
	Name string `form:"name"`
}

// ProductBrandResp 品牌信息 Response
type ProductBrandResp struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	PicURL      string    `json:"picUrl"`
	Sort        int       `json:"sort"`
	Description string    `json:"description"`
	Status      int       `json:"status"`
	CreateTime  time.Time `json:"createTime"`
}
