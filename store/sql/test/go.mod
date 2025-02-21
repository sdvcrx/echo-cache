module github.com/sdvcrx/echo-cache/store/sql/test

replace github.com/sdvcrx/echo-cache/store => ../..

replace github.com/sdvcrx/echo-cache/store/sql => ../

go 1.23.0

toolchain go1.24.0

require (
	github.com/go-sql-driver/mysql v1.9.0
	github.com/lib/pq v1.10.9
	github.com/sdvcrx/echo-cache/store/sql v0.3.0
	github.com/stretchr/testify v1.10.0
	modernc.org/sqlite v1.35.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sdvcrx/echo-cache/store v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20250218142911-aa4b98e5adaa // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.61.13 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.2 // indirect
)
