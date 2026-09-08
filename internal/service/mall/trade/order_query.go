package trade

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/pkg/area"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	trade2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade/delivery/client"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type TradeOrderQueryService struct {
	q                    *query.Query
	expressClientFactory client.ExpressClientFactory
	deliveryExpressSvc   *DeliveryExpressService
}

func NewTradeOrderQueryService(q *query.Query, expressClientFactory client.ExpressClientFactory, deliveryExpressSvc *DeliveryExpressService) *TradeOrderQueryService {
	return &TradeOrderQueryService{
		q:                    q,
		expressClientFactory: expressClientFactory,
		deliveryExpressSvc:   deliveryExpressSvc,
	}
}

// ... GetOrder, GetOrderPage methods ...
// (Skipping middle methods for brevity in edit, but replace_file_content needs exact match or range.
// I will just replace the struct/constructor first, then the method separately to avoid massive context issues)

// GetOrder 获得交易订单
func (s *TradeOrderQueryService) GetOrder(ctx context.Context, id int64) (*trade.TradeOrder, error) {
	return s.q.TradeOrder.WithContext(ctx).Where(s.q.TradeOrder.ID.Eq(id)).First()
}

// GetOrderPage 获得交易订单分页
func (s *TradeOrderQueryService) GetOrderPage(ctx context.Context, uId int64, r *trade2.AppTradeOrderPageReq) (*pagination.PageResult[*trade.TradeOrder], error) {
	q := s.q.TradeOrder.WithContext(ctx).Where(s.q.TradeOrder.UserID.Eq(uId))

	if r.Status != nil {
		q = q.Where(s.q.TradeOrder.Status.Eq(*r.Status))
	}
	if r.CommentStatus != nil {
		q = q.Where(s.q.TradeOrder.CommentStatus.Eq(model.NewBitBool(*r.CommentStatus)))
	}

	list, total, err := q.Order(s.q.TradeOrder.ID.Desc()).FindByPage(r.GetOffset(), r.PageSize)
	if err != nil {
		return nil, err
	}

	result := &pagination.PageResult[*trade.TradeOrder]{
		List:  list,
		Total: total,
	}
	return result, nil
}

// GetOrderItemListByOrderId 获得交易订单项列表
func (s *TradeOrderQueryService) GetOrderItemListByOrderId(ctx context.Context, orderId int64) ([]*trade.TradeOrderItem, error) {
	return s.q.TradeOrderItem.WithContext(ctx).Where(s.q.TradeOrderItem.OrderID.Eq(orderId)).Find()
}

// GetOrderItemListByOrderIds 获得交易订单项列表
func (s *TradeOrderQueryService) GetOrderItemListByOrderIds(ctx context.Context, orderIds []int64) ([]*trade.TradeOrderItem, error) {
	return s.q.TradeOrderItem.WithContext(ctx).Where(s.q.TradeOrderItem.OrderID.In(orderIds...)).Find()
}

// GetOrderCount 获得交易订单数量
func (s *TradeOrderQueryService) GetOrderCount(ctx context.Context, userId int64, status *int, commentStatus *bool) (int64, error) {
	q := s.q.TradeOrder.WithContext(ctx).Where(s.q.TradeOrder.UserID.Eq(userId))
	if status != nil {
		q = q.Where(s.q.TradeOrder.Status.Eq(*status))
	}
	if commentStatus != nil {
		q = q.Where(s.q.TradeOrder.CommentStatus.Eq(model.NewBitBool(*commentStatus)))
	}
	return q.Count()
}

// GetOrderItem 获得交易订单项
func (s *TradeOrderQueryService) GetOrderItem(ctx context.Context, userId int64, id int64) (*trade.TradeOrderItem, error) {
	return s.q.TradeOrderItem.WithContext(ctx).Where(s.q.TradeOrderItem.ID.Eq(id), s.q.TradeOrderItem.UserID.Eq(userId)).First()
}

// GetOrderPageForAdmin 获得交易订单分页 (Admin)
func (s *TradeOrderQueryService) GetOrderPageForAdmin(ctx context.Context, r *trade2.TradeOrderPageReq) (*pagination.PageResult[*trade.TradeOrder], error) {
	// 1. 构建查询条件
	q := s.buildOrderQuery(ctx, r)

	// 2. 统计总数
	total, err := q.Count()
	if err != nil {
		return nil, err
	}

	// 3. 分页查询
	offset := (r.PageNo - 1) * r.PageSize
	list, err := q.Order(s.q.TradeOrder.ID.Desc()).Offset(offset).Limit(r.PageSize).Find()
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*trade.TradeOrder]{
		List:  list,
		Total: total,
	}, nil
}

// GetExpressTrackList 获得物流轨迹 (App - requires UserId)
func (s *TradeOrderQueryService) GetExpressTrackList(ctx context.Context, id int64, userId int64) ([]*trade2.ExpressTrackRespVO, error) {
	// 查询订单
	order, err := s.q.TradeOrder.WithContext(ctx).Where(s.q.TradeOrder.ID.Eq(id), s.q.TradeOrder.UserID.Eq(userId)).First()
	if err != nil {
		return nil, errors.NewBizError(2002001, "订单不存在") // ORDER_NOT_FOUND code
	}
	return s.getExpressTrackList(ctx, order)
}

// GetExpressTrackListById 获得物流轨迹 (Admin - no UserId check)
func (s *TradeOrderQueryService) GetExpressTrackListById(ctx context.Context, id int64) ([]*trade2.ExpressTrackRespVO, error) {
	// 查询订单
	order, err := s.q.TradeOrder.WithContext(ctx).Where(s.q.TradeOrder.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.NewBizError(2002001, "订单不存在") // ORDER_NOT_FOUND code
	}
	return s.getExpressTrackList(ctx, order)
}

func (s *TradeOrderQueryService) getExpressTrackList(ctx context.Context, order *trade.TradeOrder) ([]*trade2.ExpressTrackRespVO, error) {
	if order.LogisticsID == 0 {
		return []*trade2.ExpressTrackRespVO{}, nil
	}
	// 查询物流公司
	express, err := s.deliveryExpressSvc.GetDeliveryExpress(ctx, order.LogisticsID)
	if err != nil || express == nil {
		return nil, errors.NewBizError(2002015, "物流公司不存在") // EXPRESS_NOT_EXISTS
	}

	// 获得客户端
	expressClient := s.expressClientFactory.GetDefaultExpressClient()
	if expressClient == nil {
		return nil, errors.NewBizError(500, "物流客户端未配置")
	}

	// 查询物流轨迹
	tracks, err := expressClient.GetExpressTrackList(&client.ExpressTrackQueryReqDTO{
		ExpressCode: express.Code,
		LogisticsNo: order.LogisticsNo,
		Phone:       order.ReceiverMobile,
	})
	if err != nil {
		return nil, err
	}

	// Convert to []trade2.ExpressTrackRespVO
	var res []*trade2.ExpressTrackRespVO
	for _, t := range tracks {
		res = append(res, &trade2.ExpressTrackRespVO{
			Time:    t.Time,
			Content: t.Context,
		})
	}
	return res, nil
}

// GetOrderSummary 获得交易订单统计
func (s *TradeOrderQueryService) GetOrderSummary(ctx context.Context, r *trade2.TradeOrderPageReq) (*trade2.TradeOrderSummaryResp, error) {
	// 1. 构建查询条件 (复用逻辑)
	q := s.buildOrderQuery(ctx, r)

	// 2. 定义聚合结果结构
	type AggResult struct {
		RefundStatus int   `gorm:"column:refund_status"` // Group by RefundStatus
		Count        int64 `gorm:"column:count"`         // Count(*)
		Price        int64 `gorm:"column:price"`         // Sum(pay_price)
	}
	var results []AggResult

	// 3. 执行聚合查询
	// Select: refund_status, count(*) as count, sum(pay_price) as price
	err := q.Select(
		s.q.TradeOrder.RefundStatus,
		s.q.TradeOrder.ID.Count().As("count"),
		s.q.TradeOrder.PayPrice.Sum().As("price"),
	).Group(s.q.TradeOrder.RefundStatus).Scan(&results)

	if err != nil {
		return nil, err
	}

	// 4. 聚合到响应结构
	summary := &trade2.TradeOrderSummaryResp{}
	for _, res := range results {
		if res.RefundStatus == 0 { // None (Unrefunded)
			summary.OrderCount += res.Count
			summary.OrderPayPrice += res.Price
		} else { // Any Refund Status (Applied, Successful, etc.)
			summary.AfterSaleCount += res.Count
			summary.AfterSalePrice += res.Price
		}
	}

	return summary, nil
}

// buildOrderQuery 构建订单查询条件
func (s *TradeOrderQueryService) buildOrderQuery(ctx context.Context, r *trade2.TradeOrderPageReq) query.ITradeOrderDo {
	q := s.q.TradeOrder.WithContext(ctx)

	// 1. 用户信息过滤 (Nickname/Mobile) -> 转换为 UserID 列表
	if r.UserNickname != "" || r.UserMobile != "" {
		u := s.q.MemberUser
		uq := u.WithContext(ctx)
		if r.UserNickname != "" {
			uq = uq.Where(u.Nickname.Like("%" + r.UserNickname + "%"))
		}
		if r.UserMobile != "" {
			uq = uq.Where(u.Mobile.Like("%" + r.UserMobile + "%"))
		}
		users, err := uq.Select(u.ID).Find()
		if err == nil {
			var userIds []int64
			for _, user := range users {
				userIds = append(userIds, user.ID)
			}
			// 如果没有匹配的用户，则 UserID IN (NULL)，直接返回空结果查询
			if len(userIds) == 0 {
				q = q.Where(s.q.TradeOrder.ID.Eq(-1)) // Force empty
			} else {
				q = q.Where(s.q.TradeOrder.UserID.In(userIds...))
			}
		}
	}

	// 2. 基础字段过滤
	if r.No != "" {
		q = q.Where(s.q.TradeOrder.No.Like("%" + r.No + "%"))
	}
	if r.UserID != nil {
		q = q.Where(s.q.TradeOrder.UserID.Eq(*r.UserID))
	}
	if r.Type != nil {
		q = q.Where(s.q.TradeOrder.Type.Eq(*r.Type))
	}
	if r.Status != nil {
		q = q.Where(s.q.TradeOrder.Status.Eq(*r.Status))
	}
	if r.CommentStatus != nil {
		q = q.Where(s.q.TradeOrder.CommentStatus.Eq(model.NewBitBool(*r.CommentStatus)))
	}
	if r.PayChannelCode != "" {
		q = q.Where(s.q.TradeOrder.PayChannelCode.Eq(r.PayChannelCode))
	}
	if r.Terminal != nil {
		q = q.Where(s.q.TradeOrder.Terminal.Eq(*r.Terminal))
	}

	// 3. 物流/自提相关
	if r.DeliveryType != nil {
		q = q.Where(s.q.TradeOrder.DeliveryType.Eq(*r.DeliveryType))
	}
	if r.LogisticsID != nil {
		q = q.Where(s.q.TradeOrder.LogisticsID.Eq(*r.LogisticsID))
	}
	if len(r.PickUpStoreIDs) > 0 {
		q = q.Where(s.q.TradeOrder.PickUpStoreID.In(r.PickUpStoreIDs...))
	}
	if r.PickUpVerifyCode != "" {
		q = q.Where(s.q.TradeOrder.PickUpVerifyCode.Like("%" + r.PickUpVerifyCode + "%")) // Fuzzy match? Usually exact but Java might be partial? aligning with 'Like' for safety on codes unless strict required. Java wrapper usually 'eq' for codes. checking... assume Like for verify code is rare, usually Eq. Let's use Eq for verify code to be strict.
		// Re-thinking: Verify code is usually exact match. But if request is "Like", then Like. Admin dashboard often uses partial match.
		// Let's stick to Like for flexibility in admin search.
		q = q.Where(s.q.TradeOrder.PickUpVerifyCode.Like("%" + r.PickUpVerifyCode + "%"))
	}

	// 4. 时间范围
	if len(r.CreateTime) == 2 {
		q = q.Where(s.q.TradeOrder.CreateTime.Between(
			s.parseTime(r.CreateTime[0]),
			s.parseTime(r.CreateTime[1]),
		))
	}

	return q
}

// parseTime 辅助解析时间
func (s *TradeOrderQueryService) parseTime(tStr string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", tStr)
	return t
}

// GetOrderLogListByOrderId 获得交易订单日志列表
func (s *TradeOrderQueryService) GetOrderLogListByOrderId(ctx context.Context, orderId int64) ([]*trade.TradeOrderLog, error) {
	return s.q.TradeOrderLog.WithContext(ctx).Where(s.q.TradeOrderLog.OrderID.Eq(orderId)).Order(s.q.TradeOrderLog.CreateTime.Desc()).Find()
}

func (s *TradeOrderQueryService) FillAppOrderDetail(ctx context.Context, order *trade.TradeOrder, res *trade2.AppTradeOrderDetailResp) error {
	res.PickUpStoreID = order.PickUpStoreID
	res.PickUpVerifyCode = order.PickUpVerifyCode
	res.ReceiverAreaName = area.Format(order.ReceiverAreaID)
	res.PayChannelName = consts.PayChannelName(order.PayChannelCode)
	if order.PayOrderID != nil {
		payment, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(*order.PayOrderID)).First()
		if err != nil {
			return err
		}
		res.PayExpireTime = types.ToJsonDateTimePtr(&payment.ExpireTime)
	}
	if order.LogisticsID > 0 {
		express, err := s.deliveryExpressSvc.GetDeliveryExpress(ctx, order.LogisticsID)
		if err != nil {
			return err
		}
		if express != nil {
			res.LogisticsName = express.Name
		}
	}
	return nil
}
