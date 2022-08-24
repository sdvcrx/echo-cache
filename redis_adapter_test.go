package cache

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
)

func TestCacheRedisAdapter(t *testing.T) {
	db, mock := redismock.NewClientMock()
	ra := &RedisAdapter{
		client: db,
	}
	key := "cacheKey"
	body := []byte("OK")
	resp := NewResponse(200, nil, body)
	respb, errm := resp.Marshal()
	assert.NoError(t, errm)

	t.Run("Get success", func(t *testing.T) {
		mock.ExpectGet(key).SetVal(string(respb))
		res, err := ra.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, body, res.Body)
		mock.ExpectationsWereMet()
	})

	t.Run("Get nil", func(t *testing.T) {
		mock.ExpectGet(key).RedisNil()
		res, err := ra.Get(key)
		assert.NoError(t, err)
		// redis return nil means cacheKey not found
		// return nil as response
		assert.Nil(t, res)
		mock.ExpectationsWereMet()
	})

	t.Run("Get error", func(t *testing.T) {
		mock.ExpectGet(key).SetErr(redis.ErrClosed)
		res, err := ra.Get(key)
		assert.ErrorIs(t, err, redis.ErrClosed)
		assert.Nil(t, res)
		mock.ExpectationsWereMet()
	})

	t.Run("Set success", func(t *testing.T) {
		ttl := time.Minute
		mock.ExpectSet(key, respb, ttl).SetVal("")
		err := ra.Set(key, resp, ttl)
		assert.NoError(t, err)
	})

	t.Run("Set error", func(t *testing.T) {
		mock.ExpectSet(key, respb, 0).SetErr(redis.ErrClosed)
		err := ra.Set(key, resp, 0)
		assert.ErrorIs(t, err, redis.ErrClosed)
	})
}

func TestRedisAdapterWithRealServer(t *testing.T) {
	db := redis.NewClient(&redis.Options{})
	if err := db.Ping(context.Background()).Err(); err != nil {
		t.Skip("Cannot connect to redis server, skip test redis with real server")
	}
	ra := &RedisAdapter{
		client: db,
	}
	key := "cacheKey"
	body := []byte("OK")
	resp := NewResponse(200, nil, body)

	t.Run("Set", func(t *testing.T) {
		err := ra.Set(key, resp, time.Minute)
		assert.NoError(t, err)
	})

	t.Run("Get", func(t *testing.T) {
		resp, err := ra.Get(key)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Set with TTL", func(t *testing.T) {
		ttl := time.Second
		err := ra.Set(key, resp, ttl)
		assert.NoError(t, err)

		time.Sleep(ttl)
		resp, err := ra.Get(key)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})
}
