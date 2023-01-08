package character

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache is an instance of a key-value store with contents specific to
// each instance and are not shared between instances. By default its
// entries do not expire without a TTL.
type Cache struct {
	cacheInstance *gocache.Cache
}

func NewCache() *Cache {
	return &Cache{cacheInstance: gocache.New(-1, 10*time.Second)}
}

// Put sets a key/value pair in the cache with an optional duration. Passing 0 for
// ttl will cause the default expiration to be used and -1 will not set a ttl.
func (c *Cache) Put(key string, value interface{}, ttl time.Duration) {
	c.cacheInstance.Set(key, value, ttl)
}

// Get fetches a value from the cache, returning the value as well as whether
// or not the value was found (semantics similar to map).
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.cacheInstance.Get(key)
}
