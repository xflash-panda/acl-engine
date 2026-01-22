package singsite

import (
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

func TestLoadFromFile(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	reader, codes, err := LoadFromFile(testFile)
	require.NoError(t, err)
	defer reader.Close()

	// Verify we got some codes
	assert.NotEmpty(t, codes, "should have loaded some site codes")
	t.Logf("Loaded %d site codes", len(codes))

	// Check some common codes exist
	codeSet := make(map[string]bool)
	for _, code := range codes {
		codeSet[code] = true
	}

	// These are common geosite codes that should exist
	commonCodes := []string{"cn", "google", "github"}
	for _, code := range commonCodes {
		if codeSet[code] {
			t.Logf("Found common code: %s", code)
		}
	}
}

func TestReaderRead(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	reader, codes, err := LoadFromFile(testFile)
	require.NoError(t, err)
	defer reader.Close()

	// Find a code to test
	var testCode string
	for _, code := range codes {
		if code == "google" || code == "cn" || code == "github" {
			testCode = code
			break
		}
	}

	if testCode == "" && len(codes) > 0 {
		testCode = codes[0]
	}

	require.NotEmpty(t, testCode, "should have at least one code to test")

	// Read items for the test code
	items, err := reader.Read(testCode)
	require.NoError(t, err)
	assert.NotEmpty(t, items, "should have items for code %s", testCode)

	t.Logf("Code %q has %d items", testCode, len(items))

	// Verify item structure
	for i, item := range items {
		if i >= 5 {
			break // Only check first 5 items
		}
		assert.True(t, item.Type <= RuleTypeDomainRegex, "item type should be valid")
		assert.NotEmpty(t, item.Value, "item value should not be empty")
		t.Logf("  Item %d: type=%d, value=%s", i, item.Type, item.Value)
	}
}

func TestReaderReadMultipleCodes(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	reader, codes, err := LoadFromFile(testFile)
	require.NoError(t, err)
	defer reader.Close()

	// Read multiple codes to verify seek works correctly
	readCount := 0
	for _, code := range codes {
		if readCount >= 10 {
			break
		}
		items, err := reader.Read(code)
		require.NoError(t, err, "should read code %s", code)
		assert.NotNil(t, items, "items should not be nil for code %s", code)
		readCount++
	}

	t.Logf("Successfully read %d codes", readCount)
}

func TestReaderReadNonExistentCode(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	reader, _, err := LoadFromFile(testFile)
	require.NoError(t, err)
	defer reader.Close()

	// Try to read a non-existent code
	_, err = reader.Read("this-code-should-not-exist-12345")
	assert.Error(t, err, "should return error for non-existent code")
}

func TestLoadFromFileNonExistent(t *testing.T) {
	_, _, err := LoadFromFile("/nonexistent/path/geosite.db")
	assert.Error(t, err, "should return error for non-existent file")
}

func TestLoadGeoSite(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)
	assert.NotEmpty(t, result, "should have loaded geosite data")

	t.Logf("LoadGeoSite returned %d entries", len(result))

	// Check that codes are lowercase
	for code := range result {
		assert.Equal(t, code, lowercaseString(code), "code should be lowercase: %s", code)
	}
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
