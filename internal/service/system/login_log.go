package system

import (
	"context"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"go.uber.org/zap"
)

type LoginLogService struct {
	q *query.Query
}

func NewLoginLogService(q *query.Query) *LoginLogService {
	return &LoginLogService{q: q}
}

// GetLoginLogPage 获取登录日志分页
func (s *LoginLogService) GetLoginLogPage(ctx context.Context, r *system.LoginLogPageReq) (*pagination.PageResult[*model.SystemLoginLog], error) {
	q := s.q.SystemLoginLog.WithContext(ctx)

	// 过滤条件
	if r.UserIP != "" {
		q = q.Where(s.q.SystemLoginLog.UserIP.Like("%" + r.UserIP + "%"))
	}
	if r.Username != "" {
		q = q.Where(s.q.SystemLoginLog.Username.Like("%" + r.Username + "%"))
	}
	if r.Status != nil {
		// status = true means result = 0 (success), status = false means result != 0
		if *r.Status {
			q = q.Where(s.q.SystemLoginLog.Result.Eq(0))
		} else {
			q = q.Where(s.q.SystemLoginLog.Result.Neq(0))
		}
	}
	if len(r.CreateTime) == 2 {
		q = q.Where(s.q.SystemLoginLog.CreateTime.Between(r.CreateTime[0], r.CreateTime[1]))
	}

	// 分页
	pageNo := r.PageNo
	pageSize := r.PageSize
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNo - 1) * pageSize

	total, err := q.Count()
	if err != nil {
		return nil, err
	}

	list, err := q.Order(s.q.SystemLoginLog.ID.Desc()).Offset(offset).Limit(pageSize).Find()
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*model.SystemLoginLog]{
		List:  list,
		Total: total,
	}, nil
}

// CreateLoginLog 记录登录日志
func (s *LoginLogService) CreateLoginLog(ctx context.Context, userId int64, userType int, username, ip, userAgent string, logType int, result int) {
	s.writeLog(ctx, &model.SystemLoginLog{
		LogType:   logType,
		TraceID:   "", // TODO: Extract traceId from context
		UserID:    userId,
		UserType:  userType,
		Username:  username,
		Result:    result,
		UserIP:    ip,
		UserAgent: userAgent,
	})
}

// CreateLogoutLog 记录登出日志
func (s *LoginLogService) CreateLogoutLog(ctx context.Context, userId int64, userType int, username, ip, userAgent string) {
	s.writeLog(ctx, &model.SystemLoginLog{
		LogType:   consts.LogoutLogTypeSelf,
		UserID:    userId,
		UserType:  userType,
		Username:  username,
		UserIP:    ip,
		UserAgent: userAgent,
		Result:    consts.LoginResultSuccess,
	})
}

func (s *LoginLogService) writeLog(ctx context.Context, event *model.SystemLoginLog) {
	tenant, ok := pkgcontext.TenantID(ctx)
	if !ok {
		zap.L().Error("Refusing authentication audit without trusted tenant", zap.Int64("userId", event.UserID))
		return
	}
	// A short synchronous write preserves audit ordering and never retains gin.Context.
	auditCtx, cancel := context.WithTimeout(pkgcontext.WithTenant(context.Background(), tenant), 2*time.Second)
	defer cancel()
	if err := s.q.SystemLoginLog.WithContext(auditCtx).Create(event); err != nil {
		zap.L().Error("Failed to persist authentication audit", zap.Int64("tenantId", tenant), zap.Int64("userId", event.UserID), zap.Error(err))
	}
}
