package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sdvcrx/echo-cache/store"
)

const (
	SQLite DBName = iota
	PostgreSQL
	MySQL
)

type SQLStoreOption struct {
	DB        *sql.DB
	Ctx       context.Context
	TableName string
	DBName    DBName
}

type SQLStore struct {
	SQLStoreOption
	dialect *sqlDialect
	ticket  *time.Ticker

	stmtGet          *sql.Stmt
	stmtSet          *sql.Stmt
	stmtCleanExpired *sql.Stmt
}

var _ store.Store = (*SQLStore)(nil)

var DefaultSQLStoreOption = SQLStoreOption{
	Ctx:       context.Background(),
	TableName: "echo_cache",
	DBName:    SQLite,
}

func New(option SQLStoreOption) store.Store {
	sqlStore := &SQLStore{
		SQLStoreOption: DefaultSQLStoreOption,
	}

	if option.Ctx != nil {
		sqlStore.Ctx = option.Ctx
	}
	if option.DB != nil {
		sqlStore.DB = option.DB
	}
	if option.TableName != "" {
		sqlStore.TableName = option.TableName
	}
	if option.DBName != SQLite {
		sqlStore.DBName = option.DBName
	}

	sqlStore.init()
	sqlStore.startCleanExpired()
	return sqlStore
}

func (sa *SQLStore) createTable() {
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

func (sa *SQLStore) prepareGet(ctx context.Context) *sql.Stmt {
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

func (sa *SQLStore) prepareSet(ctx context.Context) *sql.Stmt {
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

func (sa *SQLStore) prepareCleanExpired(ctx context.Context) *sql.Stmt {
	placeholder := "?"
	if sa.DBName == PostgreSQL {
		placeholder = "$1"
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE expired_at < %s", sa.TableName, placeholder)
	return must(sa.DB.PrepareContext(ctx, query))
}

func (sa *SQLStore) init() {
	if sa.TableName == "" {
		log.Fatalln("echo-cache sqlstore: tableName cannot be empty")
	}
	if sa.DB == nil {
		log.Fatalln("echo-cache sqlstore: db cannot be nil")
	}
	if sa.DBName.String() == "invalid" {
		log.Fatalln("echo-cache sqlstore: dbName is invalid")
	}
	// TODO quote tableName

	sa.dialect = getDialect(sa.DBName)

	sa.createTable()
	sa.stmtGet = sa.prepareGet(sa.Ctx)
	sa.stmtSet = sa.prepareSet(sa.Ctx)
	sa.stmtCleanExpired = sa.prepareCleanExpired(sa.Ctx)
}

// The lock of clearning expired cache
var sqlCleanMutex sync.Mutex

func (sa *SQLStore) cleanExpired() error {
	sqlCleanMutex.Lock()
	defer sqlCleanMutex.Unlock()

	_, err := sa.stmtCleanExpired.Exec(time.Now().UnixMilli())
	return err
}

func (sa *SQLStore) startCleanExpired() {
	sa.ticket = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-sa.Ctx.Done():
				sa.DB.Close()
				sa.ticket.Stop()
				return
			case <-sa.ticket.C:
				if err := sa.cleanExpired(); err != nil {
					log.Println("Failed to cleanup expired data:", err)
				}
			}
		}
	}()
}

func (sa *SQLStore) Get(key string) ([]byte, error) {
	b := []byte{}
	err := sa.stmtGet.QueryRowContext(sa.Ctx, key, time.Now().UnixMilli()).Scan(&b)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

func (sa *SQLStore) Set(key string, val []byte, ttl time.Duration) error {
	_, err := sa.stmtSet.ExecContext(sa.Ctx, key, val, time.Now().Add(ttl).UnixMilli())
	return err
}
