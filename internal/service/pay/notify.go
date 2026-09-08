package pay

import (
	"bytes"
	"context"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	pay2 "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NotifyFrequency 通知频率，单位为秒
var NotifyFrequency = []int{15, 15, 30, 180, 1800, 1800, 1800, 3600}

// notifyHTTPClient 复用连接，超时在每个请求的 Context 上控制
var notifyHTTPClient = &http.Client{Timeout: notifyTimeout}

// PayNotifyTokenHeader 支付中心 → 商城业务通知的内部调用令牌头
const PayNotifyTokenHeader = "X-Pay-Notify-Token"

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

// notifyConcurrency 单次执行的最大并发通知数，避免任务堆积时打爆下游与连接池
// ponytail: 固定并发上限；量级上来后再改成可配置的 worker pool
const notifyConcurrency = 8

// notifyTimeout 单次通知的请求超时
const notifyTimeout = 10 * time.Second

// ExecuteNotify 执行回调通知 (Called by Job or Manually)
// 返回实际执行完成的任务数。任务使用脱离调用方取消信号的 Context，
// 保证 HTTP 请求或 Job 结束后已开始的通知不会被中途取消。
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

// buildNotifyBody 按通知类型构造商城可解析的请求体
// 对齐 Java: PayNotifyServiceImpl.executeNotifyInvoke 的 PayOrderNotifyReqDTO / PayRefundNotifyReqDTO
func buildNotifyBody(task *pay.PayNotifyTask) ([]byte, error) {
	switch task.Type {
	case PayNotifyTypeOrder:
		return json.Marshal(pay2.PayOrderNotifyReq{
			MerchantOrderId: task.MerchantOrderId,
			PayOrderID:      task.DataID,
		})
	case PayNotifyTypeRefund:
		return json.Marshal(pay2.PayRefundNotifyReqDTO{
			MerchantOrderId:  task.MerchantOrderId,
			MerchantRefundId: task.MerchantRefundId,
			PayRefundId:      task.DataID,
		})
	case PayNotifyTypeTransfer:
		return json.Marshal(map[string]any{
			"merchantTransferId": task.MerchantOrderId,
			"payTransferId":      task.DataID,
		})
	default:
		return nil, fmt.Errorf("unknown notify type: %d", task.Type)
	}
}

// isNotifySuccess 依据统一 JSON 业务码判定接收端是否真正处理成功。
// 商城接口返回 {"code":0,"msg":"","data":...}；HTTP 200 但 code != 0 属于业务失败，需要重试。
func isNotifySuccess(body []byte) bool {
	var result struct {
		Code *int `json:"code"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Code == nil {
		return false
	}
	return *result.Code == 0
}

func (s *PayNotifyService) executeNotifyTask(ctx context.Context, task *pay.PayNotifyTask) error {
	s.logger.Info("Start PayNotifyTask", zap.Int64("taskId", task.ID), zap.String("url", task.NotifyURL))

	log := &pay.PayNotifyLog{
		TaskID:      task.ID,
		NotifyTimes: task.NotifyTimes + 1,
	}

	status, responseText := s.invokeNotify(ctx, task)
	log.Response = responseText

	// 2. Update Task
	now := time.Now()
	task.LastExecuteTime = &now
	task.NotifyTimes++
	task.Status = status

	if status != PayNotifyStatusSuccess {
		if task.NotifyTimes >= task.MaxNotifyTimes {
			// 重试耗尽：置为最终失败，等待人工重放；此处应触发告警
			task.Status = PayNotifyStatusFailure
			s.logger.Error("pay notify retries exhausted, manual replay required",
				zap.Int64("taskId", task.ID),
				zap.Int("type", task.Type),
				zap.Int64("dataId", task.DataID),
				zap.String("merchantOrderId", task.MerchantOrderId))
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

// invokeNotify 发起一次通知请求，返回本次的通知状态与响应文本
func (s *PayNotifyService) invokeNotify(ctx context.Context, task *pay.PayNotifyTask) (int, string) {
	body, err := buildNotifyBody(task)
	if err != nil {
		return PayNotifyStatusRequestFailure, err.Error()
	}
	if _, err := url.ParseRequestURI(task.NotifyURL); err != nil {
		return PayNotifyStatusRequestFailure, fmt.Sprintf("invalid notify url: %v", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, notifyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, task.NotifyURL, bytes.NewReader(body))
	if err != nil {
		return PayNotifyStatusRequestFailure, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	if token := config.C.Pay.NotifyToken; token != "" {
		req.Header.Set(PayNotifyTokenHeader, token)
	}

	resp, err := notifyHTTPClient.Do(req)
	if err != nil {
		return PayNotifyStatusRequestFailure, err.Error()
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return PayNotifyStatusRequestFailure, err.Error()
	}
	if resp.StatusCode != http.StatusOK {
		return PayNotifyStatusRequestFailure, fmt.Sprintf("http %d: %s", resp.StatusCode, respBody)
	}
	if !isNotifySuccess(respBody) {
		// HTTP 200 但业务码非 0：请求成功、结果失败，仍需重试
		return PayNotifyStatusRequestSuccess, string(respBody)
	}
	return PayNotifyStatusSuccess, string(respBody)
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
