package promotion

import (
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	promotionContract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	memberModel "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppBargainHelpHandler struct {
	helpSvc *promotion.BargainHelpService
	userSvc *member.MemberUserService
}

func NewAppBargainHelpHandler(helpSvc *promotion.BargainHelpService, userSvc *member.MemberUserService) *AppBargainHelpHandler {
	return &AppBargainHelpHandler{
		helpSvc: helpSvc,
		userSvc: userSvc,
	}
}

// CreateBargainHelp 砍价助力
// Java: POST /create, returns ReducePrice
func (h *AppBargainHelpHandler) CreateBargainHelp(c *gin.Context) {
	var r promotion2.AppBargainHelpCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteError(c, 1001004001, "参数校验失败")
		return
	}
	help, err := h.helpSvc.CreateBargainHelp(c.Request.Context(), c.GetInt64(context.CtxUserIDKey), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, help.ReducePrice)
}

// GetBargainHelpList 获得砍价助力列表
// Java: GET /list
func (h *AppBargainHelpHandler) GetBargainHelpList(c *gin.Context) {
	recordId := utils.ParseInt64(c.Query("recordId"))
	if recordId == 0 {
		response.WriteSuccess(c, []promotionContract.AppBargainHelpRespVO{})
		return
	}

	list, err := h.helpSvc.GetBargainHelpList(c.Request.Context(), recordId)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Fetch User Info
	userIds := make([]int64, len(list))
	for i, item := range list {
		userIds[i] = item.UserID
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

	resList := make([]promotionContract.AppBargainHelpRespVO, len(list))
	for i, item := range list {
		vo := promotionContract.AppBargainHelpRespVO{
			UserID:      item.UserID,
			ReducePrice: item.ReducePrice,
			CreateTime:  types.ToJsonDateTime(item.CreateTime),
		}
		if u, ok := userMap[item.UserID]; ok {
			vo.Nickname = u.Nickname
			vo.Avatar = u.Avatar
		}
		resList[i] = vo
	}
	response.WriteSuccess(c, resList)
}
