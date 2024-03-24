package loadingcache_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/goldstd/loadingcache"
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

func (t *testLoadFunc) Load(key any) (any, error) {
	if t.fail {
		return nil, errors.New("failing on request")
	}
	return fmt.Sprint(key), nil
}

// intHashCodeFunc is a test hash code function for ints which just passes them through
var intHashCodeFunc = func(k any) int {
	return k.(int)
}

func matrixBenchmark(b *testing.B,
	options loadingcache.Options,
	setupFunc matrixBenchmarkSetupFunc,
	testFunc matrixBenchmarkFunc,
) {
	matrixOptions := cacheMatrixOptions(options)
	b.ResetTimer()
	for name := range matrixOptions {
		cache := matrixOptions[name].New()
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
		cache := cacheOptions.New()
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
	cacheOptions loadingcache.Options
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

func cacheMatrixOptions(baseOptions loadingcache.Options) map[string]loadingcache.Options {
	matrix := map[string]loadingcache.Options{}

	simpleOptions := baseOptions
	simpleOptions.ShardCount = 1
	matrix["Simple"] = simpleOptions

	for _, shardCount := range []uint32{2, 3, 16, 32} {
		shardedOptions := baseOptions
		shardedOptions.ShardCount = shardCount
		shardedOptions.HashCodeFunc = intHashCodeFunc
		matrix[fmt.Sprintf("Sharded (%d)", shardCount)] = shardedOptions
	}
	return matrix
}
