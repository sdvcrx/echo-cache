package cache

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/go-redis/redis/v8"
)

type RedisAdapter struct {
	client *redis.Client
}

func NewRedisAdapter(opt *redis.Options) *RedisAdapter {
	return &RedisAdapter{
		client: redis.NewClient(opt),
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
	_, err = ra.client.Set(context.Background(), key, b, 0).Result()
	return err
}
