package stats

import (
	"sync"
	"time"
)

// InternalStats is an internal stats recorder
//
// All recording functions are thread-safe.
type InternalStats struct {
	evictionCount    int64
	hitCount         int64
	missCount        int64
	loadSuccessCount int64
	loadErrorCount   int64
	loadTotalTime    time.Duration

	statsLock sync.RWMutex
}

// Eviction increments the number of evictions
func (s *InternalStats) Eviction() {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.evictionCount++
}

// Hit increments the number of hits
func (s *InternalStats) Hit() {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.hitCount++
}

// Miss increments the number of misses
func (s *InternalStats) Miss() {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.missCount++
}

// LoadSuccess increments the number of success loads
func (s *InternalStats) LoadSuccess() {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.loadSuccessCount++
}

// LoadError increments the number of error loads
func (s *InternalStats) LoadError() {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.loadErrorCount++
}

// LoadTime increments the total load time
func (s *InternalStats) LoadTime(loadTime time.Duration) {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.loadTotalTime += loadTime
}

// EvictionCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) EvictionCount() int64 {
	return s.evictionCount
}

// HitCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) HitCount() int64 {
	return s.hitCount
}

// HitRate implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) HitRate() float64 {
	s.statsLock.RLock()
	defer s.statsLock.RUnlock()

	requestCount := s.hitCount + s.missCount
	if requestCount == 0 {
		return 1
	}
	return float64(s.hitCount) / float64(requestCount)
}

// MissCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) MissCount() int64 {
	return s.missCount
}

// MissRate implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) MissRate() float64 {
	s.statsLock.RLock()
	defer s.statsLock.RUnlock()

	requestCount := s.hitCount + s.missCount
	if requestCount == 0 {
		return 0
	}
	return float64(s.missCount) / float64(requestCount)
}

// RequestCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) RequestCount() int64 {
	return s.hitCount + s.missCount
}

// LoadSuccessCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) LoadSuccessCount() int64 {
	return s.loadSuccessCount
}

// LoadErrorCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) LoadErrorCount() int64 {
	return s.loadErrorCount
}

// LoadErrorRate implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) LoadErrorRate() float64 {
	s.statsLock.RLock()
	defer s.statsLock.RUnlock()

	totalLoads := s.loadSuccessCount + s.loadErrorCount
	if totalLoads == 0 {
		return 0
	}
	return float64(s.loadErrorCount) / float64(totalLoads)
}

// LoadCount implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) LoadCount() int64 {
	s.statsLock.RLock()
	defer s.statsLock.RUnlock()
	return s.loadSuccessCount + s.loadErrorCount
}

// LoadTotalTime implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) LoadTotalTime() time.Duration {
	return s.loadTotalTime
}

// AverageLoadPenalty implements the Stats interface. Refer to its documentation for more details
func (s *InternalStats) AverageLoadPenalty() time.Duration {
	s.statsLock.RLock()
	defer s.statsLock.RUnlock()

	totalLoads := s.loadSuccessCount + s.loadErrorCount
	if totalLoads == 0 {
		return 0
	}
	return s.loadTotalTime / time.Duration(totalLoads)
}

// Add adds up two stats stores
func (s *InternalStats) Add(s2 *InternalStats) *InternalStats {
	s.statsLock.RLock()
	defer s.statsLock.RUnlock()
	s2.statsLock.RLock()
	defer s2.statsLock.RUnlock()
	return &InternalStats{
		evictionCount:    s.evictionCount + s2.evictionCount,
		hitCount:         s.hitCount + s2.hitCount,
		missCount:        s.missCount + s2.missCount,
		loadSuccessCount: s.loadSuccessCount + s2.loadSuccessCount,
		loadErrorCount:   s.loadErrorCount + s2.loadErrorCount,
		loadTotalTime:    s.loadTotalTime + s2.loadTotalTime,
	}
}
