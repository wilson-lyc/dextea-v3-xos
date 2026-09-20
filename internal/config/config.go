// Package config 负责加载与解析服务配置。
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config 服务全量配置。
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	MySQL   MySQLConfig   `yaml:"mysql"`
	Redis   RedisConfig   `yaml:"redis"`
	Storage StorageConfig `yaml:"storage"`
	Nacos   NacosConfig   `yaml:"nacos"`
}

type ServerConfig struct {
	// Addr gRPC 监听地址，如 ":15001"。
	Addr string `yaml:"addr"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (m MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&timeout=5s&readTimeout=10s&writeTimeout=10s",
		m.User, m.Password, m.Host, m.Port, m.Database)
}

type RedisConfig struct {
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	TTL      time.Duration `yaml:"gallery-ttl"` // 图库缓存过期时间，如 "10m"。
}

func (r RedisConfig) Addr() string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

type StorageConfig struct {
	MaxUploadSize int64                 `yaml:"max-upload-size"`
	Sources       map[string]SourceSpec `yaml:"sources"`
}

const DefaultMaxUploadSize int64 = 20 << 20

type SourceSpec struct {
	Vendor        string `yaml:"vendor"`
	Endpoint      string `yaml:"endpoint"`
	Region        string `yaml:"region"`
	AccessKey     string `yaml:"access-key"`
	SecretKey     string `yaml:"secret-key"`
	UseSSL        bool   `yaml:"use-ssl"`
	DefaultBucket string `yaml:"default-bucket"`
}

// NacosConfig 与 dextea-product、dextea-store-service 保持同一字段和环境变量契约。
type NacosConfig struct {
	Enabled     bool    `yaml:"enabled"`
	ServerAddr  string  `yaml:"server-addr"`
	ServerPort  uint64  `yaml:"server-port"`
	NamespaceID string  `yaml:"namespace-id"`
	ServiceName string  `yaml:"service-name"`
	GroupName   string  `yaml:"group-name"`
	ClusterName string  `yaml:"cluster-name"`
	Weight      float64 `yaml:"weight"`
	Username    string  `yaml:"username"`
	Password    string  `yaml:"password"`
	InstanceIP  string  `yaml:"instance-ip"`
}

// Default 返回本地开发默认值。敏感信息仍必须通过环境变量或部署 Secret 注入。
func Default() Config {
	return Config{
		Server: ServerConfig{Addr: ":15001"},
		MySQL:  MySQLConfig{Host: "127.0.0.1", Port: 20001, User: "dextea", Database: "dextea"},
		Redis:  RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0, TTL: 10 * time.Minute},
	}
}

// Load 从 YAML 文件读取配置。优先级为：系统环境变量 > .env > YAML。
func Load(path string) (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("加载 .env 失败: %w", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}
	// 支持配置文件中的 ${ENV} 占位符，与原 XOS 配置保持兼容。
	raw = []byte(os.ExpandEnv(string(raw)))

	cfg := Default()
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}
	if err := applyEnv(&cfg); err != nil {
		return Config{}, err
	}
	if err := applyDefaults(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) error {
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":15001"
	}
	if cfg.MySQL.Port == 0 {
		cfg.MySQL.Port = 20001
	}
	if cfg.Redis.Host == "" {
		cfg.Redis.Host = "127.0.0.1"
	}
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	if cfg.Redis.TTL == 0 {
		cfg.Redis.TTL = 10 * time.Minute
	}
	if cfg.Storage.MaxUploadSize <= 0 {
		cfg.Storage.MaxUploadSize = DefaultMaxUploadSize
	}
	if len(cfg.Storage.Sources) == 0 {
		return fmt.Errorf("未配置任何 storage.sources")
	}
	if cfg.Nacos.Enabled {
		if cfg.Nacos.ServerAddr == "" {
			return fmt.Errorf("nacos.enabled=true 时必须配置 nacos.server-addr")
		}
		if cfg.Nacos.ServerPort == 0 {
			cfg.Nacos.ServerPort = 8848
		}
		if cfg.Nacos.ServiceName == "" {
			cfg.Nacos.ServiceName = "dextea-xos"
		}
		if cfg.Nacos.GroupName == "" {
			cfg.Nacos.GroupName = "DEFAULT_GROUP"
		}
		if cfg.Nacos.ClusterName == "" {
			cfg.Nacos.ClusterName = "DEFAULT"
		}
		if cfg.Nacos.Weight <= 0 {
			cfg.Nacos.Weight = 1
		}
	}
	return nil
}

// applyEnv 统一读取服务、数据库、缓存、对象存储和 Nacos 环境变量。
func applyEnv(cfg *Config) error {
	if value, ok := lookupEnv("SERVER_ADDR"); ok {
		cfg.Server.Addr = value
	}
	if value, ok := lookupEnv("MYSQL_HOST"); ok {
		cfg.MySQL.Host = value
	}
	if value, ok := lookupEnv("MYSQL_PORT"); ok {
		port, err := parsePort("MYSQL_PORT", value)
		if err != nil {
			return err
		}
		cfg.MySQL.Port = port
	}
	if value, ok := lookupEnv("MYSQL_USER"); ok {
		cfg.MySQL.User = value
	}
	if value, ok := os.LookupEnv("MYSQL_PASSWORD"); ok {
		cfg.MySQL.Password = value
	}
	if value, ok := lookupEnv("MYSQL_DATABASE"); ok {
		cfg.MySQL.Database = value
	}
	if value, ok := lookupEnv("REDIS_HOST"); ok {
		cfg.Redis.Host = value
	}
	if value, ok := lookupEnv("REDIS_PORT"); ok {
		port, err := parsePort("REDIS_PORT", value)
		if err != nil {
			return err
		}
		cfg.Redis.Port = port
	}
	if value, ok := os.LookupEnv("REDIS_PASSWORD"); ok {
		cfg.Redis.Password = value
	}
	if value, ok := lookupEnv("REDIS_DB"); ok {
		db, err := strconv.Atoi(value)
		if err != nil || db < 0 {
			return fmt.Errorf("REDIS_DB 必须是非负整数: %q", value)
		}
		cfg.Redis.DB = db
	}
	if value, ok := lookupEnv("REDIS_GALLERY_TTL"); ok {
		ttl, err := time.ParseDuration(value)
		if err != nil || ttl <= 0 {
			return fmt.Errorf("REDIS_GALLERY_TTL 必须是正 duration，例如 10m: %q", value)
		}
		cfg.Redis.TTL = ttl
	}
	if value, ok := lookupEnv("STORAGE_MAX_UPLOAD_SIZE"); ok {
		size, err := strconv.ParseInt(value, 10, 64)
		if err != nil || size <= 0 {
			return fmt.Errorf("STORAGE_MAX_UPLOAD_SIZE 必须是正整数: %q", value)
		}
		cfg.Storage.MaxUploadSize = size
	}
	if err := applyStorageEnv(cfg); err != nil {
		return err
	}
	return applyNacosEnv(&cfg.Nacos)
}

func applyStorageEnv(cfg *Config) error {
	sourceName := "minio-dev"
	if value, ok := lookupEnv("STORAGE_SOURCE_NAME"); ok {
		sourceName = value
	}
	source := cfg.Storage.Sources[sourceName]
	hasOverride := false
	setString := func(key string, target *string) {
		if value, ok := lookupEnv(key); ok {
			*target = value
			hasOverride = true
		}
	}
	setString("STORAGE_SOURCE_VENDOR", &source.Vendor)
	setString("STORAGE_SOURCE_ENDPOINT", &source.Endpoint)
	setString("STORAGE_SOURCE_REGION", &source.Region)
	setString("STORAGE_SOURCE_ACCESS_KEY", &source.AccessKey)
	setString("STORAGE_SOURCE_SECRET_KEY", &source.SecretKey)
	setString("STORAGE_SOURCE_DEFAULT_BUCKET", &source.DefaultBucket)
	// 兼容旧版 XOS 示例中的 OSS_* 命名，新的统一契约使用 STORAGE_SOURCE_*。
	if value, ok := os.LookupEnv("OSS_ACCESS_KEY"); ok {
		source.AccessKey, hasOverride = value, true
	}
	if value, ok := os.LookupEnv("OSS_SECRET_KEY"); ok {
		source.SecretKey, hasOverride = value, true
	}
	if value, ok := lookupEnv("STORAGE_SOURCE_USE_SSL"); ok {
		useSSL, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("STORAGE_SOURCE_USE_SSL 必须是 true 或 false: %q", value)
		}
		source.UseSSL, hasOverride = useSSL, true
	}
	if hasOverride {
		if cfg.Storage.Sources == nil {
			cfg.Storage.Sources = make(map[string]SourceSpec)
		}
		cfg.Storage.Sources[sourceName] = source
	}
	return nil
}

func applyNacosEnv(cfg *NacosConfig) error {
	if value, ok := os.LookupEnv("NACOS_ENABLED"); ok && strings.TrimSpace(value) != "" {
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("NACOS_ENABLED 必须是 true 或 false: %w", err)
		}
		cfg.Enabled = enabled
	}
	if value, ok := lookupEnv("NACOS_SERVER_HOST"); ok {
		cfg.ServerAddr = value
	}
	if value, ok := lookupEnv("NACOS_SERVER_PORT"); ok {
		port, err := strconv.ParseUint(value, 10, 16)
		if err != nil || port == 0 {
			return fmt.Errorf("NACOS_SERVER_PORT 必须是 1-65535 的整数: %q", value)
		}
		cfg.ServerPort = port
	}
	if value, ok := lookupEnv("NACOS_NAMESPACE"); ok {
		cfg.NamespaceID = value
	}
	if value, ok := lookupEnv("NACOS_GROUP"); ok {
		cfg.GroupName = value
	}
	if value, ok := lookupEnv("NACOS_CLUSTER"); ok {
		cfg.ClusterName = value
	}
	if value, ok := lookupEnv("NACOS_SERVICE_NAME"); ok {
		cfg.ServiceName = value
	}
	if value, ok := os.LookupEnv("NACOS_USERNAME"); ok {
		cfg.Username = value
	}
	if value, ok := os.LookupEnv("NACOS_PASSWORD"); ok {
		cfg.Password = value
	}
	if value, ok := lookupEnv("NACOS_INSTANCE_IP"); ok {
		cfg.InstanceIP = value
	}
	if value, ok := lookupEnv("NACOS_WEIGHT"); ok {
		weight, err := strconv.ParseFloat(value, 64)
		if err != nil || weight <= 0 {
			return fmt.Errorf("NACOS_WEIGHT 必须是大于 0 的数字: %q", value)
		}
		cfg.Weight = weight
	}
	return nil
}

func lookupEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func parsePort(name, value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("%s 必须是 1-65535 的整数: %q", name, value)
	}
	return port, nil
}
