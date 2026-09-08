package pay

import (
	"context"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReviewNotifyTokenCrossOriginRedirect(t *testing.T) {
	old := config.C
	defer func() { config.C = old }()
	config.C = &config.Config{}
	config.C.Pay.NotifyToken = "synthetic-review-token"
	var received string
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get(PayNotifyTokenHeader)
		w.Write([]byte(`{"code":0}`))
	}))
	defer destination.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	config.C.Pay.TrustedNotifyURLs = []string{origin.URL}
	svc := &PayNotifyService{}
	status, _ := svc.invokeNotify(context.Background(), &model.PayNotifyTask{Type: PayNotifyTypeOrder, DataID: 1, MerchantOrderId: "1", NotifyURL: origin.URL})
	if received != "" {
		t.Fatalf("global notify credential forwarded to another origin; status=%d", status)
	}
}

func TestReviewNotifyRejectsUntrustedDestination(t *testing.T) {
	old := config.C
	defer func() { config.C = old }()
	config.C = &config.Config{Pay: config.PayConfig{NotifyToken: "synthetic-test-token"}}
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.Write([]byte(`{"code":0}`)) }))
	defer srv.Close()
	status, _ := (&PayNotifyService{}).invokeNotify(context.Background(), &model.PayNotifyTask{Type: PayNotifyTypeOrder, NotifyURL: srv.URL})
	if called || status == PayNotifyStatusSuccess {
		t.Fatal("untrusted endpoint was called")
	}
}
