package infra

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/infra"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"go.uber.org/zap"
)

// JobStatus 任务状态
const (
	JobStatusInit   = 0 // 初始化
	JobStatusNormal = 1 // 开启
	JobStatusStop   = 2 // 暂停
)

type JobService struct {
	// ponytail: serialize mutations in this single process; multi-instance scheduling needs an elected owner.
	mu        sync.Mutex
	q         *query.Query
	scheduler *Scheduler
}

func NewJobService(q *query.Query, scheduler *Scheduler) *JobService {
	return &JobService{q: q, scheduler: scheduler}
}

// CreateJob 创建定时任务
func (s *JobService) CreateJob(ctx context.Context, r *infra.JobSaveReq) (int64, error) {
	if r == nil {
		return 0, errors.New("任务参数不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validateJobHandlerExists(r.HandlerName); err != nil {
		return 0, err
	}
	if err := s.scheduler.ValidateCronExpression(r.CronExpression); err != nil {
		return 0, err
	}
	if err := s.validateHandlerNameUnique(ctx, r.HandlerName, nil); err != nil {
		return 0, err
	}
	monitorTimeout := r.MonitorTimeout
	if monitorTimeout == nil {
		value := 0
		monitorTimeout = &value
	}
	job := &model.InfraJob{Name: r.Name, Status: JobStatusNormal, HandlerName: r.HandlerName,
		HandlerParam: r.HandlerParam, CronExpression: r.CronExpression, RetryCount: r.RetryCount,
		RetryInterval: r.RetryInterval, MonitorTimeout: monitorTimeout}
	// Commit the desired state before another connection or the scheduler reads it.
	if err := s.q.InfraJob.WithContext(ctx).Create(job); err != nil {
		return 0, err
	}
	return job.ID, s.syncJob(ctx, job.ID)
}

// UpdateJob commits the new configuration before replacing the scheduled task.
func (s *JobService) UpdateJob(ctx context.Context, r *infra.JobSaveReq) error {
	if r == nil || r.ID == nil {
		return errors.New("任务 ID 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validateJobHandlerExists(r.HandlerName); err != nil {
		return err
	}
	if err := s.scheduler.ValidateCronExpression(r.CronExpression); err != nil {
		return err
	}
	if err := s.validateHandlerNameUnique(ctx, r.HandlerName, r.ID); err != nil {
		return err
	}
	monitorTimeout := 0
	if r.MonitorTimeout != nil {
		monitorTimeout = *r.MonitorTimeout
	}
	c := s.q.InfraJob
	info, err := c.WithContext(ctx).Where(c.ID.Eq(*r.ID), c.Status.Eq(JobStatusNormal)).Updates(map[string]interface{}{
		"name": r.Name, "handler_name": r.HandlerName, "handler_param": r.HandlerParam,
		"cron_expression": r.CronExpression, "retry_count": r.RetryCount,
		"retry_interval": r.RetryInterval, "monitor_timeout": monitorTimeout,
	})
	if err != nil {
		return err
	}
	if info.RowsAffected != 1 {
		return errors.New("任务不存在、无权访问或未开启")
	}
	return s.syncJob(ctx, *r.ID)
}

// syncJob reports a durable, recoverable desired-state mismatch instead of false success.
func (s *JobService) syncJob(ctx context.Context, id int64) error {
	if s.scheduler == nil {
		return errors.New("调度器未初始化")
	}
	if err := s.scheduler.SyncJob(ctx, id); err != nil {
		s.scheduler.log.Error("Persisted job needs scheduling retry", zap.Int64("jobId", id), zap.Error(err))
		return fmt.Errorf("任务 %d 已保存，调度同步失败，请重试同步: %w", id, err)
	}
	return nil
}

// validateJobHandlerExists 校验 Handler 是否已注册
func (s *JobService) validateJobHandlerExists(handlerName string) error {
	if s.scheduler == nil {
		return errors.New("调度器未初始化")
	}
	if !s.scheduler.HasHandler(handlerName) {
		return fmt.Errorf("定时任务处理器 '%s' 不存在，请检查并确保已在 Scheduler 注册。当前可用 Handler: %v",
			handlerName, s.scheduler.GetRegisteredHandlers())
	}
	return nil
}

// validateHandlerNameUnique 校验 Handler 唯一性
func (s *JobService) validateHandlerNameUnique(ctx context.Context, handlerName string, excludeID *int64) error {
	query := s.q.InfraJob.WithContext(ctx).Where(s.q.InfraJob.HandlerName.Eq(handlerName))
	if excludeID != nil {
		query = query.Where(s.q.InfraJob.ID.Neq(*excludeID))
	}
	count, err := query.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该处理器名称已被使用")
	}
	return nil
}

// DeleteJob 删除定时任务
func (s *JobService) DeleteJob(ctx context.Context, id int64) error {
	return s.DeleteJobList(ctx, []int64{id})
}

// DeleteJobList validates the whole batch before changing persistent or scheduled state.
func (s *JobService) DeleteJobList(ctx context.Context, ids []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			return errors.New("无效任务 ID")
		}
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	if len(unique) == 0 {
		return nil
	}
	err := s.q.Transaction(func(tx *query.Query) error {
		c := tx.InfraJob
		rows, err := c.WithContext(ctx).Where(c.ID.In(unique...)).Find()
		if err != nil {
			return err
		}
		if len(rows) != len(unique) {
			return errors.New("任务不存在或无权访问")
		}
		info, err := c.WithContext(ctx).Where(c.ID.In(unique...)).Delete()
		if err != nil {
			return err
		}
		if info.RowsAffected != int64(len(unique)) {
			return errors.New("任务状态已变化")
		}
		return nil
	})
	if err != nil {
		return err
	}
	var result error
	for _, id := range unique {
		result = errors.Join(result, s.syncJob(ctx, id))
	}
	return result
}

// GetJob 获取定时任务
func (s *JobService) GetJob(ctx context.Context, id int64) (*model.InfraJob, error) {
	return s.q.InfraJob.WithContext(ctx).Where(s.q.InfraJob.ID.Eq(id)).First()
}

// GetJobPage 获取定时任务分页
func (s *JobService) GetJobPage(ctx context.Context, r *infra.JobPageReq) (*pagination.PageResult[*model.InfraJob], error) {
	q := s.q.InfraJob.WithContext(ctx)

	if r.Name != "" {
		q = q.Where(s.q.InfraJob.Name.Like("%" + r.Name + "%"))
	}
	if r.HandlerName != "" {
		q = q.Where(s.q.InfraJob.HandlerName.Like("%" + r.HandlerName + "%"))
	}
	if r.Status != nil {
		q = q.Where(s.q.InfraJob.Status.Eq(*r.Status))
	}

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

	list, err := q.Order(s.q.InfraJob.ID.Desc()).Offset(offset).Limit(pageSize).Find()
	if err != nil {
		return nil, err
	}

	return &pagination.PageResult[*model.InfraJob]{
		List:  list,
		Total: total,
	}, nil
}

// UpdateJobStatus 更新定时任务状态
func (s *JobService) UpdateJobStatus(ctx context.Context, id int64, status int) error {
	if status != JobStatusNormal && status != JobStatusStop {
		return errors.New("无效任务状态")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.q.InfraJob
	info, err := c.WithContext(ctx).Where(c.ID.Eq(id)).Update(c.Status, status)
	if err != nil {
		return err
	}
	if info.RowsAffected != 1 {
		return errors.New("任务不存在或无权访问")
	}
	return s.syncJob(ctx, id)
}

// TriggerJob 触发定时任务
func (s *JobService) TriggerJob(ctx context.Context, id int64) error {
	if s.scheduler != nil {
		return s.scheduler.TriggerJob(ctx, id)
	}
	return errors.New("调度器未初始化")
}

// SyncJob 同步定时任务 (从数据库重加载)
func (s *JobService) SyncJob(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scheduler == nil {
		return errors.New("调度器未初始化")
	}
	// Include stopped/deleted records so a failed removal can also be retried.
	jobs, err := s.q.InfraJob.WithContext(ctx).Unscoped().Find()
	if err != nil {
		return err
	}
	var result error
	for _, job := range jobs {
		result = errors.Join(result, s.syncJob(ctx, job.ID))
	}
	return result
}

// GetJobNextTimes 获取下几次执行时间
func (s *JobService) GetJobNextTimes(ctx context.Context, id int64, count int) ([]string, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, errors.New("任务不存在")
	}

	// 使用 Cron 库解析
	// 注意：这里需要引入 robo/cron 或类似的库解析 Cron 表达式
	// 为了简化，这里暂时返回空列表，后续补充具体 parsing 逻辑或调用 scheduler 方法
	// 如果 Scheduler 暴露了 Parse 逻辑最好

	return s.scheduler.GetNextTimes(job.CronExpression, count)
}
