package promotion

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strconv"

	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	promotionContract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type AppCombinationRecordHandler struct {
	svc promotion.CombinationRecordService
}

func NewAppCombinationRecordHandler(svc promotion.CombinationRecordService) *AppCombinationRecordHandler {
	return &AppCombinationRecordHandler{svc: svc}
}

// GetCombinationRecordSummary 获得拼团记录的概要信息
func (h *AppCombinationRecordHandler) GetCombinationRecordSummary(c *gin.Context) {
	summary, err := h.svc.GetCombinationRecordSummary(c.Request.Context())
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, summary)
}

// GetHeadCombinationRecordList 获得团长发起的拼团记录
func (h *AppCombinationRecordHandler) GetHeadCombinationRecordList(c *gin.Context) {
	activityID, _ := strconv.ParseInt(c.Query("activityId"), 10, 64)
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	status, _ := strconv.Atoi(c.Query("status"))
	list, err := h.svc.GetHeadCombinationRecordList(c.Request.Context(), activityID, status, count)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Convert to RespVO
	vos := make([]*promotionContract.AppCombinationRecordRespVO, len(list))
	for i, item := range list {
		vos[i] = &promotionContract.AppCombinationRecordRespVO{
			ID:               item.ID,
			ActivityID:       item.ActivityID,
			Nickname:         item.Nickname,
			Avatar:           item.Avatar,
			ExpireTime:       types.ToJsonDateTimePtr(&item.ExpireTime),
			UserSize:         item.UserSize,
			UserCount:        item.UserCount,
			Status:           item.Status,
			OrderID:          item.OrderID,
			SpuName:          item.SpuName,
			PicUrl:           item.PicUrl,
			Count:            item.Count,
			CombinationPrice: item.CombinationPrice,
		}
	}
	response.WriteSuccess(c, vos)
}

// GetCombinationRecordPage 获得我的拼团记录分页
func (h *AppCombinationRecordHandler) GetCombinationRecordPage(c *gin.Context) {
	var req promotion2.AppCombinationRecordPageReq
	req.PageNo, _ = strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	req.PageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	req.Status, _ = strconv.Atoi(c.DefaultQuery("status", "0"))

	userId := c.GetInt64("userId") // Requires Auth Middleware

	list, err := h.svc.GetCombinationRecordPage(c.Request.Context(), userId, req)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, list)
}

// GetCombinationRecordDetail 获得拼团记录明细
func (h *AppCombinationRecordHandler) GetCombinationRecordDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil {
		response.WriteBizError(c, errors.NewBizError(400, "Invalid ID"))
		return
	}

	userId := c.GetInt64("userId")

	detail, err := h.svc.GetCombinationRecordDetail(c.Request.Context(), userId, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, detail)
}
