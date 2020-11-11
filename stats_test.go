package loadingcache_test

import (
	"context"
	"testing"
	"time"

	"github.com/Hartimer/loadingcache"
	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestStatsHitAndMiss(t *testing.T) {
	matrixTest(t, matrixTestOptions{}, func(t *testing.T, _ context.Context, cache loadingcache.Cache) {
		_, err := cache.Get(1)
		require.Error(t, err)
		require.Equal(t, int64(1), cache.Stats().MissCount())

		cache.Put(1, "a")
		_, err = cache.Get(1)
		require.NoError(t, err)
		require.Equal(t, int64(1), cache.Stats().HitCount())

		require.Equal(t, float64(0.5), cache.Stats().HitRate())
		require.Equal(t, float64(0.5), cache.Stats().MissRate())
		require.Equal(t, int64(2), cache.Stats().RequestCount())
	})
}

func TestLoadTimes(t *testing.T) {
	mockClock := clock.NewMock()
	loadTime := 100 * time.Millisecond
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Clock: mockClock,
			Load: func(key interface{}) (interface{}, error) {
				// Simulating that the loading takes 100ms
				mockClock.Add(loadTime)
				return key, nil
			},
		},
	},
		func(t *testing.T, ctx context.Context, cache loadingcache.Cache) {
			for i := 1; i <= 10; i++ {
				_, err := cache.Get(i)
				require.NoError(t, err)
				require.Equal(t, time.Duration(i)*loadTime, cache.Stats().LoadTotalTime())
			}
			// Since there were no misses, the average load time should match the individual load time
			require.Equal(t, loadTime, cache.Stats().AverageLoadPenalty())
		})
}

func TestLoadSuccessAndError(t *testing.T) {
	loadFunc := &testLoadFunc{}
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Load: loadFunc.LoadFunc,
		},
	},
		func(t *testing.T, _ context.Context, cache loadingcache.Cache) {
			// Clean up after ourselves
			defer func() {
				loadFunc.fail = false
			}()

			// Doing 10 successful loads
			i := 0
			for ; i < 10; i++ {
				_, err := cache.Get(i)
				require.NoError(t, err)
			}
			// Doing 6 error loads
			loadFunc.fail = true
			for ; i < 16; i++ {
				_, err := cache.Get(i)
				require.Error(t, err)
			}

			require.Equal(t, int64(10), cache.Stats().LoadSuccessCount())
			require.Equal(t, int64(6), cache.Stats().LoadErrorCount())
			require.Equal(t, float64(0.375), cache.Stats().LoadErrorRate())
		})
}
