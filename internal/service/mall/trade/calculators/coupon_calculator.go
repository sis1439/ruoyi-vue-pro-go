package calculators

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	promotionModel "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// CouponPriceCalculator 优惠券价格计算器
type CouponPriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	couponSvc *promotion.CouponUserService
}

// NewCouponPriceCalculator 创建优惠券价格计算器
func NewCouponPriceCalculator(
	couponSvc *promotion.CouponUserService,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *CouponPriceCalculator {
	return &CouponPriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			consts.CalculatorNameCoupon,
			consts.OrderCoupon,
			helper,
			logger,
		),
		couponSvc: couponSvc,
	}
}

// Calculate 执行优惠券价格计算
func (c *CouponPriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只有【普通】订单，才允许使用优惠劵 (对齐 Java TradeCouponPriceCalculator#calculate)
	if resp.Type != consts.TradeOrderTypeNormal {
		if req.CouponID != nil && *req.CouponID > 0 {
			return pkgErrors.NewBizError(1011902004, "优惠券仅限普通订单使用")
		}
		return nil
	}

	// 1.1 加载用户的优惠劵列表
	coupons, err := c.couponSvc.GetUnusedCouponList(ctx, req.UserID)
	if err != nil {
		c.LogError(ctx, req, err, "获取用户优惠券列表失败")
		return err
	}
	// 过滤过期的优惠券
	now := time.Now()
	coupons = lo.Filter(coupons, func(coupon *promotionModel.PromotionCoupon, _ int) bool {
		return !now.After(coupon.ValidEndTime)
	})

	// 1.2 计算优惠劵的使用条件
	resp.Coupons = c.calculateCoupons(coupons, resp)

	// 2. 校验优惠劵是否可用
	if req.CouponID == nil || *req.CouponID <= 0 {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行优惠券价格计算",
		zap.Int64("couponId", *req.CouponID),
	)

	couponBO, foundBO := lo.Find(resp.Coupons, func(item tradeSvc.TradePriceCalculateCouponBO) bool {
		return item.ID == *req.CouponID
	})
	coupon, found := lo.Find(coupons, func(item *promotionModel.PromotionCoupon) bool {
		return item.ID == *req.CouponID
	})

	if !foundBO || !found {
		c.LogError(ctx, req, nil, "优惠券不存在")
		return pkgErrors.NewBizError(1011902001, "优惠券不存在")
	}
	if !couponBO.Match {
		reason := ""
		if couponBO.MismatchReason != nil {
			reason = *couponBO.MismatchReason
		}
		c.LogError(ctx, req, nil, reason)
		return pkgErrors.NewBizError(1011902001, reason)
	}

	// 3.1 计算可以优惠的金额
	orderItems := c.filterMatchCouponOrderItems(resp.Items, coupon)
	totalPayPrice := c.Helper.CalculateTotalPayPrice(orderItems, false)
	couponPrice := c.getCouponPrice(coupon, totalPayPrice)

	// 3.2 计算分摊的优惠金额
	divideCouponPrices := c.Helper.DividePrice(orderItems, couponPrice)

	// 4.1 记录使用的优惠券
	resp.CouponID = *req.CouponID

	// 4.2 记录优惠明细
	c.Helper.AddPromotion(resp, &tradeSvc.TradePriceCalculatePromotionBO{
		ID:            coupon.ID,
		Name:          coupon.Name,
		Type:          consts.PromotionTypeCoupon,
		TotalPrice:    totalPayPrice,
		DiscountPrice: couponPrice,
		Match:         true,
		Description:   fmt.Sprintf("优惠券：省 %.2f 元", float64(couponPrice)/100.0),
		Items:         c.buildPromotionItems(orderItems, divideCouponPrices),
	})

	// 4.3 更新 SKU 优惠金额
	// 由于 DividePrice 使用的是 orderItems 的副本，我们需要将结果映射回原始 resp.Items
	matchSkuMap := make(map[int64]int)
	for i, item := range orderItems {
		matchSkuMap[item.SkuID] = divideCouponPrices[i]
	}

	for i := range resp.Items {
		if discount, ok := matchSkuMap[resp.Items[i].SkuID]; ok {
			resp.Items[i].CouponPrice = discount
			c.Helper.RecountPayPrice(&resp.Items[i])

			c.LogCalculation(ctx, req, "分摊优惠券折扣",
				zap.Int64("skuId", resp.Items[i].SkuID),
				zap.Int("dividedCouponPrice", discount),
				zap.Int("totalCouponPrice", resp.Items[i].CouponPrice),
			)
		}
	}
	c.Helper.UpdateResponsePrice(resp) // 对齐 Java recountAllPrice

	return nil
}

// calculateCoupons 计算优惠券列表及其匹配状态 (对齐 Java TradeCouponPriceCalculator#calculateCoupons)
func (c *CouponPriceCalculator) calculateCoupons(coupons []*promotionModel.PromotionCoupon, resp *tradeSvc.TradePriceCalculateRespBO) []tradeSvc.TradePriceCalculateCouponBO {
	res := make([]tradeSvc.TradePriceCalculateCouponBO, 0, len(coupons))
	now := time.Now()

	for _, coupon := range coupons {
		bo := tradeSvc.TradePriceCalculateCouponBO{
			ID:                 coupon.ID,
			Name:               coupon.Name,
			UsePrice:           coupon.UsePrice,
			ValidStartTime:     coupon.ValidStartTime.UnixMilli(), // 毫秒时间戳
			ValidEndTime:       coupon.ValidEndTime.UnixMilli(),   // 毫秒时间戳
			DiscountType:       coupon.DiscountType,
			DiscountPercent:    coupon.DiscountPercent,
			DiscountPrice:      coupon.DiscountPrice,
			DiscountLimitPrice: lo.FromPtr(coupon.DiscountLimitPrice),
			Match:              true,
			MismatchReason:     nil, // 默认为nil
		}

		// 1.1 优惠劵未到使用时间
		if now.Before(coupon.ValidStartTime) {
			reason := "优惠券未到使用时间"
			bo.Match = false
			bo.MismatchReason = &reason
		}

		if bo.Match {
			// 1.2 优惠劵没有匹配的商品
			orderItems := c.filterMatchCouponOrderItems(resp.Items, coupon)
			if len(orderItems) == 0 {
				reason := "优惠券没有匹配的商品"
				bo.Match = false
				bo.MismatchReason = &reason
			} else {
				// 1.3 差 %1$,.2f 元可用优惠劵
				totalPayPrice := c.Helper.CalculateTotalPayPrice(orderItems, false)
				if totalPayPrice < coupon.UsePrice {
					reason := fmt.Sprintf("差 %.2f 元可用优惠券", float64(coupon.UsePrice-totalPayPrice)/100.0)
					bo.Match = false
					bo.MismatchReason = &reason
				} else {
					// 1.4 优惠金额超过订单金额
					couponPrice := c.getCouponPrice(coupon, totalPayPrice)
					if couponPrice >= totalPayPrice {
						reason := "优惠金额超过订单金额"
						bo.Match = false
						bo.MismatchReason = &reason
					}
				}
			}
		}

		res = append(res, bo)
	}
	return res
}

// getCouponPrice 计算优惠金额 (对齐 Java TradeCouponPriceCalculator#getCouponPrice)
func (c *CouponPriceCalculator) getCouponPrice(coupon *promotionModel.PromotionCoupon, totalPayPrice int) int {
	switch coupon.DiscountType {
	case consts.DiscountTypePrice: // 满减
		return coupon.DiscountPrice
	case consts.DiscountTypePercent: // 折扣
		amount := totalPayPrice - (totalPayPrice * coupon.DiscountPercent / 100)
		if coupon.DiscountLimitPrice != nil {
			return lo.Min([]int{amount, *coupon.DiscountLimitPrice})
		}
		return amount
	}
	return 0
}

// filterMatchCouponOrderItems 获得优惠劵可使用的订单项（商品）列表 (对齐 Java TradeCouponPriceCalculator#filterMatchCouponOrderItems)
func (c *CouponPriceCalculator) filterMatchCouponOrderItems(items []tradeSvc.TradePriceCalculateItemRespBO, coupon *promotionModel.PromotionCoupon) []tradeSvc.TradePriceCalculateItemRespBO {
	res := make([]tradeSvc.TradePriceCalculateItemRespBO, 0)
	for i := range items {
		item := items[i]
		if !item.Selected {
			continue
		}

		// 校验范围
		matched := false
		switch coupon.ProductScope {
		case consts.ProductScopeAll:
			matched = true
		case consts.ProductScopeSpu:
			matched = lo.Contains(coupon.ProductScopeValues, item.SpuID)
		case consts.ProductScopeCategory:
			matched = lo.Contains(coupon.ProductScopeValues, item.CategoryID)
		}

		if matched {
			res = append(res, item)
		}
	}
	return res
}

// buildPromotionItems 构建促销项明细
func (c *CouponPriceCalculator) buildPromotionItems(items []tradeSvc.TradePriceCalculateItemRespBO, prices []int) []tradeSvc.TradePriceCalculatePromotionItemBO {
	res := make([]tradeSvc.TradePriceCalculatePromotionItemBO, len(items))
	for i, item := range items {
		res[i] = tradeSvc.TradePriceCalculatePromotionItemBO{
			SkuID:         item.SkuID,
			TotalPrice:    item.Price * item.Count,
			DiscountPrice: prices[i],
			PayPrice:      item.PayPrice - prices[i],
		}
	}
	return res
}

// IsApplicable 判断是否适用于当前订单类型
func (c *CouponPriceCalculator) IsApplicable(orderType int) bool {
	return orderType == consts.TradeOrderTypeNormal
}
