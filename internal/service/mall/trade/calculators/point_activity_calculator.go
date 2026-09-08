package calculators

import (
	"context"

	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// PointActivityPriceCalculator 积分商城价格计算器
type PointActivityPriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	promotionCalculator tradeSvc.PromotionPriceCalculator
}

// NewPointActivityPriceCalculator 创建积分商城价格计算器
func NewPointActivityPriceCalculator(
	promotionCalculator tradeSvc.PromotionPriceCalculator,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *PointActivityPriceCalculator {
	return &PointActivityPriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			tradeModel.CalculatorNamePoint,
			tradeModel.OrderPointActivity,
			helper,
			logger,
		),
		promotionCalculator: promotionCalculator,
	}
}

// Calculate 执行积分商城价格计算
func (c *PointActivityPriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只处理积分订单
	if resp.Type != tradeModel.TradeOrderTypePoint {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行积分商城价格计算",
		zap.Int64("pointActivityId", req.PointActivityId),
	)

	// 积分商城订单只允许一个商品
	if len(req.Items) != 1 {
		err := pkgErrors.NewBizError(1011900001, "积分商城时，只允许选择一个商品")
		c.LogError(ctx, req, err, "积分商城订单商品数量验证失败")
		return err
	}

	item := req.Items[0]

	// 需要先找到对应的SPU ID
	var spuID int64
	for _, respItem := range resp.Items {
		if respItem.SkuID == item.SkuID {
			spuID = respItem.SpuID
			break
		}
	}

	if spuID == 0 {
		err := pkgErrors.NewBizError(1011900001, "未找到商品SPU信息")
		c.LogError(ctx, req, err, "积分商城商品SPU查找失败")
		return err
	}

	// 计算积分商城价格
	pointPrice, err := c.promotionCalculator.CalculatePointPrice(ctx, req.PointActivityId, spuID, item.SkuID, item.Count)
	if err != nil {
		c.LogError(ctx, req, err, "计算积分商城价格失败")
		return err
	}

	// 更新响应中的商品项价格
	for i := range resp.Items {
		if resp.Items[i].SkuID == item.SkuID {
			originalPayPrice := resp.Items[i].PayPrice
			promotionDiscount := originalPayPrice - pointPrice

			resp.Items[i].DiscountPrice += promotionDiscount
			resp.Items[i].PayPrice = pointPrice

			c.LogCalculation(ctx, req, "积分商城价格计算完成",
				zap.Int64("skuId", item.SkuID),
				zap.Int64("spuId", spuID),
				zap.Int("originalPrice", originalPayPrice),
				zap.Int("pointPrice", pointPrice),
				zap.Int("promotionDiscount", promotionDiscount),
			)
			break
		}
	}

	return nil
}

// IsApplicable 判断是否适用于当前订单类型
func (c *PointActivityPriceCalculator) IsApplicable(orderType int) bool {
	return orderType == tradeModel.TradeOrderTypePoint
}
