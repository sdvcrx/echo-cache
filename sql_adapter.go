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
	ticket  *time.Ticker

	stmtGet          *sql.Stmt
	stmtSet          *sql.Stmt
	stmtCleanExpired *sql.Stmt
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
	adapter.startCleanExpired()
	return adapter
}

func (sa *SQLAdapter) createTable() {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
cache_key %s PRIMARY KEY,
value %s,
expired_at %s
)`, sa.tableName, sa.dialect.TypeText, sa.dialect.TypeBytes, sa.dialect.TypeBigInt)

	_, err := sa.db.ExecContext(sa.ctx, query)
	if err != nil {
		log.Fatalln(err)
	}
}

func (sa *SQLAdapter) prepareGet(ctx context.Context) *sql.Stmt {
	whereClause := "cache_key = ? AND expired_at > ?"
	if sa.dbName == PostgreSQL {
		whereClause = "cache_key = $1 AND expired_at > $2"
	}

	query := fmt.Sprintf(
		"SELECT value FROM %s WHERE %s",
		sa.tableName, whereClause,
	)
	return Must(sa.db.PrepareContext(ctx, query))
}

func (sa *SQLAdapter) prepareSet(ctx context.Context) *sql.Stmt {
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
	return Must(sa.db.PrepareContext(ctx, query))
}

func (sa *SQLAdapter) prepareCleanExpired(ctx context.Context) *sql.Stmt {
	placeholder := "?"
	if sa.dbName == PostgreSQL {
		placeholder = "$1"
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE expired_at < %s", sa.tableName, placeholder)
	return Must(sa.db.PrepareContext(ctx, query))
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
	sa.stmtGet = sa.prepareGet(sa.ctx)
	sa.stmtSet = sa.prepareSet(sa.ctx)
	sa.stmtCleanExpired = sa.prepareCleanExpired(sa.ctx)
}

func (sa *SQLAdapter) cleanExpired() error {
	_, err := sa.stmtCleanExpired.Exec(time.Now().UnixMilli())
	return err
}

func (sa *SQLAdapter) startCleanExpired() {
	sa.ticket = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-sa.ctx.Done():
				sa.db.Close()
				sa.ticket.Stop()
				return
			case <-sa.ticket.C:
				if _, err := sa.stmtCleanExpired.Exec(time.Now().UnixMilli()); err != nil {
					log.Println("Failed to cleanup expired data", err)
				}
			}
		}
	}()
}

func (sa *SQLAdapter) Get(key string) (*Response, error) {
	b := []byte{}
	err := sa.stmtGet.QueryRowContext(sa.ctx, key, time.Now().UnixMilli()).Scan(&b)
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

	_, err = sa.stmtSet.ExecContext(sa.ctx, key, b, time.Now().Add(ttl).UnixMilli())
	return err
}
