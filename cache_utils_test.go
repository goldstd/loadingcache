package loadingcache_test

import (
	"context"
	"fmt"
	"hash/fnv"
	"testing"

	"github.com/Hartimer/loadingcache"
	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"go.uber.org/goleak"
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
	if _, err := h.Write([]byte(k.(string))); err != nil {
		panic(err)
	}
	return int(h.Sum32())
}

// intHashCodeFunc is a test hash code function for ints which just passes them through
var intHashCodeFunc = func(k interface{}) int {
	return k.(int)
}

func matrixBenchmark(b *testing.B,
	options loadingcache.CacheOptions,
	setupFunc matrixBenchmarkSetupFunc,
	testFunc matrixBenchmarkFunc) {
	matrixOptions := cacheMatrixOptions(options)
	b.ResetTimer()
	for name := range matrixOptions {
		cache := loadingcache.New(matrixOptions[name])
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
	defer goleak.VerifyNone(t)
	matrixOptions := cacheMatrixOptions(options.cacheOptions)
	for name := range matrixOptions {
		utils := &matrixTestUtils{}
		cacheOptions := matrixOptions[name]
		if cacheOptions.Clock == nil {
			mockClock := clock.NewMock()
			utils.clock = mockClock
			cacheOptions.Clock = mockClock
		}
		ctx := put(context.Background(), utils)
		cache := loadingcache.New(cacheOptions)
		if options.setupFunc != nil {
			options.setupFunc(t, cache)
		}
		t.Run(name, func(t *testing.T) {
			defer cache.Close()
			testFunc(t, ctx, cache)
		})
	}
}

type matrixTestOptions struct {
	cacheOptions loadingcache.CacheOptions
	setupFunc    func(t *testing.T, cache loadingcache.Cache)
}

type matrixTestUtils struct {
	clock *clock.Mock
}

type utilsKey struct{}

func put(ctx context.Context, utils *matrixTestUtils) context.Context {
	return context.WithValue(ctx, utilsKey{}, utils)
}

func get(ctx context.Context) *matrixTestUtils {
	val := ctx.Value(utilsKey{})
	if val == nil {
		panic("could not find utils in context")
	}
	return val.(*matrixTestUtils)
}

type matrixTestFunc func(t *testing.T, ctx context.Context, cache loadingcache.Cache)

func cacheMatrixOptions(baseOptions loadingcache.CacheOptions) map[string]loadingcache.CacheOptions {
	matrix := map[string]loadingcache.CacheOptions{}

	simpleOptions := baseOptions
	simpleOptions.ShardCount = 1
	matrix["Simple"] = simpleOptions

	for _, shardCount := range []int{2, 3, 16, 32} {
		shardedOptions := baseOptions
		shardedOptions.ShardCount = shardCount
		shardedOptions.HashCodeFunc = intHashCodeFunc
		matrix[fmt.Sprintf("Sharded (%d)", shardCount)] = shardedOptions
	}
	return matrix
}
