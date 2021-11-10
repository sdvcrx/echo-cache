package cache

import (
	"errors"
	"time"

	"github.com/bluele/gcache"
)

type MemoryAdapter struct {
	maxSize int
	gc      gcache.Cache
}

var _ CacheAdapter = &MemoryAdapter{}

func NewMemoryAdapter(size int) CacheAdapter {
	gc := gcache.New(size).LRU().Build()
	return &MemoryAdapter{
		maxSize: size,
		gc:      gc,
	}
}

func (ma *MemoryAdapter) Get(key string) (*Response, error) {
	resp, err := ma.gc.Get(key)
	if err != nil && errors.Is(err, gcache.KeyNotFoundError) {
		return nil, nil
	}
	return resp.(*Response), err
}

func (ma *MemoryAdapter) Set(key string, response *Response, ttl time.Duration) error {
	return ma.gc.SetWithExpire(key, response, ttl)
}
