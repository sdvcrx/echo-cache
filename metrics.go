package cache

type Metrics interface {
	// The total number of cache hits
	CacheHits()
	// The total number of cache misses
	CacheMisses()
	// The current size of the cache in bytes
	CacheSize(size float64)
	// The time it takes for the middleware to retrieve data from the cache
	CacheLatency(latency float64)
	// The total number of errors encountered while interacting with the cache
	CacheError()
}

type dummyMetrics struct{}

func (m *dummyMetrics) CacheHits() {
	// empty
}

func (m *dummyMetrics) CacheMisses() {
	// empty
}

func (m *dummyMetrics) CacheSize(size float64) {
	// empty
}

func (m *dummyMetrics) CacheLatency(latency float64) {
	// empty
}

func (m *dummyMetrics) CacheError() {
	// empty
}
