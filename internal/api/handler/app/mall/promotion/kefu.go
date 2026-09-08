package promotion

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"strconv"

	"github.com/gin-gonic/gin"
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"

	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
)

type AppKefuHandler struct {
	svc promotion.KefuService
}

func NewAppKefuHandler(svc promotion.KefuService) *AppKefuHandler {
	return &AppKefuHandler{svc: svc}
}

// GetMessageList 获得消息列表 (对齐 Java AppKeFuMessageController.getKefuMessageList)
func (h *AppKefuHandler) GetMessageList(c *gin.Context) {
	var r promotion2.KefuMessageListReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	// 获取当前登录用户ID (对齐 Java: getLoginUserId())
	userID := context.GetUserId(c)
	// 调用会员端消息列表方法 (对齐 Java: getKeFuMessageList(pageReqVO, getLoginUserId()))
	res, err := h.svc.GetMessageListForMember(c, r, userID)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}

// UpdateMessageReadStatus 更新客服消息已读状态
func (h *AppKefuHandler) UpdateMessageReadStatus(c *gin.Context) {
	conversationID, _ := strconv.ParseInt(c.Query("conversationId"), 10, 64)
	if conversationID == 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	// 获取当前登录用户ID
	userID := context.GetUserId(c)
	if err := h.svc.UpdateMessageReadStatus(c, conversationID, userID, 1); err != nil { // SenderType 1 = User
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// SendMessage 发送消息
func (h *AppKefuHandler) SendMessage(c *gin.Context) {
	var r promotion2.KefuMessageCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	// 获取当前登录用户ID
	userID := context.GetUserId(c)                  // 假设中间件注入了 userId
	id, err := h.svc.CreateMessage(c, r, userID, 1) // SenderType 1 = User
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, id)
}
