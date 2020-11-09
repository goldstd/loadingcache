package loadingcache_test

import (
	"fmt"
	"testing"

	"github.com/Hartimer/loadingcache"
)

func BenchmarkGetMiss(b *testing.B) {
	matrixTest(b, noopSetupFunc, func(b *testing.B, cache loadingcache.Cache) {
		for i := 0; i < b.N; i++ {
			cache.Get(i)
		}
	})
}

func BenchmarkGetHit(b *testing.B) {
	matrixTest(b,
		func(b *testing.B, cache loadingcache.Cache) {
			cache.Put(1, "a")
		},
		func(b *testing.B, cache loadingcache.Cache) {
			for i := 0; i < b.N; i++ {
				cache.Get(1)
			}
		})
}

func BenchmarkPutNew(b *testing.B) {
	matrixTest(b, noopSetupFunc, func(b *testing.B, cache loadingcache.Cache) {
		for i := 0; i < b.N; i++ {
			cache.Put(i, 1)
		}
	})
}

func BenchmarkPutReplace(b *testing.B) {
	cache := loadingcache.New(loadingcache.CacheOptions{})
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put("a", 1)
	}
}

func BenchmarkPutAtMaxSize(b *testing.B) {
	cache := loadingcache.New(loadingcache.CacheOptions{
		MaxSize: 1,
	})
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(i, 1)
	}
}

func matrixTest(b *testing.B, setupFunc matrixSetupFunc, testFunc matrixTestFunc) {
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

type matrixSetupFunc func(b *testing.B, cache loadingcache.Cache)

var noopSetupFunc = func(b *testing.B, cache loadingcache.Cache) {}

type matrixTestFunc func(b *testing.B, cache loadingcache.Cache)

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
