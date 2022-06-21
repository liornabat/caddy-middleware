package hocoosmiddleware

import (
	"sync"
	"time"
)

type cacheItem struct {
	value string
	ttl   time.Time
}
type localCache struct {
	sync.RWMutex
	defaultTTL time.Duration
	cache      map[string]*cacheItem
}

func newLocalCache(defaultTTL time.Duration) *localCache {
	lc := &localCache{
		defaultTTL: defaultTTL,
		cache:      map[string]*cacheItem{},
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			lc.Clear()
		}
	}()
	return lc
}

func (c *localCache) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()
	item, ok := c.cache[key]
	if !ok {
		return "", false
	}
	if item.ttl.Before(time.Now()) {
		delete(c.cache, key)
		return "", false
	}
	return item.value, true
}
func (c *localCache) Set(key string, value string) {
	c.Lock()
	defer c.Unlock()
	c.cache[key] = &cacheItem{
		value: value,
		ttl:   time.Now().Add(c.defaultTTL),
	}
}
func (c *localCache) Delete(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.cache, key)
}
func (c *localCache) Clear() {
	c.Lock()
	defer c.Unlock()
	for k := range c.cache {
		if c.cache[k].ttl.Before(time.Now()) {
			delete(c.cache, k)
		}
	}
}
