package promotion

import (
	app "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"strconv"

	"github.com/gin-gonic/gin"
	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"

	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
)

type AppArticleHandler struct {
	articleSvc  promotion.ArticleService
	categorySvc promotion.ArticleCategoryService
}

func NewAppArticleHandler(articleSvc promotion.ArticleService, categorySvc promotion.ArticleCategoryService) *AppArticleHandler {
	return &AppArticleHandler{articleSvc: articleSvc, categorySvc: categorySvc}
}

// GetArticleCategoryList 获得文章分类列表
func (h *AppArticleHandler) GetArticleCategoryList(c *gin.Context) {
	res, err := h.categorySvc.GetArticleCategorySimpleList(c)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, res)
}

// GetArticlePage 获得文章分页
func (h *AppArticleHandler) GetArticlePage(c *gin.Context) {
	var r promotion2.ArticlePageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	res, err := h.articleSvc.GetArticlePageApp(c, r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	list := make([]*app.AppArticleResp, 0, len(res.List))
	for _, item := range res.List {
		list = append(list, articleResponse(item))
	}
	response.WriteSuccess(c, &pagination.PageResult[*app.AppArticleResp]{List: list, Total: res.Total})
}

// GetArticle 获得文章详情
func (h *AppArticleHandler) GetArticle(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("id"), 10, 64)
	title := c.Query("title")

	if id == 0 && title == "" {
		response.WriteBizError(c, errors.ErrParam)
		return
	}

	// 1. Get Detail
	var res *promotion2.ArticleRespVO
	var err error
	if id > 0 {
		res, err = h.articleSvc.GetArticle(c, id)
	} else {
		res, err = h.articleSvc.GetLastArticleByTitle(c, title)
	}

	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// 2. Increment Browse Count
	// If it was queried by title, we need the actual ID for browse count update
	_ = h.articleSvc.AddArticleBrowseCount(c, res.ID)

	response.WriteSuccess(c, articleResponse(res))
}

func articleResponse(v *promotion2.ArticleRespVO) *app.AppArticleResp {
	if v == nil {
		return nil
	}
	return &app.AppArticleResp{ID: v.ID, Title: v.Title, Author: v.Author, CategoryID: v.CategoryID, PicURL: v.PicURL, Introduction: v.Introduction, Content: v.Content, CreateTime: types.ToJsonDateTime(v.CreateTime), BrowseCount: v.BrowseCount, SpuID: v.SpuID}
}
