package product

import (
	"context"
	"time"

	product2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/samber/lo"
)

type ProductBrowseHistoryService struct {
	q      *query.Query
	spuSvc *ProductSpuService
}

func NewProductBrowseHistoryService(q *query.Query, spuSvc *ProductSpuService) *ProductBrowseHistoryService {
	return &ProductBrowseHistoryService{
		q:      q,
		spuSvc: spuSvc,
	}
}

// CreateBrowseHistory 创建浏览记录 (Async logic handled by caller or here? Java uses @Async. Here just standard sync, or go routine if truly needed. We'll do sync for simplicity first, or go routine inside)
func (s *ProductBrowseHistoryService) CreateBrowseHistory(ctx context.Context, userId, spuId int64) error {
	// Java: Check if exists? Java impl doesn't check existence usually, it just inserts logs.
	// But Wait, DO key is ID.
	// Typically browse history is "Latest view".
	// If allow duplicate?
	// Java ServiceImpl logic:
	// ProductBrowseHistoryDO history = browseHistoryMapper.selectByUserIdAndSpuId(userId, spuId);
	// if (history != null) {
	//     browseHistoryMapper.updateById(new ProductBrowseHistoryDO().setId(history.getId()).setUpdateTime(LocalDateTime.now()));
	//     return;
	// }
	// browseHistoryMapper.insert(new ProductBrowseHistoryDO().setUserId(userId).setSpuId(spuId));

	h := s.q.ProductBrowseHistory
	history, err := h.WithContext(ctx).Where(h.UserID.Eq(userId), h.SpuID.Eq(spuId)).First()
	if err == nil && history != nil {
		// Update time
		_, err := h.WithContext(ctx).Where(h.ID.Eq(history.ID)).Update(h.UpdateTime, time.Now())
		return err
	}

	// Insert new
	newHistory := &product.ProductBrowseHistory{
		UserID:      userId,
		SpuID:       spuId,
		UserDeleted: false,
	}
	return h.WithContext(ctx).Create(newHistory)
}

// HideUserBrowseHistory 隐藏(删除)用户浏览记录
func (s *ProductBrowseHistoryService) HideUserBrowseHistory(ctx context.Context, userId int64, spuIds []int64) error {
	h := s.q.ProductBrowseHistory
	q := h.WithContext(ctx).Where(h.UserID.Eq(userId))

	if len(spuIds) > 0 {
		q = q.Where(h.SpuID.In(spuIds...))
	}

	// Java uses logical delete by updating user_deleted = true
	// "void hideUserBrowseHistory(Long userId, Collection<Long> spuId);"
	// productBrowseHistoryMapper.updateUserDeleted(userId, spuIds, true);

	_, err := q.Update(h.UserDeleted, model.BitBool(true))
	return err
}

// GetBrowseHistoryPage (Admin & App share similar logic but differing reqs?)
// Java has separate ReqVOs but logic is similar. Admin uses BrowseHistoryPageReqVO, App uses AppBrowseHistoryPageReqVO
// GetBrowseHistoryPage (Admin & App share similar logic but differing reqs?)
// Java has separate ReqVOs but logic is similar. Admin uses BrowseHistoryPageReqVO, App uses AppBrowseHistoryPageReqVO
func (s *ProductBrowseHistoryService) GetBrowseHistoryPage(ctx context.Context, r *product2.ProductBrowseHistoryPageReq) (*pagination.PageResult[product2.ProductBrowseHistoryResp], error) {
	h := s.q.ProductBrowseHistory
	q := h.WithContext(ctx)

	if r.UserId > 0 {
		q = q.Where(h.UserID.Eq(r.UserId))
	}
	if r.SpuId > 0 {
		q = q.Where(h.SpuID.Eq(r.SpuId))
	}
	if r.UserDeleted != nil {
		q = q.Where(h.UserDeleted.Eq(model.BitBool(*r.UserDeleted)))
	}
	if len(r.CreateTime) == 2 {
		q = q.Where(h.CreateTime.Between(r.CreateTime[0], r.CreateTime[1]))
	}

	list, total, err := q.Order(h.UpdateTime.Desc()).FindByPage(r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}

	// Fill SPU Info
	spuIds := lo.Map(list, func(item *product.ProductBrowseHistory, _ int) int64 {
		return item.SpuID
	})
	spuList, err := s.spuSvc.GetSpuList(ctx, spuIds)
	if err != nil {
		return nil, err
	}
	spuMap := lo.KeyBy(spuList, func(item *product2.ProductSpuResp) int64 { return item.ID })

	result := lo.Map(list, func(item *product.ProductBrowseHistory, _ int) product2.ProductBrowseHistoryResp {
		r := product2.ProductBrowseHistoryResp{
			ID:     item.ID,
			UserID: item.UserID,
			SpuID:  item.SpuID,
		}
		if spu, ok := spuMap[item.SpuID]; ok {
			r.SpuName = spu.Name
			r.PicURL = spu.PicURL
			r.Price = int64(spu.Price)
			r.SalesCount = spu.SalesCount
			r.Stock = spu.Stock
		}
		return r
	})

	return &pagination.PageResult[product2.ProductBrowseHistoryResp]{
		List:  result,
		Total: total,
	}, nil
}

// GetAppBrowseHistoryPage
func (s *ProductBrowseHistoryService) GetAppBrowseHistoryPage(ctx context.Context, userId int64, r *product2.AppProductBrowseHistoryPageReq) (*pagination.PageResult[product2.AppProductBrowseHistoryResp], error) {
	h := s.q.ProductBrowseHistory
	q := h.WithContext(ctx).Where(h.UserID.Eq(userId)).Where(h.UserDeleted.Eq(model.BitBool(false)))

	list, total, err := q.Order(h.UpdateTime.Desc()).FindByPage(r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}

	spuIds := lo.Map(list, func(item *product.ProductBrowseHistory, _ int) int64 {
		return item.SpuID
	})
	spuList, err := s.spuSvc.GetSpuList(ctx, spuIds)
	if err != nil {
		return nil, err
	}
	spuMap := lo.KeyBy(spuList, func(item *product2.ProductSpuResp) int64 { return item.ID })

	result := lo.Map(list, func(item *product.ProductBrowseHistory, _ int) product2.AppProductBrowseHistoryResp {
		r := product2.AppProductBrowseHistoryResp{
			ID:    item.ID,
			SpuID: item.SpuID,
		}
		if spu, ok := spuMap[item.SpuID]; ok {
			r.SpuName = spu.Name
			r.PicURL = spu.PicURL
			r.Price = int64(spu.Price)
			r.SalesCount = spu.SalesCount
			r.Stock = spu.Stock
		}
		return r
	})

	return &pagination.PageResult[product2.AppProductBrowseHistoryResp]{
		List:  result,
		Total: total,
	}, nil
}
