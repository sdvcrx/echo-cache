module github.com/sdvcrx/echo-cache

replace github.com/sdvcrx/echo-cache/store => ./store

replace github.com/sdvcrx/echo-cache/store/memory => ./store/memory

go 1.23

require (
	github.com/labstack/echo/v4 v4.13.3
	github.com/sdvcrx/echo-cache/store v0.3.0
	github.com/sdvcrx/echo-cache/store/memory v0.3.0
	github.com/stretchr/testify v1.10.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/phuslu/lru v1.0.18 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
