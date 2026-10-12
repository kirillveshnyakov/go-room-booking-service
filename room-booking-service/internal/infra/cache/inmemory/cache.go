package inmemory

import (
	"sync"
	"time"
)

type element[K comparable, V any] struct {
	key        K
	value      V
	expiration time.Time
	next       *element[K, V]
	prev       *element[K, V]
}

type Option[K comparable, V any] func(*Cache[K, V])

func WithCapacity[K comparable, V any](capacity int) Option[K, V] {
	return func(cache *Cache[K, V]) {
		if capacity > 0 {
			cache.capacity = capacity
		}
	}
}

func WithGCInterval[K comparable, V any](interval time.Duration) Option[K, V] {
	return func(cache *Cache[K, V]) {
		if interval >= 0 {
			cache.gcInterval = interval
		}
	}
}

func WithDefaultTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(cache *Cache[K, V]) {
		if ttl >= 0 {
			cache.defaultTTL = ttl
		}
	}
}

type Cache[K comparable, V any] struct {
	kv map[K]*element[K, V]

	head *element[K, V]
	tail *element[K, V]

	mu       sync.Mutex
	capacity int

	gcInterval time.Duration
	defaultTTL time.Duration

	closeGC   chan struct{}
	closeOnce sync.Once
	done      chan struct{}
}

func New[K comparable, V any](opts ...Option[K, V]) *Cache[K, V] {
	head := &element[K, V]{}
	tail := &element[K, V]{}

	head.next = tail
	tail.prev = head

	cache := &Cache[K, V]{
		kv:         make(map[K]*element[K, V]),
		head:       head,
		tail:       tail,
		capacity:   1000,
		gcInterval: 1 * time.Minute,
		defaultTTL: 5 * time.Minute,
		closeGC:    make(chan struct{}),
		done:       make(chan struct{}),
	}
	for _, option := range opts {
		option(cache)
	}

	if cache.gcInterval > 0 {
		cache.startGC()
	} else {
		close(cache.done)
	}

	return cache
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var res V

	element, ok := c.kv[key]
	if !ok {
		return res, false
	}

	if !element.expiration.IsZero() && !time.Now().Before(element.expiration) {
		c.delete(key)
		return res, false
	}

	c.promote(element)

	return element.value, true
}

func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	node, ok := c.kv[key]
	if !ok {
		if len(c.kv) >= c.capacity {
			c.delete(c.tail.prev.key)
		}
		node = &element[K, V]{key: key, value: value}
		c.kv[key] = node
	} else {
		node.value = value
	}
	node.expiration = expiration

	c.promote(node)
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.delete(key)
}

func (c *Cache[K, V]) Close() {
	c.closeOnce.Do(func() { close(c.closeGC) })
	<-c.done
}

func (c *Cache[K, V]) promote(element *element[K, V]) {
	c.detach(element)

	element.next = c.head.next
	element.prev = c.head

	c.head.next.prev = element
	c.head.next = element
}

func (c *Cache[K, V]) delete(key K) {
	element, ok := c.kv[key]
	if ok {
		delete(c.kv, key)
		c.detach(element)
	}
}

func (c *Cache[K, V]) detach(element *element[K, V]) {
	next := element.next
	prev := element.prev

	if next != nil && prev != nil {
		next.prev = prev
		prev.next = next
		element.next = nil
		element.prev = nil
	}
}

func (c *Cache[K, V]) startGC() {
	go func() {
		ticker := time.NewTicker(c.gcInterval)
		defer ticker.Stop()
		defer close(c.done)

		for {
			select {
			case <-ticker.C:
				c.clean()
			case <-c.closeGC:
				return
			}
		}
	}()
}

func (c *Cache[K, V]) clean() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, element := range c.kv {
		if !element.expiration.IsZero() && !time.Now().Before(element.expiration) {
			c.delete(key)
		}
	}
}
