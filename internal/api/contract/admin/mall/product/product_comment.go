package product

import (
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/model"
)

// ProductCommentPageReq 商品评价分页 Request
type ProductCommentPageReq struct {
	PageNo       int      `form:"pageNo" binding:"required,min=1"`
	PageSize     int      `form:"pageSize" binding:"required,min=1,max=100"`
	UserNickname string   `form:"userNickname"`
	OrderID      int64    `form:"orderId"`
	SpuID        int64    `form:"spuId"`
	SpuName      string   `form:"spuName"`
	Scores       int      `form:"scores"`
	ReplyStatus  *bool    `form:"replyStatus"`
	CreateTime   []string `form:"createTime[]"` // time range
}

// ProductCommentUpdateVisibleReq 更新评论可见性 Request
type ProductCommentUpdateVisibleReq struct {
	ID      int64 `json:"id" binding:"required"`
	Visible *bool `json:"visible" binding:"required"`
}

// ProductCommentReplyReq 商家回复 Request
type ProductCommentReplyReq struct {
	ID           model.FlexInt64 `json:"id" binding:"required"`
	ReplyContent string          `json:"replyContent" binding:"required"` // 商家必须填写回复内容
}

// AppProductCommentCreateReq 添加自评 Request (App端)
type AppProductCommentCreateReq struct {
	OrderItemID       int64    `json:"orderItemId" binding:"required"`
	Anonymous         bool     `json:"anonymous"`
	Content           string   `json:"content" binding:"required"`
	PicURLs           []string `json:"picUrls"`
	Scores            int      `json:"scores" binding:"required,min=1,max=5"`
	DescriptionScores int      `json:"descriptionScores" binding:"required,min=1,max=5"`
	BenefitScores     int      `json:"benefitScores" binding:"required,min=1,max=5"`
}

// ProductCommentCreateReq 商品评价创建 Request (后台)
type ProductCommentCreateReq struct {
	UserID            model.FlexInt64 `json:"userId"`
	OrderItemID       model.FlexInt64 `json:"orderItemId"`
	UserNickname      string          `json:"userNickname" binding:"required"`
	UserAvatar        string          `json:"userAvatar" binding:"required"`
	SpuID             model.FlexInt64 `json:"spuId"`
	SkuID             model.FlexInt64 `json:"skuId" binding:"required"`
	DescriptionScores int             `json:"descriptionScores" binding:"required,min=1,max=5"`
	BenefitScores     int             `json:"benefitScores" binding:"required,min=1,max=5"`
	Content           string          `json:"content" binding:"required"`
	PicURLs           []string        `json:"picUrls"`
}

// AppProductCommentPageReq 商品评价分页 Request (App端)
type AppProductCommentPageReq struct {
	PageNo   int   `form:"pageNo" binding:"required,min=1"`
	PageSize int   `form:"pageSize" binding:"required,min=1,max=100"`
	SpuID    int64 `form:"spuId" binding:"required"`
	Type     int   `form:"type"` // 0: 全部, 1: 好评, 2: 中评, 3: 差评, 4: 有图
}

// ProductCommentResp 商品评价响应 (Admin)
type ProductCommentResp struct {
	ID                int64                    `json:"id"`
	UserID            int64                    `json:"userId"`
	UserNickname      string                   `json:"userNickname"`
	UserAvatar        string                   `json:"userAvatar"`
	Anonymous         bool                     `json:"anonymous"`
	OrderID           int64                    `json:"orderId"`
	OrderItemID       int64                    `json:"orderItemId"`
	SpuID             int64                    `json:"spuId"`
	SpuName           string                   `json:"spuName"`
	SkuID             int64                    `json:"skuId"`
	SkuPicURL         string                   `json:"skuPicUrl"`
	SkuProperties     []ProductSkuPropertyResp `json:"skuProperties"`
	Visible           bool                     `json:"visible"`
	Scores            int                      `json:"scores"`
	DescriptionScores int                      `json:"descriptionScores"`
	BenefitScores     int                      `json:"benefitScores"`
	Content           string                   `json:"content"`
	PicURLs           []string                 `json:"picUrls"`
	ReplyStatus       bool                     `json:"replyStatus"`
	ReplyUserID       int64                    `json:"replyUserId"`
	ReplyContent      string                   `json:"replyContent"`
	ReplyTime         *time.Time               `json:"replyTime"`
	CreateTime        time.Time                `json:"createTime"`
}

// AppProductCommentResp 商品评价响应 (App)
type AppProductCommentResp struct {
	UserID            int64                    `json:"userId"`
	Anonymous         bool                     `json:"anonymous"`
	OrderID           int64                    `json:"orderId"`
	OrderItemID       int64                    `json:"orderItemId"`
	SkuID             int64                    `json:"skuId"`
	SpuID             int64                    `json:"spuId"`
	DescriptionScores int                      `json:"descriptionScores"`
	BenefitScores     int                      `json:"benefitScores"`
	ReplyStatus       bool                     `json:"replyStatus"`
	ReplyUserID       int64                    `json:"replyUserId"`
	AdditionalContent *string                  `json:"additionalContent"`
	AdditionalPicUrls []string                 `json:"additionalPicUrls"`
	AdditionalTime    *types.JsonDateTime      `json:"additionalTime"`
	SpuName           string                   `json:"spuName"`
	ReplyTime         *types.JsonDateTime      `json:"replyTime"`
	ID                int64                    `json:"id"`
	UserNickname      string                   `json:"userNickname"`
	UserAvatar        string                   `json:"userAvatar"`
	Scores            int                      `json:"scores"`
	Content           string                   `json:"content"`
	PicURLs           []string                 `json:"picUrls"`
	ReplyContent      string                   `json:"replyContent"`
	SkuProperties     []ProductSkuPropertyResp `json:"skuProperties"`
	CreateTime        types.JsonDateTime       `json:"createTime"`
}
