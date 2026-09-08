package pay

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	payreq "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	payrepo "github.com/wxlbd/ruoyi-mall-go/internal/repo/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func reviewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.PostgreSQL(t)
	for _, statement := range []string{
		"UPDATE pay_app SET app_key='review-app',status=0 WHERE id=1",
		"INSERT INTO pay_channel(id,app_id,code,status,tenant_id) VALUES(1,1,'wx_lite',0,1)",
		"INSERT INTO pay_order(id,app_id,merchant_order_id,price,status,tenant_id,refund_price) VALUES(1,1,'1',100,0,1,0)",
		"INSERT INTO pay_order_extension(id,order_id,channel_id,no,status,tenant_id) VALUES(1,1,1,'review-P1',0,1)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	return db.WithContext(pkgcontext.WithTenant(context.Background(), 1))
}
func TestReviewNotifyInsertFailureMustRollbackPayment(t *testing.T) {
	db := reviewDB(t)
	q := query.Use(db)
	ns := &PayNotifyService{q: q}
	s := &PayOrderService{q: q, channelSvc: NewPayChannelService(q, client.NewPayClientFactory()), notifySvc: ns}
	reached, sameTx := false, false
	var paymentPool gorm.ConnPool
	db.Callback().Update().Before("gorm:update").Register("review-payment-pool", func(tx *gorm.DB) {
		if tx.Statement.Table == "pay_order" {
			paymentPool = tx.Statement.ConnPool
		}
	})
	db.Callback().Create().Before("gorm:create").Register("review-fail-notify", func(tx *gorm.DB) {
		if tx.Statement.Table == "pay_notify_task" {
			reached = true
			sameTx = tx.Statement.ConnPool == paymentPool
			tx.AddError(errors.New("injected notification INSERT failure"))
		}
	})
	err := s.NotifyOrder(pkgcontext.WithTenant(context.Background(), 1), 1, &client.OrderResp{Status: consts.PayOrderStatusSuccess, OutTradeNo: "review-P1", Price: 100})
	var status int
	db.Model(&payModel.PayOrder{}).Where("id=1").Pluck("status", &status)
	var tasks int64
	db.Model(&payModel.PayNotifyTask{}).Count(&tasks)
	t.Logf("notifyInsertAttempted=%v insertSharesPaymentTransaction=%v returnedError=%v persistedPaymentStatus=%d tasks=%d", reached, sameTx, err, status, tasks)
	if !reached {
		t.Fatal("test did not reach injection")
	}
	if !sameTx || err == nil || status != consts.PayOrderStatusWaiting {
		t.Fatal("notification INSERT failure was ignored and paid state committed without durable delivery")
	}
}
func TestReviewZeroPaidAmountMustBeRejected(t *testing.T) {
	db := reviewDB(t)
	q := query.Use(db)
	s := &PayOrderService{q: q, channelSvc: NewPayChannelService(q, client.NewPayClientFactory()), notifySvc: &PayNotifyService{q: q}}
	err := s.NotifyOrder(pkgcontext.WithTenant(context.Background(), 1), 1, &client.OrderResp{Status: consts.PayOrderStatusSuccess, OutTradeNo: "review-P1", Price: 0})
	var status int
	db.Model(&payModel.PayOrder{}).Where("id=1").Pluck("status", &status)
	if err == nil && status == consts.PayOrderStatusSuccess {
		t.Fatal("zero/missing channel amount accepted for 100-fen payment")
	}
}

type reviewCachedClient struct{ client.PayClient }

func (*reviewCachedClient) Init() error { return nil }
func TestReviewDisabledCachedChannelMustBeRejected(t *testing.T) {
	db := reviewDB(t)
	q := query.Use(db)
	factory := client.NewPayClientFactory()
	client.RegisterCreator("review_cache", func(int64, string, string) (client.PayClient, error) { return &reviewCachedClient{}, nil })
	if _, err := factory.CreateOrUpdatePayClient(1, "review_cache", "{}"); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&payModel.PayChannel{}).Where("id=1").Update("status", 1).Error; err != nil {
		t.Fatal(err)
	}
	c, err := NewPayChannelService(q, factory).GetPayClient(pkgcontext.WithTenant(context.Background(), 1), 1)
	if err == nil && c != nil {
		t.Fatal("disabled channel returns cached client without checking database")
	}
}

func TestReviewPaymentReplayAndChannelBinding(t *testing.T) {
	db := reviewDB(t)
	q := query.Use(db)
	ctx := db.Statement.Context
	svc := &PayOrderService{q: q, channelSvc: NewPayChannelService(q, client.NewPayClientFactory()), notifySvc: &PayNotifyService{q: q}}
	dto := &client.OrderResp{Status: consts.PayOrderStatusSuccess, OutTradeNo: "review-P1", Price: 100}
	for range 3 {
		if err := svc.NotifyOrder(ctx, 1, dto); err != nil {
			t.Fatal(err)
		}
	}
	var tasks int64
	if err := db.Model(&payModel.PayNotifyTask{}).Count(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if tasks != 1 {
		t.Fatalf("duplicate tasks: %d", tasks)
	}
	dto.Price = 0
	if err := svc.NotifyOrder(ctx, 1, dto); err == nil {
		t.Fatal("duplicate with invalid amount accepted")
	}
}
func TestReviewRefundNotificationFailureRollsBackTotal(t *testing.T) {
	db := reviewDB(t)
	q := query.Use(db)
	ctx := db.Statement.Context
	if err := db.Model(&payModel.PayOrder{}).Where("id=1").Updates(map[string]any{"status": consts.PayOrderStatusSuccess, "channel_id": 1, "no": "review-P1"}).Error; err != nil {
		t.Fatal(err)
	}
	refund := &payModel.PayRefund{ID: 1, No: "R-review", AppID: 1, OrderID: 1, OrderNo: "review-P1", ChannelID: 1, MerchantOrderId: "1", MerchantRefundId: "refund-1", PayPrice: 100, RefundPrice: 80, Status: consts.PayRefundStatusWaiting}
	if err := db.Create(refund).Error; err != nil {
		t.Fatal(err)
	}
	svc := &PayRefundService{q: q, channelSvc: NewPayChannelService(q, client.NewPayClientFactory()), orderSvc: &PayOrderService{q: q}, notifySvc: &PayNotifyService{q: q}}
	if err := db.Callback().Create().Before("gorm:create").Register("fail-refund-notify", func(tx *gorm.DB) {
		if tx.Statement.Table == "pay_notify_task" {
			tx.AddError(errors.New("injected notify failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	dto := &client.RefundResp{Status: consts.PayRefundStatusSuccess, OutTradeNo: "review-P1", OutRefundNo: "R-review"}
	if err := svc.NotifyRefund(ctx, 1, dto); err == nil {
		t.Fatal("notification failure ignored")
	}
	var order payModel.PayOrder
	if err := db.First(&order, 1).Error; err != nil {
		t.Fatal(err)
	}
	var saved payModel.PayRefund
	if err := db.First(&saved, 1).Error; err != nil {
		t.Fatal(err)
	}
	if order.RefundPrice != 0 || saved.Status != consts.PayRefundStatusWaiting {
		t.Fatalf("partial refund commit: total=%d status=%d", order.RefundPrice, saved.Status)
	}
	db.Callback().Create().Remove("fail-refund-notify")
	for range 2 {
		if err := svc.NotifyRefund(ctx, 1, dto); err != nil {
			t.Fatal(err)
		}
	}
	db.First(&order, 1)
	var tasks int64
	db.Model(&payModel.PayNotifyTask{}).Count(&tasks)
	if order.RefundPrice != 80 || tasks != 1 {
		t.Fatalf("replay total=%d tasks=%d", order.RefundPrice, tasks)
	}
}

type reviewRefundClient struct {
	client.PayClient
	calls atomic.Int32
}

func (c *reviewRefundClient) Init() error { return nil }
func (c *reviewRefundClient) UnifiedRefund(context.Context, *client.UnifiedRefundReq) (*client.RefundResp, error) {
	c.calls.Add(1)
	return nil, errors.New("synthetic remote timeout")
}
func TestReviewRefundReservationConcurrency(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 3})
	defer rdb.Close()
	db := reviewDB(t)
	q := query.Use(db)
	ctx, cancel := context.WithTimeout(db.Statement.Context, 10*time.Second)
	defer cancel()
	if err := db.Model(&payModel.PayOrder{}).Where("id=1").Updates(map[string]any{"status": consts.PayOrderStatusSuccess, "channel_id": 1, "no": "review-P1"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&payModel.PayChannel{}).Where("id=1").Updates(map[string]any{"code": "review_refund", "config": &payModel.PayClientConfig{ConfigType: payModel.ConfigTypeNone}}).Error; err != nil {
		t.Fatal(err)
	}
	fake := &reviewRefundClient{}
	client.RegisterCreator("review_refund", func(int64, string, string) (client.PayClient, error) { return fake, nil })
	factory := client.NewPayClientFactory()
	channel := NewPayChannelService(q, factory)
	svc := NewPayRefundService(q, NewPayAppService(q, channel), channel, &PayOrderService{q: q}, &PayNotifyService{q: q}, payrepo.NewPayNoRedisDAO(rdb))
	// Both requests complete the initial pending-refund read before either reserves funds.
	var initial atomic.Int32
	ready := make(chan struct{})
	if err := db.Callback().Query().After("gorm:query").Register("refund-race", func(tx *gorm.DB) {
		if tx.Statement.Table == "pay_refund" {
			n := initial.Add(1)
			if n == 2 {
				close(ready)
			}
			if n <= 2 {
				select {
				case <-ready:
				case <-ctx.Done():
					tx.AddError(ctx.Err())
				}
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CreateRefund(ctx, &payreq.PayRefundCreateReq{AppKey: "review-app", MerchantOrderId: "1", MerchantRefundId: fmt.Sprintf("refund-%d", i), Price: 80})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	passed := 0
	for err := range errs {
		if err == nil {
			passed++
		}
	}
	var rows []payModel.PayRefund
	if err := db.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if passed != 1 || len(rows) != 1 || rows[0].RefundPrice != 80 || rows[0].Status != consts.PayRefundStatusWaiting || fake.calls.Load() != 1 {
		t.Fatalf("over-reservation: accepted=%d rows=%+v remote=%d", passed, rows, fake.calls.Load())
	}
}

func TestReviewTransferNotificationSharesTransaction(t *testing.T) {
	db := reviewDB(t)
	q := query.Use(db)
	ctx := db.Statement.Context
	transfer := &payModel.PayTransfer{ID: 1, No: "T-review", AppID: 1, ChannelID: 1, Status: PayTransferStatusWaiting, Price: 100}
	if err := db.Create(transfer).Error; err != nil {
		t.Fatal(err)
	}
	svc := &PayTransferService{transferRepo: payrepo.NewPayTransferRepository(q), channelSvc: NewPayChannelService(q, client.NewPayClientFactory()), notifySvc: &PayNotifyService{q: q}, logger: zap.NewNop()}
	if err := db.Callback().Create().Before("gorm:create").Register("fail-transfer-notify", func(tx *gorm.DB) {
		if tx.Statement.Table == "pay_notify_task" {
			tx.AddError(errors.New("injected notify failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	dto := &client.TransferResp{Status: PayTransferStatusSuccess, OutTradeNo: "T-review"}
	if err := svc.NotifyTransfer(ctx, 1, dto); err == nil {
		t.Fatal("notification failure ignored")
	}
	var saved payModel.PayTransfer
	if err := db.First(&saved, 1).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != PayTransferStatusWaiting {
		t.Fatal("partial transfer commit")
	}
	db.Callback().Create().Remove("fail-transfer-notify")
	for range 2 {
		if err := svc.NotifyTransfer(ctx, 1, dto); err != nil {
			t.Fatal(err)
		}
	}
	var tasks int64
	db.Model(&payModel.PayNotifyTask{}).Count(&tasks)
	if tasks != 1 {
		t.Fatalf("duplicate transfer notifications: %d", tasks)
	}
}
