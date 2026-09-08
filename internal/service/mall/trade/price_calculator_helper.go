package trade

import (
	"fmt"

	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"go.uber.org/zap"
)

// PriceCalculatorHelper 价格计算辅助工具
// 提供通用的价格计算方法和工具函数
type PriceCalculatorHelper struct {
	logger *zap.Logger
}

// NewPriceCalculatorHelper 创建价格计算辅助工具
func NewPriceCalculatorHelper(logger *zap.Logger) *PriceCalculatorHelper {
	return &PriceCalculatorHelper{
		logger: logger,
	}
}

// BuildCalculateResp 构建计算响应对象
func (h *PriceCalculatorHelper) BuildCalculateResp(req *TradePriceCalculateReqBO) *TradePriceCalculateRespBO {
	resp := &TradePriceCalculateRespBO{
		Type:       tradeModel.TradeOrderTypeNormal, // 默认为普通订单
		Price:      TradePriceCalculatePriceBO{},
		Items:      make([]TradePriceCalculateItemRespBO, 0),
		Success:    true,
		Coupons:    make([]TradePriceCalculateCouponBO, 0),
		Promotions: make([]TradePriceCalculatePromotionBO, 0),
	}

	// 根据请求参数确定订单类型
	if req.SeckillActivityId > 0 {
		resp.Type = tradeModel.TradeOrderTypeSeckill
	} else if req.BargainRecordId > 0 {
		resp.Type = tradeModel.TradeOrderTypeBargain
	} else if req.CombinationActivityId > 0 {
		resp.Type = tradeModel.TradeOrderTypeCombination
	} else if req.PointActivityId > 0 {
		resp.Type = tradeModel.TradeOrderTypePoint
	}

	return resp
}

// DividePrice 按支付金额比例分摊折扣
// 对应 Java：TradePriceCalculatorHelper#dividePrice
func (h *PriceCalculatorHelper) DividePrice(items []TradePriceCalculateItemRespBO, totalDiscount int) []int {
	if len(items) == 0 {
		return []int{}
	}

	// 计算所有【已选中】项的总支付金额
	totalPayPrice := 0
	for _, item := range items {
		if item.Selected {
			totalPayPrice += item.PayPrice
		}
	}

	if totalPayPrice == 0 {
		return make([]int, len(items))
	}

	// 按比例分摊
	dividedPrices := make([]int, len(items))
	remainPrice := totalDiscount

	for i := 0; i < len(items); i++ {
		// 1. 如果是未选中，则分摊为 0
		if !items[i].Selected {
			dividedPrices[i] = 0
			continue
		}
		// 2. 如果选中，则按照百分比进行分摊
		if i < len(items)-1 {
			// 前 n-1 项按比例计算
			dividedPrices[i] = int(float64(totalDiscount) * (float64(items[i].PayPrice) / float64(totalPayPrice)))
			remainPrice -= dividedPrices[i]
		} else {
			// 最后一项用剩余金额（避免舍入误差）
			dividedPrices[i] = remainPrice
		}
	}

	h.logger.Debug("价格分摊计算",
		zap.Int("totalDiscount", totalDiscount),
		zap.Int("totalPayPrice", totalPayPrice),
		zap.Ints("dividedPrices", dividedPrices),
	)

	return dividedPrices
}

// RecountPayPrice 重新计算支付金额
// 对应 Java：TradePriceCalculatorHelper#recountPayPrice
// PayPrice = Price * Count - DiscountPrice + DeliveryPrice - CouponPrice - PointPrice - VipPrice
func (h *PriceCalculatorHelper) RecountPayPrice(item *TradePriceCalculateItemRespBO) {
	originalPayPrice := item.PayPrice

	item.PayPrice = item.Price*item.Count - item.DiscountPrice + item.DeliveryPrice - item.CouponPrice - item.PointPrice - item.VipPrice

	h.logger.Debug("重新计算支付金额",
		zap.Int64("skuId", item.SkuID),
		zap.Int("originalPayPrice", originalPayPrice),
		zap.Int("newPayPrice", item.PayPrice),
		zap.Int("price", item.Price),
		zap.Int("count", item.Count),
		zap.Int("discountPrice", item.DiscountPrice),
		zap.Int("deliveryPrice", item.DeliveryPrice),
		zap.Int("couponPrice", item.CouponPrice),
		zap.Int("pointPrice", item.PointPrice),
		zap.Int("vipPrice", item.VipPrice),
	)
}

// AddPromotion 添加促销活动明细
func (h *PriceCalculatorHelper) AddPromotion(resp *TradePriceCalculateRespBO, promotion *TradePriceCalculatePromotionBO) {
	if promotion == nil {
		return
	}

	resp.Promotions = append(resp.Promotions, *promotion)

	h.logger.Debug("添加促销活动明细",
		zap.Int64("promotionId", promotion.ID),
		zap.String("promotionName", promotion.Name),
		zap.Int("promotionType", promotion.Type),
		zap.Int("discountPrice", promotion.DiscountPrice),
	)
}

// FormatMoney 格式化金额（分转元）
func (h *PriceCalculatorHelper) FormatMoney(cents int) string {
	return fmt.Sprintf("%.2f", float64(cents)/100.0)
}

// CalculateRatePrice 计算折扣价（对齐 Java MoneyUtils#calculateRatePrice）
// discountPercent: 折扣百分比，例如 80 代表 80%
func (h *PriceCalculatorHelper) CalculateRatePrice(price int, discountPercent int) int {
	if discountPercent <= 0 {
		return 0
	}
	// 计算折扣价：price * discountPercent / 100
	targetPrice := (price * discountPercent) / 100

	h.logger.Debug("计算折扣价格",
		zap.Int("originalPrice", price),
		zap.Int("discountPercent", discountPercent),
		zap.Int("targetPrice", targetPrice),
	)

	return targetPrice
}

// ValidateOrderType 验证订单类型是否支持特定计算器
func (h *PriceCalculatorHelper) ValidateOrderType(orderType int, supportedTypes []int) bool {
	for _, supportedType := range supportedTypes {
		if orderType == supportedType {
			return true
		}
	}
	return false
}

// CalculateTotalPrice 计算总价格
func (h *PriceCalculatorHelper) CalculateTotalPrice(items []TradePriceCalculateItemRespBO, selectedOnly bool) int {
	totalPrice := 0
	for _, item := range items {
		if selectedOnly && !item.Selected {
			continue
		}
		totalPrice += item.Price * item.Count
	}
	return totalPrice
}

// CalculateTotalPayPrice 计算总支付金额
func (h *PriceCalculatorHelper) CalculateTotalPayPrice(items []TradePriceCalculateItemRespBO, selectedOnly bool) int {
	totalPayPrice := 0
	for _, item := range items {
		if selectedOnly && !item.Selected {
			continue
		}
		totalPayPrice += item.PayPrice
	}
	return totalPayPrice
}

// UpdateResponsePrice 更新响应对象的价格信息
func (h *PriceCalculatorHelper) UpdateResponsePrice(resp *TradePriceCalculateRespBO) {
	// 重新计算总价格
	totalPrice := h.CalculateTotalPrice(resp.Items, true)
	totalPayPrice := h.CalculateTotalPayPrice(resp.Items, true)

	// 计算各种折扣总额
	totalDiscountPrice := 0
	totalCouponPrice := 0
	totalPointPrice := 0
	totalVipPrice := 0
	totalDeliveryPrice := 0

	for _, item := range resp.Items {
		if !item.Selected {
			continue
		}
		totalDiscountPrice += item.DiscountPrice
		totalCouponPrice += item.CouponPrice
		totalPointPrice += item.PointPrice
		totalVipPrice += item.VipPrice
		totalDeliveryPrice += item.DeliveryPrice
	}

	// 更新价格信息
	resp.Price.TotalPrice = totalPrice
	resp.Price.DiscountPrice = totalDiscountPrice
	resp.Price.CouponPrice = totalCouponPrice
	resp.Price.PointPrice = totalPointPrice
	resp.Price.VipPrice = totalVipPrice
	resp.Price.DeliveryPrice = totalDeliveryPrice
	resp.Price.PayPrice = totalPayPrice

	h.logger.Debug("更新响应价格信息",
		zap.Int("totalPrice", totalPrice),
		zap.Int("totalPayPrice", totalPayPrice),
		zap.Int("totalDiscountPrice", totalDiscountPrice),
		zap.Int("totalCouponPrice", totalCouponPrice),
		zap.Int("totalPointPrice", totalPointPrice),
		zap.Int("totalVipPrice", totalVipPrice),
		zap.Int("totalDeliveryPrice", totalDeliveryPrice),
	)
}

// ValidateItemsConsistency 验证商品项一致性
func (h *PriceCalculatorHelper) ValidateItemsConsistency(req *TradePriceCalculateReqBO, resp *TradePriceCalculateRespBO) error {
	if len(req.Items) != len(resp.Items) {
		return fmt.Errorf("请求商品项数量(%d)与响应商品项数量(%d)不一致", len(req.Items), len(resp.Items))
	}

	// 验证每个商品项的SKU ID是否匹配
	reqSkuMap := make(map[int64]bool)
	for _, item := range req.Items {
		reqSkuMap[item.SkuID] = true
	}

	for _, item := range resp.Items {
		if !reqSkuMap[item.SkuID] {
			return fmt.Errorf("响应中包含请求中不存在的SKU ID: %d", item.SkuID)
		}
	}

	return nil
}

// CalculateDiscountRate 计算折扣率
func (h *PriceCalculatorHelper) CalculateDiscountRate(originalPrice, discountPrice int) float64 {
	if originalPrice <= 0 {
		return 0
	}
	return float64(discountPrice) / float64(originalPrice) * 100
}

// ApplyMinimumPrice 应用最低价格限制
func (h *PriceCalculatorHelper) ApplyMinimumPrice(price int, minimumPrice int) int {
	if price < minimumPrice {
		h.logger.Debug("应用最低价格限制",
			zap.Int("originalPrice", price),
			zap.Int("minimumPrice", minimumPrice),
		)
		return minimumPrice
	}
	return price
}

// SplitPriceByWeight 按权重分摊价格
func (h *PriceCalculatorHelper) SplitPriceByWeight(totalAmount int, weights []int) []int {
	if len(weights) == 0 {
		return []int{}
	}

	// 计算总权重
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	if totalWeight == 0 {
		return make([]int, len(weights))
	}

	// 按权重分摊
	result := make([]int, len(weights))
	remainAmount := totalAmount

	for i := 0; i < len(weights)-1; i++ {
		amount := int(int64(totalAmount) * int64(weights[i]) / int64(totalWeight))
		result[i] = amount
		remainAmount -= amount
	}

	// 最后一项使用剩余金额
	result[len(weights)-1] = remainAmount

	h.logger.Debug("按权重分摊价格",
		zap.Int("totalAmount", totalAmount),
		zap.Ints("weights", weights),
		zap.Ints("result", result),
	)

	return result
}

// BuildPromotionDetail 构建促销活动明细
func (h *PriceCalculatorHelper) BuildPromotionDetail(
	activityID int64,
	activityName string,
	promotionType int,
	totalPrice int,
	discountPrice int,
	skuIDs []int64,
) *TradePriceCalculatePromotionBO {
	promotion := &TradePriceCalculatePromotionBO{
		ID:            activityID,
		Name:          activityName,
		Type:          promotionType,
		TotalPrice:    totalPrice,
		DiscountPrice: discountPrice,
		Match:         true,
		Items:         make([]TradePriceCalculatePromotionItemBO, 0),
	}

	// 添加商品项明细
	for _, skuID := range skuIDs {
		promotion.Items = append(promotion.Items, TradePriceCalculatePromotionItemBO{
			SkuID: skuID,
		})
	}

	return promotion
}

// CalculateVipDiscount 计算VIP折扣
func (h *PriceCalculatorHelper) CalculateVipDiscount(originalPrice int, vipDiscountPercent int) int {
	if vipDiscountPercent <= 0 || vipDiscountPercent >= 100 {
		return 0
	}

	// 计算VIP折扣后的价格
	discountedPrice := h.CalculateRatePrice(originalPrice, vipDiscountPercent)
	return originalPrice - discountedPrice
}

// RoundPrice 价格舍入处理
func (h *PriceCalculatorHelper) RoundPrice(price float64) int {
	// 四舍五入到分
	return int(price + 0.5)
}

// ValidatePriceRange 验证价格范围
func (h *PriceCalculatorHelper) ValidatePriceRange(price int, minPrice int, maxPrice int) error {
	if price < minPrice {
		return fmt.Errorf("价格%d低于最低限制%d", price, minPrice)
	}
	if maxPrice > 0 && price > maxPrice {
		return fmt.Errorf("价格%d超过最高限制%d", price, maxPrice)
	}
	return nil
}

// CalculateAveragePrice 计算平均价格
func (h *PriceCalculatorHelper) CalculateAveragePrice(items []TradePriceCalculateItemRespBO, selectedOnly bool) int {
	totalPrice := 0
	totalCount := 0

	for _, item := range items {
		if selectedOnly && !item.Selected {
			continue
		}
		totalPrice += item.Price * item.Count
		totalCount += item.Count
	}

	if totalCount == 0 {
		return 0
	}

	return totalPrice / totalCount
}
