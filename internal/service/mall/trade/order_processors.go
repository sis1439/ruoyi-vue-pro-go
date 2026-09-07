package trade

import (
	"context"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"strconv"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"go.uber.org/zap"
)

// CreateOrderProcessor 创建订单业务处理器
type CreateOrderProcessor struct {
	*BaseOrderHandler
	q         *query.Query
	skuSvc    ProductSkuServiceAPI
	couponSvc CouponUserServiceAPI
	memberSvc MemberUserServiceAPI
	logger    *zap.Logger
}

// NewCreateOrderProcessor 创建订单处理器构造函数
func NewCreateOrderProcessor(
	q *query.Query,
	skuSvc ProductSkuServiceAPI,
	couponSvc CouponUserServiceAPI,
	memberSvc MemberUserServiceAPI,
	logger *zap.Logger,
) *CreateOrderProcessor {
	return &CreateOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("create", []string{"create"}),
		q:                q,
		skuSvc:           skuSvc,
		couponSvc:        couponSvc,
		memberSvc:        memberSvc,
		logger:           logger,
	}
}

// Handle 处理创建订单操作
// 注意：实际的订单创建逻辑在 TradeOrderUpdateService.CreateOrder 中实现
// 这个处理器主要用于订单创建的前置和后置钩子
func (p *CreateOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理创建订单请求",
		zap.Int64("userId", handleReq.UserID),
		zap.Any("createReq", handleReq.CreateReq),
	)

	// 订单创建的主逻辑在 TradeOrderUpdateService.CreateOrder 中
	// 这里返回成功，表示处理器链可以继续
	return &OrderHandleResponse{
		Success: true,
		Message: "订单创建处理器执行成功",
	}, nil
}

func (p *CreateOrderProcessor) AfterOrderCreate(ctx context.Context, handleReq *OrderHandleRequest, resp *OrderHandleResponse) error {
	order := resp.Order
	orderItems := handleReq.OrderItems

	// 1. 扣减库存
	if len(orderItems) > 0 {
		stockItems := make([]product.ProductSkuUpdateStockItemReq, len(orderItems))
		for i, item := range orderItems {
			stockItems[i] = product.ProductSkuUpdateStockItemReq{
				ID:        item.SkuID,
				IncrCount: -int(item.Count), // 负数表示扣减
			}
		}
		if err := p.skuSvc.UpdateSkuStock(ctx, &product.ProductSkuUpdateStockReq{Items: stockItems}); err != nil {
			p.logger.Error("扣减库存失败", zap.Error(err), zap.Int64("orderId", order.ID))
			return err
		}
	}

	// 2. 使用优惠券
	if order.CouponID > 0 {
		if err := p.couponSvc.UseCoupon(ctx, order.UserID, order.CouponID, order.ID); err != nil {
			p.logger.Error("使用优惠券失败", zap.Error(err), zap.Int64("orderId", order.ID), zap.Int64("couponId", order.CouponID))
			return err
		}
	}

	// 3. 扣减积分
	if order.UsePoint > 0 {
		if err := member.NewMemberPointRecordService(p.q, nil).CreatePointRecord(ctx, order.UserID, -order.UsePoint, tradeModel.MemberPointBizTypeOrderUse, strconv.FormatInt(order.ID, 10)); err != nil {
			p.logger.Error("扣减积分失败", zap.Int64("orderId", order.ID), zap.Int("usePoint", order.UsePoint))
			return err
		}
	}

	return nil
}

// PayOrderProcessor 支付订单业务处理器
type PayOrderProcessor struct {
	*BaseOrderHandler
	q      *query.Query
	paySvc PayOrderServiceAPI
	logger *zap.Logger
}

// NewPayOrderProcessor 支付订单处理器构造函数
func NewPayOrderProcessor(q *query.Query, paySvc PayOrderServiceAPI, logger *zap.Logger) *PayOrderProcessor {
	return &PayOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("pay", []string{"pay"}),
		q:                q,
		paySvc:           paySvc,
		logger:           logger,
	}
}

// Handle 处理支付订单操作
func (p *PayOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理订单支付请求",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("payOrderId", handleReq.PayOrderID),
	)

	// 1. 查询订单信息
	order, err := p.q.TradeOrder.WithContext(ctx).
		Where(p.q.TradeOrder.ID.Eq(handleReq.OrderID)).
		First()
	if err != nil {
		return nil, ErrOrderNotExists()
	}

	// 2. 校验支付参数
	if handleReq.PayOrderID == 0 {
		return nil, fmt.Errorf("支付单编号不能为空")
	}

	// 3. 校验订单状态
	// 对应 Java: if (!TradeOrderStatusEnum.isUnpaid(order.getStatus()) || order.getPayStatus())
	if order.Status != tradeModel.TradeOrderStatusUnpaid || order.PayStatus {
		// 特殊：支付单号相同，直接返回，说明重复回调（幂等处理）
		// 对应 Java: if (ObjectUtil.equals(order.getPayOrderId(), payOrderId))
		if order.PayOrderID != nil && *order.PayOrderID == handleReq.PayOrderID {
			p.logger.Warn("订单已支付，且支付单号相同，直接返回",
				zap.Int64("orderId", order.ID),
				zap.Int64("payOrderId", handleReq.PayOrderID),
			)
			return &OrderHandleResponse{Order: order, Success: true, Message: "订单已支付"}, nil
		}

		p.logger.Error("订单支付失败：订单状态不正确",
			zap.Int64("orderId", order.ID),
			zap.Int64("reqPayOrderId", handleReq.PayOrderID),
			zap.Int64p("orderPayOrderId", order.PayOrderID),
			zap.Int("status", order.Status),
		)
		return nil, fmt.Errorf("订单不处于待支付状态")
	}

	// 4. 校验支付单合法性 (对应 Java: validatePayOrderPaid)
	// 4.1 查询支付单
	payOrder, err := p.paySvc.GetOrder(ctx, handleReq.PayOrderID)
	if err != nil {
		p.logger.Error("获取支付单失败", zap.Error(err))
		return nil, ErrOrderNotExists()
	}
	if payOrder == nil {
		p.logger.Error("支付单不存在", zap.Int64("payOrderId", handleReq.PayOrderID))
		return nil, ErrOrderNotExists()
	}

	// 4.2 校验支付单状态
	if payOrder.Status != pay.PayOrderStatusSuccess {
		p.logger.Error("支付单未支付成功",
			zap.Int64("payOrderId", payOrder.ID),
			zap.Int("status", payOrder.Status),
		)
		return nil, fmt.Errorf("支付单未支付成功")
	}

	// 4.3 校验支付金额一致
	if payOrder.Price != order.PayPrice {
		p.logger.Error("支付金额不匹配",
			zap.Int64("orderId", order.ID),
			zap.Int("orderPrice", order.PayPrice),
			zap.Int("payPrice", payOrder.Price),
		)
		return nil, fmt.Errorf("支付金额不匹配")
	}

	// 4.4 校验商户订单号一致
	if payOrder.MerchantOrderId != order.No {
		p.logger.Error("支付单商户订单号不匹配",
			zap.Int64("orderId", order.ID),
			zap.String("orderNo", order.No),
			zap.String("payOrderNo", payOrder.MerchantOrderId),
		)
		return nil, fmt.Errorf("支付单不匹配")
	}

	// 5. 在事务中更新订单状态
	err = p.q.Transaction(func(tx *query.Query) error {
		now := time.Now()
		updateData := map[string]interface{}{
			"status":           tradeModel.TradeOrderStatusUndelivered,
			"pay_status":       model.BitBool(true),
			"pay_time":         now,
			"pay_channel_code": payOrder.ChannelCode, // 记录支付渠道
			"update_time":      now,
		}

		_, err := tx.TradeOrder.WithContext(ctx).
			Where(tx.TradeOrder.ID.Eq(handleReq.OrderID)).
			Updates(updateData)
		return err
	})

	if err != nil {
		p.logger.Error("支付订单事务失败",
			zap.Error(err),
			zap.Int64("orderId", handleReq.OrderID),
			zap.Int64("payOrderId", handleReq.PayOrderID),
		)
		return nil, err
	}

	// 6. 更新订单对象状态
	order.Status = tradeModel.TradeOrderStatusUndelivered
	order.PayStatus = true
	payTime := time.Now()
	order.PayTime = &payTime
	order.PayChannelCode = payOrder.ChannelCode

	// 7. 记录操作成功日志
	p.logger.Info("订单支付处理成功",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("payOrderId", handleReq.PayOrderID),
		zap.Int64("userId", order.UserID),
		zap.Int("beforeStatus", tradeModel.TradeOrderStatusUnpaid),
		zap.Int("afterStatus", tradeModel.TradeOrderStatusUndelivered),
		zap.Int("payPrice", order.PayPrice),
	)

	return &OrderHandleResponse{
		Order:   order,
		Success: true,
		Message: "订单支付成功",
	}, nil
}

// DeliveryOrderProcessor 发货订单业务处理器
type DeliveryOrderProcessor struct {
	*BaseOrderHandler
	q      *query.Query
	logger *zap.Logger
}

// NewDeliveryOrderProcessor 发货订单处理器构造函数
func NewDeliveryOrderProcessor(q *query.Query, logger *zap.Logger) *DeliveryOrderProcessor {
	return &DeliveryOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("delivery", []string{"delivery"}),
		q:                q,
		logger:           logger,
	}
}

// Handle 处理订单发货操作
func (p *DeliveryOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理订单发货请求",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("logisticsId", handleReq.LogisticsID),
		zap.String("trackingNo", handleReq.TrackingNo),
	)

	// 1. 查询订单信息
	order, err := p.q.TradeOrder.WithContext(ctx).
		Where(p.q.TradeOrder.ID.Eq(handleReq.OrderID)).
		First()
	if err != nil {
		return nil, ErrOrderNotExists()
	}

	// 2. 在事务中更新订单状态
	err = p.q.Transaction(func(tx *query.Query) error {
		now := time.Now()
		updateData := map[string]interface{}{
			"status":        tradeModel.TradeOrderStatusDelivered,
			"logistics_id":  handleReq.LogisticsID,
			"logistics_no":  handleReq.TrackingNo,
			"delivery_time": now,
			"update_time":   now,
		}

		_, err := tx.TradeOrder.WithContext(ctx).
			Where(tx.TradeOrder.ID.Eq(handleReq.OrderID)).
			Updates(updateData)
		return err
	})

	if err != nil {
		p.logger.Error("订单发货事务失败",
			zap.Error(err),
			zap.Int64("orderId", handleReq.OrderID),
			zap.Int64("logisticsId", handleReq.LogisticsID),
			zap.String("trackingNo", handleReq.TrackingNo),
		)
		return nil, err
	}

	// 3. 更新订单对象状态
	order.Status = tradeModel.TradeOrderStatusDelivered
	order.LogisticsID = handleReq.LogisticsID
	order.LogisticsNo = handleReq.TrackingNo
	deliveryTime := time.Now()
	order.DeliveryTime = &deliveryTime

	// 4. 记录操作成功日志
	p.logger.Info("订单发货处理成功",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("logisticsId", handleReq.LogisticsID),
		zap.String("trackingNo", handleReq.TrackingNo),
		zap.Int64("userId", order.UserID),
		zap.Int("beforeStatus", tradeModel.TradeOrderStatusUndelivered),
		zap.Int("afterStatus", tradeModel.TradeOrderStatusDelivered),
	)

	return &OrderHandleResponse{
		Order:   order,
		Success: true,
		Message: "订单发货成功",
	}, nil
}

// ReceiveOrderProcessor 确认收货订单业务处理器
type ReceiveOrderProcessor struct {
	*BaseOrderHandler
	q      *query.Query
	logger *zap.Logger
}

// NewReceiveOrderProcessor 确认收货订单处理器构造函数
func NewReceiveOrderProcessor(q *query.Query, logger *zap.Logger) *ReceiveOrderProcessor {
	return &ReceiveOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("receive", []string{"receive"}),
		q:                q,
		logger:           logger,
	}
}

// Handle 处理确认收货操作
func (p *ReceiveOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理订单确认收货请求",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("userId", handleReq.UserID),
	)

	// 1. 查询订单信息
	order, err := p.q.TradeOrder.WithContext(ctx).
		Where(p.q.TradeOrder.ID.Eq(handleReq.OrderID)).
		First()
	if err != nil {
		return nil, ErrOrderNotExists()
	}

	// 2. 在事务中更新订单状态
	err = p.q.Transaction(func(tx *query.Query) error {
		now := time.Now()
		updateData := map[string]interface{}{
			"status":       tradeModel.TradeOrderStatusCompleted,
			"receive_time": now,
			"finish_time":  now,
			"update_time":  now,
		}

		_, err := tx.TradeOrder.WithContext(ctx).
			Where(tx.TradeOrder.ID.Eq(handleReq.OrderID)).
			Updates(updateData)
		return err
	})

	if err != nil {
		p.logger.Error("确认收货事务失败",
			zap.Error(err),
			zap.Int64("orderId", handleReq.OrderID),
			zap.Int64("userId", handleReq.UserID),
		)
		return nil, err
	}

	// 3. 更新订单对象状态
	order.Status = tradeModel.TradeOrderStatusCompleted
	receiveTime := time.Now()
	order.ReceiveTime = &receiveTime
	order.FinishTime = &receiveTime

	// 4. 记录操作成功日志
	p.logger.Info("订单确认收货处理成功",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("userId", handleReq.UserID),
		zap.Int("beforeStatus", tradeModel.TradeOrderStatusDelivered),
		zap.Int("afterStatus", tradeModel.TradeOrderStatusCompleted),
		zap.String("orderNo", order.No),
	)

	return &OrderHandleResponse{
		Order:   order,
		Success: true,
		Message: "订单确认收货成功",
	}, nil
}

// CancelOrderProcessor 取消订单业务处理器
type CancelOrderProcessor struct {
	*BaseOrderHandler
	q         *query.Query
	skuSvc    ProductSkuServiceAPI
	couponSvc CouponUserServiceAPI
	memberSvc MemberUserServiceAPI
	logger    *zap.Logger
}

// NewCancelOrderProcessor 取消订单处理器构造函数
func NewCancelOrderProcessor(
	q *query.Query,
	skuSvc ProductSkuServiceAPI,
	couponSvc CouponUserServiceAPI,
	memberSvc MemberUserServiceAPI,
	logger *zap.Logger,
) *CancelOrderProcessor {
	return &CancelOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("cancel", []string{"cancel"}),
		q:                q,
		skuSvc:           skuSvc,
		couponSvc:        couponSvc,
		memberSvc:        memberSvc,
		logger:           logger,
	}
}

// Handle 处理取消订单操作
func (p *CancelOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理订单取消请求",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("userId", handleReq.UserID),
		zap.Int("cancelType", handleReq.CancelType),
		zap.String("cancelReason", handleReq.CancelReason),
	)

	// 1. 查询订单信息
	order, err := p.q.TradeOrder.WithContext(ctx).
		Where(p.q.TradeOrder.ID.Eq(handleReq.OrderID)).
		First()
	if err != nil {
		return nil, ErrOrderNotExists()
	}

	// 2. 在事务中处理订单取消
	err = p.q.Transaction(func(tx *query.Query) error {
		now := time.Now()
		updateData := map[string]interface{}{
			"status":        tradeModel.TradeOrderStatusCanceled,
			"cancel_time":   now,
			"cancel_type":   handleReq.CancelType,
			"cancel_reason": handleReq.CancelReason,
			"update_time":   now,
		}

		_, err := tx.TradeOrder.WithContext(ctx).
			Where(tx.TradeOrder.ID.Eq(handleReq.OrderID)).
			Updates(updateData)
		return err
	})

	if err != nil {
		p.logger.Error("取消订单事务失败",
			zap.Error(err),
			zap.Int64("orderId", handleReq.OrderID),
			zap.Int64("userId", handleReq.UserID),
			zap.Int("cancelType", handleReq.CancelType),
			zap.String("cancelReason", handleReq.CancelReason),
		)
		return nil, err
	}

	// 3. 更新订单对象状态
	order.Status = tradeModel.TradeOrderStatusCanceled
	cancelTime := time.Now()
	order.CancelTime = &cancelTime
	order.CancelType = handleReq.CancelType

	// 4. 记录操作成功日志
	p.logger.Info("订单取消处理成功",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("userId", handleReq.UserID),
		zap.Int("cancelType", handleReq.CancelType),
		zap.String("cancelReason", handleReq.CancelReason),
		zap.Int("beforeStatus", order.Status),
		zap.Int("afterStatus", tradeModel.TradeOrderStatusCanceled),
		zap.String("orderNo", order.No),
	)

	return &OrderHandleResponse{
		Order:   order,
		Success: true,
		Message: "订单取消成功",
	}, nil
}

func (p *CancelOrderProcessor) AfterCancelOrder(ctx context.Context, handleReq *OrderHandleRequest, resp *OrderHandleResponse) error {
	order := resp.Order
	orderItems := handleReq.OrderItems

	// 1. 退还库存
	if len(orderItems) > 0 {
		stockItems := make([]product.ProductSkuUpdateStockItemReq, len(orderItems))
		for i, item := range orderItems {
			stockItems[i] = product.ProductSkuUpdateStockItemReq{
				ID:        item.SkuID,
				IncrCount: int(item.Count), // 正数表示增加（退还）
			}
		}
		if err := p.skuSvc.UpdateSkuStock(ctx, &product.ProductSkuUpdateStockReq{Items: stockItems}); err != nil {
			p.logger.Error("退还库存失败", zap.Error(err), zap.Int64("orderId", order.ID))
			return err
		}
	}

	// 2. 退还优惠券
	if order.CouponID > 0 {
		if err := p.couponSvc.ReturnCouponForOrder(ctx, order.UserID, order.CouponID, order.ID); err != nil {
			p.logger.Error("退还优惠券失败", zap.Error(err), zap.Int64("orderId", order.ID), zap.Int64("couponId", order.CouponID))
			return err
		}
	}

	// 3. 退还积分
	if order.UsePoint > 0 {
		if err := member.NewMemberPointRecordService(p.q, nil).CreatePointRecord(ctx, order.UserID, order.UsePoint, tradeModel.MemberPointBizTypeOrderUseCancel, strconv.FormatInt(order.ID, 10)); err != nil {
			p.logger.Error("退还积分失败", zap.Int64("orderId", order.ID), zap.Int("usePoint", order.UsePoint))
			return err
		}
	}

	return nil
}

// RefundOrderProcessor 退款订单业务处理器
type RefundOrderProcessor struct {
	*BaseOrderHandler
	q      *query.Query
	logger *zap.Logger
}

// NewRefundOrderProcessor 退款订单处理器构造函数
func NewRefundOrderProcessor(q *query.Query, logger *zap.Logger) *RefundOrderProcessor {
	return &RefundOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("refund", []string{"refund"}),
		q:                q,
		logger:           logger,
	}
}

// Handle 处理订单退款操作
func (p *RefundOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理订单退款请求",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("refundPrice", handleReq.RefundPrice),
		zap.String("refundReason", handleReq.RefundReason),
	)

	// 1. 查询订单信息
	order, err := p.q.TradeOrder.WithContext(ctx).
		Where(p.q.TradeOrder.ID.Eq(handleReq.OrderID)).
		First()
	if err != nil {
		return nil, ErrOrderNotExists()
	}

	// 2. 计算退款金额
	refundAmount := int(handleReq.RefundPrice)
	if refundAmount <= 0 {
		refundAmount = order.PayPrice // 默认全额退款
	}

	// 3. 在事务中处理退款
	err = p.q.Transaction(func(tx *query.Query) error {
		now := time.Now()
		updateData := map[string]interface{}{
			"refund_price": order.RefundPrice + refundAmount,
			"update_time":  now,
		}

		// 如果全额退款，更新订单状态
		if order.RefundPrice+refundAmount >= order.PayPrice {
			updateData["status"] = tradeModel.TradeOrderStatusCanceled
			updateData["cancel_time"] = now
			updateData["cancel_type"] = tradeModel.OrderCancelTypeSystem
		}

		_, err := tx.TradeOrder.WithContext(ctx).
			Where(tx.TradeOrder.ID.Eq(handleReq.OrderID)).
			Updates(updateData)
		return err
	})

	if err != nil {
		p.logger.Error("退款订单事务失败",
			zap.Error(err),
			zap.Int64("orderId", handleReq.OrderID),
			zap.Int("refundAmount", refundAmount),
			zap.String("refundReason", handleReq.RefundReason),
		)
		return nil, err
	}

	// 4. 更新订单对象状态
	beforeRefundPrice := order.RefundPrice
	order.RefundPrice += refundAmount
	isFullRefund := order.RefundPrice >= order.PayPrice

	if isFullRefund {
		order.Status = tradeModel.TradeOrderStatusCanceled
		cancelTime := time.Now()
		order.CancelTime = &cancelTime
		order.CancelType = tradeModel.OrderCancelTypeSystem
	}

	// 5. 记录操作成功日志
	p.logger.Info("订单退款处理成功",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int("refundAmount", refundAmount),
		zap.Int("beforeRefundPrice", beforeRefundPrice),
		zap.Int("afterRefundPrice", order.RefundPrice),
		zap.Int("payPrice", order.PayPrice),
		zap.Bool("isFullRefund", isFullRefund),
		zap.String("refundReason", handleReq.RefundReason),
		zap.String("orderNo", order.No),
	)

	return &OrderHandleResponse{
		Order:   order,
		Success: true,
		Message: "订单退款成功",
		Data: map[string]interface{}{
			"refundAmount": refundAmount,
			"totalRefund":  order.RefundPrice,
		},
	}, nil
}

// PickUpOrderProcessor 核销订单业务处理器
type PickUpOrderProcessor struct {
	*BaseOrderHandler
	q      *query.Query
	logger *zap.Logger
}

// NewPickUpOrderProcessor 核销订单处理器构造函数
func NewPickUpOrderProcessor(q *query.Query, logger *zap.Logger) *PickUpOrderProcessor {
	return &PickUpOrderProcessor{
		BaseOrderHandler: NewBaseOrderHandler("pickup", []string{"pickup"}),
		q:                q,
		logger:           logger,
	}
}

// Handle 处理订单核销操作
func (p *PickUpOrderProcessor) Handle(ctx context.Context, handleReq *OrderHandleRequest) (*OrderHandleResponse, error) {
	p.logger.Info("开始处理订单核销请求",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("adminId", handleReq.AdminID),
	)

	// 1. 查询订单信息
	order, err := p.q.TradeOrder.WithContext(ctx).
		Where(p.q.TradeOrder.ID.Eq(handleReq.OrderID)).
		First()
	if err != nil {
		return nil, ErrOrderNotExists()
	}

	// 2. 验证订单状态和配送方式
	if order.DeliveryType != tradeModel.DeliveryTypePickUp {
		return nil, ErrOrderNotPickUp()
	}

	if order.Status != tradeModel.TradeOrderStatusDelivered {
		return nil, ErrOrderStatusError()
	}

	// 3. 在事务中更新订单状态
	err = p.q.Transaction(func(tx *query.Query) error {
		now := time.Now()
		updateData := map[string]interface{}{
			"status":       tradeModel.TradeOrderStatusCompleted,
			"receive_time": now,
			"finish_time":  now,
			"update_time":  now,
		}

		_, err := tx.TradeOrder.WithContext(ctx).
			Where(tx.TradeOrder.ID.Eq(handleReq.OrderID)).
			Updates(updateData)
		return err
	})

	if err != nil {
		p.logger.Error("核销订单事务失败",
			zap.Error(err),
			zap.Int64("orderId", handleReq.OrderID),
			zap.Int64("adminId", handleReq.AdminID),
		)
		return nil, err
	}

	// 4. 更新订单对象状态
	order.Status = tradeModel.TradeOrderStatusCompleted
	finishTime := time.Now()
	order.ReceiveTime = &finishTime
	order.FinishTime = &finishTime

	// 5. 记录操作成功日志
	p.logger.Info("订单核销处理成功",
		zap.Int64("orderId", handleReq.OrderID),
		zap.Int64("adminId", handleReq.AdminID),
		zap.Int64("userId", order.UserID),
		zap.Int("beforeStatus", tradeModel.TradeOrderStatusDelivered),
		zap.Int("afterStatus", tradeModel.TradeOrderStatusCompleted),
		zap.String("orderNo", order.No),
		zap.Int64("pickUpStoreId", order.PickUpStoreID),
	)

	return &OrderHandleResponse{
		Order:   order,
		Success: true,
		Message: "订单核销成功",
	}, nil
}
