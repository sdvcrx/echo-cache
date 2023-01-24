package cache

import (
	"errors"
	"time"

	"github.com/bluele/gcache"
)

type MemoryType int

func (m MemoryType) String() string {
	switch m {
	case TYPE_SIMPLE:
		return "simple"
	case TYPE_LRU:
		return "lru"
	case TYPE_LFU:
		return "lfu"
	case TYPE_ARC:
		return "arc"
	}
	return "simple"
}

const (
	TYPE_SIMPLE MemoryType = iota + 1
	TYPE_LRU
	TYPE_LFU
	TYPE_ARC
)

type MemoryAdapter struct {
	gc gcache.Cache
}

var _ CacheAdapter = &MemoryAdapter{}

func NewMemoryAdapter(size int, memoryType MemoryType) CacheAdapter {
	gc := gcache.
		New(size).
		EvictType(memoryType.String()).
		Build()

	return &MemoryAdapter{
		gc: gc,
	}
}

func (ma *MemoryAdapter) Get(key string) ([]byte, error) {
	resp, err := ma.gc.Get(key)
	if err != nil {
		if errors.Is(err, gcache.KeyNotFoundError) {
			return nil, nil
		}
		return nil, err
	}
	return resp.([]byte), err
}

func (ma *MemoryAdapter) Set(key string, val []byte, ttl time.Duration) error {
	if ttl == 0 {
		return ma.gc.Set(key, val)
	}
	return ma.gc.SetWithExpire(key, val, ttl)
}
