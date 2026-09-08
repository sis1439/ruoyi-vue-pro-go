package system

import (
	"strconv"

	system2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *system.AuthService
}

func NewAuthHandler(svc *system.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login 登录接口
// @Router /system/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req system2.AuthLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, resp)
}

// GetPermissionInfo 获取权限信息
// @Router /system/auth/get-permission-info [get]
func (h *AuthHandler) GetPermissionInfo(c *gin.Context) {
	resp, err := h.svc.GetPermissionInfo(c)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, resp)
}

// Logout 登出
// @Router /system/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从 Header 获取 token
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
	}

	err := h.svc.Logout(c.Request.Context(), token)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// RefreshToken 刷新令牌
// @Router /system/auth/refresh-token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken := c.Query("refreshToken")
	if refreshToken == "" {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	resp, err := h.svc.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, resp)
}

// SmsLogin 短信登录
// @Router /system/auth/sms-login [post]
func (h *AuthHandler) SmsLogin(c *gin.Context) {
	var r system2.AuthSmsLoginReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	resp, err := h.svc.SmsLogin(c.Request.Context(), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, resp)
}

// SendSmsCode 发送短信验证码
// @Router /system/auth/send-sms-code [post]
func (h *AuthHandler) SendSmsCode(c *gin.Context) {
	var r system2.AuthSmsSendReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	err := h.svc.SendSmsCode(c.Request.Context(), &r, c.ClientIP())
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// Register 注册
// @Router /system/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var r system2.AuthRegisterReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, resp)
}

// ResetPassword 重置密码
// @Router /system/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var r system2.AuthResetPasswordReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	err := h.svc.ResetPassword(c.Request.Context(), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// SocialAuthRedirect 社交授权跳转
// @Router /system/auth/social-auth-redirect [get]
func (h *AuthHandler) SocialAuthRedirect(c *gin.Context) {
	socialType := c.Query("type")
	redirectUri := c.Query("redirectUri")

	if socialType == "" {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	// 转换类型
	typeInt, err := strconv.Atoi(socialType)
	if err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	url, err := h.svc.SocialAuthRedirect(c.Request.Context(), typeInt, redirectUri)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, url)
}

// SocialLogin 社交登录
// @Router /system/auth/social-login [post]
func (h *AuthHandler) SocialLogin(c *gin.Context) {
	var r system2.AuthSocialLoginReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	resp, err := h.svc.SocialLogin(c.Request.Context(), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, resp)
}

func (h *AuthHandler) InspectTenant(c *gin.Context) {
	target, err := strconv.ParseInt(c.Query("targetTenantId"), 10, 64)
	if err != nil || target <= 0 {
		response.WriteError(c, 400, "targetTenantId必须为正数")
		return
	}
	result, err := h.svc.InspectTenant(c.Request.Context(), target)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, result)
}
