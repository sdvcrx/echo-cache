package cache

type dbName int

func (n dbName) String() string {
	switch n {
	case PostgreSQL:
		return "postgresql"
	case SQLite:
		return "sqlite"
	case MySQL:
		return "mysql"
	default:
		return "invalid"
	}
}

type sqlDialect struct {
	TypeText    string
	TypeBytes   string
	TypeBigInt  string
	TypeBindVar string
}

var (
	sqliteDialect = &sqlDialect{
		TypeText:   "TEXT",
		TypeBytes:  "BLOB",
		TypeBigInt: "INTEGER",
	}
	postgresqlDialect = &sqlDialect{
		TypeText:   "TEXT",
		TypeBytes:  "BYTEA",
		TypeBigInt: "BIGINT",
	}
	mysqlDialect = &sqlDialect{
		TypeText:   "VARCHAR(255)",
		TypeBytes:  "BLOB",
		TypeBigInt: "BIGINT",
	}
)

func getDialect(name dbName) *sqlDialect {
	switch name {
	case SQLite:
		return sqliteDialect
	case PostgreSQL:
		return postgresqlDialect
	case MySQL:
		return mysqlDialect
	default:
		return sqliteDialect
	}
}
