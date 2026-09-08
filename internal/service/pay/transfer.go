package pay

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"

	reqPay "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	respPay "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	modelPay "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	repoPay "github.com/wxlbd/ruoyi-mall-go/internal/repo/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

const transferNoPrefix = "T"

type PayTransferService struct {
	transferRepo  repoPay.PayTransferRepository
	appSvc        *PayAppService
	channelSvc    *PayChannelService
	notifySvc     *PayNotifyService
	clientFactory *client.PayClientFactory
	noRedisDAO    *repoPay.PayNoRedisDAO
	logger        *zap.Logger
}

func NewPayTransferService(
	transferRepo repoPay.PayTransferRepository,
	appSvc *PayAppService,
	channelSvc *PayChannelService,
	notifySvc *PayNotifyService,
	clientFactory *client.PayClientFactory,
	noRedisDAO *repoPay.PayNoRedisDAO,
	logger *zap.Logger,
) *PayTransferService {
	return &PayTransferService{
		transferRepo:  transferRepo,
		appSvc:        appSvc,
		channelSvc:    channelSvc,
		notifySvc:     notifySvc,
		clientFactory: clientFactory,
		noRedisDAO:    noRedisDAO,
		logger:        logger,
	}
}

// GetTransferPage 获得转账单分页
func (s *PayTransferService) GetTransferPage(ctx context.Context, req *reqPay.PayTransferPageReq) (*pagination.PageResult[*respPay.PayTransferResp], error) {
	// 1. 查询转账单分页
	pageResult, err := s.transferRepo.SelectPage(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. 拼接应用名称
	appIds := lo.Map(pageResult.List, func(item *modelPay.PayTransfer, _ int) int64 {
		return item.AppID
	})
	appMap, err := s.appSvc.GetAppMap(ctx, appIds)
	if err != nil {
		return nil, err
	}

	// 3. 转换返回结果
	resultList := make([]*respPay.PayTransferResp, len(pageResult.List))
	for i, item := range pageResult.List {
		resp := &respPay.PayTransferResp{
			ID:                 item.ID,
			No:                 item.No,
			AppID:              item.AppID,
			ChannelID:          item.ChannelID,
			ChannelCode:        item.ChannelCode,
			MerchantTransferID: item.MerchantTransferID,
			Subject:            item.Subject,
			Price:              item.Price,
			UserAccount:        item.UserAccount,
			UserName:           item.UserName,
			Status:             item.Status,
			SuccessTime:        item.SuccessTime,
			NotifyURL:          item.NotifyURL,
			UserIP:             item.UserIP,
			ChannelExtras:      item.ChannelExtras,
			ChannelTransferNo:  item.ChannelTransferNo,
			ChannelErrorCode:   item.ChannelErrorCode,
			ChannelErrorMsg:    item.ChannelErrorMsg,
			ChannelNotifyData:  item.ChannelNotifyData,
			ChannelPackageInfo: item.ChannelPackageInfo,
			CreateTime:         item.CreateTime,
			UpdateTime:         item.UpdateTime,
			Creator:            item.Creator,
			Updater:            item.Updater,
			Deleted:            item.Deleted,
			TenantID:           item.TenantID,
		}
		if app, ok := appMap[item.AppID]; ok {
			resp.AppName = app.Name
		}
		resultList[i] = resp
	}

	return pagination.NewPageResult(resultList, pageResult.Total), nil
}

// GetTransfer 获得转账单
func (s *PayTransferService) GetTransfer(ctx context.Context, id int64) (*respPay.PayTransferResp, error) {
	transfer, err := s.transferRepo.SelectById(ctx, id)
	if err != nil {
		return nil, err
	}
	if transfer == nil {
		return nil, nil
	}

	resp := &respPay.PayTransferResp{
		ID:                 transfer.ID,
		No:                 transfer.No,
		AppID:              transfer.AppID,
		ChannelID:          transfer.ChannelID,
		ChannelCode:        transfer.ChannelCode,
		MerchantTransferID: transfer.MerchantTransferID,
		Subject:            transfer.Subject,
		Price:              transfer.Price,
		UserAccount:        transfer.UserAccount,
		UserName:           transfer.UserName,
		Status:             transfer.Status,
		SuccessTime:        transfer.SuccessTime,
		NotifyURL:          transfer.NotifyURL,
		UserIP:             transfer.UserIP,
		ChannelExtras:      transfer.ChannelExtras,
		ChannelTransferNo:  transfer.ChannelTransferNo,
		ChannelErrorCode:   transfer.ChannelErrorCode,
		ChannelErrorMsg:    transfer.ChannelErrorMsg,
		ChannelNotifyData:  transfer.ChannelNotifyData,
		ChannelPackageInfo: transfer.ChannelPackageInfo,
		CreateTime:         transfer.CreateTime,
		UpdateTime:         transfer.UpdateTime,
		Creator:            transfer.Creator,
		Updater:            transfer.Updater,
		Deleted:            transfer.Deleted,
		TenantID:           transfer.TenantID,
	}

	app, err := s.appSvc.GetApp(ctx, transfer.AppID)
	if err != nil {
		return nil, err
	}
	if app != nil {
		resp.AppName = app.Name
	}
	return resp, nil
}

// CreateTransfer 创建转账单
// 对齐 Java: PayTransferServiceImpl.createTransfer
func (s *PayTransferService) CreateTransfer(ctx context.Context, req *reqPay.PayTransferCreateReq) (*respPay.PayTransferCreateResp, error) {
	// 1.1 校验 App
	app, err := s.appSvc.ValidPayApp(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	// 1.2 校验支付渠道是否有效
	channel, err := s.channelSvc.GetChannelByAppIdAndCode(ctx, app.ID, req.ChannelCode)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, errors.New("pay channel not found")
	}
	if _, err := s.channelSvc.ValidPayChannel(ctx, channel.ID); err != nil {
		return nil, err
	}

	payClient, err := s.channelSvc.GetPayClient(ctx, channel.ID)
	if err != nil {
		return nil, err
	}
	if payClient == nil {
		s.logger.Error("[createTransfer][渠道编号找不到对应的支付客户端]", zap.Int64("channelId", channel.ID))
		return nil, errors.New("pay client not found")
	}

	// 1.3 校验转账单已经发起过转账
	transfer, err := s.validateTransferCanCreate(ctx, req, app.ID)
	if err != nil {
		return nil, err
	}

	// 2.1 情况一：不存在创建转账单，则进行创建
	if transfer == nil {
		no, err := s.noRedisDAO.Generate(ctx, transferNoPrefix)
		if err != nil {
			return nil, err
		}
		transfer = &modelPay.PayTransfer{
			No:                 no,
			AppID:              channel.AppID,
			ChannelID:          channel.ID,
			ChannelCode:        req.ChannelCode,
			MerchantTransferID: req.MerchantTransferID,
			Subject:            req.Subject,
			Price:              req.Price,
			Type:               req.Type,
			UserName:           req.UserName,
			UserAccount:        req.UserAccount,
			Status:             consts.PayTransferStatusWaiting,
			NotifyURL:          app.TransferNotifyURL,
			UserIP:             req.UserIP,
			ChannelExtras:      req.ChannelExtras,
		}
		if req.Type == consts.PayTransferTypeWxBalance {
			transfer.UserAccount = req.OpenID
		} else if req.Type == consts.PayTransferTypeAlipayBalance {
			transfer.UserAccount = req.AlipayLogonID
		}
		if err := s.transferRepo.Create(ctx, transfer); err != nil {
			return nil, err
		}
	} else {
		// 2.2 情况二：存在创建转账单，但是状态为关闭，则更新为等待中
		_, err := s.transferRepo.UpdateByIdAndStatus(ctx, transfer.ID, []int{consts.PayTransferStatusClosed},
			&modelPay.PayTransfer{Status: consts.PayTransferStatusWaiting})
		if err != nil {
			return nil, err
		}
		transfer.Status = consts.PayTransferStatusWaiting
	}

	// 3. 调用三方渠道发起转账
	var unifiedTransferResp *client.TransferResp
	unifiedReq := &client.UnifiedTransferReq{
		OutTradeNo:    transfer.No,
		Subject:       transfer.Subject,
		Price:         transfer.Price,
		UserName:      transfer.UserName,
		UserAccount:   transfer.UserAccount,
		UserIP:        req.UserIP,
		ChannelExtras: req.ChannelExtras,
	}
	unifiedTransferResp, err = payClient.UnifiedTransfer(ctx, unifiedReq)
	if err != nil {
		// 注意这里仅打印异常，不进行抛出
		// 原因是：虽然调用支付渠道进行转账发生异常（网络请求超时），实际转账成功。这个结果，后续转账轮询可以拿到。
		s.logger.Error("[createTransfer][转账编号发生异常]",
			zap.Int64("transferId", transfer.ID),
			zap.Any("req", req),
			zap.Error(err))
	}

	// 4. 通知转账结果
	if unifiedTransferResp != nil {
		s.NotifyTransfer(ctx, channel.ID, unifiedTransferResp)
	}

	return &respPay.PayTransferCreateResp{
		ID:                 transfer.ID,
		Status:             transfer.Status,
		ChannelPackageInfo: lo.If(unifiedTransferResp != nil, unifiedTransferResp.ChannelPackageInfo).Else(""),
	}, nil
}

// validateTransferCanCreate 校验转账单是否可以创建
// 对齐 Java: PayTransferServiceImpl.validateTransferCanCreate
func (s *PayTransferService) validateTransferCanCreate(ctx context.Context, req *reqPay.PayTransferCreateReq, appId int64) (*modelPay.PayTransfer, error) {
	transfer, err := s.transferRepo.SelectByAppIdAndMerchantTransferId(ctx, appId, req.MerchantTransferID)
	if err != nil {
		return nil, err
	}
	if transfer != nil {
		// 只有转账单状态为关闭，才能再次发起转账
		if transfer.Status != consts.PayTransferStatusClosed {
			return nil, errors.New("转账单已存在且状态不是关闭，无法重新发起")
		}
		// 校验参数是否一致
		if req.Price != transfer.Price {
			return nil, errors.New("转账金额不匹配")
		}
		if req.ChannelCode != transfer.ChannelCode {
			return nil, errors.New("转账渠道不匹配")
		}
	}
	// 如果状态为等待状态：不知道渠道转账是否发起成功
	// 特殊：允许使用相同的 no 再次发起转账，渠道会保证幂等
	return transfer, nil
}

// NotifyTransfer 转账回调通知
// 对齐 Java: PayTransferServiceImpl.notifyTransfer(Long channelId, PayTransferRespDTO notify)
func (s *PayTransferService) NotifyTransfer(ctx context.Context, channelId int64, notify *client.TransferResp) error {
	// 校验渠道是否有效
	channel, err := s.channelSvc.ValidPayChannel(ctx, channelId)
	if err != nil {
		return err
	}
	// 通知转账结果给对应的业务
	if notify == nil {
		return errors.New("转账通知缺失")
	}
	return repo.InTransaction(ctx, s.notifySvc.q, func(ctx context.Context, _ *query.Query) error { return s.notifyTransferInternal(ctx, channel, notify) })
}

// notifyTransferInternal 内部转账通知处理
// 对齐 Java: PayTransferServiceImpl.notifyTransfer(PayChannelDO channel, PayTransferRespDTO notify)
func (s *PayTransferService) notifyTransferInternal(ctx context.Context, channel *modelPay.PayChannel, notify *client.TransferResp) error {
	// 转账成功的回调
	if notify.Status == PayTransferStatusSuccess {
		return s.notifyTransferSuccess(ctx, channel, notify)
	}
	// 转账关闭的回调
	if notify.Status == PayTransferStatusClosed {
		return s.notifyTransferClosed(ctx, channel, notify)
	}
	// 转账处理中的回调
	if notify.Status == PayTransferStatusProcessing {
		return s.notifyTransferProgressing(ctx, channel, notify)
	}
	// WAITING 状态无需处理
	return nil
}

// notifyTransferProgressing 处理转账进行中的回调
// 对齐 Java: PayTransferServiceImpl.notifyTransferProgressing
func (s *PayTransferService) notifyTransferProgressing(ctx context.Context, channel *modelPay.PayChannel, notify *client.TransferResp) error {
	// 1. 校验
	transfer, err := s.transferRepo.SelectByAppIdAndNo(ctx, channel.AppID, notify.OutTradeNo)
	if err != nil {
		return err
	}
	if transfer == nil {
		return errors.New("转账单不存在")
	}
	if transfer.Status == PayTransferStatusProcessing {
		// 如果已经是转账中，直接返回，不用重复更新
		s.logger.Info("[notifyTransferProgressing][transfer已经是转账中状态，无需更新]", zap.Int64("transferId", transfer.ID))
		return nil
	}
	if transfer.Status != PayTransferStatusWaiting {
		return errors.New("转账单状态不是等待中，无法更新为进行中")
	}

	// 2. 更新状态
	updateCounts, err := s.transferRepo.UpdateByIdAndStatus(ctx, transfer.ID, []int{PayTransferStatusWaiting},
		&modelPay.PayTransfer{
			Status:             PayTransferStatusProcessing,
			ChannelPackageInfo: notify.ChannelPackageInfo,
		})
	if err != nil {
		return err
	}
	if updateCounts == 0 {
		return errors.New("转账单状态不是等待中，无法更新为进行中")
	}
	s.logger.Info("[notifyTransferProgressing][transfer更新为转账进行中状态]", zap.Int64("transferId", transfer.ID))
	return nil
}

// notifyTransferSuccess 处理转账成功的回调
// 对齐 Java: PayTransferServiceImpl.notifyTransferSuccess
func (s *PayTransferService) notifyTransferSuccess(ctx context.Context, channel *modelPay.PayChannel, notify *client.TransferResp) error {
	// 1. 校验状态
	transfer, err := s.transferRepo.SelectByAppIdAndNo(ctx, channel.AppID, notify.OutTradeNo)
	if err != nil {
		return err
	}
	if transfer == nil {
		return errors.New("转账单不存在")
	}
	if transfer.Status == PayTransferStatusSuccess {
		// 如果已成功，直接返回，不用重复更新
		s.logger.Info("[notifyTransferSuccess][transfer已经是成功状态，无需更新]", zap.Int64("transferId", transfer.ID))
		return nil
	}
	if transfer.Status != PayTransferStatusWaiting && transfer.Status != PayTransferStatusProcessing {
		return errors.New("转账单状态不是等待中或进行中，无法更新为成功")
	}

	// 2. 更新状态
	notifyDataJSON, _ := json.Marshal(notify)
	updateCounts, err := s.transferRepo.UpdateByIdAndStatus(ctx, transfer.ID,
		[]int{PayTransferStatusWaiting, PayTransferStatusProcessing},
		&modelPay.PayTransfer{
			Status:            PayTransferStatusSuccess,
			SuccessTime:       &notify.SuccessTime,
			ChannelTransferNo: notify.ChannelTransferNo,
			ChannelNotifyData: string(notifyDataJSON),
		})
	if err != nil {
		return err
	}
	if updateCounts == 0 {
		return errors.New("转账单状态不是等待中或进行中，无法更新为成功")
	}
	s.logger.Info("[notifyTransferSuccess][transfer更新为已转账]", zap.Int64("transferId", transfer.ID))

	// 3. 插入转账通知记录
	return s.notifySvc.CreatePayNotifyTask(ctx, PayNotifyTypeTransfer, transfer.ID)
}

// notifyTransferClosed 处理转账关闭的回调
// 对齐 Java: PayTransferServiceImpl.notifyTransferClosed
func (s *PayTransferService) notifyTransferClosed(ctx context.Context, channel *modelPay.PayChannel, notify *client.TransferResp) error {
	// 1. 校验状态
	transfer, err := s.transferRepo.SelectByAppIdAndNo(ctx, channel.AppID, notify.OutTradeNo)
	if err != nil {
		return err
	}
	if transfer == nil {
		return errors.New("转账单不存在")
	}
	if transfer.Status == PayTransferStatusClosed {
		// 如果已是关闭状态，直接返回，不用重复更新
		s.logger.Info("[notifyTransferClosed][transfer已经是关闭状态，无需更新]", zap.Int64("transferId", transfer.ID))
		return nil
	}
	if transfer.Status != PayTransferStatusWaiting && transfer.Status != PayTransferStatusProcessing {
		return errors.New("转账单状态不是等待中或进行中，无法更新为关闭")
	}

	// 2. 更新状态
	notifyDataJSON, _ := json.Marshal(notify)
	updateCount, err := s.transferRepo.UpdateByIdAndStatus(ctx, transfer.ID,
		[]int{PayTransferStatusWaiting, PayTransferStatusProcessing},
		&modelPay.PayTransfer{
			Status:            PayTransferStatusClosed,
			ChannelTransferNo: notify.ChannelTransferNo,
			ChannelNotifyData: string(notifyDataJSON),
			ChannelErrorCode:  notify.ChannelErrorCode,
			ChannelErrorMsg:   notify.ChannelErrorMsg,
		})
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return errors.New("转账单状态不是等待中或进行中，无法更新为关闭")
	}
	s.logger.Info("[notifyTransferClosed][transfer更新为关闭状态]", zap.Int64("transferId", transfer.ID))

	// 3. 插入转账通知记录
	return s.notifySvc.CreatePayNotifyTask(ctx, PayNotifyTypeTransfer, transfer.ID)
}

// SyncTransfer 同步转账单状态
// 对齐 Java: PayTransferServiceImpl.syncTransfer()
func (s *PayTransferService) SyncTransfer(ctx context.Context) (int, error) {
	list, err := s.transferRepo.SelectListByStatus(ctx, []int{PayTransferStatusWaiting, PayTransferStatusProcessing})
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, nil
	}

	count := 0
	for _, transfer := range list {
		if s.syncTransfer(ctx, transfer) {
			count++
		}
	}
	return count, nil
}

// SyncTransferById 同步指定转账单状态
// 对齐 Java: PayTransferServiceImpl.syncTransfer(Long id)
func (s *PayTransferService) SyncTransferById(ctx context.Context, id int64) error {
	transfer, err := s.transferRepo.SelectById(ctx, id)
	if err != nil {
		return err
	}
	if transfer == nil {
		return errors.New("转账单不存在")
	}
	s.syncTransfer(ctx, transfer)
	return nil
}

// syncTransfer 同步单个转账单
// 对齐 Java: PayTransferServiceImpl.syncTransfer(PayTransferDO transfer)
func (s *PayTransferService) syncTransfer(ctx context.Context, transfer *modelPay.PayTransfer) bool {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("[syncTransfer][transfer同步转账单状态异常]",
				zap.Int64("transferId", transfer.ID),
				zap.Any("error", r))
		}
	}()

	// 1. 查询转账订单信息
	payClient, err := s.channelSvc.GetPayClient(ctx, transfer.ChannelID)
	if err != nil {
		return false
	}
	if payClient == nil {
		s.logger.Error("[syncTransfer][渠道编号找不到对应的支付客户端]", zap.Int64("channelId", transfer.ChannelID))
		return false
	}
	resp, err := payClient.GetTransfer(ctx, transfer.No)
	if err != nil {
		s.logger.Error("[syncTransfer][查询转账订单失败]",
			zap.Int64("transferId", transfer.ID),
			zap.Error(err))
		return false
	}

	// 2. 回调转账结果
	if err := s.NotifyTransfer(ctx, transfer.ChannelID, resp); err != nil {
		s.logger.Error("[syncTransfer][回调转账结果失败]",
			zap.Int64("transferId", transfer.ID),
			zap.Error(err))
		return false
	}
	return true
}

// GetTransferByNo 根据转账单号获取转账单
func (s *PayTransferService) GetTransferByNo(ctx context.Context, no string) (*modelPay.PayTransfer, error) {
	return s.transferRepo.SelectByNo(ctx, no)
}
