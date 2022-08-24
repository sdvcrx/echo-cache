package cache

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CacheKeyFunc func(prefix string, req *http.Request) string

type CacheConfig struct {
	Skipper       middleware.Skipper
	CachePrefix   string
	CacheKey      CacheKeyFunc
	CacheDuration time.Duration
	Adapter       CacheAdapter
}

func DefaultCacheKey(prefix string, req *http.Request) string {
	return fmt.Sprintf("%s-%s-%s", prefix, req.Method, req.URL)
}

var (
	DefaultCachePrefix   = "cache"
	DefaultCacheDuration = time.Duration(0)
	DefaultCacheConfig   = CacheConfig{
		Skipper:       middleware.DefaultSkipper,
		CachePrefix:   DefaultCachePrefix,
		CacheDuration: DefaultCacheDuration,
		CacheKey:      DefaultCacheKey,
	}
)

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func CacheWithConfig(config CacheConfig) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = DefaultCacheConfig.Skipper
	}
	if config.CachePrefix == "" {
		config.CachePrefix = DefaultCachePrefix
	}
	if config.CacheKey == nil {
		config.CacheKey = DefaultCacheKey
	}
	if config.CacheDuration == 0 {
		config.CacheDuration = DefaultCacheDuration
	}
	if config.Adapter == nil {
		// Default Adapter
		config.Adapter = NewMemoryAdapter(100, TYPE_LRU)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			// before response
			req := c.Request()
			key := config.CacheKey(config.CachePrefix, req)

			cachedResponse, err := config.Adapter.Get(key)

			if err != nil {
				c.Logger().Warnf("Failed to get cache, err=%s", err)
			} else if cachedResponse != nil {
				for k, v := range cachedResponse.Headers {
					c.Response().Header().Set(k, strings.Join(v, ","))
				}
				c.Response().WriteHeader(cachedResponse.StatusCode)
				_, err = c.Response().Write(cachedResponse.Body)
				return err
			}

			// copy from https://github.com/labstack/echo/blob/master/middleware/body_dump.go
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer

			// start
			if err := next(c); err != nil {
				c.Error(err)
			}

			// don't cache status code != 200
			// TODO add canCache
			if c.Response().Status >= 400 {
				return nil
			}
			// cache it here
			resp := NewResponse(writer.statusCode, writer.Header(), resBody.Bytes())
			if err = config.Adapter.Set(key, resp, config.CacheDuration); err != nil {
				c.Logger().Warnf("Failed to save cache, key=%s err=%s", key, err)
			}
			return nil
		}
	}
}

func Cache() echo.MiddlewareFunc {
	return CacheWithConfig(DefaultCacheConfig)
}
