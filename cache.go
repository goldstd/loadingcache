package loadingcache

import (
	"errors"
	"time"

	"github.com/benbjohnson/clock"
)

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

// CacheOption describes an option that can configure the cache
type CacheOption func(Cache)

// WithClock allows passing a custom clock to be used with the cache.
//
// This is useful for testing, where controlling time is important.
func WithClock(clk clock.Clock) CacheOption {
	return func(cache Cache) {
		if g, ok := cache.(*genericCache); ok {
			g.clock = clk
		}
	}
}

// ExpireAfterWrite configures the cache to expire entries after
// a given duration after writing.
func ExpireAfterWrite(duration time.Duration) CacheOption {
	return func(cache Cache) {
		if g, ok := cache.(*genericCache); ok {
			g.expireAfterWrite = duration
		}
	}
}

// genericCache is an implementation of a cache where keys and values are
// of type interface{}
type genericCache struct {
	clock            clock.Clock
	expireAfterWrite time.Duration
	data             map[interface{}]interface{}
	dataWriteTime    map[interface{}]time.Time
}

// NewGenericCache returns a new instance of a generic cache
func NewGenericCache(options ...CacheOption) Cache {
	cache := &genericCache{
		clock:         clock.New(),
		data:          map[interface{}]interface{}{},
		dataWriteTime: map[interface{}]time.Time{},
	}
	for _, option := range options {
		option(cache)
	}
	return cache
}

func (g *genericCache) Get(key interface{}) (interface{}, error) {
	val, exists := g.data[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	if writeTime, exists := g.dataWriteTime[key]; exists && g.clock.Now().After(writeTime.Add(g.expireAfterWrite)) {
		return nil, ErrKeyNotFound
	}

	return val, nil
}

func (g *genericCache) Put(key interface{}, value interface{}) {
	g.data[key] = value
	if g.expireAfterWrite > 0 {
		g.dataWriteTime[key] = g.clock.Now()
	}
}

func (g *genericCache) Invalidate(key interface{}) {
	delete(g.data, key)
}
