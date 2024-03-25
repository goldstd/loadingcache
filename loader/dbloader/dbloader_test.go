package dbloader

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goldstd/loadingcache"
	"github.com/goldstd/loadingcache/loader/redisloader"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisDBLoad(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/gcache")
	assert.Nil(t, err)
	defer db.Close()

	dbLoader := NewDBLoader(db, "select v from gcache where k = ?", Value{})

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	rdb.Del(context.Background(), "k1")

	exp := 10 * time.Millisecond
	redisLoader := redisloader.New(rdb, exp).
		WithLoader(dbLoader)
	c := loadingcache.Config{
		AsyncLoad:        true,
		Load:             redisLoader,
		ExpireAfterWrite: exp,
	}.Build()
	obj, err := c.Get("k1")
	assert.Nil(t, err)
	assert.Equal(t, obj.(Value), Value{V: "mysql value"})
}

type Value struct {
	V string
}

func TestDBLoad(t *testing.T) {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/gcache")
	assert.Nil(t, err)
	defer db.Close()

	exp := 10 * time.Millisecond

	loader := NewDBLoader(db, "select v from gcache where k = ?", Value{})
	c := loadingcache.Config{
		AsyncLoad:        true,
		Load:             loader,
		ExpireAfterWrite: exp,
	}.Build()
	obj, err := c.Get("k1")
	assert.Nil(t, err)
	assert.Equal(t, obj.(Value), Value{V: "mysql value"})
}
