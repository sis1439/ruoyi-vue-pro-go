package trade

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	trade2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	tradeRepo "github.com/wxlbd/ruoyi-mall-go/internal/repo/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type TradeAfterSaleService struct {
	q                    *query.Query
	orderSvc             *TradeOrderUpdateService
	orderQuerySvc        *TradeOrderQueryService
	expressSvc           *DeliveryExpressService
	noDAO                *tradeRepo.TradeNoRedisDAO
	payRefundSvc         *pay.PayRefundService
	combinationRecordSvc promotion.CombinationRecordService
	memberUserSvc        *member.MemberUserService
	afterSaleLogSvc      *AfterSaleLogService
	logSvc               *TradeOrderLogService
	configSvc            *TradeConfigService
	payAppSvc            *pay.PayAppService
}

func NewTradeAfterSaleService(
	q *query.Query,
	orderSvc *TradeOrderUpdateService,
	orderQuerySvc *TradeOrderQueryService,
	expressSvc *DeliveryExpressService,
	noDAO *tradeRepo.TradeNoRedisDAO,
	payRefundSvc *pay.PayRefundService,
	combinationRecordSvc promotion.CombinationRecordService,
	memberUserSvc *member.MemberUserService,
	afterSaleLogSvc *AfterSaleLogService,
	logSvc *TradeOrderLogService,
	configSvc *TradeConfigService,
	payAppSvc *pay.PayAppService,
) *TradeAfterSaleService {
	return &TradeAfterSaleService{
		q:                    q,
		orderSvc:             orderSvc,
		orderQuerySvc:        orderQuerySvc,
		expressSvc:           expressSvc,
		noDAO:                noDAO,
		payRefundSvc:         payRefundSvc,
		combinationRecordSvc: combinationRecordSvc,
		memberUserSvc:        memberUserSvc,
		afterSaleLogSvc:      afterSaleLogSvc,
		logSvc:               logSvc,
		configSvc:            configSvc,
		payAppSvc:            payAppSvc,
	}
}

// CreateAfterSale 创建售后 (App)
func (s *TradeAfterSaleService) CreateAfterSale(ctx context.Context, userId int64, r *trade2.AppAfterSaleCreateReq) (int64, error) {
	// 1. 前置校验
	item, err := s.validateOrderItemApplicable(ctx, userId, r)
	if err != nil {
		return 0, err
	}

	// 2. 执行创建逻辑 (事务)
	var afterSaleId int64
	err = s.q.Transaction(func(tx *query.Query) error {
		// 2.1 存储售后订单
		afterSale, err := s.createAfterSaleDOWithQuery(ctx, tx, r, item)
		if err != nil {
			return err
		}
		afterSaleId = afterSale.ID

		// 2.2 更新订单项的售后状态
		if err := s.orderSvc.UpdateOrderItemWhenAfterSaleCreate(ctx, item.OrderID, item.ID, afterSaleId); err != nil {
			return fmt.Errorf("更新订单项售后状态失败: %w", err)
		}

		// 2.3 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       userId,
			UserType:     consts.UserTypeMember,
			AfterSaleID:  afterSale.ID,
			BeforeStatus: consts.AfterSaleStatusNone,
			AfterStatus:  consts.AfterSaleStatusApply,
			OperateType:  consts.AfterSaleOperateTypeMemberCreate,
			Content:      "用户申请售后",
		}); err != nil {
			return fmt.Errorf("记录售后日志失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return afterSaleId, nil
}

func (s *TradeAfterSaleService) validateOrderItemApplicable(ctx context.Context, userId int64, r *trade2.AppAfterSaleCreateReq) (*trade.TradeOrderItem, error) {
	// 校验订单项存在
	item, err := s.q.TradeOrderItem.WithContext(ctx).Where(s.q.TradeOrderItem.ID.Eq(r.OrderItemID), s.q.TradeOrderItem.UserID.Eq(userId)).First()
	if err != nil {
		return nil, fmt.Errorf("订单项不存在")
	}

	// 已申请售后，不允许再发起售后申请
	if item.AfterSaleStatus != consts.AfterSaleStatusNone {
		return nil, fmt.Errorf("该订单项已申请售后")
	}

	// 申请的退款金额，不能超过商品的价格
	if r.RefundPrice > item.PayPrice {
		return nil, fmt.Errorf("退款金额不能超过商品支付价格")
	}

	// 校验订单存在
	order, err := s.q.TradeOrder.WithContext(ctx).Where(s.q.TradeOrder.ID.Eq(item.OrderID), s.q.TradeOrder.UserID.Eq(userId)).First()
	if err != nil {
		return nil, fmt.Errorf("订单不存在")
	}

	// 已取消，无法发起售后
	if order.Status == consts.TradeOrderStatusCanceled {
		return nil, fmt.Errorf("订单已取消，无法发起售后")
	}

	// 未支付，无法发起售后
	if order.Status == consts.TradeOrderStatusUnpaid {
		return nil, fmt.Errorf("订单未支付，无法发起售后")
	}

	// 如果是【退货退款】的情况，需要额外校验是否发货
	if r.Way == consts.AfterSaleWayReturnAndRefund && order.Status < consts.TradeOrderStatusDelivered {
		return nil, fmt.Errorf("订单未发货，无法申请退货退款")
	}

	// 如果是拼团订单，则进行中不允许售后
	if order.CombinationRecordID > 0 {
		record, err := s.combinationRecordSvc.GetCombinationRecord(ctx, order.CombinationRecordID)
		if err == nil && record != nil && record.Status == 0 { // 0: IN_PROGRESS
			return nil, fmt.Errorf("拼团正在进行中，不允许售后")
		}
	}

	return item, nil
}

func (s *TradeAfterSaleService) createAfterSaleDOWithQuery(ctx context.Context, tx *query.Query, r *trade2.AppAfterSaleCreateReq, item *trade.TradeOrderItem) (*trade.AfterSale, error) {
	// 生成售后单号
	afterSaleNo, err := s.noDAO.GenerateAfterSaleNo(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate after sale no failed: %w", err)
	}

	pics, _ := json.Marshal(r.ApplyPicURLs)
	props, _ := json.Marshal(item.Properties)

	// 获取订单流水号用于检索
	order, _ := tx.TradeOrder.WithContext(ctx).Where(tx.TradeOrder.ID.Eq(item.OrderID)).First()

	afterSale := &trade.AfterSale{
		No:               afterSaleNo,
		Status:           consts.AfterSaleStatusApply,
		Way:              r.Way,
		Type:             consts.AfterSaleTypeInSale,
		UserID:           item.UserID,
		ApplyReason:      r.ApplyReason,
		ApplyDescription: r.ApplyDescription,
		ApplyPicURLs:     string(pics),
		OrderID:          item.OrderID,
		OrderNo:          order.No,
		OrderItemID:      item.ID,
		SpuID:            item.SpuID,
		SpuName:          item.SpuName,
		SkuID:            item.SkuID,
		Properties:       string(props),
		PicURL:           item.PicURL,
		Count:            r.Count,
		RefundPrice:      r.RefundPrice,
	}

	// 标记是售中还是售后
	if order.Status == consts.TradeOrderStatusCompleted {
		afterSale.Type = consts.AfterSaleTypeAfterSale
	}

	if err := tx.AfterSale.WithContext(ctx).Create(afterSale); err != nil {
		return nil, err
	}

	return afterSale, nil
}

// GetAfterSale 获得售后详情 (App)
func (s *TradeAfterSaleService) GetAfterSale(ctx context.Context, userId int64, id int64) (*trade2.AppAfterSaleResp, error) {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(id), s.q.AfterSale.UserID.Eq(userId)).First()
	if err != nil {
		return nil, err
	}

	return s.convertToAppAfterSaleResp(as), nil
}

func (s *TradeAfterSaleService) convertToAppAfterSaleResp(as *trade.AfterSale) *trade2.AppAfterSaleResp {
	res := &trade2.AppAfterSaleResp{
		ID:               as.ID,
		No:               as.No,
		Status:           as.Status,
		Way:              as.Way,
		Type:             as.Type,
		ApplyReason:      as.ApplyReason,
		ApplyDescription: as.ApplyDescription,
		CreateTime:       as.CreateTime,
		UpdateTime:       as.UpdateTime,
		OrderID:          as.OrderID,
		OrderNo:          as.OrderNo,
		OrderItemID:      as.OrderItemID,
		SpuID:            as.SpuID,
		SpuName:          as.SpuName,
		SkuID:            as.SkuID,
		PicURL:           as.PicURL,
		Count:            as.Count,
		AuditReason:      as.AuditReason,
		RefundPrice:      as.RefundPrice,
		LogisticsID:      as.LogisticsID,
		LogisticsNo:      as.LogisticsNo,
		ReceiveReason:    as.ReceiveReason,
	}

	// JSON 解析
	if as.ApplyPicURLs != "" {
		_ = json.Unmarshal([]byte(as.ApplyPicURLs), &res.ApplyPicURLs)
	}
	if as.Properties != "" {
		_ = json.Unmarshal([]byte(as.Properties), &res.Properties)
	}

	// 时间指针处理
	if !as.AuditTime.IsZero() {
		res.AuditTime = &as.AuditTime
	}
	if !as.RefundTime.IsZero() {
		res.RefundTime = &as.RefundTime
	}
	if !as.DeliveryTime.IsZero() {
		res.DeliveryTime = &as.DeliveryTime
	}
	if !as.ReceiveTime.IsZero() {
		res.ReceiveTime = &as.ReceiveTime
	}

	return res
}

// GetUserAfterSalePage 获得售后分页 (App)
func (s *TradeAfterSaleService) GetUserAfterSalePage(ctx context.Context, userId int64, r *trade2.AppAfterSalePageReq) (*pagination.PageResult[*trade2.AppAfterSaleResp], error) {
	q := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.UserID.Eq(userId))
	if r.Status != nil {
		q = q.Where(s.q.AfterSale.Status.Eq(*r.Status))
	}

	list, total, err := q.Order(s.q.AfterSale.ID.Desc()).FindByPage(r.GetOffset(), r.PageSize)
	if err != nil {
		return nil, err
	}

	resList := make([]*trade2.AppAfterSaleResp, len(list))
	for i, as := range list {
		resList[i] = s.convertToAppAfterSaleResp(as)
	}

	return &pagination.PageResult[*trade2.AppAfterSaleResp]{
		List:  resList,
		Total: total,
	}, nil
}

// GetAfterSalePage 获得售后分页 (Admin)
func (s *TradeAfterSaleService) GetAfterSalePage(ctx context.Context, r *trade2.TradeAfterSalePageReq) (*pagination.PageResult[*trade2.AfterSalePageItemResp], error) {
	q := s.q.AfterSale.WithContext(ctx)
	if r.No != "" {
		q = q.Where(s.q.AfterSale.No.Like("%" + r.No + "%"))
	}
	if r.UserID != nil {
		q = q.Where(s.q.AfterSale.UserID.Eq(*r.UserID))
	}
	if r.Status != nil {
		q = q.Where(s.q.AfterSale.Status.Eq(*r.Status))
	}

	list, total, err := q.Order(s.q.AfterSale.ID.Desc()).FindByPage(r.GetOffset(), r.PageSize)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return &pagination.PageResult[*trade2.AfterSalePageItemResp]{
			List:  []*trade2.AfterSalePageItemResp{},
			Total: total,
		}, nil
	}

	// 聚合用户信息
	userIds := make([]int64, 0, len(list))
	for _, as := range list {
		userIds = append(userIds, as.UserID)
	}
	userMap, _ := s.memberUserSvc.GetUserRespMap(ctx, userIds)

	resList := make([]*trade2.AfterSalePageItemResp, len(list))
	for i, as := range list {
		item := &trade2.AfterSalePageItemResp{
			ID:          as.ID,
			No:          as.No,
			Status:      as.Status,
			Type:        as.Type,
			Way:         as.Way,
			UserID:      as.UserID,
			ApplyReason: as.ApplyReason,
			SpuName:     as.SpuName,
			PicURL:      as.PicURL,
			Count:       as.Count,
			RefundPrice: as.RefundPrice,
			CreateTime:  as.CreateTime,
		}
		if userMap != nil {
			item.User = userMap[as.UserID]
		}
		resList[i] = item
	}

	return &pagination.PageResult[*trade2.AfterSalePageItemResp]{
		List:  resList,
		Total: total,
	}, nil
}

// CancelAfterSale 取消售后
func (s *TradeAfterSaleService) CancelAfterSale(ctx context.Context, userId int64, id int64) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(id), s.q.AfterSale.UserID.Eq(userId)).First()
	if err != nil {
		return err
	}
	if as.Status != consts.AfterSaleStatusApply {
		return fmt.Errorf("状态不允许取消")
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态
		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(id)).Update(tx.AfterSale.Status, consts.AfterSaleStatusBuyerCancel); err != nil {
			return err
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       userId,
			UserType:     consts.UserTypeMember,
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  consts.AfterSaleStatusBuyerCancel,
			OperateType:  consts.AfterSaleOperateTypeMemberCancel,
			Content:      "用户取消售后申请",
		}); err != nil {
			return err
		}

		// 3. 更新订单项状态为【未申请】
		if err := s.orderSvc.UpdateOrderItemWhenAfterSaleCancel(ctx, as.OrderID, as.OrderItemID); err != nil {
			return err
		}
		return nil
	})
}

// AgreeAfterSale 同意售后
func (s *TradeAfterSaleService) AgreeAfterSale(ctx context.Context, adminUserId int64, id int64) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(id)).First()
	if err != nil {
		return err
	}
	if as.Status != consts.AfterSaleStatusApply {
		return fmt.Errorf("售后单状态不是申请中，无法同意")
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态
		newStatus := consts.AfterSaleStatusSellerAgree
		if as.Way == consts.AfterSaleWayRefund {
			newStatus = consts.AfterSaleStatusWaitRefund
		}

		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(id)).Updates(trade.AfterSale{
			Status:      newStatus,
			AuditTime:   time.Now(),
			AuditUserID: adminUserId,
		}); err != nil {
			return err
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       adminUserId,
			UserType:     consts.UserTypeAdmin,
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  newStatus,
			OperateType:  consts.AfterSaleOperateTypeAdminAgreeApply,
			Content:      "管理员同意售后申请",
		}); err != nil {
			return err
		}

		// 3. 更新订单项状态
		if _, err := tx.TradeOrderItem.WithContext(ctx).Where(tx.TradeOrderItem.ID.Eq(as.OrderItemID)).Update(tx.TradeOrderItem.AfterSaleStatus, int32(newStatus)); err != nil {
			return err
		}
		return nil
	})
}

// DisagreeAfterSale 拒绝售后 (审核不通过)
func (s *TradeAfterSaleService) DisagreeAfterSale(ctx context.Context, adminUserId int64, req *trade2.TradeAfterSaleDisagreeReq) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(req.ID)).First()
	if err != nil {
		return err
	}
	if as.Status != consts.AfterSaleStatusApply {
		return fmt.Errorf("售后单状态不是申请中，无法拒绝")
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态
		newStatus := consts.AfterSaleStatusSellerDisagree
		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(req.ID)).Updates(trade.AfterSale{
			Status:      newStatus,
			AuditReason: req.AuditReason,
			AuditTime:   time.Now(),
			AuditUserID: adminUserId,
		}); err != nil {
			return err
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       adminUserId,
			UserType:     consts.UserTypeAdmin,
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  newStatus,
			OperateType:  consts.AfterSaleOperateTypeAdminDisagreeApply,
			Content:      fmt.Sprintf("管理员拒绝售后申请，原因：%s", req.AuditReason),
		}); err != nil {
			return err
		}

		// 3. 更新交易订单项的售后状态为【未申请】
		if err := s.orderSvc.UpdateOrderItemWhenAfterSaleCancel(ctx, as.OrderID, as.OrderItemID); err != nil {
			return err
		}
		return nil
	})
}

// RefundAfterSale 退款
func (s *TradeAfterSaleService) RefundAfterSale(ctx context.Context, adminUserId int64, userIp string, id int64) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(id)).First()
	if err != nil {
		return err
	}
	if as.Status != consts.AfterSaleStatusWaitRefund {
		return fmt.Errorf("售后单状态不是待退款，无法退款")
	}

	// 1. 获取交易配置以获取支付 AppKey
	config, err := s.configSvc.GetTradeConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取交易配置失败: %w", err)
	}
	payApp, err := s.payAppSvc.GetApp(ctx, config.AppID)
	if err != nil {
		return fmt.Errorf("获取支付应用失败: %w", err)
	}

	// 2. 发起退款申请
	refundReq := &pay2.PayRefundCreateReq{
		AppKey:           payApp.AppKey,
		UserIP:           userIp,
		MerchantOrderId:  strconv.FormatInt(as.OrderID, 10),
		MerchantRefundId: strconv.FormatInt(as.ID, 10),
		Reason:           fmt.Sprintf("售后退款: %s", as.SpuName),
		Price:            as.RefundPrice,
	}
	payRefundId, err := s.payRefundSvc.CreateRefund(ctx, refundReq)
	if err != nil {
		return fmt.Errorf("发起退款申请失败: %w", err)
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 更新售后单的支付退款 ID
		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(id)).Update(tx.AfterSale.PayRefundID, payRefundId); err != nil {
			return err
		}

		// 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       adminUserId,
			UserType:     2, // Admin
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  as.Status,
			OperateType:  consts.AfterSaleOperateTypeAdminRefund,
			Content:      "管理员发起退款申请",
		}); err != nil {
			return err
		}

		return nil
	})
}

// GetAfterSaleDetail 获得售后详情 (Admin)
func (s *TradeAfterSaleService) GetAfterSaleDetail(ctx context.Context, id int64) (*trade2.TradeAfterSaleDetailResp, error) {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	res := &trade2.TradeAfterSaleDetailResp{
		ID:               as.ID,
		No:               as.No,
		Status:           as.Status,
		Type:             as.Type,
		Way:              as.Way,
		UserID:           as.UserID,
		ApplyReason:      as.ApplyReason,
		ApplyDescription: as.ApplyDescription,
		OrderID:          as.OrderID,
		OrderNo:          as.OrderNo,
		OrderItemID:      as.OrderItemID,
		SpuID:            as.SpuID,
		SpuName:          as.SpuName,
		SkuID:            as.SkuID,
		PicURL:           as.PicURL,
		Count:            as.Count,
		RefundPrice:      as.RefundPrice,
		AuditUserID:      as.AuditUserID,
		AuditReason:      as.AuditReason,
		PayRefundID:      as.PayRefundID,
		LogisticsID:      as.LogisticsID,
		LogisticsNo:      as.LogisticsNo,
		ReceiveReason:    as.ReceiveReason,
		CreateTime:       as.CreateTime,
	}

	// 1. JSON 转换
	if as.ApplyPicURLs != "" {
		_ = json.Unmarshal([]byte(as.ApplyPicURLs), &res.ApplyPicURLs)
	}

	// 2. 时间指针转换
	if !as.AuditTime.IsZero() {
		res.AuditTime = &as.AuditTime
	}
	if !as.RefundTime.IsZero() {
		res.RefundTime = &as.RefundTime
	}
	if !as.DeliveryTime.IsZero() {
		res.DeliveryTime = &as.DeliveryTime
	}
	if !as.ReceiveTime.IsZero() {
		res.ReceiveTime = &as.ReceiveTime
	}

	// 3. 聚合关联数据
	// 3.1 订单
	order, _ := s.orderQuerySvc.GetOrder(ctx, as.OrderID)
	if order != nil {
		res.Order = &trade2.TradeOrderBase{
			ID:         order.ID,
			No:         order.No,
			UserID:     order.UserID,
			Status:     order.Status,
			CreateTime: order.CreateTime,
			PayPrice:   order.PayPrice,
		}
	}

	// 3.2 订单项
	item, _ := s.orderQuerySvc.GetOrderItem(ctx, as.UserID, as.OrderItemID)
	if item != nil {
		res.OrderItem = &trade2.AfterSaleOrderItem{
			TradeOrderItemBase: trade2.TradeOrderItemBase{
				ID:      item.ID,
				OrderID: item.OrderID,
				SpuID:   item.SpuID,
				SpuName: item.SpuName,
				SkuID:   item.SkuID,
				PicURL:  item.PicURL,
				Count:   item.Count,
				Price:   item.Price,
			},
		}
		if len(item.Properties) > 0 {
			for _, p := range item.Properties {
				res.OrderItem.Properties = append(res.OrderItem.Properties, product.ProductSkuPropertyResp{
					PropertyID:   p.PropertyID,
					PropertyName: p.PropertyName,
					ValueID:      p.ValueID,
					ValueName:    p.ValueName,
				})
			}
		}
	}

	// 3.3 用户
	userMap, _ := s.memberUserSvc.GetUserRespMap(ctx, []int64{as.UserID})
	if userMap != nil {
		res.User = userMap[as.UserID]
	}

	// 3.4 日志
	logs, _ := s.afterSaleLogSvc.GetAfterSaleLogList(ctx, as.ID)
	if len(logs) > 0 {
		res.Logs = make([]trade2.AfterSaleLogResp, len(logs))
		for i, l := range logs {
			res.Logs[i] = trade2.AfterSaleLogResp{
				ID:           l.ID,
				AfterSaleID:  l.AfterSaleID,
				BeforeStatus: l.BeforeStatus,
				AfterStatus:  l.AfterStatus,
				OperateType:  l.OperateType,
				UserType:     l.UserType,
				UserID:       l.UserID,
				Content:      l.Content,
				CreateTime:   l.CreateTime,
			}
		}
	}

	return res, nil
}

// ReceiveAfterSale 确认收货 (Admin)
func (s *TradeAfterSaleService) ReceiveAfterSale(ctx context.Context, adminUserId int64, id int64) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(id)).First()
	if err != nil {
		return err
	}

	// 校验状态：只有待收货状态才能确认收货
	if as.Status != consts.AfterSaleStatusBuyerDelivery {
		return fmt.Errorf("售后状态不允许确认收货")
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态为待退款
		newStatus := consts.AfterSaleStatusWaitRefund
		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(id)).Updates(trade.AfterSale{
			Status:      newStatus,
			ReceiveTime: time.Now(),
		}); err != nil {
			return err
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       adminUserId,
			UserType:     consts.UserTypeAdmin,
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  newStatus,
			OperateType:  consts.AfterSaleOperateTypeAdminAgreeReceive,
			Content:      "管理员确认收货",
		}); err != nil {
			return err
		}

		// 3. 更新订单项状态
		if _, err := tx.TradeOrderItem.WithContext(ctx).Where(tx.TradeOrderItem.ID.Eq(as.OrderItemID)).Update(tx.TradeOrderItem.AfterSaleStatus, int32(newStatus)); err != nil {
			return err
		}
		return nil
	})
}

// DeliveryAfterSale 用户退回货物 (App)
func (s *TradeAfterSaleService) DeliveryAfterSale(ctx context.Context, userId int64, req *trade2.AppAfterSaleDeliveryReq) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(req.ID), s.q.AfterSale.UserID.Eq(userId)).First()
	if err != nil {
		return fmt.Errorf("售后单不存在")
	}

	// 校验状态：只有已同意状态才能填写物流信息
	if as.Status != consts.AfterSaleStatusSellerAgree {
		return fmt.Errorf("售后状态不允许填写物流信息")
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态为买家已发货
		newStatus := consts.AfterSaleStatusBuyerDelivery
		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(req.ID)).Updates(trade.AfterSale{
			Status:       newStatus,
			LogisticsID:  req.LogisticsId,
			LogisticsNo:  req.LogisticsNo,
			DeliveryTime: time.Now(),
		}); err != nil {
			return err
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       userId,
			UserType:     consts.UserTypeMember,
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  newStatus,
			OperateType:  consts.AfterSaleOperateTypeMemberDelivery,
			Content:      "用户退回货物",
		}); err != nil {
			return err
		}

		// 3. 更新订单项状态
		if _, err := tx.TradeOrderItem.WithContext(ctx).Where(tx.TradeOrderItem.ID.Eq(as.OrderItemID)).Update(tx.TradeOrderItem.AfterSaleStatus, int32(newStatus)); err != nil {
			return err
		}

		return nil
	})
}

// UpdateAfterSaleRefunded 更新售后单为已退款 (Callback from Pay)
func (s *TradeAfterSaleService) UpdateAfterSaleRefunded(ctx context.Context, afterSaleId int64, payRefundId int64) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(afterSaleId)).First()
	if err != nil {
		return err
	}
	if as.Status == consts.AfterSaleStatusComplete { // Already completed
		return nil
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态为完成
		newStatus := consts.AfterSaleStatusComplete
		// 条件状态转换 + 影响行数校验：重复/乱序的退款回调不得重复更新订单项金额
		result, err := tx.AfterSale.WithContext(ctx).
			Where(tx.AfterSale.ID.Eq(afterSaleId), tx.AfterSale.Status.Eq(as.Status)).
			Updates(trade.AfterSale{
				Status:      newStatus,
				RefundTime:  time.Now(),
				PayRefundID: payRefundId,
			})
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return nil // 已被并发回调处理，幂等返回
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       0, // System
			UserType:     consts.UserTypeAdmin,
			AfterSaleID:  as.ID,
			BeforeStatus: as.Status,
			AfterStatus:  newStatus,
			OperateType:  consts.AfterSaleOperateTypeAdminRefund,
			Content:      "支付退款成功，售后完成",
		}); err != nil {
			return err
		}

		// 3. 更新订单项状态
		if err := s.orderSvc.UpdateOrderItemWhenAfterSaleSuccess(ctx, as.OrderID, as.OrderItemID, as.RefundPrice); err != nil {
			return err
		}
		return nil
	})
}

// UpdateRefunded 更新退款状态 (Unified Facade)
func (s *TradeAfterSaleService) UpdateRefunded(ctx context.Context, req *pay2.PayRefundNotifyReqDTO) error {
	if strings.HasPrefix(req.MerchantRefundId, "order-") {
		orderId, err := strconv.ParseInt(strings.TrimPrefix(req.MerchantRefundId, "order-"), 10, 64)
		if err != nil {
			return err
		}
		return s.orderSvc.UpdatePaidOrderRefunded(ctx, orderId, req.PayRefundId)
	} else {
		afterSaleId, err := strconv.ParseInt(req.MerchantRefundId, 10, 64)
		if err != nil {
			return err
		}
		return s.UpdateAfterSaleRefunded(ctx, afterSaleId, req.PayRefundId)
	}
}

// GetUserAfterSaleCount 获得用户的售后数量 (进行中)
func (s *TradeAfterSaleService) GetUserAfterSaleCount(ctx context.Context, userId int64) (int64, error) {
	// Java 中进行的售后包括：申请中、卖家同意、买家已发货、待退款
	return s.q.AfterSale.WithContext(ctx).
		Where(s.q.AfterSale.UserID.Eq(userId)).
		Where(s.q.AfterSale.Status.In(
			consts.AfterSaleStatusApply,
			consts.AfterSaleStatusSellerAgree,
			consts.AfterSaleStatusBuyerDelivery,
			consts.AfterSaleStatusWaitRefund)).
		Count()
}

// RefuseAfterSale 拒绝收货 (Admin)
func (s *TradeAfterSaleService) RefuseAfterSale(ctx context.Context, adminUserId int64, req *trade2.TradeAfterSaleRefuseReq) error {
	as, err := s.q.AfterSale.WithContext(ctx).Where(s.q.AfterSale.ID.Eq(req.ID)).First()
	if err != nil {
		return fmt.Errorf("售后单不存在")
	}

	// 校验状态：必须是已发货状态才能拒绝收货
	if as.Status != consts.AfterSaleStatusBuyerDelivery {
		return fmt.Errorf("售后状态不是买家已发货，不能拒绝收货")
	}

	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 更新售后单状态为卖家拒绝收货
		if _, err := tx.AfterSale.WithContext(ctx).Where(tx.AfterSale.ID.Eq(req.ID)).Updates(trade.AfterSale{
			Status:        consts.AfterSaleStatusSellerRefuse,
			ReceiveTime:   time.Now(),
			ReceiveReason: req.RefuseMemo,
		}); err != nil {
			return err
		}

		// 2. 记录售后日志
		if err := tx.AfterSaleLog.WithContext(ctx).Create(&trade.AfterSaleLog{
			UserID:       adminUserId,
			UserType:     consts.UserTypeAdmin,
			AfterSaleID:  as.ID,
			BeforeStatus: consts.AfterSaleStatusBuyerDelivery,
			AfterStatus:  consts.AfterSaleStatusSellerRefuse,
			OperateType:  consts.AfterSaleOperateTypeAdminDisagreeReceive,
			Content:      fmt.Sprintf("卖家拒绝收货，原因：%s", req.RefuseMemo),
		}); err != nil {
			return err
		}

		// 3. 更新订单项状态为【未申请】(对齐 Java)
		if _, err := tx.TradeOrderItem.WithContext(ctx).Where(tx.TradeOrderItem.ID.Eq(as.OrderItemID)).Update(tx.TradeOrderItem.AfterSaleStatus, consts.AfterSaleStatusNone); err != nil {
			return err
		}

		return nil
	})
}
