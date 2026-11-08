package cache

import (
	"errors"
	"sync"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func createEchoContext(e *echo.Echo, url string) (*echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

type dumyStore struct {
	mock.Mock
}

func (da *dumyStore) Get(key string) ([]byte, error) {
	args := da.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (da *dumyStore) Set(key string, val []byte, ttl time.Duration) error {
	args := da.Called(key, val, ttl)
	return args.Error(0)
}

type memoryStore struct {
	data sync.Map
}

func (m *memoryStore) Get(key string) ([]byte, error) {
	v, _ := m.data.Load(key)
	if v == nil {
		return nil, nil
	}
	return v.([]byte), nil
}

func (m *memoryStore) Set(key string, val []byte, ttl time.Duration) error {
	m.data.Store(key, val)
	return nil
}

func createDumpStore(cacheKey string) *dumyStore {
	store := new(dumyStore)
	if cacheKey != "" {
		store.On("Get", cacheKey).Return(([]byte)(nil), nil)
		store.On("Set", cacheKey, mock.Anything, mock.Anything).Return(nil)
	}
	return store
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
	suite.handler = func(c *echo.Context) error {
		c.Response().Header().Set("X-TEST", "OK")
		return c.String(http.StatusOK, "OK")
	}
}

func (suite *middlewareTestSuite) testCacheKey(prefix string, req *http.Request) string {
	return "key"
}

func (suite *middlewareTestSuite) testSkipper(
	skipperFunc middleware.Skipper,
	c *echo.Context,
	expected bool,
) {
	suite.Equal(expected, skipperFunc(c))

	store := createDumpStore("key")

	middleware := CacheWithConfig(CacheConfig{
		Skipper:  skipperFunc,
		Store:    store,
		CacheKey: suite.testCacheKey,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	if expected {
		// should skip cache middleware
		store.AssertNotCalled(suite.T(), "Get", mock.Anything)
		store.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
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
	skipper := func(c *echo.Context) bool {
		return c.Request().URL.Path != path
	}
	c, _ := createEchoContext(suite.e, "/dont-cache-me")

	suite.testSkipper(skipper, c, true)
}

func (suite *middlewareTestSuite) testCanCacheResponse(
	canCacheRespFunc middleware.Skipper,
	expected bool,
	handler echo.HandlerFunc,
) {
	c, _ := createEchoContext(suite.e, "/cache-me")

	store := createDumpStore("key")

	middleware := CacheWithConfig(CacheConfig{
		CanCacheResponse: canCacheRespFunc,
		Store:            store,
		CacheKey:         suite.testCacheKey,
	})
	err := middleware(handler)(c)

	suite.NoError(err)
	suite.Equal(expected, canCacheRespFunc(c))

	if !expected {
		store.AssertCalled(suite.T(), "Get", mock.Anything)
		// should skip cache response
		store.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
	}
}

func (suite *middlewareTestSuite) TestDefaultCanCacheResponse() {
	suite.Run("Can cache response", func() {
		handler := func(c *echo.Context) error {
			return c.String(http.StatusOK, "OK")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponse, true, handler)
	})

	suite.Run("Skip response error", func() {
		handler := func(c *echo.Context) error {
			return c.String(http.StatusBadRequest, "params missing")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponse, false, handler)
	})

	suite.Run("Skip set-cookie header", func() {
		handler := func(c *echo.Context) error {
			c.SetCookie(&http.Cookie{
				Name:  "uid",
				Value: "1",
			})
			return c.String(http.StatusOK, "OK")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponse, false, handler)
	})

	suite.Run("Skip cache-control header", func() {
		handler := func(c *echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, "private")
			return c.String(http.StatusOK, "ok")
		}
		suite.testCanCacheResponse(DefaultCanCacheResponse, false, handler)
	})

	suite.Run("Skip response body too large", func() {
		handler := func(c *echo.Context) error {
			data := make([]byte, int(2e7))
			return c.Blob(http.StatusOK, "application/octet-stream", data)
		}
		suite.testCanCacheResponse(DefaultCanCacheResponse, false, handler)
	})
}

func (suite *middlewareTestSuite) TestCachePrefix() {
	prefix := "[CACHE]"
	url := "/api/resource?name=echo"
	c, _ := createEchoContext(suite.e, url)

	key := prefix + "-GET-" + url
	store := createDumpStore(key)

	middleware := CacheWithConfig(CacheConfig{
		Store:       store,
		CachePrefix: prefix,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// should call cache middleware with `key` as cache key
	store.AssertCalled(suite.T(), "Get", key)
	store.AssertCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestCacheHit() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	store := createDumpStore("")
	// mock data
	hdr := http.Header{}
	hdr.Set("X-RESP", "OK")
	resp := NewResponse(201, hdr, []byte("OK"))
	b, err := suite.enc.Marshal(resp)
	suite.NoError(err)
	store.On("Get", key).Return(b, nil)

	middleware := CacheWithConfig(CacheConfig{
		Store:   store,
		Encoder: suite.enc,
	})
	err = middleware(suite.handler)(c)
	suite.NoError(err)

	// should cache statusCode, headers and body
	suite.Equal(201, rec.Code)
	suite.Equal("OK", rec.Result().Header.Get("X-RESP"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	store.AssertCalled(suite.T(), "Get", key)
	store.AssertNotCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestCacheHeader() {
	url := "/"
	store := &memoryStore{}

	headerValues := []string{"1", "2", "3"}

	handler := func(c *echo.Context) error {
		for _, v := range headerValues {
			c.Response().Header().Add("X-TEST", v)
		}
		return c.String(http.StatusOK, "OK")
	}

	middleware := CacheWithConfig(CacheConfig{
		Store:   store,
		Encoder: suite.enc,
	})

	for range 3 {
		c, rec := createEchoContext(suite.e, url)
		err := middleware(handler)(c)
		suite.NoError(err)

		suite.Equal(200, rec.Code)
		suite.Equal(headerValues, rec.Header().Values("X-TEST"))
	}
}

func (suite *middlewareTestSuite) TestImplicitOKStatusIsCached() {
	store := &memoryStore{}
	middleware := CacheWithConfig(CacheConfig{
		Store:   store,
		Encoder: suite.enc,
	})

	c, rec := createEchoContext(suite.e, "/implicit-status")
	err := middleware(func(c *echo.Context) error {
		_, err := c.Response().Write([]byte("OK"))
		return err
	})(c)
	suite.NoError(err)
	suite.Equal(http.StatusOK, rec.Code)

	handlerCalled := false
	c, rec = createEchoContext(suite.e, "/implicit-status")
	err = middleware(func(c *echo.Context) error {
		handlerCalled = true
		return nil
	})(c)
	suite.NoError(err)
	suite.False(handlerCalled)
	suite.Equal(http.StatusOK, rec.Code)
	suite.Equal("OK", rec.Body.String())
}

func (suite *middlewareTestSuite) TestResponseWriterCanBeUnwrapped() {
	store := createDumpStore("key")
	unwrapped := false
	middleware := CacheWithConfig(CacheConfig{
		Store:    store,
		CacheKey: suite.testCacheKey,
		CanCacheResponse: func(c *echo.Context) bool {
			_, err := echo.UnwrapResponse(c.Response())
			unwrapped = err == nil
			return false
		},
	})

	c, _ := createEchoContext(suite.e, "/unwrap")
	err := middleware(suite.handler)(c)
	suite.NoError(err)
	suite.True(unwrapped)
	store.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestHandlerErrorIsReturnedWithoutCaching() {
	store := createDumpStore("key")
	expectedErr := errors.New("handler failed")
	middleware := CacheWithConfig(CacheConfig{
		Store:    store,
		CacheKey: suite.testCacheKey,
	})

	c, _ := createEchoContext(suite.e, "/handler-error")
	err := middleware(func(c *echo.Context) error {
		return expectedErr
	})(c)
	suite.ErrorIs(err, expectedErr)
	store.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestUnwrapFailureSkipsCaching() {
	store := createDumpStore("key")
	middleware := CacheWithConfig(CacheConfig{
		Store:            store,
		CacheKey:         suite.testCacheKey,
		CanCacheResponse: func(c *echo.Context) bool { return true },
	})

	c, _ := createEchoContext(suite.e, "/unwrap-error")
	err := middleware(func(c *echo.Context) error {
		c.SetResponse(httptest.NewRecorder())
		return c.String(http.StatusOK, "OK")
	})(c)
	suite.NoError(err)
	store.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestSaveHit() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	store := createDumpStore(key)

	middleware := CacheWithConfig(CacheConfig{
		Store: store,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// should cache statusCode, headers and body
	suite.Equal(200, rec.Code)
	suite.Equal("OK", rec.Result().Header.Get("X-TEST"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	store.AssertCalled(suite.T(), "Get", key)
	store.AssertCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func (suite *middlewareTestSuite) TestGetCacheError() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	store := createDumpStore("")
	store.On("Get", key).Return(([]byte)(nil), errors.New("GetCacheError"))
	store.On("Set", key, mock.Anything, mock.Anything).Return(nil)

	middleware := CacheWithConfig(CacheConfig{
		Store: store,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// get cache failed but still can get response
	suite.Equal(200, rec.Code)
	suite.Equal("OK", rec.Result().Header.Get("X-TEST"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	store.AssertCalled(suite.T(), "Get", key)
	store.AssertCalled(suite.T(), "Set", key, mock.Anything, mock.Anything)
}

func TestCacheMiddleware(t *testing.T) {
	suite.Run(t, new(middlewareTestSuite))
}

func TestCacheConfigToMiddleware(t *testing.T) {
	mw, err := (CacheConfig{}).ToMiddleware()
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	calls := 0
	handler := mw(func(c *echo.Context) error {
		calls++
		return c.String(http.StatusOK, "cached response")
	})
	for range 2 {
		c, rec := createEchoContext(e, "/to-middleware")
		if err := handler(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK || rec.Body.String() != "cached response" {
			t.Fatalf("unexpected response: status=%d body=%q", rec.Code, rec.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("handler called %d times, want 1", calls)
	}
}
