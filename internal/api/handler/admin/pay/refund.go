package pay

import (
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/xuri/excelize/v2"
)

type PayRefundHandler struct {
	svc      *paySvc.PayRefundService
	appSvc   *paySvc.PayAppService
	orderSvc *paySvc.PayOrderService
}

func NewPayRefundHandler(svc *paySvc.PayRefundService, appSvc *paySvc.PayAppService, orderSvc *paySvc.PayOrderService) *PayRefundHandler {
	return &PayRefundHandler{
		svc:      svc,
		appSvc:   appSvc,
		orderSvc: orderSvc,
	}
}

// GetRefund 获得退款订单
func (h *PayRefundHandler) GetRefund(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	refund, err := h.svc.GetRefund(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	if refund == nil {
		response.WriteSuccess(c, &pay2.PayRefundDetailsResp{})
		return
	}

	app, _ := h.appSvc.GetApp(c, refund.AppID)

	// 查询原订单信息用于填充 Order 对象
	var order *pay.PayOrder
	if refund.OrderID > 0 {
		order, _ = h.orderSvc.GetOrder(c, refund.OrderID)
	}

	r := convertRefundDetailsResp(refund, app, order)
	response.WriteSuccess(c, r)
}

// GetRefundPage 获得退款订单分页
func (h *PayRefundHandler) GetRefundPage(c *gin.Context) {
	var r pay2.PayRefundPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	pageResult, err := h.svc.GetRefundPage(c, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Enrich App Info
	var appIds []int64
	for _, item := range pageResult.List {
		appIds = append(appIds, item.AppID)
	}
	appMap, _ := h.appSvc.GetAppMap(c, appIds)

	list := make([]*pay2.PayRefundResp, 0, len(pageResult.List))
	for _, item := range pageResult.List {
		list = append(list, convertRefundResp(item, appMap[item.AppID]))
	}

	response.WriteSuccess(c, pagination.PageResult[*pay2.PayRefundResp]{
		List:  list,
		Total: pageResult.Total,
	})
}

// ExportRefundExcel 导出退款订单 Excel
func (h *PayRefundHandler) ExportRefundExcel(c *gin.Context) {
	var r pay2.PayRefundExportReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	list, err := h.svc.GetRefundList(c, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Fetch Apps for naming
	appIds := make([]int64, 0, len(list))
	for _, item := range list {
		appIds = append(appIds, item.AppID)
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
	// 对齐 Java PayRefundExcelVO 顺序: 支付退款编号, 创建时间, 支付金额, 退款金额, 商户退款单号, 退款单号, 渠道退款单号, 商户支付单号, 渠道支付单号, 退款状态, 退款渠道, 成功时间, 支付应用, 退款原因
	headers := []string{"支付退款编号", "创建时间", "支付金额", "退款金额", "商户退款单号", "退款单号", "渠道退款单号", "商户支付单号", "渠道支付单号", "退款状态", "退款渠道", "成功时间", "支付应用", "退款原因"}
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
		case consts.PayRefundStatusWaiting:
			statusStr = "等待退款"
		case consts.PayRefundStatusSuccess:
			statusStr = "退款成功"
		case consts.PayRefundStatusFailure:
			statusStr = "退款失败"
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.CreateTime.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), float64(item.PayPrice)/100.0) // MoneyConvert
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), float64(item.RefundPrice)/100.0)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), item.MerchantRefundId)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.No)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), item.ChannelRefundNo)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), item.MerchantOrderId)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), item.ChannelOrderNo)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), statusStr)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), item.ChannelCode)
		if item.SuccessTime != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), item.SuccessTime.Format("2006-01-02 15:04:05"))
		}
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), appName)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), item.Reason)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=pay_refund_list.xlsx")
	if err := f.Write(c.Writer); err != nil {
		response.WriteBizError(c, err)
		return
	}
}

// Helpers

func convertRefundResp(refund *pay.PayRefund, app *pay.PayApp) *pay2.PayRefundResp {
	r := &pay2.PayRefundResp{}
	copier.Copy(r, refund)
	if app != nil {
		r.AppName = app.Name
	}
	return r
}

func convertRefundDetailsResp(refund *pay.PayRefund, app *pay.PayApp, order *pay.PayOrder) *pay2.PayRefundDetailsResp {
	r := &pay2.PayRefundDetailsResp{}
	copier.Copy(&r.PayRefundResp, refund)
	if app != nil {
		r.AppName = app.Name
		r.App = &pay2.PayAppResp{}
		copier.Copy(r.App, app)
	}
	// 填充原订单信息
	if order != nil {
		r.Order = &pay2.RefundOrder{
			Subject: order.Subject,
		}
	}
	return r
}
