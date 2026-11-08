package cache

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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
// Based on https://vercel.com/docs/edge-cache#cacheable-response-criteria
func DefaultCacheSkipper(c *echo.Context) bool {
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

// Retrurn true if cacheControl contain the `private`, `no-cache` or `no-store` directives.
func CacheControlNoCache(cacheControl string) bool {
	if cacheControl == "" {
		return false
	}

	iter := strings.SplitSeq(cacheControl, ",")
	for directive := range iter {
		dir := strings.TrimSpace(directive)
		if dir == "private" || dir == "no-cache" || dir == "no-store" {
			return true
		}
	}

	return false
}

// Default canCacheResponse will skip response cache if:
// - Response status code not in (200, 301, 308)
// - Response headers not contains `set-cookie`
// - Response does not exceed 10MB in content length
// - Response does not contain the `private`, `no-cache` or `no-store` directives in the Cache-Control header.
// Based on https://vercel.com/docs/edge-cache#cacheable-response-criteria
func DefaultCanCacheResponse(c *echo.Context) bool {
	resp, err := echo.UnwrapResponse(c.Response())
	if err != nil {
		return false
	}

	// Response status code must be 200, 301, or 308
	if resp.Status != http.StatusOK &&
		resp.Status != http.StatusMovedPermanently &&
		resp.Status != http.StatusPermanentRedirect {
		return false
	}

	// Response must not contain the `set-cookie` header.
	if resp.Header().Get(echo.HeaderSetCookie) != "" {
		return false
	}

	// Response does not contain the `private`, `no-cache` or `no-store` directives in the Cache-Control header.
	respCacheControl := resp.Header().Get(echo.HeaderCacheControl)
	if CacheControlNoCache(respCacheControl) {
		return false
	}

	// Response must not exceed `10MB` in content length.
	if resp.Size > 10*SizeMB {
		return false
	}

	return true
}

var (
	DefaultCachePrefix   = "cache"
	DefaultCacheDuration = time.Duration(0)
	DefaultCacheConfig   = CacheConfig{
		Skipper:          DefaultCacheSkipper,
		CanCacheResponse: DefaultCanCacheResponse,
		CachePrefix:      DefaultCachePrefix,
		CacheDuration:    DefaultCacheDuration,
		CacheKey:         DefaultCacheKey,
	}
)

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if err != nil && errors.Is(err, http.ErrNotSupported) {
		panic(errors.New("response writer flushing is not supported"))
	}
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *bodyDumpResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// CacheWithConfig returns cache middleware with the given configuration.
func CacheWithConfig(config CacheConfig) echo.MiddlewareFunc {
	mw, err := config.ToMiddleware()
	if err != nil {
		panic(err)
	}
	return mw
}

// ToMiddleware converts CacheConfig to middleware or returns an error for invalid configuration.
func (config CacheConfig) ToMiddleware() (echo.MiddlewareFunc, error) {
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
		return func(c *echo.Context) error {
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
				c.Logger().Error("echo-cache: failed to get cache", "error", err)
			} else if cached != nil {
				var cachedResponse Response
				if err := config.Encoder.Unmarshal(cached, &cachedResponse); err != nil {
					config.Metrics.CacheError()
					c.Logger().Error("echo-cache: failed to unmarshal response", "error", err)
					return nil
				}

				maps.Copy(c.Response().Header(), cachedResponse.Headers)
				c.Response().WriteHeader(cachedResponse.StatusCode)
				_, err = c.Response().Write(cachedResponse.Body)
				if err != nil {
					c.Logger().Error("echo-cache: failed to write response", "error", err)
				}
				config.Metrics.CacheHits()
				config.Metrics.CacheLatency(float64(time.Since(start).Seconds()))
				return nil
			}

			config.Metrics.CacheMisses()

			resBody := bufferPool.Get().(*bytes.Buffer)
			defer func() {
				resBody.Reset()
				bufferPool.Put(resBody)
			}()

			// copy from https://github.com/labstack/echo/blob/master/middleware/body_dump.go
			mw := io.MultiWriter(c.Response(), resBody)
			writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response()}
			c.SetResponse(writer)

			// start
			if err := next(c); err != nil {
				return err
			}

			// check if can cache response
			if !config.CanCacheResponse(c) {
				return nil
			}

			// cache it here
			resp, err := echo.UnwrapResponse(c.Response())
			if err != nil {
				config.Metrics.CacheError()
				c.Logger().Error("echo-cache: failed to unwrap response", "error", err)
				return nil
			}
			cachedResponse := NewResponse(resp.Status, writer.Header(), resBody.Bytes())
			b, err := config.Encoder.Marshal(cachedResponse)
			if err != nil {
				c.Logger().Error("echo-cache: failed to marshal response", "error", err)
				return nil
			}
			config.Metrics.CacheSize(float64(len(b)))
			if err = config.Store.Set(key, b, config.CacheDuration); err != nil {
				c.Logger().Error("echo-cache: failed to save cache", "key", key, "error", err)
			}
			return nil
		}
	}, nil
}

func Cache() echo.MiddlewareFunc {
	return CacheWithConfig(DefaultCacheConfig)
}
