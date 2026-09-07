package pay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	tradeModel "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	paySvc "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"net/http/httptest"
	"testing"
)

func TestMemberPayOrderOwnership(t *testing.T) {
	db := testutil.PostgreSQL(t)
	owned := payModel.PayOrder{ID: 850001, AppID: 1, MerchantOrderId: "security-pay-owner", Subject: "private", Status: 0, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	business := tradeModel.TradeOrder{ID: 850001, No: "security-trade-owner", UserID: 850001, PayOrderID: &owned.ID, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&business).Error; err != nil {
		t.Fatal(err)
	}
	wallet := payModel.PayWallet{ID: 850001, UserID: 850001, UserType: 1, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	recharge := payModel.PayWalletRecharge{ID: 850001, WalletID: wallet.ID, PayOrderID: 850002, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	if err := db.Create(&wallet).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	svc := paySvc.NewPayOrderService(query.Use(db), nil, nil, nil, nil, nil)
	handler := NewAppPayOrderHandler(svc, nil)
	// All channel/SDK dependencies are nil; unauthorized calls must stop before any use.
	request := func(userID, tenantID int64, method, path, body string) map[string]any {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			pkgContext.SetLoginUser(c, &pkgContext.LoginUser{UserID: userID, UserType: 1, TenantID: tenantID})
			c.Next()
		})
		router.GET("/order/get", handler.GetOrder)
		router.POST("/order/submit", handler.Submit)
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("invalid response %s", w.Body.String())
		}
		return result
	}
	own := request(850001, 1, "GET", "/order/get?id=850001", "")
	if own["code"] != float64(0) {
		t.Fatalf("owner rejected: %v", own)
	}
	for _, tc := range []struct{ user, tenant int64 }{{850002, 1}, {850001, 2}} {
		for _, sync := range []string{"false", "true"} {
			r := request(tc.user, tc.tenant, "GET", fmt.Sprintf("/order/get?id=850001&sync=%s", sync), "")
			if r["code"] != float64(403) {
				t.Fatalf("other member or tenant GET admitted: %v", r)
			}
		}
		r := request(tc.user, tc.tenant, "POST", "/order/submit", `{"id":850001,"channelCode":"wx_lite"}`)
		if r["code"] != float64(403) {
			t.Fatalf("other member or tenant submit admitted: %v", r)
		}
	}
	for _, tc := range []struct {
		user int64
		want bool
	}{{850001, true}, {850002, false}} {
		ctx := pkgContext.WithLoginUser(t.Context(), &pkgContext.LoginUser{UserID: tc.user, UserType: 1, TenantID: 1})
		err := svc.ValidateMemberOrderOwner(ctx, 850002)
		if (err == nil) != tc.want {
			t.Fatalf("wallet ownership user=%d err=%v", tc.user, err)
		}
	}
	var untouched payModel.PayOrder
	if err := db.WithContext(pkgContext.WithTenant(t.Context(), 1)).First(&untouched, owned.ID).Error; err != nil || untouched.Status != 0 {
		t.Fatalf("unauthorized request changed order: %v %v", untouched, err)
	}
	var extensions int64
	if err := db.WithContext(pkgContext.WithTenant(t.Context(), 1)).Model(&payModel.PayOrderExtension{}).Count(&extensions).Error; err != nil || extensions != 0 {
		t.Fatalf("unauthorized submit created extension: %d %v", extensions, err)
	}
}
func TestMemberTransferSyncDisabled(t *testing.T) {
	router := gin.New()
	router.GET("/sync", NewAppPayTransferHandler(nil).SyncTransfer)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/sync?id=1", nil))
	if w.Code != 403 {
		t.Fatalf("unsupported transfer reached service: %d", w.Code)
	}
}
