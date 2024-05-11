package cache

import (
	"time"

	"github.com/phuslu/lru"
)

type MemoryAdapter struct {
	cache *lru.TTLCache[string, []byte]
}

var _ CacheAdapter = &MemoryAdapter{}

func NewMemoryAdapter(size int) CacheAdapter {
	return &MemoryAdapter{
		cache: lru.NewTTLCache[string, []byte](size),
	}
}

func (ma *MemoryAdapter) Get(key string) ([]byte, error) {
	val, _ := ma.cache.Get(key)

	return val, nil
}

func (ma *MemoryAdapter) Set(key string, val []byte, ttl time.Duration) error {
	ma.cache.Set(key, val, ttl)
	return nil
}
