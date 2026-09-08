package promotion

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

// AppCombinationActivityRespVO 拼团活动 App Response
type AppCombinationActivityRespVO struct {
	ID               int64              `json:"id"`
	Name             string             `json:"name"`
	SpuID            int64              `json:"spuId"`
	SkuId            int64              `json:"skuId"`
	PicUrl           string             `json:"picUrl"`
	CombinationPrice int                `json:"combinationPrice"`
	MarketPrice      int                `json:"marketPrice"`
	UserSize         int                `json:"userSize"`
	Stock            int                `json:"stock"`
	StartTime        types.JsonDateTime `json:"startTime"`
	EndTime          types.JsonDateTime `json:"endTime"`
}

// AppCombinationRecordRespVO 拼团记录 App Response
type AppCombinationRecordRespVO struct {
	ID               int64               `json:"id"`
	UserID           int64               `json:"userId"`
	ActivityID       int64               `json:"activityId"`
	SpuID            int64               `json:"spuId"`
	SkuID            int64               `json:"skuId"`
	PicUrl           string              `json:"picUrl"`
	SpuName          string              `json:"spuName"`
	Nickname         string              `json:"nickname"`
	Avatar           string              `json:"avatar"`
	Status           int                 `json:"status"`
	UserSize         int                 `json:"userSize"`
	UserCount        int                 `json:"userCount"`
	CombinationPrice int                 `json:"combinationPrice"`
	StartTime        types.JsonDateTime  `json:"startTime"`
	EndTime          types.JsonDateTime  `json:"endTime"`
	ExpireTime       *types.JsonDateTime `json:"expireTime"`
	OrderID          int64               `json:"orderId"`
	Count            int                 `json:"count"`
}
