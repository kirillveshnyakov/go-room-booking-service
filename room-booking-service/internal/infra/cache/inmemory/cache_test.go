package inmemory

import (
	"sync"
	"testing"
	"time"
)

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := New[string, int](
		WithCapacity[string, int](2),
		WithGCInterval[string, int](0),
	)
	t.Cleanup(cache.Close)

	cache.Set("first", 1, 0)
	cache.Set("second", 2, 0)
	if value, ok := cache.Get("first"); !ok || value != 1 {
		t.Fatalf("Get(first) = (%d, %t), want (1, true)", value, ok)
	}

	cache.Set("third", 3, 0)
	if _, ok := cache.Get("second"); ok {
		t.Fatal("least recently used entry was not evicted")
	}
	if _, ok := cache.Get("first"); !ok {
		t.Fatal("recently used entry was evicted")
	}
	if _, ok := cache.Get("third"); !ok {
		t.Fatal("new entry is missing")
	}

	cache.Set("first", 10, 0)
	if value, ok := cache.Get("first"); !ok || value != 10 {
		t.Fatalf("updated entry = (%d, %t), want (10, true)", value, ok)
	}
	if _, ok := cache.Get("third"); !ok {
		t.Fatal("updating an entry evicted another entry")
	}
}

func TestCacheWithoutDefaultTTL(t *testing.T) {
	cache := New[string, int](
		WithDefaultTTL[string, int](0),
		WithGCInterval[string, int](0),
	)
	t.Cleanup(cache.Close)

	cache.Set("permanent", 1, UseDefaultTTL)
	cache.Set("temporary", 2, 100*time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	if value, ok := cache.Get("permanent"); !ok || value != 1 {
		t.Fatalf("permanent entry = (%d, %t), want (1, true)", value, ok)
	}
	if _, ok := cache.Get("temporary"); ok {
		t.Fatal("expired entry was returned")
	}
}

func TestCachePeriodicCleanupAndClose(t *testing.T) {
	cache := New[string, int](
		WithDefaultTTL[string, int](100*time.Millisecond),
		WithGCInterval[string, int](10*time.Millisecond),
	)
	t.Cleanup(cache.Close)

	cache.Set("expired", 1, UseDefaultTTL)
	cache.Set("permanent", 2, 0)
	deadline := time.Now().Add(2 * time.Second)
	for {
		cache.mu.Lock()
		_, expiredPresent := cache.kv["expired"]
		_, permanentPresent := cache.kv["permanent"]
		cache.mu.Unlock()
		if !permanentPresent {
			t.Fatal("periodic cleanup removed a permanent entry")
		}
		if !expiredPresent {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("periodic cleanup did not remove expired entry")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if value, ok := cache.Get("permanent"); !ok || value != 2 {
		t.Fatalf("permanent entry after cleanup = (%d, %t), want (2, true)", value, ok)
	}

	cache.Close()
	cache.Close()
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := New[int, int](
		WithCapacity[int, int](16),
		WithGCInterval[int, int](0),
	)
	t.Cleanup(cache.Close)

	var workers sync.WaitGroup
	for worker := range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range 100 {
				key := worker*100 + i
				cache.Set(key, i, 0)
				cache.Get(key)
				cache.Delete(key)
			}
		}()
	}
	workers.Wait()
}

func TestSetPromotesExistingKey(t *testing.T) {
	cache := New[string, int](
		WithCapacity[string, int](2),
		WithGCInterval[string, int](0),
	)
	t.Cleanup(cache.Close)

	cache.Set("A", 1, 0)
	cache.Set("B", 2, 0)
	cache.Set("A", 10, 0)
	cache.Set("C", 3, 0)

	if _, ok := cache.Get("B"); ok {
		t.Fatal("B should be evicted after Set promotes A")
	}
	if value, ok := cache.Get("A"); !ok || value != 10 {
		t.Fatalf("Get(A) = (%d, %t), want (10, true)", value, ok)
	}
	if value, ok := cache.Get("C"); !ok || value != 3 {
		t.Fatalf("Get(C) = (%d, %t), want (3, true)", value, ok)
	}
}

func TestSetRefreshesTTL(t *testing.T) {
	cache := New[string, int](WithGCInterval[string, int](0))
	t.Cleanup(cache.Close)

	const originalTTL = 100 * time.Millisecond
	const refreshedTTL = 500 * time.Millisecond
	originalDeadline := time.Now().Add(originalTTL)
	cache.Set("key", 1, originalTTL)
	cache.Set("key", 2, refreshedTTL)
	refreshedDeadline := time.Now().Add(refreshedTTL)

	if wait := time.Until(originalDeadline.Add(20 * time.Millisecond)); wait > 0 {
		time.Sleep(wait)
	}
	if value, ok := cache.Get("key"); !ok || value != 2 {
		t.Fatalf("Get(key) after original TTL = (%d, %t), want (2, true)", value, ok)
	}

	if wait := time.Until(refreshedDeadline.Add(20 * time.Millisecond)); wait > 0 {
		time.Sleep(wait)
	}
	if _, ok := cache.Get("key"); ok {
		t.Fatal("key is still available after refreshed TTL")
	}
}

func TestConcurrentSameKey(t *testing.T) {
	cache := New[string, int](
		WithCapacity[string, int](2),
		WithGCInterval[string, int](0),
	)
	t.Cleanup(cache.Close)

	start := make(chan struct{})
	var workers sync.WaitGroup
	for worker := range 9 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for i := range 100 {
				switch worker % 3 {
				case 0:
					cache.Set("shared", i, 0)
				case 1:
					cache.Get("shared")
				case 2:
					cache.Delete("shared")
				}
			}
		}()
	}
	close(start)
	workers.Wait()

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.head.prev != nil || cache.tail.next != nil {
		t.Fatal("sentinel pointers are corrupted")
	}
	count := 0
	previous := cache.head
	for node := cache.head.next; node != cache.tail; node = node.next {
		if node == nil || node.prev != previous || node.next == nil {
			t.Fatal("linked list pointers are corrupted")
		}
		if cache.kv[node.key] != node {
			t.Fatal("linked list and map disagree")
		}
		previous = node
		count++
		if count > len(cache.kv) {
			t.Fatal("linked list has extra elements or a cycle")
		}
	}
	if cache.tail.prev != previous || count != len(cache.kv) {
		t.Fatalf("linked list has %d elements, map has %d", count, len(cache.kv))
	}
}

func TestSetTTLSemantics(t *testing.T) {
	const shortTTL = 100 * time.Millisecond
	tests := []struct {
		name       string
		defaultTTL time.Duration
		ttl        time.Duration
		update     bool
		initialTTL time.Duration
		wantTTL    time.Duration
	}{
		{name: "zero overrides default", defaultTTL: shortTTL, ttl: 0},
		{name: "explicit TTL", defaultTTL: time.Hour, ttl: shortTTL, wantTTL: shortTTL},
		{name: "temporary becomes permanent", defaultTTL: shortTTL, initialTTL: shortTTL, update: true, ttl: 0},
		{name: "permanent becomes temporary", initialTTL: 0, update: true, ttl: shortTTL, wantTTL: shortTTL},
		{name: "named default", defaultTTL: shortTTL, ttl: UseDefaultTTL, wantTTL: shortTTL},
		{name: "other negative uses default", defaultTTL: shortTTL, ttl: -2 * time.Second, wantTTL: shortTTL},
		{name: "zero default is permanent", defaultTTL: 0, ttl: UseDefaultTTL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cache := New[string, int](
				WithDefaultTTL[string, int](test.defaultTTL),
				WithGCInterval[string, int](0),
			)
			t.Cleanup(cache.Close)
			if test.update {
				cache.Set("key", 1, test.initialTTL)
			}
			before := time.Now()
			cache.Set("key", 2, test.ttl)
			after := time.Now()
			cache.mu.Lock()
			expiration := cache.kv["key"].expiration
			cache.mu.Unlock()
			if test.wantTTL == 0 {
				if !expiration.IsZero() {
					t.Fatalf("permanent entry has expiration %v", expiration)
				}
			} else if expiration.Before(before.Add(test.wantTTL)) || expiration.After(after.Add(test.wantTTL)) {
				t.Fatalf("expiration %v does not match TTL %v", expiration, test.wantTTL)
			}
			time.Sleep(150 * time.Millisecond)
			value, ok := cache.Get("key")
			if test.wantTTL == 0 {
				if !ok || value != 2 {
					t.Fatalf("permanent entry = (%d, %t), want (2, true)", value, ok)
				}
			} else if ok {
				t.Fatal("expired entry is still available")
			}
		})
	}
}
