# Loading Cache

[![PkgGoDev](https://pkg.go.dev/badge/github.com/Hartimer/loadingcache)](https://pkg.go.dev/github.com/Hartimer/loadingcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/Hartimer/loadingcache)](https://goreportcard.com/report/github.com/Hartimer/loadingcache)
[![Coverage](https://codecov.io/gh/Hartimer/loadingcache/branch/master/graph/badge.svg)](https://codecov.io/gh/Hartimer/loadingcache)

At its core, loading cache is a rather simple cache implementation.
It is heavily inspired by [Guava](https://github.com/google/guava/wiki/CachesExplained).

# Basics

You can use it as a simple, no fuss cache.

```go
cache := loadingcache.New(loadingcache.CacheOptions{})

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
if errors.Is(err, loadingcache.ErrKeyNotFound) {
    fmt.Println("That key does not exist")
}

// Evicting
cache.Invalidate("a")
cache.Invalidate("b", "c")
cache.InvalidateAll()

// Output: 1
// 2
// That key does not exist
```

# Advanced

You can also use more advanced options.

```go
cache := loadingcache.New(loadingcache.CacheOptions{
    MaxSize:          2,
    ExpireAfterRead:  2 * time.Minute,
    ExpireAfterWrite: time.Minute,
    RemovalListeners: []loadingcache.RemovalListener{
        func(notification loadingcache.RemovalNotification) {
            fmt.Printf("Entry removed due to %s\n", notification.Reason)
        },
    },
    Load: func(key interface{}) (interface{}, error) {
        fmt.Printf("Loading key %v\n", key)
        return fmt.Sprint(key), nil
    },
})

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
```

# Type Safety

A type safe wrapper is also provided in the form of a code generator.

```go
//go:generate go run github.com/Hartimer/loadingcache/cmd/typedcache -name CoolCache -key "github.com/Hartimer/loadingcache/example.Name" -value int64
```

will produce a strongly typed cache, with all the goodies.

```go
// Example demonstrates using a strongly typed cache
type Example struct {
	cache CoolCache
}

// New returns a new instance of the example
func New() *Example {
	return &Example{
		//nolint:gomnd
		cache: NewCoolCache(CoolCacheOptions{MaxSize: 10}),
	}
}

// Set sents an entry in the cache
func (e *Example) Set(name Name, age int64) {
	e.cache.Put(name, age)
}

// AddAges returns the sum of ages identified by names
func (e *Example) AddAges(names ...Name) int64 {
	var total int64 = 0
	for _, name := range names {
		age, err := e.cache.Get(name)
		if err != nil {
			panic(err)
		}
		total += age
	}
	return total
}
```

# Benchmarks

Although preliminary, below you can find some benchemarks (included in the repo).

```
go test -benchmem -bench .
goos: darwin
goarch: amd64
pkg: github.com/Hartimer/loadingcache
BenchmarkGetMiss/Simple-4     	  562602	      2139 ns/op	     712 B/op	       9 allocs/op
BenchmarkGetMiss/Sharded_(2)-4         	  381688	      3057 ns/op	    1064 B/op	      13 allocs/op
BenchmarkGetMiss/Sharded_(3)-4         	  389958	      3018 ns/op	    1064 B/op	      13 allocs/op
BenchmarkGetMiss/Sharded_(16)-4        	  390182	      3029 ns/op	    1064 B/op	      13 allocs/op
BenchmarkGetMiss/Sharded_(32)-4        	  386788	      3015 ns/op	    1064 B/op	      12 allocs/op
BenchmarkGetHit/Simple-4               	 7663969	       151 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetHit/Sharded_(2)-4          	 6717175	       173 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetHit/Sharded_(3)-4          	 6806174	       173 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetHit/Sharded_(16)-4         	 6413422	       174 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetHit/Sharded_(32)-4         	 6795751	       173 ns/op	       0 B/op	       0 allocs/op
BenchmarkPutNew/Sharded_(16)-4         	   80506	    121933 ns/op	     172 B/op	       2 allocs/op
BenchmarkPutNew/Sharded_(32)-4         	  146749	    116118 ns/op	     187 B/op	       2 allocs/op
BenchmarkPutNew/Simple-4               	   10000	    214555 ns/op	     184 B/op	       2 allocs/op
BenchmarkPutNew/Sharded_(2)-4          	   10000	    109179 ns/op	     184 B/op	       2 allocs/op
BenchmarkPutNew/Sharded_(3)-4          	   16292	    157432 ns/op	      88 B/op	       2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(3)-4         	 1542378	       815 ns/op	     147 B/op	       2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(16)-4        	 2090314	       738 ns/op	     147 B/op	       2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(32)-4        	 1713048	       635 ns/op	     160 B/op	       2 allocs/op
BenchmarkPutNewNoPreWrite/Simple-4              	 1629452	       900 ns/op	      91 B/op	       2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(2)-4         	 1635944	       755 ns/op	      91 B/op	       2 allocs/op
BenchmarkPutReplace-4                           	 1867154	       607 ns/op	      80 B/op	       1 allocs/op
BenchmarkPutAtMaxSize-4                         	 1668778	       643 ns/op	      88 B/op	       1 allocs/op
```