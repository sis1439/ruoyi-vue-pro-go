package member

import (
	"context"
	"errors"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"math"
	"strings"

	member2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	pkgErrors "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"gorm.io/gorm/clause"
)

type MemberPointRecordService struct {
	q             *query.Query
	memberUserSvc *MemberUserService
}

func NewMemberPointRecordService(q *query.Query, memberUserSvc *MemberUserService) *MemberPointRecordService {
	return &MemberPointRecordService{
		q:             q,
		memberUserSvc: memberUserSvc,
	}
}

// GetPointRecordPage 获得用户积分记录分页
// 对应 Java: MemberPointRecordServiceImpl.getPointRecordPage(MemberPointRecordPageReqVO)
func (s *MemberPointRecordService) GetPointRecordPage(ctx context.Context, r *member2.MemberPointRecordPageReq) (*pagination.PageResult[*member.MemberPointRecord], error) {
	q := s.q.MemberPointRecord.WithContext(ctx)

	// 根据用户昵称查询出用户 ids
	if r.Nickname != "" {
		users, err := s.memberUserSvc.GetUserListByNickname(ctx, r.Nickname)
		if err != nil {
			return nil, err
		}
		// 如果查询用户结果为空直接返回无需继续查询
		if len(users) == 0 {
			return pagination.NewEmptyPageResult[*member.MemberPointRecord](), nil
		}
		var userIds []int64
		for _, u := range users {
			userIds = append(userIds, u.ID)
		}
		q = q.Where(s.q.MemberPointRecord.UserID.In(userIds...))
	}

	// 业务类型过滤
	if r.BizType != nil {
		q = q.Where(s.q.MemberPointRecord.BizType.Eq(*r.BizType))
	}

	// 标题模糊查询
	if r.Title != "" {
		q = q.Where(s.q.MemberPointRecord.Title.Like("%" + r.Title + "%"))
	}

	q = q.Order(s.q.MemberPointRecord.ID.Desc())

	list, count, err := q.FindByPage(r.PageNo, r.PageSize)
	if err != nil {
		return nil, err
	}
	return pagination.NewPageResult(list, count), nil
}

// GetAppPointRecordPage 获得用户App积分记录分页
// 对应 Java: MemberPointRecordServiceImpl.getPointRecordPage(Long userId, AppMemberPointRecordPageReqVO)
func (s *MemberPointRecordService) GetAppPointRecordPage(ctx context.Context, userId int64, r *member2.AppMemberPointRecordPageReq) (*pagination.PageResult[*member.MemberPointRecord], error) {
	q := s.q.MemberPointRecord.WithContext(ctx).Where(s.q.MemberPointRecord.UserID.Eq(userId))
	if len(r.CreateTime) > 0 {
		start, end, err := types.ParseTimeRange(r.CreateTime)
		if err != nil {
			return nil, err
		}
		q = q.Where(s.q.MemberPointRecord.CreateTime.Between(start, end))
	}

	// 增减状态过滤
	if r.AddStatus != nil {
		if *r.AddStatus {
			// 增加积分：点数大于0
			q = q.Where(s.q.MemberPointRecord.Point.Gt(0))
		} else {
			// 扣减积分：点数小于0
			q = q.Where(s.q.MemberPointRecord.Point.Lt(0))
		}
	}

	q = q.Order(s.q.MemberPointRecord.ID.Desc())

	list, count, err := q.FindByPage(r.GetOffset(), r.GetLimit())
	if err != nil {
		return nil, err
	}
	return pagination.NewPageResult(list, count), nil
}

// CreatePointRecord 创建积分记录
// 对应 Java: MemberPointRecordServiceImpl.createPointRecord
// 参数说明：
//   - userId: 用户ID
//   - point: 变动积分（正数增加，负数扣减）
//   - bizType: 业务类型（使用 member.MemberPointBizType 枚举）
//   - bizId: 业务编码
func (s *MemberPointRecordService) CreatePointRecord(ctx context.Context, userId int64, point int, bizType consts.MemberPointBizType, bizId string) error {
	// 积分为0时不处理
	if point == 0 {
		return nil
	}

	if point < math.MinInt32 || point > math.MaxInt32 {
		return errors.New("积分变动超出范围")
	}
	return repo.InTransaction(ctx, s.q, func(ctx context.Context, tx *query.Query) error {
		// 1. 校验用户积分余额
		user, err := tx.MemberUser.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where(tx.MemberUser.ID.Eq(userId)).First()
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("用户不存在")
		}

		userPoint := int(user.Point)
		totalPoint := userPoint + point // 用户变动后的积分
		if totalPoint < 0 || totalPoint > math.MaxInt32 {
			// 积分不足时记录日志并返回（对应 Java 的 log.error + return）
			return pkgErrors.NewBizError(1004014003, "用户积分余额不足")
		}

		// 2. 更新用户积分
		u := tx.MemberUser
		info, err := u.WithContext(ctx).Where(u.ID.Eq(userId), u.Point.Eq(user.Point)).Update(u.Point, u.Point.Add(int32(point)))
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return pkgErrors.NewBizError(1004014003, "用户积分余额不足")
		}

		// 3. 增加积分记录
		// 格式化描述：将 {} 占位符替换为积分值
		description := strings.ReplaceAll(bizType.Description, "{}", fmt.Sprintf("%d", abs(point)))

		record := &member.MemberPointRecord{
			UserID:      userId,
			BizID:       bizId,
			BizType:     bizType.Type,
			Title:       bizType.Name,
			Description: description,
			Point:       point,
			TotalPoint:  totalPoint,
		}
		return tx.MemberPointRecord.WithContext(ctx).Create(record)
	})
}

// abs 返回绝对值
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
