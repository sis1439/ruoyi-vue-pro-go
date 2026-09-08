package trade

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	dto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	productmodel "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sync"
	"testing"
)

func TestJavaPriceRoundingAndNegativeItems(t *testing.T) {
	h := NewPriceCalculatorHelper(zap.NewNop())
	items := []TradePriceCalculateItemRespBO{{PayPrice: 56, Selected: true}, {PayPrice: 96, Selected: true}, {PayPrice: 176, Selected: true}}
	require.Equal(t, []int{147, 251, 463}, h.DividePrice(items, 861))
	items = []TradePriceCalculateItemRespBO{{Price: 100, Count: 1, DiscountPrice: 120, Selected: true}, {Price: 5, Count: 1, Selected: true}}
	for i := range items {
		h.RecountPayPrice(&items[i])
	}
	require.Equal(t, -20, items[0].PayPrice)
	require.Equal(t, -15, h.CalculateTotalPayPrice(items, true))
}
func TestJavaAfterSalePostgresCASAndRollback(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenant.WithTenant(context.Background(), 1)
	q := query.Use(db)
	svc := &TradeAfterSaleService{q: q, orderSvc: &TradeOrderUpdateService{q: q, logger: zap.NewNop()}}
	for _, status := range []int{consts.AfterSaleStatusApply, consts.AfterSaleStatusSellerAgree, consts.AfterSaleStatusBuyerDelivery} {
		as := &model.AfterSale{UserID: 1, Status: status, OrderID: 1, OrderItemID: 1}
		require.NoError(t, db.WithContext(ctx).Create(as).Error)
		require.NoError(t, svc.CancelAfterSale(ctx, 1, as.ID))
		var saved model.AfterSale
		require.NoError(t, db.WithContext(ctx).First(&saved, as.ID).Error)
		require.Equal(t, consts.AfterSaleStatusBuyerCancel, saved.Status)
	}
	as := &model.AfterSale{UserID: 1, Status: consts.AfterSaleStatusApply}
	require.NoError(t, db.WithContext(ctx).Create(as).Error)
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("alignment:fail-log", func(tx *gorm.DB) {
		if tx.Statement.Table == "trade_after_sale_log" {
			tx.AddError(errors.New("injected log failure"))
		}
	}))
	require.ErrorContains(t, svc.AgreeAfterSale(ctx, 9, as.ID), "injected log failure")
	require.NoError(t, db.Callback().Create().Remove("alignment:fail-log"))
	var saved model.AfterSale
	require.NoError(t, db.WithContext(ctx).First(&saved, as.ID).Error)
	require.Equal(t, consts.AfterSaleStatusApply, saved.Status)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, status := range []int{consts.AfterSaleStatusBuyerCancel, consts.AfterSaleStatusSellerAgree} {
		wg.Add(1)
		go func(status int) {
			defer wg.Done()
			results <- q.Transaction(func(tx *query.Query) error {
				return svc.updateAfterSale(ctx, tx, as, map[string]interface{}{"status": status})
			})
		}(status)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	require.Equal(t, 1, success)
	other := tenant.WithTenant(context.Background(), 2)
	require.Error(t, svc.CancelAfterSale(other, 1, as.ID))
}
func TestJavaZeroRefundPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenant.WithTenant(context.Background(), 1)
	q := query.Use(db)
	order := &model.TradeOrder{UserID: 1, Status: consts.TradeOrderStatusDelivered}
	require.NoError(t, db.WithContext(ctx).Create(order).Error)
	item := &model.TradeOrderItem{OrderID: order.ID, UserID: 1, AfterSaleStatus: consts.TradeOrderItemAfterSaleStatusApply}
	require.NoError(t, db.WithContext(ctx).Create(item).Error)
	as := &model.AfterSale{UserID: 1, OrderID: order.ID, OrderItemID: item.ID, Status: consts.AfterSaleStatusWaitRefund, RefundPrice: 0}
	require.NoError(t, db.WithContext(ctx).Create(as).Error)
	svc := &TradeAfterSaleService{q: q, orderSvc: &TradeOrderUpdateService{q: q, logger: zap.NewNop()}}
	require.NoError(t, svc.RefundAfterSale(ctx, 9, "127.0.0.1", as.ID))
	require.NoError(t, svc.UpdateAfterSaleRefunded(ctx, as.ID, 0))
	require.NoError(t, db.WithContext(ctx).First(as, as.ID).Error)
	require.Equal(t, consts.AfterSaleStatusComplete, as.Status)
	require.False(t, as.RefundTime.IsZero())
	require.Zero(t, as.PayRefundID)
	var count int64
	require.NoError(t, db.WithContext(ctx).Model(&model.AfterSaleLog{}).Where("after_sale_id = ?", as.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestJavaCancelExcludesAfterSaleStockPostgres(t *testing.T) {
	f := newOrderFixture(t, []int{2, 1}, 0)
	processor := NewCancelOrderProcessor(f.svc.q, f.svc.skuSvc, nil, nil, zap.NewNop())
	req := &OrderHandleRequest{OrderItems: []*model.TradeOrderItem{{SkuID: 1, Count: 1, AfterSaleStatus: consts.TradeOrderItemAfterSaleStatusSuccess}, {SkuID: 2, Count: 1, AfterSaleStatus: consts.TradeOrderItemAfterSaleStatusNone}}}
	require.NoError(t, processor.AfterCancelOrder(f.ctx, req, &OrderHandleResponse{Order: &model.TradeOrder{}}))
	var skus []productmodel.ProductSku
	require.NoError(t, f.db.WithContext(f.ctx).Order("id").Find(&skus).Error)
	require.Equal(t, 2, skus[0].Stock)
	require.Equal(t, 2, skus[1].Stock)
}

func TestJavaOrderDetailDeletedExpressPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenant.WithTenant(context.Background(), 1)
	q := query.Use(db)
	express := &model.TradeDeliveryExpress{Name: "historical"}
	require.NoError(t, db.WithContext(ctx).Create(express).Error)
	delivery := NewDeliveryExpressService(q)
	require.NoError(t, delivery.DeleteDeliveryExpress(ctx, express.ID))
	svc := NewTradeOrderQueryService(q, nil, delivery)
	res := &dto.AppTradeOrderDetailResp{}
	require.NoError(t, svc.FillAppOrderDetail(ctx, &model.TradeOrder{LogisticsID: express.ID}, res))
	require.Empty(t, res.LogisticsName)
}
