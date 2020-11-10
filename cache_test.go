package loadingcache_test

import (
	"testing"
	"time"

	"github.com/Hartimer/loadingcache"
	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestBasicMethods(t *testing.T) {
	matrixTest(t, matrixTestOptions{}, func(t *testing.T, cache loadingcache.Cache) {
		// Getting a key that does not exist should error
		_, err := cache.Get(1)
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))

		// Invalidating a key that doesn't exist
		cache.Invalidate(1)

		// Adding values
		cache.Put(1, 1)
		cache.Put(2, 2)
		cache.Put(3, 3)

		// Values exist
		val, err := cache.Get(1)
		require.NoError(t, err)
		require.Equal(t, 1, val)
		val, err = cache.Get(2)
		require.NoError(t, err)
		require.Equal(t, 2, val)
		val, err = cache.Get(3)
		require.NoError(t, err)
		require.Equal(t, 3, val)

		// Invalidate key and get it
		cache.Invalidate(1)
		_, err = cache.Get(1)
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))

		// Invalidate multiple keys at once
		cache.Put(1, 1)
		cache.Put(2, 2)
		cache.Invalidate(1, 2)
		_, err = cache.Get(1)
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))
		_, err = cache.Get(2)
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))

		// Invalidate all keys
		cache.Put(1, 1)
		cache.Put(2, 2)
		cache.InvalidateAll()
		_, err = cache.Get(1)
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))
		_, err = cache.Get(2)
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))
	})
}

func TestExpireAfterWrite(t *testing.T) {
	mockClock := clock.NewMock()
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Clock:            mockClock,
			ExpireAfterWrite: time.Minute,
		},
	},
		func(t *testing.T, cache loadingcache.Cache) {
			cache.Put(1, 1)
			val, err := cache.Get(1)
			require.NoError(t, err)
			require.Equal(t, 1, val)

			// Advance clock up to the expiry threshold
			mockClock.Add(time.Minute)

			// Value should still be returned
			val, err = cache.Get(1)
			require.NoError(t, err)
			require.Equal(t, 1, val)

			// Moving just past the threshold should yield no value
			mockClock.Add(1)
			_, err = cache.Get(1)
			require.Error(t, err)
			require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))
		})
}

func TestExpireAfterRead(t *testing.T) {
	mockClock := clock.NewMock()
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Clock:           mockClock,
			ExpireAfterRead: time.Minute,
		},
	},
		func(t *testing.T, cache loadingcache.Cache) {
			cache.Put(1, 1)
			val, err := cache.Get(1)
			require.NoError(t, err)
			require.Equal(t, 1, val)

			// Advance clock up to the expiry threshold
			mockClock.Add(time.Minute)

			// Value should still be returned
			val, err = cache.Get(1)
			require.NoError(t, err)
			require.Equal(t, 1, val)

			// Since the value was read, we can move the clock another chunk
			// Advance clock up to the expiry threshold
			mockClock.Add(time.Minute)

			// Value should still be returned
			val, err = cache.Get(1)
			require.NoError(t, err)
			require.Equal(t, 1, val)

			// Moving just past the threshold should yield no value
			mockClock.Add(time.Minute + 1)
			_, err = cache.Get(1)
			require.Error(t, err)
			require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))
		})
}

func TestLoadFunc(t *testing.T) {
	loadFunc := &testLoadFunc{}
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Load: loadFunc.LoadFunc,
		},
	},
		func(t *testing.T, cache loadingcache.Cache) {
			defer func() {
				// The looad func is shared by the multiple iterations of the test.
				// Ensure we cleanup after ourselves.
				loadFunc.fail = false
			}()
			// Getting a value that does not exist should load it
			val, err := cache.Get(1)
			require.NoError(t, err)
			require.Equal(t, "1", val)

			// Getting a value that the loader fails to error should propagate the error
			loadFunc.fail = true
			_, err = cache.Get(2)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failing on request")

			// Adding the value manually should succeeed
			cache.Put(2, "true")
			val, err = cache.Get(2)
			require.NoError(t, err)
			require.Equal(t, "true", val)

			// After invalidating, getting should fail again
			cache.Invalidate(2)
			_, err = cache.Get(2)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failing on request")
		})
}

func TestMaxSize(t *testing.T) {
	// TODO MaxSize is currently not properly enforced in a sharded environment
	caches := []loadingcache.Cache{
		loadingcache.New(loadingcache.CacheOptions{
			MaxSize: 1,
		}),
		loadingcache.New(loadingcache.CacheOptions{
			MaxSize:      1,
			ShardCount:   3,
			HashCodeFunc: stringHashCodeFunc,
		}),
	}
	for _, cache := range caches {
		// With a capacity of one element, adding a second element
		// should remove the first
		cache.Put("a", 1)
		cache.Put("b", 2)

		_, err := cache.Get("a")
		require.Error(t, err)
		require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))

		val, err := cache.Get("b")
		require.NoError(t, err)
		require.Equal(t, 2, val)
	}
}

func TestRemovalListeners(t *testing.T) {
	t.Skip("TODO Fix enforcemento of MaxSize")
	mockClock := clock.NewMock()
	removalListener := &testRemovalListener{}
	removalListener2 := &testRemovalListener{}
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Clock:            mockClock,
			ExpireAfterRead:  time.Minute,
			ExpireAfterWrite: 2 * time.Minute,
			MaxSize:          1,
			RemovalListeners: []loadingcache.RemovalListener{removalListener.Listener, removalListener2.Listener},
		},
	},
		func(t *testing.T, cache loadingcache.Cache) {
			defer func() {
				// The listeners are shared by the multiple iterations of the test.
				// Ensure we cleanup after ourselves.
				removalListener.lastRemovalNotification = loadingcache.RemovalNotification{}
				removalListener2.lastRemovalNotification = loadingcache.RemovalNotification{}
			}()

			// Removal due to replacement
			cache.Put(1, 10)
			cache.Put(1, 1)
			lastNotification := removalListener.lastRemovalNotification
			lastNotification2 := removalListener2.lastRemovalNotification
			require.Equal(t, loadingcache.RemovalReasonReplaced, lastNotification.Reason)
			require.Equal(t, loadingcache.RemovalReasonReplaced, lastNotification2.Reason)
			require.Equal(t, 1, lastNotification.Key)
			require.Equal(t, 10, lastNotification.Value)

			// Removal due to size
			cache.Put(2, 2)
			lastNotification = removalListener.lastRemovalNotification
			lastNotification2 = removalListener2.lastRemovalNotification
			require.Equal(t, loadingcache.RemovalReasonSize, lastNotification.Reason)
			require.Equal(t, loadingcache.RemovalReasonSize, lastNotification2.Reason)
			require.Equal(t, 1, lastNotification.Key)
			require.Equal(t, 1, lastNotification.Value)

			// Removal due to read expiration
			mockClock.Add(time.Minute + 1)
			// We don't care about the value or error, we just want to trigger the eviction
			_, _ = cache.Get(2)
			lastNotification = removalListener.lastRemovalNotification
			lastNotification2 = removalListener2.lastRemovalNotification
			require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification.Reason)
			require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification2.Reason)
			require.Equal(t, 2, lastNotification.Key)
			require.Equal(t, 2, lastNotification.Value)

			// Removal due to write expiration
			cache.Put(2, 3)
			mockClock.Add(time.Minute)
			// Doing a read to refresh the expiry
			_, _ = cache.Get(2)
			mockClock.Add(time.Minute)
			// Doing a another read to refresh the expiry
			_, _ = cache.Get(2)
			mockClock.Add(1)
			// We don't care about the value or error, we just want to trigger the eviction
			_, _ = cache.Get(2)
			lastNotification = removalListener.lastRemovalNotification
			lastNotification2 = removalListener2.lastRemovalNotification
			require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification.Reason)
			require.Equal(t, loadingcache.RemovalReasonExpired, lastNotification2.Reason)
			require.Equal(t, 2, lastNotification.Key)
			require.Equal(t, 3, lastNotification.Value)
		})
}

func TestBackgroudEvict(t *testing.T) {
	mockClock := clock.NewMock()
	matrixTest(t, matrixTestOptions{
		cacheOptions: loadingcache.CacheOptions{
			Clock:            mockClock,
			ExpireAfterWrite: 20 * time.Second,
			BackgroundEvict:  true,
		},
	},
		func(t *testing.T, cache loadingcache.Cache) {
			// Add an item
			cache.Put(1, "a")

			// Advance 10 seconds, which should call the backgroud evicter
			mockClock.Add(10 * time.Second)

			// The value should still be there
			_, err := cache.Get(1)
			require.NoError(t, err)

			// Moving the clock past the write threshold
			// TODO: Added an extra second otherwise the ticker wouldn't trigger
			mockClock.Add(11 * time.Second)

			// Since background eviction runs in a go routine, we
			// can only guess when it'll be done.
			// Let's try getting an error for 500ms
			for i := 0; i < 50; i++ {
				_, err = cache.Get(1)
				if err != nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			require.Error(t, err)
			require.Equal(t, loadingcache.ErrKeyNotFound, errors.Cause(err))
		})
}
