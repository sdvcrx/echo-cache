module github.com/sdvcrx/echo-cache

replace github.com/sdvcrx/echo-cache/store => ./store

replace github.com/sdvcrx/echo-cache/store/memory => ./store/memory

go 1.26.0

require (
	github.com/labstack/echo/v5 v5.4.4
	github.com/sdvcrx/echo-cache/store v0.3.0
	github.com/sdvcrx/echo-cache/store/memory v0.3.0
	github.com/stretchr/testify v1.11.1
	github.com/vmihailenco/msgpack/v5 v5.4.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/phuslu/lru v1.0.18 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
