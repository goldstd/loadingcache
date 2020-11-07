package loadingcache

import (
	"errors"
	"sync"
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

// LoadFunc represents a function that given a key, it returns a value or an error.
type LoadFunc func(interface{}) (interface{}, error)

// Clock allows passing a custom clock to be used with the cache.
//
// This is useful for testing, where controlling time is important.
func Clock(clk clock.Clock) CacheOption {
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

// ExpireAfterRead configures the cache to expire entries after
// a given duration after reading.
func ExpireAfterRead(duration time.Duration) CacheOption {
	return func(cache Cache) {
		if g, ok := cache.(*genericCache); ok {
			g.expireAfterRead = duration
		}
	}
}

// Load configures a loading function
func Load(f LoadFunc) CacheOption {
	return func(cache Cache) {
		if g, ok := cache.(*genericCache); ok {
			g.loadFunc = f
		}
	}
}

// genericCache is an implementation of a cache where keys and values are
// of type interface{}
type genericCache struct {
	clock    clock.Clock
	loadFunc LoadFunc

	expireAfterWrite  time.Duration
	dataWriteTime     map[interface{}]time.Time
	dataWriteTimeLock sync.Mutex

	expireAfterRead  time.Duration
	dataReadTime     map[interface{}]time.Time
	dataReadTimeLock sync.Mutex

	data map[interface{}]interface{}
}

// NewGenericCache returns a new instance of a generic cache
func NewGenericCache(options ...CacheOption) Cache {
	cache := &genericCache{
		clock:            clock.New(),
		expireAfterWrite: 0,
		expireAfterRead:  0,
		data:             map[interface{}]interface{}{},
		dataWriteTime:    map[interface{}]time.Time{},
		dataReadTime:     map[interface{}]time.Time{},
	}
	for _, option := range options {
		option(cache)
	}
	return cache
}

func (g *genericCache) Get(key interface{}) (interface{}, error) {
	val, exists := g.data[key]
	if !exists {
		return g.load(key)
	}

	g.dataWriteTimeLock.Lock()
	writeTime, exists := g.dataWriteTime[key]
	g.dataWriteTimeLock.Unlock()
	if exists && g.clock.Now().After(writeTime.Add(g.expireAfterWrite)) {
		return g.load(key)
	}

	g.dataReadTimeLock.Lock()
	readTime, exists := g.dataReadTime[key]
	g.dataReadTimeLock.Unlock()
	if exists && g.clock.Now().After(readTime.Add(g.expireAfterRead)) {
		return g.load(key)
	}
	g.dataReadTimeLock.Lock()
	g.dataReadTime[key] = g.clock.Now()
	g.dataReadTimeLock.Unlock()

	return val, nil
}

func (g *genericCache) load(key interface{}) (interface{}, error) {
	if g.loadFunc == nil {
		return nil, ErrKeyNotFound
	}
	val, err := g.loadFunc(key)
	if err != nil {
		return nil, err
	}
	g.Put(key, val)
	return val, nil
}

func (g *genericCache) Put(key interface{}, value interface{}) {
	g.data[key] = value
	if g.expireAfterWrite > 0 {
		g.dataWriteTimeLock.Lock()
		g.dataWriteTime[key] = g.clock.Now()
		g.dataWriteTimeLock.Unlock()
	}
	if g.expireAfterRead > 0 {
		g.dataReadTimeLock.Lock()
		g.dataReadTime[key] = g.clock.Now()
		g.dataReadTimeLock.Unlock()
	}
}

func (g *genericCache) Invalidate(key interface{}) {
	delete(g.data, key)
}
