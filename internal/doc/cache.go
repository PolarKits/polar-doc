package doc

import (
	"container/list"
	"context"
	"sync"
)

// LRUCache is a thread-safe LRU cache with a byte-size limit.
// When the total size of cached entries exceeds maxBytes, the least
// recently used entries are evicted until the cache fits within the limit.
type LRUCache struct {
	maxBytes     int64
	currentBytes int64
	mu           sync.Mutex
	items        map[string]*list.Element // key -> *list.Element of cachedItem
	order        *list.List               // front=most recent, back=least recent
}

// cachedItem holds a cached value and its size in bytes.
type cachedItem struct {
	key   string
	value []byte
	size  int64
}

// NewLRUCache creates a new LRUCache with the specified maximum byte size.
func NewLRUCache(maxBytes int64) *LRUCache {
	return &LRUCache{
		maxBytes: maxBytes,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves a cached value by key. Returns the value and true if found,
// or nil and false if not found or on context cancellation.
func (c *LRUCache) Get(ctx context.Context, key string) ([]byte, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	return elem.Value.(*cachedItem).value, true
}

// Put stores a value in the cache. If the key already exists, its value
// is updated and it is moved to the most recently used position.
// If the new entry would exceed maxBytes, existing entries are evicted
// (from least recently used) until there is room.
func (c *LRUCache) Put(ctx context.Context, key string, value []byte) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	itemSize := int64(len(value))

	// If key exists, update and move to front
	if elem, ok := c.items[key]; ok {
		oldItem := elem.Value.(*cachedItem)
		c.currentBytes -= oldItem.size
		c.order.Remove(elem)
	}

	// Evict LRU entries until we have room
	for c.currentBytes+itemSize > c.maxBytes && c.order.Back() != nil {
		back := c.order.Back()
		oldItem := back.Value.(*cachedItem)
		c.currentBytes -= oldItem.size
		delete(c.items, oldItem.key)
		c.order.Remove(back)
	}

	// If the new item itself exceeds maxBytes, skip caching it
	if itemSize > c.maxBytes {
		return
	}

	// Add new item at front
	item := &cachedItem{key: key, value: value, size: itemSize}
	elem := c.order.PushFront(item)
	c.items[key] = elem
	c.currentBytes += itemSize
}

// DefaultPageCacheMaxBytes is the default maximum size for page content cache.
const DefaultPageCacheMaxBytes = 100 * 1024 * 1024 // 100 MiB
