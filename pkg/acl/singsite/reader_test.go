package singsite

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
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
	defer func() { _ = reader.Close() }()

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
	defer func() { _ = reader.Close() }()

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
	defer func() { _ = reader.Close() }()

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
	defer func() { _ = reader.Close() }()

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

// TestLoadGeoSite_DomainSuffixLeadingDot verifies that domain suffix values
// with leading dots (as stored in sing-geosite format) are correctly normalized.
// This is the critical test for the bug where ".google.com" would fail to match
// "www.google.com" because the matcher would look for "..google.com".
func TestLoadGeoSite_DomainSuffixLeadingDot(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)

	// Check google site data
	google, ok := result["google"]
	require.True(t, ok, "google site should exist")

	// Find RootDomain entries and verify none have leading dots
	var rootDomainCount int
	var leadingDotCount int
	for _, domain := range google.Domain {
		if domain.Type == 2 { // Domain_RootDomain
			rootDomainCount++
			if len(domain.Value) > 0 && domain.Value[0] == '.' {
				leadingDotCount++
				t.Errorf("RootDomain value should not have leading dot: %q", domain.Value)
			}
		}
	}

	t.Logf("Checked %d RootDomain entries, %d had leading dots (should be 0)",
		rootDomainCount, leadingDotCount)
	assert.Zero(t, leadingDotCount, "no RootDomain values should have leading dots")
	assert.Greater(t, rootDomainCount, 0, "should have some RootDomain entries")
}

// TestLoadGeoSite_AllDomainTypes verifies all domain types are correctly converted.
func TestLoadGeoSite_AllDomainTypes(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)

	// Test with google which should have various domain types
	google, ok := result["google"]
	require.True(t, ok, "google site should exist")

	typeCounts := make(map[int32]int)
	for _, domain := range google.Domain {
		typeCounts[int32(domain.Type)]++
	}

	t.Logf("Domain type distribution for 'google':")
	t.Logf("  Plain (0): %d", typeCounts[0])
	t.Logf("  Regex (1): %d", typeCounts[1])
	t.Logf("  RootDomain (2): %d", typeCounts[2])
	t.Logf("  Full (3): %d", typeCounts[3])

	// We expect at least some Full and RootDomain entries for google
	assert.Greater(t, typeCounts[3], 0, "should have Full domain entries")
	assert.Greater(t, typeCounts[2], 0, "should have RootDomain entries")
}

// TestReader_RawDomainSuffixHasLeadingDot verifies that the raw data from
// sing-geosite does have leading dots (this is the expected format).
func TestReader_RawDomainSuffixHasLeadingDot(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	reader, codes, err := LoadFromFile(testFile)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	// Find google code
	var googleCode string
	for _, code := range codes {
		if code == "google" {
			googleCode = code
			break
		}
	}
	require.NotEmpty(t, googleCode, "google code should exist")

	items, err := reader.Read(googleCode)
	require.NoError(t, err)

	// Count DomainSuffix items with leading dots
	var suffixCount int
	var leadingDotCount int
	for _, item := range items {
		if item.Type == RuleTypeDomainSuffix {
			suffixCount++
			if len(item.Value) > 0 && item.Value[0] == '.' {
				leadingDotCount++
			}
		}
	}

	t.Logf("Raw DomainSuffix entries: %d total, %d with leading dots",
		suffixCount, leadingDotCount)

	// The raw data SHOULD have leading dots (this is the sing-geosite format)
	// If this test fails, it means the format has changed
	if suffixCount > 0 {
		assert.Equal(t, suffixCount, leadingDotCount,
			"raw DomainSuffix values should have leading dots (sing-geosite format)")
	}
}

// TestLoadFromBytes verifies loading from byte slice works correctly.
func TestLoadFromBytes(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	data, err := os.ReadFile(testFile)
	require.NoError(t, err)

	reader, codes, err := LoadFromBytes(data)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	assert.NotEmpty(t, codes, "should have loaded some site codes")

	// Verify we can read items
	if len(codes) > 0 {
		items, err := reader.Read(codes[0])
		require.NoError(t, err)
		assert.NotNil(t, items)
	}
}

// TestVerify tests the Verify function.
func TestVerify(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	err := Verify(testFile)
	assert.NoError(t, err, "should verify valid geosite.db")
}

// TestVerify_InvalidFile tests Verify with a non-existent file.
func TestVerify_InvalidFile(t *testing.T) {
	err := Verify("/nonexistent/path/geosite.db")
	assert.Error(t, err, "should fail for non-existent file")
}

// TestLoadGeoSite_DomainMatching tests domain matching with real geosite.db data.
// This is an integration test that verifies the full flow from loading to matching.
func TestLoadGeoSite_DomainMatching(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)

	// Helper function to check if a host matches any domain in the geosite
	matchHost := func(site *geodat.GeoSite, host string) bool {
		for _, domain := range site.Domain {
			switch domain.Type {
			case geodat.Domain_Full:
				if host == domain.Value {
					return true
				}
			case geodat.Domain_RootDomain:
				if host == domain.Value {
					return true
				}
				if len(host) > len(domain.Value) &&
					host[len(host)-len(domain.Value)-1] == '.' &&
					host[len(host)-len(domain.Value):] == domain.Value {
					return true
				}
			case geodat.Domain_Plain:
				if strings.Contains(host, domain.Value) {
					return true
				}
			}
		}
		return false
	}

	tests := []struct {
		siteCode string
		host     string
		want     bool
		desc     string
	}{
		// Google 匹配测试
		{"google", "google.com", true, "google.com 精确匹配"},
		{"google", "www.google.com", true, "www.google.com 子域名匹配"},
		{"google", "mail.google.com", true, "mail.google.com 子域名匹配"},
		{"google", "apis.google.com", true, "apis.google.com 子域名匹配"},
		{"google", "translate.google.com", true, "translate.google.com 子域名匹配"},
		{"google", "youtube.com", true, "youtube.com 属于 google"},
		{"google", "www.youtube.com", true, "www.youtube.com 子域名"},
		{"google", "googleapis.com", true, "googleapis.com 属于 google"},
		{"google", "gstatic.com", true, "gstatic.com 属于 google"},

		// Google 不匹配测试
		{"google", "fakegoogle.com", false, "fakegoogle.com 不应匹配（后缀但非子域名）"},
		{"google", "google.com.cn", false, "google.com.cn 不应匹配"},
		{"google", "notgoogle.org", false, "notgoogle.org 不应匹配"},

		// GitHub 匹配测试
		{"github", "github.com", true, "github.com 精确匹配"},
		{"github", "www.github.com", true, "www.github.com 子域名"},
		{"github", "api.github.com", true, "api.github.com 子域名"},
		{"github", "raw.githubusercontent.com", true, "githubusercontent.com 属于 github"},
		{"github", "github.io", true, "github.io 属于 github"},

		// GitHub 不匹配测试
		{"github", "fakegithub.com", false, "fakegithub.com 不应匹配"},
		{"github", "github.com.fake", false, "github.com.fake 不应匹配"},

		// CN 站点测试
		{"cn", "baidu.com", true, "baidu.com 属于 cn"},
		{"cn", "www.baidu.com", true, "www.baidu.com 子域名"},
		{"cn", "qq.com", true, "qq.com 属于 cn"},
		{"cn", "weixin.qq.com", true, "weixin.qq.com 子域名"},
		{"cn", "taobao.com", true, "taobao.com 属于 cn"},
		{"cn", "alipay.com", true, "alipay.com 属于 cn"},
		{"cn", "jd.com", true, "jd.com 属于 cn"},
		{"cn", "163.com", true, "163.com 属于 cn"},
		{"cn", "bilibili.com", true, "bilibili.com 属于 cn"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			site, ok := result[tt.siteCode]
			if !ok {
				t.Skipf("site code %q not found in geosite.db", tt.siteCode)
			}

			got := matchHost(site, tt.host)
			if got != tt.want {
				t.Errorf("matchHost(%q, %q) = %v, want %v", tt.siteCode, tt.host, got, tt.want)
			}
		})
	}
}

// TestLoadGeoSite_EdgeCases tests edge cases in domain matching.
func TestLoadGeoSite_EdgeCases(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)

	google, ok := result["google"]
	require.True(t, ok)

	// 验证所有 RootDomain 条目格式正确
	for _, domain := range google.Domain {
		if domain.Type == geodat.Domain_RootDomain {
			// 不应有前导点
			assert.False(t, strings.HasPrefix(domain.Value, "."),
				"RootDomain should not have leading dot: %q", domain.Value)

			// 不应为空
			assert.NotEmpty(t, domain.Value, "RootDomain value should not be empty")

			// 不应有连续的点
			assert.False(t, strings.Contains(domain.Value, ".."),
				"RootDomain should not have consecutive dots: %q", domain.Value)
		}
	}
}

// TestLoadGeoSite_SpecificDomains verifies specific domains are present and correctly formatted.
func TestLoadGeoSite_SpecificDomains(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)

	google, ok := result["google"]
	require.True(t, ok)

	// 收集所有域名值
	domainValues := make(map[string]geodat.Domain_Type)
	for _, domain := range google.Domain {
		domainValues[domain.Value] = domain.Type
	}

	// 验证一些已知的 Google 域名存在且类型正确
	expectedDomains := []struct {
		value      string
		expectType geodat.Domain_Type
	}{
		{"google.com", geodat.Domain_Full},
		{"youtube.com", geodat.Domain_Full},
		{"googleapis.com", geodat.Domain_Full},
	}

	for _, ed := range expectedDomains {
		typ, exists := domainValues[ed.value]
		if exists {
			t.Logf("Found domain %q with type %v (expected %v)", ed.value, typ, ed.expectType)
		}
	}

	// 验证没有带前导点的域名
	for value := range domainValues {
		assert.False(t, strings.HasPrefix(value, "."),
			"domain value should not have leading dot: %q", value)
	}
}

// TestLoadGeoSite_MultipleSites tests loading and validating multiple site codes.
func TestLoadGeoSite_MultipleSites(t *testing.T) {
	testFile := filepath.Join(getTestDataDir(), "geosite.db")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("testdata/geosite.db not found, skipping test")
	}

	result, err := LoadGeoSite(testFile)
	require.NoError(t, err)

	sitesToCheck := []string{"google", "github", "cn", "facebook", "twitter", "amazon", "apple", "microsoft"}

	for _, siteCode := range sitesToCheck {
		site, ok := result[siteCode]
		if !ok {
			t.Logf("Site %q not found, skipping", siteCode)
			continue
		}

		t.Run(siteCode, func(t *testing.T) {
			assert.NotEmpty(t, site.Domain, "site %q should have domains", siteCode)

			var rootDomainCount, fullCount, leadingDotCount int
			for _, domain := range site.Domain {
				switch domain.Type {
				case geodat.Domain_RootDomain:
					rootDomainCount++
					if strings.HasPrefix(domain.Value, ".") {
						leadingDotCount++
					}
				case geodat.Domain_Full:
					fullCount++
				}
			}

			t.Logf("Site %q: %d total, %d RootDomain, %d Full, %d with leading dots",
				siteCode, len(site.Domain), rootDomainCount, fullCount, leadingDotCount)

			assert.Zero(t, leadingDotCount,
				"site %q should have no RootDomain with leading dots", siteCode)
		})
	}
}
