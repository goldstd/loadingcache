package redisloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/goldstd/loadingcache"
	"github.com/redis/go-redis/v9"
)

type Marshaller interface {
	MarshalRedis(any) (string, error)
	UnmarshalRedis(string) (any, error)
}

type KeyAdapter interface {
	AdaptKey(any) (string, error)
}

type RedisLoader struct {
	redis.UniversalClient
	Marshaller
	loadingcache.Loader
	KeyAdapter
	Expiration time.Duration
}

func New(client redis.UniversalClient, exp time.Duration) *RedisLoader {
	return &RedisLoader{UniversalClient: client, Expiration: exp}
}

func (r *RedisLoader) WithMarshaller(marshaller Marshaller) *RedisLoader {
	r.Marshaller = marshaller
	return r
}

func (r *RedisLoader) WithLoader(loader loadingcache.Loader) *RedisLoader {
	r.Loader = loader
	if r.Marshaller == nil {
		r.Marshaller = &JSONMarshaller{}
	}
	return r
}

func (r *RedisLoader) WithKeyAdapter(keyAdapter KeyAdapter) *RedisLoader {
	r.KeyAdapter = keyAdapter
	return r
}

func (r *RedisLoader) Load(key any, cache loadingcache.Cache) (any, error) {
	if r.KeyAdapter != nil {
		var err error
		key, err = r.AdaptKey(key)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()

	result, err := r.Get(ctx, key.(string)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	if r.Loader == nil {
		return r.UnmarshalRedis(result)
	}

	value, err := r.Loader.Load(key, cache)
	if err != nil {
		return nil, err
	}

	redisValue, err := r.MarshalRedis(value)
	if err != nil {
		return nil, err
	}

	if err := r.Set(ctx, key.(string), redisValue, r.Expiration).Err(); err != nil {
		return nil, fmt.Errorf("redis set %s: %w", key.(string), err)
	}

	return value, nil
}

type JSONMarshaller struct{}

func (JSONMarshaller) MarshalRedis(a any) (string, error) {
	data, err := json.Marshal(a)
	return string(data), err
}

func (JSONMarshaller) UnmarshalRedis(s string) (any, error) {
	var data any
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
