package cache

import (
	"context"
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
	pgDB, _ := sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	if pgDB.Ping() == nil {
		tests = append(tests, testCase{PostgreSQL, pgDB})
	}

	// docker run -it --rm -e MYSQL_DATABASE=cache -e MYSQL_ROOT_PASSWORD=mysql -p 3306:3306 mysql
	mysqlDB, _ := sql.Open("mysql", "root:mysql@/cache")
	if mysqlDB.Ping() == nil {
		tests = append(tests, testCase{MySQL, mysqlDB})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, tt := range tests {
		sa := NewSQLAdapter(SQLAdapterOption{
			Ctx:    ctx,
			DB:     tt.db,
			DBName: tt.dbName,
		})

		t.Run(tt.dbName.String(), func(t *testing.T) {
			key := "key"
			body := []byte("OK")

			t.Run("Set", func(t *testing.T) {
				// resp := NewResponse(200, nil, body)
				err := sa.Set(key, body, time.Minute)
				assert.NoError(t, err)
			})

			t.Run("Get", func(t *testing.T) {
				resp, err := sa.Get(key)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, body, resp)
			})

			t.Run("Set override", func(t *testing.T) {
				// resp := NewResponse(201, nil, []byte("NOT OK"))
				valNew := []byte("NOT OK")
				err := sa.Set(key, valNew, time.Minute)
				assert.NoError(t, err)

				res, err := sa.Get(key)
				if assert.NoError(t, err) {
					assert.NotNil(t, res)
					assert.Equal(t, valNew, res)
				}
			})

			t.Run("Set with TTL", func(t *testing.T) {
				ttl := time.Second
				// resp := NewResponse(201, nil, []byte("NOT OK"))
				err := sa.Set(key, body, ttl)
				assert.NoError(t, err)

				time.Sleep(ttl / 2)
				// still valid
				res, err := sa.Get(key)
				if assert.NoError(t, err) {
					assert.NotNil(t, res)
					assert.Equal(t, body, res)
				}

				// expired
				time.Sleep(ttl)
				res, err = sa.Get(key)
				if assert.NoError(t, err) {
					assert.Nil(t, res)
				}
			})
		})
	}
}
