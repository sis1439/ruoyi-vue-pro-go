package promotion

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/samber/lo"
	product "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	prodSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type CombinationActivityService interface {
	// Admin
	CreateCombinationActivity(ctx context.Context, req promotion2.CombinationActivityCreateReq) (int64, error)
	UpdateCombinationActivity(ctx context.Context, req promotion2.CombinationActivityUpdateReq) error
	CloseCombinationActivity(ctx context.Context, id int64) error
	DeleteCombinationActivity(ctx context.Context, id int64) error
	GetCombinationActivity(ctx context.Context, id int64) (*promotion2.CombinationActivityRespVO, error)
	GetCombinationActivityPage(ctx context.Context, req promotion2.CombinationActivityPageReq) (*pagination.PageResult[*promotion2.CombinationActivityPageItemRespVO], error)
	GetCombinationActivityMap(ctx context.Context, ids []int64) (map[int64]*promotion.PromotionCombinationActivity, error)
	GetCombinationActivityListByIds(ctx context.Context, ids []int64) ([]*promotion2.CombinationActivityRespVO, error)
	GetCombinationActivityListByIdsForApp(ctx context.Context, ids []int64) ([]*promotion2.AppCombinationActivityRespVO, error)

	// App
	GetCombinationActivityList(ctx context.Context, count int) ([]*promotion2.AppCombinationActivityRespVO, error)
	GetCombinationActivityPageForApp(ctx context.Context, req pagination.PageParam) (*pagination.PageResult[*promotion2.AppCombinationActivityRespVO], error)
	GetCombinationActivityDetail(ctx context.Context, id int64) (*promotion2.AppCombinationActivityDetailRespVO, error)
	ValidateCombinationActivityCanJoin(ctx context.Context, activityID int64) (*promotion.PromotionCombinationActivity, error)
	// GetMatchCombinationActivityBySpuId 获取指定 SPU 的进行中的拼团活动
	GetMatchCombinationActivityBySpuId(ctx context.Context, spuId int64) (*promotion.PromotionCombinationActivity, error)
}

type combinationActivityService struct {
	q      *query.Query
	spuSvc *prodSvc.ProductSpuService
	skuSvc *prodSvc.ProductSkuService
}

func NewCombinationActivityService(q *query.Query, spuSvc *prodSvc.ProductSpuService, skuSvc *prodSvc.ProductSkuService) CombinationActivityService {
	return &combinationActivityService{
		q:      q,
		spuSvc: spuSvc,
		skuSvc: skuSvc,
	}
}

func (s *combinationActivityService) CreateCombinationActivity(ctx context.Context, req promotion2.CombinationActivityCreateReq) (int64, error) {
	// 1.1 校验商品
	if err := s.validateProducts(ctx, req.Products); err != nil {
		return 0, err
	}
	// 1.2 校验商品冲突
	if err := s.validateProductConflict(ctx, req.SpuID, 0); err != nil {
		return 0, err
	}

	// 2. 插入活动
	activity := &promotion.PromotionCombinationActivity{
		Name:             req.Name,
		SpuID:            req.SpuID,
		TotalLimitCount:  req.TotalLimitCount,
		SingleLimitCount: req.SingleLimitCount,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		UserSize:         req.UserSize,
		VirtualGroup:     req.VirtualGroup,
		Status:           consts.CommonStatusEnable, // 使用 CommonStatusEnable 常量替代魔法数字 1
		LimitDuration:    req.LimitDuration,
	}

	err := s.q.Transaction(func(tx *query.Query) error {
		if err := tx.PromotionCombinationActivity.WithContext(ctx).Create(activity); err != nil {
			return err
		}

		// 3. 插入商品
		products := make([]*promotion.PromotionCombinationProduct, len(req.Products))
		for i, p := range req.Products {
			products[i] = &promotion.PromotionCombinationProduct{
				ActivityID:        activity.ID,
				SpuID:             p.SpuID,
				SkuID:             p.SkuID,
				CombinationPrice:  p.CombinationPrice,
				ActivityStatus:    activity.Status,
				ActivityStartTime: activity.StartTime,
				ActivityEndTime:   activity.EndTime,
			}
		}
		if err := tx.PromotionCombinationProduct.WithContext(ctx).Create(products...); err != nil {
			return err
		}
		return nil
	})

	return activity.ID, err
}

func (s *combinationActivityService) UpdateCombinationActivity(ctx context.Context, req promotion2.CombinationActivityUpdateReq) error {
	// 1. 校验是否存在
	old, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.Eq(req.ID)).First()
	if err != nil {
		return errors.NewBizError(1001006000, "拼团活动不存在")
	}
	if old.Status == consts.CommonStatusDisable { // 使用 CommonStatusDisable 常量替代魔法数字 0
		return errors.NewBizError(1001006010, "拼团活动已关闭，不能修改")
	}

	// 2.1 校验商品
	if err := s.validateProducts(ctx, req.Products); err != nil {
		return err
	}
	// 2.2 校验商品冲突
	if err := s.validateProductConflict(ctx, req.SpuID, req.ID); err != nil {
		return err
	}

	// 3. 更新
	activity := &promotion.PromotionCombinationActivity{
		ID:               req.ID,
		Name:             req.Name,
		SpuID:            req.SpuID,
		TotalLimitCount:  req.TotalLimitCount,
		SingleLimitCount: req.SingleLimitCount,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		UserSize:         req.UserSize,
		VirtualGroup:     req.VirtualGroup,
		LimitDuration:    req.LimitDuration,
	}

	return s.q.Transaction(func(tx *query.Query) error {
		if _, err := tx.PromotionCombinationActivity.WithContext(ctx).Where(tx.PromotionCombinationActivity.ID.Eq(req.ID)).Updates(activity); err != nil {
			return err
		}

		// 删除旧商品
		if _, err := tx.PromotionCombinationProduct.WithContext(ctx).Where(tx.PromotionCombinationProduct.ActivityID.Eq(req.ID)).Delete(); err != nil {
			return err
		}

		// 插入新商品
		products := make([]*promotion.PromotionCombinationProduct, len(req.Products))
		for i, p := range req.Products {
			products[i] = &promotion.PromotionCombinationProduct{
				ActivityID:        activity.ID,
				SpuID:             p.SpuID,
				SkuID:             p.SkuID,
				CombinationPrice:  p.CombinationPrice,
				ActivityStatus:    old.Status,
				ActivityStartTime: activity.StartTime,
				ActivityEndTime:   activity.EndTime,
			}
		}
		if err := tx.PromotionCombinationProduct.WithContext(ctx).Create(products...); err != nil {
			return err
		}
		return nil
	})
}

func (s *combinationActivityService) DeleteCombinationActivity(ctx context.Context, id int64) error {
	activity, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.Eq(id)).First()
	if err != nil {
		return errors.NewBizError(1001006000, "拼团活动不存在")
	}
	if activity.Status == consts.CommonStatusEnable { // 使用 CommonStatusEnable 常量替代魔法数字 1
		return errors.NewBizError(1001006011, "拼团活动进行中，无法删除")
	}
	_, err = s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.Eq(id)).Delete()
	return err
}

// CloseCombinationActivity 关闭拼团活动
func (s *combinationActivityService) CloseCombinationActivity(ctx context.Context, id int64) error {
	q := s.q.PromotionCombinationActivity
	activity, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return errors.NewBizError(1001006000, "拼团活动不存在")
	}
	if activity.Status == consts.CommonStatusDisable { // 使用 CommonStatusDisable 常量，已禁用
		return errors.NewBizError(1001006012, "拼团活动已关闭")
	}
	_, err = q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.Status, consts.CommonStatusDisable) // 使用 CommonStatusDisable 常量
	return err
}

func (s *combinationActivityService) validateProductConflict(ctx context.Context, spuID int64, activityID int64) error {
	q := s.q.PromotionCombinationActivity
	query := q.WithContext(ctx).Where(q.Status.Eq(consts.CommonStatusEnable), q.SpuID.Eq(spuID)) // 使用 CommonStatusEnable 常量 & SpuID match
	if activityID > 0 {
		query = query.Where(q.ID.Neq(activityID))
	}
	count, err := query.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.NewBizError(1001006008, "该商品已存在于其他拼团活动中")
	}
	return nil
}

func (s *combinationActivityService) GetCombinationActivity(ctx context.Context, id int64) (*promotion2.CombinationActivityRespVO, error) {
	activity, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.NewBizError(1001006000, "拼团活动不存在")
	}
	prods, err := s.q.PromotionCombinationProduct.WithContext(ctx).Where(s.q.PromotionCombinationProduct.ActivityID.Eq(id)).Find()
	if err != nil {
		return nil, err
	}

	vo := &promotion2.CombinationActivityRespVO{
		ID:               activity.ID,
		Name:             activity.Name,
		SpuID:            activity.SpuID,
		TotalLimitCount:  activity.TotalLimitCount,
		SingleLimitCount: activity.SingleLimitCount,
		StartTime:        activity.StartTime,
		EndTime:          activity.EndTime,
		UserSize:         activity.UserSize,
		VirtualGroup:     activity.VirtualGroup,
		LimitDuration:    activity.LimitDuration,
		Status:           activity.Status,
		CreateTime:       activity.CreateTime,
		Products:         make([]promotion2.CombinationProductRespVO, len(prods)),
	}

	for i, p := range prods {
		vo.Products[i] = promotion2.CombinationProductRespVO{
			SpuID:             p.SpuID,
			SkuID:             p.SkuID,
			CombinationPrice:  p.CombinationPrice,
			ActivityStatus:    p.ActivityStatus,
			ActivityStartTime: p.ActivityStartTime,
			ActivityEndTime:   p.ActivityEndTime,
		}
	}

	// 补全 SPU 信息
	spu, _ := s.spuSvc.GetSpu(ctx, activity.SpuID)
	if spu != nil {
		vo.SpuName = spu.Name
		vo.PicUrl = spu.PicURL
		vo.MarketPrice = spu.MarketPrice
	}
	// 补全最低价
	if len(prods) > 0 {
		vo.CombinationPrice = lo.Min(lo.Map(prods, func(p *promotion.PromotionCombinationProduct, _ int) int {
			return p.CombinationPrice
		}))
	}

	return vo, nil
}

func (s *combinationActivityService) GetCombinationActivityPage(ctx context.Context, req promotion2.CombinationActivityPageReq) (*pagination.PageResult[*promotion2.CombinationActivityPageItemRespVO], error) {
	q := s.q.PromotionCombinationActivity.WithContext(ctx)
	if req.Name != "" {
		q = q.Where(s.q.PromotionCombinationActivity.Name.Like("%" + req.Name + "%"))
	}
	if req.Status != nil {
		q = q.Where(s.q.PromotionCombinationActivity.Status.Eq(*req.Status))
	}
	if len(req.CreateTime) == 2 && req.CreateTime[0] != nil && req.CreateTime[1] != nil {
		q = q.Where(s.q.PromotionCombinationActivity.CreateTime.Between(*req.CreateTime[0], *req.CreateTime[1]))
	}

	list, total, err := q.Order(s.q.PromotionCombinationActivity.ID.Desc()).FindByPage(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return &pagination.PageResult[*promotion2.CombinationActivityPageItemRespVO]{List: []*promotion2.CombinationActivityPageItemRespVO{}, Total: total}, nil
	}

	// 1. 获取 SPU 信息
	spuIDs := lo.Map(list, func(item *promotion.PromotionCombinationActivity, _ int) int64 { return item.SpuID })
	spuList, _ := s.spuSvc.GetSpuList(ctx, spuIDs)
	spuMap := lo.KeyBy(spuList, func(item *product.ProductSpuResp) int64 { return item.ID })

	// 2. 获取商品最低价
	activityIDs := lo.Map(list, func(item *promotion.PromotionCombinationActivity, _ int) int64 { return item.ID })
	products, _ := s.q.PromotionCombinationProduct.WithContext(ctx).Where(s.q.PromotionCombinationProduct.ActivityID.In(activityIDs...)).Find()
	productGroups := lo.GroupBy(products, func(p *promotion.PromotionCombinationProduct) int64 { return p.ActivityID })
	priceMap := lo.MapValues(productGroups, func(prods []*promotion.PromotionCombinationProduct, _ int64) int {
		return lo.Min(lo.Map(prods, func(p *promotion.PromotionCombinationProduct, _ int) int {
			return p.CombinationPrice
		}))
	})
	prodMap := lo.MapValues(productGroups, func(prods []*promotion.PromotionCombinationProduct, _ int64) []promotion2.CombinationProductRespVO {
		return lo.Map(prods, func(p *promotion.PromotionCombinationProduct, _ int) promotion2.CombinationProductRespVO {
			return promotion2.CombinationProductRespVO{
				SpuID:             p.SpuID,
				SkuID:             p.SkuID,
				CombinationPrice:  p.CombinationPrice,
				ActivityStatus:    p.ActivityStatus,
				ActivityStartTime: p.ActivityStartTime,
				ActivityEndTime:   p.ActivityEndTime,
			}
		})
	})

	// 3. 获取统计数据 (直接查询 Record 表避免循环依赖)
	qr := s.q.PromotionCombinationRecord
	// groupCountMap: 开团组数 (HeadID = 0)
	var groupCounts []struct {
		ActivityID int64
		Count      int64
	}
	_ = qr.WithContext(ctx).Where(qr.ActivityID.In(activityIDs...), qr.HeadID.Eq(0)).
		Select(qr.ActivityID, qr.ID.Count().As("count")).Group(qr.ActivityID).Scan(&groupCounts)
	groupCountMap := lo.SliceToMap(groupCounts, func(item struct {
		ActivityID int64
		Count      int64
	}) (int64, int) {
		return item.ActivityID, int(item.Count)
	})

	// groupSuccessCountMap: 成团组数 (HeadID = 0 && Status = 1)
	var groupSuccessCounts []struct {
		ActivityID int64
		Count      int64
	}
	_ = qr.WithContext(ctx).Where(qr.ActivityID.In(activityIDs...), qr.HeadID.Eq(0), qr.Status.Eq(consts.PromotionCombinationRecordStatusSuccess)). // 使用常量替代魔法数字 1
																			Select(qr.ActivityID, qr.ID.Count().As("count")).Group(qr.ActivityID).Scan(&groupSuccessCounts)
	groupSuccessCountMap := lo.SliceToMap(groupSuccessCounts, func(item struct {
		ActivityID int64
		Count      int64
	}) (int64, int) {
		return item.ActivityID, int(item.Count)
	})

	// recordCountMap: 购买次数 (总记录数)
	var recordCounts []struct {
		ActivityID int64
		Count      int64
	}
	_ = qr.WithContext(ctx).Where(qr.ActivityID.In(activityIDs...)).
		Select(qr.ActivityID, qr.ID.Count().As("count")).Group(qr.ActivityID).Scan(&recordCounts)
	recordCountMap := lo.SliceToMap(recordCounts, func(item struct {
		ActivityID int64
		Count      int64
	}) (int64, int) {
		return item.ActivityID, int(item.Count)
	})

	// 4. 组合结果
	result := make([]*promotion2.CombinationActivityPageItemRespVO, len(list))
	for i, item := range list {
		vo := &promotion2.CombinationActivityPageItemRespVO{
			CombinationActivityRespVO: promotion2.CombinationActivityRespVO{
				ID:               item.ID,
				Name:             item.Name,
				SpuID:            item.SpuID,
				TotalLimitCount:  item.TotalLimitCount,
				SingleLimitCount: item.SingleLimitCount,
				StartTime:        item.StartTime,
				EndTime:          item.EndTime,
				UserSize:         item.UserSize,
				VirtualGroup:     item.VirtualGroup,
				LimitDuration:    item.LimitDuration,
				Status:           item.Status,
				CreateTime:       item.CreateTime,
				CombinationPrice: priceMap[item.ID],
				Products:         prodMap[item.ID],
			},
		}
		// SPU 信息
		if spu, ok := spuMap[item.SpuID]; ok {
			vo.SpuName = spu.Name
			vo.PicUrl = spu.PicURL
			vo.MarketPrice = spu.MarketPrice
		}
		// 统计字段
		vo.GroupCount = groupCountMap[item.ID]
		vo.GroupSuccessCount = groupSuccessCountMap[item.ID]
		vo.RecordCount = recordCountMap[item.ID]

		result[i] = vo
	}
	return &pagination.PageResult[*promotion2.CombinationActivityPageItemRespVO]{
		List:  result,
		Total: total,
	}, nil
}

func (s *combinationActivityService) GetCombinationActivityMap(ctx context.Context, ids []int64) (map[int64]*promotion.PromotionCombinationActivity, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	list, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	return lo.KeyBy(list, func(item *promotion.PromotionCombinationActivity) int64 { return item.ID }), nil
}

// GetCombinationActivityListByIds 获得拼团活动列表，基于活动编号数组
// Java: CombinationActivityController#getCombinationActivityListByIds
func (s *combinationActivityService) GetCombinationActivityListByIds(ctx context.Context, ids []int64) ([]*promotion2.CombinationActivityRespVO, error) {
	if len(ids) == 0 {
		return []*promotion2.CombinationActivityRespVO{}, nil
	}

	// 1. 获得开启的活动列表
	list, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	enabledList := lo.Filter(list, func(activity *promotion.PromotionCombinationActivity, _ int) bool {
		return activity.Status != consts.CommonStatusDisable
	})
	if len(enabledList) == 0 {
		return []*promotion2.CombinationActivityRespVO{}, nil
	}

	// 2. 获取 Product 列表
	activityIds := lo.Map(enabledList, func(activity *promotion.PromotionCombinationActivity, _ int) int64 { return activity.ID })
	products, err := s.q.PromotionCombinationProduct.WithContext(ctx).
		Where(s.q.PromotionCombinationProduct.ActivityID.In(activityIds...)).
		Find()
	if err != nil {
		return nil, err
	}

	// 3. 获取 SPU 列表
	spuIds := lo.Map(enabledList, func(activity *promotion.PromotionCombinationActivity, _ int) int64 { return activity.SpuID })
	spuList, err := s.spuSvc.GetSpuList(ctx, spuIds)
	if err != nil {
		return nil, err
	}
	spuMap := lo.KeyBy(spuList, func(spu *product.ProductSpuResp) int64 { return spu.ID })

	// 4. 组合返回数据
	productGroups := lo.GroupBy(products, func(p *promotion.PromotionCombinationProduct) int64 { return p.ActivityID })
	productMap := lo.MapValues(productGroups, func(prods []*promotion.PromotionCombinationProduct, _ int64) []promotion2.CombinationProductRespVO {
		return lo.Map(prods, func(p *promotion.PromotionCombinationProduct, _ int) promotion2.CombinationProductRespVO {
			return promotion2.CombinationProductRespVO{
				SpuID:             p.SpuID,
				SkuID:             p.SkuID,
				CombinationPrice:  p.CombinationPrice,
				ActivityStatus:    p.ActivityStatus,
				ActivityStartTime: p.ActivityStartTime,
				ActivityEndTime:   p.ActivityEndTime,
			}
		})
	})

	result := lo.Map(enabledList, func(activity *promotion.PromotionCombinationActivity, _ int) *promotion2.CombinationActivityRespVO {
		vo := &promotion2.CombinationActivityRespVO{
			ID:               activity.ID,
			Name:             activity.Name,
			SpuID:            activity.SpuID,
			TotalLimitCount:  activity.TotalLimitCount,
			SingleLimitCount: activity.SingleLimitCount,
			StartTime:        activity.StartTime,
			EndTime:          activity.EndTime,
			UserSize:         activity.UserSize,
			VirtualGroup:     activity.VirtualGroup,
			LimitDuration:    activity.LimitDuration,
			Status:           activity.Status,
			CreateTime:       activity.CreateTime,
			Products:         productMap[activity.ID],
		}
		// 补全 SPU 信息
		if spu, ok := spuMap[activity.SpuID]; ok {
			vo.SpuName = spu.Name
			vo.PicUrl = spu.PicURL
			vo.MarketPrice = spu.MarketPrice
		}
		// 补全最低价
		if len(vo.Products) > 0 {
			vo.CombinationPrice = lo.Min(lo.Map(vo.Products, func(p promotion2.CombinationProductRespVO, _ int) int {
				return p.CombinationPrice
			}))
		}
		return vo
	})

	return result, nil
}

func (s *combinationActivityService) GetCombinationActivityListByIdsForApp(ctx context.Context, ids []int64) ([]*promotion2.AppCombinationActivityRespVO, error) {
	if len(ids) == 0 {
		return []*promotion2.AppCombinationActivityRespVO{}, nil
	}
	list, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	return s.buildAppActivityList(ctx, list)
}

func (s *combinationActivityService) GetCombinationActivityList(ctx context.Context, count int) ([]*promotion2.AppCombinationActivityRespVO, error) {
	q := s.q.PromotionCombinationActivity
	list, err := q.WithContext(ctx).
		Where(q.Status.Eq(consts.CommonStatusEnable)). // 使用 CommonStatusEnable 常量替代魔法数字
		Order(q.ID.Desc()).                            // Usually Sort desc
		Limit(count).
		Find()
	if err != nil {
		return nil, err
	}

	return s.buildAppActivityList(ctx, list)
}

func (s *combinationActivityService) GetCombinationActivityPageForApp(ctx context.Context, p pagination.PageParam) (*pagination.PageResult[*promotion2.AppCombinationActivityRespVO], error) {
	q := s.q.PromotionCombinationActivity
	list, total, err := q.WithContext(ctx).
		Where(q.Status.Eq(consts.CommonStatusEnable)). // 使用 CommonStatusEnable 常量替代魔法数字
		Order(q.ID.Desc()).
		FindByPage(p.GetOffset(), p.GetLimit())
	if err != nil {
		return nil, err
	}

	vos, err := s.buildAppActivityList(ctx, list)
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*promotion2.AppCombinationActivityRespVO]{
		List:  vos,
		Total: total,
	}, nil
}

func (s *combinationActivityService) GetCombinationActivityDetail(ctx context.Context, id int64) (*promotion2.AppCombinationActivityDetailRespVO, error) {
	activity, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.NewBizError(1001006000, "拼团活动不存在")
	}
	if activity.Status != consts.CommonStatusEnable { // 使用 CommonStatusEnable 常量替代魔法数字
		return nil, errors.NewBizError(1001006001, "拼团活动已关闭")
	}

	prods, err := s.q.PromotionCombinationProduct.WithContext(ctx).Where(s.q.PromotionCombinationProduct.ActivityID.Eq(id)).Find()
	if err != nil {
		return nil, err
	}

	// 成功的拼团数量 (Status = 1 && HeadID = 0)
	qr := s.q.PromotionCombinationRecord
	successCount, _ := qr.WithContext(ctx).Where(qr.ActivityID.Eq(id), qr.HeadID.Eq(0), qr.Status.Eq(consts.PromotionCombinationRecordStatusSuccess)). // 使用常量替代魔法数字 1
																				Count() // 使用常量替代魔法数字 1

	detailVo := &promotion2.AppCombinationActivityDetailRespVO{
		ID:               activity.ID,
		Name:             activity.Name,
		Status:           activity.Status,
		StartTime:        types.ToJsonDateTimePtr(&activity.StartTime),
		EndTime:          types.ToJsonDateTimePtr(&activity.EndTime),
		UserSize:         activity.UserSize,
		SuccessCount:     int(successCount),
		SpuID:            activity.SpuID,
		TotalLimitCount:  activity.TotalLimitCount,
		SingleLimitCount: activity.SingleLimitCount,
		Products:         make([]promotion2.AppCombinationActivityDetailProduct, len(prods)),
	}

	for i, p := range prods {
		detailVo.Products[i] = promotion2.AppCombinationActivityDetailProduct{
			SkuID:            p.SkuID,
			CombinationPrice: p.CombinationPrice,
		}
	}

	return detailVo, nil
}

func (s *combinationActivityService) ValidateCombinationActivityCanJoin(ctx context.Context, activityID int64) (*promotion.PromotionCombinationActivity, error) {
	activity, err := s.q.PromotionCombinationActivity.WithContext(ctx).Where(s.q.PromotionCombinationActivity.ID.Eq(activityID)).First()
	if err != nil {
		return nil, errors.NewBizError(1001006000, "拼团活动不存在")
	}
	if activity.Status != consts.CommonStatusEnable {
		return nil, errors.NewBizError(1001006001, "拼团活动已关闭")
	}
	now := time.Now()
	if now.Before(activity.StartTime) {
		return nil, errors.NewBizError(1001006002, "拼团活动未开始")
	}
	if now.After(activity.EndTime) {
		return nil, errors.NewBizError(1001006003, "拼团活动已结束")
	}
	return activity, nil
}

func (s *combinationActivityService) GetMatchCombinationActivityBySpuId(ctx context.Context, spuId int64) (*promotion.PromotionCombinationActivity, error) {
	now := time.Now()
	q := s.q.PromotionCombinationActivity
	return q.WithContext(ctx).
		Where(q.SpuID.Eq(spuId)).
		Where(q.Status.Eq(consts.CommonStatusEnable)).
		Where(q.StartTime.Lt(now)).
		Where(q.EndTime.Gt(now)).
		First()
}

func (s *combinationActivityService) validateProducts(ctx context.Context, products []promotion2.CombinationProductBaseVO) error {
	for _, p := range products {
		if _, err := s.spuSvc.GetSpu(ctx, p.SpuID); err != nil {
			return err
		}
		if _, err := s.skuSvc.GetSku(ctx, p.SkuID); err != nil {
			return err
		}
	}
	return nil
}

func (s *combinationActivityService) buildAppActivityList(ctx context.Context, list []*promotion.PromotionCombinationActivity) ([]*promotion2.AppCombinationActivityRespVO, error) {
	if len(list) == 0 {
		return []*promotion2.AppCombinationActivityRespVO{}, nil
	}
	spuIds := lo.Map(list, func(item *promotion.PromotionCombinationActivity, _ int) int64 {
		return item.SpuID
	})
	spuList, err := s.spuSvc.GetSpuList(ctx, spuIds)
	if err != nil {
		return nil, err
	}
	spuMap := lo.KeyBy(spuList, func(item *product.ProductSpuResp) int64 {
		return item.ID
	})

	activityIds := lo.Map(list, func(item *promotion.PromotionCombinationActivity, _ int) int64 {
		return item.ID
	})
	products, _ := s.q.PromotionCombinationProduct.WithContext(ctx).Where(s.q.PromotionCombinationProduct.ActivityID.In(activityIds...)).Find()
	productGroups := lo.GroupBy(products, func(p *promotion.PromotionCombinationProduct) int64 { return p.ActivityID })
	priceMap := lo.MapValues(productGroups, func(prods []*promotion.PromotionCombinationProduct, _ int64) int {
		return lo.Min(lo.Map(prods, func(p *promotion.PromotionCombinationProduct, _ int) int {
			return p.CombinationPrice
		}))
	})

	result := make([]*promotion2.AppCombinationActivityRespVO, 0, len(list))
	for _, item := range list {
		// ✅ 核心修复: 过滤无效 SPU (不存在或非上架状态)
		spu, ok := spuMap[item.SpuID]
		if !ok || spu.Status != consts.ProductSpuStatusEnable {
			continue
		}
		result = append(result, &promotion2.AppCombinationActivityRespVO{
			ID:               item.ID,
			Name:             item.Name,
			UserSize:         item.UserSize,
			SpuID:            item.SpuID,
			SpuName:          spu.Name,
			PicUrl:           spu.PicURL,
			MarketPrice:      spu.MarketPrice,
			CombinationPrice: priceMap[item.ID],
		})
	}
	return result, nil
}
