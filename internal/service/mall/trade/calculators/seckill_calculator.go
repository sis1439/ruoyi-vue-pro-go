package calculators

import (
	"context"

	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// SeckillActivityPriceCalculator 秒杀活动价格计算器
type SeckillActivityPriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	seckillSvc *promotion.SeckillActivityService
}

// NewSeckillActivityPriceCalculator 创建秒杀活动价格计算器
func NewSeckillActivityPriceCalculator(
	seckillSvc *promotion.SeckillActivityService,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *SeckillActivityPriceCalculator {
	return &SeckillActivityPriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			tradeModel.CalculatorNameSeckill,
			tradeModel.OrderSeckillActivity,
			helper,
			logger,
		),
		seckillSvc: seckillSvc,
	}
}

// Calculate 执行秒杀活动价格计算
func (c *SeckillActivityPriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只处理秒杀订单
	if resp.Type != tradeModel.TradeOrderTypeSeckill {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行秒杀活动价格计算",
		zap.Int64("seckillActivityId", req.SeckillActivityId),
	)

	// 秒杀订单只允许一个商品
	if len(req.Items) != 1 {
		err := pkgErrors.NewBizError(1011900001, "秒杀时，只允许选择一个商品")
		c.LogError(ctx, req, err, "秒杀订单商品数量验证失败")
		return err
	}

	item := req.Items[0]

	// 验证秒杀活动并获取秒杀价格
	activity, seckillProd, err := c.seckillSvc.ValidateJoinSeckill(ctx, req.SeckillActivityId, item.SkuID, item.Count)
	if err != nil {
		c.LogError(ctx, req, err, "验证秒杀活动失败")
		return err
	}

	// 计算秒杀价格
	seckillTotal := seckillProd.SeckillPrice * item.Count

	// 更新响应中的商品项价格
	for i := range resp.Items {
		if resp.Items[i].SkuID == item.SkuID {
			originalPayPrice := resp.Items[i].PayPrice
			promotionDiscount := originalPayPrice - seckillTotal

			resp.Items[i].DiscountPrice += promotionDiscount
			c.Helper.RecountPayPrice(&resp.Items[i])
			c.Helper.AddPromotion(resp, c.Helper.BuildPromotionDetail(activity.ID, activity.Name, tradeModel.PromotionTypeSeckillActivity, originalPayPrice, promotionDiscount, []int64{item.SkuID}))

			c.LogCalculation(ctx, req, "秒杀价格计算完成",
				zap.Int64("skuId", item.SkuID),
				zap.Int("originalPrice", originalPayPrice),
				zap.Int("seckillPrice", seckillTotal),
				zap.Int("promotionDiscount", promotionDiscount),
			)
			break
		}
	}

	return nil
}

// IsApplicable 判断是否适用于当前订单类型
func (c *SeckillActivityPriceCalculator) IsApplicable(orderType int) bool {
	return orderType == tradeModel.TradeOrderTypeSeckill
}
