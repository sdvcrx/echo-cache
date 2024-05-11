package memorystore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryStore(t *testing.T) {
	cache := New(20)
	key := "cacheKey"
	body := []byte("OK")

	t.Run("Get/Set success", func(t *testing.T) {
		ttl := time.Second
		cache.Set(key, body, ttl)

		r, err := cache.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, body, r)
	})

	t.Run("Get nil", func(t *testing.T) {
		r, err := cache.Get("nil")
		assert.NoError(t, err)
		assert.Nil(t, r)
	})

	t.Run("Set val expired", func(t *testing.T) {
		ttl := 2 * time.Second
		key := "expired"
		assert.NoError(t, cache.Set(key, body, ttl))

		time.Sleep(ttl / 2)
		r, err := cache.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, body, r)

		time.Sleep(ttl/2 + 100*time.Millisecond)
		r, err = cache.Get(key)
		assert.NoError(t, err)
		assert.Nil(t, r)
	})

	t.Run("Set with zero duration", func(t *testing.T) {
		key := "0s"
		err := cache.Set(key, body, 0)
		assert.NoError(t, err)

		time.Sleep(1 * time.Second)
		r, err := cache.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, body, r)
	})
}
