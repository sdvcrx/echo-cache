package cache

import (
	"errors"
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
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

func (da *dumyAdapter) Get(key string) (*Response, error) {
	args := da.Called(key)
	return args.Get(0).(*Response), args.Error(1)
}

func (da *dumyAdapter) Set(key string, response *Response) error {
	args := da.Called(key, response)
	return args.Error(0)
}

func createDumpAdapter(cacheKey string) *dumyAdapter {
	adapter := new(dumyAdapter)
	if cacheKey != "" {
		adapter.On("Get", cacheKey).Return((*Response)(nil), nil)
		adapter.On("Set", cacheKey, mock.Anything).Return(nil)
	}
	return adapter
}

type middlewareTestSuite struct {
	suite.Suite
	e       *echo.Echo
	handler echo.HandlerFunc
}

func (suite *middlewareTestSuite) SetupTest() {
	suite.e = echo.New()
	suite.handler = func(c echo.Context) error {
		c.Response().Header().Set("X-TEST", "OK")
		return c.String(http.StatusOK, "OK")
	}
}

func (suite *middlewareTestSuite) testCacheKey(prefix string, req *http.Request) string {
	return "key"
}

func (suite *middlewareTestSuite) TestSkipper() {
	path := "/cache"
	skipper := func(c echo.Context) bool {
		return c.Request().URL.Path != path
	}
	c, _ := createEchoContext(suite.e, "/dont-cache-me")
	suite.Equal(true, skipper(c))

	adapter := createDumpAdapter("key")

	middleware := CacheWithConfig(CacheConfig{
		Skipper:  skipper,
		Adapter:  adapter,
		CacheKey: suite.testCacheKey,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// should skip cache middleware
	adapter.AssertNotCalled(suite.T(), "Get", mock.Anything)
	adapter.AssertNotCalled(suite.T(), "Set", mock.Anything, mock.Anything)
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
	adapter.AssertCalled(suite.T(), "Set", key, mock.Anything)
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
	adapter.On("Get", key).Return(resp, nil)

	middleware := CacheWithConfig(CacheConfig{
		Adapter: adapter,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// should cache statusCode, headers and body
	suite.Equal(201, rec.Code)
	suite.Equal("OK", rec.Header().Get("X-RESP"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertNotCalled(suite.T(), "Set", key, mock.Anything)
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
	suite.Equal("OK", rec.Header().Get("X-TEST"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertCalled(suite.T(), "Set", key, mock.Anything)
}

func (suite *middlewareTestSuite) TestGetCacheError() {
	url := "/"
	c, rec := createEchoContext(suite.e, url)

	key := "cache-GET-" + url
	adapter := createDumpAdapter("")
	adapter.On("Get", key).Return((*Response)(nil), errors.New("GetCacheError"))
	adapter.On("Set", key, mock.Anything).Return(nil)

	middleware := CacheWithConfig(CacheConfig{
		Adapter: adapter,
	})
	err := middleware(suite.handler)(c)
	suite.NoError(err)

	// get cache failed but still can get response
	suite.Equal(200, rec.Code)
	suite.Equal("OK", rec.Header().Get("X-TEST"))
	suite.Equal("OK", rec.Body.String())

	// should hit cache
	adapter.AssertCalled(suite.T(), "Get", key)
	adapter.AssertCalled(suite.T(), "Set", key, mock.Anything)
}

func TestCacheMiddleware(t *testing.T) {
	suite.Run(t, new(middlewareTestSuite))
}
