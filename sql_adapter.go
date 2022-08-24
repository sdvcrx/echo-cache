package cache

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

type SQLAdapter struct {
	db        *sql.DB
	tableName string
}

func NewSQLAdapter(db *sql.DB, tableName string) CacheAdapter {
	adapter := &SQLAdapter{
		db:        db,
		tableName: tableName,
	}
	adapter.init()
	return adapter
}

func (sa *SQLAdapter) init() {
	if sa.tableName == "" {
		log.Fatalln("echo-cache sql_adapter: tableName cannot be empty")
	}
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
key TEXT UNIQUE,
value BLOB,
expired_at INTEGER
)`, sa.tableName)
	_, err := sa.db.Exec(stmt)
	if err != nil {
		log.Fatalln(err)
	}
}

func (sa *SQLAdapter) Get(key string) (*Response, error) {
	b := []byte{}
	queryStmt := fmt.Sprintf("SELECT value FROM %s WHERE key = $1 AND expired_at > $2", sa.tableName)
	err := sa.db.QueryRow(queryStmt, key, time.Now().UnixMilli()).Scan(&b)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return NewResponseFromJSON(b)
}

func (sa *SQLAdapter) Set(key string, response *Response, ttl time.Duration) error {
	insertStmt := fmt.Sprintf(`INSERT INTO %s
(key, value, expired_at) VALUES ($1, $2, $3)
ON CONFLICT (key)
DO UPDATE SET value = EXCLUDED.value, expired_at = EXCLUDED.expired_at
`, sa.tableName)

	b, err := response.Marshal()
	if err != nil {
		return err
	}

	_, err = sa.db.Exec(insertStmt, key, b, time.Now().Add(ttl).UnixMilli())
	return err
}
