package cache

import (
	"errors"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func createEchoContext(e *echo.Echo, url string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

type dumyAdapter struct {
	mock.Mock
}

func (da *dumyAdapter) Get(key string) ([]byte, error) {
	args := da.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (da *dumyAdapter) Set(key string, val []byte, ttl time.Duration) error {
	args := da.Called(key, val, ttl)
	return args.Error(0)
}

func createDumpAdapter(cacheKey string) *dumyAdapter {
	adapter := new(dumyAdapter)
	if cacheKey != "" {
		adapter.On("Get", cacheKey).Return(([]byte)(nil), nil)
		adapter.On("Set", cacheKey, mock.Anything, mock.Anything).Return(nil)
	}
	return adapter
}

type middlewareTestSuite struct {
	suite.Suite
	enc     Encoder
	e       *echo.Echo
	handler echo.HandlerFunc
}

func (suite *middlewareTestSuite) SetupTest() {
	suite.e = echo.New()
	suite.enc = &JSONEncoder{}
	suite.handler = func(c echo.Context) error {
		c.Response().Header().Set("X-TEST", "OK")
		return c.String(http.StatusOK, "OK")
	}
}

func (suite *middlewareTestSuite) testCacheKey(prefix string, req *http.Request) string {
	return "key"
}

func (suite *middlewareTestSuite) testSkipper(
	skipperFunc middleware.Skipper,
	c echo.Context,
	expected bool,
) {
	suite.Equal(expected, skipperFunc(c))

	adapter := createDumpAdapter("key")

	middleware := CacheWithConfig(CacheConfig{
		Skipper:  skipperFunc,
		Adapter:  adapter,
		CacheKey: suite.testCacheKey,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	if expected {
		// should skip cache middleware
		adapter.AssertNotCalled(suite.T(), "Get", mock.Anything)
		adapter.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
	}
}

func (suite *middlewareTestSuite) TestDefaultSkipper() {
	suite.Run("Skip POST req", func() {
		req := httptest.NewRequest(http.MethodPost, "/cache", nil)
		rec := httptest.NewRecorder()
		c := suite.e.NewContext(req, rec)

		suite.testSkipper(DefaultCacheSkipper, c, true)
	})

	suite.Run("Skip req with range header", func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Range", "bytes=0-1023")
		rec := httptest.NewRecorder()
		c := suite.e.NewContext(req, rec)

		suite.testSkipper(DefaultCacheSkipper, c, true)
	})
}

func (suite *middlewareTestSuite) TestSkipper() {
	path := "/cache"
	skipper := func(c echo.Context) bool {
		return c.Request().URL.Path != path
	}
	c, _ := createEchoContext(suite.e, "/dont-cache-me")

	suite.testSkipper(skipper, c, true)
}

func (suite *middlewareTestSuite) testCanCacheResponse(
	skipperFunc middleware.Skipper,
	expected bool,
	handler echo.HandlerFunc,
) {
	c, _ := createEchoContext(suite.e, "/cache-me")

	adapter := createDumpAdapter("key")

	middleware := CacheWithConfig(CacheConfig{
		CanCacheResponse: skipperFunc,
		Adapter:          adapter,
		CacheKey:         suite.testCacheKey,
	})
	err := middleware(handler)(c)

	suite.NoError(err)
	suite.Equal(expected, skipperFunc(c))

	if expected {
		adapter.AssertCalled(suite.T(), "Get", mock.Anything)
		// should skip cache response
		adapter.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
	}
}

func (suite *middlewareTestSuite) TestDefaultCanCacheResponse() {
	suite.Run("Can cache response", func() {
		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponseSkipper, false, handler)
	})

	suite.Run("Skip response error", func() {
		handler := func(c echo.Context) error {
			return c.String(http.StatusBadRequest, "params missing")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponseSkipper, true, handler)
	})

	suite.Run("Skip set-cookie header", func() {
		handler := func(c echo.Context) error {
			c.SetCookie(&http.Cookie{
				Name:  "uid",
				Value: "1",
			})
			return c.String(http.StatusOK, "OK")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponseSkipper, true, handler)
	})

	suite.Run("Skip response body too large", func() {
		handler := func(c echo.Context) error {
			data := make([]byte, int(2e7), int(2e7))
			return c.Blob(http.StatusOK, "application/octet-stream", data)
		}
		suite.testCanCacheResponse(DefaultCanCacheResponseSkipper, true, handler)
	})
}

func (suite *middlewareTestSuite) TestCachePrefix() {
	prefix := "[CACHE]"
	url := "/api/resource?name=echo"
	c, _ := createEchoContext(suite.e, url)

	key := prefix + "-GET-" + url
	adapter := createDumpAdapter(key)

	middleware := CacheWithConfig(CacheConfig{
		Adapter:     adapter,
		CachePrefix: prefix,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// should call cache middleware with `key` as cache key
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestCacheHit() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	adapter := createDumpAdapter("")
	// mock data
	hdr := http.Header{}
	hdr.Set("X-RESP", "OK")
	resp := NewResponse(201, hdr, []byte("OK"))
	b, err := suite.enc.Marshal(resp)
	suite.NoError(err)
	adapter.On("Get", key).Return(b, nil)

	middleware := CacheWithConfig(CacheConfig{
		Adapter: adapter,
		Encoder: suite.enc,
	})
	err = middleware(suite.handler)(c)
	suite.NoError(err)

	// should cache statusCode, headers and body
	suite.Equal(201, rec.Code)
	suite.Equal("OK", rec.Result().Header.Get("X-RESP"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertNotCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestCacheHeader() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	memory := NewMemoryAdapter(10, TYPE_LRU)

	headerValues := []string{"1", "2", "3"}

	handler := func(c echo.Context) error {
		for _, v := range headerValues {
			c.Response().Header().Add("X-TEST", v)
		}
		return c.String(http.StatusOK, "OK")
	}

	middleware := CacheWithConfig(CacheConfig{
		Adapter: memory,
		Encoder: suite.enc,
	})

	for i := 0; i < 3; i++ {
		err := middleware(handler)(c)
		suite.NoError(err)

		suite.Equal(200, rec.Code)
		suite.Equal(headerValues, rec.Header().Values("X-TEST"))
	}
}

func (suite *middlewareTestSuite) TestSaveHit() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	adapter := createDumpAdapter(key)

	middleware := CacheWithConfig(CacheConfig{
		Adapter: adapter,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// should cache statusCode, headers and body
	suite.Equal(200, rec.Code)
	suite.Equal("OK", rec.Result().Header.Get("X-TEST"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestGetCacheError() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	adapter := createDumpAdapter("")
	adapter.On("Get", key).Return(([]byte)(nil), errors.New("GetCacheError"))
	adapter.On("Set", key, mock.Anything, mock.Anything).Return(nil)

	middleware := CacheWithConfig(CacheConfig{
		Adapter: adapter,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// get cache failed but still can get response
	suite.Equal(200, rec.Code)
	suite.Equal("OK", rec.Result().Header.Get("X-TEST"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func TestCacheMiddleware(t *testing.T) {
	suite.Run(t, new(middlewareTestSuite))
}
