package pay

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	payModel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	service "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

type reviewRejectedNotify struct {
	client.PayClient
	parsed bool
}

func (*reviewRejectedNotify) Init() error { return nil }
func (c *reviewRejectedNotify) ParseOrderNotify(*client.NotifyData) (*client.OrderResp, error) {
	c.parsed = true
	return nil, errors.New("invalid signature")
}
func (c *reviewRejectedNotify) ParseRefundNotify(*client.NotifyData) (*client.RefundResp, error) {
	c.parsed = true
	return nil, errors.New("invalid signature")
}
func (c *reviewRejectedNotify) ParseTransferNotify(*client.NotifyData) (*client.TransferResp, error) {
	c.parsed = true
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
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	fake := &reviewRejectedNotify{}
	client.RegisterCreator("review_rejected", func(int64, string, string) (client.PayClient, error) { return fake, nil })
	factory := client.NewPayClientFactory()
	handler := NewPayNotifyHandler(nil, nil, service.NewPayChannelService(query.Use(db), factory), nil, nil, nil, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(pkgcontext.WithTenant(context.Background(), 1))
	})
	router.POST("/order/:channelId", handler.NotifyOrder)
	router.POST("/refund/:channelId", handler.NotifyRefund)
	router.POST("/transfer/:channelId", handler.NotifyTransfer)
	for _, kind := range []string{"order", "refund", "transfer"} {
		t.Run(kind, func(t *testing.T) {
			fake.parsed = false
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/"+kind+"/1", nil))
			if !fake.parsed {
				t.Fatal("parser was not exercised")
			}
			if w.Code < 400 {
				t.Fatalf("invalid callback acknowledged: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
