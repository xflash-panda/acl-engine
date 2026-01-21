package acl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

func TestNilGeoLoader(t *testing.T) {
	loader := &NilGeoLoader{}

	geoIP, err := loader.LoadGeoIP()
	assert.NoError(t, err)
	assert.Nil(t, geoIP)

	geoSite, err := loader.LoadGeoSite()
	assert.NoError(t, err)
	assert.Nil(t, geoSite)
}

func TestFileGeoLoader_EmptyPath(t *testing.T) {
	loader := &FileGeoLoader{}

	geoIP, err := loader.LoadGeoIP()
	assert.NoError(t, err)
	assert.Nil(t, geoIP)

	geoSite, err := loader.LoadGeoSite()
	assert.NoError(t, err)
	assert.Nil(t, geoSite)
}

func TestFileGeoLoader_NonExistentFile(t *testing.T) {
	loader := &FileGeoLoader{
		GeoIPPath:   "/nonexistent/path/geoip.dat",
		GeoSitePath: "/nonexistent/path/geosite.dat",
	}

	_, err := loader.LoadGeoIP()
	assert.Error(t, err)

	_, err = loader.LoadGeoSite()
	assert.Error(t, err)
}

func TestFileGeoLoader_FormatDetection(t *testing.T) {
	tests := []struct {
		name             string
		geoIPPath        string
		geoIPFormat      GeoIPFormat
		expectedIPFormat GeoIPFormat
		geoSitePath      string
		geoSiteFormat    GeoSiteFormat
		expectedSiteFormat GeoSiteFormat
	}{
		{
			name:             "auto detect dat",
			geoIPPath:        "geoip.dat",
			expectedIPFormat: GeoIPFormatDAT,
			geoSitePath:      "geosite.dat",
			expectedSiteFormat: GeoSiteFormatDAT,
		},
		{
			name:             "auto detect mmdb",
			geoIPPath:        "country.mmdb",
			expectedIPFormat: GeoIPFormatMMDB,
		},
		{
			name:             "auto detect metadb",
			geoIPPath:        "geoip.metadb",
			expectedIPFormat: GeoIPFormatMetaDB,
		},
		{
			name:             "auto detect sing",
			geoSitePath:      "geosite.db",
			expectedSiteFormat: GeoSiteFormatSing,
		},
		{
			name:             "explicit format overrides auto",
			geoIPPath:        "geoip.dat",
			geoIPFormat:      GeoIPFormatMMDB,
			expectedIPFormat: GeoIPFormatMMDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &FileGeoLoader{
				GeoIPPath:     tt.geoIPPath,
				GeoIPFormat:   tt.geoIPFormat,
				GeoSitePath:   tt.geoSitePath,
				GeoSiteFormat: tt.geoSiteFormat,
			}

			if tt.geoIPPath != "" {
				got := loader.getGeoIPFormat()
				assert.Equal(t, tt.expectedIPFormat, got)
			}
			if tt.geoSitePath != "" {
				got := loader.getGeoSiteFormat()
				assert.Equal(t, tt.expectedSiteFormat, got)
			}
		})
	}
}

func TestFileGeoLoader_FormatNotDetectable(t *testing.T) {
	loader := &FileGeoLoader{
		GeoIPPath:   "geoip.unknown",
		GeoSitePath: "geosite.unknown",
	}

	_, err := loader.LoadGeoIP()
	assert.ErrorIs(t, err, ErrGeoIPFormatNotSet)

	_, err = loader.LoadGeoSite()
	assert.ErrorIs(t, err, ErrGeoSiteFormatNotSet)
}

func TestAutoGeoLoader_FormatNotSet(t *testing.T) {
	loader := &AutoGeoLoader{
		DataDir: "/tmp/test",
	}

	_, err := loader.LoadGeoIP()
	assert.ErrorIs(t, err, ErrGeoIPFormatNotSet)

	_, err = loader.LoadGeoSite()
	assert.ErrorIs(t, err, ErrGeoSiteFormatNotSet)
}

func TestAutoGeoLoader_FormatDetection(t *testing.T) {
	tests := []struct {
		name             string
		geoIPPath        string
		geoIPFormat      GeoIPFormat
		expectedIPFormat GeoIPFormat
		geoSitePath      string
		geoSiteFormat    GeoSiteFormat
		expectedSiteFormat GeoSiteFormat
	}{
		{
			name:             "explicit format",
			geoIPFormat:      GeoIPFormatMMDB,
			expectedIPFormat: GeoIPFormatMMDB,
			geoSiteFormat:    GeoSiteFormatSing,
			expectedSiteFormat: GeoSiteFormatSing,
		},
		{
			name:             "detect from path",
			geoIPPath:        "/path/to/geoip.mmdb",
			expectedIPFormat: GeoIPFormatMMDB,
			geoSitePath:      "/path/to/geosite.db",
			expectedSiteFormat: GeoSiteFormatSing,
		},
		{
			name:             "format takes precedence over path",
			geoIPPath:        "/path/to/geoip.dat",
			geoIPFormat:      GeoIPFormatMMDB,
			expectedIPFormat: GeoIPFormatMMDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &AutoGeoLoader{
				GeoIPPath:     tt.geoIPPath,
				GeoIPFormat:   tt.geoIPFormat,
				GeoSitePath:   tt.geoSitePath,
				GeoSiteFormat: tt.geoSiteFormat,
			}

			if tt.expectedIPFormat != "" {
				got := loader.getGeoIPFormat()
				assert.Equal(t, tt.expectedIPFormat, got)
			}
			if tt.expectedSiteFormat != "" {
				got := loader.getGeoSiteFormat()
				assert.Equal(t, tt.expectedSiteFormat, got)
			}
		})
	}
}

func TestAutoGeoLoader_PathGeneration(t *testing.T) {
	tests := []struct {
		name             string
		dataDir          string
		geoIPPath        string
		geoIPFormat      GeoIPFormat
		expectedIPPath   string
		geoSitePath      string
		geoSiteFormat    GeoSiteFormat
		expectedSitePath string
	}{
		{
			name:             "explicit path",
			geoIPPath:        "/custom/path/geoip.mmdb",
			geoIPFormat:      GeoIPFormatMMDB,
			expectedIPPath:   "/custom/path/geoip.mmdb",
			geoSitePath:      "/custom/path/geosite.db",
			geoSiteFormat:    GeoSiteFormatSing,
			expectedSitePath: "/custom/path/geosite.db",
		},
		{
			name:             "from datadir dat",
			dataDir:          "/data",
			geoIPFormat:      GeoIPFormatDAT,
			expectedIPPath:   "/data/geoip.dat",
			geoSiteFormat:    GeoSiteFormatDAT,
			expectedSitePath: "/data/geosite.dat",
		},
		{
			name:             "from datadir mmdb",
			dataDir:          "/data",
			geoIPFormat:      GeoIPFormatMMDB,
			expectedIPPath:   "/data/geoip.mmdb",
		},
		{
			name:             "from datadir metadb",
			dataDir:          "/data",
			geoIPFormat:      GeoIPFormatMetaDB,
			expectedIPPath:   "/data/geoip.metadb",
		},
		{
			name:             "from datadir sing",
			dataDir:          "/data",
			geoSiteFormat:    GeoSiteFormatSing,
			expectedSitePath: "/data/geosite.db",
		},
		{
			name:             "no datadir",
			geoIPFormat:      GeoIPFormatDAT,
			expectedIPPath:   "geoip.dat",
			geoSiteFormat:    GeoSiteFormatDAT,
			expectedSitePath: "geosite.dat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &AutoGeoLoader{
				DataDir:       tt.dataDir,
				GeoIPPath:     tt.geoIPPath,
				GeoIPFormat:   tt.geoIPFormat,
				GeoSitePath:   tt.geoSitePath,
				GeoSiteFormat: tt.geoSiteFormat,
			}

			if tt.expectedIPPath != "" {
				got := loader.getGeoIPPath()
				assert.Equal(t, tt.expectedIPPath, got)
			}
			if tt.expectedSitePath != "" {
				got := loader.getGeoSitePath()
				assert.Equal(t, tt.expectedSitePath, got)
			}
		})
	}
}

func TestAutoGeoLoader_ShouldDownload(t *testing.T) {
	tmpDir := t.TempDir()

	loader := &AutoGeoLoader{
		DataDir: tmpDir,
	}

	// Non-existent file should need download
	assert.True(t, loader.shouldDownload(filepath.Join(tmpDir, "nonexistent.dat")))

	// Create an empty file
	emptyFile := filepath.Join(tmpDir, "empty.dat")
	err := os.WriteFile(emptyFile, []byte{}, 0600)
	require.NoError(t, err)
	assert.True(t, loader.shouldDownload(emptyFile))

	// Create a non-empty file
	validFile := filepath.Join(tmpDir, "valid.dat")
	err = os.WriteFile(validFile, []byte("content"), 0600)
	require.NoError(t, err)
	assert.False(t, loader.shouldDownload(validFile))
}

func TestAutoGeoLoader_DownloadNoURL(t *testing.T) {
	tmpDir := t.TempDir()

	loader := &AutoGeoLoader{
		DataDir:       tmpDir,
		GeoIPFormat:   GeoIPFormatDAT,
		GeoSiteFormat: GeoSiteFormatDAT,
		// No URLs set
	}

	// Should fail because no URL is configured
	_, err := loader.LoadGeoIP()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download URL not configured")

	_, err = loader.LoadGeoSite()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download URL not configured")
}

func TestLoadGeoIPFunctions(t *testing.T) {
	// Test that loadGeoIP returns appropriate errors for non-existent files
	_, err := loadGeoIP("/nonexistent/file.dat", GeoIPFormatDAT)
	assert.Error(t, err)

	_, err = loadGeoIP("/nonexistent/file.mmdb", GeoIPFormatMMDB)
	assert.Error(t, err)

	_, err = loadGeoIP("/nonexistent/file.metadb", GeoIPFormatMetaDB)
	assert.Error(t, err)

	// Test unsupported format
	_, err = loadGeoIP("/any/file", "unknown")
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestLoadGeoSiteFunctions(t *testing.T) {
	// Test that loadGeoSite returns appropriate errors for non-existent files
	_, err := loadGeoSite("/nonexistent/file.dat", GeoSiteFormatDAT)
	assert.Error(t, err)

	_, err = loadGeoSite("/nonexistent/file.db", GeoSiteFormatSing)
	assert.Error(t, err)

	// Test unsupported format
	_, err = loadGeoSite("/any/file", "unknown")
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestNewFileGeoLoader(t *testing.T) {
	loader := NewFileGeoLoader("/path/to/geoip.dat", "/path/to/geosite.dat")
	assert.Equal(t, "/path/to/geoip.dat", loader.GeoIPPath)
	assert.Equal(t, "/path/to/geosite.dat", loader.GeoSitePath)
}

func TestFileGeoLoader_Caching(t *testing.T) {
	// Create a mock loader that tracks how many times load is called
	// Since we can't easily mock the file system, we test with empty paths
	// which should return nil without error and be cached
	loader := &FileGeoLoader{}

	// First call
	result1, err1 := loader.LoadGeoIP()
	assert.NoError(t, err1)
	assert.Nil(t, result1)

	// Second call should return same cached result
	result2, err2 := loader.LoadGeoIP()
	assert.NoError(t, err2)
	assert.Nil(t, result2)

	// Same for GeoSite
	result3, err3 := loader.LoadGeoSite()
	assert.NoError(t, err3)
	assert.Nil(t, result3)

	result4, err4 := loader.LoadGeoSite()
	assert.NoError(t, err4)
	assert.Nil(t, result4)
}

func TestAutoGeoLoader_Caching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal valid dat files for testing
	// Note: This requires understanding the protobuf format
	// For now we just test that caching works with nil values

	loader := &AutoGeoLoader{
		DataDir:       tmpDir,
		GeoIPFormat:   GeoIPFormatDAT,
		GeoSiteFormat: GeoSiteFormatDAT,
	}

	// Without URLs and files, it should fail
	// But if we manually set the cache, it should return cached values
	loader.geoIPMap = map[string]*geodat.GeoIP{
		"test": {CountryCode: "TEST"},
	}
	loader.geoSiteMap = map[string]*geodat.GeoSite{
		"test": {CountryCode: "TEST"},
	}

	// Should return cached values
	result, err := loader.LoadGeoIP()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TEST", result["test"].CountryCode)

	result2, err := loader.LoadGeoSite()
	assert.NoError(t, err)
	assert.NotNil(t, result2)
	assert.Equal(t, "TEST", result2["test"].CountryCode)
}

func TestAutoGeoLoader_Logger(t *testing.T) {
	var logged []string
	loader := &AutoGeoLoader{
		Logger: func(format string, args ...interface{}) {
			logged = append(logged, format)
		},
	}

	loader.log("test message %s", "arg")
	assert.Len(t, logged, 1)
	assert.Equal(t, "test message %s", logged[0])

	// Test nil logger doesn't panic
	loader2 := &AutoGeoLoader{}
	loader2.log("should not panic")
}

func TestAutoGeoLoader_UpdateInterval(t *testing.T) {
	loader := &AutoGeoLoader{}
	assert.Equal(t, DefaultUpdateInterval, loader.getUpdateInterval())

	loader.UpdateInterval = 24 * 3600 * 1000000000 // 1 day in nanoseconds
	assert.Equal(t, loader.UpdateInterval, loader.getUpdateInterval())
}
