package middleware

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoginRateLimitRealRedis(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("set T09_REDIS_ADDR for real Redis integration")
	}
	r := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1})
	defer r.Close()
	if err := r.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	tenant := time.Now().UnixNano()
	engine := gin.New()
	engine.Use(LoginRateLimit(r))
	engine.NoRoute(func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(413)
			return
		}
		c.Status(204)
	})
	request := func(path, peer string, id int64, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.RemoteAddr = peer
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", time.Now().Nanosecond()%250))
		req.Header.Set("X-Real-IP", "203.0.113.1")
		req = req.WithContext(pkgContext.WithTenant(req.Context(), id))
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}
	var wg sync.WaitGroup
	var allowed, denied atomic.Int32
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := []string{"/admin-api/system/auth/login", "/app-api/member/auth/sms-login", "/app-api/member/user/reset-password", "/app-api/member/auth/send-sms-code"}[i%4]
			w := request(path, "192.0.2.1:1234", tenant, "")
			switch w.Code {
			case 204:
				allowed.Add(1)
			case 429:
				denied.Add(1)
				if w.Header().Get("Retry-After") == "" {
					t.Error("missing retry header")
				}
			default:
				t.Errorf("unexpected %d: %s", w.Code, w.Body)
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 10 || denied.Load() != 30 {
		t.Fatalf("allowed=%d denied=%d", allowed.Load(), denied.Load())
	}
	if w := request("/admin-api/system/auth/login", "192.0.2.1:99", tenant+1, ""); w.Code != 204 {
		t.Fatal("tenant budgets not isolated")
	}
	if w := request("/admin-api/system/auth/login", "192.0.2.2:99", tenant, ""); w.Code != 204 {
		t.Fatal("IP budgets not isolated")
	}
	for i := 0; i < 31; i++ {
		want := 204
		if i == 30 {
			want = 429
		}
		if w := request("/admin-api/system/captcha/get", "192.0.2.1:1", tenant, ""); w.Code != want {
			t.Fatalf("captcha %d status %d", i, w.Code)
		}
	}
	if w := request("/admin-api/system/auth/login", "192.0.2.2:99", 0, ""); w.Code != 403 {
		t.Fatal("tenantless request accepted")
	}
	if w := request("/admin-api/system/auth/login", "invalid", tenant, ""); w.Code != 400 {
		t.Fatal("invalid peer accepted")
	}
	if w := request("/admin-api/system/auth/login", "192.0.2.2:99", tenant, strings.Repeat("a", 16385)); w.Code != 413 {
		t.Fatal("oversized body accepted")
	}
	key := fmt.Sprintf("security:login-rate:%d:192.0.2.1:auth", tenant)
	ttl := r.PTTL(context.Background(), key).Val()
	if ttl <= 0 || ttl > time.Minute {
		t.Fatalf("unbounded budget TTL %v", ttl)
	}
	r.PExpire(context.Background(), key, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if w := request("/admin-api/system/auth/login", "192.0.2.1:1", tenant, ""); w.Code != 204 {
		t.Fatal("budget did not expire")
	}
	req := httptest.NewRequest("POST", "/admin-api/system/auth/login", strings.NewReader(strings.Repeat("a", 16385)))
	req.ContentLength = -1
	req = req.WithContext(pkgContext.WithTenant(req.Context(), tenant+2))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != 413 {
		t.Fatalf("chunked oversized body: %d", w.Code)
	}
	r.Close()
	if w := request("/admin-api/system/auth/login", "192.0.2.1:1", tenant, ""); w.Code != 503 {
		t.Fatal("Redis outage accepted")
	}
	if w := request("/public/health", "192.0.2.1:1", tenant, ""); w.Code != 204 {
		t.Fatal("unrelated route blocked")
	}
}

func TestLoginRateLimitNilRedis(t *testing.T) {
	engine := gin.New()
	engine.Use(LoginRateLimit(nil))
	engine.POST("/admin-api/system/auth/login", func(c *gin.Context) { t.Error("handler reached") })
	req := httptest.NewRequest("POST", "/admin-api/system/auth/login", nil)
	req = req.WithContext(pkgContext.WithTenant(req.Context(), 1))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != 503 {
		t.Fatalf("nil Redis status %d", w.Code)
	}
}
