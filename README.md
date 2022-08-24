# echo-cache

A simple [echo](https://echo.labstack.com/) Cache middleware.

## Install

```bash
go get -u github.com/sdvcrx/echo-cache
```

## Usage

### Basic Usage

```go
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
    Skipper       middleware.Skipper
    CachePrefix   string
    CacheKey      CacheKeyFunc
    CacheDuration time.Duration
    Adapter       CacheAdapter
}
```

## LICENSE

MIT
