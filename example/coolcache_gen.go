// Code generated by github.com/Hartimer/loadingcache/cmd/typedcache, DO NOT EDIT.
package example

import (
	"time"

	"github.com/Hartimer/loadingcache"
	"github.com/benbjohnson/clock"
)

type CoolCache interface {
	Get(key Name) (int64, error)
	Put(key Name, value int64)
	Invalidate(key Name, keys ...Name)
	InvalidateAll()
}

type CoolCacheOptions struct {
	Clock            clock.Clock
	ExpireAfterWrite time.Duration
	ExpireAfterRead  time.Duration
	Load             LoadFunc
	MaxSize          int32
	RemovalListeners []RemovalListener
	ShardCount       int
	HashCodeFunc     func(key Name) int
}

func (c CoolCacheOptions) expiresAfterRead() bool {
	return c.ExpireAfterRead > 0
}

func (c CoolCacheOptions) expiresAfterWrite() bool {
	return c.ExpireAfterWrite > 0
}

type LoadFunc func(Name) (int64, error)

type RemovalNotification struct {
	Key    Name
	Value  int64
	Reason loadingcache.RemovalReason
}

type RemovalListener func(RemovalNotification)

type internalImplementation struct {
	genericCache loadingcache.Cache
}

func NewCoolCache(options CoolCacheOptions) CoolCache {
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
			return options.Load(key.(Name))
		}
	}

	if options.HashCodeFunc != nil {
		finalOptions.HashCodeFunc = func(key interface{}) int {
			return options.HashCodeFunc(key.(Name))
		}
	}

	for i := range options.RemovalListeners {
		removalListener := options.RemovalListeners[i]
		finalOptions.RemovalListeners[i] = func(notification loadingcache.RemovalNotification) {
			typedNotification := RemovalNotification{
				Key:    notification.Key.(Name),
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

func (i *internalImplementation) Get(key Name) (int64, error) {
	val, err := i.genericCache.Get(key)
	if err != nil {
		return 0, err
	}
	typedVal, ok := val.(int64)
	if !ok {
		panic(loadingcache.ErrTypeMismatch(typedVal, val))
	}
	return typedVal, nil
}

func (i *internalImplementation) Put(key Name, value int64) {
	i.genericCache.Put(key, value)
}

func (i *internalImplementation) Invalidate(key Name, keys ...Name) {
	genericKeys := make([]interface{}, len(keys))
	for i, k := range keys {
		genericKeys[i] = k
	}
	i.genericCache.Invalidate(key, genericKeys...)
}

func (i *internalImplementation) InvalidateAll() {
	i.genericCache.InvalidateAll()
}
