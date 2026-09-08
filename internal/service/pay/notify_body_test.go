package pay

import (
	"encoding/json"
	"testing"

	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
)

// TestBuildNotifyBody 锁定 T06：通知请求体必须携带真实标识，不能是 "{}"。
func TestBuildNotifyBody(t *testing.T) {
	body, err := buildNotifyBody(&pay.PayNotifyTask{
		Type: PayNotifyTypeOrder, DataID: 99, MerchantOrderId: "1024",
	})
	if err != nil {
		t.Fatal(err)
	}
	var order struct {
		MerchantOrderId string `json:"merchantOrderId"`
		PayOrderID      int64  `json:"payOrderId"`
	}
	if err := json.Unmarshal(body, &order); err != nil {
		t.Fatal(err)
	}
	if order.MerchantOrderId != "1024" || order.PayOrderID != 99 {
		t.Fatalf("订单通知体不正确: %s", body)
	}

	body, err = buildNotifyBody(&pay.PayNotifyTask{
		Type: PayNotifyTypeRefund, DataID: 7, MerchantOrderId: "1024", MerchantRefundId: "order-1024",
	})
	if err != nil {
		t.Fatal(err)
	}
	var refund struct {
		MerchantRefundId string `json:"merchantRefundId"`
		PayRefundId      int64  `json:"payRefundId"`
	}
	if err := json.Unmarshal(body, &refund); err != nil {
		t.Fatal(err)
	}
	if refund.MerchantRefundId != "order-1024" || refund.PayRefundId != 7 {
		t.Fatalf("退款通知体不正确: %s", body)
	}

	if _, err := buildNotifyBody(&pay.PayNotifyTask{Type: 99}); err == nil {
		t.Fatal("未知通知类型应报错")
	}
}

// TestIsNotifySuccess 锁定 T06：成功判定依据统一 JSON 业务码，而非裸字符串。
func TestIsNotifySuccess(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{`{"code":0,"msg":"","data":true}`, true},
		{`{"code":1004001001,"msg":"订单不存在"}`, false},
		{`success`, false},   // 旧实现会误判为成功
		{`SUCCESS`, false},   // 同上
		{`{"data":true}`, false}, // 缺 code 视为失败，触发重试
		{``, false},
	}
	for _, c := range cases {
		if got := isNotifySuccess([]byte(c.body)); got != c.want {
			t.Errorf("isNotifySuccess(%q) = %v, want %v", c.body, got, c.want)
		}
	}
}
