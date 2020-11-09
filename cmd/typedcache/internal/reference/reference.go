// Package reference holds a sample rendered typed cache which is used
// to facilitate updating the template.
//
// Whenever changes are made to the typed cache implementation, this file
// makes it easy to do a quick sanity check.
// The contents of this file are not meant to be used by anyone!
package reference

import (
	"time"

	"github.com/Hartimer/loadingcache"
	"github.com/benbjohnson/clock"
)

type TypedCache interface {
	Get(key string) (int64, error)
	Put(key string, value int64)
	Invalidate(key string, keys ...string)
	InvalidateAll()
}

type CacheOptions struct {
	Clock            clock.Clock
	ExpireAfterWrite time.Duration
	ExpireAfterRead  time.Duration
	Load             LoadFunc
	MaxSize          int32
	RemovalListeners []RemovalListener
	ShardCount       int
	HashCodeFunc     func(key string) int
}

func (c CacheOptions) expiresAfterRead() bool {
	return c.ExpireAfterRead > 0
}

func (c CacheOptions) expiresAfterWrite() bool {
	return c.ExpireAfterWrite > 0
}

type LoadFunc func(string) (int64, error)

type RemovalNotification struct {
	Key    string
	Value  int64
	Reason loadingcache.RemovalReason
}

type RemovalListener func(RemovalNotification)

type internalImplementation struct {
	genericCache loadingcache.Cache
}

func NewTypedCache(options CacheOptions) TypedCache {
	finalOptions := loadingcache.CacheOptions{
		Clock:            options.Clock,
		ExpireAfterWrite: options.ExpireAfterWrite,
		ExpireAfterRead:  options.ExpireAfterRead,
		MaxSize:          options.MaxSize,
		RemovalListeners: make([]loadingcache.RemovalListener, len(options.RemovalListeners)),
		ShardCount:       options.ShardCount,
	}

	if options.Load != nil {
		finalOptions.Load = func(key interface{}) (interface{}, error) {
			return options.Load(key.(string))
		}
	}

	if options.HashCodeFunc != nil {
		finalOptions.HashCodeFunc = func(key interface{}) int {
			return options.HashCodeFunc(key.(string))
		}
	}

	for i := range options.RemovalListeners {
		removalListener := options.RemovalListeners[i]
		finalOptions.RemovalListeners[i] = func(notification loadingcache.RemovalNotification) {
			typedNotification := RemovalNotification{
				Key:    notification.Key.(string),
				Value:  notification.Value.(int64),
				Reason: notification.Reason,
			}
			removalListener(typedNotification)
		}
	}

	return &internalImplementation{
		genericCache: loadingcache.New(finalOptions),
	}
}

func (i *internalImplementation) Get(key string) (int64, error) {
	val, err := i.genericCache.Get(key)
	if err != nil {
		return 0, err
	}
	typedVal, ok := val.(int64)
	if !ok {
		return loadingcache.ErrTypeMismatch(typedVal, val)
	}
	return typedVal, nil
}

func (i *internalImplementation) Put(key string, value int64) {
	i.genericCache.Put(key, value)
}

func (i *internalImplementation) Invalidate(key string, keys ...string) {
	genericKeys := make([]interface{}, len(keys))
	for i, k := range keys {
		genericKeys[i] = k
	}
	i.genericCache.Invalidate(key, genericKeys...)
}

func (i *internalImplementation) InvalidateAll() {
	i.genericCache.InvalidateAll()
}
