package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用全局配置。
type Config struct {
	RPC       RPCConfig       `mapstructure:"rpc"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

// RedisConfig Redis 连接与缓存配置。
type RedisConfig struct {
	Host     string        `mapstructure:"host"`
	Port     string        `mapstructure:"port"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	TTL      time.Duration `mapstructure:"gallery-ttl"` // 图库缓存过期时间，如 "10m"
}

func (r RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

// TelemetryConfig OpenTelemetry 配置，导出端点走标准 OTEL_EXPORTER_OTLP_ENDPOINT 环境变量
type TelemetryConfig struct {
	Enable         bool   `mapstructure:"enable"`
	ServiceName    string `mapstructure:"service-name"`
	ServiceVersion string `mapstructure:"service-version"`
}

type RPCConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// StorageConfig yml 中 storage 段的结构。
type StorageConfig struct {
	MaxUploadSize int64                 `mapstructure:"max-upload-size"` // 单文件大小上限（字节），0 表示使用默认值
	Sources       map[string]SourceSpec `mapstructure:"sources"`
}

// DefaultMaxUploadSize 未配置 max-upload-size 时使用的默认单文件上限（20MB）。
const DefaultMaxUploadSize int64 = 20 << 20

// SourceSpec 单个对象存储源的配置，统一 S3 协议接入。
type SourceSpec struct {
	Vendor        string `mapstructure:"vendor"`   // 厂商标识：minio / aws-s3 / aliyun-oss ...
	Endpoint      string `mapstructure:"endpoint"` // 不带 http(s) 前缀
	Region        string `mapstructure:"region"`
	AccessKey     string `mapstructure:"access-key"`
	SecretKey     string `mapstructure:"secret-key"`
	UseSSL        bool   `mapstructure:"use-ssl"`
	DefaultBucket string `mapstructure:"default-bucket"`
}

// Load 加载配置：默认读取 configs/config.yaml，可用 CONFIG_PATH 覆盖。
// 配置值支持 ${ENV} 形式的环境变量引用，便于后续迁移到 nacos 配置中心。
func Load() (*Config, error) {
	path := getEnv("CONFIG_PATH", "configs/config.yaml")

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	// 手动展开 ${ENV} 引用，viper 不会自动处理配置文件中的环境变量占位
	for _, key := range v.AllKeys() {
		val, ok := v.Get(key).(string)
		if !ok || !strings.HasPrefix(val, "${") {
			continue
		}
		if expanded := os.ExpandEnv(val); expanded != "" {
			v.Set(key, expanded)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Telemetry.ServiceName == "" {
		cfg.Telemetry.ServiceName = "dextea-xos"
	}
	if cfg.RPC.Host == "" {
		cfg.RPC.Host = "0.0.0.0"
	}
	if cfg.RPC.Port == "" {
		cfg.RPC.Port = "9091"
	}
	if cfg.Redis.Host == "" {
		cfg.Redis.Host = "127.0.0.1"
	}
	if cfg.Redis.Port == "" {
		cfg.Redis.Port = "6379"
	}
	if cfg.Redis.TTL == 0 {
		cfg.Redis.TTL = 10 * time.Minute
	}
	if len(cfg.Storage.Sources) == 0 {
		return nil, fmt.Errorf("no storage sources configured")
	}
	if cfg.Storage.MaxUploadSize <= 0 {
		cfg.Storage.MaxUploadSize = DefaultMaxUploadSize
	}
	return &cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
