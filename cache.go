package loadingcache

import "errors"

// ErrKeyNotFound represents an error indicating that the key was not found
var ErrKeyNotFound error = errors.New("Key not found")

// Cache describe the base interface to interact with a generic cache.
//
// This interface reduces all keys and values to a generic interface{}.
type Cache interface {
	// Get returns the value associated with a given key. If no entry exists for
	// the provided key, loadingcache.ErrKeyNotFound is returned.
	Get(key interface{}) (interface{}, error)

	// Put adds a value to the cache identified by a key.
	// If a value already exists associated with that key, it
	// is replaced.
	Put(key interface{}, value interface{})

	// Invalidate removes a key from the cache. If no key exists it is a noop.
	Invalidate(key interface{})
}

// genericCache is an implementation of a cache where keys and values are
// of type interface{}
type genericCache struct {
	data map[interface{}]interface{}
}

// NewGenericCache returns a new instance of a generic cache
func NewGenericCache() Cache {
	return &genericCache{
		data: map[interface{}]interface{}{},
	}
}

func (g *genericCache) Get(key interface{}) (interface{}, error) {
	val, exists := g.data[key]
	if !exists {
		return nil, ErrKeyNotFound
	}
	return val, nil
}

func (g *genericCache) Put(key interface{}, value interface{}) {
	g.data[key] = value
}

func (g *genericCache) Invalidate(key interface{}) {
	delete(g.data, key)
}
