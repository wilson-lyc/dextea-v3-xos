package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("SERVER_ADDR", ":19091")
	t.Setenv("MYSQL_HOST", "mysql")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("MYSQL_USER", "app")
	t.Setenv("MYSQL_PASSWORD", "mysql-password")
	t.Setenv("MYSQL_DATABASE", "xos")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_GALLERY_TTL", "15m")
	t.Setenv("STORAGE_SOURCE_ENDPOINT", "storage:9000")
	t.Setenv("STORAGE_SOURCE_ACCESS_KEY", "access")
	t.Setenv("STORAGE_SOURCE_SECRET_KEY", "secret")
	t.Setenv("STORAGE_SOURCE_USE_SSL", "true")
	t.Setenv("STORAGE_SOURCE_DEFAULT_BUCKET", "images")
	t.Setenv("NACOS_ENABLED", "true")
	t.Setenv("NACOS_SERVER_HOST", "nacos")
	t.Setenv("NACOS_SERVER_PORT", "18848")
	t.Setenv("NACOS_NAMESPACE", "namespace")
	t.Setenv("NACOS_GROUP", "group")
	t.Setenv("NACOS_CLUSTER", "cluster")
	t.Setenv("NACOS_SERVICE_NAME", "xos")
	t.Setenv("NACOS_WEIGHT", "2.5")
	t.Setenv("NACOS_INSTANCE_IP", "10.0.0.2")

	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
server:
  addr: ":9091"
mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  database: "dextea"
redis:
  host: "127.0.0.1"
  port: 6379
  gallery-ttl: 10m
storage:
  sources:
    minio-dev:
      vendor: minio
      endpoint: 127.0.0.1:9000
      default-bucket: original
nacos:
  enabled: false
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Addr != ":19091" || cfg.MySQL.Host != "mysql" || cfg.MySQL.Port != 3307 || cfg.MySQL.User != "app" || cfg.MySQL.Password != "mysql-password" || cfg.MySQL.Database != "xos" {
		t.Fatalf("service/database env not applied: %+v %+v", cfg.Server, cfg.MySQL)
	}
	if cfg.Redis.Host != "redis" || cfg.Redis.Port != 6380 || cfg.Redis.DB != 2 || cfg.Redis.TTL != 15*time.Minute {
		t.Fatalf("redis env not applied: %+v", cfg.Redis)
	}
	source := cfg.Storage.Sources["minio-dev"]
	if source.Endpoint != "storage:9000" || source.AccessKey != "access" || source.SecretKey != "secret" || !source.UseSSL || source.DefaultBucket != "images" {
		t.Fatalf("storage env not applied: %+v", source)
	}
	if !cfg.Nacos.Enabled || cfg.Nacos.ServerAddr != "nacos" || cfg.Nacos.ServerPort != 18848 || cfg.Nacos.NamespaceID != "namespace" || cfg.Nacos.GroupName != "group" || cfg.Nacos.ClusterName != "cluster" || cfg.Nacos.ServiceName != "xos" || cfg.Nacos.InstanceIP != "10.0.0.2" || cfg.Nacos.Weight != 2.5 {
		t.Fatalf("nacos env not applied: %+v", cfg.Nacos)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("MYSQL_PORT", "not-a-port")
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("storage:\n  sources:\n    local:\n      endpoint: local:9000\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want invalid MYSQL_PORT error")
	}
}

func TestLoadUsesConsistentDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("storage:\n  sources:\n    local:\n      endpoint: local:9000\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Addr != ":15001" || cfg.MySQL.Port != 20001 || cfg.Redis.Port != 6379 || cfg.Redis.TTL != 10*time.Minute {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}
