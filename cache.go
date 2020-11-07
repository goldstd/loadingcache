package loadingcache

import (
	"errors"
	"math"
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

// MaxSize limits the number of entries allowed in the cache.
// If the limit is achieved, an eviction process will take place,
// this means that eviction policies will be executed such as write
// time, read time or random entry if no evection policy frees up
// space.
func MaxSize(maxSize int32) CacheOption {
	return func(cache Cache) {
		if g, ok := cache.(*genericCache); ok {
			g.maxSize = maxSize
		}
	}
}

// genericCache is an implementation of a cache where keys and values are
// of type interface{}
type genericCache struct {
	clock    clock.Clock
	loadFunc LoadFunc
	maxSize  int32

	expireAfterWrite time.Duration
	dataWriteTime    map[interface{}]time.Time

	expireAfterRead time.Duration
	dataReadTime    map[interface{}]time.Time

	data     map[interface{}]interface{}
	dataLock sync.RWMutex
}

// NewGenericCache returns a new instance of a generic cache
func NewGenericCache(options ...CacheOption) Cache {
	cache := &genericCache{
		clock:         clock.New(),
		maxSize:       math.MaxInt32,
		data:          map[interface{}]interface{}{},
		dataWriteTime: map[interface{}]time.Time{},
		dataReadTime:  map[interface{}]time.Time{},
	}
	for _, option := range options {
		option(cache)
	}
	return cache
}

func (g *genericCache) Get(key interface{}) (interface{}, error) {
	g.dataLock.RLock()
	val, exists := g.data[key]
	if !exists {
		g.dataLock.RUnlock()
		return g.load(key)
	}
	g.dataLock.RUnlock()

	writeTime, exists := g.dataWriteTime[key]
	if exists && g.clock.Now().After(writeTime.Add(g.expireAfterWrite)) {
		return g.load(key)
	}

	readTime, exists := g.dataReadTime[key]
	if exists && g.clock.Now().After(readTime.Add(g.expireAfterRead)) {
		return g.load(key)
	}
	if g.expireAfterRead > 0 {
		// It is possible that this will race. It will only be a problem
		// if the expiry thresholds have to be respected with a high
		// degree of precision (which is subjective).
		g.dataReadTime[key] = g.clock.Now()
	}

	return val, nil
}

func (g *genericCache) load(key interface{}) (interface{}, error) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	if g.loadFunc == nil {
		return nil, ErrKeyNotFound
	}

	// It is possible that another call loaded the value for this key.
	// Let's do a double check if that was the case, since we have
	// the lock.
	if val, exists := g.data[key]; exists {
		return val, nil
	}

	val, err := g.loadFunc(key)
	if err != nil {
		return nil, err
	}
	g.internalPut(key, val)
	return val, nil
}

// internalPut actually saves the values into the internal structures.
// It does not handle any synchronization, leaving that to the caller.
func (g *genericCache) internalPut(key interface{}, value interface{}) {
	if int32(len(g.data)) >= g.maxSize {
		// If eviction is needed it currently removes a random entry,
		// since maps do not have a deterministic order.
		// TODO: Apply smarter eviction policies if available
		for toEvict := range g.data {
			delete(g.data, toEvict)
			break
		}
	}
	g.data[key] = value
	if g.expireAfterWrite > 0 {
		g.dataWriteTime[key] = g.clock.Now()
	}
	if g.expireAfterRead > 0 {
		g.dataReadTime[key] = g.clock.Now()
	}
}

func (g *genericCache) Put(key interface{}, value interface{}) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	g.internalPut(key, value)
}

func (g *genericCache) Invalidate(key interface{}) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	delete(g.data, key)
}
