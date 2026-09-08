package client

import (
	"fmt"
	"sync"
)

// PayClientFactory 支付客户端工厂
type PayClientFactory struct {
	clients map[int64]PayClient
	mutex   sync.RWMutex
}

func NewPayClientFactory() *PayClientFactory {
	return &PayClientFactory{
		clients: make(map[int64]PayClient),
	}
}

// GetPayClient 获得支付客户端。进程重启后缓存为空，调用方应改用
// PayChannelService.GetOrCreatePayClient 以便从数据库重建。
func (f *PayClientFactory) GetPayClient(channelID int64) PayClient {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.clients[channelID]
}

// ClientCreator 渠道客户端构造函数。channelCode 必须透传，渠道实现依赖它分派下单方式。
type ClientCreator func(channelID int64, channelCode string, config string) (PayClient, error)

var creators = make(map[string]ClientCreator)

func RegisterCreator(channelCode string, creator ClientCreator) {
	creators[channelCode] = creator
}

// SupportedChannelCodes 已注册的渠道编码，供配置校验使用
func SupportedChannelCodes() []string {
	codes := make([]string, 0, len(creators))
	for code := range creators {
		codes = append(codes, code)
	}
	return codes
}

// CreateOrUpdatePayClient 创建或更新支付客户端
func (f *PayClientFactory) CreateOrUpdatePayClient(channelID int64, channelCode string, config string) (PayClient, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	creator, ok := creators[channelCode]
	if !ok {
		// 不回退到 Mock：配置错误必须显式失败，否则会伪造支付成功
		return nil, fmt.Errorf("支付渠道 %s 未实现", channelCode)
	}

	newClient, err := creator(channelID, channelCode, config)
	if err != nil {
		return nil, err
	}
	if err := newClient.Init(); err != nil {
		return nil, err
	}
	f.clients[channelID] = newClient
	return newClient, nil
}

// RemovePayClient 渠道停用/删除后移除缓存，避免停用渠道仍可下单
func (f *PayClientFactory) RemovePayClient(channelID int64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.clients, channelID)
}
