package system

import (
	"context"
	"encoding/json"
	"fmt"
	contract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/system"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"strings"
	"testing"
	"time"
)

func TestSmsAuditSecrets(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	ctx := pkgContext.WithTenant(context.Background(), time.Now().UnixNano())
	q := query.Use(db)
	logs := NewSmsLogService(q)
	params := map[string]any{"code": "739281", "token": "provider-token-fixture"}
	template := &model.SystemSmsTemplate{ID: 1, Code: "user-sms-login", ChannelId: 1}
	id, err := logs.CreateSmsLogWithStatus(ctx, "13812345678", 1, 1, true, template, "OTP 739281", params)
	if err != nil {
		t.Fatal(err)
	}
	assertSafe := func(value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"739281", "13812345678", "provider-token-fixture"} {
			if strings.Contains(string(data), secret) {
				t.Fatalf("audit leaks %s", secret)
			}
		}
	}
	var stored model.SystemSmsLog
	if err := db.WithContext(ctx).First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	assertSafe(stored)
	if params["code"] != "739281" {
		t.Fatal("provider input was mutated")
	}
	sender := NewSmsSendService(q, nil, logs, nil)
	sender.updateLogSendFail(ctx, id, fmt.Errorf("739281 mobile 13812345678 provider-token-fixture"))
	if err := db.WithContext(ctx).First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	assertSafe(stored)
	sender.updateLogSendSuccess(ctx, id, &client.SmsSendResp{ApiSendCode: "OK", ApiSendMsg: "739281 mobile 13812345678", ApiRequestId: "provider-token-fixture", ApiSerialNo: "provider-token-fixture"})
	if err := db.WithContext(ctx).First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	assertSafe(stored)
	if stored.ApiSendCode != "OK" {
		t.Fatal("delivery-confirmation status changed")
	}
	// Seed a pre-fix row directly: both page and export consume the same redacted DTO.
	legacy := &model.SystemSmsLog{TemplateCode: "user-sms-login", TemplateContent: "OTP 739281", TemplateParams: params, Mobile: "13812345678", ApiSendMsg: "provider-token-fixture", ApiReceiveMsg: "739281", ApiRequestId: "provider-token-fixture", ApiSerialNo: "provider-token-fixture", ApiSendCode: "739281", ApiReceiveCode: "739281"}
	if err := db.WithContext(ctx).Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	page, err := logs.GetSmsLogPage(ctx, &contract.SmsLogPageReq{PageParam: pagination.PageParam{PageNo: 1, PageSize: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatal(page.Total)
	}
	assertSafe(page)
}
