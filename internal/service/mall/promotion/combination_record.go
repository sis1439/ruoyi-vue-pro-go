package promotion

import (
	"context"
	stderrors "errors"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"time"

	promotion2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	prodSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	memberSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
)

type CombinationRecordService interface {
	// App
	GetCombinationRecordSummary(ctx context.Context) (*promotion2.AppCombinationRecordSummaryRespVO, error)
	GetCombinationRecordPage(ctx context.Context, userID int64, req promotion2.AppCombinationRecordPageReq) (*pagination.PageResult[*promotion2.AppCombinationRecordRespVO], error)
	GetCombinationRecordDetail(ctx context.Context, userID int64, id int64) (*promotion2.AppCombinationRecordDetailRespVO, error)
	GetLatestCombinationRecordList(ctx context.Context, count int) ([]*promotion.PromotionCombinationRecord, error)
	GetHeadCombinationRecordList(ctx context.Context, activityID int64, status int, count int) ([]*promotion.PromotionCombinationRecord, error)

	// Internal (for Order)
	ValidateCombinationRecord(ctx context.Context, userID int64, activityID int64, headID int64, skuID int64, count int) (*promotion.PromotionCombinationActivity, *promotion.PromotionCombinationProduct, error)
	CreateCombinationRecord(ctx context.Context, record *promotion.PromotionCombinationRecord) (int64, error)
	// Admin
	GetCombinationRecordPageAdmin(ctx context.Context, req *promotion2.CombinationRecordPageReq) (*pagination.PageResult[*promotion.PromotionCombinationRecord], error)
	GetCombinationRecordSummaryAdmin(ctx context.Context) (*promotion2.CombinationRecordSummaryVO, error)
	ExpireCombinationRecord(ctx context.Context) error
	GetCombinationRecord(ctx context.Context, id int64) (*promotion.PromotionCombinationRecord, error)
	GetCombinationRecordByOrderId(ctx context.Context, userID int64, orderID int64) (*promotion.PromotionCombinationRecord, error)
}

// CombinationTradeOrderService 跨模块依赖接口，用于解除循环依赖
type CombinationTradeOrderService interface {
	CancelPaidOrder(ctx context.Context, uId int64, id int64, cancelType int) error
}

// CombinationSocialClientService 跨模块依赖接口
type CombinationSocialClientService interface {
	SendWxaSubscribeMessage(ctx context.Context, r *system.SocialWxaSubscribeMessageSendReq) error
}

type combinationRecordService struct {
	q           *query.Query
	activitySvc CombinationActivityService
	userSvc     *memberSvc.MemberUserService
	spuSvc      *prodSvc.ProductSpuService
	skuSvc      *prodSvc.ProductSkuService
	tradeSvc    CombinationTradeOrderService
	socialSvc   CombinationSocialClientService
}

func NewCombinationRecordService(
	q *query.Query,
	activitySvc CombinationActivityService,
	userSvc *memberSvc.MemberUserService,
	spuSvc *prodSvc.ProductSpuService,
	skuSvc *prodSvc.ProductSkuService,
	tradeSvc CombinationTradeOrderService,
	socialSvc CombinationSocialClientService,
) CombinationRecordService {
	return &combinationRecordService{
		q:           q,
		activitySvc: activitySvc,
		userSvc:     userSvc,
		spuSvc:      spuSvc,
		skuSvc:      skuSvc,
		tradeSvc:    tradeSvc,
		socialSvc:   socialSvc,
	}
}

func (s *combinationRecordService) GetCombinationRecordSummary(ctx context.Context) (*promotion2.AppCombinationRecordSummaryRespVO, error) {
	q := s.q.PromotionCombinationRecord

	count, err := q.WithContext(ctx).Distinct(q.UserID).Count()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return &promotion2.AppCombinationRecordSummaryRespVO{UserCount: 0, Avatars: []string{}}, nil
	}

	records, err := s.GetLatestCombinationRecordList(ctx, consts.AppCombinationRecordSummaryAvatarCount)
	if err != nil {
		return nil, err
	}

	avatars := make([]string, 0)
	for _, r := range records {
		if r.Avatar != "" {
			avatars = append(avatars, r.Avatar)
		}
	}

	return &promotion2.AppCombinationRecordSummaryRespVO{
		UserCount: count,
		Avatars:   avatars,
	}, nil
}

func (s *combinationRecordService) GetCombinationRecordPage(ctx context.Context, userID int64, req promotion2.AppCombinationRecordPageReq) (*pagination.PageResult[*promotion2.AppCombinationRecordRespVO], error) {
	q := s.q.PromotionCombinationRecord
	do := q.WithContext(ctx).Where(q.UserID.Eq(userID))
	if req.Status != 0 {
		do = do.Where(q.Status.Eq(req.Status))
	}
	list, total, err := do.Order(q.CreateTime.Desc()).FindByPage(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}

	result := make([]*promotion2.AppCombinationRecordRespVO, len(list))
	for i, item := range list {
		result[i] = &promotion2.AppCombinationRecordRespVO{
			ID:               item.ID,
			ActivityID:       item.ActivityID,
			Nickname:         item.Nickname,
			Avatar:           item.Avatar,
			ExpireTime:       types.ToJsonDateTime(item.ExpireTime),
			UserSize:         item.UserSize,
			UserCount:        item.UserCount,
			Status:           item.Status,
			OrderID:          item.OrderID,
			SpuName:          item.SpuName,
			PicUrl:           item.PicUrl,
			Count:            item.Count,
			CombinationPrice: item.CombinationPrice,
		}
	}
	return &pagination.PageResult[*promotion2.AppCombinationRecordRespVO]{List: result, Total: total}, nil
}

func (s *combinationRecordService) GetCombinationRecordDetail(ctx context.Context, userID int64, id int64) (*promotion2.AppCombinationRecordDetailRespVO, error) {
	// 1. 查找这条拼团记录
	record, err := s.GetCombinationRecord(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. 查找该拼团的参团记录
	var headRecord *promotion.PromotionCombinationRecord
	var memberRecords []*promotion.PromotionCombinationRecord
	if record.HeadID == consts.PromotionCombinationRecordHeadIDGroup { // 情况一：团长
		headRecord = record
		memberRecords, err = s.q.PromotionCombinationRecord.WithContext(ctx).Where(s.q.PromotionCombinationRecord.HeadID.Eq(record.ID)).Find()
	} else { // 情况二：团员
		headRecord, err = s.GetCombinationRecord(ctx, record.HeadID)
		if err != nil {
			return nil, err
		}
		memberRecords, err = s.q.PromotionCombinationRecord.WithContext(ctx).Where(s.q.PromotionCombinationRecord.HeadID.Eq(headRecord.ID)).Find()
	}
	if err != nil {
		return nil, err
	}

	// 拼接数据
	allRecords := append([]*promotion.PromotionCombinationRecord{headRecord}, memberRecords...)
	memberVOs := make([]promotion2.AppCombinationRecordRespVO, len(allRecords))
	var userOrderId int64
	for i, r := range allRecords {
		memberVOs[i] = promotion2.AppCombinationRecordRespVO{
			ID:               r.ID,
			ActivityID:       r.ActivityID,
			Nickname:         r.Nickname,
			Avatar:           r.Avatar,
			ExpireTime:       types.ToJsonDateTime(r.ExpireTime),
			UserSize:         r.UserSize,
			UserCount:        r.UserCount,
			Status:           r.Status,
			OrderID:          r.OrderID,
			SpuName:          r.SpuName,
			PicUrl:           r.PicUrl,
			Count:            r.Count,
			CombinationPrice: r.CombinationPrice,
		}
		if r.UserID == userID {
			userOrderId = r.OrderID
		}
	}

	return &promotion2.AppCombinationRecordDetailRespVO{
		HeadRecord:    memberVOs[0],
		MemberRecords: memberVOs,
		OrderID:       userOrderId,
	}, nil
}

func (s *combinationRecordService) GetLatestCombinationRecordList(ctx context.Context, count int) ([]*promotion.PromotionCombinationRecord, error) {
	q := s.q.PromotionCombinationRecord
	return q.WithContext(ctx).Where(q.Status.Eq(consts.PromotionCombinationRecordStatusSuccess)).Order(q.CreateTime.Desc()).Limit(count).Find()
}

func (s *combinationRecordService) GetHeadCombinationRecordList(ctx context.Context, activityID int64, status int, count int) ([]*promotion.PromotionCombinationRecord, error) {
	q := s.q.PromotionCombinationRecord
	tx := q.WithContext(ctx).Where(q.Status.Eq(status), q.HeadID.Eq(consts.PromotionCombinationRecordHeadIDGroup))
	if activityID > 0 {
		tx = tx.Where(q.ActivityID.Eq(activityID))
	}
	return tx.Order(q.CreateTime.Desc()).Limit(count).Find()
}

func (s *combinationRecordService) ValidateCombinationRecord(ctx context.Context, userID int64, activityID int64, headID int64, skuID int64, count int) (*promotion.PromotionCombinationActivity, *promotion.PromotionCombinationProduct, error) {
	activity, err := s.activitySvc.ValidateCombinationActivityCanJoin(ctx, activityID)
	if err != nil {
		return nil, nil, err
	}

	// 1.3 校验是否超出单次限购数量
	if count > activity.SingleLimitCount {
		return nil, nil, errors.NewBizError(1001006012, "单次限购数量超出")
	}

	prod, err := s.q.PromotionCombinationProduct.WithContext(ctx).Where(
		s.q.PromotionCombinationProduct.ActivityID.Eq(activityID),
		s.q.PromotionCombinationProduct.SkuID.Eq(skuID),
	).First()
	if err != nil {
		return nil, nil, errors.NewBizError(1001006004, "拼团活动商品不存在")
	}

	if headID > 0 {
		head, err := s.q.PromotionCombinationRecord.WithContext(ctx).Where(s.q.PromotionCombinationRecord.ID.Eq(headID)).First()
		if err != nil {
			return nil, nil, errors.NewBizError(1001006005, "拼团不存在")
		}
		if head.Status != consts.PromotionCombinationRecordStatusInProgress {
			return nil, nil, errors.NewBizError(1001006006, "拼团已结束")
		}
		if head.UserCount >= head.UserSize {
			return nil, nil, errors.NewBizError(1001006007, "拼团人数已满")
		}
		// 校验拼团是否过期（有父拼团的时候只校验父拼团的过期时间）
		if time.Now().After(head.ExpireTime) {
			return nil, nil, errors.NewBizError(1001006003, "拼团活动已结束")
		}
	} else {
		// 校验当前活动是否结束(自己是父拼团的时候才校验活动是否结束)
		if time.Now().After(activity.EndTime) {
			return nil, nil, errors.NewBizError(1001006003, "拼团活动已结束")
		}
	}

	// 6.1 校验是否有拼团记录 (Already IN_PROGRESS) & Total Limit
	// Status!=2 (Failed) means InProgress(0) or Success(1)
	records, err := s.q.PromotionCombinationRecord.WithContext(ctx).Where(
		s.q.PromotionCombinationRecord.UserID.Eq(userID),
		s.q.PromotionCombinationRecord.ActivityID.Eq(activityID),
		s.q.PromotionCombinationRecord.Status.Neq(consts.PromotionCombinationRecordStatusFailed),
	).Find()
	if err != nil {
		return nil, nil, err
	}

	totalCount := 0
	for _, r := range records {
		if r.Status == consts.PromotionCombinationRecordStatusInProgress {
			return nil, nil, errors.NewBizError(1001006013, "您已有该活动的拼团记录")
		}
		totalCount += r.Count
	}
	if totalCount+count > activity.TotalLimitCount {
		return nil, nil, errors.NewBizError(1001006014, "总限购数量超出")
	}

	return activity, prod, nil
}

func (s *combinationRecordService) CreateCombinationRecord(ctx context.Context, record *promotion.PromotionCombinationRecord) (int64, error) {
	err := s.q.Transaction(func(tx *query.Query) error {
		if err := tx.PromotionCombinationRecord.WithContext(ctx).Create(record); err != nil {
			return err
		}
		// Update Head Status if joining
		if record.HeadID > 0 {
			return s.updateCombinationRecordWhenCreate(ctx, tx, record.HeadID)
		}
		return nil
	})
	return record.ID, err
}

// updateCombinationRecordWhenCreate 更新拼团记录状态
func (s *combinationRecordService) updateCombinationRecordWhenCreate(ctx context.Context, tx *query.Query, headID int64) error {
	// 1. Get Head
	head, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(tx.PromotionCombinationRecord.ID.Eq(headID)).First()
	if err != nil {
		return err
	}
	// 2. Get Members
	members, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(tx.PromotionCombinationRecord.HeadID.Eq(headID)).Find()
	if err != nil {
		return err
	}

	// 3. Get Activity for UserSize
	activity, err := s.activitySvc.GetCombinationActivity(ctx, head.ActivityID)
	if err != nil {
		return err
	}

	// 4. Update
	totalCount := 1 + len(members) // Head + Members
	isFull := totalCount >= activity.UserSize

	updates := make([]*promotion.PromotionCombinationRecord, 0, totalCount)
	updates = append(updates, head)
	updates = append(updates, members...)

	now := time.Now()
	for _, r := range updates {
		r.UserCount = totalCount
		if isFull {
			r.Status = consts.PromotionCombinationRecordStatusSuccess
			r.EndTime = now
		}
		if _, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(tx.PromotionCombinationRecord.ID.Eq(r.ID)).Updates(r); err != nil {
			return err
		}
	}
	if isFull {
		for _, r := range updates {
			s.sendCombinationResultMessage(ctx, r)
		}
	}
	return nil
}

func (s *combinationRecordService) sendCombinationResultMessage(ctx context.Context, record *promotion.PromotionCombinationRecord) {
	// 构建并发送模版消息
	_ = s.socialSvc.SendWxaSubscribeMessage(ctx, &system.SocialWxaSubscribeMessageSendReq{
		UserID:        record.UserID,
		UserType:      consts.UserTypeMember,
		TemplateTitle: "COMBINATION_SUCCESS",
		Page:          "pages/order/detail?id=" + fmt.Sprintf("%d", record.OrderID),
		Messages: map[string]interface{}{
			"thing1": "商品拼团活动",
			"thing2": "恭喜您拼团成功！我们将尽快为您发货。",
		},
	})
}

// GetCombinationRecordPageAdmin 获得拼团记录分页 (Admin)
func (s *combinationRecordService) GetCombinationRecordPageAdmin(ctx context.Context, req *promotion2.CombinationRecordPageReq) (*pagination.PageResult[*promotion.PromotionCombinationRecord], error) {
	q := s.q.PromotionCombinationRecord
	do := q.WithContext(ctx)

	if req.Status != nil {
		do = do.Where(q.Status.Eq(*req.Status))
	}
	if len(req.DateRange) == 2 {
		do = do.Where(q.CreateTime.Between(req.DateRange[0], req.DateRange[1]))
	}

	list, total, err := do.Order(q.CreateTime.Desc()).FindByPage(int((req.PageNo-1)*req.PageSize), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*promotion.PromotionCombinationRecord]{List: list, Total: total}, nil
}

// GetCombinationRecordSummaryAdmin 获得拼团记录的概要信息 (Admin)
// 对齐 Java: CombinationRecordController#getCombinationRecordSummary
func (s *combinationRecordService) GetCombinationRecordSummaryAdmin(ctx context.Context) (*promotion2.CombinationRecordSummaryVO, error) {
	q := s.q.PromotionCombinationRecord

	// 1. 获取拼团用户参与数量 (去重)
	userCount, err := q.WithContext(ctx).Distinct(q.UserID).Count()
	if err != nil {
		return nil, err
	}

	// 2. 获取成团记录数量 (Status=Success, HeadID=Group 表示团长)
	successCount, err := q.WithContext(ctx).Where(q.Status.Eq(consts.PromotionCombinationRecordStatusSuccess), q.HeadID.Eq(consts.PromotionCombinationRecordHeadIDGroup)).Count()
	if err != nil {
		return nil, err
	}

	// 3. 获取虚拟成团记录数量 (VirtualGroup=true, HeadID=Group)
	virtualGroupCount, err := q.WithContext(ctx).Where(q.VirtualGroup.Eq(model.BitBool(true)), q.HeadID.Eq(consts.PromotionCombinationRecordHeadIDGroup)).Count()
	if err != nil {
		return nil, err
	}

	return &promotion2.CombinationRecordSummaryVO{
		UserCount:         userCount,
		SuccessCount:      successCount,
		VirtualGroupCount: virtualGroupCount,
	}, nil
}

func (s *combinationRecordService) GetCombinationRecord(ctx context.Context, id int64) (*promotion.PromotionCombinationRecord, error) {
	return s.q.PromotionCombinationRecord.WithContext(ctx).Where(s.q.PromotionCombinationRecord.ID.Eq(id)).First()
}

func (s *combinationRecordService) GetCombinationRecordByOrderId(ctx context.Context, userID int64, orderID int64) (*promotion.PromotionCombinationRecord, error) {
	return s.q.PromotionCombinationRecord.WithContext(ctx).Where(
		s.q.PromotionCombinationRecord.UserID.Eq(userID),
		s.q.PromotionCombinationRecord.OrderID.Eq(orderID),
	).First()
}
func (s *combinationRecordService) ExpireCombinationRecord(ctx context.Context) error {
	q := s.q.PromotionCombinationRecord
	// 1. 查找所有过期的、进行中的团长记录
	heads, err := q.WithContext(ctx).Where(q.HeadID.Eq(0), q.Status.Eq(consts.PromotionCombinationRecordStatusInProgress), q.ExpireTime.Lt(time.Now())).Find()
	if err != nil {
		return err
	}
	if len(heads) == 0 {
		return nil
	}

	activityIDs := make([]int64, 0, len(heads))
	for _, head := range heads {
		activityIDs = append(activityIDs, head.ActivityID)
	}
	activities, err := s.activitySvc.GetCombinationActivityMap(ctx, activityIDs)
	if err != nil {
		return err
	}
	var failures []error
	for _, head := range heads {
		activity := activities[head.ActivityID]
		if activity != nil && activity.VirtualGroup {
			err = s.handleVirtualGroupRecord(ctx, head)
		} else {
			err = s.handleExpireRecord(ctx, head)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("expire combination %d: %w", head.ID, err))
		}
	}
	return stderrors.Join(failures...)
}

func (s *combinationRecordService) handleExpireRecord(ctx context.Context, head *promotion.PromotionCombinationRecord) error {
	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 获取所有相关的记录
		records, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(
			tx.PromotionCombinationRecord.ID.Eq(head.ID),
		).Or(tx.PromotionCombinationRecord.HeadID.Eq(head.ID)).Find()
		if err != nil {
			return err
		}

		// 2. 更新状态为已失败
		ids := make([]int64, len(records))
		for i, r := range records {
			ids[i] = r.ID
		}
		if _, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(tx.PromotionCombinationRecord.ID.In(ids...)).Update(tx.PromotionCombinationRecord.Status, consts.PromotionCombinationRecordStatusFailed); err != nil {
			return err
		}

		// 3. 取消订单并退款 (对齐 Java: tradeOrderApi.cancelPaidOrder)
		for _, r := range records {
			if err := s.tradeSvc.CancelPaidOrder(ctx, r.UserID, r.OrderID, consts.TradeOrderCancelTypeCombinationClose); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *combinationRecordService) handleVirtualGroupRecord(ctx context.Context, head *promotion.PromotionCombinationRecord) error {
	return s.q.Transaction(func(tx *query.Query) error {
		// 1. 获取所有相关的记录
		records, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(
			tx.PromotionCombinationRecord.ID.Eq(head.ID),
		).Or(tx.PromotionCombinationRecord.HeadID.Eq(head.ID)).Find()
		if err != nil {
			return err
		}

		// 2. 更新状态为成功
		ids := make([]int64, len(records))
		for i, r := range records {
			ids[i] = r.ID
		}
		now := time.Now()
		if _, err := tx.PromotionCombinationRecord.WithContext(ctx).Where(tx.PromotionCombinationRecord.ID.In(ids...)).Updates(map[string]interface{}{
			"status":        consts.PromotionCombinationRecordStatusSuccess,
			"end_time":      &now,
			"user_count":    head.UserSize,
			"virtual_group": model.BitBool(true),
		}); err != nil {
			return err
		}

		// 3. 补全虚拟记录 (对齐 Java: CombinationActivityConvert.INSTANCE.convertVirtualRecordList)
		lackCount := head.UserSize - len(records)
		if lackCount > 0 {
			virtualRecords := make([]*promotion.PromotionCombinationRecord, lackCount)
			for i := 0; i < lackCount; i++ {
				virtualRecords[i] = &promotion.PromotionCombinationRecord{
					ActivityID:       head.ActivityID,
					CombinationPrice: head.CombinationPrice,
					SpuID:            head.SpuID,
					SpuName:          head.SpuName,
					PicUrl:           head.PicUrl,
					SkuID:            head.SkuID,
					Count:            0, // 虚拟成员不占购销量
					UserID:           0, // 虚拟成员
					Nickname:         "虚拟用户",
					Avatar:           "",
					HeadID:           head.ID,
					Status:           consts.PromotionCombinationRecordStatusSuccess,
					OrderID:          0,
					UserSize:         head.UserSize,
					UserCount:        head.UserSize,
					VirtualGroup:     model.BitBool(true),
					StartTime:        head.StartTime,
					EndTime:          now,
					ExpireTime:       head.ExpireTime,
				}
			}
			if err := tx.PromotionCombinationRecord.WithContext(ctx).Create(virtualRecords...); err != nil {
				return err
			}
		}
		return nil
	})
}
