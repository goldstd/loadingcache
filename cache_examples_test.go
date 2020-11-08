package loadingcache_test

import (
	"fmt"
	"time"

	"github.com/Hartimer/loadingcache"
)

func ExampleSimpleUsage() {
	cache := loadingcache.NewGenericCache()

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
	if err == loadingcache.ErrKeyNotFound {
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

func ExampleAdvancedUsage() {
	cache := loadingcache.NewGenericCache(
		loadingcache.MaxSize(2),
		loadingcache.ExpireAfterRead(2*time.Minute),
		loadingcache.ExpireAfterWrite(time.Minute),
		loadingcache.RemovalListener(func(notification loadingcache.RemovalNotification) {
			fmt.Printf("Entry removed due to %s\n", notification.Reason)
		}),
		loadingcache.Load(func(key interface{}) (interface{}, error) {
			fmt.Printf("Loading key %v\n", key)
			return fmt.Sprint(key), nil
		}),
	)

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