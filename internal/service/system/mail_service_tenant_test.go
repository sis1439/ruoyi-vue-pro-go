package system

import (
	"context"
	"errors"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"gorm.io/gorm"
	"testing"
)

func TestMailSameCodeTenantSelection(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	a := pkgContext.WithTenant(context.Background(), 1901)
	b := pkgContext.WithTenant(context.Background(), 1902)
	aa := &model.SystemMailAccount{Mail: "a@example.invalid", Host: "smtp-a.invalid", Username: "a-user", Password: "a-fixture", Port: 465, SslEnable: true}
	ab := &model.SystemMailAccount{Mail: "b@example.invalid", Host: "smtp-b.invalid", Username: "b-user", Password: "b-fixture", Port: 587, StarttlsEnable: true}
	if err := db.WithContext(a).Create(aa).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(b).Create(ab).Error; err != nil {
		t.Fatal(err)
	}
	ta := &model.SystemMailTemplate{Code: "shared-code", AccountID: aa.ID, Title: "A {name}", Content: "A-body", Params: model.StringListFromCSV{"name"}}
	tb := &model.SystemMailTemplate{Code: "shared-code", AccountID: ab.ID, Title: "B {name}", Content: "B-body", Params: model.StringListFromCSV{"name"}}
	if err := db.WithContext(a).Create(ta).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(b).Create(tb).Error; err != nil {
		t.Fatal(err)
	}
	stopped := errors.New("test stops at audit write before SMTP")
	var captured *model.SystemMailLog
	var selected *model.SystemMailAccount
	if err := db.Callback().Create().Before("gorm:create").Register("test:no_smtp", func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*model.SystemMailLog); ok {
			copy := *row
			captured = &copy
			tx.AddError(stopped)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Query().After("gorm:query").Register("test:account_selection", func(tx *gorm.DB) {
		if tx.Error == nil {
			if row, ok := tx.Statement.Dest.(*model.SystemMailAccount); ok {
				copy := *row
				selected = &copy
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	service := NewMailService(db)
	check := func(ctx context.Context, account *model.SystemMailAccount, template *model.SystemMailTemplate, title string) {
		t.Helper()
		captured = nil
		selected = nil
		_, err := service.SendSingleMail(ctx, []string{"recipient@example.invalid"}, nil, nil, 0, 0, "shared-code", map[string]any{"name": "Test"})
		if !errors.Is(err, stopped) {
			t.Fatalf("send did not reach tenant audit boundary: %v", err)
		}
		tenant, _ := pkgContext.TenantID(ctx)
		if captured == nil || captured.TenantID != tenant || captured.AccountID != account.ID || captured.TemplateID != template.ID || captured.FromMail != account.Mail || captured.TemplateTitle != title {
			t.Fatal("wrong tenant template/account rendered")
		}
		if selected == nil || selected.ID != account.ID || selected.Host != account.Host || selected.Username != account.Username || selected.Password != account.Password || selected.Port != account.Port || selected.SslEnable != account.SslEnable || selected.StarttlsEnable != account.StarttlsEnable {
			t.Fatal("wrong tenant SMTP configuration selected")
		}
	}
	check(a, aa, ta, "A Test")
	check(b, ab, tb, "B Test")
	check(a, aa, ta, "A Test")
	// A fresh read immediately observes changed configuration; no cache refresh exists.
	if err := db.WithContext(a).Model(aa).Update("host", "smtp-a-updated.invalid").Error; err != nil {
		t.Fatal(err)
	}
	aa.Host = "smtp-a-updated.invalid"
	if err := db.WithContext(a).Model(ta).Update("title", "A-updated {name}").Error; err != nil {
		t.Fatal(err)
	}
	check(a, aa, ta, "A-updated Test")
	check(b, ab, tb, "B Test")
}

func TestMailExplicitTenantAndDependencyFailures(t *testing.T) {
	// Deliberately omit TenantPlugin here: the send-path lookups must still be scoped.
	db := testutil.PostgreSQL(t)
	a := pkgContext.WithTenant(context.Background(), 1911)
	b := pkgContext.WithTenant(context.Background(), 1912)
	account := &model.SystemMailAccount{Mail: "b@example.invalid", Host: "smtp-b.invalid", TenantBaseDO: model.TenantBaseDO{TenantID: 1912}}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	foreign := &model.SystemMailTemplate{Code: "foreign-account", AccountID: account.ID, TenantBaseDO: model.TenantBaseDO{TenantID: 1911}}
	own := &model.SystemMailTemplate{Code: "b-only", AccountID: account.ID, TenantBaseDO: model.TenantBaseDO{TenantID: 1912}}
	if err := db.Create(foreign).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(own).Error; err != nil {
		t.Fatal(err)
	}
	queryCount := 0
	failureTable := ""
	queryFailure := errors.New("injected database query failure")
	if err := db.Callback().Query().Before("gorm:query").Register("test:mail_query_failure", func(tx *gorm.DB) {
		queryCount++
		if tx.Statement.Table == failureTable {
			tx.AddError(queryFailure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	service := NewMailService(db)
	if queryCount != 0 {
		t.Fatal("constructor queried tenant data")
	}
	stopped := errors.New("stop before SMTP")
	auditCalls := 0
	if err := db.Callback().Create().Before("gorm:create").Register("test:no_mail_network", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*model.SystemMailLog); ok {
			auditCalls++
			tx.AddError(stopped)
		}
	}); err != nil {
		t.Fatal(err)
	}
	send := func(ctx context.Context, code string) error {
		_, err := service.SendSingleMail(ctx, []string{"recipient@example.invalid"}, nil, nil, 0, 0, code, nil)
		return err
	}
	if err := send(a, "b-only"); !errors.Is(err, consts.ErrMailTemplateNotExists) {
		t.Fatalf("foreign template accepted: %v", err)
	}
	if err := send(a, "foreign-account"); !errors.Is(err, consts.ErrMailAccountNotExists) {
		t.Fatalf("foreign SMTP config accepted: %v", err)
	}
	if _, err := service.GetMailAccount(a, account.ID); !errors.Is(err, consts.ErrMailAccountNotExists) {
		t.Fatal("foreign account lookup accepted")
	}
	before := queryCount
	if err := send(context.Background(), "b-only"); err == nil {
		t.Fatal("tenantless send accepted")
	}
	if queryCount != before {
		t.Fatal("tenantless send queried DB")
	}
	if auditCalls != 0 {
		t.Fatal("invalid request reached mail audit")
	}
	if err := send(b, "b-only"); !errors.Is(err, stopped) {
		t.Fatal(err)
	}
	for _, table := range []string{"system_mail_template", "system_mail_account"} {
		failureTable = table
		if err := send(b, "b-only"); !errors.Is(err, queryFailure) {
			t.Fatalf("query error not propagated: %v", err)
		}
	}
	failureTable = "system_mail_template"
	if err := service.DeleteMailAccount(b, account.ID); !errors.Is(err, queryFailure) {
		t.Fatalf("relationship count error not propagated: %v", err)
	}
	failureTable = ""
	if _, err := service.SendSingleMail(b, []string{"recipient@example.invalid"}, nil, nil, 0, 0, "b-only", map[string]any{"bad": make(chan int)}); err == nil || errors.Is(err, stopped) {
		t.Fatal("parameter serialization error ignored")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	if err := send(b, "b-only"); err == nil || errors.Is(err, consts.ErrMailTemplateNotExists) {
		t.Fatalf("database outage hidden: %v", err)
	}
	if _, err := NewMailService(nil).SendSingleMail(b, nil, nil, nil, 0, 0, "b-only", nil); err == nil {
		t.Fatal("nil database accepted")
	}
}

func TestMailRecipientLookupUsesTenantModel(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	a := pkgContext.WithTenant(context.Background(), 1921)
	b := pkgContext.WithTenant(context.Background(), 1922)
	account := &model.SystemMailAccount{Mail: "a@example.invalid", Host: "smtp-a.invalid"}
	if err := db.WithContext(a).Create(account).Error; err != nil {
		t.Fatal(err)
	}
	template := &model.SystemMailTemplate{Code: "recipient", AccountID: account.ID}
	if err := db.WithContext(a).Create(template).Error; err != nil {
		t.Fatal(err)
	}
	userA := &model.SystemUser{Username: "a-mail", Email: "recipient-a@example.invalid"}
	userB := &model.SystemUser{Username: "b-mail", Email: "recipient-b@example.invalid"}
	if err := db.WithContext(a).Create(userA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(b).Create(userB).Error; err != nil {
		t.Fatal(err)
	}
	stopped := errors.New("stop before SMTP")
	var recipient string
	if err := db.Callback().Create().Before("gorm:create").Register("test:recipient_no_network", func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*model.SystemMailLog); ok {
			if len(row.ToMails) == 1 {
				recipient = row.ToMails[0]
			}
			tx.AddError(stopped)
		}
	}); err != nil {
		t.Fatal(err)
	}
	service := NewMailService(db)
	_, err := service.SendSingleMail(a, nil, nil, nil, userA.ID, consts.UserTypeAdmin, "recipient", nil)
	if !errors.Is(err, stopped) || recipient != userA.Email {
		t.Fatalf("tenant recipient not resolved: %v", err)
	}
	recipient = ""
	if _, err := service.SendSingleMail(a, nil, nil, nil, userB.ID, consts.UserTypeAdmin, "recipient", nil); err == nil || errors.Is(err, stopped) {
		t.Fatal("foreign recipient accepted")
	}
	if recipient != "" {
		t.Fatal("foreign recipient reached SMTP boundary")
	}
	if _, err := service.SendSingleMail(a, nil, nil, nil, userA.ID, consts.UserTypeMember, "recipient", nil); !errors.Is(err, consts.ErrMailSendMailNotExists) {
		t.Fatal("unsupported member email silently queried")
	}
	lookupFailure := errors.New("recipient database unavailable")
	if err := db.Callback().Query().Before("gorm:query").Register("test:mail_recipient_error", func(tx *gorm.DB) {
		if tx.Statement.Table == "system_users" {
			tx.AddError(lookupFailure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SendSingleMail(a, nil, nil, nil, userA.ID, consts.UserTypeAdmin, "recipient", nil); !errors.Is(err, lookupFailure) {
		t.Fatal("recipient query failure swallowed")
	}
}
