package member

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"strconv"

	system2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type AppSocialUserHandler struct {
	svc *system.SocialUserService
}

func NewAppSocialUserHandler(svc *system.SocialUserService) *AppSocialUserHandler {
	return &AppSocialUserHandler{svc: svc}
}

// Bind 绑定社交用户
// @Router /member/social-user/bind [post]
func (h *AppSocialUserHandler) Bind(c *gin.Context) {
	var r system2.AppSocialUserBindReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, err)
		return
	}

	userID := context.GetUserId(c)
	openid, err := h.svc.BindSocialUser(c, userID, consts.UserTypeMember, &system2.SocialUserBindReq{
		Type:  r.Type,
		Code:  r.Code,
		State: r.State,
	})
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, openid)
}

// Unbind 解绑社交用户
// @Router /member/social-user/unbind [delete]
func (h *AppSocialUserHandler) Unbind(c *gin.Context) {
	var r system2.AppSocialUserUnbindReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, err)
		return
	}

	userID := context.GetUserId(c)
	err := h.svc.UnbindSocialUser(c, userID, consts.UserTypeMember, r.Type, r.OpenID)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, true)
}

// Get 获取社交用户
// @Router /member/social-user/get [get]
func (h *AppSocialUserHandler) Get(c *gin.Context) {
	socialTypeStr := c.Query("type")
	socialType, _ := strconv.Atoi(socialTypeStr)
	userID := context.GetUserId(c)

	socialUsers, err := h.svc.GetSocialUserList(c, userID, consts.UserTypeMember)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	var target *system2.AppSocialUserResp
	for _, su := range socialUsers {
		if su.Type == socialType {
			target = &system2.AppSocialUserResp{
				Openid:   su.Openid,
				Nickname: su.Nickname,
				Avatar:   su.Avatar,
			}
			break
		}
	}

	response.WriteSuccess(c, target)
}

// GetWxaQrcode 获得微信小程序码
// @Router /member/social-user/wxa-qrcode [post]
func (h *AppSocialUserHandler) GetWxaQrcode(c *gin.Context) {
	path := c.Query("path")
	width, _ := strconv.Atoi(c.DefaultQuery("width", "430"))

	qrCode, err := h.svc.GetWxaQrcode(c, consts.UserTypeMember, path, width)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, qrCode)
}

// GetSubscribeTemplateList 获得微信小程序订阅模板列表
// @Router /member/social-user/get-subscribe-template-list [get]
func (h *AppSocialUserHandler) GetSubscribeTemplateList(c *gin.Context) {
	templates, err := h.svc.GetSubscribeTemplateList(c, consts.UserTypeMember)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, templates)
}
