package cache

import (
	"errors"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockGCache struct {
	mock.Mock
	gcache.Cache
}

func (m *mockGCache) Get(key interface{}) (interface{}, error) {
	ret := m.Called(key)

	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0), ret.Error(1)
}

func (m *mockGCache) Set(key, value interface{}) error {
	ret := m.Called(key, value)
	return ret.Error(0)
}

func (m *mockGCache) SetWithExpire(key, value interface{}, expiration time.Duration) error {
	ret := m.Called(key, value, expiration)
	return ret.Error(0)
}

func TestNewMemoryAdapter(t *testing.T) {
	gc := &mockGCache{}
	cache := &MemoryAdapter{
		gc: gc,
	}
	key := "cacheKey"
	body := []byte("OK")
	ttl := time.Minute

	t.Run("Get success", func(t *testing.T) {
		gc.On("Get", key).Return(body, nil).Once()
		r, err := cache.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, body, r)
		gc.AssertCalled(t, "Get", key)
	})

	t.Run("Get nil", func(t *testing.T) {
		gc.On("Get", key).Return(nil, gcache.KeyNotFoundError).Once()
		r, err := cache.Get(key)
		assert.NoError(t, err)
		assert.Nil(t, r)
		gc.AssertCalled(t, "Get", key)
	})

	t.Run("Get error", func(t *testing.T) {
		ErrGet := errors.New("Get Errror")
		gc.On("Get", key).Return(nil, ErrGet).Once()
		r, err := cache.Get(key)
		assert.Nil(t, r)
		assert.ErrorIs(t, err, ErrGet)
		gc.AssertCalled(t, "Get", key)
	})

	t.Run("Set success", func(t *testing.T) {
		gc.On("SetWithExpire", key, body, ttl).Return(nil).Once()

		err := cache.Set(key, body, ttl)
		assert.NoError(t, err)
		gc.AssertCalled(t, "SetWithExpire", key, body, ttl)
	})

	t.Run("Set with zero duration", func(t *testing.T) {
		gc.On("Set", key, body).Return(nil).Once()

		err := cache.Set(key, body, 0)
		assert.NoError(t, err)
		gc.AssertCalled(t, "Set", key, body)
	})

	t.Run("Set error", func(t *testing.T) {
		ErrSet := errors.New("Set Errror")
		gc.On("SetWithExpire", key, body, ttl).Return(ErrSet).Once()

		err := cache.Set(key, body, ttl)
		assert.ErrorIs(t, err, ErrSet)
		gc.AssertCalled(t, "SetWithExpire", key, body, ttl)
	})
}
