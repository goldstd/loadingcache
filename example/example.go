package example

//go:generate go run github.com/Hartimer/loadingcache/cmd/typedcache -name CoolCache -key "github.com/Hartimer/loadingcache/example.Name" -value int64

// Name just represents a deeper type
type Name string

// Example demonstrats using a strongly typed cache
type Example struct {
	cache CoolCache
}

// New returns a new instance of the example
func New() *Example {
	return &Example{
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
