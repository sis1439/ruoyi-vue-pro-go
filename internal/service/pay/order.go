package pay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	payrepo "github.com/wxlbd/ruoyi-mall-go/internal/repo/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"gorm.io/gorm"
)

type PayOrderService struct {
	q          *query.Query
	appSvc     *PayAppService
	channelSvc *PayChannelService
	clientFac  *client.PayClientFactory
	notifySvc  *PayNotifyService
	noDAO      *payrepo.PayNoRedisDAO
}

func NewPayOrderService(q *query.Query, appSvc *PayAppService, channelSvc *PayChannelService, clientFac *client.PayClientFactory, notifySvc *PayNotifyService, noDAO *payrepo.PayNoRedisDAO) *PayOrderService {
	return &PayOrderService{
		q:          q,
		appSvc:     appSvc,
		channelSvc: channelSvc,
		clientFac:  clientFac,
		notifySvc:  notifySvc,
		noDAO:      noDAO,
	}
}

func (s *PayOrderService) GetOrder(ctx context.Context, id int64) (*pay.PayOrder, error) {
	return s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(id)).First()
}

func (s *PayOrderService) GetOrderMap(ctx context.Context, ids []int64) (map[int64]*pay.PayOrder, error) {
	if len(ids) == 0 {
		return make(map[int64]*pay.PayOrder), nil
	}
	list, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*pay.PayOrder, len(list))
	for _, item := range list {
		res[item.ID] = item
	}
	return res, nil
}

// GetOrderPage 获得支付订单分页
func (s *PayOrderService) GetOrderPage(ctx context.Context, req *pay2.PayOrderPageReq) (*pagination.PageResult[*pay.PayOrder], error) {
	q := s.q.PayOrder.WithContext(ctx)
	if req.AppID > 0 {
		q = q.Where(s.q.PayOrder.AppID.Eq(req.AppID))
	}
	if req.ChannelCode != "" {
		q = q.Where(s.q.PayOrder.ChannelCode.Eq(req.ChannelCode))
	}
	if req.MerchantOrderId != "" {
		q = q.Where(s.q.PayOrder.MerchantOrderId.Eq(req.MerchantOrderId))
	}
	if req.Subject != "" {
		q = q.Where(s.q.PayOrder.Subject.Like("%" + req.Subject + "%"))
	}
	if req.No != "" {
		q = q.Where(s.q.PayOrder.No.Eq(req.No))
	}
	if req.Status != nil {
		q = q.Where(s.q.PayOrder.Status.Eq(*req.Status))
	}

	total, err := q.Count()
	if err != nil {
		return nil, err
	}
	list, err := q.Limit(req.GetLimit()).Offset(req.GetOffset()).Order(s.q.PayOrder.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*pay.PayOrder]{
		List:  list,
		Total: total,
	}, nil
}

// CreateOrder 创建支付单
func (s *PayOrderService) CreateOrder(ctx context.Context, reqDTO *pay2.PayOrderCreateReq) (int64, error) {
	q := repo.QueryFromContext(ctx, s.q)
	app, err := s.appSvc.ValidPayAppByAppKey(ctx, reqDTO.AppKey)
	if err != nil {
		return 0, err
	}

	existOrder, err := q.PayOrder.WithContext(ctx).
		Where(q.PayOrder.AppID.Eq(app.ID), q.PayOrder.MerchantOrderId.Eq(reqDTO.MerchantOrderId)).
		First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if existOrder != nil {
		return existOrder.ID, nil
	}

	// 创建支付交易单 (对齐 Java: 使用 app.OrderNotifyURL)
	order := &pay.PayOrder{
		AppID:           app.ID,
		MerchantOrderId: reqDTO.MerchantOrderId,
		Subject:         reqDTO.Subject,
		Body:            reqDTO.Body,
		NotifyURL:       app.OrderNotifyURL, // 对齐 Java: 使用 app 的回调地址
		Price:           reqDTO.Price,
		ExpireTime:      reqDTO.ExpireTime, // 对齐 Java: 使用传入的过期时间
		Status:          consts.PayOrderStatusWaiting,
		RefundPrice:     0,
		UserIP:          reqDTO.UserIP,
	}

	if err := q.PayOrder.WithContext(ctx).Create(order); err != nil {
		return 0, err
	}
	return order.ID, nil
}

// ... GetOrderCountByAppId ...

// SubmitOrder 提交支付订单
func (s *PayOrderService) SubmitOrder(ctx context.Context, reqVO *pay2.PayOrderSubmitReq, userIP string) (*pay2.PayOrderSubmitResp, error) {
	order, err := s.validateOrderCanSubmit(ctx, reqVO.ID)
	if err != nil {
		return nil, err
	}

	channel, err := s.validateChannelCanSubmit(ctx, order.AppID, reqVO.ChannelCode)
	if err != nil {
		return nil, err
	}

	// Generate No
	no, err := s.generateNo(ctx)
	if err != nil {
		return nil, err
	}

	// Create Extension
	ext := &pay.PayOrderExtension{
		OrderID:     order.ID,
		No:          no,
		ChannelID:   channel.ID,
		ChannelCode: channel.Code,
		UserIP:      userIP,
		Status:      PayOrderStatusWaiting,
	}
	if err := s.q.PayOrderExtension.WithContext(ctx).Create(ext); err != nil {
		return nil, err
	}

	// Get Pay Client
	payClient, err := NewPayChannelService(s.q, s.clientFac).GetPayClient(ctx, channel.ID)
	if err != nil {
		return nil, err
	}

	// Call UnifiedOrder (对齐 Java: 使用渠道特定的回调 URL)
	unifiedReq := &client.UnifiedOrderReq{
		UserIP:        userIP,
		OutTradeNo:    no,
		Subject:       order.Subject,
		Body:          order.Body,
		NotifyURL:     s.genChannelOrderNotifyUrl(channel), // 对齐 Java: 渠道回调 URL
		ReturnURL:     reqVO.ReturnUrl,
		Price:         order.Price,
		ExpireTime:    order.ExpireTime,
		DisplayMode:   reqVO.DisplayMode,
		ChannelExtras: reqVO.ChannelExtras, // JSAPI 依赖其中的 openid
	}
	unifiedResp, err := payClient.UnifiedOrder(ctx, unifiedReq)
	if err != nil {
		return nil, err
	}

	// ✅ 新增：处理直接支付成功的场景（对应 Java 163-180 行）
	if unifiedResp != nil {
		// Complete local notification before the request context is released; propagate failures.
		if err := s.NotifyOrder(ctx, channel.ID, unifiedResp); err != nil {
			return nil, err
		}

		// 7.2 检查渠道错误码并抛出异常
		if unifiedResp.ChannelErrorCode != "" {
			return nil, fmt.Errorf("支付渠道错误 [%s]: %s",
				unifiedResp.ChannelErrorCode, unifiedResp.ChannelErrorMsg)
		}

		// 7.3 重新查询最新订单状态（对齐 Java，确保返回给前端的是最新状态）
		order, _ = s.GetOrder(ctx, order.ID)
	}

	// Return response
	return &pay2.PayOrderSubmitResp{
		Status:         unifiedResp.Status,
		DisplayMode:    unifiedResp.DisplayMode,
		DisplayContent: unifiedResp.DisplayContent,
	}, nil
}

func (s *PayOrderService) validateOrderCanSubmit(ctx context.Context, id int64) (*pay.PayOrder, error) {
	order, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(id)).First()
	if err != nil {
		return nil, gorm.ErrRecordNotFound
	}
	if order.Status == PayOrderStatusSuccess {
		return nil, errors.New("order already paid")
	}
	if order.Status != PayOrderStatusWaiting {
		return nil, errors.New("order status not waiting")
	}
	if order.ExpireTime.Before(time.Now()) {
		return nil, errors.New("order expired")
	}
	return order, nil
}

// validateOrderActuallyPaid 校验订单是否真正已支付
// 对齐 Java: PayOrderService.validateOrderActuallyPaid(Long id)
// 这是一个防守性的方法，用于在支付回调丢失时通过主动查询渠道来补偿
func (s *PayOrderService) ValidateOrderActuallyPaid(ctx context.Context, orderID int64) (*pay.PayOrder, error) {
	// 1. 查询订单
	order, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(orderID)).First()
	if err != nil {
		return nil, err
	}

	// 2. 如果订单已支付或已关闭，直接返回
	if order.Status != PayOrderStatusWaiting {
		return order, nil
	}

	// 3. 查询订单的支付扩展信息（待支付的）
	ext, err := s.q.PayOrderExtension.WithContext(ctx).
		Where(s.q.PayOrderExtension.OrderID.Eq(orderID), s.q.PayOrderExtension.Status.Eq(PayOrderStatusWaiting)).
		First()
	if err != nil || ext == nil {
		// 没有待支付的拓展单，可能已经处理过了
		return order, nil
	}

	// 4. 获取支付客户端
	payClient, err := NewPayChannelService(s.q, s.clientFac).GetPayClient(ctx, ext.ChannelID)
	if err != nil {
		return nil, err
	}
	if payClient == nil {
		// 如果客户端不存在，则无法查询，直接返回
		return order, nil
	}

	// 5. 主动向渠道查询订单状态
	respDTO, err := payClient.GetOrder(ctx, ext.No)
	if err != nil {
		// 查询失败，记录日志但不中断，保持原状态
		return order, nil
	}

	// 6. 如果支付未成功，直接返回
	if respDTO.Status != PayOrderStatusSuccess {
		return order, nil
	}

	// 7. 支付成功，调用 NotifyOrder 来处理回调
	// 这会更新订单状态并记录支付通知
	_ = s.NotifyOrder(ctx, ext.ChannelID, respDTO)

	// 8. 重新查询并返回最新的订单信息
	updatedOrder, _ := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(orderID)).First()
	if updatedOrder != nil {
		return updatedOrder, nil
	}
	return order, nil
}

func (s *PayOrderService) validateChannelCanSubmit(ctx context.Context, appId int64, channelCode string) (*pay.PayChannel, error) {
	// app validation is implicit or done separately
	return s.channelSvc.GetChannelByAppIdAndCode(ctx, appId, channelCode)
}

// genChannelOrderNotifyUrl 根据支付渠道生成回调地址
// 对齐 Java: payProperties.getOrderNotifyUrl() + "/" + channel.getId()
func (s *PayOrderService) genChannelOrderNotifyUrl(channel *pay.PayChannel) string {
	return fmt.Sprintf("%s/%d", config.C.Pay.OrderNotifyURL, channel.ID)
}

func (s *PayOrderService) generateNo(ctx context.Context) (string, error) {
	return s.noDAO.Generate(ctx, "P")
}

// GetOrderExtension 获得支付订单拓展
func (s *PayOrderService) GetOrderExtension(ctx context.Context, id int64) (*pay.PayOrderExtension, error) {
	return s.q.PayOrderExtension.WithContext(ctx).Where(s.q.PayOrderExtension.ID.Eq(id)).First()
}

// ExpireOrder 清理过期的支付订单
// 对齐 Java: PayOrderService.expireOrder()
// 这是一个定时任务，应该每分钟或每5分钟执行一次
// 返回值：清理的过期订单数量
func (s *PayOrderService) ExpireOrder(ctx context.Context) (int64, error) {
	var expiredCount int64

	err := s.q.Transaction(func(tx *query.Query) error {
		now := time.Now()

		// 1. 查询所有待支付且已过期的订单
		expiredOrders, err := tx.PayOrder.WithContext(ctx).
			Where(
				tx.PayOrder.Status.Eq(PayOrderStatusWaiting),
				tx.PayOrder.ExpireTime.Lt(now),
			).
			Find()
		if err != nil {
			return err
		}

		if len(expiredOrders) == 0 {
			return nil
		}

		expiredCount = int64(len(expiredOrders))

		// 2. 批量更新订单状态为关闭
		orderIDs := make([]int64, 0, len(expiredOrders))
		for _, order := range expiredOrders {
			orderIDs = append(orderIDs, order.ID)
		}

		_, err = tx.PayOrder.WithContext(ctx).
			Where(tx.PayOrder.ID.In(orderIDs...)).
			Updates(map[string]interface{}{
				"status": PayOrderStatusClosed,
			})
		if err != nil {
			return err
		}

		// 3. 批量更新关联的拓展单为关闭（只更新还是待支付的）
		_, err = tx.PayOrderExtension.WithContext(ctx).
			Where(
				tx.PayOrderExtension.OrderID.In(orderIDs...),
				tx.PayOrderExtension.Status.Eq(PayOrderStatusWaiting),
			).
			Updates(map[string]interface{}{
				"status": PayOrderStatusClosed,
			})

		return err
	})

	return expiredCount, err
}

// syncOrderInterval 单个支付单主动查单的最小间隔
const syncOrderInterval = 3 * time.Second

// SyncOrderQuietlyThrottled 带频率闸门的主动查单，供 App 端 /pay/order/get?sync=true 使用，
// 避免前端轮询把渠道查单接口打爆。Redis 不可用时不放行。
func (s *PayOrderService) SyncOrderQuietlyThrottled(ctx context.Context, id int64) {
	ok, err := s.noDAO.AcquireSyncSlot(ctx, id, syncOrderInterval)
	if err != nil || !ok {
		return
	}
	s.SyncOrderQuietly(ctx, id)
}

// SyncOrderQuietly 同步订单的支付状态 (Quietly)
// 对齐 Java: PayOrderServiceImpl.syncOrderQuietly
func (s *PayOrderService) SyncOrderQuietly(ctx context.Context, id int64) {
	// 1. 查询待支付订单拓展
	extensions, err := s.q.PayOrderExtension.WithContext(ctx).
		Where(s.q.PayOrderExtension.OrderID.Eq(id), s.q.PayOrderExtension.Status.Eq(PayOrderStatusWaiting)).
		Find()
	if err != nil {
		return
	}

	// 2. 遍历执行同步
	for _, ext := range extensions {
		s.syncOrder(ctx, ext)
	}
}

// syncOrder 同步单个支付拓展单
// 对齐 Java: PayOrderServiceImpl.syncOrder(PayOrderExtensionDO)
func (s *PayOrderService) syncOrder(ctx context.Context, orderExtension *pay.PayOrderExtension) bool {
	// 1.1 查询支付订单信息
	payClient, err := NewPayChannelService(s.q, s.clientFac).GetPayClient(ctx, orderExtension.ChannelID)
	if err != nil {
		return false
	}
	if payClient == nil {
		return false
	}

	respDTO, err := payClient.GetOrder(ctx, orderExtension.No)
	if err != nil {
		return false
	}

	// 如果查询到订单不存在,PayClient 返回的状态为关闭。但此时不能关闭订单。
	// 存在以下场景:拉起渠道支付后,短时间内用户未及时完成支付,但是该订单同步定时任务恰巧自动触发了,
	// 主动查询结果为订单不存在。当用户支付成功之后,该订单状态在渠道的回调中无法从已关闭改为已支付,造成重大影响。
	// 考虑此定时任务是异常场景的兜底操作,因此这里不做变更,优先以回调为准。
	if respDTO.Status == PayOrderStatusClosed {
		return false
	}

	// 1.2 回调支付结果
	s.NotifyOrder(ctx, orderExtension.ChannelID, respDTO)

	// 2. 如果是已支付,则返回 true
	return respDTO.Status == PayOrderStatusSuccess
}

// NotifyOrder 通知并更新订单的支付结果（已包装事务）
// 对齐 Java: @Transactional 的 PayOrderService.notifyOrder(Long channelId, PayOrderRespDTO notify)
func (s *PayOrderService) NotifyOrder(ctx context.Context, channelID int64, notify *client.OrderResp) error {
	// 校验支付渠道是否有效
	channel, err := s.channelSvc.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}

	// 使用 GORM 事务包装（对齐 Java @Transactional）
	return s.q.Transaction(func(tx *query.Query) error {
		switch notify.Status {
		case PayOrderStatusSuccess:
			// 情况一: 支付成功的回调
			return s.notifyOrderSuccessTx(ctx, tx, channel, notify)
		case PayOrderStatusClosed:
			// 情况二: 支付失败的回调
			return s.notifyOrderClosedTx(ctx, tx, channel, notify)
		default:
			// 情况三: WAITING 无需处理
			// 情况四: REFUND 通过退款回调处理
			return nil
		}
	})
}

// notifyOrderSuccessTx 在事务内处理支付成功的回调
func (s *PayOrderService) notifyOrderSuccessTx(ctx context.Context, tx *query.Query, channel *pay.PayChannel, notify *client.OrderResp) error {
	// 1. 更新 PayOrderExtension 支付成功
	orderExtension, err := s.updateOrderExtensionSuccessTx(ctx, tx, notify)
	if err != nil {
		return err
	}

	// 2. 更新 PayOrder 支付成功
	paid, err := s.updateOrderSuccessTx(ctx, tx, channel, orderExtension, notify)
	if err != nil {
		return err
	}
	if paid {
		// 如果之前已经成功回调，则直接返回，不用重复记录支付通知记录
		return nil
	}

	// 3. 插入支付通知记录
	s.notifySvc.CreatePayNotifyTask(ctx, PayNotifyTypeOrder, orderExtension.OrderID)

	return nil
}

// updateOrderExtensionSuccessTx 在事务内更新 PayOrderExtension 支付成功
func (s *PayOrderService) updateOrderExtensionSuccessTx(ctx context.Context, tx *query.Query, notify *client.OrderResp) (*pay.PayOrderExtension, error) {
	// 1. 查询 PayOrderExtension
	orderExtension, err := tx.PayOrderExtension.WithContext(ctx).
		Where(tx.PayOrderExtension.No.Eq(notify.OutTradeNo)).
		First()
	if err != nil {
		return nil, fmt.Errorf("支付订单拓展不存在")
	}

	// 如果已经是成功，直接返回，不用重复更新
	if orderExtension.Status == PayOrderStatusSuccess {
		return orderExtension, nil
	}

	// 校验状态，必须是待支付
	if orderExtension.Status != PayOrderStatusWaiting {
		return nil, fmt.Errorf("支付订单拓展状态不是待支付")
	}

	// 2. 更新 PayOrderExtension (使用乐观锁)
	notifyDataJSON, _ := json.Marshal(notify)
	result, err := tx.PayOrderExtension.WithContext(ctx).
		Where(tx.PayOrderExtension.ID.Eq(orderExtension.ID), tx.PayOrderExtension.Status.Eq(PayOrderStatusWaiting)).
		Updates(map[string]interface{}{
			"status":              PayOrderStatusSuccess,
			"channel_notify_data": string(notifyDataJSON),
		})

	if err != nil || result.RowsAffected == 0 {
		return nil, fmt.Errorf("支付订单拓展状态不是待支付")
	}

	orderExtension.Status = PayOrderStatusSuccess
	return orderExtension, nil
}

// updateOrderSuccessTx 在事务内更新 PayOrder 支付成功
// 返回值: 是否之前已经成功回调
func (s *PayOrderService) updateOrderSuccessTx(ctx context.Context, tx *query.Query, channel *pay.PayChannel, orderExtension *pay.PayOrderExtension, notify *client.OrderResp) (bool, error) {
	// 1. 判断 PayOrder 是否处于待支付
	order, err := tx.PayOrder.WithContext(ctx).
		Where(tx.PayOrder.ID.Eq(orderExtension.OrderID)).
		First()
	if err != nil {
		return false, fmt.Errorf("支付订单不存在")
	}

	// 如果已经是成功，直接返回，不用重复更新
	if order.Status == PayOrderStatusSuccess && order.ExtensionID == orderExtension.ID {
		return true, nil
	}

	// 校验状态，必须是待支付
	if order.Status != PayOrderStatusWaiting {
		return false, fmt.Errorf("支付订单状态不是待支付")
	}

	// 1.1 校验渠道实收金额与支付单金额一致，避免改价回调把订单置为已支付
	if notify.Price > 0 && notify.Price != order.Price {
		return false, fmt.Errorf("支付金额不匹配: 渠道 %d, 订单 %d", notify.Price, order.Price)
	}

	// 2. 更新 PayOrder (使用乐观锁)
	channelFeePrice := int(float64(order.Price) * channel.FeeRate / 100.0)
	now := time.Now()

	result, err := tx.PayOrder.WithContext(ctx).
		Where(tx.PayOrder.ID.Eq(order.ID), tx.PayOrder.Status.Eq(PayOrderStatusWaiting)).
		Updates(map[string]interface{}{
			"status":            PayOrderStatusSuccess,
			"channel_id":        channel.ID,
			"channel_code":      channel.Code,
			"success_time":      &now,
			"extension_id":      orderExtension.ID,
			"no":                orderExtension.No,
			"channel_order_no":  notify.ChannelOrderNo,
			"channel_user_id":   notify.ChannelUserID,
			"channel_fee_rate":  channel.FeeRate,
			"channel_fee_price": channelFeePrice,
		})

	if err != nil || result.RowsAffected == 0 {
		return false, fmt.Errorf("支付订单状态不是待支付")
	}

	return false, nil
}

// notifyOrderClosedTx 在事务内处理支付关闭的回调
func (s *PayOrderService) notifyOrderClosedTx(ctx context.Context, tx *query.Query, channel *pay.PayChannel, notify *client.OrderResp) error {
	// 查询 PayOrderExtension
	orderExtension, err := tx.PayOrderExtension.WithContext(ctx).
		Where(tx.PayOrderExtension.No.Eq(notify.OutTradeNo)).
		First()
	if err != nil {
		return fmt.Errorf("支付订单拓展不存在")
	}

	// 如果已经是关闭，直接返回
	if orderExtension.Status == PayOrderStatusClosed {
		return nil
	}

	// 如果已经是成功，不更新为关闭（避免状态冲突）
	if orderExtension.Status == PayOrderStatusSuccess {
		return nil
	}

	// 校验状态，必须是待支付
	if orderExtension.Status != PayOrderStatusWaiting {
		return fmt.Errorf("支付订单拓展状态不是待支付")
	}

	// 更新拓展单为关闭
	result, err := tx.PayOrderExtension.WithContext(ctx).
		Where(tx.PayOrderExtension.ID.Eq(orderExtension.ID), tx.PayOrderExtension.Status.Eq(PayOrderStatusWaiting)).
		Updates(map[string]interface{}{
			"status":             PayOrderStatusClosed,
			"channel_error_code": notify.ChannelErrorCode,
			"channel_error_msg":  notify.ChannelErrorMsg,
		})
	if err != nil || result.RowsAffected == 0 {
		return fmt.Errorf("支付订单拓展状态已改变")
	}

	// 更新主订单为关闭
	order, err := tx.PayOrder.WithContext(ctx).
		Where(tx.PayOrder.ID.Eq(orderExtension.OrderID)).
		First()
	if err != nil {
		return fmt.Errorf("支付订单不存在")
	}

	if order.Status != PayOrderStatusWaiting {
		return nil // 订单已处理过，不需要重复关闭
	}

	result, err = tx.PayOrder.WithContext(ctx).
		Where(tx.PayOrder.ID.Eq(order.ID), tx.PayOrder.Status.Eq(PayOrderStatusWaiting)).
		Updates(map[string]interface{}{
			"status": PayOrderStatusClosed,
		})
	if err != nil || result.RowsAffected == 0 {
		return fmt.Errorf("支付订单状态已改变")
	}

	return nil
}

// UpdatePayOrderPrice 更新支付订单价格
// 对齐 Java: PayOrderService.updatePayOrderPrice(Long id, Integer payPrice)
func (s *PayOrderService) UpdatePayOrderPrice(ctx context.Context, id int64, payPrice int) error {
	// 1. 查询支付订单
	order, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(id)).First()
	if err != nil {
		return fmt.Errorf("支付订单不存在")
	}

	// 2. 校验状态：必须是待支付
	if order.Status != PayOrderStatusWaiting {
		return fmt.Errorf("支付订单状态不是待支付，无法修改价格")
	}

	// 3. 如果价格相同，直接返回
	if order.Price == payPrice {
		return nil
	}

	// 4. 更新订单价格
	_, err = s.q.PayOrder.WithContext(ctx).
		Where(s.q.PayOrder.ID.Eq(id)).
		Update(s.q.PayOrder.Price, payPrice)

	return err
}

// UpdateOrderRefundPrice 更新订单退款金额
// 对齐 Java: PayOrderService.updateOrderRefundPrice(Long id, Integer incrRefundPrice)
func (s *PayOrderService) UpdateOrderRefundPrice(ctx context.Context, id int64, incrRefundPrice int) error {
	order, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(id)).First()
	if err != nil {
		return fmt.Errorf("支付订单不存在")
	}

	// 校验状态：必须是已支付或已退款
	if order.Status != PayOrderStatusSuccess && order.Status != PayOrderStatusRefund {
		return fmt.Errorf("支付订单状态不是已支付或已退款")
	}

	// 校验退款金额不能超过支付金额
	if order.RefundPrice+incrRefundPrice > order.Price {
		return fmt.Errorf("退款金额超过支付金额")
	}

	// 更新订单 (使用乐观锁)
	result, err := s.q.PayOrder.WithContext(ctx).
		Where(s.q.PayOrder.ID.Eq(id), s.q.PayOrder.Status.Eq(order.Status)).
		Updates(map[string]interface{}{
			"refund_price": order.RefundPrice + incrRefundPrice,
			"status":       PayOrderStatusRefund,
		})

	if err != nil || result.RowsAffected == 0 {
		return fmt.Errorf("支付订单状态不是已支付或已退款")
	}

	return nil
}

// GetOrderList 获得支付订单列表 (Export)
func (s *PayOrderService) GetOrderList(ctx context.Context, req *pay2.PayOrderExportReq) ([]*pay.PayOrder, error) {
	q := s.q.PayOrder.WithContext(ctx)
	if req.AppID > 0 {
		q = q.Where(s.q.PayOrder.AppID.Eq(req.AppID))
	}
	if req.ChannelCode != "" {
		q = q.Where(s.q.PayOrder.ChannelCode.Eq(req.ChannelCode))
	}
	if req.MerchantOrderId != "" {
		q = q.Where(s.q.PayOrder.MerchantOrderId.Eq(req.MerchantOrderId))
	}
	if req.Subject != "" {
		q = q.Where(s.q.PayOrder.Subject.Like("%" + req.Subject + "%"))
	}
	if req.No != "" {
		q = q.Where(s.q.PayOrder.No.Eq(req.No))
	}
	if req.Status != nil {
		q = q.Where(s.q.PayOrder.Status.Eq(*req.Status))
	}
	return q.Order(s.q.PayOrder.ID.Desc()).Find()
}

// SyncOrder 同步指定时间之后的待支付订单状态
// 对齐 Java: PayOrderServiceImpl.syncOrder(LocalDateTime minCreateTime)
func (s *PayOrderService) SyncOrder(ctx context.Context, minCreateTime time.Time) (int, error) {
	// 1. 查询指定创建时间内的待支付订单拓展
	extensions, err := s.q.PayOrderExtension.WithContext(ctx).
		Where(
			s.q.PayOrderExtension.Status.Eq(PayOrderStatusWaiting),
			s.q.PayOrderExtension.CreateTime.Gte(minCreateTime),
		).
		Find()
	if err != nil {
		return 0, err
	}
	if len(extensions) == 0 {
		return 0, nil
	}

	// 2. 遍历执行
	count := 0
	for _, ext := range extensions {
		if s.syncOrder(ctx, ext) {
			count++
		}
	}
	return count, nil
}

// ValidateMemberOrderOwner resolves the owner from the business record, not a client-supplied user ID.
func (s *PayOrderService) ValidateMemberOrderOwner(ctx context.Context, payOrderID int64) error {
	user := pkgContext.GetLoginUserFromContext(ctx)
	if user == nil || user.UserType != consts.UserTypeMember || user.UserID <= 0 || payOrderID <= 0 {
		return pkgErrors.NewBizError(403, "无权访问支付订单")
	}
	trade := s.q.TradeOrder
	count, err := trade.WithContext(ctx).Where(trade.PayOrderID.Eq(payOrderID), trade.UserID.Eq(user.UserID)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	recharge := s.q.PayWalletRecharge
	record, err := recharge.WithContext(ctx).Where(recharge.PayOrderID.Eq(payOrderID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return pkgErrors.NewBizError(403, "无权访问支付订单")
	}
	if err != nil {
		return err
	}
	wallet := s.q.PayWallet
	count, err = wallet.WithContext(ctx).Where(wallet.ID.Eq(record.WalletID), wallet.UserID.Eq(user.UserID), wallet.UserType.Eq(user.UserType)).Count()
	if err != nil {
		return err
	}
	if count != 1 {
		return pkgErrors.NewBizError(403, "无权访问支付订单")
	}
	return nil
}
