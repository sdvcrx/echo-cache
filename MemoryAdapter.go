package cache

import (
	"errors"
	"time"

	"github.com/bluele/gcache"
)

type MemoryAdapter struct {
	maxSize int
	ttl     time.Duration
	gc      gcache.Cache
}

var _ CacheAdapter = &MemoryAdapter{}

func NewMemoryAdapter(size int, ttl time.Duration) CacheAdapter {
	gc := gcache.New(size).LRU().Build()
	return &MemoryAdapter{
		maxSize: size,
		ttl:     ttl,
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

func (ma *MemoryAdapter) Set(key string, response *Response) error {
	return ma.gc.SetWithExpire(key, response, ma.ttl)
}
