package loadingcache_test

import (
	"fmt"
	"hash/fnv"
	"testing"

	"github.com/Hartimer/loadingcache"
	"github.com/pkg/errors"
)

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

// stringHashCodeFunc is a test hash code function for strings which uses fnv.New32a
var stringHashCodeFunc = func(k interface{}) int {
	h := fnv.New32a()
	h.Write([]byte(k.(string)))
	return int(h.Sum32())
}

// intHashCodeFunc is a test hash code function for ints which just passes them through
var intHashCodeFunc = func(k interface{}) int {
	return k.(int)
}

func matrixBenchmark(b *testing.B, setupFunc matrixBenchmarkSetupFunc, testFunc matrixBenchmarkFunc) {
	matrix := cacheMatrix()
	b.ResetTimer()
	for name := range matrix {
		cache := matrix[name]
		setupFunc(b, cache)
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			testFunc(b, cache)
		})
	}
}

type matrixBenchmarkSetupFunc func(b *testing.B, cache loadingcache.Cache)

var noopBenchmarkSetupFunc = func(b *testing.B, cache loadingcache.Cache) {}

type matrixBenchmarkFunc func(b *testing.B, cache loadingcache.Cache)

func matrixTest(t *testing.T, options matrixTestOptions, testFunc matrixTestFunc) {
	matrix := cacheMatrixWithOptions(options.cacheOptions)
	for name := range matrix {
		cache := matrix[name]
		if options.setupFunc != nil {
			options.setupFunc(t, cache)
		}
		t.Run(name, func(t *testing.T) {
			testFunc(t, cache)
		})
	}
}

type matrixTestSetupFunc func(t *testing.T, cache loadingcache.Cache)

type matrixTestOptions struct {
	cacheOptions loadingcache.CacheOptions
	setupFunc    func(t *testing.T, cache loadingcache.Cache)
}

var noopTestSetupFunc = func(t *testing.T, cache loadingcache.Cache) {}

type matrixTestFunc func(t *testing.T, cache loadingcache.Cache)

func cacheMatrix() map[string]loadingcache.Cache {
	return cacheMatrixWithOptions(loadingcache.CacheOptions{})
}

func cacheMatrixWithOptions(options loadingcache.CacheOptions) map[string]loadingcache.Cache {
	matrix := map[string]loadingcache.Cache{}

	simpleOptions := options
	simpleOptions.ShardCount = 1
	matrix["Simple"] = loadingcache.New(simpleOptions)

	for _, shardCount := range []int{2, 3, 16, 32} {
		shardedOptions := options
		shardedOptions.ShardCount = shardCount
		shardedOptions.HashCodeFunc = intHashCodeFunc
		matrix[fmt.Sprintf("Sharded (%d)", shardCount)] = loadingcache.New(shardedOptions)
	}
	return matrix
}
