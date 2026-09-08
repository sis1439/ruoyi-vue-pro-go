package middleware

import (
	"crypto/subtle"

	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// PayNotifyToken 校验支付中心 → 商城业务通知的内部调用令牌。
// 这些路由（如 /trade/order/update-paid）是公开可达的，令牌用于建立信任边界；
// 令牌之外，接收端仍必须重新查询可信支付记录校验状态、金额与归属，两者不可互相替代。
//
// 未配置令牌时放行（本地开发）——config.Validate 已保证非 local 环境必须配置。
func PayNotifyToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := config.C.Pay.NotifyToken
		if expected == "" {
			c.Next()
			return
		}
		got := c.GetHeader(pay.PayNotifyTokenHeader)
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			response.WriteError(c, 401, "无效的通知令牌")
			c.Abort()
			return
		}
		c.Next()
	}
}
