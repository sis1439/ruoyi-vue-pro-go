package pay

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/pay"
	payWalletSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay/wallet"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
)

type AppPayWalletTransactionHandler struct {
	svc       *payWalletSvc.PayWalletTransactionService
	walletSvc *payWalletSvc.PayWalletService
}

func NewAppPayWalletTransactionHandler(svc *payWalletSvc.PayWalletTransactionService, walletSvc *payWalletSvc.PayWalletService) *AppPayWalletTransactionHandler {
	return &AppPayWalletTransactionHandler{svc: svc, walletSvc: walletSvc}
}

// GetWalletTransactionPage 获得钱包流水分页
func (h *AppPayWalletTransactionHandler) GetWalletTransactionPage(c *gin.Context) {
	userId := context.GetUserId(c)
	userType := context.GetUserType(c)

	// 1. 获取钱包
	wallet, err := h.walletSvc.GetOrCreateWallet(c, userId, userType)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 2. 查询流水
	var r pay.PayWalletTransactionPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}
	r.CreateTime = types.QueryTimeRange(c.Request.URL.Query(), "createTime")
	r.WalletID = wallet.ID

	pageResult, err := h.svc.GetWalletTransactionPage(c, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 3. 转换返回
	list := make([]pay2.AppPayWalletTransactionResp, len(pageResult.List))
	for i, item := range pageResult.List {
		list[i] = pay2.AppPayWalletTransactionResp{
			BizType:    item.BizType,
			Price:      int64(item.Price),
			Title:      item.Title,
			CreateTime: types.ToJsonDateTime(item.CreateTime),
		}
	}

	response.WriteSuccess(c, pagination.PageResult[pay2.AppPayWalletTransactionResp]{
		List:  list,
		Total: pageResult.Total,
	})
}

// GetWalletTransactionSummary 获得钱包流水统计
func (h *AppPayWalletTransactionHandler) GetWalletTransactionSummary(c *gin.Context) {
	userId := context.GetUserId(c)
	userType := context.GetUserType(c)

	// 解析时间参数
	createTimeStrs := c.QueryArray("createTime")
	var createTime []time.Time
	if len(createTimeStrs) == 2 {
		for _, ts := range createTimeStrs {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, time.Local)
			if err == nil {
				createTime = append(createTime, t)
			}
		}
	}

	totalIncome, totalExpense, err := h.svc.GetWalletTransactionSummary(c, userId, userType, createTime)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, pay2.AppPayWalletTransactionSummaryResp{
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
	})
}
