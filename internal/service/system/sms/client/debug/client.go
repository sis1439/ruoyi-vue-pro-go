package debug

import (
	"context"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client"

	"go.uber.org/zap"
)

type SmsClient struct {
	id        int64
	apiKey    string
	apiSecret string
	signature string
}

func NewSmsClient(channel *model.SystemSmsChannel) (client.SmsClient, error) {
	return &SmsClient{
		id:        channel.ID,
		apiKey:    channel.ApiKey,
		apiSecret: channel.ApiSecret,
		signature: channel.Signature,
	}, nil
}

func (c *SmsClient) GetCode() string {
	return consts.SMSChannelCodeDebug
}

func (c *SmsClient) SendSms(ctx context.Context, mobile string, apiTemplateId string, templateParams []client.KeyValue) (*client.SmsSendResp, error) {
	zap.L().Info("Debug SMS send simulated")
	return &client.SmsSendResp{
		ApiSendCode:  "SUCCESS",
		ApiSendMsg:   "Debug Send Success",
		ApiRequestId: "DEBUG_REQUEST_ID",
		ApiSerialNo:  "DEBUG_SERIAL_NO",
	}, nil
}
