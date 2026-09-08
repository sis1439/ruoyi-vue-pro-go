package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var C = new(Config)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Log      LogConfig      `mapstructure:"log"`
	Database DatabaseConfig `mapstructure:"database"`
	Security SecurityConfig `mapstructure:"security"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Trade    TradeConfig    `mapstructure:"trade"`
	Pay      PayConfig      `mapstructure:"pay"`
	IoT      IoTConfig      `mapstructure:"iot"`
}

type IoTConfig struct {
	Core    IoTCoreConfig    `mapstructure:"core"`
	Gateway IoTGatewayConfig `mapstructure:"gateway"`
}

type IoTCoreConfig struct {
	MessageBus MessageBusConfig `mapstructure:"message_bus"`
}

type MessageBusConfig struct {
	WorkerNum int `mapstructure:"worker_num"`
	QueueSize int `mapstructure:"queue_size"`
}

type IoTGatewayConfig struct {
	ServerID       string           `mapstructure:"server_id"`
	MaxConnections int              `mapstructure:"max_connections"`
	MQTT           MQTTClientConfig `mapstructure:"mqtt"`
}

// MQTTClientConfig MQTT 客户端配置 (复用 internal 定义，或者搬迁到这里)
// 为了解耦，我们在 pkg/config 定义一份，internal/iot/gateway/mqtt_client.go 可以直接使用 pkg/config 的 struct 或者进行转换
// 鉴于 internal 不应该被 pkg 引用，所以我们在 pkg/config 定义 DTO
type MQTTClientConfig struct {
	Broker           string   `mapstructure:"broker"`
	ClientID         string   `mapstructure:"client_id"`
	Username         string   `mapstructure:"username"`
	Password         string   `mapstructure:"password"`
	KeepAlive        string   `mapstructure:"keep_alive"` // YAML string "60s"
	ConnectTimeout   string   `mapstructure:"connect_timeout"`
	AutoReconnect    bool     `mapstructure:"auto_reconnect"`
	CleanSession     bool     `mapstructure:"clean_session"`
	SubscribeTopics  []string `mapstructure:"subscribe_topics"`
	DefaultCodecType string   `mapstructure:"default_codec_type"`
	TopicPrefix      string   `mapstructure:"topic_prefix"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	// JWTSecret 认证签名密钥。非 local 环境必须显式配置，不得使用内置默认值。
	JWTSecret string `mapstructure:"jwt_secret"`
}

// IsProdLike 非本地环境（含 dev/test/staging/prod）适用更严格的配置校验
func (c AppConfig) IsProdLike() bool {
	return c.Env != "" && c.Env != "local"
}

type HTTPConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

type SecurityConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

type DatabaseConfig struct {
	Driver      string `mapstructure:"driver"`
	DSN         string `mapstructure:"dsn"`
	MaxIdle     int    `mapstructure:"max_idle"`
	MaxOpen     int    `mapstructure:"max_open"`
	MaxLifetime int    `mapstructure:"max_lifetime"`
}

func (c SecurityConfig) Validate() error {
	secret := strings.TrimSpace(c.JWTSecret)
	lower := strings.ToLower(secret)
	distinct := map[rune]bool{}
	for _, r := range secret {
		distinct[r] = true
	}
	if len(secret) < 32 || len(distinct) < 8 || strings.Contains(lower, "change") || strings.Contains(lower, "placeholder") || strings.Contains(lower, "ruoyi-mall-secret") || strings.Contains(lower, "your-secret") {
		return fmt.Errorf("security.jwt_secret must be an independent secret of at least 32 bytes, supplied through RUOYI_JWT_SECRET")
	}
	return nil
}

type MySQLConfig struct {
	DSN         string `mapstructure:"dsn"`
	MaxIdle     int    `mapstructure:"max_idle"`
	MaxOpen     int    `mapstructure:"max_open"`
	MaxLifetime int    `mapstructure:"max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type TradeConfig struct {
	Express ExpressConfig `mapstructure:"express"`
}

type ExpressConfig struct {
	Client string      `mapstructure:"client"`
	Kd100  Kd100Config `mapstructure:"kd100"`
	// KdNiao KdNiaoConfig `mapstructure:"kdniao"`
}

type Kd100Config struct {
	Customer string `mapstructure:"customer"`
	Key      string `mapstructure:"key"`
}

type PayConfig struct {
	OrderNotifyURL  string `mapstructure:"order_notify_url"`
	RefundNotifyURL string `mapstructure:"refund_notify_url"`
	OrderNoPrefix   string `mapstructure:"order_no_prefix"`
	WalletPayAppKey string `mapstructure:"wallet_pay_app_key"`
	// NotifyToken 支付中心 → 商城业务通知的共享令牌。
	// 商城侧的 /trade/order/update-paid 等回调是公开路由，
	// 除了重新查询可信支付记录外，再用它建立内部调用的信任边界。
	// 非 local 环境必须配置。
	NotifyToken string `mapstructure:"notify_token"`
}

func Load() error {
	// 读取环境变量
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "local"
	}

	viper.SetConfigName("config." + env) // e.g. config.local
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")    // 相对路径
	viper.AddConfigPath("../config") // 兼容测试路径

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := viper.Unmarshal(C); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if C.App.Env == "" {
		C.App.Env = env
	}

	if value, ok := os.LookupEnv("RUOYI_JWT_SECRET"); ok {
		C.Security.JWTSecret = value
	}
	if value, ok := os.LookupEnv("RUOYI_DATABASE_DSN"); ok {
		C.Database.DSN = value
	}
	if value, ok := os.LookupEnv("RUOYI_DATABASE_DRIVER"); ok {
		C.Database.Driver = value
	}
	return C.Validate()
}

// Validate 启动期配置校验。缺少关键配置时给出可操作错误并拒绝启动，
// 不用默认值假装成功——用默认值启动会在支付回调、认证等路径静默失败。
func (c *Config) Validate() error {
	var problems []string
	if err := c.Security.Validate(); err != nil { problems = append(problems, err.Error()) }

	if c.Database.DSN == "" {
		problems = append(problems, "database.dsn 未配置：数据库无法连接")
	}
	if c.Redis.Addr == "" {
		problems = append(problems, "redis.addr 未配置：单号生成、通知锁、登录白名单都依赖 Redis")
	}
	// 支付回调地址用于拼接渠道回调 URL（pay/order.go genChannelOrderNotifyUrl）。
	// 留空会让渠道拿到形如 "/3" 的非法地址，支付结果永远回不来。
	if c.Pay.OrderNotifyURL == "" {
		problems = append(problems, "pay.order_notify_url 未配置：支付渠道回调地址无法拼接")
	}
	if c.Pay.RefundNotifyURL == "" {
		problems = append(problems, "pay.refund_notify_url 未配置：退款渠道回调地址无法拼接")
	}

	if c.App.IsProdLike() {
		if c.Security.JWTSecret == "" {
			problems = append(problems, "app.jwt_secret 未配置：非 local 环境不允许使用内置默认密钥")
		}
		if c.Pay.NotifyToken == "" {
			problems = append(problems, "pay.notify_token 未配置：支付中心到商城的业务通知缺少信任边界")
		}
		for _, u := range []string{c.Pay.OrderNotifyURL, c.Pay.RefundNotifyURL} {
			if strings.HasPrefix(u, "http://") {
				problems = append(problems, "支付回调地址必须使用 https: "+u)
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("配置校验失败 (env=%s):\n  - %s", c.App.Env, strings.Join(problems, "\n  - "))
	}
	return nil
}
