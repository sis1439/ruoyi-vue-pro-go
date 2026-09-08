package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	memberModel "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	systemSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"gorm.io/gorm"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TrustedTenant resolves anonymous traffic from registered hostnames, never a bare tenant header.
// Authenticated requests must also reference an active, unexpired tenant.
func TrustedTenant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/admin-api/") && !strings.HasPrefix(path, "/app-api/") {
			c.Next()
			return
		}
		if strings.Contains(path, "/system/tenant/get-") || strings.HasSuffix(path, "/system/tenant/simple-list") {
			c.Next()
			return
		}
		if path == "/app-api/member/auth/refresh-token" || path == "/admin-api/system/auth/refresh-token" {
			kind := 2
			if strings.HasPrefix(path, "/app-api/") {
				kind = 1
			}
			token, err := systemSvc.NewOAuth2TokenService().GetRefreshToken(c.Request.Context(), c.Query("refreshToken"), kind)
			if err != nil {
				c.AbortWithStatusJSON(401, response.Error(401, "刷新令牌无效"))
				return
			}
			header := c.GetHeader("tenant-id")
			visit := c.GetHeader("visit-tenant-id")
			expected := strconv.FormatInt(token.TenantID, 10)
			if (header != "" && header != expected) || (visit != "" && visit != expected) {
				c.AbortWithStatusJSON(403, response.Error(403, "租户不匹配"))
				return
			}
			var tenant model.SystemTenant
			if db == nil || db.WithContext(c.Request.Context()).Where("id = ? AND status = ? AND deleted = ?", token.TenantID, 0, model.BitBool(false)).First(&tenant).Error != nil || (!tenant.ExpireDate.IsZero() && !tenant.ExpireDate.After(time.Now())) {
				c.AbortWithStatusJSON(403, response.Error(403, "租户不可用"))
				return
			}
			c.Set(pkgContext.CtxTenantIDKey, token.TenantID)
			c.Request = c.Request.WithContext(pkgContext.WithTenant(c.Request.Context(), token.TenantID))
			c.Next()
			return
		}
		if obtainAuthorization(c) != "" {
			if !authenticateIdentity(c, true) {
				return
			}
			user := pkgContext.GetLoginUser(c)
			var tenant model.SystemTenant
			if db == nil || db.WithContext(c.Request.Context()).Where("id = ? AND status = ? AND deleted = ?", user.TenantID, 0, model.BitBool(false)).First(&tenant).Error != nil || (!tenant.ExpireDate.IsZero() && !tenant.ExpireDate.After(time.Now())) {
				c.AbortWithStatusJSON(403, response.Error(403, "租户不可用"))
				return
			}
			var identity any = &model.SystemUser{}
			if user.UserType == 1 {
				identity = &memberModel.MemberUser{}
			}
			var active int64
			if err := db.WithContext(c.Request.Context()).Model(identity).Where("id = ? AND tenant_id = ? AND status = ? AND deleted = ?", user.UserID, user.TenantID, 0, model.BitBool(false)).Count(&active).Error; err != nil || active != 1 {
				c.AbortWithStatusJSON(401, response.Error(401, "用户不可用"))
				return
			}
			if user.UserType == 2 {
				q := query.Use(db)
				if err := systemSvc.NewPermissionService(q, systemSvc.NewRoleService(q)).ValidateAdminScope(c.Request.Context(), user.UserID); err != nil {
					c.AbortWithStatusJSON(403, response.Error(403, "角色数据范围未启用或校验失败"))
					return
				}
			}
			c.Next()
			return
		}
		var tenants []model.SystemTenant
		if db == nil || db.WithContext(c.Request.Context()).Where("status = ? AND deleted = ?", 0, model.BitBool(false)).Find(&tenants).Error != nil {
			c.AbortWithStatusJSON(503, response.Error(503, "租户服务不可用"))
			return
		}
		host := tenantHostname(c.Request.Host)
		selected := int64(0)
		header := c.GetHeader("tenant-id")
		for _, tenant := range tenants {
			if tenant.ID <= 0 || (!tenant.ExpireDate.IsZero() && !tenant.ExpireDate.After(time.Now())) {
				continue
			}
			if header != "" && header != strconv.FormatInt(tenant.ID, 10) {
				continue
			}
			for _, website := range tenant.Websites {
				if host != "" && tenantHostname(website) == host {
					if selected != 0 && selected != tenant.ID {
						c.AbortWithStatusJSON(400, response.Error(400, "租户域名不唯一"))
						return
					}
					selected = tenant.ID
				}
			}
		}
		if selected <= 0 || (c.GetHeader("visit-tenant-id") != "" && c.GetHeader("visit-tenant-id") != strconv.FormatInt(selected, 10)) {
			c.AbortWithStatusJSON(403, response.Error(403, "租户未授权"))
			return
		}
		ctx := pkgContext.WithTenant(c.Request.Context(), selected)
		c.Request = c.Request.WithContext(ctx)
		// Gin.Context.Value checks Keys before delegating to Request.Context.
		c.Set(pkgContext.CtxTenantIDKey, selected)
		c.Next()
	}
}
func tenantHostname(raw string) string {
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		raw = u.Host
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	return strings.ToLower(strings.TrimSuffix(raw, "."))
}
