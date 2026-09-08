package system

import (
	"context"
	"strings"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/samber/lo"
)

type SmsLogService struct {
	q *query.Query
}

func NewSmsLogService(q *query.Query) *SmsLogService {
	return &SmsLogService{
		q: q,
	}
}

// CreateSmsLog 创建短信日志
func (s *SmsLogService) CreateSmsLog(ctx context.Context, item *model.SystemSmsLog) (int64, error) {
	redacted := redactSmsAudit(item)
	err := s.q.SystemSmsLog.WithContext(ctx).Create(redacted)
	item.ID = redacted.ID
	return item.ID, err
}

// CreateSmsLogWithStatus 创建短信日志（根据 isSend 参数设置不同的状态）
func (s *SmsLogService) CreateSmsLogWithStatus(ctx context.Context, mobile string, userId int64, userType int32, isSend bool,
	template *model.SystemSmsTemplate, content string, templateParams map[string]interface{}) (int64, error) {

	// 根据是否需要发送设置不同的状态
	sendStatus := consts.SmsSendStatusInit
	if !isSend {
		sendStatus = consts.SmsSendStatusIgnore
	}

	log := &model.SystemSmsLog{
		ChannelId:       template.ChannelId,
		ChannelCode:     template.ChannelCode,
		TemplateId:      template.ID,
		TemplateCode:    template.Code,
		TemplateType:    template.Type,
		TemplateContent: content,
		TemplateParams:  templateParams,
		ApiTemplateId:   template.ApiTemplateId,
		Mobile:          mobile,
		UserId:          userId,
		UserType:        userType,
		SendStatus:      sendStatus,
		SendTime:        nil,
		ReceiveStatus:   consts.SmsReceiveStatusInit,
	}
	return s.CreateSmsLog(ctx, log)
}

// UpdateSmsLog 更新短信日志
func (s *SmsLogService) UpdateSmsLog(ctx context.Context, item *model.SystemSmsLog) error {
	_, err := s.q.SystemSmsLog.WithContext(ctx).Where(s.q.SystemSmsLog.ID.Eq(item.ID)).Updates(redactSmsAudit(item))
	return err
}

// UpdateSmsLogFields 更新短信日志指定字段
func (s *SmsLogService) UpdateSmsLogFields(ctx context.Context, logId int64, updates map[string]interface{}) error {
	item, err := s.q.SystemSmsLog.WithContext(ctx).Where(s.q.SystemSmsLog.ID.Eq(logId)).First()
	if err != nil {
		return err
	}
	if sensitiveSmsAudit(item) {
		safe := make(map[string]any, len(updates))
		for key, value := range updates {
			safe[key] = value
		}
		for _, key := range []string{"template_content", "template_params", "mobile", "api_send_msg", "api_receive_msg", "api_request_id", "api_serial_no", "api_receive_code"} {
			if _, ok := safe[key]; ok {
				if key == "template_params" {
					safe[key] = map[string]any{"code": "[REDACTED]"}
				} else {
					safe[key] = "[REDACTED]"
				}
			}
		}
		if code, ok := safe["api_send_code"]; ok {
			if text, ok := code.(string); ok && strings.EqualFold(text, "OK") {
				safe["api_send_code"] = "OK"
			} else {
				safe["api_send_code"] = "FAILED"
			}
		}
		updates = safe
	}

	_, err = s.q.SystemSmsLog.WithContext(ctx).Where(s.q.SystemSmsLog.ID.Eq(logId)).Updates(updates)
	return err
}

// GetSmsLogPage 获得短信日志分页
func (s *SmsLogService) GetSmsLogPage(ctx context.Context, req *system.SmsLogPageReq) (*pagination.PageResult[*system.SmsLogRespVO], error) {
	l := s.q.SystemSmsLog
	qb := l.WithContext(ctx)

	if req.ChannelId != nil {
		qb = qb.Where(l.ChannelId.Eq(*req.ChannelId))
	}
	if req.TemplateId != nil {
		qb = qb.Where(l.TemplateId.Eq(*req.TemplateId))
	}
	if req.Mobile != "" {
		qb = qb.Where(l.Mobile.Like("%" + req.Mobile + "%"))
	}
	if req.SendStatus != nil {
		qb = qb.Where(l.SendStatus.Eq(*req.SendStatus))
	}
	if req.ReceiveStatus != nil {
		qb = qb.Where(l.ReceiveStatus.Eq(*req.ReceiveStatus))
	}

	total, err := qb.Count()
	if err != nil {
		return nil, err
	}

	list, err := qb.Order(l.ID.Desc()).Offset(req.GetOffset()).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*system.SmsLogRespVO]{
		List:  lo.Map(list, func(item *model.SystemSmsLog, _ int) *system.SmsLogRespVO { return s.convertResp(item) }),
		Total: total,
	}, nil
}

func (s *SmsLogService) convertResp(item *model.SystemSmsLog) *system.SmsLogRespVO {
	item = redactSmsAudit(item)
	return &system.SmsLogRespVO{
		ID:              item.ID,
		ChannelId:       item.ChannelId,
		ChannelCode:     item.ChannelCode,
		TemplateId:      item.TemplateId,
		TemplateCode:    item.TemplateCode,
		TemplateType:    item.TemplateType,
		TemplateContent: item.TemplateContent,
		TemplateParams:  item.TemplateParams,
		ApiTemplateId:   item.ApiTemplateId,
		Mobile:          item.Mobile,
		UserId:          item.UserId,
		UserType:        item.UserType,
		SendStatus:      item.SendStatus,
		SendTime:        item.SendTime,
		ApiSendCode:     item.ApiSendCode,
		ApiSendMsg:      item.ApiSendMsg,
		ApiRequestId:    item.ApiRequestId,
		ApiSerialNo:     item.ApiSerialNo,
		ReceiveStatus:   item.ReceiveStatus,
		ReceiveTime:     item.ReceiveTime,
		ApiReceiveCode:  item.ApiReceiveCode,
		ApiReceiveMsg:   item.ApiReceiveMsg,
		CreateTime:      item.CreateTime,
	}
}

// Redact on write and read: historical rows must not leak through page or export.
func sensitiveSmsAudit(item *model.SystemSmsLog) bool {
	for _, scene := range SceneMap {
		if item.TemplateCode == scene.TemplateCode {
			return true
		}
	}
	for key := range item.TemplateParams {
		if strings.EqualFold(key, "code") {
			return true
		}
	}
	return false
}
func redactSmsAudit(item *model.SystemSmsLog) *model.SystemSmsLog {
	copy := *item
	if !sensitiveSmsAudit(item) {
		return &copy
	}
	copy.TemplateContent = "[REDACTED]"
	copy.TemplateParams = map[string]any{"code": "[REDACTED]"}
	copy.Mobile = "[REDACTED]"
	copy.ApiSendMsg = "[REDACTED]"
	copy.ApiReceiveMsg = "[REDACTED]"
	copy.ApiRequestId = ""
	copy.ApiSerialNo = ""
	copy.ApiReceiveCode = ""
	if strings.EqualFold(copy.ApiSendCode, "OK") {
		copy.ApiSendCode = "OK"
	} else if copy.ApiSendCode != "" {
		copy.ApiSendCode = "FAILED"
	}
	return &copy
}
