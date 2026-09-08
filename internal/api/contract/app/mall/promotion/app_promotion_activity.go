package promotion

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

// AppActivityRespVO 用户 App - 营销活动 Response VO
type AppActivityRespVO struct {
	Id        int64               `json:"id"`        // 活动编号
	Type      int                 `json:"type"`      // 活动类型 (PromotionTypeEnum)
	Name      string              `json:"name"`      // 活动名称
	SpuId     int64               `json:"spuId"`     // spu 编号
	StartTime *types.JsonDateTime `json:"startTime"` // 活动开始时间
	EndTime   *types.JsonDateTime `json:"endTime"`   // 活动结束时间
}
