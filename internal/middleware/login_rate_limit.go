package middleware

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Increment and expiry are one operation; denied requests never extend the window.
var loginBudget = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return n
`)

// LoginRateLimit must run after InjectContext and TrustedTenant. All authentication
// methods share a tenant/peer-IP budget, preventing endpoint switching bypasses.
// Peer IP deliberately ignores forwarding headers until a trusted proxy policy exists.
func LoginRateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		bucket, limit := loginRateBucket(c.Request.URL.Path)
		if bucket == "" || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		tenant, ok := pkgContext.TenantID(c.Request.Context())
		if !ok {
			c.AbortWithStatusJSON(403, response.Error(403, "租户未授权"))
			return
		}
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		ip := net.ParseIP(host)
		if err != nil || ip == nil {
			c.AbortWithStatusJSON(400, response.Error(400, "客户端地址无效"))
			return
		}
		if c.Request.ContentLength > 16<<10 {
			c.AbortWithStatusJSON(413, response.Error(413, "请求过大"))
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
		}
		if rdb == nil {
			c.AbortWithStatusJSON(503, response.Error(503, "认证服务不可用"))
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		count, err := loginBudget.Run(ctx, rdb.WithTimeout(time.Second), []string{fmt.Sprintf("security:login-rate:%d:%s:%s", tenant, ip.String(), bucket)}, 60000).Int64()
		cancel()
		if err != nil {
			c.AbortWithStatusJSON(503, response.Error(503, "认证服务不可用"))
			return
		}
		if count > limit {
			c.Header("Retry-After", strconv.Itoa(60))
			c.AbortWithStatusJSON(429, response.Error(429, "请求过于频繁"))
			return
		}
		c.Next()
	}
}

func loginRateBucket(path string) (string, int64) {
	if path == "/admin-api/system/captcha/get" || path == "/admin-api/system/captcha/check" {
		return "captcha", 30
	}
	if strings.HasPrefix(path, "/admin-api/system/auth/") || strings.HasPrefix(path, "/app-api/member/auth/") || path == "/app-api/member/user/reset-password" {
		return "auth", 10
	}
	return "", 0
}
