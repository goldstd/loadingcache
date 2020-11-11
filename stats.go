package loadingcache

import "time"

// Stats exposes cache relevant metrics.
//
// Be aware that this interface may be exposing a live stats collector, and as such
// if you manually calculate rates, values may differ if calls to the cache have occurred
// between calls.
type Stats interface {
	// EvictionCount is the number of times an entry has been evicted
	EvictionCount() int64

	// HitCount the number of times Cache lookup methods have returned a cached value
	HitCount() int64

	// HitRate is the ratio of cache requests which were hits. This is defined as
	// hitCount / requestCount, or 1.0 when requestCount == 0
	HitRate() float64

	// MissCount is the number of times Cache lookup methods have returned an uncached
	// (newly loaded) value
	MissCount() int64

	// MissRate is the ratio of cache requests which were misses. This is defined as
	// missCount / requestCount, or 0.0 when requestCount == 0
	MissRate() float64

	// RequestCount is the number of times Cache lookup methods have returned either a cached or
	// uncached value. This is defined as hitCount + missCount
	RequestCount() int64

	// LoadSuccessCount is the number of times Cache lookup methods have successfully
	// loaded a new value
	LoadSuccessCount() int64

	// LoadErrorCount is the number of times Cache lookup methods threw an exception while loading
	// a new value
	LoadErrorCount() int64

	// LoadErrorRate is the ratio of cache loading attempts which threw exceptions.
	// This is defined as loadExceptionCount / (loadSuccessCount + loadExceptionCount), or 0.0 when
	// loadSuccessCount + loadExceptionCount == 0
	LoadErrorRate() float64

	// LoadCount the total number of times that Cache lookup methods attempted to load new values.
	// This includes both successful load operations, as well as those that threw exceptions.
	// This is defined as loadSuccessCount + loadExceptionCount
	LoadCount() int64

	// LoadTotalTime is the total duration the cache has spent loading new values
	LoadTotalTime() time.Duration

	// AverageLoadPenalty is the average duration spent loading new values. This is defined as
	// totalLoadTime / (loadSuccessCount + loadExceptionCount).
	AverageLoadPenalty() time.Duration
}
