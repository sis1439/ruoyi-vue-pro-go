package context

import (
	"context"

	"github.com/gin-gonic/gin"
)

const (
	CtxUserIDKey     = "userID"
	CtxTenantIDKey   = "trustedTenantID"
	CtxLoginUserKey  = "loginUser"
	CtxGinContextKey = "GinContext" // 用于在 context.Context 中传递 gin.Context
)

// LoginUser 登录用户信息，与 Java 的 LoginUser 对齐
type LoginUser struct {
	UserID   int64  `json:"userId"`
	UserType int    `json:"userType"` // 1: Member, 2: Admin
	TenantID int64  `json:"tenantId"`
	DeptID   *int64 `json:"deptId"` // 部门ID (用于数据权限)
	Nickname string `json:"nickname"`
}

func GetLoginUserID(c *gin.Context) int64 {
	v, exists := c.Get(CtxUserIDKey)
	if !exists {
		return 0
	}
	if id, ok := v.(int64); ok {
		return id
	}
	return 0
}

func GetUserId(c *gin.Context) int64 {
	return GetLoginUserID(c)
}

func GetUserType(c *gin.Context) int {
	user := GetLoginUser(c)
	if user == nil {
		return 0 // Default to Member
	}
	return user.UserType
}

// GetLoginUser 获取完整的登录用户信息
func GetLoginUser(c *gin.Context) *LoginUser {
	v, exists := c.Get(CtxLoginUserKey)
	if !exists {
		return nil
	}
	if user, ok := v.(*LoginUser); ok {
		return user
	}
	return nil
}

// SetLoginUser 设置登录用户信息到上下文
func SetLoginUser(c *gin.Context, user *LoginUser) {
	if user != nil {
		c.Set(CtxUserIDKey, user.UserID)
		c.Set(CtxLoginUserKey, user)
		c.Request = c.Request.WithContext(WithLoginUser(c.Request.Context(), user))
	}
}

// GetTenantId 获得租户编号
func GetTenantId(c *gin.Context) int64 {
	user := GetLoginUser(c)
	if user == nil {
		return 0
	}
	return user.TenantID
}

// GetLoginUserFromContext 从context.Context中获取登录用户
// 用于非Gin场景(如GORM回调)
func GetLoginUserFromContext(ctx context.Context) *LoginUser {
	if ctx == nil {
		return nil
	}

	if user, ok := ctx.Value(CtxLoginUserKey).(*LoginUser); ok {
		return user
	}

	// 尝试从context中获取gin.Context
	if ginCtx, ok := ctx.Value(CtxGinContextKey).(*gin.Context); ok {
		return GetLoginUser(ginCtx)
	}

	// 尝试直接从context中获取LoginUser
	if user, ok := ctx.Value(CtxLoginUserKey).(*LoginUser); ok {
		return user
	}

	return nil
}

// WithLoginUser carries a copy of a trusted identity into jobs and database calls.
func WithLoginUser(ctx context.Context, user *LoginUser) context.Context {
	if user == nil {
		return ctx
	}
	identity := *user
	return context.WithValue(ctx, CtxLoginUserKey, &identity)
}

// WithTenant is for tenant identities resolved by trusted server-side configuration or records.
func WithTenant(ctx context.Context, tenantID int64) context.Context {
	return context.WithValue(ctx, CtxTenantIDKey, tenantID)
}

func TenantID(ctx context.Context) (int64, bool) {
	if user := GetLoginUserFromContext(ctx); user != nil {
		return user.TenantID, user.TenantID > 0
	}
	if ctx == nil {
		return 0, false
	}
	id, _ := ctx.Value(CtxTenantIDKey).(int64)
	return id, id > 0
}
