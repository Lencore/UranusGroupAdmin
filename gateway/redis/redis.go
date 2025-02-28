package redis

import (
	"app/dto"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type Redis struct {
	client    *redis.Client
	namespace string
}

func NewClient(config dto.Config) (*Redis, error) {
	opt, err := redis.ParseURL(config.Redis.Addr)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ping := client.Ping(context.Background())

	_, err = ping.Result()
	if err != nil {
		return nil, fmt.Errorf("redisClient: %s", err.Error())
	}

	return &Redis{
		client:    client,
		namespace: config.Redis.NameSpace,
	}, nil
}

func (r *Redis) Client() *redis.Client {
	return r.client
}

func (r *Redis) keyWithNamespace(key string) string {
	return r.namespace + ":" + key
}

func (r *Redis) Has(key string) bool {
	cacheKey := r.keyWithNamespace(key)

	get := r.client.Get(context.Background(), cacheKey)

	return get.Err() == nil
}

func (r *Redis) Set(key string, value interface{}) error {
	cacheKey := r.keyWithNamespace(key)

	set := r.client.Set(context.Background(), cacheKey, value, 0)

	return set.Err()
}

func (r *Redis) SetWithTTL(key string, value interface{}, expiration time.Duration) error {
	cacheKey := r.keyWithNamespace(key)

	set := r.client.Set(context.Background(), cacheKey, value, expiration)

	return set.Err()
}

func (r *Redis) GetString(key string) (string, error) {
	cacheKey := r.keyWithNamespace(key)
	get := r.client.Get(context.Background(), cacheKey)

	return get.Result()
}

func (r *Redis) GetInt(key string) (int, error) {
	cacheKey := r.keyWithNamespace(key)
	get := r.client.Get(context.Background(), cacheKey)

	return get.Int()
}

func (r *Redis) GetInt64(key string) (int64, error) {
	cacheKey := r.keyWithNamespace(key)
	get := r.client.Get(context.Background(), cacheKey)

	return get.Int64()
}

func (r *Redis) GetBytes(key string) ([]byte, error) {
	cacheKey := r.keyWithNamespace(key)
	get := r.client.Get(context.Background(), cacheKey)

	return get.Bytes()
}

func (r *Redis) Del(key string) error {
	cacheKey := r.keyWithNamespace(key)
	del := r.client.Del(context.Background(), cacheKey)

	return del.Err()
}
