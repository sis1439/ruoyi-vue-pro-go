package pay

import (
	"context"
	"errors"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

type PayAppService struct {
	q          *query.Query
	channelSvc *PayChannelService
}

func NewPayAppService(q *query.Query, channelSvc *PayChannelService) *PayAppService {
	return &PayAppService{
		q:          q,
		channelSvc: channelSvc,
	}
}

// CreateApp 创建支付应用
func (s *PayAppService) CreateApp(ctx context.Context, req *pay2.PayAppCreateReq) (int64, error) {
	// 1. 校验 AppKey 唯一
	if err := s.validateAppKeyUnique(ctx, 0, req.AppKey); err != nil {
		return 0, err
	}

	// 2. 插入
	app := &pay.PayApp{
		AppKey:            req.AppKey,
		Name:              req.Name,
		Status:            req.Status,
		Remark:            req.Remark,
		OrderNotifyURL:    req.OrderNotifyURL,
		RefundNotifyURL:   req.RefundNotifyURL,
		TransferNotifyURL: req.TransferNotifyURL,
	}
	err := s.q.PayApp.WithContext(ctx).Create(app)
	if err != nil {
		return 0, err
	}
	return app.ID, nil
}

// UpdateApp 更新支付应用
func (s *PayAppService) UpdateApp(ctx context.Context, req *pay2.PayAppUpdateReq) error {
	// 1. 校验存在
	if _, err := s.validateAppExists(ctx, req.ID); err != nil {
		return err
	}
	// 2. 校验 AppKey 唯一
	if err := s.validateAppKeyUnique(ctx, req.ID, req.AppKey); err != nil {
		return err
	}

	// 3. 更新
	_, err := s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.ID.Eq(req.ID)).Updates(pay.PayApp{
		AppKey:            req.AppKey,
		Name:              req.Name,
		Status:            req.Status,
		Remark:            req.Remark,
		OrderNotifyURL:    req.OrderNotifyURL,
		RefundNotifyURL:   req.RefundNotifyURL,
		TransferNotifyURL: req.TransferNotifyURL,
	})
	return err
}

// UpdateAppStatus 更新支付应用状态
func (s *PayAppService) UpdateAppStatus(ctx context.Context, req *pay2.PayAppUpdateStatusReq) error {
	// 1. 校验存在
	if _, err := s.validateAppExists(ctx, req.ID); err != nil {
		return err
	}
	// 2. 更新
	_, err := s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.ID.Eq(req.ID)).Update(s.q.PayApp.Status, req.Status)
	return err
}

// DeleteApp 删除支付应用
func (s *PayAppService) DeleteApp(ctx context.Context, id int64) error {
	// 1. 校验存在
	if _, err := s.validateAppExists(ctx, id); err != nil {
		return err
	}
	// TODO: 校验关联数据 (Order, Refund)

	// 2. 删除
	_, err := s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.ID.Eq(id)).Delete()
	return err
}

// GetApp 获得支付应用
func (s *PayAppService) GetApp(ctx context.Context, id int64) (*pay.PayApp, error) {
	return s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.ID.Eq(id)).First()
}

// GetAppByAppKey 根据 AppKey 获得支付应用
func (s *PayAppService) GetAppByAppKey(ctx context.Context, appKey string) (*pay.PayApp, error) {
	q := repo.QueryFromContext(ctx, s.q)
	return q.PayApp.WithContext(ctx).Where(q.PayApp.AppKey.Eq(appKey)).First()
}

// GetAppMap 获得支付应用 Map
func (s *PayAppService) GetAppMap(ctx context.Context, ids []int64) (map[int64]*pay.PayApp, error) {
	if len(ids) == 0 {
		return map[int64]*pay.PayApp{}, nil
	}
	list, err := s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	appMap := make(map[int64]*pay.PayApp, len(list))
	for _, app := range list {
		appMap[app.ID] = app
	}
	return appMap, nil
}

// GetAppList 获得所有支付应用列表
func (s *PayAppService) GetAppList(ctx context.Context) ([]*pay.PayApp, error) {
	return s.q.PayApp.WithContext(ctx).Find()
}

// GetAppPage 获得支付应用分页
func (s *PayAppService) GetAppPage(ctx context.Context, req *pay2.PayAppPageReq) (*pagination.PageResult[*pay2.PayAppPageItemResp], error) {
	q := s.q.PayApp.WithContext(ctx)
	if req.Name != "" {
		q = q.Where(s.q.PayApp.Name.Like("%" + req.Name + "%"))
	}
	if req.Status != nil {
		q = q.Where(s.q.PayApp.Status.Eq(*req.Status))
	}
	if req.Remark != "" {
		q = q.Where(s.q.PayApp.Remark.Like("%" + req.Remark + "%"))
	}

	apps, total, err := q.FindByPage(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}

	// 获得应用渠道
	appIds := lo.Map(apps, func(item *pay.PayApp, _ int) int64 {
		return item.ID
	})
	channels, err := s.channelSvc.GetChannelListByAppIds(ctx, appIds)
	if err != nil {
		return nil, err
	}

	// 转换结果
	list := make([]*pay2.PayAppPageItemResp, len(apps))
	for i, app := range apps {
		item := &pay2.PayAppPageItemResp{
			PayAppResp: pay2.PayAppResp{
				ID:                app.ID,
				AppKey:            app.AppKey,
				Name:              app.Name,
				Status:            app.Status,
				Remark:            app.Remark,
				OrderNotifyURL:    app.OrderNotifyURL,
				RefundNotifyURL:   app.RefundNotifyURL,
				TransferNotifyURL: app.TransferNotifyURL,
				CreateTime:        app.CreateTime,
			},
			ChannelCodes: []string{},
		}
		// 匹配渠道
		for _, channel := range channels {
			if channel.AppID == app.ID && channel.Status == consts.CommonStatusEnable {
				item.ChannelCodes = append(item.ChannelCodes, channel.Code)
			}
		}
		list[i] = item
	}

	return &pagination.PageResult[*pay2.PayAppPageItemResp]{
		List:  list,
		Total: total,
	}, nil
}

// Private Methods

func (s *PayAppService) validateAppExists(ctx context.Context, id int64) (*pay.PayApp, error) {
	app, err := s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.ID.Eq(id)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.NewBizError(1006000000, "支付应用不存在") // PAY_APP_NOT_FOUND
		}
		return nil, err
	}
	return app, nil
}

func (s *PayAppService) validateAppKeyUnique(ctx context.Context, id int64, appKey string) error {
	app, err := s.q.PayApp.WithContext(ctx).Where(s.q.PayApp.AppKey.Eq(appKey)).First()
	if err == nil && app != nil {
		if id == 0 || app.ID != id {
			return pkgErrors.NewBizError(1006000001, "支付应用 AppKey 已存在") // PAY_APP_EXIST_KEY_ERROR
		}
	}
	return nil
}

// ValidPayApp 校验支付应用是否有效
func (s *PayAppService) ValidPayApp(ctx context.Context, id int64) (*pay.PayApp, error) {
	app, err := s.validateAppExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if app.Status != consts.CommonStatusEnable { // ENABLE = 0
		return nil, pkgErrors.NewBizError(1006000002, "支付应用处于关闭状态") // PAY_APP_IS_DISABLE
	}
	return app, nil
}

// ValidPayAppByAppKey 校验支付应用是否有效
func (s *PayAppService) ValidPayAppByAppKey(ctx context.Context, appKey string) (*pay.PayApp, error) {
	app, err := s.GetAppByAppKey(ctx, appKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgErrors.NewBizError(1006000000, "支付应用不存在") // PAY_APP_NOT_FOUND
		}
		return nil, err
	}
	if app.Status != consts.CommonStatusEnable { // ENABLE = 0
		return nil, pkgErrors.NewBizError(1006000002, "支付应用处于关闭状态") // PAY_APP_IS_DISABLE
	}
	return app, nil
}
