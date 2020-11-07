package loadingcache_test

import (
	"errors"
	"fmt"
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

	// Invalidate multiple keys at once
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Invalidate("a", "b")
	_, err = cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)
	_, err = cache.Get("b")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)

	// Invalidate all keys
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.InvalidateAll()
	_, err = cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)
	_, err = cache.Get("b")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)

}

func TestExpireAfterWrite(t *testing.T) {
	mockClock := clock.NewMock()
	cache := loadingcache.NewGenericCache(
		loadingcache.Clock(mockClock),
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
		loadingcache.Clock(mockClock),
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

func TestLoadFunc(t *testing.T) {
	loadFunc := &testLoadFunc{}
	cache := loadingcache.NewGenericCache(loadingcache.Load(loadFunc.LoadFunc))

	// Getting a value that does not exist should load it
	val, err := cache.Get("a")
	require.NoError(t, err)
	require.Equal(t, "a", val)

	// Getting a value that the loader fails to error should propagate the error
	loadFunc.fail = true
	_, err = cache.Get("b")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failing on request")

	// Adding the value manually should succeeed
	cache.Put("b", "true")
	val, err = cache.Get("b")
	require.NoError(t, err)
	require.Equal(t, "true", val)

	// After invalidating, getting should fail again
	cache.Invalidate("b")
	_, err = cache.Get("b")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failing on request")
}

func TestMaxSize(t *testing.T) {
	cache := loadingcache.NewGenericCache(loadingcache.MaxSize(1))

	// With a capacity of one element, adding a second element
	// should remove the first
	cache.Put("a", 1)
	cache.Put("b", 2)

	_, err := cache.Get("a")
	require.Error(t, err)
	require.Equal(t, loadingcache.ErrKeyNotFound, err)

	val, err := cache.Get("b")
	require.NoError(t, err)
	require.Equal(t, 2, val)
}

func TestRemovalListeners(t *testing.T) {
	mockClock := clock.NewMock()
	removalListener := &testRemovalListener{}
	removalListener2 := &testRemovalListener{}
	cache := loadingcache.NewGenericCache(
		loadingcache.Clock(mockClock),
		loadingcache.ExpireAfterRead(time.Minute),
		loadingcache.ExpireAfterWrite(2*time.Minute),
		loadingcache.MaxSize(1),
		loadingcache.RemovalListener(removalListener.Listener),
		loadingcache.RemovalListener(removalListener2.Listener),
	)

	// Removal due to replacement
	cache.Put("a", 10)
	cache.Put("a", 1)
	lastNotification := removalListener.lastRemovalNotification
	lastNotification2 := removalListener2.lastRemovalNotification
	require.Equal(t, loadingcache.RemovalReasonReplaced, lastNotification.Reason)
	require.Equal(t, loadingcache.RemovalReasonReplaced, lastNotification2.Reason)
	require.Equal(t, "a", lastNotification.Key)
	require.Equal(t, 10, lastNotification.Value)

	// Removal due to size
	cache.Put("b", 2)
	lastNotification = removalListener.lastRemovalNotification
	lastNotification2 = removalListener2.lastRemovalNotification
	require.Equal(t, loadingcache.RemovalReasonSize, lastNotification.Reason)
	require.Equal(t, loadingcache.RemovalReasonSize, lastNotification2.Reason)
	require.Equal(t, "a", lastNotification.Key)
	require.Equal(t, 1, lastNotification.Value)

	// Removal due to read expiration
	mockClock.Add(time.Minute + 1)
	// We don't care about the value or error, we just want to trigger the eviction
	_, _ = cache.Get("b")
	lastNotification = removalListener.lastRemovalNotification
	lastNotification2 = removalListener2.lastRemovalNotification
	require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification.Reason)
	require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification2.Reason)
	require.Equal(t, "b", lastNotification.Key)
	require.Equal(t, 2, lastNotification.Value)

	// Removal due to write expiration
	cache.Put("b", 3)
	mockClock.Add(time.Minute)
	// Doing a read to refresh the expiry
	_, _ = cache.Get("b")
	mockClock.Add(time.Minute)
	// Doing a another read to refresh the expiry
	_, _ = cache.Get("b")
	mockClock.Add(1)
	// We don't care about the value or error, we just want to trigger the eviction
	_, _ = cache.Get("b")
	lastNotification = removalListener.lastRemovalNotification
	lastNotification2 = removalListener2.lastRemovalNotification
	require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification.Reason)
	require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification2.Reason)
	require.Equal(t, "b", lastNotification.Key)
	require.Equal(t, 3, lastNotification.Value)
}

type testRemovalListener struct {
	lastRemovalNotification loadingcache.RemovalNotification
}

func (t *testRemovalListener) Listener(notification loadingcache.RemovalNotification) {
	t.lastRemovalNotification = notification
}

// testLoadFunc provides a configurable loading function that may fail
type testLoadFunc struct {
	fail bool
}

func (t *testLoadFunc) LoadFunc(key interface{}) (interface{}, error) {
	if t.fail {
		return nil, errors.New("failing on request")
	}
	return fmt.Sprint(key), nil
}

func ExampleSimpleUsage() {
	cache := loadingcache.NewGenericCache()

	// Addign some values and reading them
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)
	val1, _ := cache.Get("a") // Don't forget to check for errors
	fmt.Printf("%v\n", val1)
	val2, _ := cache.Get("b") // Don't forget to check for errors
	fmt.Printf("%v\n", val2)

	// Getting a value that does not exist
	_, err := cache.Get("d")
	if err == loadingcache.ErrKeyNotFound {
		fmt.Println("That key does not exist")
	}

	// Evicting
	cache.Invalidate("a")
	cache.Invalidate("b", "c")
	cache.InvalidateAll()

	// Output: 1
	// 2
	// That key does not exist
}

func ExampleAdvancedUsage() {
	cache := loadingcache.NewGenericCache(
		loadingcache.MaxSize(2),
		loadingcache.ExpireAfterRead(2*time.Minute),
		loadingcache.ExpireAfterWrite(time.Minute),
		loadingcache.RemovalListener(func(notification loadingcache.RemovalNotification) {
			fmt.Printf("Entry removed due to %s\n", notification.Reason)
		}),
		loadingcache.Load(func(key interface{}) (interface{}, error) {
			fmt.Printf("Loading key %v\n", key)
			return fmt.Sprint(key), nil
		}),
	)

	cache.Put(1, "1")
	val1, _ := cache.Get(1)
	fmt.Printf("%v\n", val1)

	val2, _ := cache.Get(2)
	fmt.Printf("%v\n", val2)

	val3, _ := cache.Get(3)
	fmt.Printf("%v\n", val3)

	// Output: 1
	// Loading key 2
	// 2
	// Loading key 3
	// Entry removed due to SIZE
	// 3
}
