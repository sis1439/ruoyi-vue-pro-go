package calculators

import (
	"context"

	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	"go.uber.org/zap"
)

// RewardActivityPriceCalculator 满减送活动价格计算器
type RewardActivityPriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	rewardActivitySvc *promotion.RewardActivityService
}

// NewRewardActivityPriceCalculator 创建满减送活动价格计算器
func NewRewardActivityPriceCalculator(
	rewardActivitySvc *promotion.RewardActivityService,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *RewardActivityPriceCalculator {
	return &RewardActivityPriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			tradeModel.CalculatorNameReward,
			tradeModel.OrderRewardActivity,
			helper,
			logger,
		),
		rewardActivitySvc: rewardActivitySvc,
	}
}

// Calculate 执行满减送活动价格计算
func (c *RewardActivityPriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只处理普通订单
	if resp.Type != tradeModel.TradeOrderTypeNormal {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行满减送活动价格计算")

	// 构建活动匹配项
	matchItems := make([]promotion.ActivityMatchItem, 0)
	for _, item := range resp.Items {
		if !item.Selected { // 过滤未选中项，对齐Java版本逻辑
			continue
		}
		matchItems = append(matchItems, promotion.ActivityMatchItem{
			SkuID:      item.SkuID,
			SpuID:      item.SpuID,
			CategoryID: item.CategoryID,
			Price:      item.Price,
			Count:      item.Count,
		})
	}

	if len(matchItems) == 0 {
		return nil
	}

	// 计算满减送活动
	activityDiscount, rewardResults, err := c.rewardActivitySvc.CalculateRewardActivity(ctx, matchItems)
	if err != nil {
		c.LogError(ctx, req, err, "计算满减送活动失败")
		return err
	}

	if activityDiscount <= 0 {
		return nil
	}

	c.LogCalculation(ctx, req, "满减送活动计算结果",
		zap.Int("activityDiscount", activityDiscount),
		zap.Int("resultCount", len(rewardResults)),
	)

	// 每个活动只在匹配该活动的 SKU 之间分摊。
	for _, result := range rewardResults {
		indices := make([]int, 0)
		items := make([]tradeSvc.TradePriceCalculateItemRespBO, 0)
		for i, item := range resp.Items {
			if !item.Selected {
				continue
			}
			for _, skuID := range result.SkuIDs {
				if item.SkuID == skuID {
					indices = append(indices, i)
					items = append(items, item)
					break
				}
			}
		}
		prices := c.Helper.DividePrice(items, result.TotalDiscount)
		for i, index := range indices {
			resp.Items[index].DiscountPrice += prices[i]
			c.Helper.RecountPayPrice(&resp.Items[index])
		}
	}

	// 添加促销活动明细到响应
	for _, res := range rewardResults {
		p := &tradeSvc.TradePriceCalculatePromotionBO{
			ID:            res.ActivityID,
			Name:          res.ActivityName,
			Type:          tradeModel.PromotionTypeRewardActivity,
			TotalPrice:    res.TotalPrice,
			DiscountPrice: res.TotalDiscount,
			Match:         true,
		}

		// 添加商品项明细
		for _, skuID := range res.SkuIDs {
			p.Items = append(p.Items, tradeSvc.TradePriceCalculatePromotionItemBO{
				SkuID: skuID,
			})
		}

		c.Helper.AddPromotion(resp, p)
	}

	return nil
}

// IsApplicable 判断是否适用于当前订单类型
func (c *RewardActivityPriceCalculator) IsApplicable(orderType int) bool {
	return orderType == tradeModel.TradeOrderTypeNormal
}
