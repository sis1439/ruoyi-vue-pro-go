package product

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	api "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenantcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"gorm.io/gorm"
)

func productTransactionFixture(t *testing.T) (*gorm.DB, context.Context, *ProductSpuService, *api.ProductSpuSaveReq) {
	t.Helper()
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenantcontext.WithTenant(context.Background(), 1)
	q := query.Use(db)
	values := NewProductPropertyValueService(q)
	properties := NewProductPropertyService(q, values)
	sku := NewProductSkuService(q, properties, values)
	spu := NewProductSpuService(q, sku, NewProductBrandService(q), NewProductCategoryService(q))
	require.NoError(t, db.WithContext(ctx).Create(&product.ProductBrand{ID: 1, Name: "brand"}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&product.ProductCategory{ID: 2, ParentID: 1, Name: "category"}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&product.ProductProperty{ID: 1, Name: "size"}).Error)
	for _, id := range []int64{1, 2, 3} {
		require.NoError(t, db.WithContext(ctx).Create(&product.ProductPropertyValue{ID: id, PropertyID: 1, Name: "value"}).Error)
	}
	yes := true
	req := &api.ProductSpuSaveReq{Name: "initial", Keyword: "keyword", Description: "description", CategoryID: 2, BrandID: 1, PicURL: "pic", SliderPicURLs: []string{"slider"}, Sort: 9, SpecType: &yes, SubCommissionType: &yes, GiveIntegral: 4, VirtualSalesCount: 5, DeliveryTypes: []int{1, 2}, DeliveryTemplateID: 1}
	for _, id := range []int64{1, 2} {
		req.Skus = append(req.Skus, &api.ProductSkuSaveReq{Price: 100, MarketPrice: 200, CostPrice: 50, Stock: 3, Weight: 2, Volume: 4, BarCode: "bar", PicURL: "sku", FirstBrokeragePrice: 2, SecondBrokeragePrice: 1, Properties: []api.ProductSkuPropertyReq{{PropertyID: 1, ValueID: id}}})
	}
	return db, ctx, spu, req
}

func productSnapshot(t *testing.T, db *gorm.DB, ctx context.Context) ([]product.ProductSpu, []product.ProductSku) {
	t.Helper()
	var spus []product.ProductSpu
	var skus []product.ProductSku
	require.NoError(t, db.WithContext(ctx).Unscoped().Order("id").Find(&spus).Error)
	require.NoError(t, db.WithContext(ctx).Unscoped().Order("id").Find(&skus).Error)
	return spus, skus
}

func TestProductTransactionPostgresCreateFailure(t *testing.T) {
	for _, stage := range []string{"second-sku", "commit"} {
		t.Run(stage, func(t *testing.T) {
			db, ctx, svc, req := productTransactionFixture(t)
			pool, err := db.DB()
			require.NoError(t, err)
			if stage == "second-sku" {
				req.Skus[1].BarCode = "reject"
				_, err = pool.Exec(`CREATE FUNCTION reject_sku() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.bar_code = 'reject' THEN RAISE EXCEPTION 'second SKU failure'; END IF; RETURN NEW; END $$`)
				require.NoError(t, err)
				_, err = pool.Exec(`CREATE TRIGGER reject_sku BEFORE INSERT ON product_sku FOR EACH ROW EXECUTE FUNCTION reject_sku()`)
				require.NoError(t, err)
			} else {
				_, err = pool.Exec(`CREATE FUNCTION reject_spu() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'SPU commit failure'; END $$`)
				require.NoError(t, err)
				_, err = pool.Exec(`CREATE CONSTRAINT TRIGGER reject_spu AFTER INSERT ON product_spu DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_spu()`)
				require.NoError(t, err)
			}
			id, err := svc.CreateSpu(ctx, req)
			require.Error(t, err)
			require.ErrorContains(t, err, "failure")
			require.Zero(t, id)
			spus, skus := productSnapshot(t, db, ctx)
			require.Empty(t, spus)
			require.Empty(t, skus)
		})
	}
}

func TestProductTransactionPostgresUpdateAndDelete(t *testing.T) {
	db, ctx, svc, req := productTransactionFixture(t)
	id, err := svc.CreateSpu(ctx, req)
	require.NoError(t, err)
	req.ID = id
	originalSpus, originalSkus := productSnapshot(t, db, ctx)
	require.Len(t, originalSkus, 2)
	for i := range req.Skus {
		req.Skus[i].ID = originalSkus[i].ID
	}
	no := false
	req.SubCommissionType = &no
	req.Name = "updated"
	req.Keyword = ""
	req.Description = ""
	req.Sort = 0
	req.GiveIntegral = 0
	req.VirtualSalesCount = 0
	req.SliderPicURLs = []string{}
	req.DeliveryTemplateID = 0
	for _, sku := range req.Skus {
		sku.Price = 0
		sku.MarketPrice = 0
		sku.CostPrice = 0
		sku.Stock = 0
		sku.Weight = 0
		sku.Volume = 0
		sku.BarCode = ""
		sku.PicURL = ""
		sku.FirstBrokeragePrice = 0
		sku.SecondBrokeragePrice = 0
	}
	n := 0
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:sku-fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "product_sku" {
			n++
			if n == 2 {
				tx.AddError(errors.New("second SKU update failure"))
			}
		}
	}))
	require.ErrorContains(t, svc.UpdateSpu(ctx, req), "second SKU update failure")
	require.Equal(t, 2, n)
	spus, skus := productSnapshot(t, db, ctx)
	require.Equal(t, originalSpus, spus)
	require.Equal(t, originalSkus, skus)
	require.NoError(t, db.Callback().Update().Remove("test:sku-fail"))
	require.NoError(t, svc.UpdateSpu(ctx, req))
	spus, skus = productSnapshot(t, db, ctx)
	require.Equal(t, "updated", spus[0].Name)
	require.Empty(t, spus[0].Keyword)
	require.Empty(t, spus[0].Description)
	require.Empty(t, spus[0].SliderPicURLs)
	require.False(t, bool(spus[0].SubCommissionType))
	require.Zero(t, spus[0].Price)
	require.Zero(t, spus[0].Stock)
	require.Zero(t, spus[0].Sort)
	require.Zero(t, spus[0].GiveIntegral)
	require.Zero(t, spus[0].VirtualSalesCount)
	for _, sku := range skus {
		require.Zero(t, sku.Price)
		require.Zero(t, sku.MarketPrice)
		require.Zero(t, sku.CostPrice)
		require.Zero(t, sku.Stock)
		require.Zero(t, sku.Weight)
		require.Zero(t, sku.Volume)
		require.Empty(t, sku.BarCode)
		require.Empty(t, sku.PicURL)
		require.Zero(t, sku.FirstBrokeragePrice)
		require.Zero(t, sku.SecondBrokeragePrice)
	}
	// Updating one SKU, adding another and removing the omitted SKU is one transaction.
	beforeSpus, beforeSkus := productSnapshot(t, db, ctx)
	req.Skus[1] = &api.ProductSkuSaveReq{Price: 30, Stock: 2, Properties: []api.ProductSkuPropertyReq{{PropertyID: 1, ValueID: 3}}}
	require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register("test:delete-fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "product_sku" {
			tx.AddError(errors.New("SKU deletion failure"))
		}
	}))
	require.ErrorContains(t, svc.UpdateSpu(ctx, req), "SKU deletion failure")
	spus, skus = productSnapshot(t, db, ctx)
	require.Equal(t, beforeSpus, spus)
	require.Equal(t, beforeSkus, skus)
	require.NoError(t, db.Callback().Delete().Remove("test:delete-fail"))
	require.NoError(t, svc.UpdateSpu(ctx, req))
	var live []product.ProductSku
	require.NoError(t, db.WithContext(ctx).Order("id").Find(&live).Error)
	require.Len(t, live, 2)
	require.Equal(t, originalSkus[0].ID, live[0].ID)
	require.NotEqual(t, originalSkus[1].ID, live[1].ID)
	require.NoError(t, db.WithContext(ctx).Model(&product.ProductSpu{}).Where("id = ?", id).Update("status", -1).Error)
	beforeSpus, beforeSkus = productSnapshot(t, db, ctx)
	require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register("test:delete-fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "product_sku" {
			tx.AddError(errors.New("SKU deletion failure"))
		}
	}))
	require.ErrorContains(t, svc.DeleteSpu(ctx, id), "SKU deletion failure")
	spus, skus = productSnapshot(t, db, ctx)
	require.Equal(t, beforeSpus, spus)
	require.Equal(t, beforeSkus, skus)
	require.NoError(t, db.Callback().Delete().Remove("test:delete-fail"))
	require.NoError(t, svc.DeleteSpu(ctx, id))
	spus, skus = productSnapshot(t, db, ctx)
	require.True(t, bool(spus[0].Deleted))
	for _, sku := range skus {
		require.True(t, bool(sku.Deleted))
	}
	var count int64
	require.NoError(t, db.WithContext(ctx).Model(&product.ProductSku{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestProductTransactionPostgresForeignSku(t *testing.T) {
	db, ctx, svc, req := productTransactionFixture(t)
	id, err := svc.CreateSpu(ctx, req)
	require.NoError(t, err)
	req.ID = id
	beforeSpus, beforeSkus := productSnapshot(t, db, ctx)
	other := tenantcontext.WithTenant(context.Background(), 2)
	foreign := &product.ProductSku{SpuID: 999, Price: 99, Stock: 4}
	require.NoError(t, db.WithContext(other).Create(foreign).Error)
	req.Name = "should roll back"
	req.Skus[0].ID = foreign.ID
	require.Error(t, svc.UpdateSpu(ctx, req))
	spus, skus := productSnapshot(t, db, ctx)
	require.Equal(t, beforeSpus, spus)
	require.Equal(t, beforeSkus, skus)
	var saved product.ProductSku
	require.NoError(t, db.WithContext(other).First(&saved, foreign.ID).Error)
	require.Equal(t, 99, saved.Price)
	require.Equal(t, 4, saved.Stock)
	require.Equal(t, int64(999), saved.SpuID)
}

func TestProductTransactionPostgresJoinsCaller(t *testing.T) {
	db, ctx, svc, req := productTransactionFixture(t)
	rollback := errors.New("outer rollback")
	err := repo.InTransaction(ctx, query.Use(db), func(ctx context.Context, _ *query.Query) error {
		id, err := svc.CreateSpu(ctx, req)
		if err != nil {
			return err
		}
		req.ID = id
		req.Name = "within same transaction"
		if err := svc.UpdateSpu(ctx, req); err != nil {
			return err
		}
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	spus, skus := productSnapshot(t, db, ctx)
	require.Empty(t, spus)
	require.Empty(t, skus)
}
