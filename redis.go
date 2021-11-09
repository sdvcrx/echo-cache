package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisAdapter struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisAdapter(opt *redis.Options, ttl time.Duration) CacheAdapter {
	return &RedisAdapter{
		client: redis.NewClient(opt),
		ttl:    ttl,
	}
}

var _ CacheAdapter = &RedisAdapter{}

func (ra *RedisAdapter) Get(key string) (*Response, error) {
	cachedResponse, err := ra.client.Get(context.Background(), key).Result()
	if err != nil {
		// no data
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return NewResponseFromJSON(cachedResponse)
}

func (ra *RedisAdapter) Set(key string, response *Response) error {
	if response == nil {
		return nil
	}
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = ra.client.Set(context.Background(), key, b, ra.ttl).Result()
	return err
}
