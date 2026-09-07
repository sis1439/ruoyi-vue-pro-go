package system

import (
	"context"
	"errors"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"gorm.io/gorm"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSmsCodeTenantReplayAndFailures(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	r := captchaRedis(t)
	tenant := time.Now().UnixNano()
	ctx := pkgContext.WithTenant(context.Background(), tenant)
	s := NewSmsCodeService(query.Use(db), r, nil)
	var calls atomic.Int32
	var code string
	s.send = func(_ context.Context, _ string, _ int64, _ string, params map[string]any) (int64, error) {
		calls.Add(1)
		code = params["code"].(string)
		return 1, nil
	}
	phone := "13800009001"
	if err := s.SendSmsCode(ctx, phone, 1, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("not six digits")
	}
	if err := s.ValidateSmsCode(ctx, phone, 1, code); err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateSmsCode(pkgContext.WithTenant(ctx, tenant+1), phone, 1, code); err == nil {
		t.Fatal("cross-tenant accepted")
	}
	if err := s.ValidateSmsCode(ctx, phone, 4, code); err == nil {
		t.Fatal("cross-scene accepted")
	}
	if err := s.SendSmsCode(ctx, phone, 4, ""); !errors.Is(err, ErrSmsCodeSendTooFast) {
		t.Fatalf("scene bypass: %v", err)
	}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.UseSmsCode(ctx, phone, 1, code, "") == nil {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("SMS replay accepted %d", accepted.Load())
	}
	var record model.SystemSmsCode
	if err := db.WithContext(ctx).Where("mobile = ?", phone).First(&record).Error; err != nil || !record.Used {
		t.Fatalf("audit state %v %v", record.Used, err)
	}
	if record.Code != "[OTP]" {
		t.Fatal("audit row contains OTP instead of marker")
	}
	// Failed delivery never activates a code.
	s.send = func(_ context.Context, _ string, _ int64, _ string, params map[string]any) (int64, error) {
		calls.Add(1)
		code = params["code"].(string)
		return 0, fmt.Errorf("delivery failed")
	}
	if err := s.SendSmsCode(ctx, "13800009002", 1, ""); err == nil {
		t.Fatal("delivery failure swallowed")
	}
	if err := s.ValidateSmsCode(ctx, "13800009002", 1, code); err == nil {
		t.Fatal("failed delivery code valid")
	}
	// A database create error must prevent the sender from being called.
	before := calls.Load()
	db.Callback().Create().Before("gorm:create").Register("test:sms_create_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "system_sms_code" {
			tx.AddError(fmt.Errorf("database write failed"))
		}
	})
	if err := s.SendSmsCode(ctx, "13800009003", 1, ""); err == nil {
		t.Fatal("database failure swallowed")
	}
	if calls.Load() != before {
		t.Fatal("sent after database failure")
	}
	db.Callback().Create().Remove("test:sms_create_failure")
	// Database audit failure rejects authentication and still burns the code.
	s.send = func(_ context.Context, _ string, _ int64, _ string, params map[string]any) (int64, error) {
		code = params["code"].(string)
		return 1, nil
	}
	if err := s.SendSmsCode(ctx, "13800009004", 1, ""); err != nil {
		t.Fatal(err)
	}
	db.Callback().Update().Before("gorm:update").Register("test:sms_update_failure", func(tx *gorm.DB) { tx.AddError(fmt.Errorf("database audit failed")) })
	if err := s.UseSmsCode(ctx, "13800009004", 1, code, ""); err == nil {
		t.Fatal("audit failure accepted")
	}
	db.Callback().Update().Remove("test:sms_update_failure")
	if err := s.UseSmsCode(ctx, "13800009004", 1, code, ""); err == nil {
		t.Fatal("audit-failed code replay")
	}
	// Real database unavailable, rather than only an injected statement error.
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	before = calls.Load()
	if err := s.SendSmsCode(ctx, "13800009005", 1, ""); err == nil {
		t.Fatal("database outage accepted")
	}
	if calls.Load() != before {
		t.Fatal("sent with database down")
	}
	r.Close()
	if err := s.SendSmsCode(ctx, "13800009006", 1, ""); err == nil {
		t.Fatal("Redis outage send accepted")
	}
	if err := s.UseSmsCode(ctx, phone, 1, code, ""); err == nil {
		t.Fatal("Redis outage auth accepted")
	}
	if err := s.SendSmsCode(context.Background(), phone, 1, ""); err == nil {
		t.Fatal("tenantless SMS")
	}
}

func TestSmsCodeConcurrentBudget(t *testing.T) {
	db := testutil.PostgreSQL(t)
	r := captchaRedis(t)
	tenant := time.Now().UnixNano()
	ctx := pkgContext.WithTenant(context.Background(), tenant)
	s := NewSmsCodeService(query.Use(db), r, nil)
	var sends atomic.Int32
	s.send = func(context.Context, string, int64, string, map[string]any) (int64, error) {
		sends.Add(1)
		return 1, nil
	}
	var wg sync.WaitGroup
	var success atomic.Int32
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.SendSmsCode(ctx, "13800009007", 1, "") == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if sends.Load() != 1 || success.Load() != 1 {
		t.Fatalf("sends=%d successes=%d", sends.Load(), success.Load())
	}
	rate := fmt.Sprintf("%s{%d:%s}", SmsCodeRateLimitPrefix, tenant, "13800009007")
	for i := 1; i < 10; i++ {
		r.Del(ctx, rate)
		if err := s.SendSmsCode(ctx, "13800009007", 1, ""); err != nil {
			t.Fatal(err)
		}
	}
	r.Del(ctx, rate)
	if err := s.SendSmsCode(ctx, "13800009007", 4, ""); !errors.Is(err, ErrSmsCodeExceedMaxPerDay) {
		t.Fatalf("daily budget bypass: %v", err)
	}
	if sends.Load() != 10 {
		t.Fatalf("sent %d", sends.Load())
	}
}

func TestSmsCodeDisabledAndDebugDelivery(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	r := captchaRedis(t)
	ctx := pkgContext.WithTenant(context.Background(), time.Now().UnixNano())
	q := query.Use(db)
	// Nil factory ensures any accidental attempt to reach a provider fails the test.
	sender := NewSmsSendService(q, NewSmsTemplateService(q), NewSmsLogService(q), nil)
	s := NewSmsCodeService(q, r, sender)
	channel := &model.SystemSmsChannel{Code: "DEBUG", Status: 0}
	if err := db.WithContext(ctx).Create(channel).Error; err != nil {
		t.Fatal(err)
	}
	template := &model.SystemSmsTemplate{Code: GetSceneEnum(1).TemplateCode, ChannelId: channel.ID, ChannelCode: "DEBUG", Status: 1}
	if err := db.WithContext(ctx).Create(template).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.SendSmsCode(ctx, "13800009009", 1, ""); err == nil {
		t.Fatal("disabled template accepted")
	}
	if err := db.WithContext(ctx).Model(template).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.SendSmsCode(ctx, "13800009010", 1, ""); err == nil {
		t.Fatal("debug channel accepted")
	}
	for _, phone := range []string{"13800009009", "13800009010"} {
		key, _ := s.getCacheKey(ctx, phone, 1)
		if n := r.Exists(ctx, key).Val(); n != 0 {
			t.Fatal("unconfirmed code activated")
		}
	}
	var count int64
	if err := db.WithContext(ctx).Model(&model.SystemSmsLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("sender reached")
	}
}

func TestSmsCodeIncorrectExpiredAndActivationFailure(t *testing.T) {
	db := testutil.PostgreSQL(t)
	r := captchaRedis(t)
	ctx := pkgContext.WithTenant(context.Background(), time.Now().UnixNano())
	s := NewSmsCodeService(query.Use(db), r, nil)
	var code string
	s.send = func(_ context.Context, _ string, _ int64, _ string, params map[string]any) (int64, error) {
		code = params["code"].(string)
		return 1, nil
	}
	if err := s.SendSmsCode(ctx, "13800009011", 1, ""); err != nil {
		t.Fatal(err)
	}
	wrong := "000000"
	if wrong == code {
		wrong = "000001"
	}
	if err := s.UseSmsCode(ctx, "13800009011", 1, wrong, ""); err == nil {
		t.Fatal("wrong code accepted")
	}
	if err := s.UseSmsCode(ctx, "13800009011", 1, code, ""); err == nil {
		t.Fatal("wrong attempt not consumed")
	}
	if err := s.SendSmsCode(ctx, "13800009012", 1, ""); err != nil {
		t.Fatal(err)
	}
	key, _ := s.getCacheKey(ctx, "13800009012", 1)
	r.PExpire(ctx, key, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if err := s.UseSmsCode(ctx, "13800009012", 1, code, ""); err == nil {
		t.Fatal("expired code accepted")
	}
	// Dependency fails after successful delivery: return failure and leave no usable code.
	s.send = func(_ context.Context, _ string, _ int64, _ string, params map[string]any) (int64, error) {
		code = params["code"].(string)
		r.Close()
		return 1, nil
	}
	if err := s.SendSmsCode(ctx, "13800009013", 1, ""); err == nil {
		t.Fatal("activation failure swallowed")
	}
	other := captchaRedis(t)
	key, _ = s.getCacheKey(ctx, "13800009013", 1)
	if n := other.Exists(ctx, key).Val(); n != 0 {
		t.Fatal("code active after activation failure")
	}
}
