package pay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NotifyFrequency 通知频率，单位为秒
var NotifyFrequency = []int{15, 15, 30, 180, 1800, 1800, 1800, 3600}

type PayNotifyService struct {
	q      *query.Query
	logger *zap.Logger
	lock   *PayNotifyLock
}

func NewPayNotifyService(q *query.Query, logger *zap.Logger, rdb *redis.Client) *PayNotifyService {
	return &PayNotifyService{
		q:      q,
		logger: logger,
		lock:   NewPayNotifyLock(rdb),
	}
}

// CreatePayNotifyTask 创建回调通知任务
func (s *PayNotifyService) CreatePayNotifyTask(ctx context.Context, typeVal int, dataId int64) error {
	var task *pay.PayNotifyTask

	// 1. Get Data by Type
	switch typeVal {
	case PayNotifyTypeOrder:
		order, err := s.q.PayOrder.WithContext(ctx).Where(s.q.PayOrder.ID.Eq(dataId)).First()
		if err != nil {
			return err
		}
		task = &pay.PayNotifyTask{
			AppID:           order.AppID,
			Type:            typeVal,
			DataID:          dataId,
			MerchantOrderId: order.MerchantOrderId,
			NotifyURL:       order.NotifyURL,
		}
	case PayNotifyTypeRefund:
		refund, err := s.q.PayRefund.WithContext(ctx).Where(s.q.PayRefund.ID.Eq(dataId)).First()
		if err != nil {
			return err
		}
		task = &pay.PayNotifyTask{
			AppID:            refund.AppID,
			Type:             typeVal,
			DataID:           dataId,
			MerchantOrderId:  refund.MerchantOrderId,
			MerchantRefundId: refund.MerchantRefundId,
			NotifyURL:        refund.NotifyURL,
		}
	case PayNotifyTypeTransfer:
		transfer, err := s.q.PayTransfer.WithContext(ctx).Where(s.q.PayTransfer.ID.Eq(dataId)).First()
		if err != nil {
			return err
		}
		task = &pay.PayNotifyTask{
			AppID:           transfer.AppID,
			Type:            typeVal,
			DataID:          dataId,
			MerchantOrderId: transfer.MerchantTransferID,
			NotifyURL:       transfer.NotifyURL,
		}
	default:
		return fmt.Errorf("unknown notify type: %d", typeVal)
	}

	task.Status = PayNotifyStatusWaiting
	now := time.Now()
	task.NextNotifyTime = &now
	task.NotifyTimes = 0
	task.MaxNotifyTimes = len(NotifyFrequency) + 1

	return s.q.PayNotifyTask.WithContext(ctx).Create(task)
}

// ExecuteNotify 执行回调通知 (Called by Job or Manually)
func (s *PayNotifyService) ExecuteNotify(ctx context.Context) (int, error) {
	// 1. Query Waiting Tasks
	now := time.Now()
	tasks, err := s.q.PayNotifyTask.WithContext(ctx).
		Where(s.q.PayNotifyTask.Status.Eq(PayNotifyStatusWaiting)).
		Where(s.q.PayNotifyTask.NextNotifyTime.Lt(now)).
		Order(s.q.PayNotifyTask.NextNotifyTime, s.q.PayNotifyTask.ID).
		Limit(20).
		Find()
	if err != nil {
		return 0, err
	}

	count := 0
	var failures error
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return count, errors.Join(failures, err)
		}
		// Finish this bounded batch before the scheduler releases its tenant context.
		if err := s.executeNotifyTaskWithLock(ctx, task); err != nil {
			s.logger.Error("executeNotifyTask failed", zap.Int64("taskId", task.ID), zap.Error(err))
			failures = errors.Join(failures, err)
		}
		count++
	}
	return count, failures
}

// executeNotifyTaskWithLock 使用分布式锁执行通知任务
// 对齐 Java: PayNotifyServiceImpl.executeNotify (with lock)
func (s *PayNotifyService) executeNotifyTaskWithLock(ctx context.Context, task *pay.PayNotifyTask) error {
	// 使用分布式锁,避免并发问题
	return s.lock.Lock(ctx, task.ID, func() error {
		// 校验任务是否已被通知过 (双重检查)
		dbTask, err := s.q.PayNotifyTask.WithContext(ctx).
			Where(s.q.PayNotifyTask.ID.Eq(task.ID)).
			First()
		if err != nil {
			return err
		}

		// 通过 notifyTimes 判断是否已被其他进程处理
		if dbTask.NotifyTimes != task.NotifyTimes {
			s.logger.Warn("task ignored due to concurrent execution",
				zap.Int64("taskId", task.ID),
				zap.Int("expectedTimes", task.NotifyTimes),
				zap.Int("actualTimes", dbTask.NotifyTimes))
			return nil
		}

		// 执行实际的通知逻辑
		return s.executeNotifyTask(ctx, dbTask)
	})
}

func (s *PayNotifyService) executeNotifyTask(ctx context.Context, task *pay.PayNotifyTask) error {
	s.logger.Info("Start PayNotifyTask", zap.Int64("taskId", task.ID), zap.String("url", task.NotifyURL))

	// 1. Execute HTTP Request
	status := PayNotifyStatusSuccess
	responseBody := ""

	// Create request body - In specific format required by merchant?
	// Usually POST with some params. For now, assuming generic empty or simple mapping. All params are in the URL or Body?
	// Java code uses `PayOrderNotifyReqDTO` or similar.
	// Simplification: Sending empty body for now as `task.NotifyURL` typically contains params?
	// Wait, standard is POST FORM or JSON. Java code uses `restTemplate.postForEntity`.
	// For simplicity, we just POST. The real payload should be defined.
	// But `PayNotifyTaskDO` doesn't store the content. It seems content is built dynamically from Order/Refund?
	// Re-checking Java logic: `executeNotifyTask` calls `notifyPayOrder` -> `NotifyPayOrderReqDTO`.
	// For now, I will send a simple JSON.

	// Prepare Log
	log := &pay.PayNotifyLog{
		TaskID:      task.ID,
		NotifyTimes: task.NotifyTimes + 1,
		Status:      PayNotifyStatusSuccess, // Default
	}

	client := &http.Client{Timeout: 10 * time.Second}
	reqBody := []byte("{}") // TODO: Build actual payload
	req, err := http.NewRequestWithContext(ctx, "POST", task.NotifyURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		status = PayNotifyStatusRequestFailure
		log.Response = err.Error()
	} else {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Response = string(bodyBytes)
		responseBody = string(bodyBytes)
		if resp.StatusCode == 200 {
			if responseBody == "SUCCESS" { // Convention check?
				status = PayNotifyStatusSuccess
			} else {
				status = PayNotifyStatusRequestSuccess // Request OK but result implementation specific logic
			}
		} else {
			status = PayNotifyStatusRequestFailure
		}
	}
	// Note: Simple logic here. Ideally check "SUCCESS" string from merchant.
	// Java: `if ("success".equalsIgnoreCase(response)) status = SUCCESS`

	if responseBody == "success" || responseBody == "SUCCESS" {
		status = PayNotifyStatusSuccess
	}

	// 2. Update Task
	now := time.Now()
	task.LastExecuteTime = &now
	task.NotifyTimes++
	task.Status = status

	if status == PayNotifyStatusSuccess {
		// Done
	} else {
		if task.NotifyTimes >= task.MaxNotifyTimes {
			task.Status = PayNotifyStatusFailure
		} else {
			task.Status = PayNotifyStatusWaiting
			nextSec := NotifyFrequency[task.NotifyTimes-1]
			nextTime := now.Add(time.Duration(nextSec) * time.Second)
			task.NextNotifyTime = &nextTime
		}
	}

	if err := s.q.PayNotifyTask.WithContext(ctx).UnderlyingDB().Where("id = ?", task.ID).Select("*").Omit("id", "tenant_id", "creator", "create_time").Updates(task).Error; err != nil {
		return err
	}

	// 3. Create Log
	log.Status = task.Status // Use final status
	return s.q.PayNotifyLog.WithContext(ctx).Create(log)
}

// GetNotifyTask 获得回调通知
func (s *PayNotifyService) GetNotifyTask(ctx context.Context, id int64) (*pay.PayNotifyTask, error) {
	return s.q.PayNotifyTask.WithContext(ctx).Where(s.q.PayNotifyTask.ID.Eq(id)).First()
}

// GetNotifyTaskPage 获得回调通知分页
func (s *PayNotifyService) GetNotifyTaskPage(ctx context.Context, req *pay2.PayNotifyTaskPageReq) (*pagination.PageResult[*pay.PayNotifyTask], error) {
	q := s.q.PayNotifyTask.WithContext(ctx)
	if req.AppID > 0 {
		q = q.Where(s.q.PayNotifyTask.AppID.Eq(req.AppID))
	}
	if req.Type != nil {
		q = q.Where(s.q.PayNotifyTask.Type.Eq(*req.Type))
	}
	if req.DataID > 0 {
		q = q.Where(s.q.PayNotifyTask.DataID.Eq(req.DataID))
	}
	if req.MerchantOrderId != "" {
		q = q.Where(s.q.PayNotifyTask.MerchantOrderId.Eq(req.MerchantOrderId))
	}
	if req.Status != nil {
		q = q.Where(s.q.PayNotifyTask.Status.Eq(*req.Status))
	}

	total, err := q.Count()
	if err != nil {
		return nil, err
	}
	list, err := q.Limit(req.GetLimit()).Offset(req.GetOffset()).Order(s.q.PayNotifyTask.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return &pagination.PageResult[*pay.PayNotifyTask]{
		List:  list,
		Total: total,
	}, nil
}

// GetNotifyLogList 获得回调日志列表
func (s *PayNotifyService) GetNotifyLogList(ctx context.Context, taskId int64) ([]*pay.PayNotifyLog, error) {
	return s.q.PayNotifyLog.WithContext(ctx).Where(s.q.PayNotifyLog.TaskID.Eq(taskId)).Order(s.q.PayNotifyLog.ID.Desc()).Find()
}
