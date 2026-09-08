package system

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
)

func TestLoginAuditKeepsTenantAfterRequestEnd(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	s := NewLoginLogService(query.Use(db))
	ctx, cancel := context.WithCancel(pkgcontext.WithTenant(context.Background(), 1))
	cancel()
	s.CreateLoginLog(ctx, 17, 1, "synthetic", "127.0.0.1", "test", 100, 0)
	s.CreateLogoutLog(ctx, 17, 1, "synthetic", "127.0.0.1", "test")
	s.CreateLogoutLog(context.Background(), 17, 1, "synthetic", "127.0.0.1", "test")
	for tenant, want := range map[int64]int64{1: 2, 2: 0} {
		var count int64
		if err := db.WithContext(pkgcontext.WithTenant(context.Background(), tenant)).Model(&model.SystemLoginLog{}).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("tenant%d count%d want%d", tenant, count, want)
		}
	}
}

func TestNotifyTemplateSelectionIsTenantScoped(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	s := NewNotifyService(repo.NewNotifyTemplateRepository(q), repo.NewNotifyMessageRepository(q))
	for tenant, content := range map[int64]string{1: "tenant A", 2: "tenant B"} {
		ctx := pkgcontext.WithTenant(context.Background(), tenant)
		template := &model.SystemNotifyTemplate{Name: "fixture", Code: "same-code", Content: content, Status: 0}
		if err := db.WithContext(ctx).Create(template).Error; err != nil {
			t.Fatal(err)
		}
	}
	for tenant, want := range map[int64]string{1: "tenant A", 2: "tenant B"} {
		ctx := pkgcontext.WithTenant(context.Background(), tenant)
		id, err := s.SendNotify(ctx, 17, 1, "same-code", nil)
		if err != nil {
			t.Fatal(err)
		}
		var message model.SystemNotifyMessage
		if err := db.WithContext(ctx).First(&message, id).Error; err != nil {
			t.Fatal(err)
		}
		if message.TemplateContent != want || message.TenantID != tenant {
			t.Fatalf("wrong tenant template: %+v", message)
		}
	}
	if _, err := s.SendNotify(context.Background(), 17, 1, "same-code", nil); err == nil {
		t.Fatal("tenantless template lookup succeeded")
	}
}
