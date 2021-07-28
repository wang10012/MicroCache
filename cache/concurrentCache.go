package cache

import (
	"MicroCache/cache/LRU"
	"sync"
)

// concurrentCache 控制并发
type concurrentCache struct {
	mutex      sync.Mutex
	lru        *LRU.Cache
	cacheBytes int64
}

func (c *concurrentCache) add(key string, value OnlyReadBytes) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// Lazy Initialization 第一次使用的时候再初始化，提高性能
	if c.lru == nil {
		c.lru = LRU.NewCache(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *concurrentCache) get(key string) (value OnlyReadBytes, ok bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(OnlyReadBytes), ok
	}

	return
}
