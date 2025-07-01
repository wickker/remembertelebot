package ristrettocache

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/ristretto/v2"
)

type Cache[T any] struct {
	Cache *ristretto.Cache[string, string]
}

func NewCache[T any]() (*Cache[T], error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, string]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 30, // maximum cost of cache (1GB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		return nil, err
	}

	return &Cache[T]{
		Cache: cache,
	}, nil
}

func (c *Cache[T]) Set(key string, val T) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to set cache [key: %s][value: %s]: %w", key, val, err)
	}
	cost := len(string(bytes))
	c.Cache.Set(key, string(bytes), int64(cost))
	c.Cache.Wait()
	return nil
}

func (c *Cache[T]) Get(key string) (T, error) {
	var val T
	str, found := c.Cache.Get(key)
	if !found {
		return val, nil
	}
	if err := json.Unmarshal([]byte(str), &val); err != nil {
		return val, fmt.Errorf("failed to get cache [key: %s]: %w", key, err)
	}
	return val, nil
}

func (c *Cache[T]) Delete(key string) {
	c.Cache.Del(key)
}
