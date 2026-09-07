package pay

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
)

type securityPayClient struct{ client.PayClient }

func (*securityPayClient) Init() error { return nil }
func TestPaymentClientTenantBoundary(t *testing.T) {
	db := testutil.PostgreSQL(t)
	app := &payModel.PayApp{ID: 830001, AppKey: "security-app", Name: "A", Status: 0, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	if err := db.Create(app).Error; err != nil {
		t.Fatal(err)
	}
	channel := &payModel.PayChannel{ID: 830001, AppID: app.ID, Code: "security_fixture", Status: 0, Config: &payModel.PayClientConfig{ConfigType: payModel.ConfigTypeNone}, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	if err := db.Create(channel).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	factory := client.NewPayClientFactory()
	client.RegisterCreator("security_fixture", func(id int64, cfg string) (client.PayClient, error) { return &securityPayClient{}, nil })
	svc := NewPayChannelService(query.Use(db), factory)
	a := pkgContext.WithTenant(context.Background(), 1)
	b := pkgContext.WithTenant(context.Background(), 2)
	own, err := svc.GetPayClient(a, channel.ID)
	if err != nil || own == nil {
		t.Fatalf("own channel: %v", err)
	}
	if other, err := svc.GetPayClient(b, channel.ID); err == nil || other != nil {
		t.Fatal("other tenant reached populated cache")
	}
	if _, err := svc.GetPayClient(context.Background(), channel.ID); err == nil {
		t.Fatal("missing context reached cache")
	}
	if err := db.WithContext(a).Model(&payModel.PayChannel{}).Where("id = ?", channel.ID).Update("status", 1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPayClient(a, channel.ID); err == nil {
		t.Fatal("disabled channel reached cache")
	}
	if err := db.WithContext(a).Model(&payModel.PayChannel{}).Where("id = ?", channel.ID).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(a).Model(&payModel.PayApp{}).Where("id = ?", app.ID).Update("status", 1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPayClient(a, channel.ID); err == nil {
		t.Fatal("disabled app reached cache")
	}
}
