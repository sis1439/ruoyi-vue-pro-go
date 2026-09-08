package config

import "testing"

func base() *Config {
	c := &Config{}
 c.Security.JWTSecret = "cd-test-8b2d6e41a079c3f5e8d2946a01bf"
	c.App.Env = "local"
	c.Database.DSN = "x"
	c.Redis.Addr = "127.0.0.1:6379"
	c.Pay.OrderNotifyURL = "http://localhost:48080/pay/notify/order"
	c.Pay.RefundNotifyURL = "http://localhost:48080/pay/notify/refund"
	return c
}

// TestValidateRejectsMissingConfig 缺关键配置必须拒绝启动，而不是用默认值假装成功。
func TestValidateRejectsMissingConfig(t *testing.T) {
	if err := base().Validate(); err != nil {
		t.Fatalf("完整的本地配置应通过: %v", err)
	}

	cases := map[string]func(*Config){
		"database.dsn":             func(c *Config) { c.Database.DSN = "" },
		"redis.addr":            func(c *Config) { c.Redis.Addr = "" },
		"pay.order_notify_url":  func(c *Config) { c.Pay.OrderNotifyURL = "" },
		"pay.refund_notify_url": func(c *Config) { c.Pay.RefundNotifyURL = "" },
	}
	for name, mutate := range cases {
		c := base()
		mutate(c)
		if err := c.Validate(); err == nil {
			t.Errorf("缺少 %s 时应拒绝启动", name)
		}
	}
}

// TestValidateRejectsWeakProdConfig 非 local 环境不得使用默认密钥或明文回调地址。
func TestValidateRejectsWeakProdConfig(t *testing.T) {
	c := base()
	c.App.Env = "prod"
	if err := c.Validate(); err == nil {
		t.Fatal("prod 环境缺少 jwt_secret 且回调为 http 时应拒绝启动")
	}

	c.App.JWTSecret = "a-real-secret"
	c.Pay.NotifyToken = "a-real-notify-token"
	c.Pay.OrderNotifyURL = "https://api.example.com/pay/notify/order"
	c.Pay.RefundNotifyURL = "https://api.example.com/pay/notify/refund"
	if err := c.Validate(); err != nil {
		t.Fatalf("完整的 prod 配置应通过: %v", err)
	}
}
