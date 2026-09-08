package member

import (
	"context"

	member2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"

	"github.com/samber/lo"
)

type MemberSignInRecordService struct {
	q               *query.Query
	signInConfigSvc *MemberSignInConfigService
	memberUserSvc   *MemberUserService
	pointRecordSvc  *MemberPointRecordService
	memberLevelSvc  *MemberLevelService
}

func NewMemberSignInRecordService(q *query.Query,
	signInConfigSvc *MemberSignInConfigService,
	memberUserSvc *MemberUserService,
	pointRecordSvc *MemberPointRecordService,
	memberLevelSvc *MemberLevelService) *MemberSignInRecordService {
	return &MemberSignInRecordService{
		q:               q,
		signInConfigSvc: signInConfigSvc,
		memberUserSvc:   memberUserSvc,
		pointRecordSvc:  pointRecordSvc,
		memberLevelSvc:  memberLevelSvc,
	}
}

// GetSignInRecordSummary 获得签到记录统计
// GetSignInRecordSummary 获得签到记录统计
func (s *MemberSignInRecordService) GetSignInRecordSummary(ctx context.Context, userId int64) (*member2.AppMemberSignInRecordSummaryResp, error) {
	summary := &member2.AppMemberSignInRecordSummaryResp{
		TotalDay:      0,
		ContinuousDay: 0,
		TodaySignIn:   false,
	}

	// 1. Count Total
	count, err := s.q.MemberSignInRecord.WithContext(ctx).Where(s.q.MemberSignInRecord.UserID.Eq(userId)).Count()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return summary, nil
	}
	summary.TotalDay = int(count)

	// 2. Last Record
	lastRecord, err := s.q.MemberSignInRecord.WithContext(ctx).
		Where(s.q.MemberSignInRecord.UserID.Eq(userId)).
		Order(s.q.MemberSignInRecord.ID.Desc()).First()
	if err != nil {
		return summary, nil
	}

	// 3. Check Today
	isToday := utils.IsToday(lastRecord.CreateTime)
	summary.TodaySignIn = isToday

	// 4. Check logic for continuous days
	if !isToday && !utils.IsYesterday(lastRecord.CreateTime) {
		// Not today and not yesterday -> Streak broken
		return summary, nil
	}

	summary.ContinuousDay = lastRecord.Day
	return summary, nil
}

// GetSignInRecordPage 获得签到记录分页
func (s *MemberSignInRecordService) GetSignInRecordPage(ctx context.Context, r *member2.MemberSignInRecordPageReq) (*pagination.PageResult[*member.MemberSignInRecord], error) {
	q := s.q.MemberSignInRecord.WithContext(ctx)

	if r.Nickname != "" {
		users, err := s.memberUserSvc.GetUserListByNickname(ctx, r.Nickname)
		if err != nil {
			return nil, err
		}
		if len(users) == 0 {
			return pagination.NewEmptyPageResult[*member.MemberSignInRecord](), nil
		}
		userIds := lo.Map(users, func(u *member.MemberUser, _ int) int64 { return u.ID })
		q = q.Where(s.q.MemberSignInRecord.UserID.In(userIds...))
	}

	if r.UserID > 0 {
		q = q.Where(s.q.MemberSignInRecord.UserID.Eq(r.UserID))
	}

	if r.Day != nil {
		q = q.Where(s.q.MemberSignInRecord.Day.Eq(*r.Day))
	}

	list, count, err := q.Order(s.q.MemberSignInRecord.ID.Desc()).FindByPage(r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}
	return pagination.NewPageResult(list, count), nil
}

// CreateSignInRecord 创建签到记录 (Transactional)
func (s *MemberSignInRecordService) CreateSignInRecord(ctx context.Context, userId int64) (*member.MemberSignInRecord, error) {
	if userId <= 0 {
		return nil, errors.NewBizError(401, "未登录")
	}
	// 1. Check if already signed today
	lastRecord, _ := s.q.MemberSignInRecord.WithContext(ctx).
		Where(s.q.MemberSignInRecord.UserID.Eq(userId)).
		Order(s.q.MemberSignInRecord.ID.Desc()).First()

	if lastRecord != nil && utils.IsToday(lastRecord.CreateTime) {
		return nil, errors.NewBizError(1004014005, "Already signed in today")
	}

	// 2. Get Configs
	status := consts.CommonStatusEnable
	configs, err := s.signInConfigSvc.GetSignInConfigList(ctx, &status)
	if err != nil {
		return nil, err
	}

	// 3. Calculate Day and Reward
	// Default day = 1
	day := 1
	// If last record was yesterday, continue streak
	if lastRecord != nil && utils.IsYesterday(lastRecord.CreateTime) {
		day = lastRecord.Day + 1
	}

	// Check max config day
	maxDay := 0
	if len(configs) > 0 {
		maxDay = configs[len(configs)-1].Day // Assumes sorted asc
	}
	if day > maxDay {
		day = 1 // Reset streak if exceeds config
	}

	rewardPoint := 0
	rewardExp := 0

	// Find config for this day
	for _, cfg := range configs {
		if cfg.Day == day {
			rewardPoint = cfg.Point
			rewardExp = cfg.Experience
			break
		}
	}

	record := &member.MemberSignInRecord{
		UserID:     userId,
		Day:        day,
		Point:      rewardPoint,
		Experience: rewardExp,
	}

	err = s.q.Transaction(func(tx *query.Query) error {
		// 1. 创建签到记录
		if err := tx.MemberSignInRecord.WithContext(ctx).Create(record); err != nil {
			return err
		}

		// 2. 增加积分
		if rewardPoint > 0 {
			// 使用签到业务类型枚举
			if err := s.pointRecordSvc.CreatePointRecord(ctx, userId, rewardPoint, consts.MemberPointBizTypeSign, utils.ToString(record.ID)); err != nil {
				return err
			}
		}

		// 3. 增加经验值
		if rewardExp > 0 {
			// Using "1" as BizType for SIGN_IN as defined in MemberExperienceBizTypeEnum.SIGN_IN
			if err := s.memberLevelSvc.AddExperience(ctx, userId, rewardExp, 1, utils.ToString(record.ID)); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return record, nil
}
