package system

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	svc "github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"github.com/xuri/excelize/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSmsLogPageAndExportRedactLegacySecrets(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	tenant := time.Now().UnixNano()
	ctx := pkgContext.WithTenant(context.Background(), tenant)
	legacy := &model.SystemSmsLog{TemplateCode: "user-sms-login", TemplateContent: "OTP 739281", TemplateParams: map[string]any{"code": "739281"}, Mobile: "13812345678", ApiSendMsg: "provider-token-fixture", ApiReceiveMsg: "739281", ApiRequestId: "provider-token-fixture", ApiSerialNo: "provider-token-fixture"}
	if err := db.WithContext(ctx).Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewSmsLogHandler(svc.NewSmsLogService(query.Use(db)))
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(pkgContext.CtxTenantIDKey, tenant)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.GET("/page", handler.GetSmsLogPage)
	engine.GET("/export", handler.ExportSmsLogExcel)
	assertSafe := func(text string) {
		t.Helper()
		for _, secret := range []string{"739281", "13812345678", "provider-token-fixture"} {
			if strings.Contains(text, secret) {
				t.Fatal("HTTP audit exposes secret")
			}
		}
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest("GET", "/page?pageNo=1&pageSize=10", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "[REDACTED]") {
		t.Fatalf("page not returned: %s", w.Body)
	}
	assertSafe(w.Body.String())
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest("GET", "/export", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	workbook, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	rows, err := workbook.GetRows("短信日志")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatal("missing export row")
	}
	for _, row := range rows {
		assertSafe(strings.Join(row, "|"))
	}
}
