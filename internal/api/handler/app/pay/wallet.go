package pay

import (
	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/pay"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	payWalletSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay/wallet"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"github.com/gin-gonic/gin"
)

type AppPayWalletHandler struct {
	walletSvc   *payWalletSvc.PayWalletService
	rechargeSvc *payWalletSvc.PayWalletRechargeService
	payOrderSvc *paySvc.PayOrderService
}

func NewAppPayWalletHandler(
	walletSvc *payWalletSvc.PayWalletService,
	rechargeSvc *payWalletSvc.PayWalletRechargeService,
	payOrderSvc *paySvc.PayOrderService,
) *AppPayWalletHandler {
	return &AppPayWalletHandler{
		walletSvc:   walletSvc,
		rechargeSvc: rechargeSvc,
		payOrderSvc: payOrderSvc,
	}
}

// GetWallet 获得钱包
func (h *AppPayWalletHandler) GetWallet(c *gin.Context) {
	userId := context.GetUserId(c)
	userType := context.GetUserType(c)

	wallet, err := h.walletSvc.GetOrCreateWallet(c, userId, userType)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, pay.AppPayWalletResp{
		Balance:       wallet.Balance,
		TotalExpense:  wallet.TotalExpense,
		TotalRecharge: wallet.TotalRecharge,
	})
}

// CreateRecharge 创建钱包充值记录
func (h *AppPayWalletHandler) CreateRecharge(c *gin.Context) {
	var r pay.AppPayWalletRechargeCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	userId := context.GetUserId(c)
	userType := context.GetUserType(c)
	userIP := c.ClientIP()
	recharge, err := h.rechargeSvc.CreateWalletRecharge(c, &pay2.PayWalletRechargeCreateReq{
		UserID:    userId,
		UserType:  userType,
		PayPrice:  *r.PayPrice,
		PackageID: *r.PackageID,
	}, userIP)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, pay.AppPayWalletRechargeCreateResp{
		ID:         recharge.ID,
		PayOrderID: recharge.PayOrderID,
	})
}

// GetRechargePage 获得钱包充值记录分页
func (h *AppPayWalletHandler) GetRechargePage(c *gin.Context) {
	var r pagination.PageParam
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}

	pageResult, err := h.rechargeSvc.GetWalletRechargePage(c, &pay2.PayWalletRechargePageReq{
		PageParam: r,
	})
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	if len(pageResult.List) == 0 {
		response.WriteSuccess(c, pagination.PageResult[pay.AppPayWalletRechargeResp]{
			List:  []pay.AppPayWalletRechargeResp{},
			Total: pageResult.Total,
		})
		return
	}

	// 拼接支付单数据
	payOrderIDs := make([]int64, len(pageResult.List))
	for i, item := range pageResult.List {
		payOrderIDs[i] = item.PayOrderID
	}
	payOrderMap, _ := h.payOrderSvc.GetOrderMap(c, payOrderIDs)

	list := make([]pay.AppPayWalletRechargeResp, len(pageResult.List))
	for i, item := range pageResult.List {
		r := pay.AppPayWalletRechargeResp{
			ID:           item.ID,
			TotalPrice:   item.TotalPrice,
			PayPrice:     item.PayPrice,
			BonusPrice:   item.BonusPrice,
			PayOrderID:   item.PayOrderID,
			RefundStatus: item.RefundStatus,
		}
		if order, ok := payOrderMap[item.PayOrderID]; ok {
			r.PayChannelCode = order.ChannelCode
			r.PayTime = types.ToJsonDateTimePtr(order.SuccessTime)
			r.PayOrderChannelOrderNo = order.ChannelOrderNo
			// 暂未实现 channelName 映射，可通过 ChannelCode 简单展示
			r.PayChannelName = order.ChannelCode
		}
		list[i] = r
	}

	response.WriteSuccess(c, pagination.PageResult[pay.AppPayWalletRechargeResp]{
		List:  list,
		Total: pageResult.Total,
	})
}
