# echo-cache

[![Build Status](https://github.com/sdvcrx/echo-cache/actions/workflows/go.yml/badge.svg)](https://github.com/sdvcrx/echo-cache/actions/workflows/go.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/sdvcrx/echo-cache)](https://pkg.go.dev/github.com/sdvcrx/echo-cache)

A simple [echo](https://echo.labstack.com/) Cache middleware.

## Requirements

- Go 1.26 or newer
- Echo v5

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
    Metrics          Metrics
}
```

Echo v5 uses a concrete context type, so custom callbacks receive `*echo.Context`:

```go
import "github.com/labstack/echo/v5"

e.Use(cache.CacheWithConfig(cache.CacheConfig{
  Skipper: func(c *echo.Context) bool {
    return c.Request().Header.Get("X-Skip-Cache") != ""
  },
}))
```

## Migrating from Echo v4

This version supports Echo v5 only. Applications upgrading from Echo v4 must update
their Echo imports to `github.com/labstack/echo/v5` and change custom cache callback
parameters from `echo.Context` to `*echo.Context`.

## LICENSE

MIT
