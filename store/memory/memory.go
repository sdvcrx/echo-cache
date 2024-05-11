package memorystore

import (
	"time"

	"github.com/phuslu/lru"
	"github.com/sdvcrx/echo-cache/store"
)

type MemoryStore struct {
	cache *lru.TTLCache[string, []byte]
}

var _ store.Store = (*MemoryStore)(nil)

func New(size int) store.Store {
	return &MemoryStore{
		cache: lru.NewTTLCache[string, []byte](size),
	}
}

func (ma *MemoryStore) Get(key string) ([]byte, error) {
	val, _ := ma.cache.Get(key)

	return val, nil
}

func (ma *MemoryStore) Set(key string, val []byte, ttl time.Duration) error {
	ma.cache.Set(key, val, ttl)
	return nil
}
