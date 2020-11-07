package loadingcache_test

import (
	"testing"
	"time"

	"github.com/Hartimer/loadingcache"
	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestBasicMethods(t *testing.T) {
	cache := loadingcache.NewGenericCache()
	require.NotNil(t, cache)

	// Getting a key that does not exist should error
	_, err := cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)

	// Invalidating a key that doesn't exist
	cache.Invalidate("a")

	// Adding values
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	// Values exist
	val, err := cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, 1, val)
	val, err = cache.Get("b")
	require.NoError(t, err)
	require.Equal(t, 2, val)
	val, err = cache.Get("c")
	require.NoError(t, err)
	require.Equal(t, 3, val)

	// Invalidate key and get it
	cache.Invalidate("a")
	_, err = cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)
}

func TestExpireAfterWrite(t *testing.T) {
	mockClock := clock.NewMock()
	cache := loadingcache.NewGenericCache(
		loadingcache.WithClock(mockClock),
		loadingcache.ExpireAfterWrite(time.Minute),
	)
	cache.Put("a", 1)
	val, err := cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	// Advance clock up to the expiry threshold
	mockClock.Add(time.Minute)

	// Value should still be returned
	val, err = cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	// Moving just past the threshold should yield no value
	mockClock.Add(1)
	_, err = cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)
}

func TestExpireAfterRead(t *testing.T) {
	mockClock := clock.NewMock()
	cache := loadingcache.NewGenericCache(
		loadingcache.WithClock(mockClock),
		loadingcache.ExpireAfterRead(time.Minute),
	)
	cache.Put("a", 1)
	val, err := cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	// Advance clock up to the expiry threshold
	mockClock.Add(time.Minute)

	// Value should still be returned
	val, err = cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	// Since the value was read, we can move the clock another chunk
	// Advance clock up to the expiry threshold
	mockClock.Add(time.Minute)

	// Value should still be returned
	val, err = cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	// Moving just past the threshold should yield no value
	mockClock.Add(time.Minute + 1)
	_, err = cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)
}
