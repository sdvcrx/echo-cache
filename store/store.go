package store

import "time"

type Store interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, ttl time.Duration) error
}
