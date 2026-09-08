package system

import (
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"net/url"
	"strconv"
	"time"

	system2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type SmsTemplateHandler struct {
	smsTemplateSvc *system.SmsTemplateService
	smsSendSvc     *system.SmsSendService
}

func NewSmsTemplateHandler(smsTemplateSvc *system.SmsTemplateService, smsSendSvc *system.SmsSendService) *SmsTemplateHandler {
	return &SmsTemplateHandler{
		smsTemplateSvc: smsTemplateSvc,
		smsSendSvc:     smsSendSvc,
	}
}

// CreateSmsTemplate 创建短信模板
func (h *SmsTemplateHandler) CreateSmsTemplate(c *gin.Context) {
	var req system2.SmsTemplateSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	id, err := h.smsTemplateSvc.CreateSmsTemplate(c, &req)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, id)
}

// UpdateSmsTemplate 更新短信模板
func (h *SmsTemplateHandler) UpdateSmsTemplate(c *gin.Context) {
	var req system2.SmsTemplateSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	if err := h.smsTemplateSvc.UpdateSmsTemplate(c, &req); err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// DeleteSmsTemplate 删除短信模板
func (h *SmsTemplateHandler) DeleteSmsTemplate(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	if err := h.smsTemplateSvc.DeleteSmsTemplate(c, id); err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// GetSmsTemplate 获得短信模板
func (h *SmsTemplateHandler) GetSmsTemplate(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	res, err := h.smsTemplateSvc.GetSmsTemplate(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}

// GetSmsTemplatePage 获得短信模板分页
func (h *SmsTemplateHandler) GetSmsTemplatePage(c *gin.Context) {
	var req system2.SmsTemplatePageReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	res, err := h.smsTemplateSvc.GetSmsTemplatePage(c, &req)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}

// ExportSmsTemplateExcel 导出短信模板 Excel
func (h *SmsTemplateHandler) ExportSmsTemplateExcel(c *gin.Context) {
	var req system2.SmsTemplatePageReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	// 设置分页大小为 -1（获取所有数据）
	req.PageSize = -1

	// 获取所有数据
	pageResult, err := h.smsTemplateSvc.GetSmsTemplatePage(c, &req)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 创建 Excel 文件
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "短信模板"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	f.SetActiveSheet(index)

	// 设置列头
	headers := []string{"编号", "短信类型", "开启状态", "模板编码", "模板名称", "模板内容", "参数数组", "备注", "短信 API 的模板编号", "短信渠道编号", "短信渠道编码", "创建时间"}
	for i, header := range headers {
		f.SetCellValue(sheetName, fmt.Sprintf("%c1", 'A'+i), header)
	}

	// 填充数据行
	for i, item := range pageResult.List {
		row := i + 2 // 从第2行开始
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.Type)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), item.Status)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), item.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.Content)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("%v", item.Params))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), item.Remark)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), item.ApiTemplateId)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), item.ChannelId)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), item.ChannelCode)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), item.CreateTime.Format("2006-01-02 15:04:05"))
	}

	// 设置列宽
	columnWidths := map[string]float64{
		"A": 10, "B": 12, "C": 12, "D": 15, "E": 15, "F": 20,
		"G": 20, "H": 15, "I": 20, "J": 15, "K": 15, "L": 20,
	}
	for col, width := range columnWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	// 设置下载响应头
	filename := fmt.Sprintf("短信模板_%d.xlsx", time.Now().Unix())
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", url.QueryEscape(filename)))

	// 写入 Excel 到响应
	if err := f.Write(c.Writer); err != nil {
		response.WriteBizError(c, err)
		return
	}
}

// SendSms 发送短信
func (h *SmsTemplateHandler) SendSms(c *gin.Context) {
	var req system2.SmsTemplateSendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	// 从 Context 获取当前登录用户 ID
	ctx := c.Request.Context()
	userId := context.GetUserId(c)

	logId, err := h.smsSendSvc.SendSingleSmsToAdmin(ctx, req.Mobile, userId, req.TemplateCode, req.TemplateParams)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, logId)
}
