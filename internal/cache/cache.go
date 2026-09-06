package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache Redis 缓存客户端，封装 JSON 序列化的旁路缓存读写。
// 所有方法在 Redis 不可用时返回错误，由调用方决定降级到数据库。
type Cache struct {
	rdb *redis.Client
}

// NewRedis 创建 Redis 客户端（不在此处 Ping，由调用方启动时校验连通性）。
func NewRedis(addr, password string, db int) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})}
}

// Ping 检查 Redis 连通性。
func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close 关闭底层连接。
func (c *Cache) Close() error {
	return c.rdb.Close()
}

// Get 读取 key 并反序列化到 dst，未命中返回 (false, nil)。
func (c *Cache) Get(ctx context.Context, key string, dst any) (bool, error) {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return false, fmt.Errorf("cache unmarshal %s: %w", key, err)
	}
	return true, nil
}

// Set 序列化 val 并写入 key。
func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("cache marshal %s: %w", key, err)
	}
	return c.rdb.Set(ctx, key, b, ttl).Err()
}

// Del 删除 key，用于旁路缓存的写后失效。
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// Incr 原子自增并返回新值，用于版本号 key（首次调用前 key 不存在则从 0 自增为 1）。
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}
