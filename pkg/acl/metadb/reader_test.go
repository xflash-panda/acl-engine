package metadb

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "testdata")
}

func TestOpenDatabase(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	db, err := OpenDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Verify database type
	dbType := db.Type()
	t.Logf("Database type: %s", dbType)
	assert.True(t, dbType == TypeMetaV0 || dbType == TypeSing || dbType == TypeMaxmind,
		"should be a valid database type")

	// Verify reader is not nil
	assert.NotNil(t, db.Reader(), "reader should not be nil")
}

func TestDatabaseLookupCode(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	db, err := OpenDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Test with some well-known IPs
	testCases := []struct {
		ip          string
		description string
	}{
		{"8.8.8.8", "Google DNS"},
		{"1.1.1.1", "Cloudflare DNS"},
		{"114.114.114.114", "China DNS"},
		{"223.5.5.5", "Alibaba DNS"},
	}

	for _, tc := range testCases {
		ip := net.ParseIP(tc.ip)
		require.NotNil(t, ip, "should parse IP: %s", tc.ip)

		codes := db.LookupCode(ip)
		t.Logf("%s (%s): %v", tc.description, tc.ip, codes)
		// We don't assert specific codes as they may vary by database version
	}
}

func TestDatabaseLookupIPv6(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	db, err := OpenDatabase(testFile)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Test with IPv6 addresses
	testCases := []struct {
		ip          string
		description string
	}{
		{"2001:4860:4860::8888", "Google DNS IPv6"},
		{"2606:4700:4700::1111", "Cloudflare DNS IPv6"},
	}

	for _, tc := range testCases {
		ip := net.ParseIP(tc.ip)
		require.NotNil(t, ip, "should parse IP: %s", tc.ip)

		codes := db.LookupCode(ip)
		t.Logf("%s (%s): %v", tc.description, tc.ip, codes)
	}
}

func TestOpenDatabaseNonExistent(t *testing.T) {
	_, err := OpenDatabase("/nonexistent/path/geoip.metadb")
	assert.Error(t, err, "should return error for non-existent file")
}

func TestLoadGeoIP(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	result, err := LoadGeoIP(testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, result, "should have loaded geoip data")

	t.Logf("LoadGeoIP returned %d country codes", len(result))

	// Check that we have some common country codes
	commonCodes := []string{"cn", "us", "jp", "hk"}
	for _, code := range commonCodes {
		if geoIP, ok := result[code]; ok {
			t.Logf("Country %s: %d CIDRs", code, len(geoIP.Cidr))
		}
	}

	// Verify structure
	for code, geoIP := range result {
		assert.NotEmpty(t, geoIP.CountryCode, "country code should not be empty")
		assert.Equal(t, code, lowercaseString(geoIP.CountryCode), "map key should match lowercase country code")
		if len(geoIP.Cidr) > 0 {
			// Check first CIDR is valid
			cidr := geoIP.Cidr[0]
			assert.NotNil(t, cidr.Ip, "CIDR IP should not be nil")
			assert.True(t, cidr.Prefix <= 128, "CIDR prefix should be valid")
		}
		break // Only check one entry
	}
}

func TestLookupIP(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	// Test with a known Chinese IP
	codes, err := LookupIP(testFile, net.ParseIP("114.114.114.114"))
	require.NoError(t, err)
	t.Logf("114.114.114.114: %v", codes)

	// Codes should be lowercase
	for _, code := range codes {
		assert.Equal(t, code, lowercaseString(code), "code should be lowercase")
	}
}

func TestVerify(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geoip.metadb not found, skipping test")
	}

	err := Verify(testFile)
	assert.NoError(t, err, "should verify valid database")

	err = Verify("/nonexistent/path/geoip.metadb")
	assert.Error(t, err, "should return error for non-existent file")
}

func lowercaseString(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
