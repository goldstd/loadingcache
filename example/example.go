package example

//go:generate go run github.com/Hartimer/loadingcache/cmd/typedcache -name CoolCache -key "github.com/Hartimer/loadingcache/example.Name" -value int64

// Name just represents a deeper type
type Name string

// Example demonstrats using a strongly typed cache
type Example struct {
	cache CoolCache
}

func (e Example) addAges(names ...Name) int64 {
	total := 0
	for _, name := range names {
		age, err := e.cache.Get(name)
		if err != nil {
			panic(err)
		}
		total += age
	}
	return total
}
