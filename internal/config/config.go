package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用全局配置。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
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
	Sources map[string]SourceSpec `mapstructure:"sources"`
}

// SourceSpec 单个对象存储源的配置，统一 S3 协议接入。
type SourceSpec struct {
	Vendor        string `mapstructure:"vendor"`         // 厂商标识：minio / aws-s3 / aliyun-oss ...
	Endpoint      string `mapstructure:"endpoint"`       // 不带 http(s) 前缀
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
	if len(cfg.Storage.Sources) == 0 {
		return nil, fmt.Errorf("no storage sources configured")
	}
	return &cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
