# echo-cache

[![Build Status](https://github.com/sdvcrx/echo-cache/actions/workflows/go.yml/badge.svg)](https://github.com/sdvcrx/echo-cache/actions/workflows/go.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/sdvcrx/echo-cache)](https://pkg.go.dev/github.com/sdvcrx/echo-cache)

A simple [echo](https://echo.labstack.com/) Cache middleware.

## Install

```bash
go get -u github.com/sdvcrx/echo-cache
```

## Usage

### Basic Usage

```go
import (
    "github.com/sdvcrx/echo-cache"
)

// ...

e.Use(cache.Cache())
```

### Custom Configuration

```go
e.Use(cache.CacheWithConfig(cache.CacheConfig{
  // ...
}))
```

Configuration:

```go
type CacheConfig struct {
    Skipper          middleware.Skipper
    CanCacheResponse middleware.Skipper
    CachePrefix      string
    CacheKey         CacheKeyFunc
    CacheDuration    time.Duration
    Store            store.Store
    Encoder          Encoder
}
```

## LICENSE

MIT
