package promovalidator

import (
	"container/list"
	"sync"
)

// LRU is a tiny, goroutine-safe LRU cache for string->bool.
type LRU struct {
	mu  sync.Mutex
	max int
	ll  *list.List
	m   map[string]*list.Element // key -> list element
}

type kv struct {
	k string
	v bool
}

func NewLRU(max int) *LRU {
	if max < 1 {
		max = 1
	}
	return &LRU{
		max: max,
		ll:  list.New(),
		m:   make(map[string]*list.Element, max),
	}
}

// Get returns (value, ok). Moves the item to the front (most recently used) on hit.
func (c *LRU) Get(k string) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[k]; ok {
		c.ll.MoveToFront(e)
		return e.Value.(kv).v, true
	}
	return false, false
}

// Add inserts/updates (k,v). If over capacity, evicts the least-recently-used.
func (c *LRU) Add(k string, v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.m[k]; ok {
		e.Value = kv{k, v}
		c.ll.MoveToFront(e)
		return
	}

	e := c.ll.PushFront(kv{k, v})
	c.m[k] = e

	if c.ll.Len() > c.max {
		last := c.ll.Back()
		if last != nil {
			ev := last.Value.(kv)
			delete(c.m, ev.k)
			c.ll.Remove(last)
		}
	}
}

// Len returns current number of entries.
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}
