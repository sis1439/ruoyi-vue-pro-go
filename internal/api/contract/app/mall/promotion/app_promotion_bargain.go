package promotion

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

// AppBargainActivityRespVO 砍价活动 Response (App)
type AppBargainActivityRespVO struct {
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	StartTime       types.JsonDateTime `json:"startTime"`
	EndTime         types.JsonDateTime `json:"endTime"`
	SpuID           int64              `json:"spuId"`
	SkuID           int64              `json:"skuId"`
	Stock           int                `json:"stock"`
	BargainMinPrice int                `json:"bargainMinPrice"`
	PicUrl          string             `json:"picUrl"`
	MarketPrice     int                `json:"marketPrice"`
}

// AppBargainActivityDetailRespVO 砍价活动详情 Response (App)
type AppBargainActivityDetailRespVO struct {
	Price       int    `json:"price"`
	Description string `json:"description"`
	AppBargainActivityRespVO
	BargainFirstPrice int    `json:"bargainFirstPrice"`
	HelpMaxCount      int    `json:"helpMaxCount"`
	BargainCount      int    `json:"bargainCount"`
	TotalLimitCount   int    `json:"totalLimitCount"`
	RandomMinPrice    int    `json:"randomMinPrice"`
	RandomMaxPrice    int    `json:"randomMaxPrice"`
	SuccessUserCount  int    `json:"successUserCount"`
	Remark            string `json:"remark"`
}

// ========== Record & Help DTOs ==========

// AppBargainRecordRespVO 砍价记录 Response
type AppBargainRecordRespVO struct {
	PayOrderID   *int64             `json:"payOrderId"`
	PayStatus    bool               `json:"payStatus"`
	ID           int64              `json:"id"`
	SpuID        int64              `json:"spuId"`
	SkuID        int64              `json:"skuId"`
	ActivityID   int64              `json:"activityId"`
	Status       int                `json:"status"`
	BargainPrice int                `json:"bargainPrice"`
	EndTime      types.JsonDateTime `json:"endTime"`
	OrderID      *int64             `json:"orderId"`
	ActivityName string             `json:"activityName"`
	PicUrl       string             `json:"picUrl"`
}

// AppBargainRecordDetailRespVO 砍价记录详情 Response
type AppBargainRecordDetailRespVO struct {
	PayOrderID        *int64             `json:"payOrderId"`
	PayStatus         bool               `json:"payStatus"`
	ID                int64              `json:"id"`
	UserID            int64              `json:"userId"`
	SpuID             int64              `json:"spuId"`
	SkuID             int64              `json:"skuId"`
	BargainFirstPrice int                `json:"bargainFirstPrice"`
	BargainPrice      int                `json:"bargainPrice"`
	Status            int                `json:"status"`
	EndTime           types.JsonDateTime `json:"endTime"`
	OrderID           *int64             `json:"orderId"`
	ActivityID        int64              `json:"activityId"`
	HelpAction        *int               `json:"helpAction"`
}

// AppBargainRecordSummaryRespVO 砍价记录概要 Response
type AppBargainRecordSummaryRespVO struct {
	SuccessUserCount int                               `json:"successUserCount"`
	SuccessList      []AppBargainRecordSummaryRecordVO `json:"successList"`
}

type AppBargainRecordSummaryRecordVO struct {
	Nickname     string `json:"nickname"`
	Avatar       string `json:"avatar"`
	ActivityName string `json:"activityName"`
}

// AppBargainHelpRespVO 砍价助力 Response
type AppBargainHelpRespVO struct {
	UserID      int64              `json:"userId"`
	Nickname    string             `json:"nickname"`
	Avatar      string             `json:"avatar"`
	ReducePrice int                `json:"reducePrice"`
	CreateTime  types.JsonDateTime `json:"createTime"`
}
