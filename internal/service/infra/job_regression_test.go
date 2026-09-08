package infra

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	contract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/infra"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pc "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type reviewJobHandler struct{}

func (*reviewJobHandler) GetHandlerName() string                { return "review-only-probe" }
func (*reviewJobHandler) Execute(context.Context, string) error { return nil }
func TestReviewCrossTenantSchedulerMutation(t *testing.T) {
	for _, action := range []string{"delete", "stop"} {
		t.Run(action, func(t *testing.T) {
			db := testutil.PostgreSQL(t)
			require.NoError(t, db.Model(&model.InfraJob{}).Where("tenant_id = ?", 1).Update("status", JobStatusStop).Error)
			require.NoError(t, db.Create(&model.SystemTenant{ID: 2, Name: "review tenant B", ExpireDate: time.Now().Add(time.Hour)}).Error)
			job := &model.InfraJob{Name: "B job", Status: JobStatusNormal, HandlerName: "review-only-probe", CronExpression: "0 0 0 * * *", TenantBaseDO: model.TenantBaseDO{TenantID: 2}}
			require.NoError(t, db.Create(job).Error)
			require.NoError(t, db.Use(&database.TenantPlugin{}))
			q := query.Use(db)
			scheduler, err := NewScheduler(q, zap.NewNop(), []JobHandler{&reviewJobHandler{}})
			require.NoError(t, err)
			t.Cleanup(func() { scheduler.Shutdown() })
			require.Contains(t, scheduler.jobMap, job.ID)
			svc := NewJobService(q, scheduler)
			ctxA := pc.WithTenant(context.Background(), 1)
			if action == "delete" {
				err = svc.DeleteJob(ctxA, job.ID)
			} else {
				err = svc.UpdateJobStatus(ctxA, job.ID, JobStatusStop)
			}
			var saved model.InfraJob
			require.NoError(t, db.WithContext(pc.WithTenant(context.Background(), 2)).First(&saved, job.ID).Error)
			t.Logf("A action=%s error=%v; B database status=%d; B scheduled=%v", action, err, saved.Status, scheduler.jobMap[job.ID] != nil)
			require.Error(t, err)
			require.Contains(t, scheduler.jobMap, job.ID, "Tenant A must not remove tenant B's live scheduled job")
		})
	}
}
func TestReviewCreateJobRegistersAfterCommit(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Model(&model.InfraJob{}).Where("tenant_id = ?", 1).Update("status", JobStatusStop).Error)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	scheduler, err := NewScheduler(q, zap.NewNop(), []JobHandler{&reviewJobHandler{}})
	require.NoError(t, err)
	t.Cleanup(func() { scheduler.Shutdown() })
	id, err := NewJobService(q, scheduler).CreateJob(pc.WithTenant(context.Background(), 1), &contract.JobSaveReq{Name: "new job", HandlerName: "review-only-probe", CronExpression: "0 0 0 * * *"})
	require.NoError(t, err, "valid new job must be visible when registered")
	require.Contains(t, scheduler.jobMap, id)
}

func TestJobMutationLifecycleAndRecovery(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Model(&model.InfraJob{}).Where("tenant_id = ?", 1).Update("status", JobStatusStop).Error)
	require.NoError(t, db.Create(&model.SystemTenant{ID: 2, Name: "B", ExpireDate: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	scheduler, err := NewScheduler(q, zap.NewNop(), []JobHandler{&reviewJobHandler{}})
	require.NoError(t, err)
	t.Cleanup(func() { scheduler.Shutdown() })
	svc := NewJobService(q, scheduler)
	a := pc.WithTenant(context.Background(), 1)
	b := pc.WithTenant(context.Background(), 2)
	req := &contract.JobSaveReq{Name: "new", HandlerName: "review-only-probe", CronExpression: "0 0 0 * * *", HandlerParam: "original"}
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	// A single connection must suffice: no root query inside an uncommitted transaction.
	id, err := svc.CreateJob(a, req)
	require.NoError(t, err)
	req.ID = &id
	other := &model.InfraJob{Name: "B job", Status: JobStatusNormal, HandlerName: req.HandlerName, CronExpression: req.CronExpression}
	require.NoError(t, db.WithContext(b).Create(other).Error)
	require.NoError(t, scheduler.SyncJob(b, other.ID))
	require.Error(t, svc.DeleteJobList(a, []int64{id, other.ID}))
	require.Contains(t, scheduler.jobMap, id)
	require.Contains(t, scheduler.jobMap, other.ID)
	require.Error(t, scheduler.SyncJob(a, other.ID))
	require.Error(t, scheduler.SyncJob(context.Background(), other.ID))
	require.Error(t, scheduler.TriggerJob(a, other.ID))
	var saved model.InfraJob
	require.NoError(t, db.WithContext(a).First(&saved, id).Error)
	require.Equal(t, JobStatusNormal, saved.Status)
	// Failed SQL writes must not alter the existing timer or its database row.
	originalTimer := scheduler.jobMap[id].ID()
	for _, operation := range []string{"update", "delete"} {
		fail := func(tx *gorm.DB) {
			if tx.Statement.Table == "infra_job" {
				tx.AddError(errors.New("review SQL failure"))
			}
		}
		if operation == "update" {
			require.NoError(t, db.Callback().Update().After("gorm:update").Before("gorm:commit_or_rollback_transaction").Register("review:fail", fail))
			req.Name = "must rollback"
			require.Error(t, svc.UpdateJob(a, req))
			require.NoError(t, db.Callback().Update().Remove("review:fail"))
		} else {
			require.NoError(t, db.Callback().Delete().After("gorm:delete").Before("gorm:commit_or_rollback_transaction").Register("review:fail", fail))
			require.Error(t, svc.DeleteJob(a, id))
			require.NoError(t, db.Callback().Delete().Remove("review:fail"))
		}
		require.NoError(t, db.WithContext(a).First(&saved, id).Error)
		require.Equal(t, "new", saved.Name)
		require.Equal(t, originalTimer, scheduler.jobMap[id].ID())
	}
	req.Name = "changed"
	req.CronExpression = "0 0 12 * * *"
	req.HandlerParam = "updated"
	require.NoError(t, svc.UpdateJob(a, req))
	require.NoError(t, db.WithContext(a).First(&saved, id).Error)
	require.Equal(t, "updated", saved.HandlerParam)
	next, err := scheduler.jobMap[id].NextRun()
	require.NoError(t, err)
	require.Equal(t, 12, next.In(time.Local).Hour())
	// Repeated and concurrent synchronization never accumulates duplicate schedules.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- svc.SyncJob(a) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Len(t, scheduler.scheduler.Jobs(), 2)
	require.NoError(t, svc.UpdateJobStatus(a, id, JobStatusStop))
	require.NotContains(t, scheduler.jobMap, id)
	require.Contains(t, scheduler.jobMap, other.ID)
	require.Error(t, svc.UpdateJobStatus(a, id, 99))
	require.NoError(t, svc.UpdateJobStatus(a, id, JobStatusNormal))
	require.Contains(t, scheduler.jobMap, id)
	// A registration error is explicit; the persisted job is recoverable by sync.
	realScheduler := scheduler.scheduler
	scheduler.scheduler = &failRegistration{Scheduler: realScheduler}
	require.ErrorContains(t, svc.SyncJob(a), "调度同步失败")
	require.NotContains(t, scheduler.jobMap, id)
	require.NoError(t, db.WithContext(a).First(&saved, id).Error)
	require.Equal(t, JobStatusNormal, saved.Status)
	scheduler.scheduler = realScheduler
	require.NoError(t, svc.SyncJob(a))
	require.Contains(t, scheduler.jobMap, id)
	// A failed timer removal can be retried after the DB tombstone has committed.
	scheduler.scheduler = &failRemoval{Scheduler: realScheduler}
	require.ErrorContains(t, svc.DeleteJob(a, id), "调度同步失败")
	require.Error(t, db.WithContext(a).First(&model.InfraJob{}, id).Error)
	require.Contains(t, scheduler.jobMap, id)
	scheduler.scheduler = realScheduler
	require.NoError(t, svc.SyncJob(a))
	require.NotContains(t, scheduler.jobMap, id)
	require.Contains(t, scheduler.jobMap, other.ID)
	require.Len(t, scheduler.scheduler.Jobs(), 1)
}

type failRegistration struct{ gocron.Scheduler }

func (*failRegistration) NewJob(gocron.JobDefinition, gocron.Task, ...gocron.JobOption) (gocron.Job, error) {
	return nil, errors.New("review registration failure")
}

type failRemoval struct{ gocron.Scheduler }

func (*failRemoval) RemoveJob(id uuid.UUID) error { return errors.New("review removal failure") }

func TestJobCreateRegistrationFailureCanRecover(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Model(&model.InfraJob{}).Where("tenant_id = ?", 1).Update("status", JobStatusStop).Error)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	scheduler, err := NewScheduler(q, zap.NewNop(), []JobHandler{&reviewJobHandler{}})
	require.NoError(t, err)
	t.Cleanup(func() { scheduler.Shutdown() })
	ctx := pc.WithTenant(context.Background(), 1)
	svc := NewJobService(q, scheduler)
	request := &contract.JobSaveReq{Name: "recover", HandlerName: "review-only-probe", CronExpression: "0 0 0 * * *"}
	require.NoError(t, db.Callback().Create().After("gorm:create").Before("gorm:commit_or_rollback_transaction").Register("review:fail-create", func(tx *gorm.DB) {
		if tx.Statement.Table == "infra_job" {
			tx.AddError(errors.New("review create SQL failure"))
		}
	}))
	id, err := svc.CreateJob(ctx, request)
	require.Error(t, err)
	require.Zero(t, id)
	require.Empty(t, scheduler.scheduler.Jobs())
	require.NoError(t, db.Callback().Create().Remove("review:fail-create"))
	count, err := q.InfraJob.WithContext(ctx).Where(q.InfraJob.HandlerName.Eq(request.HandlerName)).Count()
	require.NoError(t, err)
	require.Zero(t, count)
	realScheduler := scheduler.scheduler
	scheduler.scheduler = &failRegistration{Scheduler: realScheduler}
	id, err = svc.CreateJob(ctx, request)
	require.Positive(t, id)
	require.Error(t, err)
	require.ErrorContains(t, err, "已保存")
	require.NotContains(t, scheduler.jobMap, id)
	var row model.InfraJob
	require.NoError(t, db.WithContext(ctx).First(&row, id).Error)
	scheduler.scheduler = realScheduler
	require.NoError(t, svc.SyncJob(ctx))
	require.Contains(t, scheduler.jobMap, id)
	require.Len(t, scheduler.scheduler.Jobs(), 1)
}

func TestJobStaleTimerSkipsStoppedAndDeletedRows(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := pc.WithTenant(context.Background(), 1)
	probe := &tenantJobProbe{t: t, tenant: 1}
	row := &model.InfraJob{Name: "stale", Status: JobStatusNormal, HandlerName: probe.GetHandlerName(), CronExpression: "0 0 0 1 1 *"}
	require.NoError(t, db.WithContext(ctx).Create(row).Error)
	scheduler := &Scheduler{q: query.Use(db), log: zap.NewNop()}
	require.NoError(t, db.WithContext(ctx).Model(row).Update("status", JobStatusStop).Error)
	scheduler.executeJob(row, probe, true)
	require.Zero(t, probe.calls, "a stale timer cannot run a stopped job")
	scheduler.executeJob(row, probe, false)
	require.Equal(t, 1, probe.calls, "manual execution of an own stopped job remains supported")
	require.NoError(t, db.WithContext(ctx).Delete(row).Error)
	scheduler.executeJob(row, probe, true)
	scheduler.executeJob(row, probe, false)
	require.Equal(t, 1, probe.calls, "neither execution path can run a deleted job")
}
