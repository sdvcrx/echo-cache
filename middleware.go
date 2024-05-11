package cache

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sdvcrx/echo-cache/store"
	memorystore "github.com/sdvcrx/echo-cache/store/memory"
)

type CacheKeyFunc func(prefix string, req *http.Request) string

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

func DefaultCacheKey(prefix string, req *http.Request) string {
	return fmt.Sprintf("%s-%s-%s", prefix, req.Method, req.URL)
}

// Cache default skipper only cache GET/HEAD method
// and headers not contain `Range`
func DefaultCacheSkipper(c echo.Context) bool {
	method := c.Request().Method
	// Request must use GET or HEAD method
	if method != http.MethodGet && method != http.MethodHead {
		return true
	}
	// Request must not contain the `Range` header
	if c.Request().Header.Get("range") != "" {
		return true
	}
	return false
}

const (
	SizeKB int64 = 1024
	SizeMB int64 = 1024 * SizeKB
)

// Default canCacheResponse skipper will skip response cache if:
// - response status code not in (200, 301, 308)
// - response headers not contains `set-cookie`
func DefaultCanCacheResponseSkipper(c echo.Context) bool {
	resp := c.Response()

	// Response status code must be 200, 301, or 308
	if resp.Status != http.StatusOK &&
		resp.Status != http.StatusMovedPermanently &&
		resp.Status != http.StatusPermanentRedirect {
		return true
	}

	// Response must not contain the `set-cookie` header.
	if resp.Header().Get(echo.HeaderSetCookie) != "" {
		return true
	}

	// Response must not exceed 10MB in content length.
	if resp.Size > 10*SizeMB {
		return true
	}

	return false
}

var (
	DefaultCachePrefix   = "cache"
	DefaultCacheDuration = time.Duration(0)
	DefaultCacheConfig   = CacheConfig{
		Skipper:          DefaultCacheSkipper,
		CanCacheResponse: DefaultCanCacheResponseSkipper,
		CachePrefix:      DefaultCachePrefix,
		CacheDuration:    DefaultCacheDuration,
		CacheKey:         DefaultCacheKey,
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
	if config.CanCacheResponse == nil {
		config.CanCacheResponse = DefaultCacheConfig.CanCacheResponse
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
	if config.Store == nil {
		config.Store = memorystore.New(1024)
	}
	if config.Encoder == nil {
		config.Encoder = &MsgpackEncoder{}
	}
	if config.Metrics == nil {
		config.Metrics = &dummyMetrics{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				config.Metrics.CacheMisses()
				return next(c)
			}

			// before response
			start := time.Now()
			req := c.Request()
			key := config.CacheKey(config.CachePrefix, req)

			cached, err := config.Store.Get(key)

			if err != nil {
				config.Metrics.CacheError()
				c.Logger().Errorf("[echo-cache] Failed to get cache, err=%s", err)
			} else if cached != nil {
				var cachedResponse Response
				if err := config.Encoder.Unmarshal(cached, &cachedResponse); err != nil {
					config.Metrics.CacheError()
					c.Logger().Errorf("[echo-cache] Failed to unmarshal response, err=%s", err)
					return nil
				}

				maps.Copy(c.Response().Header(), cachedResponse.Headers)
				c.Response().WriteHeader(cachedResponse.StatusCode)
				_, err = c.Response().Write(cachedResponse.Body)
				if err != nil {
					c.Logger().Errorf("[echo-cache] Failed to write response, err=%s", err)
				}
				config.Metrics.CacheHits()
				config.Metrics.CacheLatency(float64(time.Since(start).Seconds()))
				return nil
			}

			config.Metrics.CacheMisses()

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
			// https://vercel.com/docs/concepts/functions/edge-functions/edge-caching#what-is-cached
			if config.CanCacheResponse(c) {
				return nil
			}
			// cache it here
			resp := NewResponse(writer.statusCode, writer.Header(), resBody.Bytes())
			b, err := config.Encoder.Marshal(resp)
			if err != nil {
				c.Logger().Errorf("[echo-cache] Failed to marshal response, err=%s", err)
				return nil
			}
			config.Metrics.CacheSize(float64(len(b)))
			if err = config.Store.Set(key, b, config.CacheDuration); err != nil {
				c.Logger().Errorf("[echo-cache] Failed to save cache, key=%s err=%s", key, err)
			}
			return nil
		}
	}
}

func Cache() echo.MiddlewareFunc {
	return CacheWithConfig(DefaultCacheConfig)
}
