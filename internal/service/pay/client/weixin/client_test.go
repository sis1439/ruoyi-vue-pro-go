package weixin

import (
	"context"
	"testing"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/service/pay/client"
)

// TestEveryRegisteredChannelIsDispatchable 锁定 T05：注册了却分派不了的渠道会在下单时直接失败。
// 旧代码注册 wx_wap 但只在 switch 里认 wx_h5，且构造函数把渠道写死成 wx_unknown。
func TestEveryRegisteredChannelIsDispatchable(t *testing.T) {
	registered := []string{"wx_pub", "wx_lite", "wx_app", "wx_native", "wx_wap", "wx_bar"}
	for _, code := range registered {
		if _, ok := payMethodOf(code); !ok && code != "wx_bar" {
			t.Errorf("渠道 %s 已注册但无法分派下单方式", code)
		}
	}
	if _, ok := payMethodOf("wx_unknown"); ok {
		t.Error("wx_unknown 不应是可下单渠道")
	}
}

// TestCreatorKeepsChannelCode 构造函数必须保留真实渠道编码。
func TestCreatorKeepsChannelCode(t *testing.T) {
	c, err := NewWxPayClientAsClient(3, "wx_lite", "{}")
	if err != nil {
		t.Fatal(err)
	}
	wx := c.(*WxPayClient)
	if wx.ChannelCode != "wx_lite" {
		t.Fatalf("渠道编码丢失: got %q", wx.ChannelCode)
	}
}

// TestJsapiRequiresOpenid JSAPI 必须拿到 openid 才下单；
// 提交服务不透传 channelExtras 时会在这里失败，而不是拿着空 openid 调渠道。
func TestJsapiRequiresOpenid(t *testing.T) {
	c, _ := NewWxPayClient(1, "wx_lite", "{}")
	_, err := c.jsapiOrder(context.Background(), &client.UnifiedOrderReq{OutTradeNo: "P1"})
	if err == nil {
		t.Fatal("缺少 openid 时应报错")
	}
	_, err = c.jsapiOrder(context.Background(), &client.UnifiedOrderReq{
		OutTradeNo: "P1", ChannelExtras: map[string]string{"openid": ""},
	})
	if err == nil {
		t.Fatal("空 openid 时应报错")
	}
}

// TestExpireTimeZeroValue 零值过期时间不得下发给渠道，否则订单下单即过期。
func TestExpireTimeZeroValue(t *testing.T) {
	if expireTime(time.Time{}) != nil {
		t.Fatal("零值时间应返回 nil")
	}
	now := time.Now()
	if got := expireTime(now); got == nil || !got.Equal(now) {
		t.Fatal("非零时间应原样返回")
	}
}

// TestDeref nil 指针不得 panic —— SDK 的可选字段常为 nil。
func TestDeref(t *testing.T) {
	if deref(nil) != "" {
		t.Fatal("nil 应返回空串")
	}
	v := "x"
	if deref(&v) != "x" {
		t.Fatal("非 nil 应返回值")
	}
}
