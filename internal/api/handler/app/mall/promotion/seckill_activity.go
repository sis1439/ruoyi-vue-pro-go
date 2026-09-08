package promotion

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strconv"
	"time"

	adminProduct "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	promotionModel "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	productSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// AppSeckillActivityHandler App 端秒杀活动 Handler
type AppSeckillActivityHandler struct {
	svc       *promotion.SeckillActivityService
	configSvc *promotion.SeckillConfigService
	spuSvc    *productSvc.ProductSpuService
}

// NewAppSeckillActivityHandler 创建 Handler
func NewAppSeckillActivityHandler(
	svc *promotion.SeckillActivityService,
	configSvc *promotion.SeckillConfigService,
	spuSvc *productSvc.ProductSpuService,
) *AppSeckillActivityHandler {
	return &AppSeckillActivityHandler{svc: svc, configSvc: configSvc, spuSvc: spuSvc}
}

// GetNowSeckillActivity 获取当前正在进行的秒杀活动
// 对齐 Java: AppSeckillActivityController.getNowSeckillActivity
func (h *AppSeckillActivityHandler) GetNowSeckillActivity(c *gin.Context) {
	// 1. 获取当前正在进行的秒杀时段
	configs, err := h.configSvc.GetSeckillConfigListByStatus(c.Request.Context(), consts.CommonStatusEnable)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 2. 找到当前时间所在的时段
	now := time.Now()
	var currentConfig *promotion2.AppSeckillConfigResp
	for _, cfg := range configs {
		// 解析时间段 (格式: HH:mm:ss)
		startTime, _ := time.Parse("15:04:05", cfg.StartTime)
		endTime, _ := time.Parse("15:04:05", cfg.EndTime)

		// 构建今天的开始和结束时间
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), startTime.Second(), 0, now.Location())
		todayEnd := time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), 0, now.Location())

		if now.After(todayStart) && now.Before(todayEnd) {
			currentConfig = &promotion2.AppSeckillConfigResp{
				ID:            cfg.ID,
				StartTime:     cfg.StartTime,
				EndTime:       cfg.EndTime,
				SliderPicUrls: cfg.SliderPicUrls,
			}
			break
		}
	}

	if currentConfig == nil {
		// 没有当前进行中的时段
		response.WriteSuccess(c, promotion2.AppSeckillActivityNowResp{
			Config:     nil,
			Activities: []promotion2.AppSeckillActivityResp{},
		})
		return
	}

	// 3. 获取该时段的秒杀活动
	activities, err := h.svc.GetSeckillActivityListByConfigId(c.Request.Context(), currentConfig.ID, 10)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 4. 获取 SPU 信息
	spuIds := make([]int64, len(activities))
	for i, act := range activities {
		spuIds[i] = act.SpuID
	}
	spuList, _ := h.spuSvc.GetSpuList(c.Request.Context(), spuIds)
	spuMap := make(map[int64]*adminProduct.ProductSpuResp)
	for _, spu := range spuList {
		spuMap[spu.ID] = spu
	}

	// 5. 构建响应
	actResp := make([]promotion2.AppSeckillActivityResp, 0, len(activities))
	for _, act := range activities {
		spu, ok := spuMap[act.SpuID]
		if !ok || spu.Status != consts.ProductSpuStatusEnable {
			continue
		}

		// 获取最低秒杀价
		products, _ := h.svc.GetSeckillProductListByActivityID(c.Request.Context(), act.ID)
		minPrice := 0
		if len(products) > 0 {
			minPrice = products[0].SeckillPrice
			for _, p := range products {
				if p.SeckillPrice < minPrice {
					minPrice = p.SeckillPrice
				}
			}
		}
		actResp = append(actResp, promotion2.AppSeckillActivityResp{
			ID:           act.ID,
			Name:         act.Name,
			SpuID:        act.SpuID,
			SpuName:      spu.Name,
			PicURL:       spu.PicURL,
			MarketPrice:  spu.MarketPrice,
			SeckillPrice: minPrice,
			Status:       act.Status,
			Stock:        act.Stock,
			TotalStock:   act.TotalStock,
		})
	}

	response.WriteSuccess(c, promotion2.AppSeckillActivityNowResp{
		Config:     currentConfig,
		Activities: actResp,
	})
}

// GetSeckillActivityPage 获得秒杀活动分页
// 对齐 Java: AppSeckillActivityController.getSeckillActivityPage
func (h *AppSeckillActivityHandler) GetSeckillActivityPage(c *gin.Context) {
	var r promotion2.AppSeckillActivityPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, err.Error())
		return
	}

	// 查询分页数据
	result, err := h.svc.GetSeckillActivityPageForApp(c.Request.Context(), r.ConfigID, r.PageNo, r.PageSize)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 获取 SPU 信息
	spuIds := make([]int64, len(result.List))
	for i, act := range result.List {
		spuIds[i] = act.SpuID
	}
	spuList, _ := h.spuSvc.GetSpuList(c.Request.Context(), spuIds)
	spuMap := make(map[int64]*adminProduct.ProductSpuResp)
	for _, spu := range spuList {
		spuMap[spu.ID] = spu
	}

	// 构建响应
	list := make([]promotion2.AppSeckillActivityResp, 0, len(result.List))
	for _, act := range result.List {
		spu, ok := spuMap[act.SpuID]
		if !ok || spu.Status != consts.ProductSpuStatusEnable {
			continue
		}

		// 获取最低秒杀价
		products, _ := h.svc.GetSeckillProductListByActivityID(c.Request.Context(), act.ID)
		minPrice := 0
		if len(products) > 0 {
			minPrice = products[0].SeckillPrice
			for _, p := range products {
				if p.SeckillPrice < minPrice {
					minPrice = p.SeckillPrice
				}
			}
		}
		list = append(list, promotion2.AppSeckillActivityResp{
			ID:           act.ID,
			Name:         act.Name,
			SpuID:        act.SpuID,
			SpuName:      spu.Name,
			PicURL:       spu.PicURL,
			MarketPrice:  spu.MarketPrice,
			SeckillPrice: minPrice,
			Status:       act.Status,
			Stock:        act.Stock,
			TotalStock:   act.TotalStock,
		})
	}

	response.WriteSuccess(c, pagination.PageResult[promotion2.AppSeckillActivityResp]{
		List:  list,
		Total: result.Total,
	})
}

// GetSeckillActivity 获得秒杀活动
// 对齐 Java: AppSeckillActivityController.getSeckillActivity
func (h *AppSeckillActivityHandler) GetSeckillActivity(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	act, err := h.svc.GetSeckillActivity(c.Request.Context(), id)
	if err != nil || act == nil {
		response.WriteSuccess(c, nil)
		return
	}

	// 校验状态
	if act.Status != consts.CommonStatusEnable {
		response.WriteSuccess(c, nil)
		return
	}

	// Fetch SPU Info
	spu, _, err := h.spuSvc.GetSpuDetail(c.Request.Context(), act.SpuID)
	if err != nil || spu == nil || spu.Status != consts.ProductSpuStatusEnable {
		response.WriteBizError(c, errors.NewBizError(1001004003, "秒杀活动已结束或商品已下架"))
		return
	}

	// 获取商品
	products, _ := h.svc.GetSeckillProductListByActivityID(c.Request.Context(), id)

	// 构建响应
	detail := promotion2.AppSeckillActivityDetailResp{
		ID:               act.ID,
		Name:             act.Name,
		Status:           act.Status,
		SpuID:            act.SpuID,
		StartTime:        types.ToJsonDateTimePtr(&act.StartTime),
		EndTime:          types.ToJsonDateTimePtr(&act.EndTime),
		SingleLimitCount: act.SingleLimitCount,
		TotalLimitCount:  act.TotalLimitCount,
		Stock:            act.Stock,
		TotalStock:       act.TotalStock,
		SpuName:          spu.Name,
		PicURL:           spu.PicURL,
		MarketPrice:      spu.MarketPrice,
		Products:         make([]promotion2.AppSeckillProductResp, 0, len(products)),
	}

	// Calculate Min Seckill Price
	minPrice := 0
	if len(products) > 0 {
		minPrice = products[0].SeckillPrice
	}

	for _, p := range products {
		if p.SeckillPrice < minPrice {
			minPrice = p.SeckillPrice
		}
		detail.Products = append(detail.Products, promotion2.AppSeckillProductResp{
			ID:           p.ID,
			ActivityID:   p.ActivityID,
			SpuID:        p.SpuID,
			SkuID:        p.SkuID,
			SeckillPrice: p.SeckillPrice,
			Stock:        p.Stock,
		})
	}
	detail.SeckillPrice = minPrice

	response.WriteSuccess(c, detail)
}

// GetSeckillActivityDetail 获得秒杀活动详情
// 对齐 Java: AppSeckillActivityController.getSeckillActivity (get-detail路径)
func (h *AppSeckillActivityHandler) GetSeckillActivityDetail(c *gin.Context) {
	var r promotion2.AppSeckillActivityDetailReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, err.Error())
		return
	}

	detail, err := h.svc.GetSeckillActivityDetail(c.Request.Context(), r.ID)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, detail)
}

// GetSeckillActivityListByIds 按 ID 获取秒杀活动列表
// 对齐 Java: AppSeckillActivityController.getCombinationActivityListByIds (实际是秒杀)
func (h *AppSeckillActivityHandler) GetSeckillActivityListByIds(c *gin.Context) {
	idsStr := c.Query("ids")
	var ids []int64
	var intList model.IntListFromCSV
	if err := intList.Scan(idsStr); err != nil {
		response.WriteError(c, 400, "参数错误")
		return
	}
	for _, id := range intList {
		ids = append(ids, int64(id))
	}

	// 1. 获得开启的活动列表
	activityList, err := h.svc.GetSeckillActivityListByIds(c.Request.Context(), ids)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 过滤启用状态
	var enabledActivities []*promotionModel.PromotionSeckillActivity
	for _, act := range activityList {
		if act.Status == consts.CommonStatusEnable {
			enabledActivities = append(enabledActivities, act)
		}
	}

	if len(enabledActivities) == 0 {
		response.WriteSuccess(c, []promotion2.AppSeckillActivityResp{})
		return
	}

	// 2. 获取秒杀商品信息
	activityIds := make([]int64, len(enabledActivities))
	spuIds := make([]int64, len(enabledActivities))
	for i, act := range enabledActivities {
		activityIds[i] = act.ID
		spuIds[i] = act.SpuID
	}

	// 获取秒杀商品列表
	productList, err := h.svc.GetSeckillProductListByActivityIds(c.Request.Context(), activityIds)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 获取SPU信息
	spuList, err := h.spuSvc.GetSpuList(c.Request.Context(), spuIds)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 3. 构建SPU映射
	spuMap := make(map[int64]*adminProduct.ProductSpuResp)
	for _, spu := range spuList {
		spuMap[spu.ID] = spu
	}

	// 构建商品映射 (按活动ID分组)
	productMap := make(map[int64][]*promotionModel.PromotionSeckillProduct)
	for _, product := range productList {
		productMap[product.ActivityID] = append(productMap[product.ActivityID], product)
	}

	// 4. 构建响应
	activeList := make([]promotion2.AppSeckillActivityResp, 0, len(enabledActivities))
	for _, act := range enabledActivities {
		spu, ok := spuMap[act.SpuID]
		if !ok {
			continue
		}

		// 获取最低秒杀价格
		products := productMap[act.ID]
		var minSeckillPrice int
		if len(products) > 0 {
			minSeckillPrice = products[0].SeckillPrice
			for _, p := range products {
				if p.SeckillPrice < minSeckillPrice {
					minSeckillPrice = p.SeckillPrice
				}
			}
		}

		activeList = append(activeList, promotion2.AppSeckillActivityResp{
			ID:           act.ID,
			Name:         act.Name,
			SpuID:        act.SpuID,
			SpuName:      spu.Name,
			PicURL:       spu.PicURL,
			MarketPrice:  spu.MarketPrice,
			SeckillPrice: minSeckillPrice,
			Status:       act.Status,
			Stock:        act.Stock,
			TotalStock:   act.TotalStock,
		})
	}

	response.WriteSuccess(c, activeList)
}
