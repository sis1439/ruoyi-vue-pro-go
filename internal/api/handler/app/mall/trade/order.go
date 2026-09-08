package trade

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	trade2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppTradeOrderHandler struct {
	svc          *trade.TradeOrderUpdateService
	querySvc     *trade.TradeOrderQueryService
	afterSaleSvc *trade.TradeAfterSaleService
	priceSvc     *trade.TradePriceService
}

func NewAppTradeOrderHandler(
	svc *trade.TradeOrderUpdateService,
	querySvc *trade.TradeOrderQueryService,
	afterSaleSvc *trade.TradeAfterSaleService,
	priceSvc *trade.TradePriceService,
) *AppTradeOrderHandler {
	return &AppTradeOrderHandler{
		svc:          svc,
		querySvc:     querySvc,
		afterSaleSvc: afterSaleSvc,
		priceSvc:     priceSvc,
	}
}

// SettlementOrder 获得订单结算信息
func (h *AppTradeOrderHandler) SettlementOrder(c *gin.Context) {
	var r trade2.AppTradeOrderSettlementReq
	// 支持 POST (JSON) 和 GET (Query) 以对齐多样化的 App/Web 使用场景
	if c.Request.Method == "POST" {
		if err := c.ShouldBindJSON(&r); err != nil {
			response.WriteBizError(c, errors.ErrParam)
			return
		}
	} else {
		if err := c.ShouldBindQuery(&r); err != nil {
			// fmt.Printf("SettlementOrder Bind Error: %v\n", err)
		}
		// 支持 Spring 风格的 items[0].skuId 形式
		if len(r.Items) == 0 {
			r.Items = h.parseOrderItemsFromQuery(c.Request.URL.Query())
		}
		// 备选方案: 支持 skuIds=1,2&counts=1,1 形式
		if len(r.Items) == 0 {
			var q trade2.AppTradeOrderSettlementQueryReq
			if err := c.ShouldBindQuery(&q); err == nil {
				r.Items = q.ToSettlementItems()
			}
		}

		// 验证基础字段 (PointStatus, DeliveryType)
		if r.PointStatus == nil {
			response.WriteError(c, 400, "参数错误: pointStatus 缺失")
			return
		}
		if r.DeliveryType != 0 && r.DeliveryType != 1 && r.DeliveryType != 2 {
			response.WriteError(c, 400, "参数错误: deliveryType 缺失或无效")
			return
		}
	}

	// 最终验证: 必须有商品
	if len(r.Items) == 0 {
		response.WriteError(c, 400, "参数错误: items 缺失")
		return
	}

	res, err := h.svc.SettlementOrder(c, context.GetUserId(c), &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}

// CreateOrder 创建订单
func (h *AppTradeOrderHandler) CreateOrder(c *gin.Context) {
	var r trade2.AppTradeOrderCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	terminal, _ := strconv.Atoi(c.GetHeader("terminal"))
	res, err := h.svc.CreateOrder(c, context.GetUserId(c), c.ClientIP(), terminal, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 严格对齐 Java: 返回 AppTradeOrderCreateResp
	vo := trade2.AppTradeOrderCreateResp{
		ID:         res.ID,
		PayOrderID: 0, // 默认 0
	}
	if res.PayOrderID != nil {
		vo.PayOrderID = *res.PayOrderID
	}

	response.WriteSuccess(c, vo)
}

// UpdateOrderPaid 更新订单为已支付
// 这是一个回调接口,通常由 Pay 模块通过 HTTP 调用
func (h *AppTradeOrderHandler) UpdateOrderPaid(c *gin.Context) {
	var r pay.PayOrderNotifyReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	id := utils.ParseInt64(r.MerchantOrderId)
	if id == 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	err := h.svc.UpdateOrderPaid(c, id, r.PayOrderID)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// GetOrderDetail 获得订单详情
func (h *AppTradeOrderHandler) GetOrderDetail(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	userId := context.GetUserId(c)

	// 0. sync=true：先主动同步一次支付状态（归属与频率在 Service 内校验）
	if c.Query("sync") == "true" {
		_ = h.svc.SyncOrderPayStatus(c.Request.Context(), userId, id)
	}

	// 1. 获得订单
	order, err := h.querySvc.GetOrder(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	if order == nil || order.UserID != userId {
		response.WriteBizError(c, errors.ErrNotFound)
		return
	}

	// 2. 获得订单项
	items, err := h.querySvc.GetOrderItemListByOrderId(c, order.ID)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 3. 拼接数据
	itemResps := make([]trade2.AppTradeOrderItemResp, len(items))
	for i, item := range items {
		properties := make([]product.ProductSkuPropertyResp, len(item.Properties))
		for j, p := range item.Properties {
			properties[j] = product.ProductSkuPropertyResp{
				PropertyID:   p.PropertyID,
				PropertyName: p.PropertyName,
				ValueID:      p.ValueID,
				ValueName:    p.ValueName,
			}
		}

		itemResps[i] = trade2.AppTradeOrderItemResp{
			ID:              item.ID,
			OrderID:         item.OrderID,
			SpuID:           item.SpuID,
			SpuName:         item.SpuName,
			SkuID:           item.SkuID,
			PicURL:          item.PicURL,
			Count:           item.Count,
			CommentStatus:   bool(item.CommentStatus),
			Price:           item.Price,
			PayPrice:        item.PayPrice,
			AfterSaleID:     item.AfterSaleID,
			AfterSaleStatus: item.AfterSaleStatus,
			Properties:      properties,
		}
	}

	var payOrderID int64
	if order.PayOrderID != nil {
		payOrderID = *order.PayOrderID
	}

	res := trade2.AppTradeOrderDetailResp{
		ID:                    order.ID,
		No:                    order.No,
		Type:                  order.Type,
		CreateTime:            types.ToJsonDateTime(order.CreateTime),
		UserRemark:            order.UserRemark,
		Status:                order.Status,
		ProductCount:          order.ProductCount,
		FinishTime:            types.ToJsonDateTimePtr(order.FinishTime),
		CancelTime:            types.ToJsonDateTimePtr(order.CancelTime),
		CommentStatus:         bool(order.CommentStatus),
		PayStatus:             bool(order.PayStatus),
		PayOrderID:            payOrderID,
		PayTime:               types.ToJsonDateTimePtr(order.PayTime),
		PayChannelCode:        order.PayChannelCode,
		TotalPrice:            order.TotalPrice,
		DiscountPrice:         order.DiscountPrice,
		DeliveryPrice:         order.DeliveryPrice,
		AdjustPrice:           order.AdjustPrice,
		PayPrice:              order.PayPrice,
		DeliveryType:          order.DeliveryType,
		LogisticsID:           order.LogisticsID,
		LogisticsNo:           order.LogisticsNo,
		DeliveryTime:          types.ToJsonDateTimePtr(order.DeliveryTime),
		ReceiveTime:           types.ToJsonDateTimePtr(order.ReceiveTime),
		ReceiverName:          order.ReceiverName,
		ReceiverMobile:        order.ReceiverMobile,
		ReceiverAreaID:        order.ReceiverAreaID,
		ReceiverDetailAddress: order.ReceiverDetailAddress,
		RefundStatus:          order.RefundStatus,
		RefundPrice:           order.RefundPrice,
		CouponID:              order.CouponID,
		CouponPrice:           order.CouponPrice,
		PointPrice:            order.PointPrice,
		VipPrice:              order.VipPrice,
		CombinationRecordID:   order.CombinationRecordID,
		Items:                 itemResps,
	}

	if err := h.querySvc.FillAppOrderDetail(c, order, &res); err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, res)
}

// GetOrderPage 获得订单分页
func (h *AppTradeOrderHandler) GetOrderPage(c *gin.Context) {
	var r trade2.AppTradeOrderPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteError(c, 400, err.Error())
		return
	}
	// 1. 获得分页列表
	pageResult, err := h.querySvc.GetOrderPage(c, context.GetUserId(c), &r)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}

	// 2. 获得订单项
	if len(pageResult.List) > 0 {
		orderIds := make([]int64, len(pageResult.List))
		for i, o := range pageResult.List {
			orderIds[i] = o.ID
		}
		items, err := h.querySvc.GetOrderItemListByOrderIds(c, orderIds)
		if err != nil {
			// 记录日志，但此处继续返回，或返回错误
			response.WriteError(c, 500, err.Error())
			return
		}

		// 根据 OrderID 映射订单项
		itemMap := make(map[int64][]trade2.AppTradeOrderItemResp)
		for _, item := range items {
			properties := make([]product.ProductSkuPropertyResp, len(item.Properties))
			for j, p := range item.Properties {
				properties[j] = product.ProductSkuPropertyResp{
					PropertyID:   p.PropertyID,
					PropertyName: p.PropertyName,
					ValueID:      p.ValueID,
					ValueName:    p.ValueName,
				}
			}

			itemResp := trade2.AppTradeOrderItemResp{
				ID:              item.ID,
				OrderID:         item.OrderID,
				SpuID:           item.SpuID,
				SpuName:         item.SpuName,
				SkuID:           item.SkuID,
				PicURL:          item.PicURL,
				Count:           item.Count,
				CommentStatus:   bool(item.CommentStatus),
				Price:           item.Price,
				PayPrice:        item.PayPrice,
				AfterSaleID:     item.AfterSaleID,
				AfterSaleStatus: item.AfterSaleStatus,
				Properties:      properties,
			}
			itemMap[item.OrderID] = append(itemMap[item.OrderID], itemResp)
		}

		// 3. 拼接返回 VO
		list := make([]trade2.AppTradeOrderPageItemResp, len(pageResult.List))
		for i, o := range pageResult.List {
			var payOrderID int64
			if o.PayOrderID != nil {
				payOrderID = *o.PayOrderID
			}
			list[i] = trade2.AppTradeOrderPageItemResp{
				ID:                  o.ID,
				No:                  o.No,
				Type:                o.Type,
				Status:              o.Status,
				ProductCount:        o.ProductCount,
				CommentStatus:       bool(o.CommentStatus),
				CreateTime:          types.ToJsonDateTime(o.CreateTime),
				PayOrderID:          payOrderID,
				PayPrice:            o.PayPrice,
				DeliveryType:        o.DeliveryType,
				Items:               itemMap[o.ID],
				CombinationRecordID: o.CombinationRecordID,
			}
		}

		response.WriteSuccess(c, pagination.PageResult[trade2.AppTradeOrderPageItemResp]{
			List:  list,
			Total: pageResult.Total,
		})
	} else {
		response.WriteSuccess(c, pagination.PageResult[trade2.AppTradeOrderPageItemResp]{
			List:  []trade2.AppTradeOrderPageItemResp{},
			Total: 0,
		})
	}
}

// CancelOrder 取消订单
func (h *AppTradeOrderHandler) CancelOrder(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteError(c, 400, "id is required")
		return
	}
	err := h.svc.CancelOrder(c, context.GetUserId(c), id)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, true)
}

// DeleteOrder 删除订单
func (h *AppTradeOrderHandler) DeleteOrder(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteError(c, 400, "id is required")
		return
	}
	err := h.svc.DeleteOrder(c, context.GetUserId(c), id)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, true)
}

// GetOrderCount 获得交易订单数量
func (h *AppTradeOrderHandler) GetOrderCount(c *gin.Context) {
	// 简单的 Map 返回，对齐 Java
	uid := context.GetUserId(c)
	res := make(map[string]int64)

	// 获取数量辅助函数
	getCount := func(status *int, commentStatus *bool) int64 {
		count, _ := h.querySvc.GetOrderCount(c, uid, status, commentStatus)
		return count
	}

	// 常量
	// UNPAID(0), UNDELIVERED(10), DELIVERED(20), COMPLETED(30), CANCELLED(40)
	unpaid := tradeModel.TradeOrderStatusUnpaid
	undelivered := tradeModel.TradeOrderStatusUndelivered
	delivered := tradeModel.TradeOrderStatusDelivered
	completed := tradeModel.TradeOrderStatusCompleted
	commentStatus := false

	res["allCount"] = getCount(nil, nil)
	res["unpaidCount"] = getCount(&unpaid, nil)
	res["undeliveredCount"] = getCount(&undelivered, nil)
	res["deliveredCount"] = getCount(&delivered, nil)
	res["uncommentedCount"] = getCount(&completed, &commentStatus)

	// 售后数量 (需要 TradeAfterSaleService)
	afterSaleCount, _ := h.afterSaleSvc.GetUserAfterSaleCount(c, uid)
	res["afterSaleCount"] = afterSaleCount

	response.WriteSuccess(c, res)
}

// CreateOrderItemComment 创建订单项评价
func (h *AppTradeOrderHandler) CreateOrderItemComment(c *gin.Context) {
	var r trade2.AppTradeOrderItemCommentCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteError(c, 400, err.Error())
		return
	}
	res, err := h.svc.CreateOrderItemCommentByMember(c, context.GetUserId(c), &r)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, res)
}

// GetOrderExpressTrackList 获得物流轨迹
func (h *AppTradeOrderHandler) GetOrderExpressTrackList(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteError(c, 400, "id is required")
		return
	}
	res, err := h.querySvc.GetExpressTrackList(c, id, context.GetUserId(c))
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, res)
}

// ReceiveOrder 确认收货
func (h *AppTradeOrderHandler) ReceiveOrder(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteError(c, 400, "id is required")
		return
	}
	err := h.svc.ReceiveOrder(c, context.GetUserId(c), id)
	if err != nil {
		response.WriteError(c, 500, err.Error())
		return
	}
	response.WriteSuccess(c, true)
}

// SettlementProduct 获得商品结算信息
func (h *AppTradeOrderHandler) SettlementProduct(c *gin.Context) {
	// Try parsing from string "spuIds" if user sends "1,2,3"
	str := c.Query("spuIds")
	var spuIds types.ListFromCSV[int64]
	if str != "" {
		if err := spuIds.Scan(str); err != nil {
			response.WriteBizError(c, errors.ErrParam)
			return
		}
	}
	res, err := h.priceSvc.CalculateProductPrice(c, context.GetUserId(c), spuIds)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}

// GetOrderItem 获得交易订单项
func (h *AppTradeOrderHandler) GetOrderItem(c *gin.Context) {
	id := utils.ParseInt64(c.Query("id"))
	if id == 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	item, err := h.querySvc.GetOrderItem(c, context.GetUserId(c), id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	properties := make([]product.ProductSkuPropertyResp, len(item.Properties))
	for j, p := range item.Properties {
		properties[j] = product.ProductSkuPropertyResp{
			PropertyID:   p.PropertyID,
			PropertyName: p.PropertyName,
			ValueID:      p.ValueID,
			ValueName:    p.ValueName,
		}
	}

	res := trade2.AppTradeOrderItemResp{
		ID:              item.ID,
		OrderID:         item.OrderID,
		SpuID:           item.SpuID,
		SpuName:         item.SpuName,
		SkuID:           item.SkuID,
		PicURL:          item.PicURL,
		Count:           item.Count,
		CommentStatus:   bool(item.CommentStatus),
		Price:           item.Price,
		PayPrice:        item.PayPrice,
		AfterSaleID:     item.AfterSaleID,
		AfterSaleStatus: item.AfterSaleStatus,
		Properties:      properties,
	}
	response.WriteSuccess(c, res)
}

// parseOrderItemsFromQuery 解析 items[0].skuId 格式的查询参数
func (h *AppTradeOrderHandler) parseOrderItemsFromQuery(q url.Values) []trade2.AppTradeOrderSettlementItem {
	itemsMap := make(map[int]*trade2.AppTradeOrderSettlementItem)
	indices := make([]int, 0)

	for k, v := range q {
		if !strings.HasPrefix(k, "items[") {
			continue
		}
		// 解析 items[0].skuId
		closeBracket := strings.Index(k, "]")
		if closeBracket < 0 {
			continue
		}
		indexStr := k[6:closeBracket]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			continue
		}

		if _, ok := itemsMap[index]; !ok {
			itemsMap[index] = &trade2.AppTradeOrderSettlementItem{}
			indices = append(indices, index)
		}

		prop := ""
		if len(k) > closeBracket+1 {
			prop = k[closeBracket+1:]
			prop = strings.TrimPrefix(prop, ".")
			prop = strings.TrimPrefix(prop, "[")
			prop = strings.TrimSuffix(prop, "]")
		}

		val := ""
		if len(v) > 0 {
			val = v[0]
		}

		switch prop {
		case "skuId":
			itemsMap[index].SkuID = utils.ParseInt64(val)
		case "count":
			itemsMap[index].Count, _ = strconv.Atoi(val)
		case "cartId":
			itemsMap[index].CartID = utils.ParseInt64(val)
		}
	}

	if len(indices) == 0 {
		return nil
	}

	// 排序索引以保证顺序一致性
	sort.Ints(indices)
	res := make([]trade2.AppTradeOrderSettlementItem, 0, len(indices))
	for _, idx := range indices {
		res = append(res, *itemsMap[idx])
	}
	return res
}
