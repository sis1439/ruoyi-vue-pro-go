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

	if value, ok := os.LookupEnv("RUOYI_JWT_SECRET"); ok {
		C.Security.JWTSecret = value
	}
	if value, ok := os.LookupEnv("RUOYI_DATABASE_DSN"); ok {
		C.Database.DSN = value
	}
	if value, ok := os.LookupEnv("RUOYI_DATABASE_DRIVER"); ok {
		C.Database.Driver = value
	}
	return C.Security.Validate()
}
