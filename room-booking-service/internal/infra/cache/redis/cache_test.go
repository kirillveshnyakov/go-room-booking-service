//go:build integration

package redis

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/util/testredis"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testRedis *testredis.Redis

func TestMain(m *testing.M) {
	os.Exit(runRedisTests(m))
}

func runRedisTests(m *testing.M) int {
	var err error
	testRedis, err = testredis.Start(context.Background())
	if err != nil {
		log.Printf("Redis integration tests require available Docker: %v", err)
		return 1
	}
	code := m.Run()
	if err = testRedis.Close(); err != nil {
		log.Printf("Redis integration test cleanup: %v", err)
		return 1
	}
	return code
}

func newTestCache(t *testing.T) (context.Context, *Cache, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	cache, err := NewRedisCache(testRedis.Client)
	require.NoError(t, err)
	return ctx, cache, uuid.NewString() + ":"
}

func TestCacheGetSet(t *testing.T) {
	tests := []struct {
		name    string
		initial *Entry
		value   []byte
		ttl     time.Duration
	}{
		{name: "text", value: []byte("value"), ttl: time.Minute},
		{name: "binary", value: []byte{0, 1, 127, 128, 255}, ttl: time.Minute},
		{name: "permanent", value: []byte("value"), ttl: 0},
		{
			name:    "overwrite",
			initial: &Entry{Value: []byte("old"), TTL: time.Minute},
			value:   []byte("updated"), ttl: time.Minute,
		},
		{
			name:    "refresh TTL",
			initial: &Entry{Value: []byte("old"), TTL: time.Minute},
			value:   []byte("updated"), ttl: 2 * time.Minute,
		},
		{
			name:    "remove expiration",
			initial: &Entry{Value: []byte("old"), TTL: time.Minute},
			value:   []byte("permanent"), ttl: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cache, prefix := newTestCache(t)
			key := prefix + "value"
			var initialTTL time.Duration
			if test.initial != nil {
				require.NoError(t, cache.Set(ctx, key, test.initial.Value, test.initial.TTL))
				var err error
				initialTTL, err = testRedis.Client.PTTL(ctx, key).Result()
				require.NoError(t, err)
				require.Positive(t, initialTTL)
			}
			require.NoError(t, cache.Set(ctx, key, test.value, test.ttl))
			got, found, err := cache.Get(ctx, key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, test.value, got)
			remainingTTL, err := testRedis.Client.PTTL(ctx, key).Result()
			require.NoError(t, err)
			if test.ttl == 0 {
				require.Equal(t, time.Duration(-1), remainingTTL)
			} else {
				require.Positive(t, remainingTTL)
				require.LessOrEqual(t, remainingTTL, test.ttl)
				if test.initial != nil && test.ttl > test.initial.TTL {
					require.Greater(t, remainingTTL, initialTTL)
				}
			}
		})
	}
}

func TestCacheMiss(t *testing.T) {
	ctx, cache, prefix := newTestCache(t)
	value, found, err := cache.Get(ctx, prefix+"missing")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, value)
}

func TestCacheDelete(t *testing.T) {
	tests := []struct {
		name     string
		present  bool
		attempts int
	}{
		{name: "existing key", present: true, attempts: 1},
		{name: "missing key", attempts: 1},
		{name: "repeated deletion", present: true, attempts: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cache, prefix := newTestCache(t)
			key := prefix + "deleted"
			if test.present {
				require.NoError(t, cache.Set(ctx, key, []byte("value"), 0))
			}
			for range test.attempts {
				require.NoError(t, cache.Delete(ctx, key))
				value, found, err := cache.Get(ctx, key)
				require.NoError(t, err)
				require.False(t, found)
				require.Nil(t, value)
			}
		})
	}
}

func TestCacheExpiration(t *testing.T) {
	ctx, cache, prefix := newTestCache(t)
	key := prefix + "expiring"
	require.NoError(t, cache.Set(ctx, key, []byte("value"), 250*time.Millisecond))
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		value, found, err := cache.Get(ctx, key)
		require.NoError(collect, err)
		require.False(collect, found)
		require.Nil(collect, value)
	}, 3*time.Second, 25*time.Millisecond)
}

func TestCacheGetMany(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want map[string][]byte
	}{
		{name: "all exist", keys: []string{"a", "b", "empty"}, want: map[string][]byte{"a": {0, 255}, "b": []byte("value"), "empty": {}}},
		{name: "some exist", keys: []string{"missing", "b"}, want: map[string][]byte{"b": []byte("value")}},
		{name: "none exist", keys: []string{"missing"}, want: map[string][]byte{}},
		{name: "empty keys", want: map[string][]byte{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cache, prefix := newTestCache(t)
			for key, value := range map[string][]byte{"a": {0, 255}, "b": []byte("value"), "empty": {}} {
				require.NoError(t, cache.Set(ctx, prefix+key, value, 0))
			}
			keys := make([]string, len(test.keys))
			for i, key := range test.keys {
				keys[i] = prefix + key
			}
			want := make(map[string][]byte, len(test.want))
			for key, value := range test.want {
				want[prefix+key] = value
			}
			got, err := cache.GetMany(ctx, keys)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestCacheSetMany(t *testing.T) {
	tests := []struct {
		name        string
		entries     []Entry
		expiringKey string
	}{
		{
			name: "multiple entries",
			entries: []Entry{
				{Key: "first", Value: []byte("first"), TTL: time.Minute},
				{Key: "binary", Value: []byte{0, 255}, TTL: time.Minute},
			},
		},
		{
			name: "different TTLs",
			entries: []Entry{
				{Key: "short", Value: []byte("short"), TTL: 500 * time.Millisecond},
				{Key: "long", Value: []byte("long"), TTL: time.Minute},
				{Key: "permanent", Value: []byte{0, 255}, TTL: 0},
			},
			expiringKey: "short",
		},
		{name: "nil entries"},
		{name: "empty entries", entries: []Entry{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cache, prefix := newTestCache(t)
			var entries []Entry
			if test.entries != nil {
				entries = make([]Entry, len(test.entries))
				copy(entries, test.entries)
				for i := range entries {
					entries[i].Key = prefix + entries[i].Key
				}
			}
			failures, err := cache.SetMany(ctx, entries)
			require.NoError(t, err)
			require.Empty(t, failures)
			for _, entry := range entries {
				got, found, err := cache.Get(ctx, entry.Key)
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, entry.Value, got)
				remainingTTL, err := testRedis.Client.PTTL(ctx, entry.Key).Result()
				require.NoError(t, err)
				if entry.TTL == 0 {
					require.Equal(t, time.Duration(-1), remainingTTL)
				}
			}
			if test.expiringKey == "" {
				return
			}
			expiringKey := prefix + test.expiringKey
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				_, found, err := cache.Get(ctx, expiringKey)
				require.NoError(collect, err)
				require.False(collect, found)
			}, 3*time.Second, 25*time.Millisecond)
			for _, entry := range entries {
				if entry.Key == expiringKey {
					continue
				}
				got, found, err := cache.Get(ctx, entry.Key)
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, entry.Value, got)
			}
		})
	}
}

func TestCacheInvalidTTL(t *testing.T) {
	for _, ttl := range []time.Duration{-1, -time.Second} {
		t.Run(ttl.String(), func(t *testing.T) {
			ctx, cache, prefix := newTestCache(t)
			require.ErrorIs(t, cache.Set(ctx, prefix+"single", []byte("value"), ttl), ErrInvalidTTL)
			failures, err := cache.SetMany(ctx, []Entry{
				{Key: prefix + "valid", Value: []byte("value"), TTL: time.Minute},
				{Key: prefix + "invalid", Value: []byte("value"), TTL: ttl},
			})
			require.ErrorIs(t, err, ErrInvalidTTL)
			require.Empty(t, failures)
			got, err := cache.GetMany(ctx, []string{prefix + "single", prefix + "valid", prefix + "invalid"})
			require.NoError(t, err)
			require.Empty(t, got)
		})
	}
}

func TestCacheUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	client := redisclient.NewClient(&redisclient.Options{
		Addr:                  address,
		MaxRetries:            -1,
		DialTimeout:           250 * time.Millisecond,
		ContextTimeoutEnabled: true,
	})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	cache, err := NewRedisCache(client)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	value, found, err := cache.Get(ctx, uuid.NewString())
	require.Error(t, err)
	require.False(t, found)
	require.Nil(t, value)
}

func TestCacheCanceledContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, context.Context, *Cache, string) error
	}{
		{name: "Get", run: func(_ *testing.T, ctx context.Context, cache *Cache, key string) error {
			_, _, err := cache.Get(ctx, key)
			return err
		}},
		{name: "Set", run: func(_ *testing.T, ctx context.Context, cache *Cache, key string) error {
			return cache.Set(ctx, key, []byte("changed"), 0)
		}},
		{name: "Delete", run: func(_ *testing.T, ctx context.Context, cache *Cache, key string) error {
			return cache.Delete(ctx, key)
		}},
		{name: "GetMany", run: func(_ *testing.T, ctx context.Context, cache *Cache, key string) error {
			_, err := cache.GetMany(ctx, []string{key})
			return err
		}},
		{name: "SetMany", run: func(t *testing.T, ctx context.Context, cache *Cache, key string) error {
			failures, err := cache.SetMany(ctx, []Entry{{Key: key, Value: []byte("changed")}})
			require.Len(t, failures, 1)
			require.Equal(t, 0, failures[0].Index)
			require.ErrorIs(t, failures[0].Err, context.Canceled)
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cache, prefix := newTestCache(t)
			key := prefix + "existing"
			require.NoError(t, cache.Set(ctx, key, []byte("unchanged"), 0))
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			require.ErrorIs(t, test.run(t, canceled, cache, key), context.Canceled)
			value, found, err := cache.Get(ctx, key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, []byte("unchanged"), value)
		})
	}
}

func TestNewRedisCache(t *testing.T) {
	tests := []struct {
		name    string
		client  *redisclient.Client
		wantErr error
	}{
		{name: "nil client", wantErr: ErrClientRequired},
		{name: "valid client", client: testRedis.Client},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache, err := NewRedisCache(test.client)
			require.ErrorIs(t, err, test.wantErr)
			if test.wantErr != nil {
				require.Nil(t, cache)
			} else {
				require.NotNil(t, cache)
			}
		})
	}
}
