package calculators

import (
	"context"

	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// CombinationActivityPriceCalculator 拼团活动价格计算器
type CombinationActivityPriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	promotionCalculator tradeSvc.PromotionPriceCalculator
}

// NewCombinationActivityPriceCalculator 创建拼团活动价格计算器
func NewCombinationActivityPriceCalculator(
	promotionCalculator tradeSvc.PromotionPriceCalculator,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *CombinationActivityPriceCalculator {
	return &CombinationActivityPriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			consts.CalculatorNameCombination,
			consts.TradeOrderTypeCombination,
			helper,
			logger,
		),
		promotionCalculator: promotionCalculator,
	}
}

// Calculate 执行拼团活动价格计算
func (c *CombinationActivityPriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只处理拼团订单
	if resp.Type != consts.TradeOrderTypeCombination {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行拼团活动价格计算",
		zap.Int64("combinationActivityId", req.CombinationActivityId),
		zap.Int64("combinationHeadId", req.CombinationHeadId),
	)

	// 拼团订单只允许一个商品
	if len(req.Items) != 1 {
		err := pkgErrors.NewBizError(1011900001, "拼团时，只允许选择一个商品")
		c.LogError(ctx, req, err, "拼团订单商品数量验证失败")
		return err
	}

	item := req.Items[0]

	// 计算拼团价格
	combinationPrice, err := c.promotionCalculator.CalculateCombinationPrice(ctx, req.UserID, req.CombinationActivityId, req.CombinationHeadId, item.SkuID, item.Count)
	if err != nil {
		c.LogError(ctx, req, err, "计算拼团价格失败")
		return err
	}

	// 更新响应中的商品项价格
	for i := range resp.Items {
		if resp.Items[i].SkuID == item.SkuID {
			originalPayPrice := resp.Items[i].PayPrice
			promotionDiscount := originalPayPrice - combinationPrice

			resp.Items[i].DiscountPrice += promotionDiscount
			// 重新计算支付金额
			c.Helper.RecountPayPrice(&resp.Items[i])

			// 记录促销活动明细
			c.Helper.AddPromotion(resp, &tradeSvc.TradePriceCalculatePromotionBO{
				ID:            req.CombinationActivityId,
				Name:          "拼团活动",
				Type:          consts.PromotionTypeCombinationActivity,
				TotalPrice:    originalPayPrice,
				DiscountPrice: promotionDiscount,
				Match:         true,
				Items: []tradeSvc.TradePriceCalculatePromotionItemBO{
					{
						SkuID:         item.SkuID,
						TotalPrice:    originalPayPrice,
						DiscountPrice: promotionDiscount,
						PayPrice:      combinationPrice,
					},
				},
			})

			c.LogCalculation(ctx, req, "拼团价格计算完成",
				zap.Int64("skuId", item.SkuID),
				zap.Int("originalPrice", originalPayPrice),
				zap.Int("combinationPrice", combinationPrice),
				zap.Int("promotionDiscount", promotionDiscount),
			)
			break
		}
	}

	return nil
}

// IsApplicable 判断是否适用于当前订单类型
func (c *CombinationActivityPriceCalculator) IsApplicable(orderType int) bool {
	return orderType == consts.TradeOrderTypeCombination
}
