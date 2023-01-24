package cache

import "time"

type CacheAdapter interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, ttl time.Duration) error
}
