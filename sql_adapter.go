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
	DB        *sql.DB
	Ctx       context.Context
	TableName string
	DBName    dbName
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
	Ctx:       context.Background(),
	TableName: "echo_cache",
	DBName:    SQLite,
}

func NewSQLAdapter(option SQLAdapterOption) CacheAdapter {
	adapter := &SQLAdapter{
		SQLAdapterOption: DefaultSQLAdapterOption,
	}

	if option.Ctx != nil {
		adapter.Ctx = option.Ctx
	}
	if option.DB != nil {
		adapter.DB = option.DB
	}
	if option.TableName != "" {
		adapter.TableName = option.TableName
	}
	if option.DBName != SQLite {
		adapter.DBName = option.DBName
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
)`, sa.TableName, sa.dialect.TypeText, sa.dialect.TypeBytes, sa.dialect.TypeBigInt)

	_, err := sa.DB.ExecContext(sa.Ctx, query)
	if err != nil {
		log.Fatalln(err)
	}
}

func (sa *SQLAdapter) prepareGet(ctx context.Context) *sql.Stmt {
	whereClause := "cache_key = ? AND expired_at > ?"
	if sa.DBName == PostgreSQL {
		whereClause = "cache_key = $1 AND expired_at > $2"
	}

	query := fmt.Sprintf(
		"SELECT value FROM %s WHERE %s",
		sa.TableName, whereClause,
	)
	return must(sa.DB.PrepareContext(ctx, query))
}

func (sa *SQLAdapter) prepareSet(ctx context.Context) *sql.Stmt {
	placeholder := "?, ?, ?"
	onConflict := `ON CONFLICT (cache_key) DO UPDATE
SET value = EXCLUDED.value, expired_at = EXCLUDED.expired_at`

	if sa.DBName == PostgreSQL {
		placeholder = "$1, $2, $3"
	} else if sa.DBName == MySQL {
		onConflict = `ON DUPLICATE KEY
UPDATE value = VALUES(value), expired_at = VALUES(expired_at)`
	}

	query := fmt.Sprintf(`INSERT INTO %s (cache_key, value, expired_at) VALUES (%s) %s`, sa.TableName, placeholder, onConflict)
	return must(sa.DB.PrepareContext(ctx, query))
}

func (sa *SQLAdapter) prepareCleanExpired(ctx context.Context) *sql.Stmt {
	placeholder := "?"
	if sa.DBName == PostgreSQL {
		placeholder = "$1"
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE expired_at < %s", sa.TableName, placeholder)
	return must(sa.DB.PrepareContext(ctx, query))
}

func (sa *SQLAdapter) init() {
	if sa.TableName == "" {
		log.Fatalln("echo-cache sql_adapter: tableName cannot be empty")
	}
	if sa.DB == nil {
		log.Fatalln("echo-cache sql_adapter: db cannot be nil")
	}
	if sa.DBName.String() == "invalid" {
		log.Fatalln("echo-cache sql_adapter: dbName is invalid")
	}
	// TODO quote tableName

	sa.dialect = getDialect(sa.DBName)

	sa.createTable()
	sa.stmtGet = sa.prepareGet(sa.Ctx)
	sa.stmtSet = sa.prepareSet(sa.Ctx)
	sa.stmtCleanExpired = sa.prepareCleanExpired(sa.Ctx)
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
			case <-sa.Ctx.Done():
				sa.DB.Close()
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
	err := sa.stmtGet.QueryRowContext(sa.Ctx, key, time.Now().UnixMilli()).Scan(&b)
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

	_, err = sa.stmtSet.ExecContext(sa.Ctx, key, b, time.Now().Add(ttl).UnixMilli())
	return err
}
