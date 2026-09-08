package trade

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	product2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	productModel "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	promotionModel "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"go.uber.org/zap"
)

// TradePriceService 价格计算服务
// 使用策略模式实现价格计算，对应 Java: TradePriceServiceImpl
type TradePriceService struct {
	calculators         []PriceCalculator
	helper              *PriceCalculatorHelper
	skuSvc              *product.ProductSkuService
	spuSvc              *product.ProductSpuService
	rewardActivitySvc   *promotion.RewardActivityService
	skuPromotionCalc    SkuPromotionCalculator            // 复用 SKU 优惠计算逻辑
	memberUserSvc       *member.MemberUserService         // 备用：会员用户服务
	memberLevelSvc      *member.MemberLevelService        // 备用：会员等级服务
	discountActivitySvc promotion.DiscountActivityService // 备用：折扣活动服务
	logger              *zap.Logger
}

// NewTradePriceService 创建价格计算服务
// NewTradePriceService 创建价格计算服务
func NewTradePriceService(
	calculators []PriceCalculator,
	helper *PriceCalculatorHelper,
	skuSvc *product.ProductSkuService,
	spuSvc *product.ProductSpuService,
	rewardActivitySvc *promotion.RewardActivityService,
	skuPromotionCalc SkuPromotionCalculator, // 复用 SKU 优惠计算逻辑
	discountActivitySvc promotion.DiscountActivityService,
	memberUserSvc *member.MemberUserService,
	memberLevelSvc *member.MemberLevelService,
	logger *zap.Logger,
) *TradePriceService {
	// 按优先级排序计算器
	sort.Slice(calculators, func(i, j int) bool {
		return calculators[i].GetOrder() < calculators[j].GetOrder()
	})

	logger.Info("初始化价格计算服务",
		zap.Int("calculatorCount", len(calculators)),
	)

	// 记录所有计算器信息
	for _, calculator := range calculators {
		logger.Info("注册价格计算器",
			zap.String("name", calculator.GetName()),
			zap.Int("order", calculator.GetOrder()),
		)
	}

	return &TradePriceService{
		calculators:         calculators,
		helper:              helper,
		skuSvc:              skuSvc,
		spuSvc:              spuSvc,
		rewardActivitySvc:   rewardActivitySvc,
		skuPromotionCalc:    skuPromotionCalc,
		discountActivitySvc: discountActivitySvc,
		memberUserSvc:       memberUserSvc,
		memberLevelSvc:      memberLevelSvc,
		logger:              logger,
	}
}

// CalculateOrderPrice 计算订单价格
func (s *TradePriceService) CalculateOrderPrice(ctx context.Context, req *TradePriceCalculateReqBO) (*TradePriceCalculateRespBO, error) {
	return s.calculatePriceInternal(ctx, req, true)
}

// calculatePriceInternal 内部价格计算逻辑
func (s *TradePriceService) calculatePriceInternal(ctx context.Context, req *TradePriceCalculateReqBO, checkStock bool) (*TradePriceCalculateRespBO, error) {
	s.logger.Info("开始计算订单价格",
		zap.Int64("userId", req.UserID),
		zap.Int("itemCount", len(req.Items)),
		zap.Bool("pointStatus", req.PointStatus),
		zap.Int("deliveryType", req.DeliveryType),
		zap.Bool("checkStock", checkStock),
	)

	// 1. 参数验证
	if err := s.validateRequest(req); err != nil {
		s.logger.Error("价格计算参数验证失败", zap.Error(err))
		return nil, err
	}

	// 2. 初始化响应对象
	resp := s.helper.BuildCalculateResp(req)

	// 3. 构建商品项响应
	if err := s.buildItemsResponse(ctx, req, resp, checkStock); err != nil {
		s.logger.Error("构建商品项响应失败", zap.Error(err))
		return nil, err
	}

	// 4. 按顺序执行计算器
	for _, calculator := range s.calculators {
		if !calculator.IsApplicable(resp.Type) {
			s.logger.Debug("跳过不适用的计算器",
				zap.String("calculator", calculator.GetName()),
				zap.Int("orderType", resp.Type),
			)
			continue
		}

		s.logger.Info("执行价格计算器",
			zap.String("calculator", calculator.GetName()),
			zap.Int("order", calculator.GetOrder()),
		)

		if err := calculator.Calculate(ctx, req, resp); err != nil {
			s.logger.Error("价格计算器执行失败",
				zap.String("calculator", calculator.GetName()),
				zap.Error(err),
			)
			return nil, err
		}
	}

	// 5. 更新最终价格信息
	s.helper.UpdateResponsePrice(resp)

	// 6. 最终验证
	if err := s.validateResponse(req, resp); err != nil {
		s.logger.Error("价格计算结果验证失败", zap.Error(err))
		return nil, err
	}

	s.logger.Info("订单价格计算完成",
		zap.Int64("userId", req.UserID),
		zap.Int("totalPrice", resp.Price.TotalPrice),
		zap.Int("payPrice", resp.Price.PayPrice),
		zap.Int("discountPrice", resp.Price.DiscountPrice),
		zap.Int("couponPrice", resp.Price.CouponPrice),
		zap.Int("pointPrice", resp.Price.PointPrice),
		zap.Int("deliveryPrice", resp.Price.DeliveryPrice),
	)

	return resp, nil
}

// validateRequest 验证请求参数
func (s *TradePriceService) validateRequest(req *TradePriceCalculateReqBO) error {
	// 基本参数验证
	if req.UserID <= 0 {
		return pkgErrors.NewBizError(1011900001, "用户ID不能为空")
	}

	if len(req.Items) == 0 {
		return pkgErrors.NewBizError(1011900001, "计算价格时，商品不能为空")
	}

	// 验证商品项参数
	for _, item := range req.Items {
		if item.SkuID <= 0 {
			return pkgErrors.NewBizError(1011900001, "商品SKU ID不能为空")
		}
		if item.Count <= 0 {
			return pkgErrors.NewBizError(1011900001, "商品数量必须大于0")
		}
	}

	return nil
}

// buildItemsResponse 构建商品项响应
// 对应 Java: TradePriceCalculatorHelper#buildCalculateResp
func (s *TradePriceService) buildItemsResponse(ctx context.Context, req *TradePriceCalculateReqBO, resp *TradePriceCalculateRespBO, checkStock bool) error {
	s.logger.Info("开始构建商品项响应",
		zap.Int("itemCount", len(req.Items)),
	)

	// 1. 获取所有 SKU ID
	skuIDs := make([]int64, 0, len(req.Items))
	for _, item := range req.Items {
		skuIDs = append(skuIDs, item.SkuID)
	}

	// 2. 批量获取 SKU 信息
	skuList, err := s.skuSvc.GetSkuList(ctx, skuIDs)
	if err != nil {
		s.logger.Error("获取SKU信息失败", zap.Error(err))
		return pkgErrors.NewBizError(1011900002, "获取商品SKU信息失败")
	}

	// 3. 构建 SKU Map
	skuMap := make(map[int64]*productModel.ProductSku)
	skuRespMap := make(map[int64]*product2.ProductSkuResp)
	spuIDs := make([]int64, 0)
	for _, sku := range skuList {
		skuMap[sku.ID] = &productModel.ProductSku{
			ID:     sku.ID,
			SpuID:  sku.SpuID,
			Price:  sku.Price,
			Stock:  sku.Stock,
			PicURL: sku.PicURL,
			Weight: sku.Weight,
			Volume: sku.Volume,
			// 注意：resp.ProductSkuResp 的 Properties 是 []resp.ProductSkuPropertyResp
			// 而 productModel.ProductSku 的 Properties 是 []productModel.ProductSkuProperty
			// 这里我们直接复用 skuList 里的 resp 数据即可，不需要在 Map 里存模型级的 Properties
		}
		// 为了后续方便获取原 resp 里的属性，我们可以开个 map 存 resp 数据
		skuRespMap[sku.ID] = sku
		spuIDs = append(spuIDs, sku.SpuID)
	}

	// 4. 批量获取 SPU 信息
	spuList, err := s.spuSvc.GetSpuList(ctx, spuIDs)
	if err != nil {
		s.logger.Error("获取SPU信息失败", zap.Error(err))
		return pkgErrors.NewBizError(1011900002, "获取商品SPU信息失败")
	}

	// 5. 构建 SPU Map
	spuMap := make(map[int64]*productModel.ProductSpu)
	for _, spu := range spuList {
		spuMap[spu.ID] = &productModel.ProductSpu{
			ID:                 spu.ID,
			Name:               spu.Name,
			Status:             spu.Status,
			CategoryID:         spu.CategoryID,
			PicURL:             spu.PicURL,
			GiveIntegral:       spu.GiveIntegral,
			DeliveryTypes:      spu.DeliveryTypes,
			DeliveryTemplateID: spu.DeliveryTemplateID,
		}
	}

	// 6. 构建响应商品项
	resp.Items = make([]TradePriceCalculateItemRespBO, 0, len(req.Items))
	for _, reqItem := range req.Items {
		sku, skuExists := skuMap[reqItem.SkuID]
		skuResp, _ := skuRespMap[reqItem.SkuID]
		if !skuExists {
			s.logger.Error("SKU不存在",
				zap.Int64("skuId", reqItem.SkuID),
			)
			return pkgErrors.NewBizError(1011900002, fmt.Sprintf("商品SKU[%d]不存在", reqItem.SkuID))
		}

		spu, spuExists := spuMap[sku.SpuID]
		if !spuExists {
			s.logger.Error("SPU不存在",
				zap.Int64("spuId", sku.SpuID),
			)
			return pkgErrors.NewBizError(1011900002, fmt.Sprintf("商品SPU[%d]不存在", sku.SpuID))
		}

		// 验证 SPU 状态
		if spu.Status != consts.ProductSpuStatusEnable { // 0: 上架
			s.logger.Error("商品已下架",
				zap.Int64("spuId", spu.ID),
				zap.Int("status", spu.Status),
			)
			return pkgErrors.NewBizError(1011900002, fmt.Sprintf("商品[%s]已下架", spu.Name))
		}

		// 验证库存
		if checkStock && reqItem.Count > sku.Stock {
			s.logger.Error("商品库存不足",
				zap.Int64("skuId", reqItem.SkuID),
				zap.Int("requestCount", reqItem.Count),
				zap.Int("stock", sku.Stock),
			)
			return pkgErrors.NewBizError(1011900003, fmt.Sprintf("商品[%s]库存不足", spu.Name))
		}

		// 构建商品项
		item := TradePriceCalculateItemRespBO{
			SpuID:      spu.ID,
			SkuID:      sku.ID,
			Count:      reqItem.Count,
			CartID:     reqItem.CartID,
			Selected:   reqItem.Selected,
			Price:      sku.Price,
			PayPrice:   sku.Price * reqItem.Count,
			SpuName:    spu.Name,
			PicURL:     sku.PicURL,
			CategoryID: spu.CategoryID,
			GivePoint:  spu.GiveIntegral * reqItem.Count,
			Properties: skuResp.Properties,
		}

		// 如果 SKU 没有图片，使用 SPU 的图片
		if item.PicURL == "" {
			item.PicURL = spu.PicURL
		}

		// 设置配送方式和配送模板
		if len(spu.DeliveryTypes) > 0 {
			item.DeliveryTypes = spu.DeliveryTypes
		}
		item.DeliveryTemplateID = spu.DeliveryTemplateID
		// 设置重量和体积（对齐 Java OrderItem，用于计费方式计算）
		item.Weight = skuResp.Weight
		item.Volume = skuResp.Volume

		resp.Items = append(resp.Items, item)

		s.logger.Debug("构建商品项",
			zap.Int64("skuId", item.SkuID),
			zap.String("spuName", item.SpuName),
			zap.Int("price", item.Price),
			zap.Int("count", item.Count),
			zap.Int("payPrice", item.PayPrice),
		)
	}

	s.logger.Info("商品项响应构建完成",
		zap.Int("itemCount", len(resp.Items)),
	)

	return nil
}

// validateResponse 验证响应结果
// 对应 Java: TradePriceServiceImpl#calculateOrderPrice 的最终验证
func (s *TradePriceService) validateResponse(req *TradePriceCalculateReqBO, resp *TradePriceCalculateRespBO) error {
	// 验证商品项数量一致性
	if len(resp.Items) == 0 {
		return pkgErrors.NewBizError(1011900001, "商品项不能为空")
	}

	// 验证支付金额（积分订单允许支付金额为0）
	if req.PointActivityId == 0 && resp.Price.PayPrice <= 0 {
		s.logger.Error("价格计算不正确",
			zap.Int64("userId", req.UserID),
			zap.Int("payPrice", resp.Price.PayPrice),
			zap.Int("totalPrice", resp.Price.TotalPrice),
			zap.Int("discountPrice", resp.Price.DiscountPrice),
		)
		return pkgErrors.NewBizError(1011003000, "支付金额不合法")
	}

	return nil
}

// GetApplicableCalculators 获取适用于指定订单类型的计算器
func (s *TradePriceService) GetApplicableCalculators(orderType int) []PriceCalculator {
	applicable := make([]PriceCalculator, 0)

	for _, calculator := range s.calculators {
		if calculator.IsApplicable(orderType) {
			applicable = append(applicable, calculator)
		}
	}

	return applicable
}

// CalculateProductPrice 计算商品价格
// 对应 Java: TradePriceServiceImpl#calculateProductPrice
// 返回 []AppTradeProductSettlementResp，每个 SPU 对应一个结算信息
func (s *TradePriceService) CalculateProductPrice(ctx context.Context, userId int64, spuIds []int64) ([]AppTradeProductSettlementRespBO, error) {
	s.logger.Info("开始计算商品价格",
		zap.Int64("userId", userId),
		zap.Int("spuCount", len(spuIds)),
	)

	if len(spuIds) == 0 {
		return []AppTradeProductSettlementRespBO{}, nil
	}

	// 1. 获取所有 SPU 的 SKU 列表，并按 SPU 分组
	spuIdAndSkuListMap := make(map[int64][]*productModel.ProductSku)
	var allSkuIDs []int64
	for _, spuId := range spuIds {
		skus, err := s.skuSvc.GetSkuListBySpuId(ctx, spuId)
		if err != nil {
			s.logger.Warn("获取SPU的SKU失败", zap.Int64("spuId", spuId), zap.Error(err))
			continue
		}
		spuIdAndSkuListMap[spuId] = skus
		for _, sku := range skus {
			allSkuIDs = append(allSkuIDs, sku.ID)
		}
	}

	// 2. 获取满减送活动 Map（仍需）
	var rewardActivityMap map[int64]*promotionModel.PromotionRewardActivity
	if s.rewardActivitySvc != nil {
		var err error
		rewardActivityMap, err = s.rewardActivitySvc.GetRewardActivityMapBySpuIds(ctx, spuIds)
		if err != nil {
			s.logger.Warn("获取满减送活动失败", zap.Error(err))
		}
	}
	if rewardActivityMap == nil {
		rewardActivityMap = make(map[int64]*promotionModel.PromotionRewardActivity)
	}

	// 注: discountMap 和 memberLevel 的获取已移至 SkuPromotionCalculator 接口内部
	// 这减少了代码重复并统一了优惠计算逻辑

	// 4. 针对每个 SPU 构建结果
	result := make([]AppTradeProductSettlementRespBO, 0, len(spuIds))
	for _, spuId := range spuIds {
		skuList := spuIdAndSkuListMap[spuId]
		if len(skuList) == 0 {
			continue
		}

		// 构建 SPU 结算信息
		spuVO := AppTradeProductSettlementRespBO{
			SpuID: spuId,
			Skus:  make([]AppTradeProductSettlementSkuBO, 0, len(skuList)),
		}

		// 为每个 SKU 计算优惠价格
		for _, sku := range skuList {
			skuVO := AppTradeProductSettlementSkuBO{
				ID:             sku.ID,
				PromotionPrice: sku.Price, // 默认为原价
			}

			// 使用 SkuPromotionCalculator 接口计算优惠
			// 这复用了 DiscountActivityPriceCalculator 的逻辑，避免代码重复
			if s.skuPromotionCalc != nil {
				result, err := s.skuPromotionCalc.CalculateSkuPromotion(ctx, userId, sku.ID, sku.Price)
				s.logger.Info("CalculateSkuPromotion 调用结果",
					zap.Int64("skuId", sku.ID),
					zap.Int("price", sku.Price),
					zap.Bool("resultNil", result == nil),
					zap.Error(err),
				)
				if err != nil {
					s.logger.Warn("计算 SKU 优惠失败", zap.Int64("skuId", sku.ID), zap.Error(err))
				} else if result != nil {
					skuVO.PromotionPrice = result.PromotionPrice
					skuVO.PromotionType = result.PromotionType
					skuVO.PromotionID = result.PromotionID
					skuVO.PromotionEndTime = result.PromotionEndTime
				}
			} else {
				s.logger.Warn("skuPromotionCalc 为 nil，无法计算优惠", zap.Int64("skuId", sku.ID))
			}

			spuVO.Skus = append(spuVO.Skus, skuVO)
		}

		// 获取满减送活动
		if rewardActivity, ok := rewardActivityMap[spuId]; ok {
			spuVO.RewardActivity = s.convertRewardActivityBO(rewardActivity)
		}

		result = append(result, spuVO)
	}

	return result, nil
}

// convertRewardActivityBO 转换满减送活动为 BO
func (s *TradePriceService) convertRewardActivityBO(activity *promotionModel.PromotionRewardActivity) *RewardActivityBO {
	if activity == nil {
		return nil
	}

	bo := &RewardActivityBO{
		ID:            activity.ID,
		ConditionType: activity.ConditionType,
		Rules:         make([]RewardActivityRuleBO, 0),
	}

	// 解析 Rules JSON 字符串
	if activity.Rules != "" {
		type RuleJSON struct {
			Limit                    int           `json:"limit"`
			DiscountPrice            int           `json:"discountPrice"`
			FreeDelivery             bool          `json:"freeDelivery"`
			Point                    int           `json:"point"`
			GiveCouponTemplateCounts map[int64]int `json:"giveCouponTemplateCounts"`
		}
		var rules []RuleJSON
		if err := json.Unmarshal([]byte(activity.Rules), &rules); err == nil {
			for _, rule := range rules {
				ruleBo := RewardActivityRuleBO{
					Limit:         rule.Limit,
					DiscountPrice: rule.DiscountPrice,
					FreeDelivery:  rule.FreeDelivery,
					Point:         rule.Point,
				}
				if rule.GiveCouponTemplateCounts != nil {
					ruleBo.GiveCouponTemplateCounts = rule.GiveCouponTemplateCounts
				} else {
					ruleBo.GiveCouponTemplateCounts = make(map[int64]int)
				}
				bo.Rules = append(bo.Rules, ruleBo)
			}
		}
	}

	return bo
}
