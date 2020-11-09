package reference_test

import (
	"fmt"
	"time"

	"github.com/Hartimer/loadingcache/cmd/typedcache/internal/reference"
)

func ExampleAdvancedUsage() {
	cache := reference.NewCache(reference.CacheOptions{
		MaxSize:          2,
		ExpireAfterRead:  2 * time.Minute,
		ExpireAfterWrite: time.Minute,
		RemovalListeners: []reference.RemovalListener{
			func(notification reference.RemovalNotification) {
				fmt.Printf("Entry removed due to %s\n", notification.Reason)
			},
		},
		Load: func(key string) (int64, error) {
			fmt.Printf("Loading key %s\n", key)
			return int64(len(key)), nil
		},
	})

	cache.Put("a", 1)
	var val1 int64
	val1, _ = cache.Get("a")
	fmt.Printf("%v\n", val1)

	val2, _ := cache.Get("aa")
	fmt.Printf("%v\n", val2)

	val3, _ := cache.Get("aaa")
	fmt.Printf("%v\n", val3)

	// Output: 1
	// Loading key aa
	// 2
	// Loading key aaa
	// Entry removed due to SIZE
	// 3
}
