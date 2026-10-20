package captcha

// 使用 redis 自定义 store，支持分布式部署

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// CaptchaKeyPre 验证码在 Redis 中的 key 前缀
	CaptchaKeyPre = "captcha"
)

// RedisStore 自定义的 Redis 存储器
type RedisStore struct {
	client     *redis.Client
	Expiration time.Duration
	ctx        context.Context
}

// NewRedisStore 创建一个新的 Redis 存储器实例
func NewRedisStore(client *redis.Client, expiration time.Duration) *RedisStore {
	return NewRedisStoreWithContext(context.Background(), client, expiration)
}

// NewRedisStoreWithContext 创建一个带 context 的 Redis 存储器实例
func NewRedisStoreWithContext(ctx context.Context, client *redis.Client, expiration time.Duration) *RedisStore {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RedisStore{
		client:     client,
		Expiration: expiration,
		ctx:        ctx,
	}
}

// Set 将验证码 ID 和对应的值保存到 Redis 中
func (r *RedisStore) Set(id string, value string) error {
	if r.client == nil {
		return errors.New("redis client 未初始化")
	}
	return r.client.Set(r.ctx, r.key(id), value, r.Expiration).Err()
}

// Get 根据验证码 ID 从 Redis 中获取对应的值，并可选择是否在获取后清除该值
func (r *RedisStore) Get(id string, clear bool) string {
	if r.client == nil {
		return ""
	}
	key := r.key(id)
	value, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return ""
	}
	if clear {
		r.client.Del(r.ctx, key)
	}
	return value
}

// Verify 验证用户输入的答案是否与 Redis 中的值匹配，并可选择是否在验证后清除该值
func (r *RedisStore) Verify(id, answer string, clear bool) bool {
	if r.client == nil {
		return false
	}
	key := r.key(id)
	value, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return false
	}
	if clear {
		r.client.Del(r.ctx, key)
	}
	return value == answer
}

func (r *RedisStore) key(id string) string {
	return CaptchaKeyPre + ":" + id
}
