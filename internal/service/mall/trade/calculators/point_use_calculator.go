package calculators

import (
	"context"

	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	tradeSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	memberSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// PointUsePriceCalculator 积分抵扣价格计算器
type PointUsePriceCalculator struct {
	*tradeSvc.BasePriceCalculator
	memberConfigSvc *memberSvc.MemberConfigService
	memberUserSvc   *memberSvc.MemberUserService
}

// NewPointUsePriceCalculator 创建积分抵扣价格计算器
func NewPointUsePriceCalculator(
	memberConfigSvc *memberSvc.MemberConfigService,
	memberUserSvc *memberSvc.MemberUserService,
	helper *tradeSvc.PriceCalculatorHelper,
	logger *zap.Logger,
) *PointUsePriceCalculator {
	return &PointUsePriceCalculator{
		BasePriceCalculator: tradeSvc.NewBasePriceCalculator(
			tradeModel.CalculatorNamePointUse,
			tradeModel.OrderPointUse,
			helper,
			logger,
		),
		memberConfigSvc: memberConfigSvc,
		memberUserSvc:   memberUserSvc,
	}
}

// Calculate 执行积分抵扣价格计算
func (c *PointUsePriceCalculator) Calculate(ctx context.Context, req *tradeSvc.TradePriceCalculateReqBO, resp *tradeSvc.TradePriceCalculateRespBO) error {
	// 只有启用积分抵扣且用户ID有效时才处理
	if !req.PointStatus || req.UserID <= 0 {
		return nil
	}

	c.LogCalculation(ctx, req, "开始执行积分抵扣价格计算")

	// 获取会员配置
	config, err := c.memberConfigSvc.GetConfig(ctx)
	if err != nil || config == nil || !config.PointTradeDeductEnable {
		c.LogCalculation(ctx, req, "积分抵扣功能未启用")
		return nil
	}

	// 获取用户信息
	user, err := c.memberUserSvc.GetUser(ctx, req.UserID)
	if err != nil || user == nil || user.Point <= 0 {
		c.LogCalculation(ctx, req, "用户积分不足")
		return nil
	}

	// 获取积分抵扣单位价格
	deductUnitPrice := config.PointTradeDeductUnitPrice
	if deductUnitPrice <= 0 {
		c.LogCalculation(ctx, req, "积分抵扣单位价格配置无效")
		return nil
	}

	c.LogCalculation(ctx, req, "积分抵扣配置信息",
		zap.Int("userPoint", int(user.Point)),
		zap.Int("deductUnitPrice", deductUnitPrice),
		zap.Int("maxDeductPoints", config.PointTradeDeductMaxPrice),
	)

	// 计算可用积分数量
	canUsePoints := int(user.Point)

	// 限制最大积分抵扣数量
	if config.PointTradeDeductMaxPrice > 0 && canUsePoints > config.PointTradeDeductMaxPrice {
		canUsePoints = config.PointTradeDeductMaxPrice
	}

	// 计算抵扣金额
	pointTotalValue := canUsePoints * deductUnitPrice

	// 计算当前支付金额
	currentPayPrice := c.Helper.CalculateTotalPayPrice(resp.Items, true)

	// 限制不超过应付金额（严格对齐：禁止 0 元购）
	if pointTotalValue >= currentPayPrice {
		err := pkgErrors.NewBizError(1011904005, "支付金额不能小于等于 0")
		c.LogError(ctx, req, err, "积分抵扣金额超过支付金额")
		return err
	}

	c.LogCalculation(ctx, req, "积分抵扣计算结果",
		zap.Int("canUsePoints", canUsePoints),
		zap.Int("pointTotalValue", pointTotalValue),
		zap.Int("currentPayPrice", currentPayPrice),
	)

	// 分摊积分抵扣到各项
	dividePointPrices := c.Helper.DividePrice(resp.Items, pointTotalValue)
	divideUsePoints := c.Helper.DividePrice(resp.Items, canUsePoints)

	for i := range resp.Items {
		if !resp.Items[i].Selected {
			continue
		}

		resp.Items[i].PointPrice += dividePointPrices[i]
		resp.Items[i].UsePoint = divideUsePoints[i]

		// 重新计算支付金额
		c.Helper.RecountPayPrice(&resp.Items[i])

		c.LogCalculation(ctx, req, "分摊积分抵扣",
			zap.Int64("skuId", resp.Items[i].SkuID),
			zap.Int("dividedPointPrice", dividePointPrices[i]),
			zap.Int("dividedUsePoints", divideUsePoints[i]),
			zap.Int("totalPointPrice", resp.Items[i].PointPrice),
		)
	}

	// 更新响应中的积分信息
	resp.UsePoint = canUsePoints
	resp.TotalPoint = int(user.Point)

	return nil
}

// IsApplicable 判断是否适用于当前订单类型
func (c *PointUsePriceCalculator) IsApplicable(orderType int) bool {
	// 积分抵扣适用于所有订单类型
	return true
}
