package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Entry struct {
	Key   string
	Value []byte
	TTL   time.Duration
}

type EntryFailure struct {
	Index int
	Err   error
}

type Cache struct {
	rdb *redis.Client
}

func NewRedisCache(client *redis.Client) (*Cache, error) {
	if client == nil {
		return nil, fmt.Errorf("redis cache - new: %w", ErrClientRequired)
	}
	return &Cache{
		rdb: client,
	}, nil
}

func (cache *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	bytes, err := cache.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis cache - get: %w", err)
	}

	return bytes, true, nil
}

func (cache *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl < 0 {
		return fmt.Errorf("redis cache - set: %w", ErrInvalidTTL)
	}
	if err := cache.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis cache - set: %w", err)
	}
	return nil
}

func (cache *Cache) Delete(ctx context.Context, key string) error {
	if err := cache.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis cache - delete: %w", err)
	}
	return nil
}

func (cache *Cache) GetMany(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return map[string][]byte{}, nil
	}
	values, err := cache.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis cache - get many: %w", err)
	}

	result := make(map[string][]byte, len(values))
	for i, value := range values {
		if value == nil {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("redis cache - get many: unexpected value type %T", value)
		}
		result[keys[i]] = []byte(text)
	}

	return result, nil
}

func (cache *Cache) SetMany(ctx context.Context, entries []Entry) ([]EntryFailure, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	for i, entry := range entries {
		if entry.TTL < 0 {
			return nil, fmt.Errorf("redis cache - set many: entry %d: %w", i, ErrInvalidTTL)
		}
	}
	commands, err := cache.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i := range entries {
			pipe.Set(ctx, entries[i].Key, entries[i].Value, entries[i].TTL)
		}
		return nil
	})
	var failures []EntryFailure
	for i, command := range commands {
		if command.Err() != nil {
			failures = append(failures, EntryFailure{
				Index: i,
				Err:   fmt.Errorf("redis cache - set many: entry %d: %w", i, command.Err()),
			})
		}
	}
	if err != nil {
		return failures, fmt.Errorf("redis cache - set many: %w", err)
	}
	return failures, nil
}
