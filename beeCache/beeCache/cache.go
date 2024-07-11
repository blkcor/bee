package beeCache

import (
	"github.com/blkcor/beeCache/beeCache/lru"
	"sync"
)

type cache struct {
	mx         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

func (c *cache) add(key string, value ByteView) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (byteView ByteView, ok bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
