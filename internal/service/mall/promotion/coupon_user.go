package promotion

import (
	"context"
	"errors"
	"time"

	"github.com/samber/lo"
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"gorm.io/gorm"
)

type CouponUserService struct {
	q *query.Query
}

func NewCouponUserService(q *query.Query) *CouponUserService {
	return &CouponUserService{
		q: q,
	}
}

// TakeCoupon 用户领取优惠券 (对齐 Java: AppCouponController.takeCoupon)
// 返回值: canTakeAgain - 是否可继续领取
func (s *CouponUserService) TakeCoupon(ctx context.Context, userId int64, req *promotion2.AppCouponTakeReq) (bool, error) {
	// 1. 校验模板
	template, err := s.q.PromotionCouponTemplate.WithContext(ctx).Where(s.q.PromotionCouponTemplate.ID.Eq(int64(req.TemplateID))).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("优惠券模板不存在")
		}
		return false, err
	}

	// 注意：Java版本没有检查模板状态，为保持兼容性，这里也不检查
	// if template.Status != promotion.CommonStatusEnable { // 启用状态
	//     return false, errors.New("优惠券模板已禁用")
	// }
	if template.TotalCount > 0 && template.TakeCount >= template.TotalCount {
		return false, errors.New("优惠券已领完")
	}

	// 2. 检查领取限制
	currentTakeCount := int64(0)
	if template.TakeLimitCount > 0 {
		currentTakeCount, err = s.q.PromotionCoupon.WithContext(ctx).Where(s.q.PromotionCoupon.TemplateID.Eq(template.ID), s.q.PromotionCoupon.UserID.Eq(userId)).Count()
		if err != nil {
			return false, err
		}
		if int(currentTakeCount) >= template.TakeLimitCount {
			return false, errors.New("超出领取限制")
		}
	}

	// 3. 计算有效期
	var startTime, endTime time.Time
	now := time.Now()
	switch template.ValidityType {
	case consts.CouponValidityTypeDate: // 固定日期
		if template.ValidEndTime == nil || now.After(*template.ValidEndTime) {
			return false, errors.New("优惠券已过期")
		}
		startTime = *template.ValidStartTime
		endTime = *template.ValidEndTime
	case consts.CouponValidityTypeTerm: // 领取后N天
		startTime = now.AddDate(0, 0, template.FixedStartTerm)
		endTime = startTime.AddDate(0, 0, template.FixedEndTerm)
	}

	// 4. 创建优惠券 (对齐Java CouponDO字段)
	takeType := template.TakeType
	if takeType == 0 {
		takeType = consts.CouponTakeTypeUser // 兜底：如果模板未设置领取方式，默认为手动领取
	}
	coupon := &promotion.PromotionCoupon{
		TemplateID:         template.ID,
		Name:               template.Name,
		Status:             consts.CouponStatusUnused, // 未使用
		UserID:             userId,
		TakeType:           takeType,
		UsePrice:           template.UsePriceMin, // 使用金额限制
		ValidStartTime:     startTime,
		ValidEndTime:       endTime,
		ProductScope:       template.ProductScope,       // 商品范围
		ProductScopeValues: template.ProductScopeValues, // 商品范围值
		DiscountType:       template.DiscountType,
		DiscountPrice:      template.DiscountPrice,
		DiscountPercent:    template.DiscountPercent,
		DiscountLimitPrice: template.DiscountLimitPrice,
	}

	err = s.q.Transaction(func(tx *query.Query) error {
		// 增加模板领取数量
		if _, err := tx.PromotionCouponTemplate.WithContext(ctx).Where(tx.PromotionCouponTemplate.ID.Eq(template.ID)).UpdateSimple(tx.PromotionCouponTemplate.TakeCount.Add(1)); err != nil {
			return err
		}
		// 保存优惠券
		if err := tx.PromotionCoupon.WithContext(ctx).Create(coupon); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	// 5. 检查是否可以继续领取 (对齐 Java 逻辑)
	canTakeAgain := true
	if template.TakeLimitCount > 0 {
		// 领取后数量 +1
		canTakeAgain = int(currentTakeCount)+1 < template.TakeLimitCount
	}

	return canTakeAgain, nil
}

// GetCouponPage 用户优惠券分页 (对齐 Java: AppCouponController.getCouponPage)
func (s *CouponUserService) GetCouponPage(ctx context.Context, userId int64, req *promotion2.AppCouponPageReq) (*pagination.PageResult[*promotion2.AppCouponResp], error) {
	q := s.q.PromotionCoupon.WithContext(ctx).Where(s.q.PromotionCoupon.UserID.Eq(userId))
	if req.Status != nil {
		q = q.Where(s.q.PromotionCoupon.Status.Eq(*req.Status))
	}

	q = q.Order(s.q.PromotionCoupon.ID.Desc()) // 按 ID 降序排列，对齐 Java 实现

	result, count, err := q.FindByPage(int((req.PageNo-1)*req.PageSize), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	// 转换为响应对象
	list := lo.Map(result, func(coupon *promotion.PromotionCoupon, _ int) *promotion2.AppCouponResp {
		return &promotion2.AppCouponResp{
			ID:                 coupon.ID,
			Name:               coupon.Name,
			Status:             coupon.Status,
			UsePrice:           coupon.UsePrice,
			ProductScope:       coupon.ProductScope,
			ProductScopeValues: coupon.ProductScopeValues,
			ValidStartTime:     coupon.ValidStartTime.UnixMilli(),
			ValidEndTime:       coupon.ValidEndTime.UnixMilli(),
			DiscountType:       coupon.DiscountType,
			DiscountPercent:    coupon.DiscountPercent,
			DiscountPrice:      coupon.DiscountPrice,
			DiscountLimitPrice: coupon.DiscountLimitPrice,
		}
	})

	return &pagination.PageResult[*promotion2.AppCouponResp]{
		List:  list,
		Total: count,
	}, nil
}

// GetCoupon 获得单个优惠券 (对齐 Java: CouponService.getCoupon)
func (s *CouponUserService) GetCoupon(ctx context.Context, userId int64, couponId int64) (*promotion.PromotionCoupon, error) {
	coupon, err := s.q.PromotionCoupon.WithContext(ctx).
		Where(s.q.PromotionCoupon.ID.Eq(couponId)).
		Where(s.q.PromotionCoupon.UserID.Eq(userId)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 返回 null 对齐 Java
		}
		return nil, err
	}
	return coupon, nil
}

// GetUnusedCouponCount 获得未使用的优惠劵数量 (对齐 Java: CouponService.getUnusedCouponCount)
func (s *CouponUserService) GetUnusedCouponCount(ctx context.Context, userId int64) (int64, error) {
	return s.q.PromotionCoupon.WithContext(ctx).
		Where(s.q.PromotionCoupon.UserID.Eq(userId)).
		Where(s.q.PromotionCoupon.Status.Eq(consts.CouponStatusUnused)). // 未使用
		Count()
}

// GetUnusedCouponList 获取用户所有未使用的优惠券列表
// 对齐 Java: couponApi.getCouponListByUserId(userId, CouponStatusEnum.UNUSED.getStatus())
func (s *CouponUserService) GetUnusedCouponList(ctx context.Context, userId int64) ([]*promotion.PromotionCoupon, error) {
	return s.q.PromotionCoupon.WithContext(ctx).
		Where(s.q.PromotionCoupon.UserID.Eq(userId)).
		Where(s.q.PromotionCoupon.Status.Eq(consts.CouponStatusUnused)). // 未使用
		Find()
}

// CalculateCoupon 计算优惠券金额
func (s *CouponUserService) CalculateCoupon(ctx context.Context, userId int64, couponId int64, price int64, spuIDs []int64, categoryIDs []int64) (int64, error) {
	coupon, err := s.q.PromotionCoupon.WithContext(ctx).Where(s.q.PromotionCoupon.ID.Eq(couponId), s.q.PromotionCoupon.UserID.Eq(userId)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("优惠券不存在")
		}
		return 0, err
	}

	// Check Validity
	if coupon.Status != consts.CouponStatusUnused {
		return 0, errors.New("优惠券不可用")
	}
	now := time.Now()
	if now.Before(coupon.ValidStartTime) || now.After(coupon.ValidEndTime) {
		return 0, errors.New("优惠券不在有效期内")
	}

	// Retrieve Template for Product Scope
	template, err := s.q.PromotionCouponTemplate.WithContext(ctx).Where(s.q.PromotionCouponTemplate.ID.Eq(coupon.TemplateID)).First()
	if err != nil {
		return 0, err
	}

	// Check Min Price
	if price < int64(template.UsePriceMin) {
		return 0, errors.New("未满足使用金额")
	}

	// Check Scope
	if !s.checkScope(template, spuIDs, categoryIDs) {
		return 0, errors.New("不满足商品适用范围")
	}

	// Calculate Amount
	discount := int64(0)
	switch coupon.DiscountType {
	case consts.DiscountTypePrice: // 满减
		discount = int64(coupon.DiscountPrice)
	case consts.DiscountTypePercent: // 折扣
		discount = price * int64(coupon.DiscountPercent) / 100
		if coupon.DiscountLimitPrice > 0 && discount > int64(coupon.DiscountLimitPrice) {
			discount = int64(coupon.DiscountLimitPrice)
		}
	}

	if discount > price {
		discount = price
	}

	return discount, nil
}

// UseCoupon 核销优惠券
func (s *CouponUserService) UseCoupon(ctx context.Context, userId int64, couponId int64, orderId int64) error {
	q := repo.QueryFromContext(ctx, s.q)
	coupon, err := q.PromotionCoupon.WithContext(ctx).Where(q.PromotionCoupon.ID.Eq(couponId), q.PromotionCoupon.UserID.Eq(userId)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("优惠券不存在")
		}
		return err
	}

	if coupon.Status != consts.CouponStatusUnused {
		return errors.New("优惠券不可用")
	}
	now := time.Now()
	// Strict validity check again? Maybe lenient if already checked in Price Calc?
	// Consistent behavior: check again.
	if now.Before(coupon.ValidStartTime) || now.After(coupon.ValidEndTime) {
		return errors.New("优惠券不在有效期内")
	}

	// Update Status
	// Use map for updates to include UsedTime
	info, err := q.PromotionCoupon.WithContext(ctx).Where(q.PromotionCoupon.ID.Eq(couponId), q.PromotionCoupon.UserID.Eq(userId), q.PromotionCoupon.Status.Eq(consts.CouponStatusUnused), q.PromotionCoupon.ValidStartTime.Lte(now), q.PromotionCoupon.ValidEndTime.Gte(now)).Updates(map[string]interface{}{
		"status":       consts.CouponStatusUsed, // 已使用
		"use_order_id": orderId,
		"use_time":     now,
	})
	if err == nil && info.RowsAffected != 1 {
		return errors.New("优惠券已被使用或已失效")
	}
	return err
}

// checkScope 检查适用范围
func (s *CouponUserService) checkScope(t *promotion.PromotionCouponTemplate, spuIDs []int64, categoryIDs []int64) bool {
	if t.ProductScope == consts.ProductScopeAll { // 全部商品
		return true
	}
	if t.ProductScope == consts.ProductScopeCategory { // 指定品类
		if len(t.ProductScopeValues) == 0 {
			return true
		}
		for _, v := range t.ProductScopeValues {
			for _, cid := range categoryIDs {
				if v == cid {
					return true
				}
			}
		}
		return false
	}
	if t.ProductScope == consts.ProductScopeSpu { // 指定商品
		if len(t.ProductScopeValues) == 0 {
			return true
		}
		for _, v := range t.ProductScopeValues {
			for _, sid := range spuIDs {
				if v == sid {
					return true
				}
			}
		}
		return false
	}
	return true
}

// GetCouponMatchList 获取匹配的优惠券
func (s *CouponUserService) GetCouponMatchList(ctx context.Context, userId int64, price int64, spuIDs []int64, categoryIDs []int64) ([]promotion.PromotionCoupon, error) {
	// Fetch all valid coupons
	now := time.Now()
	coupons, err := s.q.PromotionCoupon.WithContext(ctx).
		Where(s.q.PromotionCoupon.UserID.Eq(userId)).
		Where(s.q.PromotionCoupon.Status.Eq(consts.CouponStatusUnused)). // 未使用
		Where(s.q.PromotionCoupon.ValidStartTime.Lt(now)).
		Where(s.q.PromotionCoupon.ValidEndTime.Gt(now)).
		Find()
	if err != nil {
		return nil, err
	}

	var matched []promotion.PromotionCoupon
	templateIDs := make([]int64, 0)
	for _, c := range coupons {
		templateIDs = append(templateIDs, c.TemplateID)
	}
	if len(templateIDs) == 0 {
		return matched, nil
	}

	templates, err := s.q.PromotionCouponTemplate.WithContext(ctx).Where(s.q.PromotionCouponTemplate.ID.In(templateIDs...)).Find()
	if err != nil {
		return nil, err
	}
	tmplMap := make(map[int64]*promotion.PromotionCouponTemplate)
	for _, t := range templates {
		tmplMap[t.ID] = t
	}

	for _, c := range coupons {
		t, ok := tmplMap[c.TemplateID]
		if !ok {
			continue
		}
		if price < int64(t.UsePriceMin) {
			continue
		}
		if !s.checkScope(t, spuIDs, categoryIDs) {
			continue
		}
		matched = append(matched, *c)
	}

	// ... Return matched coupons ...
	return matched, nil
}

// ReturnCoupon 退还优惠券
func (s *CouponUserService) ReturnCoupon(ctx context.Context, userId int64, couponId int64) error {
	coupon, err := s.q.PromotionCoupon.WithContext(ctx).Where(s.q.PromotionCoupon.ID.Eq(couponId), s.q.PromotionCoupon.UserID.Eq(userId)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("优惠券不存在")
		}
		return err
	}

	if coupon.Status != consts.CouponStatusUsed { // 已使用
		return errors.New("优惠券状态错误，无法退还")
	}

	// Update Status to Unused (1) and clear Usage info
	updates := map[string]interface{}{
		"status":       consts.CouponStatusUnused,
		"use_order_id": nil,
		"use_time":     nil,
	}
	_, err = s.q.PromotionCoupon.WithContext(ctx).Where(s.q.PromotionCoupon.ID.Eq(couponId)).Updates(updates)
	return err
}

// ReturnCouponForOrder only releases the coupon held by this order.
func (s *CouponUserService) ReturnCouponForOrder(ctx context.Context, userId, couponId, orderId int64) error {
	q := repo.QueryFromContext(ctx, s.q)
	c := q.PromotionCoupon
	info, err := c.WithContext(ctx).Where(c.ID.Eq(couponId), c.UserID.Eq(userId), c.Status.Eq(consts.CouponStatusUsed), c.UseOrderID.Eq(orderId)).Updates(map[string]interface{}{
		"status": consts.CouponStatusUnused, "use_order_id": 0, "use_time": nil,
	})
	if err == nil && info.RowsAffected != 1 {
		return errors.New("优惠券未被此订单使用")
	}
	return err
}
