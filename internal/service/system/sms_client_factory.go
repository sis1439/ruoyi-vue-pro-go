package system

import (
	"sync"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client/aliyun"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client/debug"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client/tencent"
	"go.uber.org/zap"
)

type SmsClientFactory struct {
	// code -> SmsClient
	clients sync.Map
}

func NewSmsClientFactory() *SmsClientFactory {
	return &SmsClientFactory{}
}

func (f *SmsClientFactory) GetClient(code string) client.SmsClient {
	if v, ok := f.clients.Load(code); ok {
		return v.(client.SmsClient)
	}
	return nil
}

// CreateOrUpdateClient 创建或更新短信客户端（对齐Java实现）
// 返回创建/更新后的客户端对象，内部处理缓存
func (f *SmsClientFactory) CreateOrUpdateClient(channel *model.SystemSmsChannel) (client.SmsClient, error) {
	c, err := f.createClient(channel)
	if err != nil {
		return nil, err
	}
	f.clients.Store(channel.Code, c)
	return c, nil
}

func (f *SmsClientFactory) createClient(channel *model.SystemSmsChannel) (client.SmsClient, error) {
	switch channel.Code {
	case consts.SMSChannelCodeAliyun:
		return aliyun.NewSmsClient(channel)
	case consts.SMSChannelCodeTencent:
		return tencent.NewSmsClient(channel)
	case consts.SMSChannelCodeDebug:
		return debug.NewSmsClient(channel)
	default:
		// Fallback to debug
		return debug.NewSmsClient(channel)
	}
}

// InitClients 初始化所有客户端
func (f *SmsClientFactory) InitClients(channels []*model.SystemSmsChannel) {
	for _, channel := range channels {
		if _, err := f.CreateOrUpdateClient(channel); err != nil {
			zap.L().Error("SMS client initialization failed", zap.Int64("channel_id", channel.ID))
		}
	}
}
