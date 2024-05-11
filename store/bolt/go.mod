module github.com/sdvcrx/echo-cache/store/bolt

replace github.com/sdvcrx/echo-cache/store => ../

go 1.22

require (
	github.com/sdvcrx/echo-cache/store v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.9.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	go.etcd.io/bbolt v1.3.10
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
