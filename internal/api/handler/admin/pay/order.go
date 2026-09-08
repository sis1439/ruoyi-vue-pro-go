package pay

import (
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strconv"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	payWalletSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay/wallet"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type PayOrderHandler struct {
	svc       *paySvc.PayOrderService
	appSvc    *paySvc.PayAppService
	walletSvc *payWalletSvc.PayWalletService
}

func NewPayOrderHandler(svc *paySvc.PayOrderService, appSvc *paySvc.PayAppService, walletSvc *payWalletSvc.PayWalletService) *PayOrderHandler {
	return &PayOrderHandler{svc: svc, appSvc: appSvc, walletSvc: walletSvc}
}

// GetOrder 获得支付订单
func (h *PayOrderHandler) GetOrder(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	// Handle sync param
	syncStr := c.Query("sync")
	if syncStr == "true" {
		order, err := h.svc.GetOrder(c, id)
		if err == nil && order.Status == consts.PayOrderStatusWaiting {
			h.svc.SyncOrderQuietly(c, id)
		}
	}

	order, err := h.svc.GetOrder(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 填充应用名称
	resp := convertOrderResp(order)
	if order.AppID > 0 {
		if app, err := h.appSvc.GetApp(c, order.AppID); err == nil && app != nil {
			resp.AppName = app.Name
		}
	}

	response.WriteSuccess(c, resp)
}

// GetOrderDetail 获得支付订单详情
func (h *PayOrderHandler) GetOrderDetail(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	order, err := h.svc.GetOrder(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	app, _ := h.appSvc.GetApp(c, order.AppID)
	extension, _ := h.svc.GetOrderExtension(c, order.ExtensionID)

	detail := &pay2.PayOrderDetailsResp{
		PayOrderResp: *convertOrderResp(order),
		Extension:    convertExtensionResp(extension),
		App:          convertAppResp(app),
	}

	// 填充应用名称
	if app != nil {
		detail.AppName = app.Name
	}

	response.WriteSuccess(c, detail)
}

// GetOrderPage 获得支付订单分页
func (h *PayOrderHandler) GetOrderPage(c *gin.Context) {
	var r pay2.PayOrderPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	pageResult, err := h.svc.GetOrderPage(c, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Fetch apps for mapping
	appIds := make([]int64, 0, len(pageResult.List))
	for _, order := range pageResult.List {
		appIds = append(appIds, order.AppID)
	}
	appMap, _ := h.appSvc.GetAppMap(c, appIds)

	list := make([]pay2.PayOrderResp, 0, len(pageResult.List))
	for _, order := range pageResult.List {
		r := *convertOrderResp(order)
		if app, ok := appMap[order.AppID]; ok {
			r.AppName = app.Name
		}
		list = append(list, r)
	}

	response.WriteSuccess(c, pagination.PageResult[pay2.PayOrderResp]{
		List:  list,
		Total: pageResult.Total,
	})
}

// SubmitPayOrder 提交支付订单
func (h *PayOrderHandler) SubmitPayOrder(c *gin.Context) {
	var r pay2.PayOrderSubmitReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	// 1. Wallet payment case
	if r.ChannelCode == "wallet" { // Assuming "wallet" is the code
		if r.ChannelExtras == nil {
			r.ChannelExtras = make(map[string]string)
		}
		userID := context.GetLoginUserID(c)
		user := context.GetLoginUser(c)
		userType := consts.UserTypeUnknown // 默认未知用户类型
		if user != nil {
			userType = user.UserType
		}
		wallet, err := h.walletSvc.GetOrCreateWallet(c, userID, userType)
		if err != nil {
			response.WriteBizError(c, err)
			return
		}
		r.ChannelExtras["wallet_id"] = strconv.FormatInt(wallet.ID, 10)
	}

	// 2. Submit Order
	respVO, err := h.svc.SubmitOrder(c, &r, c.ClientIP())
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, respVO)
}

// ExportOrderExcel 导出支付订单 Excel
func (h *PayOrderHandler) ExportOrderExcel(c *gin.Context) {
	var r pay2.PayOrderExportReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	list, err := h.svc.GetOrderList(c, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Fetch Apps for naming
	appIds := make([]int64, 0, len(list))
	for _, order := range list {
		appIds = append(appIds, order.AppID)
	}
	appMap, _ := h.appSvc.GetAppMap(c, appIds)

	// Create Excel
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheetName := "Sheet1"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	f.SetActiveSheet(index)

	// Headers
	// 对齐 Java PayOrderExcelVO 顺序: 编号, 创建时间, 支付金额, 退款金额, 手续金额, 商户单号, 支付单号, 渠道单号, 支付状态, 渠道编号名称, 订单支付成功时间, 订单失效时间, 应用名称, 商品标题, 商品描述
	headers := []string{"编号", "创建时间", "支付金额", "退款金额", "手续金额", "商户单号", "支付单号", "渠道单号", "支付状态", "渠道编号名称", "订单支付成功时间", "订单失效时间", "应用名称", "商品标题", "商品描述"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Data
	for i, item := range list {
		row := i + 2
		appName := ""
		if app, ok := appMap[item.AppID]; ok {
			appName = app.Name
		}

		statusStr := "未知"
		switch item.Status {
		case consts.PayOrderStatusWaiting:
			statusStr = "等待支付"
		case consts.PayOrderStatusSuccess:
			statusStr = "支付成功"
		case consts.PayOrderStatusClosed:
			statusStr = "支付关闭"
		case consts.PayOrderStatusRefund:
			statusStr = "已退款"
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.CreateTime.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), float64(item.Price)/100.0)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), float64(item.RefundPrice)/100.0)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), float64(item.ChannelFeePrice)/100.0)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.MerchantOrderId)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), item.No)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), item.ChannelOrderNo)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), statusStr)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), item.ChannelCode)
		if item.SuccessTime != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), item.SuccessTime.Format("2006-01-02 15:04:05"))
		}
		if !item.ExpireTime.IsZero() {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), item.ExpireTime.Format("2006-01-02 15:04:05"))
		}
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), appName)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), item.Subject)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), item.Body)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=pay_order_list.xlsx")
	if err := f.Write(c.Writer); err != nil {
		response.WriteBizError(c, err)
		return
	}
}

// Helpers

func convertOrderResp(order *pay.PayOrder) *pay2.PayOrderResp {
	if order == nil {
		return nil
	}
	return &pay2.PayOrderResp{
		ID:              order.ID,
		AppID:           order.AppID,
		ChannelID:       order.ChannelID,
		ChannelCode:     order.ChannelCode,
		MerchantOrderId: order.MerchantOrderId,
		Subject:         order.Subject,
		Body:            order.Body,
		NotifyURL:       order.NotifyURL,
		Price:           int64(order.Price), // 转换为 int64
		ChannelFeeRate:  order.ChannelFeeRate,
		ChannelFeePrice: order.ChannelFeePrice,
		Status:          order.Status,
		UserIP:          order.UserIP,
		ExpireTime:      types.ToJsonDateTime(order.ExpireTime),
		SuccessTime:     types.ToJsonDateTimePtr(order.SuccessTime),
		ExtensionID:     order.ExtensionID,
		No:              order.No,
		RefundPrice:     int64(order.RefundPrice), // 转换为 int64
		ChannelUserID:   order.ChannelUserID,
		ChannelOrderNo:  order.ChannelOrderNo,
		CreateTime:      types.ToJsonDateTime(order.CreateTime),
		UpdateTime:      types.ToJsonDateTime(order.UpdateTime),
		Creator:         order.Creator,
		Updater:         order.Updater,
	}
}

func convertExtensionResp(ext *pay.PayOrderExtension) *pay2.PayOrderExtensionResp {
	if ext == nil {
		return nil
	}
	return &pay2.PayOrderExtensionResp{
		ID:                ext.ID,
		No:                ext.No,
		OrderID:           ext.OrderID,
		ChannelID:         ext.ChannelID,
		ChannelCode:       ext.ChannelCode,
		UserIP:            ext.UserIP,
		Status:            ext.Status,
		ChannelExtras:     ext.ChannelExtras,
		ChannelErrorCode:  ext.ChannelErrorCode,
		ChannelErrorMsg:   ext.ChannelErrorMsg,
		ChannelNotifyData: ext.ChannelNotifyData,
		CreateTime:        ext.CreateTime,
	}
}

func convertAppResp(app *pay.PayApp) *pay2.PayAppResp {
	// Duplicated from PayAppHandler to avoid import cycle or dependency issues
	if app == nil {
		return nil
	}
	return &pay2.PayAppResp{
		ID:                app.ID,
		AppKey:            app.AppKey,
		Name:              app.Name,
		Status:            app.Status,
		Remark:            app.Remark,
		OrderNotifyURL:    app.OrderNotifyURL,
		RefundNotifyURL:   app.RefundNotifyURL,
		TransferNotifyURL: app.TransferNotifyURL,
		CreateTime:        app.CreateTime,
	}
}
