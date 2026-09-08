package product

import (
	product2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"github.com/gin-gonic/gin"
)

type AppProductBrowseHistoryHandler struct {
	svc *product.ProductBrowseHistoryService
}

func NewAppProductBrowseHistoryHandler(svc *product.ProductBrowseHistoryService) *AppProductBrowseHistoryHandler {
	return &AppProductBrowseHistoryHandler{svc: svc}
}

// DeleteBrowseHistory 删除商品浏览记录
func (h *AppProductBrowseHistoryHandler) DeleteBrowseHistory(c *gin.Context) {
	var r product2.AppProductBrowseHistoryDeleteReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	userId := context.GetLoginUserID(c)
	if err := h.svc.HideUserBrowseHistory(c, userId, r.SpuIds); err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// CleanBrowseHistory 清空商品浏览记录
func (h *AppProductBrowseHistoryHandler) CleanBrowseHistory(c *gin.Context) {
	userId := context.GetLoginUserID(c)
	if err := h.svc.HideUserBrowseHistory(c, userId, nil); err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, true)
}

// GetBrowseHistoryPage 获得商品浏览记录分页
func (h *AppProductBrowseHistoryHandler) GetBrowseHistoryPage(c *gin.Context) {
	var r product2.AppProductBrowseHistoryPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	userId := context.GetLoginUserID(c)
	r.CreateTime = types.QueryTimeRange(c.Request.URL.Query(), "createTime")
	res, err := h.svc.GetAppBrowseHistoryPage(c, userId, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}
