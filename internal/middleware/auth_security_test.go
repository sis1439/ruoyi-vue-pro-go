package middleware

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAuthenticationBoundaries(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	oldRDB, oldSecret := cache.RDB, config.C.Security.JWTSecret
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 1})
	cache.RDB = client
	config.C.Security.JWTSecret = "middleware-secret-012345678901234567890"
	t.Cleanup(func() { cache.RDB = oldRDB; config.C.Security.JWTSecret = oldSecret; client.Close() })
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	member, _ := utils.GenerateTokenWithInfo(17, 1, 1, "", time.Hour)
	admin, _ := utils.GenerateTokenWithInfo(17, 2, 1, "", time.Hour)
	refresh, _ := utils.GenerateTypedToken(17, 1, 1, "", utils.TokenRefresh, time.Hour)
	for raw, kind := range map[string]int{member: 1, admin: 2, refresh: 1} {
		data, _ := json.Marshal(OAuth2AccessToken{AccessToken: raw, UserID: 17, UserType: kind, TenantID: 1})
		key := "oauth2_access_token:" + raw
		if err := client.Set(context.Background(), key, data, time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { client.Del(context.Background(), key) })
	}
	call := func(path, token, tenant, visit string, optional bool) int {
		r := gin.New()
		r.Use(InjectContext())
		if optional {
			r.Use(OptionalAuth())
		} else {
			r.Use(Auth())
		}
		r.GET(path, func(c *gin.Context) {
			if token != "" {
				if id, ok := pkgContext.TenantID(c.Request.Context()); !ok || id != 1 {
					t.Error("trusted identity not propagated")
				}
			}
			c.Status(204)
		})
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("tenant-id", tenant)
		req.Header.Set("visit-tenant-id", visit)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	for _, tc := range []struct {
		name, path, token, tenant, visit string
		optional                         bool
		want                             int
	}{
		{"member", "/app-api/test", member, "1", "", false, 204},
		{"admin", "/admin-api/test", admin, "1", "", false, 204},
		{"member in admin", "/admin-api/test", member, "1", "", false, 403},
		{"admin in member", "/app-api/test", admin, "1", "", false, 403},
		{"tenant spoof", "/app-api/test", member, "2", "", false, 403},
		{"visit spoof", "/app-api/test", member, "1", "2", false, 403},
		{"refresh", "/app-api/test", refresh, "1", "", false, 401},
		{"anonymous optional", "/app-api/test", "", "", "", true, 204},
		{"bad optional", "/app-api/test", "invalid", "", "", true, 401},
		{"required missing", "/app-api/test", "", "", "", false, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := call(tc.path, tc.token, tc.tenant, tc.visit, tc.optional); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
	client.Del(context.Background(), "oauth2_access_token:"+member)
	if got := call("/app-api/test", member, "1", "", false); got != 401 {
		t.Fatalf("revoked accepted %d", got)
	}
	cache.RDB = nil
	if got := call("/admin-api/test", admin, "1", "", false); got != 503 {
		t.Fatalf("nil Redis accepted %d", got)
	}
	if got := call("/admin-api/test", admin, "1", "", true); got != 503 {
		t.Fatalf("optional nil Redis accepted %d", got)
	}
	cache.RDB = redis.NewClient(&redis.Options{Addr: addr})
	cache.RDB.Close()
	if got := call("/admin-api/test", admin, "1", "", false); got != 401 {
		t.Fatalf("failed Redis accepted %d", got)
	}
	cache.RDB = client
}
