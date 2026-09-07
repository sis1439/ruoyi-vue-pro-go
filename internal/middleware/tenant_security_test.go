package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	systemHandler "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	systemSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestTrustedTenantHostBoundary(t *testing.T) {
	db := testutil.PostgreSQL(t)
	tenants := []model.SystemTenant{
		{ID: 820001, Name: "Security A", Websites: model.StringListFromCSV{"a.example.test"}, Status: 0, ExpireDate: time.Now().Add(time.Hour)},
		{ID: 820002, Name: "Security B", Websites: model.StringListFromCSV{"b.example.test"}, Status: 0, ExpireDate: time.Now().Add(time.Hour)},
		{ID: 820003, Name: "Security Expired", Websites: model.StringListFromCSV{"expired.example.test"}, Status: 0, ExpireDate: time.Now().Add(-time.Hour)},
		{ID: 820004, Name: "Security Disabled", Websites: model.StringListFromCSV{"disabled.example.test"}, Status: 1, ExpireDate: time.Now().Add(time.Hour)},
	}
	if err := db.Create(&tenants).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		host, tenant, visit string
		want                int
	}{
		{"a.example.test", "820001", "", 204}, {"a.example.test", "", "", 204}, {"a.example.test:443", "820001", "", 204},
		{"a.example.test", "820002", "", 403}, {"a.example.test", "820001", "820002", 403}, {"unknown.test", "820001", "", 403},
		{"expired.example.test", "820003", "", 403}, {"disabled.example.test", "820004", "", 403},
	} {
		t.Run(tc.host+tc.tenant+tc.visit, func(t *testing.T) {
			r := gin.New()
			r.Use(InjectContext(), TrustedTenant(db))
			r.GET("/app-api/test", func(c *gin.Context) {
				for _, ctx := range []interface{ Value(any) any }{c, c.Request.Context()} {
					id, ok := ctx.Value(pkgContext.CtxTenantIDKey).(int64)
					if !ok || id != 820001 {
						t.Fatalf("trusted tenant not propagated: %v", id)
					}
				}
				c.Status(204)
			})
			req := httptest.NewRequest("GET", "/app-api/test", nil)
			req.Host = tc.host
			req.Header.Set("tenant-id", tc.tenant)
			req.Header.Set("visit-tenant-id", tc.visit)
			req.Header.Set("X-Forwarded-Host", "a.example.test")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestAuthenticatedDisabledUserRejected(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	db := testutil.PostgreSQL(t)
	tenant := model.SystemTenant{ID: 840001, Name: "Security auth tenant", Websites: model.StringListFromCSV{"auth.example.test"}, Status: 0, ExpireDate: time.Now().Add(time.Hour)}
	user := model.SystemUser{ID: 840001, Username: "security-auth-user", Password: "unused", Status: 0, TenantBaseDO: model.TenantBaseDO{TenantID: tenant.ID}}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	oldRDB, secret := cache.RDB, config.C.Security.JWTSecret
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 1})
	cache.RDB = rdb
	config.C.Security.JWTSecret = "security-auth-test-secret-01234567890123456789"
	t.Cleanup(func() { cache.RDB = oldRDB; config.C.Security.JWTSecret = secret; rdb.Close() })
	token, err := utils.GenerateTokenWithInfo(user.ID, 2, tenant.ID, "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(OAuth2AccessToken{AccessToken: token, UserID: user.ID, UserType: 2, TenantID: tenant.ID})
	key := "oauth2_access_token:" + token
	if err = rdb.Set(context.Background(), key, data, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), key) })
	router := gin.New()
	router.Use(InjectContext(), TrustedTenant(db), Auth())
	router.GET("/admin-api/security", func(c *gin.Context) { c.Status(204) })
	call := func() int {
		req := httptest.NewRequest("GET", "/admin-api/security", nil)
		req.Host = "auth.example.test"
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}
	if got := call(); got != 204 {
		t.Fatalf("active user got %d", got)
	}
	scopeCtx := pkgContext.WithTenant(context.Background(), tenant.ID)
	for i, scope := range []int32{1, 2} {
		role := model.SystemRole{ID: 840001 + int64(i), Name: "scope", Code: fmt.Sprintf("scope-%d", i), Status: 0, Type: 2, DataScope: scope}
		if err := db.WithContext(scopeCtx).Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.WithContext(scopeCtx).Create(&model.SystemUserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if got := call(); got != 403 {
		t.Fatalf("mixed All+restricted scopes admitted: %d", got)
	}
	if err := db.WithContext(scopeCtx).Model(&model.SystemRole{}).Where("id = ?", 840002).Update("data_scope", 1).Error; err != nil {
		t.Fatal(err)
	}
	if got := call(); got != 204 {
		t.Fatalf("tenant-wide All scope rejected: %d", got)
	}
	if err := db.WithContext(pkgContext.WithTenant(context.Background(), tenant.ID)).Model(&model.SystemUser{}).Where("id = ?", user.ID).Update("status", 1).Error; err != nil {
		t.Fatal(err)
	}
	if got := call(); got != 401 {
		t.Fatalf("disabled user retained token access: %d", got)
	}
}

func TestRefreshIgnoresExpiredAccessHeader(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	db := testutil.PostgreSQL(t)
	user := model.SystemUser{ID: 870001, Username: "security-refresh-admin", Password: "unused", Status: 0, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	old, secret := cache.RDB, config.C.Security.JWTSecret
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 1})
	cache.RDB = rdb
	config.C.Security.JWTSecret = "refresh-header-test-01234567890123456789"
	t.Cleanup(func() { cache.RDB = old; config.C.Security.JWTSecret = secret; rdb.Close() })
	tokens := systemSvc.NewOAuth2TokenService()
	pair, err := tokens.CreateAccessToken(pkgContext.WithTenant(context.Background(), 1), user.ID, 2, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		rdb.Del(context.Background(), "oauth2_access_token:"+pair.AccessToken, "oauth2_refresh_token:"+pair.RefreshToken)
	})
	claims, err := utils.ParseToken(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.C.Security.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	handler := systemHandler.NewAuthHandler(systemSvc.NewAuthService(query.Use(db), nil, nil, nil, tokens, nil, nil, nil, nil))
	router := gin.New()
	router.Use(InjectContext(), TrustedTenant(db))
	router.POST("/admin-api/system/auth/refresh-token", handler.RefreshToken)
	call := func(tenant string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/admin-api/system/auth/refresh-token?refreshToken="+pair.RefreshToken, nil)
		req.Header.Set("Authorization", "Bearer "+expired)
		req.Header.Set("tenant-id", tenant)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := call("2"); w.Code != 403 {
		t.Fatalf("mismatched refresh tenant: %d %s", w.Code, w.Body.String())
	}
	w := call("1")
	var result struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result.Code != 0 || result.Data.AccessToken == "" {
		t.Fatalf("expired access prevented refresh: %d %s %v", w.Code, w.Body.String(), err)
	}
	t.Cleanup(func() {
		rdb.Del(context.Background(), "oauth2_access_token:"+result.Data.AccessToken, "oauth2_refresh_token:"+result.Data.RefreshToken)
	})
	if w := call("1"); w.Code != 401 {
		t.Fatalf("refresh replay admitted %d", w.Code)
	}
}
