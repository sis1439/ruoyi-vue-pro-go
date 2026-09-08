package promotion

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

// AppSeckillActivityDetailReq App 端 - 秒杀活动详情请求
type AppSeckillActivityDetailReq struct {
	ID int64 `form:"id" binding:"required"` // 活动 ID
}

// AppSeckillActivityPageReq App 端 - 秒杀活动分页请求 (对齐 Java: AppSeckillActivityPageReqVO)
type AppSeckillActivityPageReq struct {
	PageNo   int    `form:"pageNo" binding:"required,min=1"`
	PageSize int    `form:"pageSize" binding:"required,min=1,max=100"`
	ConfigID *int64 `form:"configId"` // 秒杀时段编号
}

// AppSeckillConfigResp App 端 - 秒杀时段响应 (对齐 Java: AppSeckillConfigRespVO)
type AppSeckillConfigResp struct {
	ID            int64    `json:"id"`
	StartTime     string   `json:"startTime"`
	EndTime       string   `json:"endTime"`
	SliderPicUrls []string `json:"sliderPicUrls"`
}

// AppSeckillActivityResp App 端 - 秒杀活动响应 (对齐 Java: AppSeckillActivityRespVO)
type AppSeckillActivityResp struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	SpuID        int64  `json:"spuId"`
	SpuName      string `json:"spuName"` // 商品名称
	PicURL       string `json:"picUrl"`
	MarketPrice  int    `json:"marketPrice"`  // 市场价 (分)
	SeckillPrice int    `json:"seckillPrice"` // 秒杀价 (分)
	Status       int    `json:"status"`       // 活动状态
	Stock        int    `json:"stock"`        // 库存
	TotalStock   int    `json:"totalStock"`   // 总库存
}

// AppSeckillActivityDetailResp App 端 - 秒杀活动详情响应 (对齐 Java: AppSeckillActivityDetailRespVO)
type AppSeckillActivityDetailResp struct {
	ID               int64               `json:"id"`
	Name             string              `json:"name"`
	Status           int                 `json:"status"` // 活动状态
	SpuID            int64               `json:"spuId"`
	StartTime        *types.JsonDateTime `json:"startTime"`
	EndTime          *types.JsonDateTime `json:"endTime"`
	SingleLimitCount int                 `json:"singleLimitCount"` // 单次限购
	TotalLimitCount  int                 `json:"totalLimitCount"`  // 总限购
	Stock            int                 `json:"stock"`            // 库存
	TotalStock       int                 `json:"totalStock"`       // 总库存
	// SPU 信息
	SpuName      string `json:"spuName"`      // 商品名称
	PicURL       string `json:"picUrl"`       // 商品主图
	MarketPrice  int    `json:"marketPrice"`  // 市场价 (分)
	SeckillPrice int    `json:"seckillPrice"` // 秒杀价 (分)
	// 秒杀商品
	Products []AppSeckillProductResp `json:"products"`
}

// AppSeckillProductResp App 端 - 秒杀商品响应
type AppSeckillProductResp struct {
	ID           int64 `json:"id"`
	ActivityID   int64 `json:"activityId"`
	SpuID        int64 `json:"spuId"`
	SkuID        int64 `json:"skuId"`
	SeckillPrice int   `json:"seckillPrice"`
	Stock        int   `json:"stock"`
}

// AppSeckillActivityNowResp App 端 - 当前秒杀活动响应 (对齐 Java: AppSeckillActivityNowRespVO)
type AppSeckillActivityNowResp struct {
	Config     *AppSeckillConfigResp    `json:"config"`     // 当前时段
	Activities []AppSeckillActivityResp `json:"activities"` // 秒杀活动列表
}
