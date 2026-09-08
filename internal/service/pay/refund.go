package pay

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	payrepo "github.com/wxlbd/ruoyi-mall-go/internal/repo/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PayRefundService struct {
	q          *query.Query
	appSvc     *PayAppService
	channelSvc *PayChannelService
	orderSvc   *PayOrderService
	notifySvc  *PayNotifyService
	noDAO      *payrepo.PayNoRedisDAO
}

func NewPayRefundService(q *query.Query, appSvc *PayAppService, channelSvc *PayChannelService, orderSvc *PayOrderService, notifySvc *PayNotifyService, noDAO *payrepo.PayNoRedisDAO) *PayRefundService {
	return &PayRefundService{
		q:          q,
		appSvc:     appSvc,
		channelSvc: channelSvc,
		orderSvc:   orderSvc,
		notifySvc:  notifySvc,
		noDAO:      noDAO,
	}
}

// CreateRefund 创建退款单
func (s *PayRefundService) CreateRefund(ctx context.Context, reqDTO *pay.PayRefundCreateReq) (int64, error) {
	// 1.1 校验 App
	app, err := s.appSvc.GetAppByAppKey(ctx, reqDTO.AppKey)
	if err != nil {
		return 0, err
	}
	if app == nil {
		return 0, errors.NewBizError(1006000000, "支付应用不存在") // PAY_APP_NOT_FOUND
	}
	app, err = s.appSvc.ValidPayApp(ctx, app.ID)
	if err != nil {
		return 0, err
	}

	// 1.2 校验支付订单
	payOrder, err := s.validatePayOrderCanRefund(ctx, app.ID, reqDTO)
	if err != nil {
		return 0, err
	}

	// 1.3 校验支付渠道是否有效
	channel, err := s.channelSvc.ValidPayChannel(ctx, payOrder.ChannelID)
	if err != nil {
		return 0, err
	}
	payClient, err := s.channelSvc.GetPayClient(ctx, channel.ID)
	if err != nil {
		return 0, err
	}
	if payClient == nil {
		return 0, errors.NewBizError(1006002000, "支付渠道找不到对应的支付客户端") // PAY_CHANNEL_CLIENT_NOT_FOUND
	}

	// 1.4 校验退款订单是否已存在
	if err := s.validatePayRefundExist(ctx, app.ID, reqDTO.MerchantRefundId); err != nil {
		return 0, err
	}

	// 2.1 创建退款单
	// Generate Refund No (R + time + 6 digits)
	no, err := s.noDAO.Generate(ctx, "R")
	if err != nil {
		return 0, fmt.Errorf("failed to generate refund no: %w", err)
	}

	refund := &payModel.PayRefund{
		No:               no,
		AppID:            app.ID,
		OrderID:          payOrder.ID,
		OrderNo:          payOrder.No,
		MerchantOrderId:  reqDTO.MerchantOrderId,
		MerchantRefundId: reqDTO.MerchantRefundId,
		NotifyURL:        app.RefundNotifyURL,
		Status:           consts.PayRefundStatusWaiting,
		PayPrice:         payOrder.Price,
		RefundPrice:      reqDTO.Price,
		Reason:           reqDTO.Reason,
		UserIP:           reqDTO.UserIP,
		ChannelID:        payOrder.ChannelID,
		ChannelCode:      payOrder.ChannelCode,
		ChannelOrderNo:   payOrder.ChannelOrderNo,
	}
	if err := repo.InTransaction(ctx, s.q, func(ctx context.Context, tx *query.Query) error {
		o := tx.PayOrder
		if _, err := o.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where(o.ID.Eq(payOrder.ID)).First(); err != nil {
			return err
		}
		if _, err := s.validatePayOrderCanRefund(ctx, app.ID, reqDTO); err != nil {
			return err
		}
		if err := s.validatePayRefundExist(ctx, app.ID, reqDTO.MerchantRefundId); err != nil {
			return err
		}
		return tx.PayRefund.WithContext(ctx).Create(refund)
	}); err != nil {
		return 0, err
	}

	// 2.2 向渠道发起退款申请
	unifiedReqDTO := &client.UnifiedRefundReq{
		OutTradeNo:  payOrder.No,
		OutRefundNo: refund.No,
		Reason:      reqDTO.Reason,
		PayPrice:    payOrder.Price,
		RefundPrice: reqDTO.Price,
		NotifyURL:   s.genChannelRefundNotifyUrl(channel),
	}
	refundRespDTO, err := payClient.UnifiedRefund(ctx, unifiedReqDTO)
	if err != nil {
		// 注意：这里仅打印异常，不进行抛出。
		// 原因是：虽然调用支付渠道进行退款发生异常（网络请求超时），实际退款成功。这个结果，后续通过退款回调、或者退款轮询补偿可以拿到。
		fmt.Printf("[createPayRefund][退款 id(%d) requestDTO(%+v) 发生异常: %v]\n", refund.ID, reqDTO, err)
	} else {
		// 2.3 处理退款返回
		if err := s.NotifyRefund(ctx, channel.ID, refundRespDTO); err != nil {
			return refund.ID, err
		}
	}

	return refund.ID, nil
}

func (s *PayRefundService) validatePayOrderCanRefund(ctx context.Context, appId int64, reqDTO *pay.PayRefundCreateReq) (*payModel.PayOrder, error) {
	q := repo.QueryFromContext(ctx, s.q)
	// Query PayOrder
	payOrder, err := q.PayOrder.WithContext(ctx).
		Where(q.PayOrder.AppID.Eq(appId), q.PayOrder.MerchantOrderId.Eq(reqDTO.MerchantOrderId)).
		First()
	if err != nil {
		if stdErrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewBizError(1006001000, "支付订单不存在") // PAY_ORDER_NOT_FOUND
		}
		return nil, err
	}

	// Check Status
	if payOrder.Status != consts.PayOrderStatusSuccess && payOrder.Status != PayOrderStatusRefund {
		return nil, errors.NewBizError(1006001001, "支付订单状态不对") // PAY_ORDER_STATUS_IS_NOT_SUCCESS
	}

	// Check Refund Price：金额必须为正的整数分，且累计退款不得超过实付
	if reqDTO.Price <= 0 {
		return nil, errors.NewBizError(1006010003, "退款金额必须大于 0")
	}
	if reqDTO.Price > payOrder.Price-payOrder.RefundPrice {
		return nil, errors.NewBizError(1006010003, "退款金额超过支付金额") // PAY_REFUND_PRICE_EXCEED
	}

	// 是否有退款中的订单
	if err := s.validateNoRefundingOrder(ctx, appId, payOrder.ID); err != nil {
		return nil, err
	}

	return payOrder, nil
}

func (s *PayRefundService) validateNoRefundingOrder(ctx context.Context, appId int64, orderId int64) error {
	q := repo.QueryFromContext(ctx, s.q)
	count, err := q.PayRefund.WithContext(ctx).
		Where(q.PayRefund.AppID.Eq(appId), q.PayRefund.OrderID.Eq(orderId),
			q.PayRefund.Status.Eq(consts.PayRefundStatusWaiting)).
		Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.NewBizError(1006010005, "该订单还有退款等待处理") // REFUND_HAS_REFUNDING
	}
	return nil
}

func (s *PayRefundService) genChannelRefundNotifyUrl(channel *payModel.PayChannel) string {
	// 对齐 Java: payProperties.getRefundNotifyUrl() + "/" + channel.getId()
	return fmt.Sprintf("%s/%d", config.C.Pay.RefundNotifyURL, channel.ID)
}

func (s *PayRefundService) validatePayRefundExist(ctx context.Context, appId int64, merchantRefundId string) error {
	q := repo.QueryFromContext(ctx, s.q)
	count, err := q.PayRefund.WithContext(ctx).
		Where(q.PayRefund.AppID.Eq(appId), q.PayRefund.MerchantRefundId.Eq(merchantRefundId)).
		Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.NewBizError(1006010002, "退款订单已存在") // PAY_REFUND_EXISTS
	}
	return nil
}

// GetRefund 获得退款订单
func (s *PayRefundService) GetRefund(ctx context.Context, id int64) (*payModel.PayRefund, error) {
	return s.q.PayRefund.WithContext(ctx).Where(s.q.PayRefund.ID.Eq(id)).First()
}

func (s *PayRefundService) GetRefundByNo(ctx context.Context, no string) (*payModel.PayRefund, error) {
	return s.q.PayRefund.WithContext(ctx).Where(s.q.PayRefund.No.Eq(no)).First()
}

func (s *PayRefundService) GetRefundCountByAppId(ctx context.Context, appId int64) (int64, error) {
	return s.q.PayRefund.WithContext(ctx).Where(s.q.PayRefund.AppID.Eq(appId)).Count()
}

// GetRefundPage 获得退款订单分页
func (s *PayRefundService) GetRefundPage(ctx context.Context, req *pay.PayRefundPageReq) (*pagination.PageResult[*payModel.PayRefund], error) {
	q := s.q.PayRefund.WithContext(ctx)
	if req.AppID > 0 {
		q = q.Where(s.q.PayRefund.AppID.Eq(req.AppID))
	}
	if req.ChannelCode != "" {
		q = q.Where(s.q.PayRefund.ChannelCode.Eq(req.ChannelCode))
	}
	if req.MerchantOrderId != "" {
		q = q.Where(s.q.PayRefund.MerchantOrderId.Eq(req.MerchantOrderId))
	}
	if req.MerchantRefundId != "" {
		q = q.Where(s.q.PayRefund.MerchantRefundId.Eq(req.MerchantRefundId))
	}
	if req.ChannelOrderNo != "" {
		q = q.Where(s.q.PayRefund.ChannelOrderNo.Eq(req.ChannelOrderNo))
	}
	if req.ChannelRefundNo != "" {
		q = q.Where(s.q.PayRefund.ChannelRefundNo.Eq(req.ChannelRefundNo))
	}
	if req.Status != nil {
		q = q.Where(s.q.PayRefund.Status.Eq(*req.Status))
	}

	total, err := q.Count()
	if err != nil {
		return nil, err
	}
	list, err := q.Limit(req.GetLimit()).Offset(req.GetOffset()).Order(s.q.PayRefund.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*payModel.PayRefund]{
		List:  list,
		Total: total,
	}, nil
}

// GetRefundList 获得退款订单列表 (Export)
func (s *PayRefundService) GetRefundList(ctx context.Context, req *pay.PayRefundExportReq) ([]*payModel.PayRefund, error) {
	q := s.q.PayRefund.WithContext(ctx)
	if req.AppID > 0 {
		q = q.Where(s.q.PayRefund.AppID.Eq(req.AppID))
	}
	if req.ChannelCode != "" {
		q = q.Where(s.q.PayRefund.ChannelCode.Eq(req.ChannelCode))
	}
	if req.MerchantOrderId != "" {
		q = q.Where(s.q.PayRefund.MerchantOrderId.Eq(req.MerchantOrderId))
	}
	if req.MerchantRefundId != "" {
		q = q.Where(s.q.PayRefund.MerchantRefundId.Eq(req.MerchantRefundId))
	}
	if req.ChannelOrderNo != "" {
		q = q.Where(s.q.PayRefund.ChannelOrderNo.Eq(req.ChannelOrderNo))
	}
	if req.ChannelRefundNo != "" {
		q = q.Where(s.q.PayRefund.ChannelRefundNo.Eq(req.ChannelRefundNo))
	}
	if req.Status != nil {
		q = q.Where(s.q.PayRefund.Status.Eq(*req.Status))
	}
	return q.Order(s.q.PayRefund.ID.Desc()).Find()
}

// NotifyRefund 处理退款回调通知
// 对齐 Java: PayRefundService.notifyRefund(Long channelId, PayRefundRespDTO notify)
func (s *PayRefundService) NotifyRefund(ctx context.Context, channelID int64, notify *client.RefundResp) error {
	if notify == nil || notify.OutRefundNo == "" {
		return fmt.Errorf("退款通知缺少退款号")
	}
	// 校验支付渠道是否有效
	channel, err := s.channelSvc.ValidPayChannel(ctx, channelID)
	if err != nil {
		return err
	}

	// 使用事务包装（对齐 Java @Transactional）
	return repo.InTransaction(ctx, s.q, func(ctx context.Context, tx *query.Query) error {
		// 情况一：退款成功
		if notify.Status == consts.PayRefundStatusSuccess {
			return s.notifyRefundSuccessTx(ctx, tx, channel, notify)
		}

		// 情况二：退款失败
		if notify.Status == consts.PayRefundStatusFailure {
			return s.notifyRefundFailureTx(ctx, tx, channel, notify)
		}

		return nil
	})
}

// notifyRefundSuccessTx 在事务内处理退款成功
func (s *PayRefundService) notifyRefundSuccessTx(ctx context.Context, tx *query.Query, channel *payModel.PayChannel, notify *client.RefundResp) error {
	// 1.1 查询 PayRefund
	refund, err := tx.PayRefund.WithContext(ctx).
		Where(tx.PayRefund.AppID.Eq(channel.AppID), tx.PayRefund.No.Eq(notify.OutRefundNo)).
		First()
	if err != nil {
		return fmt.Errorf("退款订单不存在")
	}

	if refund.ChannelID != channel.ID || notify.OutTradeNo != refund.OrderNo {
		return fmt.Errorf("退款支付单或渠道不匹配")
	}
	// 如果已经是成功，直接返回
	if refund.Status == consts.PayRefundStatusSuccess {
		return nil
	}

	// 校验状态，必须是等待状态
	if refund.Status != consts.PayRefundStatusWaiting {
		return fmt.Errorf("退款订单状态不是待退款")
	}

	// 1.2 更新 PayRefund (使用乐观锁)
	notifyDataJSON, _ := json.Marshal(notify)
	result, err := tx.PayRefund.WithContext(ctx).
		Where(tx.PayRefund.ID.Eq(refund.ID), tx.PayRefund.Status.Eq(consts.PayRefundStatusWaiting)).
		Updates(map[string]interface{}{
			"status":              consts.PayRefundStatusSuccess,
			"success_time":        notify.SuccessTime,
			"channel_refund_no":   notify.ChannelRefundNo,
			"channel_notify_data": string(notifyDataJSON),
		})

	if err != nil || result.RowsAffected == 0 {
		return fmt.Errorf("退款订单状态不是待退款")
	}

	// 2. 更新订单退款金额
	if err := s.orderSvc.UpdateOrderRefundPrice(ctx, refund.OrderID, refund.RefundPrice); err != nil {
		return err
	}

	// 3. 插入退款通知记录
	return s.notifySvc.CreatePayNotifyTask(ctx, PayNotifyTypeRefund, refund.ID)
}

// notifyRefundFailureTx 在事务内处理退款失败
func (s *PayRefundService) notifyRefundFailureTx(ctx context.Context, tx *query.Query, channel *payModel.PayChannel, notify *client.RefundResp) error {
	// 1.1 查询 PayRefund
	refund, err := tx.PayRefund.WithContext(ctx).
		Where(tx.PayRefund.AppID.Eq(channel.AppID), tx.PayRefund.No.Eq(notify.OutRefundNo)).
		First()
	if err != nil {
		return fmt.Errorf("退款订单不存在")
	}

	if refund.ChannelID != channel.ID || notify.OutTradeNo != refund.OrderNo {
		return fmt.Errorf("退款支付单或渠道不匹配")
	}
	// 如果已经是失败，直接返回
	if refund.Status == consts.PayRefundStatusFailure {
		return nil
	}

	// 校验状态，必须是等待状态
	if refund.Status != consts.PayRefundStatusWaiting {
		return fmt.Errorf("退款订单状态不是待退款")
	}

	// 1.2 更新 PayRefund (使用乐观锁)
	notifyDataJSON, _ := json.Marshal(notify)
	result, err := tx.PayRefund.WithContext(ctx).
		Where(tx.PayRefund.ID.Eq(refund.ID), tx.PayRefund.Status.Eq(consts.PayRefundStatusWaiting)).
		Updates(map[string]interface{}{
			"status":              consts.PayRefundStatusFailure,
			"channel_refund_no":   notify.ChannelRefundNo,
			"channel_notify_data": string(notifyDataJSON),
			"channel_error_code":  notify.ChannelErrorCode,
			"channel_error_msg":   notify.ChannelErrorMsg,
		})

	if err != nil || result.RowsAffected == 0 {
		return fmt.Errorf("退款订单状态不是待退款")
	}

	// 2. 插入退款通知记录
	return s.notifySvc.CreatePayNotifyTask(ctx, PayNotifyTypeRefund, refund.ID)
}

// SyncRefund 同步渠道退款的退款状态
func (s *PayRefundService) SyncRefund(ctx context.Context) (int, error) {
	// 1. 查询指定创建时间内的待退款订单
	refunds, err := s.q.PayRefund.WithContext(ctx).
		Where(s.q.PayRefund.Status.Eq(consts.PayRefundStatusWaiting)).
		Find()
	if err != nil {
		return 0, err
	}
	if len(refunds) == 0 {
		return 0, nil
	}

	// 2. 遍历执行
	count := 0
	for _, refund := range refunds {
		synced, err := s.syncRefund(ctx, refund)
		if err != nil {
			fmt.Printf("[SyncRefund][退款订单(%d) 同步失败: %v]\n", refund.ID, err)
			continue
		}
		if synced {
			count++
		}
	}
	return count, nil
}

func (s *PayRefundService) syncRefund(ctx context.Context, refund *payModel.PayRefund) (bool, error) {
	// 1.1 查询退款订单信息
	payClient, err := s.channelSvc.GetPayClient(ctx, refund.ChannelID)
	if err != nil {
		return false, err
	}
	if payClient == nil {
		return false, fmt.Errorf("渠道编号(%d) 找不到对应的支付客户端", refund.ChannelID)
	}

	respDTO, err := payClient.GetRefund(ctx, refund.OrderNo, refund.No)
	if err != nil {
		return false, err
	}

	// 1.2 回调退款结果
	if err := s.NotifyRefund(ctx, refund.ChannelID, respDTO); err != nil {
		return false, err
	}

	// 2. 如果同步到，则返回 true
	return respDTO.Status == consts.PayRefundStatusSuccess || respDTO.Status == consts.PayRefundStatusFailure, nil
}
