package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

func TestExpirableMessage(t *testing.T) {
	msg := ExpirableMessage{
		ExpiredAt: time.Now().Add(-1 * time.Minute),
	}
	assert.True(t, msg.Expired())
}

func TestBoltAdapter(t *testing.T) {
	c := NewBoltAdapter(t.TempDir() + "/bolt")
	// dont run auto cleanup
	c.ticker.Stop()
	key := "cacheKey"

	t.Run("cleanup", func(t *testing.T) {
		// add expired responses
		assert.NoError(t, c.Set(key, &Response{StatusCode: 1}, -1*time.Minute))
		assert.NoError(t, c.Set(key, &Response{StatusCode: 2}, -1*time.Minute))

		// manual trigger cleanup
		assert.NoError(t, c.cleanupExpired())

		err := c.db.View(func(tx *bbolt.Tx) error {
			bk := tx.Bucket(c.bucket)
			stat := bk.Stats()
			b := bk.Get([]byte(key))
			assert.Nil(t, b)
			assert.Equal(t, 0, stat.KeyN)
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("Set success", func(t *testing.T) {
		err := c.Set(key, NewResponse(200, nil, []byte("OK")), time.Minute*1)
		assert.NoError(t, err)
	})

	t.Run("Get success", func(t *testing.T) {
		resp, err := c.Get(key)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Get nil", func(t *testing.T) {
		resp, err := c.Get("key-dont-exist")
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Get error", func(t *testing.T) {
		key := "error"

		err := c.db.Update(func(tx *bbolt.Tx) error {
			// insert a invalid JSON string, make sure get error
			return tx.Bucket(c.bucket).Put([]byte(key), []byte("{|}"))
		})
		assert.NoError(t, err)

		res, err := c.Get(key)
		assert.EqualError(t, err, "invalid character '|' looking for beginning of object key string")
		assert.Nil(t, res)
	})
}
