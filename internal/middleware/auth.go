package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
)

const (
	// Redis Key 前缀：访问令牌，与 Java 保持一致
	RedisKeyAccessToken = "oauth2_access_token:%s"
)

// OAuth2AccessToken 访问令牌结构（用于从 Redis 解析）
type OAuth2AccessToken struct {
	AccessToken  string            `json:"accessToken"`
	RefreshToken string            `json:"refreshToken"`
	UserID       int64             `json:"userId"`
	UserType     int               `json:"userType"`
	TenantID     int64             `json:"tenantId"`
	UserInfo     map[string]string `json:"userInfo"`
}

// Auth Middleware for JWT authentication
// 使用 JWT + Redis 白名单双重验证机制
func Auth() gin.HandlerFunc { return authenticate(false) }

func authenticate(optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticateIdentity(c, optional) {
			c.Next()
		}
	}
}
func authenticateIdentity(c *gin.Context, optional bool) bool {
	token := obtainAuthorization(c)
	if token == "" && optional {
		return true
	}
	claims, err := utils.ParseToken(token)
	if err != nil {
		c.AbortWithStatusJSON(401, response.Error(401, "Token无效"))
		return false
	}
	expected := 2
	if strings.HasPrefix(c.Request.URL.Path, "/app-api/") {
		expected = 1
	}
	if claims.UserType != expected {
		c.AbortWithStatusJSON(403, response.Error(403, "用户类型不匹配"))
		return false
	}
	if tenant := c.GetHeader("tenant-id"); tenant != "" && tenant != strconv.FormatInt(claims.TenantID, 10) {
		c.AbortWithStatusJSON(403, response.Error(403, "租户不匹配"))
		return false
	}
	if visit := c.GetHeader("visit-tenant-id"); visit != "" && visit != strconv.FormatInt(claims.TenantID, 10) {
		c.AbortWithStatusJSON(403, response.Error(403, "跨租户访问未授权"))
		return false
	}
	if cache.RDB == nil {
		c.AbortWithStatusJSON(503, response.Error(503, "认证服务不可用"))
		return false
	}
	data, err := cache.RDB.Get(c.Request.Context(), fmt.Sprintf(RedisKeyAccessToken, token)).Bytes()
	if err != nil {
		c.AbortWithStatusJSON(401, response.Error(401, "Token已失效或认证服务不可用"))
		return false
	}
	var stored OAuth2AccessToken
	if json.Unmarshal(data, &stored) != nil || stored.AccessToken != token || stored.UserID != claims.UserID || stored.UserType != claims.UserType || stored.TenantID != claims.TenantID {
		c.AbortWithStatusJSON(401, response.Error(401, "Token无效"))
		return false
	}
	context.SetLoginUser(c, &context.LoginUser{UserID: claims.UserID, UserType: claims.UserType, TenantID: claims.TenantID, Nickname: claims.Nickname})
	return true
}

// obtainAuthorization 从请求头或参数中获取 Authorization Token
// 支持 Header 和 Parameter 两种方式，与 Java SecurityFrameworkUtils.obtainAuthorization 对齐
func obtainAuthorization(c *gin.Context) string {
	// 1. 先从 Authorization Header 获取
	token := c.GetHeader("Authorization")
	if token != "" {
		// Remove Bearer prefix if present
		if len(token) > 7 && strings.ToUpper(token[0:7]) == "BEARER " {
			token = token[7:]
		}
		return token
	}

	// 2. 再从 Query Parameter 获取（备选方案）
	token = c.Query("Authorization")
	if token != "" {
		// Remove Bearer prefix if present
		if len(token) > 7 && strings.ToUpper(token[0:7]) == "BEARER " {
			token = token[7:]
		}
		return token
	}

	// 3. 最后从 Form Parameter 获取
	// Preserve URL-encoded callback bytes for downstream signature checks.
	// Do not buffer multipart uploads; ParseMultipartForm may stream them to disk.
	if strings.EqualFold(c.ContentType(), "application/x-www-form-urlencoded") && c.Request.Body != nil && c.Request.PostForm == nil {
		original := c.Request.Body
		var consumed bytes.Buffer
		c.Request.Body = io.NopCloser(io.TeeReader(original, &consumed))
		token = c.PostForm("Authorization")
		c.Request.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(&consumed, original), original}
	} else {
		token = c.PostForm("Authorization")
	}
	if token != "" {
		// Remove Bearer prefix if present
		if len(token) > 7 && strings.ToUpper(token[0:7]) == "BEARER " {
			token = token[7:]
		}
		return token
	}

	return ""
}

// OptionalAuth allows anonymous requests only when no credential was supplied.
func OptionalAuth() gin.HandlerFunc { return authenticate(true) }
