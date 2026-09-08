package pay

import (
	"context"
	"errors"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"

	"gorm.io/gorm"
)

type PayChannelService struct {
	q             *query.Query
	clientFactory *client.PayClientFactory
}

func NewPayChannelService(q *query.Query, clientFactory *client.PayClientFactory) *PayChannelService {
	return &PayChannelService{
		q:             q,
		clientFactory: clientFactory,
	}
}

// CreateChannel 创建支付渠道
func (s *PayChannelService) CreateChannel(ctx context.Context, req *pay2.PayChannelCreateReq) (int64, error) {
	// 1. 校验是否重复 (AppID + Code)
	exists, err := s.GetChannelByAppIdAndCode(ctx, req.AppID, req.Code)
	if err != nil {
		return 0, err
	}
	if exists != nil {
		return 0, pkgErrors.NewBizError(1006002000, "支付渠道已存在") // PAY_CHANNEL_EXIST_SAME_CHANNEL_ERROR
	}

	// 2. 插入
	channel := &pay.PayChannel{
		Code:    req.Code,
		Status:  req.Status,
		FeeRate: req.FeeRate,
		Remark:  req.Remark,
		AppID:   req.AppID,
		Config:  req.Config,
	}
	err = s.q.PayChannel.WithContext(ctx).Create(channel)
	if err != nil {
		return 0, err
	}
	return channel.ID, nil
}

// UpdateChannel 更新支付渠道
func (s *PayChannelService) UpdateChannel(ctx context.Context, req *pay2.PayChannelUpdateReq) error {
	// 1. 校验存在
	_, err := s.validateChannelExists(ctx, req.ID)
	if err != nil {
		return err
	}

	// 2. 更新
	_, err = s.q.PayChannel.WithContext(ctx).Where(s.q.PayChannel.ID.Eq(req.ID)).Updates(pay.PayChannel{
		FeeRate: req.FeeRate,
		Remark:  req.Remark,
		Config:  req.Config,
	})
	if err == nil {
		s.clientFactory.RemovePayClient(req.ID) // 配置变更后下次调用重建
	}
	return err
}

// DeleteChannel 删除支付渠道
func (s *PayChannelService) DeleteChannel(ctx context.Context, id int64) error {
	// 1. 校验存在
	if _, err := s.validateChannelExists(ctx, id); err != nil {
		return err
	}
	// 2. 删除
	_, err := s.q.PayChannel.WithContext(ctx).Where(s.q.PayChannel.ID.Eq(id)).Delete()
	if err == nil {
		s.clientFactory.RemovePayClient(id)
	}
	return err
}

// GetChannel 获得支付渠道
func (s *PayChannelService) GetChannel(ctx context.Context, id int64) (*pay.PayChannel, error) {
	channel, err := s.q.PayChannel.WithContext(ctx).Where(s.q.PayChannel.ID.Eq(id)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 对齐Java: 查询不到返回null而不是异常
		}
		return nil, err
	}
	return channel, nil
}

// GetChannelListByAppIds 根据 AppID 集合获支付渠道列表
func (s *PayChannelService) GetChannelListByAppIds(ctx context.Context, appIds []int64) ([]*pay.PayChannel, error) {
	if len(appIds) == 0 {
		return nil, nil
	}
	return s.q.PayChannel.WithContext(ctx).Where(s.q.PayChannel.AppID.In(appIds...)).Find()
}

// Private Methods

// GetChannelByAppIdAndCode 根据 AppID 和 Code 获得支付渠道
func (s *PayChannelService) GetChannelByAppIdAndCode(ctx context.Context, appId int64, code string) (*pay.PayChannel, error) {
	channel, err := s.q.PayChannel.WithContext(ctx).Where(s.q.PayChannel.AppID.Eq(appId), s.q.PayChannel.Code.Eq(code)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 对齐Java: 查询不到返回null而不是异常
		}
		return nil, err
	}
	return channel, nil
}

// GetEnableChannelList 获得指定应用的开启的支付渠道列表
func (s *PayChannelService) GetEnableChannelList(ctx context.Context, appId int64) ([]*pay.PayChannel, error) {
	return s.q.PayChannel.WithContext(ctx).
		Where(s.q.PayChannel.AppID.Eq(appId), s.q.PayChannel.Status.Eq(0)). // 0 = Enabled
		Find()
}

func (s *PayChannelService) validateChannelExists(ctx context.Context, id int64) (*pay.PayChannel, error) {
	channel, err := s.q.PayChannel.WithContext(ctx).Where(s.q.PayChannel.ID.Eq(id)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.NewBizError(1006002002, "支付渠道不存在") // PAY_CHANNEL_NOT_FOUND
		}
		return nil, err
	}
	return channel, nil
}

// ValidPayChannel 校验支付渠道是否有效
func (s *PayChannelService) ValidPayChannel(ctx context.Context, id int64) (*pay.PayChannel, error) {
	channel, err := s.validateChannelExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if channel.Status != 0 { // 0 = Enabled
		return nil, pkgErrors.NewBizError(1006002001, "支付渠道处于关闭状态") // PAY_CHANNEL_IS_DISABLE
	}
	return channel, nil
}

// GetPayClient 获得支付客户端。缓存未命中时（进程重启、渠道配置变更后）
// 从数据库重新加载渠道并重建客户端；渠道不存在或已停用一律返回错误，
// 不回退到任何默认/Mock 实现。
// 对齐 Java: PayChannelService.getPayClient(Long id)
func (s *PayChannelService) GetPayClient(ctx context.Context, channelID int64) (client.PayClient, error) {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return nil, errors.New("trusted tenant required for payment client")
	}
	channel, err := s.ValidPayChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel.TenantID != tenant {
		return nil, errors.New("payment channel tenant mismatch")
	}
	app := s.q.PayApp
	application, err := app.WithContext(ctx).Where(app.ID.Eq(channel.AppID), app.TenantID.Eq(tenant)).First()
	if err != nil {
		return nil, err
	}
	if application.Status != 0 {
		return nil, errors.New("payment application disabled")
	}
	if s.clientFactory == nil {
		return nil, errors.New("payment client factory unavailable")
	}
	if channel.Config == nil {
		return nil, errors.New("payment channel configuration missing")
	}
	cached := s.clientFactory.GetPayClient(channelID)
	if cached != nil {
		return cached, nil
	}
	return s.clientFactory.CreateOrUpdatePayClient(channelID, channel.Code, channel.Config.ToJSON())
}
