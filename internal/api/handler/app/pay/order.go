package pay

import (
	"strconv"

	adminPay "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/pay"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	payWalletSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay/wallet"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppPayOrderHandler struct {
	svc       *paySvc.PayOrderService
	walletSvc *payWalletSvc.PayWalletService
}

func NewAppPayOrderHandler(svc *paySvc.PayOrderService, walletSvc *payWalletSvc.PayWalletService) *AppPayOrderHandler {
	return &AppPayOrderHandler{svc: svc, walletSvc: walletSvc}
}

// GetOrder 获得支付订单
func (h *AppPayOrderHandler) GetOrder(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteError(c, 400, "参数错误")
		return
	}

	if err := h.svc.ValidateMemberOrderOwner(c, id); err != nil {
		response.WriteBizError(c, err)
		return
	}
	// 处理 sync 参数
	sync := c.Query("sync") == "true"
	order, err := h.svc.GetOrder(c, id)
	if err == nil && order != nil && sync && order.Status == paySvc.PayOrderStatusWaiting {
		h.svc.SyncOrderQuietlyThrottled(c, id)
		// 重新拉取
		order, _ = h.svc.GetOrder(c, id)
	}

	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	if order == nil {
		response.WriteSuccess(c, nil)
		return
	}
	// 转换为对齐后的 VO
	vo := &adminPay.PayOrderResp{
		ID:              order.ID,
		AppID:           order.AppID,
		ChannelID:       order.ChannelID,
		ChannelCode:     order.ChannelCode,
		MerchantOrderId: order.MerchantOrderId,
		Subject:         order.Subject,
		Body:            order.Body,
		NotifyURL:       order.NotifyURL,
		Price:           int64(order.Price),
		ChannelFeeRate:  order.ChannelFeeRate,
		ChannelFeePrice: order.ChannelFeePrice,
		Status:          order.Status,
		UserIP:          order.UserIP,
		ExpireTime:      order.ExpireTime,
		SuccessTime:     order.SuccessTime,
		ExtensionID:     order.ExtensionID,
		No:              order.No,
		RefundPrice:     int64(order.RefundPrice),
		ChannelUserID:   order.ChannelUserID,
		ChannelOrderNo:  order.ChannelOrderNo,
		CreateTime:      order.CreateTime,
		Updater:         order.Updater,
	}

	response.WriteSuccess(c, vo)
}

// Submit 提交支付订单
func (h *AppPayOrderHandler) Submit(c *gin.Context) {
	var r pay.AppPayOrderSubmitReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	if err := h.svc.ValidateMemberOrderOwner(c, r.ID); err != nil {
		response.WriteBizError(c, err)
		return
	}
	// 1. 钱包支付处理
	if r.ChannelCode == "wallet" {
		if r.ChannelExtras == nil {
			r.ChannelExtras = make(map[string]string)
		}
		userID := context.GetUserId(c)
		userType := context.GetUserType(c)
		wallet, err := h.walletSvc.GetOrCreateWallet(c, userID, userType)
		if err != nil {
			response.WriteBizError(c, err)
			return
		}
		r.ChannelExtras["walletId"] = strconv.FormatInt(wallet.ID, 10)
	}

	// 2. 提交逻辑
	respVO, err := h.svc.SubmitOrder(c, &r.PayOrderSubmitReq, c.ClientIP())
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, respVO)
}
