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

	stmtGet *sql.Stmt
	stmtSet *sql.Stmt
}

func NewSQLAdapter(db *sql.DB, tableName string) CacheAdapter {
	adapter := &SQLAdapter{
		db:        db,
		tableName: tableName,
	}
	adapter.init()
	return adapter
}

func (sa *SQLAdapter) createTable() {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
key TEXT UNIQUE,
value BLOB,
expired_at INTEGER
)`, sa.tableName)

	_, err := sa.db.Exec(query)
	if err != nil {
		log.Fatalln(err)
	}
}

func (sa *SQLAdapter) prepareGet() *sql.Stmt {
	query := fmt.Sprintf("SELECT value FROM %s WHERE key = $1 AND expired_at > $2", sa.tableName)
	return Must(sa.db.Prepare(query))
}

func (sa *SQLAdapter) prepareSet() *sql.Stmt {
	query := fmt.Sprintf(`INSERT INTO %s
(key, value, expired_at) VALUES ($1, $2, $3)
ON CONFLICT (key)
DO UPDATE SET value = EXCLUDED.value, expired_at = EXCLUDED.expired_at
`, sa.tableName)
	return Must(sa.db.Prepare(query))
}

func (sa *SQLAdapter) init() {
	if sa.tableName == "" {
		log.Fatalln("echo-cache sql_adapter: tableName cannot be empty")
	}
	// TODO check and quote tableName

	sa.createTable()
	sa.stmtGet = sa.prepareGet()
	sa.stmtSet = sa.prepareSet()
}

func (sa *SQLAdapter) Get(key string) (*Response, error) {
	b := []byte{}
	err := sa.stmtGet.QueryRow(key, time.Now().UnixMilli()).Scan(&b)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return NewResponseFromJSON(b)
}

func (sa *SQLAdapter) Set(key string, response *Response, ttl time.Duration) error {
	b, err := response.Marshal()
	if err != nil {
		return err
	}

	_, err = sa.stmtSet.Exec(key, b, time.Now().Add(ttl).UnixMilli())
	return err
}
