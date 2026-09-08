package system

import (
	"context"
	"github.com/redis/go-redis/v9"
	contract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOAuthTokenSecurity(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	oldRDB, secret := cache.RDB, config.C.Security.JWTSecret
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 1})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	cache.RDB = client
	config.C.Security.JWTSecret = "security-test-secret-01234567890123456789"
	t.Cleanup(func() { cache.RDB = oldRDB; config.C.Security.JWTSecret = secret; client.Close() })
	ctx := pkgContext.WithTenant(context.Background(), 1)
	svc := NewOAuth2TokenService()
	issue := func() *OAuth2AccessToken {
		t.Helper()
		record, err := svc.CreateAccessToken(ctx, 123, 1, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			client.Del(context.Background(), tokenKey(record.AccessToken, utils.TokenAccess), tokenKey(record.RefreshToken, utils.TokenRefresh))
		})
		return record
	}
	first := issue()
	if _, err := svc.GetAccessToken(ctx, first.RefreshToken); err == nil {
		t.Fatal("refresh accepted as access")
	}
	if _, err := svc.GetRefreshToken(ctx, first.AccessToken, 1); err == nil {
		t.Fatal("access accepted as refresh")
	}
	if _, err := svc.GetRefreshToken(ctx, first.RefreshToken, 2); err == nil {
		t.Fatal("member accepted as admin")
	}
	if _, err := svc.GetAccessToken(pkgContext.WithTenant(context.Background(), 2), first.AccessToken); err == nil {
		t.Fatal("other tenant accepted")
	}
	if _, err := svc.RefreshAccessToken(ctx, first.RefreshToken, 999, 1, 1, nil); err == nil {
		t.Fatal("identity replacement accepted")
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			next, err := svc.RefreshAccessToken(ctx, first.RefreshToken, 123, 1, 1, nil)
			if err == nil {
				wins.Add(1)
				client.Del(context.Background(), tokenKey(next.AccessToken, utils.TokenAccess), tokenKey(next.RefreshToken, utils.TokenRefresh))
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("refresh winners=%d, want 1", wins.Load())
	}
	if _, err := svc.GetAccessToken(ctx, first.AccessToken); err == nil {
		t.Fatal("old access survives refresh")
	}
	if _, err := svc.GetRefreshToken(ctx, first.RefreshToken, 1); err == nil {
		t.Fatal("consumed refresh survives")
	}
	second := issue()
	if _, err := svc.RemoveAccessToken(ctx, second.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetRefreshToken(ctx, second.RefreshToken, 1); err == nil {
		t.Fatal("logout leaves refresh active")
	}
	third := issue()
	if err := client.Set(ctx, tokenKey(third.AccessToken, utils.TokenAccess), `{"userId":999}`, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetAccessToken(ctx, third.AccessToken); err == nil {
		t.Fatal("corrupt whitelist accepted")
	}
	cache.RDB = nil
	if _, err := svc.CreateAccessToken(ctx, 123, 1, 1, nil); err == nil {
		t.Fatal("issued without Redis")
	}
	if _, err := svc.RemoveAccessToken(ctx, third.AccessToken); err == nil {
		t.Fatal("logout silently succeeded without Redis")
	}
	closed := redis.NewClient(&redis.Options{Addr: addr, DB: 1})
	closed.Close()
	cache.RDB = closed
	if _, err := svc.CreateAccessToken(ctx, 123, 1, 1, nil); err == nil {
		t.Fatal("issued through failed Redis")
	}
	cache.RDB = client
}

func TestLoginRejectsInvalidCaptchaBeforeDatabase(t *testing.T) {
	old := cache.RDB
	cache.RDB = nil
	t.Cleanup(func() { cache.RDB = old })
	ctx := pkgContext.WithTenant(context.Background(), 1)
	// No repository: a supplied invalid verification must be rejected before credentials are queried.
	svc := &AuthService{}
	if _, err := svc.Login(ctx, &contract.AuthLoginReq{Username: "admin", Password: "irrelevant", CaptchaVerification: "invalid---proof"}); err == nil {
		t.Fatal("invalid supplied captcha accepted")
	}
}
