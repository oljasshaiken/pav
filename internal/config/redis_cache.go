package config

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pavillio/pav-edi/internal/domain"
)

// RedisCache stores payer configs in Redis (ElastiCache in AWS, LocalStack/local Redis in dev).
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func NewRedisCacheFromURL(redisURL string) (*RedisCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return NewRedisCache(redis.NewClient(opts)), nil
}

func (r *RedisCache) Get(key string) (domain.PayerConfig, bool) {
	ctx := context.Background()
	raw, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return domain.PayerConfig{}, false
	}
	if err != nil {
		return domain.PayerConfig{}, false
	}
	var cfg domain.PayerConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return domain.PayerConfig{}, false
	}
	return cfg, true
}

func (r *RedisCache) Set(key string, cfg domain.PayerConfig, ttl time.Duration) {
	ctx := context.Background()
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	if ttl > 0 {
		_ = r.client.Set(ctx, key, raw, ttl).Err()
		return
	}
	_ = r.client.Set(ctx, key, raw, 0).Err()
}

func (r *RedisCache) Invalidate(key string) {
	ctx := context.Background()
	_ = r.client.Del(ctx, key).Err()
}
