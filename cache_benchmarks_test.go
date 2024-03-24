package loadingcache_test

import (
	"testing"
	"time"

	"github.com/goldstd/loadingcache"
)

func BenchmarkGetMiss(b *testing.B) {
	matrixBenchmark(b,
		loadingcache.Options{},
		noopBenchmarkSetupFunc,
		func(b *testing.B, cache loadingcache.Cache) {
			for i := 0; i < b.N; i++ {
				_, _ = cache.Get(i)
			}
		})
}

func BenchmarkGetHit(b *testing.B) {
	matrixBenchmark(b,
		loadingcache.Options{},
		func(b *testing.B, cache loadingcache.Cache) {
			cache.Put(1, "a")
		},
		func(b *testing.B, cache loadingcache.Cache) {
			for i := 0; i < b.N; i++ {
				_, err := cache.Get(1)
				if err != nil {
					panic(err)
				}
			}
		})
}

func BenchmarkPutNew(b *testing.B) {
	matrixBenchmark(b,
		loadingcache.Options{},
		noopBenchmarkSetupFunc,
		func(b *testing.B, cache loadingcache.Cache) {
			for i := 0; i < b.N; i++ {
				cache.Put(i, 1)
			}
		})
}

func BenchmarkPutNewNoPreWrite(b *testing.B) {
	matrixBenchmark(b,
		loadingcache.Options{EvictInterval: time.Second},
		noopBenchmarkSetupFunc,
		func(b *testing.B, cache loadingcache.Cache) {
			for i := 0; i < b.N; i++ {
				cache.Put(i, 1)
			}
		})
}

func BenchmarkPutReplace(b *testing.B) {
	cache := loadingcache.Options{}.New()
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put("a", 1)
	}
}

func BenchmarkPutAtMaxSize(b *testing.B) {
	cache := loadingcache.Options{
		MaxSize: 1,
	}.New()
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(i, 1)
	}
}
