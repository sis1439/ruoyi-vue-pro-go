package infra

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/robfig/cron/v3"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"go.uber.org/zap"
)

// JobHandler 定时任务处理器接口
type JobHandler interface {
	Execute(ctx context.Context, param string) error
	GetHandlerName() string
}

// Scheduler 使用 gocron/v2 管理定时任务调度器
type Scheduler struct {
	scheduler gocron.Scheduler
	q         *query.Query
	log       *zap.Logger
	handlers  map[string]JobHandler
	jobMap    map[int64]gocron.Job
	mu        sync.RWMutex
}

// NewScheduler 创建新的调度器实例
func NewScheduler(q *query.Query, log *zap.Logger, handlers []JobHandler) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	scheduler := &Scheduler{
		scheduler: s,
		q:         q,
		log:       log,
		handlers:  make(map[string]JobHandler),
		jobMap:    make(map[int64]gocron.Job),
	}

	// 自动注册所有传入的任务处理器
	for _, handler := range handlers {
		scheduler.RegisterHandler(handler.GetHandlerName(), handler)
	}

	if err := scheduler.Start(context.Background()); err != nil {
		_ = s.Shutdown()
		return nil, err
	}

	return scheduler, nil
}

// RegisterHandler 按名称注册任务处理器
func (s *Scheduler) RegisterHandler(name string, handler JobHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[name] = handler
}

// HasHandler 检查指定名称的 Handler 是否已注册
func (s *Scheduler) HasHandler(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.handlers[name]
	return ok
}

// GetRegisteredHandlers 获取所有已注册的 Handler 名称
func (s *Scheduler) GetRegisteredHandlers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.handlers))
	for name := range s.handlers {
		names = append(names, name)
	}
	return names
}

// Start 从数据库加载所有启用的任务并启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	t := s.q.SystemTenant
	tenants, err := t.WithContext(ctx).Where(t.Status.Eq(0), t.ExpireDate.Gt(time.Now())).Find()
	if err != nil {
		return err
	}
	count := 0
	for _, tenant := range tenants {
		tenantCtx := pkgcontext.WithTenant(ctx, tenant.ID)
		jobs, err := s.q.InfraJob.WithContext(tenantCtx).Where(s.q.InfraJob.Status.Eq(JobStatusNormal)).Find()
		if err != nil {
			return err
		}
		for _, job := range jobs {
			if err := s.SyncJob(tenantCtx, job.ID); err != nil {
				return err
			}
			count++
		}
	}

	s.scheduler.Start()
	s.log.Info("Scheduler started", zap.Int("jobCount", count))
	return nil
}

// Shutdown 停止调度器
func (s *Scheduler) Shutdown() error {
	return s.scheduler.Shutdown()
}

// scheduleJobLocked replaces one already-authorized task; caller holds mu.
func (s *Scheduler) scheduleJobLocked(job *model.InfraJob) error {
	handler, ok := s.handlers[job.HandlerName]
	if !ok {
		return fmt.Errorf("handler not found: %s", job.HandlerName)
	}
	if err := s.ValidateCronExpression(job.CronExpression); err != nil {
		return err
	}
	if err := s.removeJobLocked(job.ID); err != nil {
		return err
	}
	scheduled, err := s.scheduler.NewJob(gocron.CronJob(job.CronExpression, true),
		gocron.NewTask(func() { s.executeJob(job, handler, true) }),
		gocron.WithName(fmt.Sprintf("job-%d", job.ID)))
	if err != nil {
		return err
	}
	if scheduled == nil {
		return fmt.Errorf("scheduler unavailable")
	}
	s.jobMap[job.ID] = scheduled
	s.log.Info("Job scheduled", zap.Int64("jobId", job.ID), zap.Int64("tenantId", job.TenantID))
	return nil
}

// executeJob 执行任务并记录结果
func (s *Scheduler) executeJob(job *model.InfraJob, handler JobHandler, scheduled bool) {
	// Never retain an HTTP context: scheduled work restores the tenant from its persisted job.
	if job.TenantID <= 0 {
		s.log.Error("Refusing job without tenant", zap.Int64("jobId", job.ID))
		return
	}
	ctx, cancel := context.WithTimeout(pkgcontext.WithTenant(context.Background(), job.TenantID), 5*time.Minute)
	defer cancel()
	tenants := s.q.SystemTenant
	if _, err := tenants.WithContext(ctx).Where(tenants.ID.Eq(job.TenantID), tenants.Status.Eq(0), tenants.ExpireDate.Gt(time.Now())).First(); err != nil {
		s.log.Error("Refusing job for unavailable tenant", zap.Int64("jobId", job.ID), zap.Error(err))
		return
	}
	c := s.q.InfraJob
	currentQuery := c.WithContext(ctx).Where(c.ID.Eq(job.ID), c.TenantID.Eq(job.TenantID), c.HandlerName.Eq(job.HandlerName))
	if scheduled {
		currentQuery = currentQuery.Where(c.Status.Eq(JobStatusNormal))
	}
	current, err := currentQuery.First()
	if err != nil {
		s.log.Warn("Skipping unavailable scheduled job", zap.Int64("jobId", job.ID), zap.Error(err))
		return
	}
	job = current
	beginTime := time.Now()

	logRecord := &model.InfraJobLog{
		JobID:        job.ID,
		HandlerName:  job.HandlerName,
		HandlerParam: job.HandlerParam,
		ExecuteIndex: 1,
		BeginTime:    beginTime,
		Status:       0,
	}
	if err := s.q.InfraJobLog.WithContext(ctx).Create(logRecord); err != nil {
		s.log.Error("Failed to persist job start", zap.Int64("jobId", job.ID), zap.Error(err))
		return
	}

	var status int
	var result string
	err = handler.Execute(ctx, job.HandlerParam)
	endTime := time.Now()
	duration := int(endTime.Sub(beginTime).Milliseconds())

	if err != nil {
		status = 2
		result = err.Error()
		s.log.Error("Job execution failed", zap.Int64("jobId", job.ID), zap.Error(err))
	} else {
		status = 1
		result = "success"
		s.log.Info("Job execution completed", zap.Int64("jobId", job.ID), zap.Int("duration", duration))
	}

	_, saveErr := s.q.InfraJobLog.WithContext(ctx).Where(s.q.InfraJobLog.ID.Eq(logRecord.ID)).Updates(map[string]interface{}{
		"end_time": endTime,
		"duration": duration,
		"status":   status,
		"result":   result,
	})
	if saveErr != nil {
		s.log.Error("Failed to persist job result", zap.Int64("jobId", job.ID), zap.Error(saveErr))
	}
}

// SyncJob is the only public scheduling mutation: verify ownership, then apply committed state.
// Unscoped reads tombstones for removal retries; the explicit tenant predicate still applies.
func (s *Scheduler) SyncJob(ctx context.Context, jobID int64) error {
	tenant, ok := pkgcontext.TenantID(ctx)
	if !ok {
		return fmt.Errorf("trusted job tenant required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.q.InfraJob
	job, err := c.WithContext(ctx).Unscoped().Where(c.ID.Eq(jobID), c.TenantID.Eq(tenant)).First()
	if err != nil {
		return err
	}
	if bool(job.Deleted) || job.Status != JobStatusNormal {
		return s.removeJobLocked(jobID)
	}
	return s.scheduleJobLocked(job)
}

func (s *Scheduler) removeJobLocked(jobID int64) error {
	scheduled, ok := s.jobMap[jobID]
	if !ok {
		return nil
	}
	if err := s.scheduler.RemoveJob(scheduled.ID()); err != nil {
		return err
	}
	delete(s.jobMap, jobID)
	return nil
}

// TriggerJob 立即执行任务
func (s *Scheduler) TriggerJob(ctx context.Context, jobID int64) error {
	tenant, ok := pkgcontext.TenantID(ctx)
	if !ok {
		return fmt.Errorf("trusted job tenant required")
	}
	job, err := s.q.InfraJob.WithContext(ctx).Where(s.q.InfraJob.ID.Eq(jobID), s.q.InfraJob.TenantID.Eq(tenant)).First()
	if err != nil {
		return err
	}

	s.mu.RLock()
	handler, ok := s.handlers[job.HandlerName]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("handler not found: %s", job.HandlerName)
	}

	go s.executeJob(job, handler, false)
	return nil
}

// ValidateCronExpression 校验 cron 表达式是否合法
func (s *Scheduler) ValidateCronExpression(cronExpression string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(cronExpression)
	if err != nil {
		return fmt.Errorf("无效的 cron 表达式: %w", err)
	}
	return nil
}

// GetNextTimes 计算 cron 表达式的下 n 次执行时间
// 支持标准 5 字段格式 (分 时 日 月 周) 和 Quartz 6 字段格式 (秒 分 时 日 月 周)
func (s *Scheduler) GetNextTimes(cronExpression string, count int) ([]string, error) {
	// 使用 robfig/cron 解析 cron 表达式
	// 添加 Second 字段以支持 6 字段的 Quartz 格式
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cronExpression)
	if err != nil {
		return nil, fmt.Errorf("无效的 cron 表达式: %w", err)
	}

	// 计算未来 count 次执行时间
	var times []string
	currentTime := time.Now()

	for i := 0; i < count; i++ {
		nextTime := schedule.Next(currentTime)
		times = append(times, nextTime.Format(time.DateTime))
		// 推进到下一次执行时间之后，以获取后续时间
		currentTime = nextTime
	}

	return times, nil
}
