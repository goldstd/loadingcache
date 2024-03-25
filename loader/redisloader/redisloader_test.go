package redisloader_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goldstd/loadingcache"
	"github.com/goldstd/loadingcache/loader/redisloader"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisCache(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()
	rdb.Del(context.Background(), "k1")

	exp := 10 * time.Millisecond
	redisLoader := redisloader.New(rdb, exp).WithLoader(&randomLoader{})
	c := loadingcache.Config{
		AsyncLoad:        true,
		Load:             redisLoader,
		ExpireAfterWrite: exp,
	}.Build()
	obj, err := c.Get("k1")
	assert.Nil(t, err)
	assert.Equal(t, obj, "hello1")

	obj, err = c.Get("k1")
	assert.Nil(t, err)
	assert.Equal(t, obj, "hello1")

	time.Sleep(exp)
	obj, err = c.Get("k1")

	// 过期了，但是还是会拿到老值，会触发异步刷新
	assert.Nil(t, err)
	assert.Equal(t, obj, "hello1")
	time.Sleep(exp)
	obj, err = c.Get("k1")
	assert.Nil(t, err)
	assert.Equal(t, obj, "hello2")
}

type randomLoader struct {
	visitorNum atomic.Int32
}

func (r *randomLoader) Load(key any) (any, error) {
	n := r.visitorNum.Add(1)
	return fmt.Sprintf("hello%d", n), nil
}
