package pay

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type PayNotifyHandler struct {
	svc         *paySvc.PayNotifyService
	appSvc      *paySvc.PayAppService
	channelSvc  *paySvc.PayChannelService
	orderSvc    *paySvc.PayOrderService
	refundSvc   *paySvc.PayRefundService
	transferSvc *paySvc.PayTransferService
	logger      *zap.Logger
}

func NewPayNotifyHandler(
	svc *paySvc.PayNotifyService,
	appSvc *paySvc.PayAppService,
	channelSvc *paySvc.PayChannelService,
	orderSvc *paySvc.PayOrderService,
	refundSvc *paySvc.PayRefundService,
	transferSvc *paySvc.PayTransferService,
	logger *zap.Logger,
) *PayNotifyHandler {
	return &PayNotifyHandler{
		svc:         svc,
		appSvc:      appSvc,
		channelSvc:  channelSvc,
		orderSvc:    orderSvc,
		refundSvc:   refundSvc,
		transferSvc: transferSvc,
		logger:      logger,
	}
}

// NotifyOrder 支付渠道的统一【支付】回调
// POST /pay/notify/order/:channelId
// 对齐 Java: PayNotifyController.notifyOrder
func (h *PayNotifyHandler) NotifyOrder(c *gin.Context) {
	channelId := utils.ParseInt64(c.Param("channelId"))
	h.logger.Info("[NotifyOrder] 收到支付回调", zap.Int64("channelId", channelId))

	// 1. 获取 PayClient
	payClient, err := h.channelSvc.GetPayClient(c.Request.Context(), channelId)
	if err != nil {
		writeChannelNotifyFailure(c)
		return
	}
	if payClient == nil {
		h.logger.Error("[NotifyOrder] 渠道编号找不到对应的支付客户端", zap.Int64("channelId", channelId))
		writeChannelNotifyFailure(c)
		return
	}

	// 2. 解析回调数据
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		writeChannelNotifyFailure(c)
		return
	}
	if strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			writeChannelNotifyFailure(c)
			return
		}
		c.Request.PostForm = values
	}
	notifyData := &client.NotifyData{
		Params:  h.queryToMap(c),
		Body:    string(body),
		Headers: h.headerToMap(c),
	}

	orderResp, err := payClient.ParseOrderNotify(notifyData)
	if err != nil {
		h.logger.Error("[NotifyOrder] 解析回调数据失败", zap.Error(err))
		writeChannelNotifyFailure(c)
		return
	}

	// 3. 处理回调
	if err := h.orderSvc.NotifyOrder(c.Request.Context(), channelId, orderResp); err != nil {
		h.logger.Error("[NotifyOrder] 处理回调失败", zap.Error(err))
		writeChannelNotifyFailure(c)
		return
	}

	h.logger.Info("[NotifyOrder] 支付回调处理成功", zap.Int64("channelId", channelId))
	c.String(http.StatusOK, "success")
}

// NotifyRefund 支付渠道的统一【退款】回调
// POST /pay/notify/refund/:channelId
// 对齐 Java: PayNotifyController.notifyRefund
func (h *PayNotifyHandler) NotifyRefund(c *gin.Context) {
	channelId := utils.ParseInt64(c.Param("channelId"))
	h.logger.Info("[NotifyRefund] 收到退款回调", zap.Int64("channelId", channelId))

	// 1. 获取 PayClient
	payClient, err := h.channelSvc.GetPayClient(c.Request.Context(), channelId)
	if err != nil {
		writeChannelNotifyFailure(c)
		return
	}
	if payClient == nil {
		h.logger.Error("[NotifyRefund] 渠道编号找不到对应的支付客户端", zap.Int64("channelId", channelId))
		writeChannelNotifyFailure(c)
		return
	}

	// 2. 解析回调数据
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		writeChannelNotifyFailure(c)
		return
	}
	if strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			writeChannelNotifyFailure(c)
			return
		}
		c.Request.PostForm = values
	}
	notifyData := &client.NotifyData{
		Params:  h.queryToMap(c),
		Body:    string(body),
		Headers: h.headerToMap(c),
	}

	refundResp, err := payClient.ParseRefundNotify(notifyData)
	if err != nil {
		h.logger.Error("[NotifyRefund] 解析回调数据失败", zap.Error(err))
		writeChannelNotifyFailure(c)
		return
	}

	// 3. 处理回调
	if err := h.refundSvc.NotifyRefund(c.Request.Context(), channelId, refundResp); err != nil {
		h.logger.Error("[NotifyRefund] 处理回调失败", zap.Error(err))
		writeChannelNotifyFailure(c)
		return
	}

	h.logger.Info("[NotifyRefund] 退款回调处理成功", zap.Int64("channelId", channelId))
	c.String(http.StatusOK, "success")
}

// NotifyTransfer 支付渠道的统一【转账】回调
// POST /pay/notify/transfer/:channelId
// 对齐 Java: PayNotifyController.notifyTransfer
func (h *PayNotifyHandler) NotifyTransfer(c *gin.Context) {
	channelId := utils.ParseInt64(c.Param("channelId"))
	h.logger.Info("[NotifyTransfer] 收到转账回调", zap.Int64("channelId", channelId))

	// 1. 获取 PayClient
	payClient, err := h.channelSvc.GetPayClient(c.Request.Context(), channelId)
	if err != nil {
		writeChannelNotifyFailure(c)
		return
	}
	if payClient == nil {
		h.logger.Error("[NotifyTransfer] 渠道编号找不到对应的支付客户端", zap.Int64("channelId", channelId))
		writeChannelNotifyFailure(c)
		return
	}

	// 2. 解析回调数据
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		writeChannelNotifyFailure(c)
		return
	}
	if strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			writeChannelNotifyFailure(c)
			return
		}
		c.Request.PostForm = values
	}
	notifyData := &client.NotifyData{
		Params:  h.queryToMap(c),
		Body:    string(body),
		Headers: h.headerToMap(c),
	}

	transferResp, err := payClient.ParseTransferNotify(notifyData)
	if err != nil {
		h.logger.Error("[NotifyTransfer] 解析回调数据失败", zap.Error(err))
		writeChannelNotifyFailure(c)
		return
	}

	// 3. 处理回调
	// 注意：需要注入 transferSvc
	if h.transferSvc == nil {
		writeChannelNotifyFailure(c)
		return
	}
	if h.transferSvc != nil {
		if err := h.transferSvc.NotifyTransfer(c.Request.Context(), channelId, transferResp); err != nil {
			h.logger.Error("[NotifyTransfer] 处理回调失败", zap.Error(err))
			writeChannelNotifyFailure(c)
			return
		}
	}

	h.logger.Info("[NotifyTransfer] 转账回调处理成功", zap.Int64("channelId", channelId))
	c.String(http.StatusOK, "success")
}

// GetNotifyTaskDetail 获得回调通知详情 (Task + Logs)
func (h *PayNotifyHandler) GetNotifyTaskDetail(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	task, err := h.svc.GetNotifyTask(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	if task == nil {
		response.WriteSuccess(c, &pay2.PayNotifyTaskDetailResp{})
		return
	}

	logs, _ := h.svc.GetNotifyLogList(c, id)
	app, _ := h.appSvc.GetApp(c, task.AppID)

	r := &pay2.PayNotifyTaskDetailResp{}
	copier.Copy(&r.PayNotifyTaskResp, task)
	if app != nil {
		r.AppName = app.Name
	}

	logResps := make([]*pay2.PayNotifyLogResp, 0, len(logs))
	for _, log := range logs {
		lr := &pay2.PayNotifyLogResp{}
		copier.Copy(lr, log)
		logResps = append(logResps, lr)
	}
	r.Logs = logResps

	response.WriteSuccess(c, r)
}

// GetNotifyTaskPage 获得回调通知分页
func (h *PayNotifyHandler) GetNotifyTaskPage(c *gin.Context) {
	var r pay2.PayNotifyTaskPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	pageResult, err := h.svc.GetNotifyTaskPage(c, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	var appIds []int64
	for _, item := range pageResult.List {
		appIds = append(appIds, item.AppID)
	}
	appMap, _ := h.appSvc.GetAppMap(c, appIds)

	list := make([]*pay2.PayNotifyTaskResp, 0, len(pageResult.List))
	for _, item := range pageResult.List {
		tr := &pay2.PayNotifyTaskResp{}
		copier.Copy(tr, item)
		if app, ok := appMap[item.AppID]; ok {
			tr.AppName = app.Name
		}
		list = append(list, tr)
	}

	response.WriteSuccess(c, pagination.PageResult[*pay2.PayNotifyTaskResp]{
		List:  list,
		Total: pageResult.Total,
	})
}

// Helpers: 将 query 和 header 转换为 map
func (h *PayNotifyHandler) queryToMap(c *gin.Context) map[string]string {
	result := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

func (h *PayNotifyHandler) headerToMap(c *gin.Context) map[string]string {
	result := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// Helpers
func convertNotifyTaskResp(task *pay.PayNotifyTask, app *pay.PayApp) *pay2.PayNotifyTaskResp {
	r := &pay2.PayNotifyTaskResp{}
	copier.Copy(r, task)
	if app != nil {
		r.AppName = app.Name
	}
	return r
}

func writeChannelNotifyFailure(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "notification was not accepted"})
}
