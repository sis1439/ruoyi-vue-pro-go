package pay

import (
	"github.com/gin-gonic/gin"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
)

type AppPayTransferHandler struct {
	svc *paySvc.PayTransferService
}

func NewAppPayTransferHandler(svc *paySvc.PayTransferService) *AppPayTransferHandler {
	return &AppPayTransferHandler{svc: svc}
}

// SyncTransfer 同步转账单
// SyncTransfer is outside the first-phase member scope: the transfer model has no member owner.
func (h *AppPayTransferHandler) SyncTransfer(c *gin.Context) {
	c.AbortWithStatusJSON(403, response.Error(403, "会员转账同步未启用"))
}
