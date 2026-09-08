package promotion

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	promotionSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
)

// AppActivityHandler 用户 App - 营销活动 Handler
type AppActivityHandler struct {
	combinationActivitySvc promotionSvc.CombinationActivityService
	seckillActivitySvc     *promotionSvc.SeckillActivityService
	bargainActivitySvc     *promotionSvc.BargainActivityService
}

func NewAppActivityHandler(
	combinationActivitySvc promotionSvc.CombinationActivityService,
	seckillActivitySvc *promotionSvc.SeckillActivityService,
	bargainActivitySvc *promotionSvc.BargainActivityService,
) *AppActivityHandler {
	return &AppActivityHandler{
		combinationActivitySvc: combinationActivitySvc,
		seckillActivitySvc:     seckillActivitySvc,
		bargainActivitySvc:     bargainActivitySvc,
	}
}

// GetActivityListBySpuId 获得单个商品，进行中的拼团、秒杀、砍价活动信息
// 对齐 Java: AppActivityController.getActivityListBySpuId
func (h *AppActivityHandler) GetActivityListBySpuId(c *gin.Context) {
	spuIdStr := c.Query("spuId")
	if spuIdStr == "" {
		response.WriteError(c, 400, "参数错误")
		return
	}
	spuId, _ := strconv.ParseInt(spuIdStr, 10, 64)

	var activityVOList = make([]promotion.AppActivityRespVO, 0)

	// 1. 拼团活动
	combinationActivity, err := h.combinationActivitySvc.GetMatchCombinationActivityBySpuId(c.Request.Context(), spuId)
	if err == nil && combinationActivity != nil {
		activityVOList = append(activityVOList, promotion.AppActivityRespVO{
			Id:        combinationActivity.ID,
			Type:      consts.PromotionTypeCombinationActivity,
			Name:      combinationActivity.Name,
			SpuId:     combinationActivity.SpuID,
			StartTime: types.ToJsonDateTimePtr(&combinationActivity.StartTime),
			EndTime:   types.ToJsonDateTimePtr(&combinationActivity.EndTime),
		})
	}

	// 2. 秒杀活动
	seckillActivity, err := h.seckillActivitySvc.GetMatchSeckillActivityBySpuId(c.Request.Context(), spuId)
	if err == nil && seckillActivity != nil {
		activityVOList = append(activityVOList, promotion.AppActivityRespVO{
			Id:        seckillActivity.ID,
			Type:      consts.PromotionTypeSeckillActivity,
			Name:      seckillActivity.Name,
			SpuId:     seckillActivity.SpuID,
			StartTime: types.ToJsonDateTimePtr(&seckillActivity.StartTime),
			EndTime:   types.ToJsonDateTimePtr(&seckillActivity.EndTime),
		})
	}

	// 3. 砍价活动
	bargainActivity, err := h.bargainActivitySvc.GetMatchBargainActivityBySpuId(c.Request.Context(), spuId)
	if err == nil && bargainActivity != nil {
		activityVOList = append(activityVOList, promotion.AppActivityRespVO{
			Id:        bargainActivity.ID,
			Type:      consts.PromotionTypeBargainActivity,
			Name:      bargainActivity.Name,
			SpuId:     bargainActivity.SpuID,
			StartTime: types.ToJsonDateTimePtr(&bargainActivity.StartTime),
			EndTime:   types.ToJsonDateTimePtr(&bargainActivity.EndTime),
		})
	}

	response.WriteSuccess(c, activityVOList)
}
