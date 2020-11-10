package stats_test

import (
	"testing"
	"time"

	"github.com/Hartimer/loadingcache"
	"github.com/Hartimer/loadingcache/internal/stats"
	"github.com/stretchr/testify/require"
)

// Validate that the internal stats implementation respects the public interface.
//
// We do this check here to avoid cyclic imports.
var _ loadingcache.Stats = &stats.InternalStats{}

func TestBasicIncrementers(t *testing.T) {
	s := &stats.InternalStats{}
	for i := int64(1); i <= 10; i++ {
		s.Hit()
		require.Equal(t, i, s.HitCount())
		s.Miss()
		require.Equal(t, i, s.MissCount())
		s.LoadSuccess()
		require.Equal(t, i, s.LoadSuccessCount())
		s.LoadError()
		require.Equal(t, i, s.LoadErrorCount())
		s.Eviction()
		require.Equal(t, i, s.EvictionCount())
		s.LoadTime(time.Minute)
		require.Equal(t, time.Duration(i)*time.Minute, s.LoadTotalTime())
	}
}

func TestRates(t *testing.T) {
	s := &stats.InternalStats{}

	// Hit rate
	for i := 0; i < 12; i++ {
		s.Hit()
		if i%4 == 0 {
			s.Miss()
		}
	}
	require.Equal(t, float64(0.8), s.HitRate())
	require.Equal(t, float64(0.2), s.MissRate())

	// Load rate
	for i := 0; i < 12; i++ {
		s.LoadSuccess()
		if i%4 == 0 {
			s.LoadError()
		}
	}
	require.Equal(t, float64(0.2), s.LoadErrorRate())
	require.Equal(t, int64(15), s.LoadCount())

	// Average load penalty
	// We know the total amount of loads is 15 (see above). Let's say altogether they
	// took 20 minutes.
	s.LoadTime(20 * time.Minute)
	require.Equal(t, time.Minute+(20*time.Second), s.AverageLoadPenalty())
}
