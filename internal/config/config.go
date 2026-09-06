package config

import "os"

// Config 应用全局配置，默认值可通过环境变量覆盖。
type Config struct {
	Host string
	Port string
	Mode string
}

func Load() *Config {
	cfg := &Config{
		Host: getEnv("SERVER_HOST", "0.0.0.0"),
		Port: getEnv("SERVER_PORT", "8080"),
		Mode: getEnv("GIN_MODE", "debug"),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
