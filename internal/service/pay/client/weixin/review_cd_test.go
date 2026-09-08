package weixin

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
	"testing"
	"time"
)

func TestReviewSignedRefundNotify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	apiKey := "0123456789abcdef0123456789abcdef"
	c, _ := NewWxPayClient(1, "wx_lite", "{}")
	c.config = &WxPayClientConfig{APIV3Key: apiKey, PublicKeyID: "PUB_KEY_ID_REVIEW", MchID: "review-mch", AppID: "review-app"}
	c.publicKey = &key.PublicKey
	plain := []byte(`{"mchid":"review-mch","out_trade_no":"P1","transaction_id":"wx1","out_refund_no":"R1","refund_id":"wx-refund1","refund_status":"SUCCESS","success_time":"2026-09-08T00:00:00+08:00","user_received_account":"review","amount":{"total":100,"refund":100,"payer_total":100,"payer_refund":100}}`)
	block, _ := aes.NewCipher([]byte(apiKey))
	aead, _ := cipher.NewGCM(block)
	nonce := "123456789012"
	aad := "refund"
	ciphertext := base64.StdEncoding.EncodeToString(aead.Seal(nil, []byte(nonce), plain, []byte(aad)))
	body, _ := json.Marshal(map[string]any{"id": "review", "event_type": "REFUND.SUCCESS", "resource_type": "encrypt-resource", "resource": map[string]string{"algorithm": "AEAD_AES_256_GCM", "ciphertext": ciphertext, "nonce": nonce, "associated_data": aad}})
	timestamp := fmt.Sprint(time.Now().Unix())
	signatureNonce := "review-signature-nonce"
	digest := sha256.Sum256([]byte(timestamp + "\n" + signatureNonce + "\n" + string(body) + "\n"))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	data := &client.NotifyData{Body: string(body), Headers: map[string]string{"Wechatpay-Timestamp": timestamp, "Wechatpay-Nonce": signatureNonce, "Wechatpay-Serial": "PUB_KEY_ID_REVIEW", "Wechatpay-Signature": base64.StdEncoding.EncodeToString(sig)}}
	defer func() {
		if v := recover(); v != nil {
			t.Errorf("valid signed and encrypted refund notification panics: %v", v)
		}
	}()
	out, err := c.ParseRefundNotify(data)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != consts.PayRefundStatusSuccess {
		t.Fatalf("status=%d", out.Status)
	}
}
