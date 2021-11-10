package cache

import (
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
