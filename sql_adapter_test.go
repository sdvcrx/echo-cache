package cache

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestCacheSQLAdapter(t *testing.T) {
	sqldb, err := sql.Open("sqlite", "file::memory:?cache=shared")
	assert.NoError(t, err)
	sqldb.SetMaxIdleConns(1000)
	sqldb.SetConnMaxLifetime(0)

	sa := NewSQLAdapter(sqldb, "tbl")

	key := "key"
	body := []byte("OK")

	t.Run("Set", func(t *testing.T) {
		resp := NewResponse(200, nil, body)
		err := sa.Set(key, resp, time.Minute)
		assert.NoError(t, err)
	})

	t.Run("Get", func(t *testing.T) {
		resp, err := sa.Get(key)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Set override", func(t *testing.T) {
		resp := NewResponse(201, nil, []byte("NOT OK"))
		err := sa.Set(key, resp, time.Minute)
		assert.NoError(t, err)

		currentResp, err := sa.Get(key)
		if assert.NoError(t, err) {
			assert.NotNil(t, currentResp)
			assert.Equal(t, resp.StatusCode, currentResp.StatusCode)
			assert.Equal(t, resp.Body, currentResp.Body)
		}
	})

	t.Run("Set with TTL", func(t *testing.T) {
		ttl := time.Second
		resp := NewResponse(201, nil, []byte("NOT OK"))
		err := sa.Set(key, resp, ttl)
		assert.NoError(t, err)

		time.Sleep(ttl / 2)
		// still valid
		currentResp, err := sa.Get(key)
		if assert.NoError(t, err) {
			assert.NotNil(t, currentResp)
			assert.Equal(t, resp, currentResp)
		}

		// expired
		time.Sleep(ttl)
		currentResp, err = sa.Get(key)
		if assert.NoError(t, err) {
			assert.Nil(t, currentResp)
		}
	})
}
