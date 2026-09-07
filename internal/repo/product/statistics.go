package product

import (
	"context"
	"fmt"
	modelProduct "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	modelTrade "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"time"

	productDto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"gorm.io/gorm"
)

// ProductStatisticsRepositoryImpl 商品统计 Repository 实现 - 使用 gorm gen Query
type ProductStatisticsRepositoryImpl struct {
	q  *query.Query
	db *gorm.DB
}

// NewProductStatisticsRepository 创建商品统计 Repository
func NewProductStatisticsRepository(q *query.Query, db *gorm.DB) *ProductStatisticsRepositoryImpl {
	return &ProductStatisticsRepositoryImpl{q: q, db: db}
}

// GetByDateRange 查询指定日期范围的商品统计数据
func (r *ProductStatisticsRepositoryImpl) GetByDateRange(ctx context.Context, beginTime, endTime time.Time) ([]*productDto.ProductStatisticsRespVO, error) {
	ps := r.q.ProductStatistics

	// 查询指定时间范围内的统计数据，按 SPU 聚合
	var results []struct {
		SpuID           int64 `gorm:"column:spu_id"`
		BrowseCount     int64 `gorm:"column:browse_count"`
		FavoriteCount   int64 `gorm:"column:favorite_count"`
		CartCount       int64 `gorm:"column:cart_count"`
		OrderCount      int64 `gorm:"column:order_count"`
		BuyCount        int64 `gorm:"column:buy_count"`
		BuyPrice        int64 `gorm:"column:buy_price"`
		AfterSaleCount  int64 `gorm:"column:after_sale_count"`
		AfterSaleRefund int64 `gorm:"column:after_sale_refund"`
	}

	err := ps.WithContext(ctx).
		Select(
			ps.SpuID,
			ps.BrowseCount.Sum().As("browse_count"),
			ps.FavoriteCount.Sum().As("favorite_count"),
			ps.CartCount.Sum().As("cart_count"),
			ps.OrderCount.Sum().As("order_count"),
			ps.OrderPayCount.Sum().As("buy_count"),
			ps.OrderPayPrice.Sum().As("buy_price"),
			ps.AfterSaleCount.Sum().As("after_sale_count"),
			ps.AfterSaleRefundPrice.Sum().As("after_sale_refund"),
		).
		Where(ps.Time.Between(beginTime, endTime)).
		// Where(ps.Deleted.Eq(false)). // Soft delete handled by GORM
		Group(ps.SpuID).
		Scan(&results)
	if err != nil {
		return nil, err
	}

	// 转换为 VO
	voList := make([]*productDto.ProductStatisticsRespVO, 0, len(results))
	for _, r := range results {
		voList = append(voList, &productDto.ProductStatisticsRespVO{
			SpuID:         r.SpuID,
			BrowseCount:   r.BrowseCount,
			FavoriteCount: r.FavoriteCount,
			BuyCount:      r.BuyCount,
			BuyPrice:      r.BuyPrice,
		})
	}

	return voList, nil
}

// GetSummaryByDateRange 获取指定日期范围的汇总统计数据
func (r *ProductStatisticsRepositoryImpl) GetSummaryByDateRange(ctx context.Context, beginTime, endTime time.Time) (*productDto.ProductStatisticsRespVO, error) {
	ps := r.q.ProductStatistics

	var result struct {
		BrowseCount   int64 `gorm:"column:browse_count"`
		FavoriteCount int64 `gorm:"column:favorite_count"`
		CartCount     int64 `gorm:"column:cart_count"`
		BuyCount      int64 `gorm:"column:buy_count"`
		BuyPrice      int64 `gorm:"column:buy_price"`
		CommentCount  int64 `gorm:"column:comment_count"`
	}

	err := ps.WithContext(ctx).
		Select(
			ps.BrowseCount.Sum().As("browse_count"),
			ps.FavoriteCount.Sum().As("favorite_count"),
			ps.CartCount.Sum().As("cart_count"),
			ps.OrderPayCount.Sum().As("buy_count"),
			ps.OrderPayPrice.Sum().As("buy_price"),
		).
		Where(ps.Time.Between(beginTime, endTime)).
		Scan(&result)
	if err != nil {
		return nil, err
	}

	return &productDto.ProductStatisticsRespVO{
		BrowseCount:   result.BrowseCount,
		FavoriteCount: result.FavoriteCount,
		BuyCount:      result.BuyCount,
		BuyPrice:      result.BuyPrice,
	}, nil
}

// GetPageGroupBySpuId 分页获取按 SPU 分组的统计数据
func (r *ProductStatisticsRepositoryImpl) GetPageGroupBySpuId(ctx context.Context, reqVO *productDto.ProductStatisticsReqVO, pageParam *pagination.PageParam) (*pagination.PageResult[*productDto.ProductStatisticsRespVO], error) {
	// 先获取所有数据，然后内存分页
	list, err := r.GetByDateRange(ctx, reqVO.Times[0], reqVO.Times[1])
	if err != nil {
		return nil, err
	}

	total := int64(len(list))
	start := (pageParam.PageNo - 1) * pageParam.PageSize
	end := start + pageParam.PageSize
	if start > int(total) {
		start = int(total)
	}
	if end > int(total) {
		end = int(total)
	}

	return &pagination.PageResult[*productDto.ProductStatisticsRespVO]{
		List:  list[start:end],
		Total: total,
	}, nil
}

// CountByDateRange 统计指定日期范围内的记录数
func (r *ProductStatisticsRepositoryImpl) CountByDateRange(ctx context.Context, beginTime, endTime time.Time) (int64, error) {
	ps := r.q.ProductStatistics
	return ps.WithContext(ctx).Where(ps.Time.Between(beginTime, endTime)).Count()
}

// StatisticsProductByDateRange 统计指定日期范围内的商品数据并入库
// 对应 Java: ProductStatisticsMapper.selectStatisticsResultPageByTimeBetween
func (r *ProductStatisticsRepositoryImpl) StatisticsProductByDateRange(ctx context.Context, date time.Time, beginTime, endTime time.Time) error {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return fmt.Errorf("trusted tenant required for product statistics")
	}
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Model queries preserve both the tenant guard and BitBool soft deletion.
		scoped := func(m any) *gorm.DB { return tx.Model(m).Where("tenant_id = ?", tenant) }
		// ponytail: paid IDs are held per tenant; stream them if tenant history makes this memory material. Bind batches stay below PostgreSQL limits.
		var paidOrderIDs []int64
		if err := scoped(&modelTrade.TradeOrder{}).Where("pay_status = ?", 1).Pluck("id", &paidOrderIDs).Error; err != nil {
			return err
		}
		// A retry replaces this tenant/day atomically; the unique live key rejects concurrent duplicates.
		if err := scoped(&modelProduct.ProductStatistics{}).Where("time = ?", day).Delete(&modelProduct.ProductStatistics{}).Error; err != nil {
			return err
		}
		var cursor int64
		for {
			var spus []modelProduct.ProductSpu
			if err := scoped(&modelProduct.ProductSpu{}).Where("id > ?", cursor).Order("id").Limit(100).Find(&spus).Error; err != nil {
				return err
			}
			if len(spus) == 0 {
				return nil
			}
			for _, spu := range spus {
				stat := modelProduct.ProductStatistics{SpuID: spu.ID, Time: day}
				stat.TenantID = tenant
				within := func(m any) *gorm.DB {
					return scoped(m).Where("spu_id = ? AND create_time BETWEEN ? AND ?", spu.ID, beginTime, endTime)
				}
				var count int64
				if err := within(&modelProduct.ProductBrowseHistory{}).Count(&count).Error; err != nil {
					return err
				}
				stat.BrowseCount = int(count)
				if err := within(&modelProduct.ProductBrowseHistory{}).Distinct("user_id").Count(&count).Error; err != nil {
					return err
				}
				stat.BrowseUserCount = int(count)
				if err := within(&modelProduct.ProductFavorite{}).Distinct("user_id").Count(&count).Error; err != nil {
					return err
				}
				stat.FavoriteCount = int(count)
				if err := within(&modelTrade.Cart{}).Distinct("user_id").Count(&count).Error; err != nil {
					return err
				}
				stat.CartCount = int(count)
				var totals struct {
					Count int64
					Price int64
				}
				if err := within(&modelTrade.TradeOrderItem{}).Select("COALESCE(SUM(count),0) AS count").Scan(&totals).Error; err != nil {
					return err
				}
				stat.OrderCount = int(totals.Count)
				for start := 0; start < len(paidOrderIDs); start += 1000 {
					end := start + 1000
					if end > len(paidOrderIDs) {
						end = len(paidOrderIDs)
					}
					if err := within(&modelTrade.TradeOrderItem{}).Where("order_id IN ?", paidOrderIDs[start:end]).Select("COALESCE(SUM(count),0) AS count, COALESCE(SUM(pay_price),0) AS price").Scan(&totals).Error; err != nil {
						return err
					}
					stat.OrderPayCount += int(totals.Count)
					stat.OrderPayPrice += int(totals.Price)
				}
				if err := within(&modelTrade.AfterSale{}).Where("refund_time IS NOT NULL").Select("COALESCE(SUM(count),0) AS count, COALESCE(SUM(refund_price),0) AS price").Scan(&totals).Error; err != nil {
					return err
				}
				stat.AfterSaleCount = int(totals.Count)
				stat.AfterSaleRefundPrice = int(totals.Price)
				if stat.BrowseUserCount > 0 {
					stat.BrowseConvertPercent = 100 * stat.OrderPayCount / stat.BrowseUserCount
				}
				if err := tx.Create(&stat).Error; err != nil {
					return err
				}
			}
			cursor = spus[len(spus)-1].ID
		}
	})
}
