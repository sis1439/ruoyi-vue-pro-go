package config

import "testing"

func TestReviewProductionDefaultSecret(t *testing.T) {
	c := base()
	c.App.Env = "prod"
	c.Security.JWTSecret = "yudao-backend-go-secret"
	c.Pay.NotifyToken = "x"
	c.Pay.OrderNotifyURL = "https://localhost/notify/order"
	c.Pay.RefundNotifyURL = "https://localhost/notify/refund"
	if err := c.Validate(); err == nil {
		t.Fatal("production accepts the source-code default JWT secret and one-byte notify token")
	}
}

func TestReviewProductionSecurityAndURLValidation(t *testing.T) {
	valid := func() *Config {
		c := base()
		c.App.Env = "prod"
		c.Pay.NotifyToken = "cd-notify-491aef6350db7c2690e5fb864ac"
		c.Pay.OrderNotifyURL = "https://api.example.com/order"
		c.Pay.RefundNotifyURL = "https://api.example.com/refund"
		c.Pay.TrustedNotifyURLs = []string{"https://mall.example.com/paid"}
		return c
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"default JWT":   func(c *Config) { c.Security.JWTSecret = "yudao-backend-go-secret" },
		"weak notify":   func(c *Config) { c.Pay.NotifyToken = "x" },
		"shared key":    func(c *Config) { c.Pay.NotifyToken = c.Security.JWTSecret },
		"unbound token": func(c *Config) { c.Pay.TrustedNotifyURLs = nil },
		"relative URL":  func(c *Config) { c.Pay.OrderNotifyURL = "/order" },
		"plaintext URL": func(c *Config) { c.Pay.RefundNotifyURL = "http://api.example.com/refund" },
		"userinfo URL":  func(c *Config) { c.Pay.TrustedNotifyURLs = []string{"https://user:secret@mall.example.com/paid"} },
	} {
		t.Run(name, func(t *testing.T) {
			c := valid()
			mutate(c)
			if err := c.Validate(); err == nil {
				t.Fatal("unsafe config accepted")
			}
		})
	}
}
