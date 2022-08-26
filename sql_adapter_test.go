package cache

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestCacheSQLAdapter(t *testing.T) {
	sqliteDB, err := sql.Open("sqlite", "file::memory:?cache=shared")
	assert.NoError(t, err)

	type testCase struct {
		dbName dbName
		db     *sql.DB
	}

	tests := []testCase{
		{SQLite, sqliteDB},
	}

	// docker run -it --rm -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:alpine
	pgDB, err := sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	if pgDB.Ping() == nil {
		tests = append(tests, testCase{PostgreSQL, pgDB})
	}

	// docker run -it --rm -e MYSQL_DATABASE=cache -e MYSQL_ROOT_PASSWORD=mysql -p 3306:3306 mysql
	mysqlDB, err := sql.Open("mysql", "root:mysql@/cache")
	if mysqlDB.Ping() == nil {
		tests = append(tests, testCase{MySQL, mysqlDB})
	}

	for _, tt := range tests {
		sa := NewSQLAdapter(SQLAdapterOption{
			db:     tt.db,
			dbName: tt.dbName,
		})

		t.Run(tt.dbName.String(), func(t *testing.T) {
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
		})
	}
}
