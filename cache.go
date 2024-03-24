// Package loadingcache provides a way for clients to create a cache capable
// of loading values on demand, should they get cache misses.
//
// You can configure the cache to expire entries after a certain amount elapses
// since the last write and/or read.
//
// This project is heavily inspired by Guava Cache (https://github.com/google/guava/wiki/CachesExplained).
//
// All errors are wrapped by github.com/pkg/errors.Wrap. If you which to check
// the type of it, please use github.com/pkg/errors.Is.
package loadingcache

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/goldstd/loadingcache/internal/stats"
	"github.com/pkg/errors"
)

// ErrKeyNotFound represents an error indicating that the key was not found
var ErrKeyNotFound = errors.New("Key not found")

// RemovalReason is an enum describing the causes for an entry to
// be removed from the cache.
type RemovalReason int

const (
	// RemovalReasonExplicit means the entry was explicitly invalidated
	RemovalReasonExplicit RemovalReason = iota

	// RemovalReasonReplaced means the entry was replaced by a new one
	RemovalReasonReplaced

	// RemovalReasonReadExpired means the entry read expired, e.g. too much time
	// since last read/write.
	RemovalReasonReadExpired

	// RemovalReasonWriteExpired means the entry write expired, e.g. too much time
	// since last read/write.
	RemovalReasonWriteExpired

	// RemovalReasonSize means the entry was removed due to the cache size.
	RemovalReasonSize
)

func (r RemovalReason) String() string {
	switch r {
	case RemovalReasonExplicit:
		return "Explicit"
	case RemovalReasonReplaced:
		return "Replaced"
	case RemovalReasonReadExpired:
		return "ReadExpired"
	case RemovalReasonWriteExpired:
		return "WriteExpired"
	case RemovalReasonSize:
		return "Size"
	}
	return "Unknown"
}

// RemovalNotification is passed to listeners everytime an entry is removed
type RemovalNotification struct {
	Key    any
	Value  any
	Reason RemovalReason
}

// RemovalListener represents a removal listener
type RemovalListener func(RemovalNotification)

// Cache describe the base interface to interact with a generic cache.
//
// This interface reduces all keys and values to a generic any.
type Cache interface {
	// Get returns the value associated with a given key. If no entry exists for
	// the provided key, loadingcache.ErrKeyNotFound is returned.
	Get(key any) (any, error)

	// Put adds a value to the cache identified by a key.
	// If a value already exists associated with that key, it
	// is replaced.
	Put(key, value any)

	// Invalidate removes keys from the cache. If a key does not exist it is a noop.
	Invalidate(keys ...any)

	// InvalidateAll invalidates all keys
	InvalidateAll()

	// Close cleans up any resources used by the cache
	Close()

	// Stats returns the current stats
	Stats() Stats

	// IsSharded tells the implementation is a sharded cache for testing.
	IsSharded() bool
}

// Options available options to initialize the cache
type Options struct {
	// Clock allows passing a custom clock to be used with the cache.
	//
	// This is useful for testing, where controlling time is important.
	Clock clock.Clock

	// Load configures a loading function
	Load LoadFunc

	// HashCodeFunc is a function that produces a hashcode of the key.
	//
	// See https://docs.oracle.com/en/java/javase/15/docs/api/java.base/java/lang/Object.html#hashCode()
	// for best practices surrounding hash code functions.
	HashCodeFunc func(key any) int

	// RemovalListeners configures a removal listeners
	RemovalListeners []RemovalListener

	// ExpireAfterWrite configures the cache to expire entries after
	// a given duration after writing.
	ExpireAfterWrite time.Duration

	// ExpireAfterRead configures the cache to expire entries after
	// a given duration after reading.
	ExpireAfterRead time.Duration

	// EvictInterval controls if a background go routine should be created
	// which automatically evicts entries that have expired. If not specified,
	// no background goroutine will be created.
	//
	// The background go routine runs with the provided frequency.
	// To avoid go routine leaks, use the close function when you're done with the cache.
	EvictInterval time.Duration

	// MaxSize limits the number of entries allowed in the cache.
	// If the limit is achieved, an eviction process will take place,
	// this means that eviction policies will be executed such as write
	// time, read time or random entry if no evection policy frees up
	// space.
	//
	// If the cache is sharded, MaxSize is applied to each shard,
	// meaning that the overall capacity will be MaxSize * ShardCount.
	MaxSize uint32

	// ShardCount indicates how many shards will be used by the cache.
	// This allows some degree of parallelism in read and writing to the cache.
	//
	// If the shard count is greater than 1, then HashCodeFunc must be provided
	// otherwise the constructor will panic.
	ShardCount uint32
}

// StringHashCodeFunc is a hash code function for strings which uses fnv.New32a
var StringHashCodeFunc = func(k any) int {
	h := fnv.New32a()
	if _, err := h.Write([]byte(k.(string))); err != nil {
		panic(err)
	}
	return int(h.Sum32())
}

func (c Options) expiresAfterRead() bool {
	return c.ExpireAfterRead > 0
}

func (c Options) expiresAfterWrite() bool {
	return c.ExpireAfterWrite > 0
}

// CacheOption describes an option that can configure the cache
type CacheOption func(Cache)

// LoadFunc represents a function that given a key, it returns a value or an error.
type LoadFunc func(any) (any, error)

type cacheEntry struct {
	value     any
	lastRead  time.Time
	lastWrite time.Time
}

// New instantiates a new cache
func (c Options) New() Cache {
	if c.Clock == nil {
		c.Clock = clock.New()
	}

	switch c.ShardCount {
	case 0, 1:
		c := &genericCache{
			Options: c,
			data:    map[any]*cacheEntry{},
			done:    make(chan struct{}),
			stats:   &stats.InternalStats{},
		}
		if c.EvictInterval > 0 {
			c.backgroundWg.Add(1)
			go c.runBackgroundEvict()
		}
		return c
	}

	if c.HashCodeFunc == nil {
		c.HashCodeFunc = StringHashCodeFunc
	}
	singleShardOptions := c
	singleShardOptions.ShardCount = 1
	s := &shardedCache{
		Options: c,
		shards:  make([]Cache, c.ShardCount),
	}
	for i := uint32(0); i < c.ShardCount; i++ {
		s.shards[i] = singleShardOptions.New()
	}
	return s
}

type shardedCache struct {
	shards []Cache
	Options
}

func (s *shardedCache) IsSharded() bool { return true }

func (s *shardedCache) Get(key any) (any, error) {
	val, err := s.shards[s.HashCodeFunc(key)%len(s.shards)].Get(key)
	return val, errors.Wrap(err, "shard get")
}

func (s *shardedCache) Put(key, value any) {
	s.shards[s.HashCodeFunc(key)%len(s.shards)].Put(key, value)
}

func (s *shardedCache) Invalidate(keys ...any) {
	for _, k := range keys {
		s.shards[s.HashCodeFunc(k)%len(s.shards)].Invalidate(k)
	}
}

func (s *shardedCache) InvalidateAll() {
	for _, shard := range s.shards {
		shard.InvalidateAll()
	}
}

func (s *shardedCache) Close() {
	for _, shard := range s.shards {
		shard.Close()
	}
}

func (s *shardedCache) Stats() Stats {
	statsSum := &stats.InternalStats{}
	for _, shard := range s.shards {
		switch typedCache := shard.(type) {
		case *genericCache:
			statsSum = statsSum.Add(typedCache.stats)
		default:
			panic(fmt.Sprintf("unsupported cache type %T", shard))
		}
	}
	return statsSum
}

// genericCache is an implementation of a cache where keys and values are
// of type any
type genericCache struct {
	data map[any]*cacheEntry

	done chan struct{}

	stats *stats.InternalStats
	Options

	backgroundWg sync.WaitGroup

	dataLock sync.RWMutex
}

func (s *genericCache) IsSharded() bool { return false }

func (g *genericCache) isExpired(entry *cacheEntry) (RemovalReason, bool) {
	if g.expiresAfterRead() && entry.lastRead.Add(g.ExpireAfterRead).Before(g.Clock.Now()) {
		return RemovalReasonReadExpired, true
	}
	if g.expiresAfterWrite() && entry.lastWrite.Add(g.ExpireAfterWrite).Before(g.Clock.Now()) {
		return RemovalReasonWriteExpired, true
	}
	return RemovalReasonExplicit, false
}

func (g *genericCache) Get(key any) (any, error) {
	g.dataLock.RLock()
	entry, exists := g.data[key]
	if !exists {
		g.dataLock.RUnlock()
		val, err := g.load(key)
		return val, errors.Wrap(err, "")
	}
	// Create a copy of the value to return to avoid concurrent updates
	toReturn := entry.value
	g.dataLock.RUnlock()

	if reason, expired := g.isExpired(entry); expired {
		g.concurrentEvict(key, reason)
		val, err := g.load(key)
		return val, errors.Wrap(err, "loading")
	}
	// It is possible that this will race. It will only be a problem
	// if the expiry thresholds have to be respected with a high
	// degree of precision (which is subjective).
	entry.lastRead = g.Clock.Now()
	g.stats.Hit()
	return toReturn, nil
}

func (g *genericCache) load(key any) (any, error) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()

	// It is possible that another call loaded the value for this key.
	// Let's do a double check if that was the case, since we have
	// the lock.
	if val, exists := g.data[key]; exists {
		g.stats.Hit()
		return val, nil
	} else if g.Load == nil {
		g.stats.Miss()
		return nil, errors.Wrap(ErrKeyNotFound, "miss")
	}

	loadStartTime := g.Clock.Now()
	val, err := g.Load(key)
	if err != nil {
		g.stats.LoadError()
		return nil, errors.Wrapf(err, "failed to load key %v", key)
	}
	g.stats.LoadTime(g.Clock.Now().Sub(loadStartTime))
	g.stats.LoadSuccess()
	g.internalPut(key, val)
	return val, nil
}

func (g *genericCache) concurrentEvict(key any, reason RemovalReason) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	g.evict(key, reason)
}

func (g *genericCache) runBackgroundEvict() {
	ticker := g.Clock.Ticker(g.EvictInterval)
	defer ticker.Stop()
	defer g.backgroundWg.Done()
	for {
		select {
		case <-g.done:
			return
		case <-ticker.C:
			g.backgroundEvict()
		}
	}
}

// backgroundEvict performs a scan of the cache in search for expired entries
// and evicts them
func (g *genericCache) backgroundEvict() {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	for key, entry := range g.data {
		if reason, expired := g.isExpired(entry); expired {
			// TODO: There's a possibility that we want to evict
			// in a go routine so we can get through
			// all expired entries as fast as possible without
			// having to sequentially wait for removal listeners.
			g.evict(key, reason)
		}
	}
}

func (g *genericCache) evict(key any, reason RemovalReason) {
	val, exists := g.data[key]
	if !exists {
		return
	}
	g.stats.Eviction()
	delete(g.data, key)

	if len(g.RemovalListeners) == 0 {
		return
	}
	notification := RemovalNotification{
		Key:    key,
		Value:  val.value,
		Reason: reason,
	}
	// Each removal listener is called on its own goroutine
	// so a slow one does not affect the others.
	// This could potentially be early optimization, but seems
	// simple enough.
	var listenerWg sync.WaitGroup
	listenerWg.Add(len(g.RemovalListeners))
	for i := range g.RemovalListeners {
		listener := g.RemovalListeners[i]
		go func() {
			defer listenerWg.Done()
			listener(notification)
		}()
	}
	listenerWg.Wait()
}

// internalPut actually saves the values into the internal structures.
// It does not handle any synchronization, leaving that to the caller.
func (g *genericCache) internalPut(key any, value any) {
	if g.MaxSize > 0 && len(g.data) >= int(g.MaxSize) {
		// If eviction is needed it currently removes a random entry,
		// since maps do not have a deterministic order.
		// TODO: Apply smarter eviction policies if available
		for toEvict := range g.data {
			g.evict(toEvict, RemovalReasonSize)
			break
		}
	}
	now := g.Clock.Now()
	g.data[key] = &cacheEntry{
		value:     value,
		lastRead:  now,
		lastWrite: now,
	}
}

// preWriteCleanup does a pass through all entries to assess if any are expired
// and should be removed.
//
// If background cleanup os enabled, this becomes a noop.
func (g *genericCache) preWriteCleanup() {
	if g.EvictInterval > 0 {
		return
	}
	for key := range g.data {
		if reason, expired := g.isExpired(g.data[key]); expired {
			g.evict(key, reason)
		}
	}
}

func (g *genericCache) Put(key, value any) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
	g.preWriteCleanup()
	if _, exists := g.data[key]; exists {
		g.evict(key, RemovalReasonReplaced)
	}
	g.internalPut(key, value)
}

func (g *genericCache) Invalidate(keys ...any) {
	g.dataLock.Lock()
	defer g.dataLock.Unlock()
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

func (g *genericCache) Close() {
	close(g.done)
	// Ensure that we wait for all background tasks to complete.
	g.backgroundWg.Wait()
}

func (g *genericCache) Stats() Stats {
	return g.stats
}
