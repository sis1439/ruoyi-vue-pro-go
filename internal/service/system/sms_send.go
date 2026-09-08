package system

import (
	"context"
	"fmt"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client"
	bzErr "github.com/wxlbd/ruoyi-mall-go/pkg/errors"

	"go.uber.org/zap"
)

type SmsSendService struct {
	q           *query.Query
	templateSvc *SmsTemplateService
	smsLogSvc   *SmsLogService
	factory     *SmsClientFactory
}

func NewSmsSendService(
	q *query.Query,
	templateSvc *SmsTemplateService,
	smsLogSvc *SmsLogService,
	factory *SmsClientFactory,
) *SmsSendService {
	return &SmsSendService{
		q:           q,
		templateSvc: templateSvc,
		smsLogSvc:   smsLogSvc,
		factory:     factory,
	}
}

// SendSingleSmsToAdmin 发送单条短信给 Admin 用户
func (s *SmsSendService) SendSingleSmsToAdmin(ctx context.Context, mobile string, userId int64, templateCode string, templateParams map[string]any) (int64, error) {
	// 如果 mobile 为空，查询 Admin 用户手机号 (此处暂略，假设 mobile 必传或调用者已处理)
	return s.SendSingleSms(ctx, mobile, userId, consts.UserTypeAdmin, templateCode, templateParams)
}

// SendSingleSmsToMember 发送单条短信给 Member 用户
func (s *SmsSendService) SendSingleSmsToMember(ctx context.Context, mobile string, userId int64, templateCode string, templateParams map[string]any) (int64, error) {
	// 如果 mobile 为空，查询 Member 用户手机号
	return s.SendSingleSms(ctx, mobile, userId, consts.UserTypeMember, templateCode, templateParams)
}

// SendSingleSms 发送单条短信（严格对齐 Java 实现）
func (s *SmsSendService) SendSingleSms(ctx context.Context, mobile string, userId int64, userType int32, templateCode string, templateParams map[string]any) (int64, error) {
	// 1. 校验短信模板是否合法
	template, err := s.validateSmsTemplate(ctx, templateCode)
	if err != nil {
		return 0, err
	}

	// 2. 校验短信渠道是否合法
	channel, err := s.validateSmsChannel(ctx, template.ChannelId)
	if err != nil {
		return 0, err
	}

	// 3. 校验手机号码是否存在
	mobile, err = s.validateMobile(mobile)
	if err != nil {
		return 0, err
	}

	// 4. 构建有序的模板参数（并验证所有参数都存在）
	kvParams, err := s.buildTemplateParams(template, templateParams)
	if err != nil {
		return 0, err
	}

	// 5. 判断是否需要发送（根据模板和渠道的启用状态）
	isSend := template.Status == consts.CommonStatusEnable && channel.Status == consts.CommonStatusEnable

	// 6. 格式化短信内容
	content := s.templateSvc.FormatSmsTemplateContent(template.Content, templateParams)

	// 7. 创建发送日志（根据 isSend 标志设置不同的状态）
	sendLogId, err := s.smsLogSvc.CreateSmsLogWithStatus(ctx, mobile, userId, userType, isSend, template, content, templateParams)
	if err != nil {
		zap.L().Error("Create SMS log failed")
		return 0, fmt.Errorf("SMS audit unavailable")
	}

	// 8. 只有当 isSend=true 时，才调用 Client 发送短信
	if !isSend {
		// 如果不需要发送，直接返回 logId（日志已经记录为 IGNORE 状态）
		return sendLogId, nil
	}

	// 9. 获取短信客户端并发送（直接创建/更新并获取客户端，对齐Java实现）
	smsClient, err := s.factory.CreateOrUpdateClient(channel)
	if err != nil {
		s.updateLogSendFail(ctx, sendLogId, fmt.Errorf("短信客户端初始化失败: %w", err))
		return sendLogId, fmt.Errorf("SMS client unavailable")
	}

	// 10. 执行发送
	sendResp, err := smsClient.SendSms(ctx, mobile, template.ApiTemplateId, kvParams)

	// 11. 更新日志
	if err != nil {
		s.updateLogSendFail(ctx, sendLogId, err)
		return sendLogId, fmt.Errorf("SMS delivery failed")
	}
	s.updateLogSendSuccess(ctx, sendLogId, sendResp)

	return sendLogId, nil
}

// validateSmsTemplate 验证短信模板
func (s *SmsSendService) validateSmsTemplate(ctx context.Context, templateCode string) (*model.SystemSmsTemplate, error) {
	t := s.q.SystemSmsTemplate
	template, err := t.WithContext(ctx).Where(t.Code.Eq(templateCode)).First()
	if err != nil {
		return nil, bzErr.NewBizError(1004003002, "短信模板不存在")
	}
	return template, nil
}

// validateSmsChannel 验证短信渠道
func (s *SmsSendService) validateSmsChannel(ctx context.Context, channelId int64) (*model.SystemSmsChannel, error) {
	c := s.q.SystemSmsChannel
	channel, err := c.WithContext(ctx).Where(c.ID.Eq(channelId)).First()
	if err != nil {
		return nil, bzErr.NewBizError(1004003001, "短信渠道不存在")
	}
	return channel, nil
}

// validateMobile 验证手机号不能为空
func (s *SmsSendService) validateMobile(mobile string) (string, error) {
	if mobile == "" {
		return "", bzErr.NewBizError(400, "手机号不能为空")
	}
	return mobile, nil
}

// buildTemplateParams 构建有序的模板参数并验证所有参数都存在
func (s *SmsSendService) buildTemplateParams(template *model.SystemSmsTemplate, templateParams map[string]any) ([]client.KeyValue, error) {
	result := make([]client.KeyValue, 0, len(template.Params))
	for _, key := range template.Params {
		value, exists := templateParams[key]
		if !exists || value == nil {
			return nil, bzErr.NewBizError(1004003003, fmt.Sprintf("缺失参数：%s", key))
		}
		result = append(result, client.KeyValue{Key: key, Value: value})
	}
	return result, nil
}

func (s *SmsSendService) updateLogSendFail(ctx context.Context, logId int64, err error) {
	now := time.Now()
	updates := map[string]any{
		"send_status":  consts.SmsSendStatusFailure,
		"send_time":    now,
		"api_send_msg": "SMS delivery failed",
	}
	_ = s.smsLogSvc.UpdateSmsLogFields(ctx, logId, updates)
}

func (s *SmsSendService) updateLogSendSuccess(ctx context.Context, logId int64, sendResp *client.SmsSendResp) {
	now := time.Now()
	updates := map[string]any{
		"send_status": consts.SmsSendStatusSuccess,
		"send_time":   now,
	}
	if sendResp != nil {
		updates["api_send_code"] = sendResp.ApiSendCode
		updates["api_send_msg"] = sendResp.ApiSendMsg
		updates["api_request_id"] = sendResp.ApiRequestId
		updates["api_serial_no"] = sendResp.ApiSerialNo
	}
	_ = s.smsLogSvc.UpdateSmsLogFields(ctx, logId, updates)
}
