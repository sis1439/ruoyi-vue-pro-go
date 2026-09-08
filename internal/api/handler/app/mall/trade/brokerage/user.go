package brokerage

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	tradeDto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/trade"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/trade/brokerage"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade/brokerage"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type AppBrokerageUserHandler struct {
	userSvc     *brokerage.BrokerageUserService
	recordSvc   *brokerage.BrokerageRecordService
	withdrawSvc *brokerage.BrokerageWithdrawService
	memberSvc   *member.MemberUserService
}

func NewAppBrokerageUserHandler(userSvc *brokerage.BrokerageUserService, recordSvc *brokerage.BrokerageRecordService, withdrawSvc *brokerage.BrokerageWithdrawService, memberSvc *member.MemberUserService) *AppBrokerageUserHandler {
	return &AppBrokerageUserHandler{
		userSvc:     userSvc,
		recordSvc:   recordSvc,
		withdrawSvc: withdrawSvc,
		memberSvc:   memberSvc,
	}
}

// GetBrokerageUser 获得个人分销信息
func (h *AppBrokerageUserHandler) GetBrokerageUser(c *gin.Context) {
	userId := c.GetInt64("userId")
	user, err := h.userSvc.GetOrCreateBrokerageUser(c, userId)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	// 如果用户为 nil（分销功能未启用），返回默认值
	if user == nil {
		respVO := &tradeDto.AppBrokerageUserRespVO{
			BrokerageEnabled: false,
			BrokeragePrice:   0,
			FrozenPrice:      0,
		}
		response.WriteSuccess(c, respVO)
		return
	}

	respVO := &tradeDto.AppBrokerageUserRespVO{
		BrokerageEnabled: bool(user.BrokerageEnabled),
		BrokeragePrice:   user.BrokeragePrice,
		FrozenPrice:      user.FrozenPrice,
	}
	response.WriteSuccess(c, respVO)
}

// BindBrokerageUser 绑定推广员
func (h *AppBrokerageUserHandler) BindBrokerageUser(c *gin.Context) {
	var r tradeDto.AppBrokerageUserBindReqVO
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}
	userId := c.GetInt64("userId")
	success, err := h.userSvc.BindBrokerageUser(c, userId, int64(r.BindUserID))
	if err != nil {
		response.WriteError(c, 500, err.Error()) // Or 400
		return
	}
	response.WriteSuccess(c, success)
}

// GetBrokerageUserSummary 获得个人分销统计
func (h *AppBrokerageUserHandler) GetBrokerageUserSummary(c *gin.Context) {
	userId := c.GetInt64("userId")
	user, err := h.userSvc.GetBrokerageUser(c, userId)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	if user == nil {
		// Should create? Java uses getBrokerageUser and assuming it exists?
		// Actually Java calls getOrCreate in /get, but here it calls get.
		// If nil, use default 0
		user = &model.BrokerageUser{} // Empty
	}

	// 1. Yesterday Price (Call Record Service)
	yesterday := time.Now().AddDate(0, 0, -1)
	beginOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, time.Local)

	// BizType: ORDER=1 (Assume). Status: SETTLEMENT=1 (Assume).
	yesterdayPrice, err := h.recordSvc.GetSummaryPriceByUserId(c, userId, tradeModel.BrokerageRecordBizTypeOrder, tradeModel.BrokerageRecordStatusSettlement, beginOfDay, endOfDay)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	// 2. Withdraw Price
	// Status: AUDIT_SUCCESS(10), WITHDRAW_SUCCESS(11)
	summaries, err := h.withdrawSvc.GetWithdrawSummaryListByUserId(c, []int64{userId}, []int{tradeModel.BrokerageWithdrawStatusAuditSuccess, tradeModel.BrokerageWithdrawStatusWithdrawSuccess})
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	withdrawPrice := 0
	if len(summaries) > 0 {
		withdrawPrice = summaries[0].Price
	}

	// 3. Count
	firstCount, _ := h.userSvc.GetBrokerageUserCountByBindUserId(c, userId, tradeModel.BrokerageUserLevelOne)
	secondCount, _ := h.userSvc.GetBrokerageUserCountByBindUserId(c, userId, tradeModel.BrokerageUserLevelTwo)

	respVO := &tradeDto.AppBrokerageUserMySummaryRespVO{
		YesterdayPrice:           yesterdayPrice,
		WithdrawPrice:            withdrawPrice,
		FirstBrokerageUserCount:  int(firstCount),
		SecondBrokerageUserCount: int(secondCount),
		BrokeragePrice:           user.BrokeragePrice,
		FrozenPrice:              user.FrozenPrice,
	}
	response.WriteSuccess(c, respVO)
}

// GetBrokerageUserChildSummaryPage 获得下级分销统计分页
func (h *AppBrokerageUserHandler) GetBrokerageUserChildSummaryPage(c *gin.Context) {
	var r tradeDto.AppBrokerageUserChildSummaryPageReqVO
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}
	userId := c.GetInt64("userId")
	pageResult, err := h.userSvc.GetBrokerageUserChildSummaryPage(c, &r, userId)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, pageResult)
}

// GetBrokerageUserRankPageByUserCount 获得分销用户排行分页（基于用户量）
func (h *AppBrokerageUserHandler) GetBrokerageUserRankPageByUserCount(c *gin.Context) {
	var r tradeDto.AppBrokerageUserRankPageReqVO
	r.Times = types.QueryTimeRange(c.Request.URL.Query(), "times")
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	pageResult, err := h.userSvc.GetBrokerageUserRankPageByUserCount(c, &r)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	// 收集用户 ID
	userIds := make([]int64, len(pageResult.List))
	for i, u := range pageResult.List {
		userIds[i] = u.ID
	}

	// 批量获取用户信息
	userMap, _ := h.memberSvc.GetUserMap(c, userIds)

	// 转换为 VO
	list := make([]tradeDto.AppBrokerageUserRankByUserCountRespVO, len(pageResult.List))
	for i, u := range pageResult.List {
		vo := tradeDto.AppBrokerageUserRankByUserCountRespVO{
			ID:                 u.ID,
			BrokerageUserCount: u.BrokerageUserCount,
		}
		if info, ok := userMap[u.ID]; ok {
			vo.Nickname = info.Nickname
			vo.Avatar = info.Avatar
		}
		list[i] = vo
	}

	response.WriteSuccess(c, &pagination.PageResult[tradeDto.AppBrokerageUserRankByUserCountRespVO]{
		List:  list,
		Total: pageResult.Total,
	})
}

// GetBrokerageUserRankPageByPrice 获得分销用户排行分页（基于佣金）
func (h *AppBrokerageUserHandler) GetBrokerageUserRankPageByPrice(c *gin.Context) {
	var r trade.AppBrokerageUserRankPageReq
	r.Times = types.QueryTimeRange(c.Request.URL.Query(), "times")
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	pageResult, err := h.recordSvc.GetBrokerageUserRankPageByPrice(c, &r)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	// 收集用户 ID 并获取用户信息
	userIds := make([]int64, len(pageResult.List))
	for i, u := range pageResult.List {
		userIds[i] = u.ID
	}
	userMap, _ := h.memberSvc.GetUserMap(c, userIds)

	// 填充用户昵称/头像
	for _, vo := range pageResult.List {
		if info, ok := userMap[vo.ID]; ok {
			vo.Nickname = info.Nickname
			vo.Avatar = info.Avatar
		}
	}

	response.WriteSuccess(c, pageResult)
}

// GetRankByPrice 获得分销用户排行（基于佣金）
func (h *AppBrokerageUserHandler) GetRankByPrice(c *gin.Context) {
	// 解析时间参数
	timesStr := c.QueryArray("times[]")
	var times []time.Time
	for _, t := range timesStr {
		parsed := parseTime(t)
		if !parsed.IsZero() {
			times = append(times, parsed)
		}
	}

	userId := c.GetInt64("userId")
	rank, err := h.recordSvc.GetUserRankByPrice(c, userId, times)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	response.WriteSuccess(c, rank)
}

// parseTime 辅助函数解析时间字符串
func parseTime(t string) time.Time {
	parsed, _ := time.Parse("2006-01-02 15:04:05", t)
	return parsed
}
