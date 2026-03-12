package metadb

import (
	"fmt"
	"net"

	lru "github.com/hashicorp/golang-lru/v2"
)

// DefaultCacheSize is the default size for the LRU cache.
const DefaultCacheSize = 1024

// CachedDatabase wraps a Database with an LRU cache for IP lookups.
// This provides significant performance improvements for hot IP addresses
// that are looked up repeatedly.
// The underlying lru.Cache is already thread-safe, so no additional
// synchronization is needed.
type CachedDatabase struct {
	db    *Database
	cache *lru.Cache[string, []string]
}

// NewCachedDatabase creates a new cached database with the default cache size.
func NewCachedDatabase(db *Database) (*CachedDatabase, error) {
	return NewCachedDatabaseWithSize(db, DefaultCacheSize)
}

// NewCachedDatabaseWithSize creates a new cached database with a custom cache size.
func NewCachedDatabaseWithSize(db *Database, cacheSize int) (*CachedDatabase, error) {
	cache, err := lru.New[string, []string](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("create LRU cache: %w", err)
	}

	return &CachedDatabase{
		db:    db,
		cache: cache,
	}, nil
}

// OpenCachedDatabase opens a GeoIP database file with caching enabled.
func OpenCachedDatabase(filename string) (*CachedDatabase, error) {
	db, err := OpenDatabase(filename)
	if err != nil {
		return nil, err
	}

	return NewCachedDatabase(db)
}

// OpenCachedDatabaseWithSize opens a GeoIP database file with a custom cache size.
func OpenCachedDatabaseWithSize(filename string, cacheSize int) (*CachedDatabase, error) {
	db, err := OpenDatabase(filename)
	if err != nil {
		return nil, err
	}

	return NewCachedDatabaseWithSize(db, cacheSize)
}

// LookupCode looks up country codes for an IP address with caching.
// Returns nil if not found.
func (c *CachedDatabase) LookupCode(ip net.IP) []string {
	key := ip.String()

	if codes, ok := c.cache.Get(key); ok {
		return codes
	}

	// Cache miss, lookup from database
	codes := c.db.LookupCode(ip)
	c.cache.Add(key, codes)
	return codes
}

// Type returns the database type.
func (c *CachedDatabase) Type() DatabaseType {
	return c.db.Type()
}

// Reader returns the underlying MaxMind DB reader.
func (c *CachedDatabase) Reader() interface{} {
	return c.db.Reader()
}

// ClearCache clears the LRU cache.
func (c *CachedDatabase) ClearCache() {
	c.cache.Purge()
}

// CacheLen returns the number of items in the cache.
func (c *CachedDatabase) CacheLen() int {
	return c.cache.Len()
}

// Close closes the database.
func (c *CachedDatabase) Close() error {
	c.cache.Purge()
	return c.db.Close()
}
