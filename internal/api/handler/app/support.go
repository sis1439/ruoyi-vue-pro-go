package app

import (
	"github.com/gin-gonic/gin"
	dto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app"
	adminInfra "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/infra"
	adminSystem "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/system"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/pkg/area"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/infra"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"math"
	"sort"
	"strconv"
	"strings"
)

// SupportHandler exposes app contracts using the existing infrastructure and delivery services.
type SupportHandler struct {
	*adminInfra.FileHandler
	*adminSystem.AreaHandler
	dict      *system.DictService
	express   *trade.DeliveryExpressService
	stores    *trade.DeliveryPickUpStoreService
	afterSale *trade.TradeAfterSaleService
	logs      *trade.AfterSaleLogService
}

func NewSupportHandler(files *infra.FileService, dict *system.DictService, express *trade.DeliveryExpressService, stores *trade.DeliveryPickUpStoreService, afterSale *trade.TradeAfterSaleService, logs *trade.AfterSaleLogService) *SupportHandler {
	return &SupportHandler{FileHandler: adminInfra.NewFileHandler(files), AreaHandler: adminSystem.NewAreaHandler(), dict: dict, express: express, stores: stores, afterSale: afterSale, logs: logs}
}
func (h *SupportHandler) GetDictData(c *gin.Context) {
	list, err := h.dict.GetSimpleDictDataList(c)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	result := make([]dto.DictDataResp, 0)
	for _, item := range list {
		if item.DictType == c.Query("type") {
			result = append(result, dto.DictDataResp{Label: item.Label, Value: item.Value})
		}
	}
	response.WriteSuccess(c, result)
}
func (h *SupportHandler) GetExpressList(c *gin.Context) {
	list, err := h.express.GetSimpleDeliveryExpressList(c)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	result := make([]dto.DeliveryExpressResp, 0, len(list))
	for _, item := range list {
		result = append(result, dto.DeliveryExpressResp{ID: item.ID, Name: item.Name})
	}
	response.WriteSuccess(c, result)
}
func storeResponse(item *tradeModel.TradeDeliveryPickUpStore, latitude, longitude *float64) dto.DeliveryPickUpStoreResp {
	var distance *float64
	if latitude != nil && longitude != nil {
		rad := math.Pi / 180
		dlat := (item.Latitude - *latitude) * rad
		dlon := (item.Longitude - *longitude) * rad
		a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(*latitude*rad)*math.Cos(item.Latitude*rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
		km := 6371 * 2 * math.Asin(math.Sqrt(math.Min(1, a)))
		distance = &km
	}
	return dto.DeliveryPickUpStoreResp{ID: item.ID, Name: item.Name, Logo: item.Logo, Phone: item.Phone, AreaID: item.AreaID, AreaName: area.Format(item.AreaID), DetailAddress: item.DetailAddress, OpeningTime: string(item.OpeningTime)[:min(5, len(item.OpeningTime))], ClosingTime: string(item.ClosingTime)[:min(5, len(item.ClosingTime))], Latitude: item.Latitude, Longitude: item.Longitude, Distance: distance}
}
func (h *SupportHandler) GetPickUpStoreList(c *gin.Context) {
	var req struct {
		Latitude  *float64 `form:"latitude" binding:"omitempty,gte=-90,lte=90"`
		Longitude *float64 `form:"longitude" binding:"omitempty,gte=-180,lte=180"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	list, err := h.stores.GetSimpleDeliveryPickUpStoreList(c)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	result := make([]dto.DeliveryPickUpStoreResp, 0, len(list))
	for _, item := range list {
		if name := c.Query("name"); name == "" || strings.Contains(item.Name, name) {
			result = append(result, storeResponse(item, req.Latitude, req.Longitude))
		}
	}
	if req.Latitude != nil && req.Longitude != nil {
		sort.SliceStable(result, func(i, j int) bool { return *result[i].Distance < *result[j].Distance })
	}
	response.WriteSuccess(c, result)
}
func (h *SupportHandler) GetPickUpStore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil || id <= 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	item, err := h.stores.GetDeliveryPickUpStore(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	if item == nil || item.Status != 0 {
		response.WriteSuccess(c, nil)
		return
	}
	response.WriteSuccess(c, storeResponse(item, nil, nil))
}
func (h *SupportHandler) GetAfterSaleLogs(c *gin.Context) {
	id, err := strconv.ParseInt(c.Query("afterSaleId"), 10, 64)
	if err != nil || id <= 0 {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	if _, err = h.afterSale.GetAfterSale(c, context.GetUserId(c), id); err != nil {
		response.WriteBizError(c, err)
		return
	}
	list, err := h.logs.GetAfterSaleLogList(c, id)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	result := make([]dto.AfterSaleLogResp, 0, len(list))
	for _, item := range list {
		result = append(result, dto.AfterSaleLogResp{ID: item.ID, Content: item.Content, CreateTime: types.ToJsonDateTime(item.CreateTime)})
	}
	response.WriteSuccess(c, result)
}
