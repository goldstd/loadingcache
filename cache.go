package loadingcache

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
)

// ErrKeyNotFound represents an error indicating that the key was not found
var ErrKeyNotFound error = errors.New("Key not found")

// RemovalReason is an enum describing the causes for an entry to
// be removed from the cache.
type RemovalReason string

const (
	// RemovalReasonExplicit means the entry was explicitly invalidated
	RemovalReasonExplicit RemovalReason = "EXPLICIT"

	// RemovalReasonReplaced means the entry was replaced by a new one
	RemovalReasonReplaced RemovalReason = "REPLACED"

	// RemovalReasonExpired means the entry expired, e.g. too much time
	// since last read/write.
	RemovalReasonExpired RemovalReason = "EXPIRED"

	// RemovalReasonSize means the entry was removed due to the cache size.
	RemovalReasonSize RemovalReason = "SIZE"
)

// RemovalNotification is passed to listeners everytime an entry is removed
type RemovalNotification struct {
	Key    interface{}
	Value  interface{}
	Reason RemovalReason
}

// RemovalListenerFunc represents a removal listener
type RemovalListenerFunc func(RemovalNotification)

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

	// Invalidate removes keys from the cache. If a key does not exists it is a noop.
	Invalidate(key interface{}, keys ...interface{})

	// InvalidateAll invalidates all keys
	InvalidateAll()
}

// CacheOption describes an option that can configure the cache
type CacheOption func(Cache)

// LoadFunc represents a function that given a key, it returns a value or an error.
type LoadFunc func(interface{}) (interface{}, error)

// cast converts Cache to *genericCache or panics if it fails
func cast(cache Cache) *genericCache {
	c, ok := cache.(*genericCache)
	if !ok {
		panic(fmt.Sprintf("unexpected implementation of Cache %T", cache))
	}
	return c
}

// Clock allows passing a custom clock to be used with the cache.
//
// This is useful for testing, where controlling time is important.
func Clock(clk clock.Clock) CacheOption {
	return func(cache Cache) {
		cast(cache).clock = clk
	}
}

// ExpireAfterWrite configures the cache to expire entries after
// a given duration after writing.
func ExpireAfterWrite(duration time.Duration) CacheOption {
	return func(cache Cache) {
		cast(cache).expireAfterWrite = duration
		cast(cache).expiresAfterWrite = true
	}
}

// ExpireAfterRead configures the cache to expire entries after
// a given duration after reading.
func ExpireAfterRead(duration time.Duration) CacheOption {
	return func(cache Cache) {
		cast(cache).expireAfterRead = duration
		cast(cache).expiresAfterRead = true
	}
}

// Load configures a loading function
func Load(f LoadFunc) CacheOption {
	return func(cache Cache) {
		cast(cache).loadFunc = f
	}
}

// MaxSize limits the number of entries allowed in the cache.
// If the limit is achieved, an eviction process will take place,
// this means that eviction policies will be executed such as write
// time, read time or random entry if no evection policy frees up
// space.
func MaxSize(maxSize int32) CacheOption {
	return func(cache Cache) {
		cast(cache).maxSize = maxSize
	}
}

// RemovalListener adds a removal listener
func RemovalListener(listener RemovalListenerFunc) CacheOption {
	return func(cache Cache) {
		g := cast(cache)
		g.removalListeners = append(g.removalListeners, listener)
	}
}

// genericCache is an implementation of a cache where keys and values are
// of type interface{}
type genericCache struct {
	clock    clock.Clock
	loadFunc LoadFunc
	maxSize  int32

	removalListeners []RemovalListenerFunc

	expireAfterWrite  time.Duration
	expiresAfterWrite bool

	expireAfterRead  time.Duration
	expiresAfterRead bool

	data     map[interface{}]*cacheEntry
	dataLock sync.RWMutex
}

type cacheEntry struct {
	key       interface{}
	value     interface{}
	lastRead  time.Time
	lastWrite time.Time
}

// NewGenericCache returns a new instance of a generic cache
func NewGenericCache(options ...CacheOption) Cache {
	cache := &genericCache{
		clock:   clock.New(),
		maxSize: math.MaxInt32,
		data:    map[interface{}]*cacheEntry{},
	}
	for _, option := range options {
		option(cache)
	}
	return cache
}

func (g *genericCache) isExpired(entry *cacheEntry) bool {
	if g.expiresAfterRead && entry.lastRead.Add(g.expireAfterRead).Before(g.clock.Now()) {
		return true
	}
	if g.expiresAfterWrite && entry.lastWrite.Add(g.expireAfterWrite).Before(g.clock.Now()) {
		return true
	}
	return false
}

func (g *genericCache) Get(key interface{}) (interface{}, error) {
	g.dataLock.RLock()
	val, exists := g.data[key]
	if !exists {
		g.dataLock.RUnlock()
		val, err := g.load(key)
		return val, errors.Wrap(err, "")
	}
	g.dataLock.RUnlock()

	if g.isExpired(val) {
		g.concurrentEvict(key, RemovalReasonExpired)
		val, err := g.load(key)
		return val, errors.Wrap(err, "")
	}
	// It is possible that this will race. It will only be a problem
	// if the expiry thresholds have to be respected with a high
	// degree of precision (which is subjective).
	val.lastRead = g.clock.Now()
	return val.value, nil
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
		return nil, errors.Wrapf(err, "failed to load key %v", key)
	}
	g.internalPut(key, val)
	return val, nil
}

func (g *genericCache) concurrentEvict(key interface{}, reason RemovalReason) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	g.evict(key, reason)
}

func (g *genericCache) evict(key interface{}, reason RemovalReason) {
	val, exists := g.data[key]
	if !exists {
		return
	}
	delete(g.data, key)

	if len(g.removalListeners) == 0 {
		return
	}
	notification := RemovalNotification{
		Key:    key,
		Value:  val.value,
		Reason: reason,
	}
	var listenerWg sync.WaitGroup
	listenerWg.Add(len(g.removalListeners))
	for i := range g.removalListeners {
		listener := g.removalListeners[i]
		go func() {
			defer listenerWg.Done()
			listener(notification)
		}()
	}
	listenerWg.Wait()
}

// internalPut actually saves the values into the internal structures.
// It does not handle any synchronization, leaving that to the caller.
func (g *genericCache) internalPut(key interface{}, value interface{}) {
	if int32(len(g.data)) >= g.maxSize {
		// If eviction is needed it currently removes a random entry,
		// since maps do not have a deterministic order.
		// TODO: Apply smarter eviction policies if available
		for toEvict := range g.data {
			g.evict(toEvict, RemovalReasonSize)
			break
		}
	}
	g.data[key] = &cacheEntry{
		key:       key,
		value:     value,
		lastRead:  g.clock.Now(),
		lastWrite: g.clock.Now(),
	}
}

func (g *genericCache) Put(key interface{}, value interface{}) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	if _, exists := g.data[key]; exists {
		g.evict(key, RemovalReasonReplaced)
	}
	g.internalPut(key, value)
}

func (g *genericCache) Invalidate(key interface{}, keys ...interface{}) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	delete(g.data, key)
	for _, k := range keys {
		delete(g.data, k)
	}
}

func (g *genericCache) InvalidateAll() {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	for key := range g.data {
		delete(g.data, key)
	}
}
