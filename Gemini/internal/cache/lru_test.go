package cache

import (
	"testing"
)

func TestNewLRUCache(t *testing.T) {
	c := NewLRUCache(10)
	if c == nil {
		t.Fatal("NewLRUCache returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("expected 0 entries, got %d", c.Len())
	}
}

func TestPutAndGet(t *testing.T) {
	c := NewLRUCache(10)
	c.Put("key1", "value1")

	v, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if v != "value1" {
		t.Errorf("expected value1, got %s", v)
	}
}

func TestGetNonExistent(t *testing.T) {
	c := NewLRUCache(10)
	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to return false")
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewLRUCache(3)
	c.Put("a", "1")
	c.Put("b", "2")
	c.Put("c", "3")
	// Cache is full. Adding "d" should evict "a" (oldest)
	c.Put("d", "4")

	if _, ok := c.Get("a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	if v, ok := c.Get("d"); !ok || v != "4" {
		t.Errorf("expected d=4, got %s, ok=%v", v, ok)
	}
}

func TestUpdateExistingKey(t *testing.T) {
	c := NewLRUCache(3)
	c.Put("a", "1")
	c.Put("a", "updated")

	v, ok := c.Get("a")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if v != "updated" {
		t.Errorf("expected 'updated', got %s", v)
	}
	if c.Len() != 1 {
		t.Errorf("expected 1 entry, got %d", c.Len())
	}
}

func TestClear(t *testing.T) {
	c := NewLRUCache(10)
	c.Put("a", "1")
	c.Put("b", "2")
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected 0 entries after clear, got %d", c.Len())
	}
}

func TestMaxCapacity(t *testing.T) {
	c := NewLRUCache(5)
	for i := 0; i < 10; i++ {
		c.Put("key", "value")
	}
	if c.Len() != 1 {
		t.Errorf("expected 1 entry (same key), got %d", c.Len())
	}
}
