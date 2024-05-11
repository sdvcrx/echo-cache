package redisstore

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sdvcrx/echo-cache/store"
)

type RedisStore struct {
	client redis.UniversalClient
}

func New(opt *redis.UniversalOptions) store.Store {
	return &RedisStore{
		client: redis.NewUniversalClient(opt),
	}
}

var _ store.Store = (*RedisStore)(nil)

func (ra *RedisStore) Get(key string) ([]byte, error) {
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

func (ra *RedisStore) Set(key string, val []byte, ttl time.Duration) error {
	_, err := ra.client.Set(context.Background(), key, val, ttl).Result()
	return err
}
