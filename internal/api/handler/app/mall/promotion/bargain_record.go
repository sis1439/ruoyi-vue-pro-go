package promotion

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	adminProduct "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	promotionContract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	memberModel "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	promotionModel "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppBargainRecordHandler struct {
	recordSvc   *promotion.BargainRecordService
	activitySvc *promotion.BargainActivityService
	userSvc     *member.MemberUserService
	spuSvc      *product.ProductSpuService
	orderSvc    *trade.TradeOrderQueryService
	helpSvc     *promotion.BargainHelpService
}

func NewAppBargainRecordHandler(recordSvc *promotion.BargainRecordService, activitySvc *promotion.BargainActivityService, userSvc *member.MemberUserService, spuSvc *product.ProductSpuService, orderSvc *trade.TradeOrderQueryService, helpSvc *promotion.BargainHelpService) *AppBargainRecordHandler {
	return &AppBargainRecordHandler{
		recordSvc:   recordSvc,
		activitySvc: activitySvc,
		userSvc:     userSvc,
		spuSvc:      spuSvc,
		orderSvc:    orderSvc,
		helpSvc:     helpSvc,
	}
}

// GetBargainRecordSummary 获得砍价记录的概要信息
// Java: GET /get-summary
func (h *AppBargainRecordHandler) GetBargainRecordSummary(c *gin.Context) {
	status := 1 // SUCCESS
	count, _ := h.recordSvc.GetBargainRecordUserCount(c.Request.Context(), 0, status)
	if count == 0 {
		response.WriteSuccess(c, promotionContract.AppBargainRecordSummaryRespVO{SuccessUserCount: 0, SuccessList: []promotionContract.AppBargainRecordSummaryRecordVO{}})
		return
	}

	records, _ := h.recordSvc.GetBargainRecordList(c.Request.Context(), status, 7)

	// Fetch Users
	userIds := make([]int64, len(records))
	activityIds := make([]int64, len(records))
	for i, r := range records {
		userIds[i] = r.UserID
		activityIds[i] = r.ActivityID
	}
	userMap := make(map[int64]*memberModel.MemberUser)
	if len(userIds) > 0 {
		um, err := h.userSvc.GetUserMap(c.Request.Context(), userIds)
		if err == nil {
			for k, v := range um {
				userMap[k] = v
			}
		}
	}

	// Fetch Activities for names
	activityMap := make(map[int64]*promotionModel.PromotionBargainActivity)
	if len(activityIds) > 0 {
		activities, err := h.activitySvc.GetBargainActivityList(c.Request.Context(), activityIds)
		if err == nil {
			for _, a := range activities {
				activityMap[a.ID] = a
			}
		}
	}

	successList := make([]promotionContract.AppBargainRecordSummaryRecordVO, len(records))
	for i, r := range records {
		item := promotionContract.AppBargainRecordSummaryRecordVO{}
		if u, ok := userMap[r.UserID]; ok {
			item.Nickname = u.Nickname
			item.Avatar = u.Avatar
		}
		if a, ok := activityMap[r.ActivityID]; ok {
			item.ActivityName = a.Name
		}
		successList[i] = item
	}

	response.WriteSuccess(c, promotionContract.AppBargainRecordSummaryRespVO{
		SuccessUserCount: int(count),
		SuccessList:      successList,
	})
}

// GetBargainRecordDetail 获得砍价记录详情
// Java: GET /get-detail
func (h *AppBargainRecordHandler) GetBargainRecordDetail(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	activityId := utils.ParseInt64(c.Query("activityId"))
	userId := c.GetInt64(pkgContext.CtxUserIDKey)

	if id == 0 && activityId == 0 {
		response.WriteError(c, 1001004001, "砍价记录编号和活动编号不能同时为空")
		return
	}

	var record *promotionModel.PromotionBargainRecord
	var err error

	if id > 0 {
		record, err = h.recordSvc.GetBargainRecord(c.Request.Context(), id)
	} else if activityId > 0 && userId > 0 {
		record, err = h.recordSvc.GetLastBargainRecord(c.Request.Context(), userId, activityId)
	}

	if err == nil && record != nil {
		activityId = record.ActivityID
	}

	// Fetch Help Action Logic
	var helpAction *int
	if userId > 0 {
		action := h.getHelpAction(c.Request.Context(), userId, record, activityId)
		if action != 0 {
			helpAction = &action
		}
	}

	res := promotionContract.AppBargainRecordDetailRespVO{
		ActivityID: activityId,
		HelpAction: helpAction,
	}

	if record != nil {
		res.ID = record.ID
		res.UserID = record.UserID
		res.SpuID = record.SpuID
		res.SkuID = record.SkuID
		res.BargainFirstPrice = record.BargainFirstPrice
		res.BargainPrice = record.BargainPrice
		res.Status = record.Status
		res.EndTime = types.ToJsonDateTime(record.EndTime)
		// Order Info
		if record.OrderID > 0 {
			res.OrderID = &record.OrderID
			order, err := h.orderSvc.GetOrder(c, record.OrderID)
			if err != nil {
				response.WriteBizError(c, err)
				return
			}
			res.PayOrderID = order.PayOrderID
			res.PayStatus = bool(order.PayStatus)
		}
	}

	response.WriteSuccess(c, res)
}

func (h *AppBargainRecordHandler) getHelpAction(ctx context.Context, userId int64, record *promotionModel.PromotionBargainRecord, activityId int64) int {
	if activityId == 0 {
		return 0
	}
	if record != nil && record.UserID == userId {
		return 0 // Own record
	}
	// 1. Check if already helped
	if record != nil {
		help, _ := h.helpSvc.GetBargainHelp(ctx, record.ID, userId)
		if help != nil {
			return 1 // SUCCESS (Already helped)
		}
	}
	// 2. Check limit
	act, _ := h.activitySvc.GetBargainActivity(ctx, activityId)
	if act != nil {
		count, _ := h.helpSvc.GetBargainHelpCountByActivity(ctx, activityId, userId)
		if count >= int64(act.BargainCount) {
			return 2 // FULL
		}
	}
	return 3 // NONE (Can help)
}

// GetBargainRecordPage 获得砍价记录分页
// Java: GET /page
func (h *AppBargainRecordHandler) GetBargainRecordPage(c *gin.Context) {
	var p pagination.PageParam
	if err := c.ShouldBindQuery(&p); err != nil {
		response.WriteError(c, 1001004001, "参数校验失败")
		return
	}

	userId := c.GetInt64(pkgContext.CtxUserIDKey)
	page, err := h.recordSvc.GetBargainRecordPage(c.Request.Context(), userId, &p)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	if page.Total == 0 {
		response.WriteSuccess(c, pagination.PageResult[promotionContract.AppBargainRecordRespVO]{List: []promotionContract.AppBargainRecordRespVO{}, Total: 0})
		return
	}

	// Fetch Activity names and PicUrls (from SPU)
	activityIds := make([]int64, len(page.List))
	spuIds := make([]int64, len(page.List))
	for i, r := range page.List {
		activityIds[i] = r.ActivityID
		spuIds[i] = r.SpuID
	}

	activityMap := make(map[int64]*promotionModel.PromotionBargainActivity)
	activities, _ := h.activitySvc.GetBargainActivityList(c.Request.Context(), activityIds)
	for _, a := range activities {
		activityMap[a.ID] = a
	}

	spuMap := make(map[int64]*adminProduct.ProductSpuResp)
	spus, _ := h.spuSvc.GetSpuList(c.Request.Context(), spuIds)
	for _, s := range spus {
		spuMap[s.ID] = s
	}

	result := make([]promotionContract.AppBargainRecordRespVO, len(page.List))
	for i, r := range page.List {
		item := promotionContract.AppBargainRecordRespVO{
			ID:           r.ID,
			SpuID:        r.SpuID,
			SkuID:        r.SkuID,
			ActivityID:   r.ActivityID,
			Status:       r.Status,
			BargainPrice: r.BargainPrice,
			EndTime:      types.ToJsonDateTime(r.EndTime),
		}
		if r.OrderID > 0 {
			item.OrderID = &r.OrderID
			order, err := h.orderSvc.GetOrder(c, r.OrderID)
			if err != nil {
				response.WriteBizError(c, err)
				return
			}
			item.PayOrderID = order.PayOrderID
			item.PayStatus = bool(order.PayStatus)
		}
		if a, ok := activityMap[r.ActivityID]; ok {
			item.ActivityName = a.Name
		}
		if s, ok := spuMap[r.SpuID]; ok {
			item.PicUrl = s.PicURL
		}
		result[i] = item
	}

	response.WriteSuccess(c, pagination.PageResult[promotionContract.AppBargainRecordRespVO]{List: result, Total: page.Total})
}

// CreateBargainRecord 创建砍价记录
// Java: POST /create
func (h *AppBargainRecordHandler) CreateBargainRecord(c *gin.Context) {
	var r promotion2.AppBargainRecordCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteError(c, 1001004001, "参数校验失败")
		return
	}
	id, err := h.recordSvc.CreateBargainRecord(c.Request.Context(), c.GetInt64(pkgContext.CtxUserIDKey), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, id)
}
