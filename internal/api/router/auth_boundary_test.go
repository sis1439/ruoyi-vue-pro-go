package router

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	infraHandlers "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/infra"
	tradeHandlers "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/mall/trade"
	systemHandlers "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/system"
	appHandlers "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app"
	"github.com/wxlbd/ruoyi-mall-go/internal/middleware"
	permissionPkg "github.com/wxlbd/ruoyi-mall-go/internal/pkg/permission"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	systemSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
)

// Allocate only exported handler containers. Services remain nil so a missing auth guard fails loudly.
func allocateHandlers(v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanSet() && f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.Struct {
			allocateHandlers(f)
		}
	}
}
func TestPrivateRoutesRejectAnonymous(t *testing.T) {
	var app *appHandlers.AppHandlers
	allocateHandlers(reflect.ValueOf(&app).Elem())
	var system *systemHandlers.Handlers
	allocateHandlers(reflect.ValueOf(&system).Elem())
	var infra *infraHandlers.Handlers
	allocateHandlers(reflect.ValueOf(&infra).Elem())
	router := gin.New()
	RegisterAppRoutes(router, app)
	RegisterSystemRoutes(router, system, infra, middleware.NewCasbinMiddleware(nil, nil))
	paths := []struct{ method, path string }{
		{"GET", "/app-api/member/sign-in/record/get-summary"}, {"GET", "/app-api/member/sign-in/record/page"}, {"POST", "/app-api/member/sign-in/record/create"},
		{"POST", "/app-api/promotion/kefu-message/send"}, {"PUT", "/app-api/promotion/kefu-message/update-read-status"}, {"GET", "/app-api/promotion/kefu-message/list"},
		{"GET", "/app-api/pay/order/get?id=1"}, {"POST", "/app-api/pay/order/submit"}, {"GET", "/app-api/pay/wallet/get"}, {"GET", "/app-api/pay/wallet-transaction/page"}, {"GET", "/app-api/pay/wallet-transaction/get-summary"}, {"POST", "/app-api/pay/wallet-recharge/create"}, {"GET", "/app-api/pay/wallet-recharge/page"}, {"GET", "/app-api/pay/transfer/sync?id=1"},
		{"GET", "/admin-api/system/user/simple-list"}, {"GET", "/admin-api/system/user/list-all-simple"}, {"GET", "/admin-api/system/dept/list"}, {"GET", "/admin-api/system/dept/list-all-simple"}, {"GET", "/admin-api/system/dept/simple-list"}, {"GET", "/admin-api/system/role/simple-list"}, {"GET", "/admin-api/system/post/simple-list"}, {"GET", "/admin-api/system/menu/simple-list"}, {"GET", "/admin-api/system/sms-channel/simple-list"},
	}
	for _, tc := range paths {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != 401 {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestTradeRoutesRequireLivePermission(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	perms := systemSvc.NewPermissionService(q, systemSvc.NewRoleService(q))
	enforcer, err := permissionPkg.InitEnforcer(db)
	if err != nil {
		t.Fatal(err)
	}
	old, secret := cache.RDB, config.C.Security.JWTSecret
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 1})
	cache.RDB = rdb
	config.C.Security.JWTSecret = "trade-rbac-fixture-secret-01234567890123456789"
	t.Cleanup(func() { cache.RDB = old; config.C.Security.JWTSecret = secret; rdb.Close() })
	token, err := utils.GenerateTokenWithInfo(880001, 2, 1, "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(middleware.OAuth2AccessToken{AccessToken: token, UserID: 880001, UserType: 2, TenantID: 1})
	key := "oauth2_access_token:" + token
	if err := rdb.Set(context.Background(), key, data, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), key) })
	var handlers *tradeHandlers.Handlers
	allocateHandlers(reflect.ValueOf(&handlers).Elem())
	router := gin.New()
	router.Use(middleware.InjectContext())
	RegisterTradeRoutes(router, handlers, middleware.NewCasbinMiddleware(enforcer, perms))
	checked := 0
	for _, route := range router.Routes() {
		if route.Path == "/admin-api/trade/after-sale/update-refunded" {
			continue
		} // provider callback has a separate contract
		checked++
		t.Run(route.Method+route.Path, func(t *testing.T) {
			req := httptest.NewRequest(route.Method, route.Path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != 403 {
				t.Fatalf("role without permission admitted: %d %s", w.Code, w.Body.String())
			}
		})
	}
	if checked < 40 {
		t.Fatalf("unexpected trade route coverage: %d", checked)
	}
}
