# Loading Cache

At its core, loading cache is a rather simple cache implementation.
It is heavily inspired by [Guava](https://github.com/google/guava/wiki/CachesExplained).

# Basics

You can use it as a simple, no fuss cache.

```go
package main

import (
	"github.com/goldstd/loadingcache"
)

func main() {
	cache := loadingcache.Options{}.New()

	// Adding some values and reading them
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
}
```

# Advanced

You can also use more advanced options.

```go
package main

import (
	"github.com/goldstd/loadingcache"
)

func main() {
	cache := loadingcache.Options{
		MaxSize:          2,
		ExpireAfterRead:  2 * time.Minute,
		ExpireAfterWrite: time.Minute,
		RemovalListeners: []loadingcache.RemovalListener{
			func(notification loadingcache.RemovalNotification) {
				fmt.Printf("Entry removed due to %s\n", notification.Reason)
			},
		},
		Load: func(key any) (any, error) {
			fmt.Printf("Loading key %v\n", key)
			return fmt.Sprint(key), nil
		},
	}.New()

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
```


# Benchmarks

Although preliminary, below you can find some benchmarks (included in the repo).

```
$ go test -benchmem -bench .
goos: darwin
goarch: amd64
pkg: github.com/goldstd/loadingcache
cpu: Intel(R) Core(TM) i7-8850H CPU @ 2.60GHz
BenchmarkGetMiss/Sharded_(2)-12           400610              2631 ns/op            1016 B/op         12 allocs/op
BenchmarkGetMiss/Sharded_(3)-12           451284              2618 ns/op            1016 B/op         12 allocs/op
BenchmarkGetMiss/Sharded_(16)-12                  447225              2709 ns/op            1016 B/op         12 allocs/op
BenchmarkGetMiss/Sharded_(32)-12                  441804              2649 ns/op            1015 B/op         12 allocs/op
BenchmarkGetMiss/Simple-12                        737130              1582 ns/op             680 B/op          8 allocs/op
BenchmarkGetHit/Simple-12                        9312319               126.5 ns/op             0 B/op          0 allocs/op
BenchmarkGetHit/Sharded_(2)-12                   8891150               137.2 ns/op             0 B/op          0 allocs/op
BenchmarkGetHit/Sharded_(3)-12                   8845831               137.3 ns/op             0 B/op          0 allocs/op
BenchmarkGetHit/Sharded_(16)-12                  8769058               136.0 ns/op             0 B/op          0 allocs/op
BenchmarkGetHit/Sharded_(32)-12                  8685018               136.1 ns/op             0 B/op          0 allocs/op
BenchmarkPutNew/Sharded_(2)-12                     12978            175136 ns/op              75 B/op          1 allocs/op
BenchmarkPutNew/Sharded_(3)-12                     20353            156025 ns/op             143 B/op          2 allocs/op
BenchmarkPutNew/Sharded_(16)-12                    93429            129674 ns/op             146 B/op          2 allocs/op
BenchmarkPutNew/Sharded_(32)-12                   145015            105135 ns/op             172 B/op          2 allocs/op
BenchmarkPutNew/Simple-12                          10000            181544 ns/op             168 B/op          2 allocs/op
BenchmarkPutNewNoPreWrite/Simple-12              1920939               627.5 ns/op           136 B/op          2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(2)-12         2016948               725.2 ns/op           133 B/op          2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(3)-12         2041030               594.4 ns/op           118 B/op          2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(16)-12                2113516               627.2 ns/op           130 B/op          2 allocs/op
BenchmarkPutNewNoPreWrite/Sharded_(32)-12                1934784               684.8 ns/op           136 B/op          2 allocs/op
BenchmarkPutReplace-12                                   3397014               341.8 ns/op            64 B/op          1 allocs/op
BenchmarkPutAtMaxSize-12                                 3076584               380.7 ns/op            71 B/op          1 allocs/op
PASS
ok      github.com/goldstd/loadingcache 74.447s

```
