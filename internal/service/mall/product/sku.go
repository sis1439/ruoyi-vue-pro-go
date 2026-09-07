package product

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
	product2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
)

type ProductSkuService struct {
	q                *query.Query
	propertySvc      *ProductPropertyService
	propertyValueSvc *ProductPropertyValueService
	spuSvc           *ProductSpuService
}

func NewProductSkuService(q *query.Query, propertySvc *ProductPropertyService, propertyValueSvc *ProductPropertyValueService) *ProductSkuService {
	s := &ProductSkuService{
		q:                q,
		propertySvc:      propertySvc,
		propertyValueSvc: propertyValueSvc,
	}
	s.propertySvc.SetSkuService(s)
	s.propertyValueSvc.SetSkuService(s)
	return s
}

func (s *ProductSkuService) SetSpuService(spuSvc *ProductSpuService) {
	s.spuSvc = spuSvc
}

// ValidateSkuList 校验 SKU 列表
func (s *ProductSkuService) ValidateSkuList(ctx context.Context, skus []*product2.ProductSkuSaveReq, specType bool) error {
	if len(skus) == 0 {
		return product.ErrSkuNotExists // 使用商品模块错误码
	}

	for _, sku := range skus {
		if sku == nil || sku.Price < 0 || sku.MarketPrice < 0 || sku.CostPrice < 0 || sku.Stock < 0 {
			return fmt.Errorf("SKU 参数不合法")
		}
	}
	if !specType && len(skus) != 1 {
		return fmt.Errorf("单规格商品只能有一个 SKU")
	}

	// 单规格，覆盖默认属性
	if !specType {
		skus[0].Properties = []product2.ProductSkuPropertyReq{
			{PropertyID: 0, PropertyName: "默认", ValueID: 0, ValueName: "默认"},
		}
		return nil
	}

	// 1. 校验属性项存在
	propertyIDs := make([]int64, 0)
	for _, sku := range skus {
		for _, p := range sku.Properties {
			propertyIDs = append(propertyIDs, p.PropertyID)
		}
	}
	propertyIDs = lo.Uniq(propertyIDs)

	existProperties, err := s.propertySvc.GetPropertyListByIds(ctx, propertyIDs)
	if err != nil {
		return err
	}
	if len(existProperties) != len(propertyIDs) {
		return product.ErrPropertyNotExists // 使用商品模块错误码
	}

	// 2. 校验，一个 SKU 下，没有重复的属性。校验方式是，遍历每个 SKU ，看看是否有重复的属性 propertyId
	propertyValues, err := s.propertyValueSvc.GetPropertyValueListByPropertyIds(ctx, propertyIDs)
	if err != nil {
		return err
	}
	propertyValueMap := lo.KeyBy(propertyValues, func(item *product.ProductPropertyValue) int64 {
		return item.ID
	})

	for _, sku := range skus {
		skuPropertyIDs := make([]int64, 0)
		for _, p := range sku.Properties {
			if val, ok := propertyValueMap[p.ValueID]; ok {
				skuPropertyIDs = append(skuPropertyIDs, val.PropertyID)
			}
		}
		if len(lo.Uniq(skuPropertyIDs)) != len(sku.Properties) {
			return product.ErrSkuPropertiesDuplicated // 使用商品模块错误码
		}
	}

	// 3. 再校验，每个 Sku 的属性值的数量，是一致的。
	attrValueIDsSize := len(skus[0].Properties)
	for i := 1; i < len(skus); i++ {
		if len(skus[i].Properties) != attrValueIDsSize {
			return product.ErrSpuAttrNumbersMustBeEquals // 使用商品模块错误码
		}
	}

	// 4. 最后校验，每个 Sku 之间不是重复的（对齐Java实现：使用Set<Set<Long>>方式）
	skuAttrValues := make(map[string]bool)
	for _, sku := range skus {
		valueIDSet := make(map[int64]bool)
		for _, p := range sku.Properties {
			valueIDSet[p.ValueID] = true
		}

		// 将valueIDSet转换为排序后的字符串作为key（对齐Java的Set<Set<Long>>逻辑）
		valueIDs := make([]int64, 0, len(valueIDSet))
		for id := range valueIDSet {
			valueIDs = append(valueIDs, id)
		}
		// 排序确保key一致性（对齐Java的Comparator.comparing）
		for i := 0; i < len(valueIDs)-1; i++ {
			for j := 0; j < len(valueIDs)-i-1; j++ {
				if valueIDs[j] > valueIDs[j+1] {
					valueIDs[j], valueIDs[j+1] = valueIDs[j+1], valueIDs[j]
				}
			}
		}

		// 生成key（对齐Java的Collectors.joining()）
		key := ""
		for _, id := range valueIDs {
			key += strconv.FormatInt(id, 10) + ","
		}

		if skuAttrValues[key] {
			return product.ErrSpuSkuNotDuplicate // 使用商品模块错误码
		}
		skuAttrValues[key] = true
	}

	return nil
}

// CreateSkuList 批量创建 SKU
func (s *ProductSkuService) CreateSkuList(ctx context.Context, spuID int64, skuReqs []*product2.ProductSkuSaveReq) error {
	if len(skuReqs) == 0 {
		return nil
	}
	skus := make([]*product.ProductSku, 0, len(skuReqs))
	for _, req := range skuReqs {
		if req == nil {
			return fmt.Errorf("SKU 参数不能为空")
		}
		sku := s.convertSkuReqToModel(spuID, req)
		sku.ID = 0
		skus = append(skus, sku)
	}
	return repo.InTransaction(ctx, s.q, func(ctx context.Context, q *query.Query) error { return q.ProductSku.WithContext(ctx).Create(skus...) })
}

// UpdateSkuList explicitly inserts new rows and updates existing rows owned by the SPU.
// Selecting writable fields preserves zero values without overwriting sales/audit data.
func (s *ProductSkuService) UpdateSkuList(ctx context.Context, spuID int64, skuReqs []*product2.ProductSkuSaveReq) error {
	return repo.InTransaction(ctx, s.q, func(ctx context.Context, q *query.Query) error {
		u := q.ProductSku
		existing, err := u.WithContext(ctx).Where(u.SpuID.Eq(spuID)).Find()
		if err != nil {
			return err
		}
		byID := make(map[int64]*product.ProductSku, len(existing))
		byProperties := make(map[string]int64, len(existing))
		for _, sku := range existing {
			byID[sku.ID] = sku
			byProperties[s.buildPropertyKey(sku)] = sku.ID
		}
		used := make(map[int64]bool, len(skuReqs))
		for _, req := range skuReqs {
			if req == nil {
				return fmt.Errorf("SKU 参数不能为空")
			}
			sku := s.convertSkuReqToModel(spuID, req)
			if sku.ID == 0 {
				sku.ID = byProperties[s.buildPropertyKey(sku)]
			}
			if sku.ID == 0 {
				if err := u.WithContext(ctx).Create(sku); err != nil {
					return err
				}
			} else {
				if byID[sku.ID] == nil || used[sku.ID] {
					return product.ErrSkuNotExists
				}
				info, err := u.WithContext(ctx).Where(u.ID.Eq(sku.ID), u.SpuID.Eq(spuID)).Select(u.Properties, u.Price, u.MarketPrice, u.CostPrice, u.BarCode, u.PicURL, u.Stock, u.Weight, u.Volume, u.FirstBrokeragePrice, u.SecondBrokeragePrice).Updates(sku)
				if err != nil {
					return err
				}
				if info.RowsAffected != 1 {
					return product.ErrSkuNotExists
				}
			}
			used[sku.ID] = true
		}
		var removed []int64
		for _, sku := range existing {
			if !used[sku.ID] {
				removed = append(removed, sku.ID)
			}
		}
		if len(removed) > 0 {
			if _, err := u.WithContext(ctx).Where(u.SpuID.Eq(spuID), u.ID.In(removed...)).Delete(); err != nil {
				return err
			}
		}
		return nil
	})
}

// buildPropertyKey 构建属性key（完全对齐Java ProductSkuConvert.buildPropertyKey方法）
func (s *ProductSkuService) buildPropertyKey(sku *product.ProductSku) string {
	if len(sku.Properties) == 0 {
		return ""
	}

	// 创建副本并按valueId排序（对齐Java第51-52行）
	properties := make([]product.ProductSkuProperty, len(sku.Properties))
	copy(properties, sku.Properties)

	// 简单排序（对齐Java的Comparator.comparing(ProductSkuDO.Property::getValueId)）
	for i := 0; i < len(properties)-1; i++ {
		for j := 0; j < len(properties)-i-1; j++ {
			if properties[j].ValueID > properties[j+1].ValueID {
				properties[j], properties[j+1] = properties[j+1], properties[j]
			}
		}
	}

	// 连接成字符串（对齐Java第53行的Collectors.joining()）
	var key strings.Builder
	for _, prop := range properties {
		key.WriteString(strconv.FormatInt(prop.ValueID, 10))
		key.WriteByte(',')
	}

	return key.String()
}

// DeleteSkuBySpuId 删除指定 SPU 的所有 SKU
func (s *ProductSkuService) DeleteSkuBySpuId(ctx context.Context, spuID int64) error {
	q := repo.QueryFromContext(ctx, s.q)
	_, err := q.ProductSku.WithContext(ctx).Where(q.ProductSku.SpuID.Eq(spuID)).Delete()
	return err
}

// GetSkuListBySpuId 获得 SPU 的 SKU 列表
func (s *ProductSkuService) GetSkuListBySpuId(ctx context.Context, spuID int64) ([]*product.ProductSku, error) {
	return s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.SpuID.Eq(spuID)).Find()
}

// GetSkuListBySpuIdForResp 获得 SPU 的 SKU 列表（返回响应格式）
func (s *ProductSkuService) GetSkuListBySpuIdForResp(ctx context.Context, spuID int64) ([]*product2.ProductSkuResp, error) {
	// 查询SKU列表
	skus, err := s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.SpuID.Eq(spuID)).Find()
	if err != nil {
		return nil, err
	}

	// 如果没有SKU，返回空数组而不是null（对齐Java版本）
	if len(skus) == 0 {
		return []*product2.ProductSkuResp{}, nil
	}

	// 转换为响应格式，确保规格属性数组的完整性
	result := make([]*product2.ProductSkuResp, len(skus))
	for i, sku := range skus {
		result[i] = s.convertSkuResp(sku)
	}

	return result, nil
}

// GetSkuListBySpuIds 获得多个 SPU 的 SKU 列表
func (s *ProductSkuService) GetSkuListBySpuIds(ctx context.Context, spuIDs []int64) ([]*product.ProductSku, error) {
	return s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.SpuID.In(spuIDs...)).Find()
}

// GetSkuList 获得 SKU 列表
func (s *ProductSkuService) GetSkuList(ctx context.Context, ids []int64) ([]*product2.ProductSkuResp, error) {
	if len(ids) == 0 {
		return []*product2.ProductSkuResp{}, nil
	}
	skus, err := s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	result := lo.Map(skus, func(item *product.ProductSku, _ int) *product2.ProductSkuResp {
		return s.convertSkuResp(item)
	})
	return result, nil
}

// UpdateSkuProperty 更新 SKU 属性名
func (s *ProductSkuService) UpdateSkuProperty(ctx context.Context, propertyID int64, propertyName string) (int64, error) {
	// 获取所有SKU（对齐Java实现：selectList()）
	skus, err := s.q.ProductSku.WithContext(ctx).Find()
	if err != nil {
		return 0, err
	}

	updateSkus := make([]*product.ProductSku, 0)
	for _, sku := range skus {
		changed := false
		for i, p := range sku.Properties {
			if p.PropertyID == propertyID {
				sku.Properties[i].PropertyName = propertyName
				changed = true
			}
		}
		if changed {
			updateSkus = append(updateSkus, sku)
		}
	}

	if len(updateSkus) == 0 {
		return 0, nil
	}

	// 批量更新SKU
	for _, sku := range updateSkus {
		if _, err := s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.ID.Eq(sku.ID)).Updates(sku); err != nil {
			return 0, err
		}
	}
	return int64(len(updateSkus)), nil
}

// UpdateSkuPropertyValue 更新 SKU 属性值名
func (s *ProductSkuService) UpdateSkuPropertyValue(ctx context.Context, valueID int64, valueName string) (int64, error) {
	// 获取所有SKU
	skus, err := s.q.ProductSku.WithContext(ctx).Find()
	if err != nil {
		return 0, err
	}

	updateSkus := make([]*product.ProductSku, 0)
	for _, sku := range skus {
		changed := false
		for i, p := range sku.Properties {
			if p.ValueID == valueID {
				sku.Properties[i].ValueName = valueName
				changed = true
			}
		}
		if changed {
			updateSkus = append(updateSkus, sku)
		}
	}

	if len(updateSkus) == 0 {
		return 0, nil
	}

	// 批量更新SKU
	for _, sku := range updateSkus {
		if _, err := s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.ID.Eq(sku.ID)).Updates(sku); err != nil {
			return 0, err
		}
	}
	return int64(len(updateSkus)), nil
}

// UpdateSkuStock 更新 SKU 库存（完全对齐Java实现）
func (s *ProductSkuService) UpdateSkuStock(ctx context.Context, updateReq *product2.ProductSkuUpdateStockReq) error {
	return repo.InTransaction(ctx, s.q, func(ctx context.Context, q *query.Query) error {
		// 更新 SKU 库存
		for _, item := range updateReq.Items {
			if item.IncrCount > 0 {
				// 增加库存：同时更新stock和sales_count（对齐Java updateStockIncr方法）
				result, err := q.ProductSku.WithContext(ctx).
					Where(q.ProductSku.ID.Eq(item.ID)).
					Updates(map[string]interface{}{
						"stock":       q.ProductSku.Stock.Add(int(item.IncrCount)),
						"sales_count": q.ProductSku.SalesCount.Sub(int(item.IncrCount)),
					})
				if err != nil {
					return err
				}
				if result.RowsAffected != 1 {
					return product.ErrSkuNotExists
				}
			} else if item.IncrCount < 0 {
				// 减少库存：检查库存充足性，同时更新stock和sales_count（对齐Java updateStockDecr方法）
				decrCount := -item.IncrCount // 取正数
				result, err := q.ProductSku.WithContext(ctx).
					Where(
						q.ProductSku.ID.Eq(item.ID),
						q.ProductSku.Stock.Gte(int(decrCount)),
					).
					Updates(map[string]interface{}{
						"stock":       q.ProductSku.Stock.Sub(int(decrCount)),
						"sales_count": q.ProductSku.SalesCount.Add(int(decrCount)),
					})
				if err != nil {
					return err
				}
				if result.RowsAffected == 0 {
					return product.ErrSkuStockNotEnough // 使用商品模块错误码
				}
			}
		}

		// 更新 SPU 库存（对齐Java第270-275行）
		if s.spuSvc != nil {
			// 获取SKU列表
			skuIDs := lo.Map(updateReq.Items, func(item product2.ProductSkuUpdateStockItemReq, _ int) int64 {
				return item.ID
			})
			skus, err := q.ProductSku.WithContext(ctx).Where(q.ProductSku.ID.In(skuIDs...)).Find()
			if err != nil {
				return err
			}

			// 构建SPU库存变化映射（对齐Java ProductSkuConvert.convertSpuStockMap）
			skuMap := lo.KeyBy(skus, func(sku *product.ProductSku) int64 { return sku.ID })
			spuStockIncr := make(map[int64]int)

			for _, item := range updateReq.Items {
				if sku, exists := skuMap[item.ID]; exists {
					spuStockIncr[sku.SpuID] += item.IncrCount
				}
			}

			// 调用SPU服务更新库存
			if err := s.spuSvc.UpdateSpuStock(ctx, spuStockIncr); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetSku 获得 SKU 信息
func (s *ProductSkuService) GetSku(ctx context.Context, id int64) (*product.ProductSku, error) {
	sku, err := s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.ID.Eq(id)).First()
	if err != nil {
		return nil, product.ErrSkuNotExists // 使用商品模块错误码
	}
	return sku, nil
}

// GetSkuDetail 获得 SKU 详情信息（返回响应格式）
func (s *ProductSkuService) GetSkuDetail(ctx context.Context, id int64) (*product2.ProductSkuResp, error) {
	// 查询SKU信息
	sku, err := s.q.ProductSku.WithContext(ctx).Where(s.q.ProductSku.ID.Eq(id)).First()
	if err != nil {
		return nil, product.ErrSkuNotExists // 使用商品模块错误码
	}

	// 转换为响应格式，确保数据完整性
	return s.convertSkuResp(sku), nil
}

func (s *ProductSkuService) convertSkuReqToModel(spuID int64, req *product2.ProductSkuSaveReq) *product.ProductSku {
	properties := make([]product.ProductSkuProperty, len(req.Properties))
	for i, p := range req.Properties {
		properties[i] = product.ProductSkuProperty{
			PropertyID:   p.PropertyID,
			PropertyName: p.PropertyName,
			ValueID:      p.ValueID,
			ValueName:    p.ValueName,
		}
	}

	return &product.ProductSku{
		ID:                   req.ID,
		SpuID:                spuID,
		Properties:           properties,
		Price:                req.Price,
		MarketPrice:          req.MarketPrice,
		CostPrice:            req.CostPrice,
		BarCode:              req.BarCode,
		PicURL:               req.PicURL,
		Stock:                req.Stock,
		Weight:               req.Weight,
		Volume:               req.Volume,
		FirstBrokeragePrice:  req.FirstBrokeragePrice,
		SecondBrokeragePrice: req.SecondBrokeragePrice,
	}
}

func (s *ProductSkuService) convertSkuResp(sku *product.ProductSku) *product2.ProductSkuResp {
	// 确保规格属性数组的完整性，空数组返回[]而不是null
	properties := make([]product2.ProductSkuPropertyResp, 0)
	if len(sku.Properties) > 0 {
		properties = make([]product2.ProductSkuPropertyResp, len(sku.Properties))
		for i, p := range sku.Properties {
			properties[i] = product2.ProductSkuPropertyResp{
				PropertyID:   p.PropertyID,
				PropertyName: p.PropertyName,
				ValueID:      p.ValueID,
				ValueName:    p.ValueName,
			}
		}
	}

	// 生成SKU名称：从属性值拼接，如"黑色 - CH510"
	var skuName string
	if len(sku.Properties) > 0 {
		var names []string
		for _, p := range sku.Properties {
			if p.ValueName != "" {
				names = append(names, p.ValueName)
			}
		}
		if len(names) > 0 {
			skuName = strings.Join(names, " - ")
		} else {
			skuName = "默认规格"
		}
	} else {
		skuName = "默认规格"
	}

	// 确保字段不为null
	picURL := sku.PicURL
	if picURL == "" {
		picURL = ""
	}

	barCode := sku.BarCode
	if barCode == "" {
		barCode = ""
	}

	return &product2.ProductSkuResp{
		ID:                   sku.ID,
		Name:                 skuName,
		SpuID:                sku.SpuID,
		Properties:           properties,
		Price:                sku.Price,
		MarketPrice:          sku.MarketPrice,
		CostPrice:            sku.CostPrice,
		BarCode:              barCode,
		PicURL:               picURL,
		Stock:                sku.Stock,
		Weight:               sku.Weight,
		Volume:               sku.Volume,
		FirstBrokeragePrice:  sku.FirstBrokeragePrice,
		SecondBrokeragePrice: sku.SecondBrokeragePrice,
		SalesCount:           sku.SalesCount,
	}
}
