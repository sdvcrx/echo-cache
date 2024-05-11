module github.com/sdvcrx/echo-cache/store/memory

replace github.com/sdvcrx/echo-cache/store => ../

go 1.22

require (
	github.com/phuslu/lru v1.0.15
	github.com/sdvcrx/echo-cache/store v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
