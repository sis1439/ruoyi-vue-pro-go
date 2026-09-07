package product

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	productModel "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
	"time"
)

func TestPostgresStatisticsTenantSoftDeleteAndRetry(t *testing.T) {
	db := testutil.PostgreSQL(t)
	now := time.Now().Truncate(time.Second)
	spu := productModel.ProductSpu{Name: "stats", TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&spu).Error)
	for _, tenant := range []int64{1, 2} {
		for i := 0; i < 2; i++ {
			history := productModel.ProductBrowseHistory{SpuID: spu.ID, UserID: int64(i + 1), TenantBaseDO: model.TenantBaseDO{TenantID: tenant}}
			require.NoError(t, db.Create(&history).Error)
			if i == 1 {
				require.NoError(t, db.Delete(&history).Error)
			}
		}
	}
	order := tradeModel.TradeOrder{No: "stat-paid", PayStatus: true, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&order).Error)
	for _, tenant := range []int64{1, 2} {
		item := tradeModel.TradeOrderItem{SpuID: spu.ID, OrderID: order.ID, Count: 2, PayPrice: 3_000_000_001, TenantBaseDO: model.TenantBaseDO{TenantID: tenant}}
		require.NoError(t, db.Create(&item).Error)
	}
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	repo := NewProductStatisticsRepository(query.Use(db), db)
	ctx := pkgContext.WithTenant(context.Background(), 1)
	require.Error(t, repo.StatisticsProductByDateRange(context.Background(), now, now.Add(-time.Hour), now.Add(time.Hour)))
	for i := 0; i < 2; i++ {
		require.NoError(t, repo.StatisticsProductByDateRange(ctx, now, now.Add(-time.Hour), now.Add(time.Hour)))
	}
	var stats []productModel.ProductStatistics
	require.NoError(t, db.WithContext(ctx).Find(&stats).Error)
	require.Len(t, stats, 1)
	require.Equal(t, 1, stats[0].BrowseCount)
	require.Equal(t, 2, stats[0].OrderPayCount)
	require.Equal(t, 3_000_000_001, stats[0].OrderPayPrice)
	require.Equal(t, int64(1), stats[0].TenantID)
	var others []productModel.ProductStatistics
	require.NoError(t, db.WithContext(pkgContext.WithTenant(context.Background(), 2)).Find(&others).Error)
	require.Empty(t, others)
}
