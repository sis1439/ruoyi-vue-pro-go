package pay

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/middleware"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	service "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type reviewRejectedNotify struct {
	client.PayClient
	parsed bool
	body   string
}

func (*reviewRejectedNotify) Init() error { return nil }
func (c *reviewRejectedNotify) ParseOrderNotify(data *client.NotifyData) (*client.OrderResp, error) {
	c.parsed = true
	c.body = data.Body
	return nil, errors.New("invalid signature")
}
func (c *reviewRejectedNotify) ParseRefundNotify(data *client.NotifyData) (*client.RefundResp, error) {
	c.parsed = true
	c.body = data.Body
	return nil, errors.New("invalid signature")
}
func (c *reviewRejectedNotify) ParseTransferNotify(data *client.NotifyData) (*client.TransferResp, error) {
	c.parsed = true
	c.body = data.Body
	return nil, errors.New("invalid signature")
}
func TestReviewRejectedCallbackMustNotReturn200(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Model(&payModel.PayApp{}).Where("id=1").Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	channel := &payModel.PayChannel{ID: 1, AppID: 1, Code: "review_rejected", Config: &payModel.PayClientConfig{ConfigType: payModel.ConfigTypeNone}, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	if err := db.Create(channel).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.SystemTenant{}).Where("id = ?", 1).Updates(map[string]any{
		"website": model.StringListFromCSV{"callback.example.test"}, "status": 0, "expire_time": time.Now().Add(time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	fake := &reviewRejectedNotify{}
	client.RegisterCreator("review_rejected", func(int64, string, string) (client.PayClient, error) { return fake, nil })
	factory := client.NewPayClientFactory()
	handler := NewPayNotifyHandler(nil, nil, service.NewPayChannelService(query.Use(db), factory), nil, nil, nil, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.InjectContext(), middleware.TrustedTenant(db))
	router.POST("/admin-api/pay/notify/order/:channelId", handler.NotifyOrder)
	router.POST("/admin-api/pay/notify/refund/:channelId", handler.NotifyRefund)
	router.POST("/admin-api/pay/notify/transfer/:channelId", handler.NotifyTransfer)
	for _, kind := range []string{"order", "refund", "transfer"} {
		t.Run(kind, func(t *testing.T) {
			fake.parsed = false
			fake.body = ""
			w := httptest.NewRecorder()
			body := "app_id=app1&sign=a%2Bb%3D&total_amount=1.00"
			req := httptest.NewRequest(http.MethodPost, "/admin-api/pay/notify/"+kind+"/1", strings.NewReader(body))
			req.Host = "callback.example.test"
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			router.ServeHTTP(w, req)
			if fake.body != body {
				t.Fatalf("callback body changed before verification: %q", fake.body)
			}
			if !fake.parsed {
				t.Fatal("parser was not exercised")
			}
			if w.Code < 400 {
				t.Fatalf("invalid callback acknowledged: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
