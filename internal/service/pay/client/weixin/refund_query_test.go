package weixin

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type refundTestTransport func(*http.Request) (*http.Response, error)

func (f refundTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRefundQueryUsesSDKAndVerifiesResponse(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	state := "SUCCESS"
	invalidSignature := false
	httpClient := &http.Client{Transport: refundTestTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.Path != "/v3/refund/domestic/refunds/R1" {
			return nil, fmt.Errorf("unexpected SDK request %s %s", req.Method, req.URL.Path)
		}
		body := fmt.Sprintf(`{"status":%q,"out_trade_no":"P1","out_refund_no":"R1","refund_id":"wx-R1"}`, state)
		timestamp := fmt.Sprint(time.Now().Unix())
		nonce := "query-nonce"
		digest := sha256.Sum256([]byte(timestamp + "\n" + nonce + "\n" + body + "\n"))
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			return nil, err
		}
		if invalidSignature {
			signature[0] ^= 1
		}
		headers := http.Header{}
		headers.Set("Wechatpay-Timestamp", timestamp)
		headers.Set("Wechatpay-Nonce", nonce)
		headers.Set("Wechatpay-Serial", "PUB_KEY_ID_TEST")
		headers.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString(signature))
		headers.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: 200, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	sdk, err := core.NewClient(context.Background(), option.WithMerchantCredential("test-mch", "test-serial", key), option.WithVerifier(verifiers.NewSHA256WithRSAPubkeyVerifier("PUB_KEY_ID_TEST", key.PublicKey)), option.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	c := &WxPayClient{coreClient: sdk}
	for value, want := range map[string]int{"SUCCESS": consts.PayRefundStatusSuccess, "CLOSED": consts.PayRefundStatusFailure, "PROCESSING": consts.PayRefundStatusWaiting, "ABNORMAL": consts.PayRefundStatusWaiting} {
		state = value
		out, err := c.GetRefund(context.Background(), "P1", "R1")
		if err != nil || out.Status != want {
			t.Fatalf("%s: %+v %v", value, out, err)
		}
	}
	if _, err := c.GetRefund(context.Background(), "different-payment", "R1"); err == nil {
		t.Fatal("mismatched payment accepted")
	}
	state = "UNKNOWN"
	if _, err := c.GetRefund(context.Background(), "P1", "R1"); err == nil {
		t.Fatal("unknown status accepted")
	}
	state = "SUCCESS"
	invalidSignature = true
	if _, err := c.GetRefund(context.Background(), "P1", "R1"); err == nil {
		t.Fatal("invalid SDK response signature accepted")
	}
}
