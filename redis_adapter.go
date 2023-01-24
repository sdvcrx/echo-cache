package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisAdapter struct {
	client *redis.Client
}

func NewRedisAdapter(opt *redis.Options) CacheAdapter {
	return &RedisAdapter{
		client: redis.NewClient(opt),
	}
}

var _ CacheAdapter = &RedisAdapter{}

func (ra *RedisAdapter) Get(key string) ([]byte, error) {
	val, err := ra.client.Get(context.Background(), key).Bytes()
	if err != nil {
		// no data
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

func (ra *RedisAdapter) Set(key string, val []byte, ttl time.Duration) error {
	_, err := ra.client.Set(context.Background(), key, val, ttl).Result()
	return err
}
