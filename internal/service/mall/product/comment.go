package product

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	product2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/samber/lo"
)

type ProductCommentService struct {
	q      *query.Query
	spuSvc *ProductSpuService
	skuSvc *ProductSkuService
}

func NewProductCommentService(q *query.Query, spuSvc *ProductSpuService, skuSvc *ProductSkuService) *ProductCommentService {
	return &ProductCommentService{
		q:      q,
		spuSvc: spuSvc,
		skuSvc: skuSvc,
	}
}

// GetCommentPage 获得商品评价分页 (Admin)
func (s *ProductCommentService) GetCommentPage(ctx context.Context, req *product2.ProductCommentPageReq) (*pagination.PageResult[*product2.ProductCommentResp], error) {
	u := s.q.ProductComment
	q := u.WithContext(ctx)

	if req.UserNickname != "" {
		q = q.Where(u.UserNickname.Like("%" + req.UserNickname + "%"))
	}
	if req.OrderID > 0 {
		q = q.Where(u.OrderID.Eq(req.OrderID))
	}
	if req.SpuID > 0 {
		q = q.Where(u.SpuID.Eq(req.SpuID))
	}
	if req.SpuName != "" {
		q = q.Where(u.SpuName.Like("%" + req.SpuName + "%"))
	}
	if req.Scores > 0 {
		q = q.Where(u.Scores.Eq(req.Scores))
	}
	if req.ReplyStatus != nil {
		q = q.Where(u.ReplyStatus.Eq(model.BitBool(*req.ReplyStatus)))
	}
	// CreateTime range handled if needed

	list, total, err := q.Order(u.ID.Desc()).FindByPage((req.PageNo-1)*req.PageSize, req.PageSize)
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*product2.ProductCommentResp]{
		List:  s.convertList(list),
		Total: total,
	}, nil
}

// UpdateCommentVisible 更新评论可见性
func (s *ProductCommentService) UpdateCommentVisible(ctx context.Context, req *product2.ProductCommentUpdateVisibleReq) error {
	_, err := s.validateCommentExists(ctx, req.ID)
	if err != nil {
		return err
	}
	_, err = s.q.ProductComment.WithContext(ctx).Where(s.q.ProductComment.ID.Eq(req.ID)).Update(s.q.ProductComment.Visible, model.BitBool(*req.Visible))
	return err
}

// ReplyComment 商家回复
func (s *ProductCommentService) ReplyComment(ctx context.Context, req *product2.ProductCommentReplyReq, loginUserID int64) error {
	_, err := s.validateCommentExists(ctx, int64(req.ID))
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = s.q.ProductComment.WithContext(ctx).Where(s.q.ProductComment.ID.Eq(int64(req.ID))).Updates(&product.ProductComment{
		ReplyStatus:  true,
		ReplyUserID:  loginUserID,
		ReplyContent: req.ReplyContent,
		ReplyTime:    &now,
	})
	return err
}

// CreateComment 创建评论 (Admin)
func (s *ProductCommentService) CreateComment(ctx context.Context, req *product2.ProductCommentCreateReq) error {
	// 校验 SKU
	sku, err := s.skuSvc.GetSku(ctx, int64(req.SkuID))
	if err != nil {
		return err
	}
	// 校验 SPU
	spu, err := s.spuSvc.GetSpu(ctx, sku.SpuID)
	if err != nil {
		return err
	}

	comment := &product.ProductComment{
		UserID:            int64(req.UserID),
		UserNickname:      req.UserNickname,
		UserAvatar:        req.UserAvatar,
		Anonymous:         false, // Admin created usually not anonymous? Or default false
		OrderItemID:       int64(req.OrderItemID),
		SpuID:             spu.ID,
		SpuName:           spu.Name,
		SkuID:             sku.ID,
		SkuPicURL:         sku.PicURL,
		SkuProperties:     sku.Properties,
		Visible:           true,
		DescriptionScores: req.DescriptionScores,
		BenefitScores:     req.BenefitScores,
		Content:           req.Content,
		PicURLs:           req.PicURLs,
		ReplyStatus:       false,
	}
	// Calc avg scores if needed. Java uses Description + Benefit + Service / 3 usually?
	comment.Scores = (comment.DescriptionScores + comment.BenefitScores) / 2

	return s.q.ProductComment.WithContext(ctx).Create(comment)
}

// GetAppCommentPage 获得商品评价分页 (App)
func (s *ProductCommentService) GetAppCommentPage(ctx context.Context, r *product2.AppProductCommentPageReq) (*pagination.PageResult[*product2.AppProductCommentResp], error) {
	u := s.q.ProductComment
	q := u.WithContext(ctx).Where(u.SpuID.Eq(r.SpuID), u.Visible.Eq(model.BitBool(true)))

	// Type filter: 0=全部, 1=好评(4-5), 2=中评(3), 3=差评(1-2), 4=有图
	switch r.Type {
	case 1:
		q = q.Where(u.Scores.Gte(consts.ProductCommentScoreGood))
	case 2:
		q = q.Where(u.Scores.Eq(consts.ProductCommentScoreNormal))
	case 3:
		q = q.Where(u.Scores.Lte(consts.ProductCommentScoreBad))
	case 4:
		q = q.Where(u.PicURLs.IsNotNull())
	}

	list, total, err := q.Order(u.ID.Desc()).FindByPage((r.PageNo-1)*r.PageSize, r.PageSize)
	if err != nil {
		return nil, err
	}

	result := lo.Map(list, func(item *product.ProductComment, _ int) *product2.AppProductCommentResp {
		nickname := item.UserNickname
		if item.Anonymous {
			nickname = "匿名用户"
		}
		return &product2.AppProductCommentResp{
			UserID: item.UserID, Anonymous: bool(item.Anonymous), OrderID: item.OrderID, OrderItemID: item.OrderItemID, SkuID: item.SkuID, SpuID: item.SpuID, DescriptionScores: item.DescriptionScores, BenefitScores: item.BenefitScores, ReplyStatus: bool(item.ReplyStatus), ReplyUserID: item.ReplyUserID,
			SpuName: item.SpuName, ReplyTime: types.ToJsonDateTimePtr(item.ReplyTime),
			ID:            item.ID,
			UserNickname:  nickname,
			UserAvatar:    item.UserAvatar,
			Scores:        item.Scores,
			Content:       item.Content,
			PicURLs:       item.PicURLs,
			ReplyContent:  item.ReplyContent,
			SkuProperties: s.convertSkuProperties(item.SkuProperties),
			CreateTime:    types.ToJsonDateTime(item.CreateTime),
		}
	})

	return &pagination.PageResult[*product2.AppProductCommentResp]{
		List:  result,
		Total: total,
	}, nil
}

// Helpers

// CreateAppComment 创建商品评价 (App)
func (s *ProductCommentService) CreateAppComment(ctx context.Context, userId int64, req *product2.AppProductCommentCreateReq) (*product.ProductComment, error) {
	// 1. Verify OrderItem
	item, err := s.q.TradeOrderItem.WithContext(ctx).Where(s.q.TradeOrderItem.ID.Eq(req.OrderItemID), s.q.TradeOrderItem.UserID.Eq(userId)).First()
	if err != nil {
		return nil, err // Order item not found or not owned by user
	}
	if item.CommentStatus {
		return nil, errors.NewBizError(1006000007, "该商品已评价") // DUPLICATE_COMMENT
	}

	// 2. Prepare Comment
	// Convert properties
	var skuProps []product.ProductSkuProperty
	// We can manually map since structs are simple
	if len(item.Properties) > 0 {
		skuProps = make([]product.ProductSkuProperty, len(item.Properties))
		for i, p := range item.Properties {
			skuProps[i] = product.ProductSkuProperty{
				PropertyID:   p.PropertyID,
				PropertyName: p.PropertyName,
				ValueID:      p.ValueID,
				ValueName:    p.ValueName,
			}
		}
	}

	comment := &product.ProductComment{
		UserID:            userId,
		Anonymous:         model.NewBitBool(req.Anonymous),
		OrderItemID:       req.OrderItemID,
		OrderID:           item.OrderID,
		SpuID:             item.SpuID,
		SpuName:           item.SpuName,
		SkuID:             item.SkuID,
		SkuPicURL:         item.PicURL,
		SkuProperties:     skuProps,
		Visible:           true,
		Scores:            (req.DescriptionScores + req.BenefitScores) / 2,
		DescriptionScores: req.DescriptionScores,
		BenefitScores:     req.BenefitScores,
		Content:           req.Content,
		PicURLs:           req.PicURLs,
		ReplyStatus:       false,
	}

	// Fetch User info for Snapshot (Optional, usually good to have)
	user, _ := s.q.MemberUser.WithContext(ctx).Where(s.q.MemberUser.ID.Eq(userId)).First()
	if user != nil {
		comment.UserNickname = user.Nickname
		comment.UserAvatar = user.Avatar
	}

	// 3. Transaction
	err = s.q.Transaction(func(tx *query.Query) error {
		// Save Comment
		if err := tx.ProductComment.WithContext(ctx).Create(comment); err != nil {
			return err
		}
		// Update OrderItem Status
		if _, err := tx.TradeOrderItem.WithContext(ctx).Where(tx.TradeOrderItem.ID.Eq(item.ID)).Update(tx.TradeOrderItem.CommentStatus, model.BitBool(true)); err != nil {
			return err
		}

		// Check Order Comment Status
		// If all items are commented, mark order as commented?
		// Logic: count items in this order where comment_status is false
		count, err := tx.TradeOrderItem.WithContext(ctx).Where(tx.TradeOrderItem.OrderID.Eq(item.OrderID), tx.TradeOrderItem.CommentStatus.Eq(model.NewBitBool(false))).Count()
		if err == nil && count == 0 {
			// All commented
			tx.TradeOrder.WithContext(ctx).Where(tx.TradeOrder.ID.Eq(item.OrderID)).Update(tx.TradeOrder.CommentStatus, model.BitBool(true))
		}

		return nil
	})

	return comment, err
}

// Helpers

func (s *ProductCommentService) validateCommentExists(ctx context.Context, id int64) (*product.ProductComment, error) {
	c, err := s.q.ProductComment.WithContext(ctx).Where(s.q.ProductComment.ID.Eq(id)).First()
	if err != nil {
		return nil, errors.NewBizError(1006000006, "评论不存在") // COMMENT_NOT_EXISTS (Mock code)
	}
	return c, nil
}

func (s *ProductCommentService) convertList(list []*product.ProductComment) []*product2.ProductCommentResp {
	return lo.Map(list, func(item *product.ProductComment, _ int) *product2.ProductCommentResp {
		return &product2.ProductCommentResp{
			ID:                item.ID,
			UserID:            item.UserID,
			UserNickname:      item.UserNickname,
			UserAvatar:        item.UserAvatar,
			Anonymous:         bool(item.Anonymous),
			OrderID:           item.OrderID,
			OrderItemID:       item.OrderItemID,
			SpuID:             item.SpuID,
			SpuName:           item.SpuName,
			SkuID:             item.SkuID,
			SkuPicURL:         item.SkuPicURL,
			SkuProperties:     s.convertSkuProperties(item.SkuProperties),
			Visible:           bool(item.Visible),
			Scores:            item.Scores,
			DescriptionScores: item.DescriptionScores,
			BenefitScores:     item.BenefitScores,
			Content:           item.Content,
			PicURLs:           item.PicURLs,
			ReplyStatus:       bool(item.ReplyStatus),
			ReplyUserID:       item.ReplyUserID,
			ReplyContent:      item.ReplyContent,
			ReplyTime:         item.ReplyTime,
			CreateTime:        item.CreateTime,
		}
	})
}

func (s *ProductCommentService) convertSkuProperties(props []product.ProductSkuProperty) []product2.ProductSkuPropertyResp {
	return lo.Map(props, func(item product.ProductSkuProperty, _ int) product2.ProductSkuPropertyResp {
		return product2.ProductSkuPropertyResp{
			PropertyID:   item.PropertyID,
			PropertyName: item.PropertyName,
			ValueID:      item.ValueID,
			ValueName:    item.ValueName,
		}
	})
}
