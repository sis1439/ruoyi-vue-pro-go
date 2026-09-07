package system

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func captchaRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("set T09_REDIS_ADDR for real Redis integration")
	}
	r := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, ReadTimeout: time.Second, WriteTimeout: time.Second})
	t.Cleanup(func() { r.Close() })
	if err := r.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestCaptchaTenantAndAtomicReplay(t *testing.T) {
	r := captchaRedis(t)
	s := NewCaptchaService(r)
	ctx := pkgContext.WithTenant(context.Background(), 901)
	generated, err := s.Generate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	data, err := r.Get(ctx, fmt.Sprintf("%s%d:%s", CaptchaKeyPrefix, 901, generated.Token)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var answer CaptchaData
	if err = json.Unmarshal(data, &answer); err != nil {
		t.Fatal(err)
	}
	other := pkgContext.WithTenant(context.Background(), 902)
	if ok, _ := s.Verify(other, generated.Token, answer.X, 5); ok {
		t.Fatal("cross-tenant challenge accepted")
	}
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, e := s.Verify(ctx, generated.Token, answer.X, 5)
			if e != nil {
				t.Error(e)
			}
			if ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted %d replays", accepted.Load())
	}
	if _, err = s.Generate(context.Background()); err == nil {
		t.Fatal("tenantless generation accepted")
	}
}

func TestCaptchaProofAndOutage(t *testing.T) {
	r := captchaRedis(t)
	s := NewCaptchaService(r)
	ctx := pkgContext.WithTenant(context.Background(), 903)
	seed := func() (string, string) {
		token := uuid.NewString()
		key, _ := captchaKey(ctx, CaptchaKeyPrefix, token)
		if err := r.Set(ctx, key, `{"x":123,"y":20}`, time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { r.Del(ctx, key, "captcha:verified:903:"+token) })
		return token, `{"x":123.25,"y":5}`
	}
	token, point := seed()
	verification := token + "---" + point
	if err := s.ConsumeVerification(ctx, verification); err == nil {
		t.Fatal("unchecked proof accepted")
	}
	if ok, err := s.Check(ctx, token, point); err != nil || !ok {
		t.Fatalf("check: %v %v", ok, err)
	}
	if err := s.ConsumeVerification(pkgContext.WithTenant(ctx, 904), verification); err == nil {
		t.Fatal("cross-tenant proof")
	}
	var wg sync.WaitGroup
	var accepted atomic.Int32
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.ConsumeVerification(ctx, verification) == nil {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("proof accepted %d times", accepted.Load())
	}
	token, point = seed()
	if ok, _ := s.Check(ctx, token, `{"x":1,"y":5}`); ok {
		t.Fatal("wrong answer")
	}
	if ok, _ := s.Check(ctx, token, point); ok {
		t.Fatal("wrong attempt was not consumed")
	}
	token, point = seed()
	if ok, err := s.Check(ctx, token, point); !ok || err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeVerification(ctx, token+"---"+`{"x":124,"y":5}`); err == nil {
		t.Fatal("altered proof")
	}
	if err := s.ConsumeVerification(ctx, token+"---"+point); err == nil {
		t.Fatal("altered attempt did not burn proof")
	}
	token, point = seed()
	if ok, err := s.Check(ctx, token, point); !ok || err != nil {
		t.Fatal(err)
	}
	key, _ := captchaKey(ctx, "captcha:verified:", token)
	r.PExpire(ctx, key, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if err := s.ConsumeVerification(ctx, token+"---"+point); err == nil {
		t.Fatal("expired proof")
	}
	// Closing this real Redis connection simulates dependency loss without stopping shared Redis.
	r.Close()
	if _, err := s.Generate(ctx); err == nil {
		t.Fatal("outage generation accepted")
	}
	if _, err := s.Check(ctx, token, point); err == nil {
		t.Fatal("outage check accepted")
	}
	if err := s.ConsumeVerification(ctx, token+"---"+point); err == nil {
		t.Fatal("outage proof accepted")
	}
	nilSvc := NewCaptchaService(nil)
	if _, err := nilSvc.Generate(ctx); err == nil {
		t.Fatal("nil cache generation")
	}
}

func TestSecurityActualRedisServerOutage(t *testing.T) {
	executable, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server executable required")
	}
	dir, err := os.MkdirTemp("/private/tmp", "t09-redis-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "redis.sock")
	cmd := exec.Command(executable, "--port", "0", "--save", "", "--appendonly", "no", "--unixsocket", socket, "--unixsocketperm", "700")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	r := redis.NewClient(&redis.Options{Network: "unix", Addr: socket, MaxRetries: -1, DialTimeout: 100 * time.Millisecond, ReadTimeout: 100 * time.Millisecond, WriteTimeout: 100 * time.Millisecond})
	defer r.Close()
	ctx := pkgContext.WithTenant(context.Background(), 905)
	deadline := time.Now().Add(3 * time.Second)
	for r.Ping(ctx).Err() != nil {
		if time.Now().After(deadline) {
			t.Fatal("private Redis did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	svc := NewCaptchaService(r)
	token := uuid.NewString()
	key, _ := captchaKey(ctx, CaptchaKeyPrefix, token)
	if err := r.Set(ctx, key, `{"x":100,"y":5}`, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	point := `{"x":100,"y":5}`
	if valid, err := svc.Check(ctx, token, point); err != nil || !valid {
		t.Fatal(err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	cmd.Wait()
	stopped = true
	started := time.Now()
	if err := svc.ConsumeVerification(ctx, token+"---"+point); err == nil {
		t.Fatal("dead Redis accepted proof")
	}
	if _, err := svc.Generate(ctx); err == nil {
		t.Fatal("dead Redis generated challenge")
	}
	sms := NewSmsCodeService(nil, r, nil)
	if err := sms.UseSmsCode(ctx, "13800009008", 1, "123456", ""); err == nil {
		t.Fatal("dead Redis accepted SMS")
	}
	if time.Since(started) > 4*time.Second {
		t.Fatal("outage request was not bounded")
	}
}
