package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	SQLite dbName = iota
	PostgreSQL
	MySQL
)

type SQLAdapterOption struct {
	db        *sql.DB
	ctx       context.Context
	tableName string
	dbName    dbName
}

type SQLAdapter struct {
	SQLAdapterOption
	dialect *sqlDialect

	stmtGet *sql.Stmt
	stmtSet *sql.Stmt
}

var DefaultSQLAdapterOption = SQLAdapterOption{
	ctx:       context.Background(),
	tableName: "echo_cache",
	dbName:    SQLite,
}

func NewSQLAdapter(option SQLAdapterOption) CacheAdapter {
	adapter := &SQLAdapter{
		SQLAdapterOption: DefaultSQLAdapterOption,
	}

	if option.ctx != nil {
		adapter.ctx = option.ctx
	}
	if option.db != nil {
		adapter.db = option.db
	}
	if option.tableName != "" {
		adapter.tableName = option.tableName
	}
	if option.dbName != SQLite {
		adapter.dbName = option.dbName
	}

	adapter.init()
	return adapter
}

func (sa *SQLAdapter) createTable() {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
cache_key %s PRIMARY KEY,
value %s,
expired_at %s
)`, sa.tableName, sa.dialect.TypeText, sa.dialect.TypeBytes, sa.dialect.TypeBigInt)

	_, err := sa.db.Exec(query)
	if err != nil {
		log.Fatalln(err)
	}
}

func (sa *SQLAdapter) prepareGet() *sql.Stmt {
	whereClause := "cache_key = ? AND expired_at > ?"
	if sa.dbName == PostgreSQL {
		whereClause = "cache_key = $1 AND expired_at > $2"
	}

	query := fmt.Sprintf(
		"SELECT value FROM %s WHERE %s",
		sa.tableName, whereClause,
	)
	return Must(sa.db.Prepare(query))
}

func (sa *SQLAdapter) prepareSet() *sql.Stmt {
	placeholder := "?, ?, ?"
	onConflict := `ON CONFLICT (cache_key) DO UPDATE
SET value = EXCLUDED.value, expired_at = EXCLUDED.expired_at`

	if sa.dbName == PostgreSQL {
		placeholder = "$1, $2, $3"
	} else if sa.dbName == MySQL {
		onConflict = `ON DUPLICATE KEY
UPDATE value = VALUES(value), expired_at = VALUES(expired_at)`
	}

	query := fmt.Sprintf(`INSERT INTO %s (cache_key, value, expired_at) VALUES (%s) %s`, sa.tableName, placeholder, onConflict)
	return Must(sa.db.Prepare(query))
}

func (sa *SQLAdapter) init() {
	if sa.tableName == "" {
		log.Fatalln("echo-cache sql_adapter: tableName cannot be empty")
	}
	if sa.db == nil {
		log.Fatalln("echo-cache sql_adapter: db cannot be nil")
	}
	if sa.dbName.String() == "invalid" {
		log.Fatalln("echo-cache sql_adapter: dbName is invalid")
	}
	// TODO quote tableName

	sa.dialect = getDialect(sa.dbName)

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
