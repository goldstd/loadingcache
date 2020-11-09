package loadingcache_test

import (
	"testing"

	"github.com/Hartimer/loadingcache"
)

func BenchmarkGetMiss(b *testing.B) {
	cache := loadingcache.NewGenericCache()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("a")
	}
}

func BenchmarkGetHit(b *testing.B) {
	cache := loadingcache.NewGenericCache()
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("a")
	}
}

func BenchmarkPutNew(b *testing.B) {
	cache := loadingcache.NewGenericCache()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(i, 1)
	}
}

func BenchmarkPutReplace(b *testing.B) {
	cache := loadingcache.NewGenericCache()
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put("a", 1)
	}
}

func BenchmarkPutAtMaxSize(b *testing.B) {
	cache := loadingcache.NewGenericCache(
		loadingcache.MaxSize(1),
	)
	cache.Put("a", 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(i, 1)
	}
}
