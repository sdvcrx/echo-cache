module github.com/sdvcrx/echo-cache/store/bolt

replace github.com/sdvcrx/echo-cache/store => ../

go 1.23

require (
	github.com/sdvcrx/echo-cache/store v0.3.0
	github.com/stretchr/testify v1.10.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	go.etcd.io/bbolt v1.4.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
