package metadb

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCachedDatabase(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	db, err := OpenDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	cached, err := NewCachedDatabase(db)
	require.NoError(t, err)
	assert.NotNil(t, cached)
	defer func() { _ = cached.Close() }()

	// Verify cache is initialized
	assert.Equal(t, 0, cached.CacheLen(), "cache should be empty initially")
}

func TestNewCachedDatabaseWithSize(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	db, err := OpenDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	cached, err := NewCachedDatabaseWithSize(db, 100)
	require.NoError(t, err)
	assert.NotNil(t, cached)
	defer func() { _ = cached.Close() }()
}

func TestOpenCachedDatabase(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	cached, err := OpenCachedDatabase(testFile)
	require.NoError(t, err)
	assert.NotNil(t, cached)
	defer func() { _ = cached.Close() }()

	// Verify database type
	dbType := cached.Type()
	assert.True(t, dbType == TypeMetaV0 || dbType == TypeSing || dbType == TypeMaxmind,
		"should be a valid database type")
}

func TestCachedDatabaseLookupCode(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	cached, err := OpenCachedDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = cached.Close() }()

	ip := net.ParseIP("8.8.8.8")
	require.NotNil(t, ip)

	// First lookup (cache miss)
	codes1 := cached.LookupCode(ip)
	assert.Equal(t, 1, cached.CacheLen(), "cache should have 1 entry after lookup")

	// Second lookup (cache hit)
	codes2 := cached.LookupCode(ip)
	assert.Equal(t, codes1, codes2, "cached result should match original")
	assert.Equal(t, 1, cached.CacheLen(), "cache size should remain 1")

	// Lookup different IP
	ip2 := net.ParseIP("1.1.1.1")
	cached.LookupCode(ip2)
	assert.Equal(t, 2, cached.CacheLen(), "cache should have 2 entries")
}

func TestCachedDatabaseClearCache(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	cached, err := OpenCachedDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = cached.Close() }()

	// Add some entries to cache
	cached.LookupCode(net.ParseIP("8.8.8.8"))
	cached.LookupCode(net.ParseIP("1.1.1.1"))
	assert.Equal(t, 2, cached.CacheLen(), "cache should have 2 entries")

	// Clear cache
	cached.ClearCache()
	assert.Equal(t, 0, cached.CacheLen(), "cache should be empty after clear")
}

func TestCachedDatabaseConcurrency(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	cached, err := OpenCachedDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = cached.Close() }()

	// Run concurrent lookups
	ips := []string{"8.8.8.8", "1.1.1.1", "114.114.114.114", "223.5.5.5"}
	done := make(chan bool, len(ips)*10)

	for i := 0; i < 10; i++ {
		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			go func(ip net.IP) {
				cached.LookupCode(ip)
				done <- true
			}(ip)
		}
	}

	// Wait for all goroutines
	for i := 0; i < len(ips)*10; i++ {
		<-done
	}

	// Verify cache has entries
	assert.True(t, cached.CacheLen() > 0, "cache should have entries after concurrent lookups")
}

func TestExtractCodesFromMetaV0(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "single string",
			input:    "US",
			expected: []string{"US"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "slice of strings",
			input:    []any{"US", "CA"},
			expected: []string{"US", "CA"},
		},
		{
			name:     "slice with empty strings",
			input:    []any{"US", "", "CA"},
			expected: []string{"US", "CA"},
		},
		{
			name:     "slice with non-strings",
			input:    []any{"US", 123, "CA"},
			expected: []string{"US", "CA"},
		},
		{
			name:     "empty slice",
			input:    []any{},
			expected: nil,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "unsupported type",
			input:    123,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCodesFromMetaV0(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmarks
func BenchmarkCachedDatabase_LookupCode_CacheHit(b *testing.B) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("testdata/geoip.metadb not found, skipping benchmark")
	}

	cached, err := OpenCachedDatabase(testFile)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = cached.Close() }()

	ip := net.ParseIP("8.8.8.8")
	// Warm up cache
	cached.LookupCode(ip)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached.LookupCode(ip)
	}
}

func BenchmarkCachedDatabase_LookupCode_CacheMiss(b *testing.B) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("testdata/geoip.metadb not found, skipping benchmark")
	}

	cached, err := OpenCachedDatabaseWithSize(testFile, 1)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = cached.Close() }()

	// Use different IPs to force cache misses
	ips := []net.IP{
		net.ParseIP("8.8.8.8"),
		net.ParseIP("1.1.1.1"),
		net.ParseIP("114.114.114.114"),
		net.ParseIP("223.5.5.5"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached.LookupCode(ips[i%len(ips)])
	}
}

func BenchmarkDatabase_LookupCode_NoCaching(b *testing.B) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("testdata/geoip.metadb not found, skipping benchmark")
	}

	db, err := OpenDatabase(testFile)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ip := net.ParseIP("8.8.8.8")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LookupCode(ip)
	}
}

func BenchmarkCachedDatabase_Mixed_Workload(b *testing.B) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("testdata/geoip.metadb not found, skipping benchmark")
	}

	cached, err := OpenCachedDatabase(testFile)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = cached.Close() }()

	// Hot IPs (80% of requests)
	hotIPs := []net.IP{
		net.ParseIP("8.8.8.8"),
		net.ParseIP("1.1.1.1"),
	}

	// Cold IPs (20% of requests)
	coldIPs := []net.IP{
		net.ParseIP("114.114.114.114"),
		net.ParseIP("223.5.5.5"),
		net.ParseIP("208.67.222.222"),
		net.ParseIP("9.9.9.9"),
		net.ParseIP("180.76.76.76"),
		net.ParseIP("119.29.29.29"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%10 < 8 {
			// 80% hot
			cached.LookupCode(hotIPs[i%len(hotIPs)])
		} else {
			// 20% cold
			cached.LookupCode(coldIPs[i%len(coldIPs)])
		}
	}
}
