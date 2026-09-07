package router

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app"
	"github.com/wxlbd/ruoyi-mall-go/internal/middleware"
	"gorm.io/gorm"
)

func InitRouter(db *gorm.DB, rdb *redis.Client,
	adminHandlers *admin.AdminHandlers,
	appHandlers *app.AppHandlers,
	casbinMiddleware *middleware.CasbinMiddleware,
) *gin.Engine {
	// Debug log to confirm router init
	fmt.Println("Initializing Router...")
	r := gin.New()
	// Forwarded client addresses are untrusted unless a deployment explicitly installs a proxy policy.
	_ = r.SetTrustedProxies(nil)
	r.Use(middleware.Recovery())
	r.Use(middleware.ErrorHandler())
	if origins := strings.TrimSpace(os.Getenv("MALL_CORS_ORIGINS")); origins != "" {
		allowed := strings.Split(origins, ",")
		for i := range allowed {
			allowed[i] = strings.TrimSpace(allowed[i])
			if allowed[i] == "*" {
				panic("MALL_CORS_ORIGINS must contain explicit origins")
			}
		}
		r.Use(cors.New(cors.Config{
			AllowOrigins: allowed,
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Authorization", "Content-Type", "Tenant-Id", "Visit-Tenant-Id", "Terminal", "Platform"},
		}))
	}
	// Query strings can carry refresh tokens; never include them in access logs.
	r.Use(gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		return fmt.Sprintf("%s %s %d %s\n", p.Method, p.Request.URL.Path, p.StatusCode, p.Latency)
	}))
	// 注入 gin.Context 到 request context，供 GORM Hook 使用
	r.Use(middleware.InjectContext())
	r.Use(middleware.TrustedTenant(db))
	r.Use(middleware.LoginRateLimit(rdb))

	// 基础路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// ========== 模块化路由注册 ==========

	// System 模块
	// WebSocket (Register at root /infra/ws to match Java path)
	r.GET("/infra/ws", adminHandlers.Infra.WebSocket.Handle) // Corrected WebSocketHandler reference

	// System 模块
	RegisterSystemRoutes(r, adminHandlers.System, adminHandlers.Infra, casbinMiddleware)

	// Product 模块
	RegisterProductRoutes(r, adminHandlers.Mall.Product, casbinMiddleware)

	// Promotion 模块
	RegisterPromotionRoutes(r, adminHandlers.Mall.Promotion, casbinMiddleware)

	// Trade 模块
	RegisterTradeRoutes(r, adminHandlers.Mall.Trade, casbinMiddleware)

	// Member 模块 (Admin)
	RegisterMemberRoutes(r, adminHandlers.Member, casbinMiddleware)

	// Pay 模块
	RegisterPayRoutes(r, adminHandlers.Pay, casbinMiddleware)

	// App 模块 (移动端)
	RegisterAppRoutes(r, appHandlers)

	// Statistics 模块
	RegisterStatisticsRoutes(r, adminHandlers.Statistics, casbinMiddleware)

	// Area 地区路由
	RegisterAreaRoutes(r, adminHandlers.System.Area)

	// IOT 模块
	RegisterIotRoutes(r, adminHandlers.Iot, casbinMiddleware)

	return r
}
