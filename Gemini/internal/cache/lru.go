package cache

import "sync"

// LRUCache is a thread-safe LRU cache for string key-value pairs.
// Used by the dispatcher to cache search/scrape results and reduce API quota usage.
type LRUCache struct {
	mu       sync.RWMutex
	items    map[string]string
	order    []string // oldest first
	maxSize  int
}

// NewLRUCache creates an LRU cache with the given maximum entry count.
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		items:   make(map[string]string, maxSize),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get returns the cached value for the given key, or empty string if not found.
func (c *LRUCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

// Put stores a value in the cache, evicting the oldest entry if at capacity.
func (c *LRUCache) Put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key already exists, update value and move to end
	if _, exists := c.items[key]; exists {
		c.items[key] = value
		// Move to end of order
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
		c.order = append(c.order, key)
		return
	}

	// Evict oldest if at capacity
	if len(c.order) >= c.maxSize {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.items, oldest)
	}

	c.items[key] = value
	c.order = append(c.order, key)
}

// Len returns the current number of entries in the cache.
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]string, c.maxSize)
	c.order = c.order[:0]
}
