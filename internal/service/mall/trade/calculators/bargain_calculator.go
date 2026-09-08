package calculators

import (
	"context"

	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// BargainActivityPriceCalculator 砍价活动价格计算器
type BargainActivityPriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	promotionCalculator tradeSvc.PromotionPriceCalculator
}

// NewBargainActivityPriceCalculator 创建砍价活动价格计算器
func NewBargainActivityPriceCalculator(
	promotionCalculator tradeSvc.PromotionPriceCalculator,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *BargainActivityPriceCalculator {
	return &BargainActivityPriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			tradeModel.CalculatorNameBargain,
			tradeModel.OrderBargainActivity,
			helper,
			logger,
		),
		promotionCalculator: promotionCalculator,
	}
}

// Calculate 执行砍价活动价格计算
func (c *BargainActivityPriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只处理砍价订单
	if resp.Type != tradeModel.TradeOrderTypeBargain {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行砍价活动价格计算",
		zap.Int64("bargainRecordId", req.BargainRecordId),
	)

	// 砍价订单只允许一个商品
	if len(req.Items) != 1 {
		err := pkgErrors.NewBizError(1011900001, "砍价时，只允许选择一个商品")
		c.LogError(ctx, req, err, "砍价订单商品数量验证失败")
		return err
	}

	item := req.Items[0]

	// 计算砍价价格
	bargainPrice, err := c.promotionCalculator.CalculateBargainPrice(ctx, req.UserID, req.BargainRecordId, item.SkuID, item.Count)
	if err != nil {
		c.LogError(ctx, req, err, "计算砍价价格失败")
		return err
	}

	// 更新响应中的商品项价格
	for i := range resp.Items {
		if resp.Items[i].SkuID == item.SkuID {
			originalPayPrice := resp.Items[i].PayPrice
			promotionDiscount := originalPayPrice - bargainPrice

			resp.Items[i].DiscountPrice += promotionDiscount
			resp.Items[i].PayPrice = bargainPrice

			c.LogCalculation(ctx, req, "砍价价格计算完成",
				zap.Int64("skuId", item.SkuID),
				zap.Int("originalPrice", originalPayPrice),
				zap.Int("bargainPrice", bargainPrice),
				zap.Int("promotionDiscount", promotionDiscount),
			)
			break
		}
	}

	return nil
}

// IsApplicable 判断是否适用于当前订单类型
func (c *BargainActivityPriceCalculator) IsApplicable(orderType int) bool {
	return orderType == tradeModel.TradeOrderTypeBargain
}
