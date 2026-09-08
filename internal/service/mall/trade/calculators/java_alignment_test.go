package calculators

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	trade "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestJavaCouponZeroAndNullLimit(t *testing.T) {
	c := &CouponPriceCalculator{}
	coupon := &model.PromotionCoupon{DiscountType: consts.DiscountTypePercent, DiscountPercent: 80}
	require.Equal(t, 200, c.getCouponPrice(coupon, 1000))
	zero := 0
	coupon.DiscountLimitPrice = &zero
	require.Zero(t, c.getCouponPrice(coupon, 1000))
	limit := 50
	coupon.DiscountLimitPrice = &limit
	require.Equal(t, 50, c.getCouponPrice(coupon, 1000))
}

func TestJavaRewardActivityScopePostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenant.WithTenant(context.Background(), 1)
	activity := &model.PromotionRewardActivity{Name: "only product one", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour), ProductScope: 2, ProductScopeValues: "[1]", ConditionType: 10, Rules: `[{"limit":100,"discountPrice":20}]`}
	require.NoError(t, db.WithContext(ctx).Create(activity).Error)
	helper := trade.NewPriceCalculatorHelper(zap.NewNop())
	c := NewRewardActivityPriceCalculator(promotion.NewRewardActivityService(query.Use(db)), helper, zap.NewNop())
	resp := &trade.TradePriceCalculateRespBO{Type: consts.TradeOrderTypeNormal, Items: []trade.TradePriceCalculateItemRespBO{{SpuID: 1, SkuID: 1, Price: 100, Count: 1, PayPrice: 100, Selected: true}, {SpuID: 2, SkuID: 2, Price: 100, Count: 1, PayPrice: 100, Selected: true}}}
	require.NoError(t, c.Calculate(ctx, &trade.TradePriceCalculateReqBO{}, resp))
	require.Equal(t, 20, resp.Items[0].DiscountPrice)
	require.Zero(t, resp.Items[1].DiscountPrice)
	require.Equal(t, 80, resp.Items[0].PayPrice)
}
