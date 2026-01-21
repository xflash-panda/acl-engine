package acl

import (
	"testing"
)

func TestDetectGeoIPFormat(t *testing.T) {
	tests := []struct {
		path     string
		expected GeoIPFormat
	}{
		{"geoip.dat", GeoIPFormatDAT},
		{"path/to/geoip.dat", GeoIPFormatDAT},
		{"/absolute/path/geoip.dat", GeoIPFormatDAT},
		{"country.mmdb", GeoIPFormatMMDB},
		{"path/to/country.mmdb", GeoIPFormatMMDB},
		{"GeoLite2-Country.mmdb", GeoIPFormatMMDB},
		{"geoip.metadb", GeoIPFormatMetaDB},
		{"path/to/geoip.metadb", GeoIPFormatMetaDB},
		{"geoip.MMDB", GeoIPFormatMMDB},     // case insensitive
		{"geoip.DAT", GeoIPFormatDAT},       // case insensitive
		{"geoip.MetaDB", GeoIPFormatMetaDB}, // case insensitive
		{"unknown.txt", ""},
		{"noextension", ""},
		{"", ""},
		{"file.db", ""}, // .db is for GeoSite
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectGeoIPFormat(tt.path)
			if got != tt.expected {
				t.Errorf("DetectGeoIPFormat(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestDetectGeoSiteFormat(t *testing.T) {
	tests := []struct {
		path     string
		expected GeoSiteFormat
	}{
		{"geosite.dat", GeoSiteFormatDAT},
		{"path/to/geosite.dat", GeoSiteFormatDAT},
		{"/absolute/path/geosite.dat", GeoSiteFormatDAT},
		{"geosite.db", GeoSiteFormatSing},
		{"path/to/geosite.db", GeoSiteFormatSing},
		{"geosite.DAT", GeoSiteFormatDAT}, // case insensitive
		{"geosite.DB", GeoSiteFormatSing}, // case insensitive
		{"unknown.txt", ""},
		{"noextension", ""},
		{"", ""},
		{"file.mmdb", ""}, // .mmdb is for GeoIP
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectGeoSiteFormat(tt.path)
			if got != tt.expected {
				t.Errorf("DetectGeoSiteFormat(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestDefaultGeoIPFilename(t *testing.T) {
	tests := []struct {
		format   GeoIPFormat
		expected string
	}{
		{GeoIPFormatDAT, "geoip.dat"},
		{GeoIPFormatMMDB, "geoip.mmdb"},
		{GeoIPFormatMetaDB, "geoip.metadb"},
		{"", "geoip.dat"},        // default
		{"unknown", "geoip.dat"}, // unknown format defaults to dat
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got := DefaultGeoIPFilename(tt.format)
			if got != tt.expected {
				t.Errorf("DefaultGeoIPFilename(%q) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestDefaultGeoSiteFilename(t *testing.T) {
	tests := []struct {
		format   GeoSiteFormat
		expected string
	}{
		{GeoSiteFormatDAT, "geosite.dat"},
		{GeoSiteFormatSing, "geosite.db"},
		{"", "geosite.dat"},        // default
		{"unknown", "geosite.dat"}, // unknown format defaults to dat
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got := DefaultGeoSiteFilename(tt.format)
			if got != tt.expected {
				t.Errorf("DefaultGeoSiteFilename(%q) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestGeoFormatConstants(t *testing.T) {
	// Test that format constants have expected values
	if GeoIPFormatDAT != "dat" {
		t.Errorf("GeoIPFormatDAT = %q, want %q", GeoIPFormatDAT, "dat")
	}
	if GeoIPFormatMMDB != "mmdb" {
		t.Errorf("GeoIPFormatMMDB = %q, want %q", GeoIPFormatMMDB, "mmdb")
	}
	if GeoIPFormatMetaDB != "metadb" {
		t.Errorf("GeoIPFormatMetaDB = %q, want %q", GeoIPFormatMetaDB, "metadb")
	}
	if GeoSiteFormatDAT != "dat" {
		t.Errorf("GeoSiteFormatDAT = %q, want %q", GeoSiteFormatDAT, "dat")
	}
	if GeoSiteFormatSing != "db" {
		t.Errorf("GeoSiteFormatSing = %q, want %q", GeoSiteFormatSing, "db")
	}
}

func TestMetaCubeXURLConstants(t *testing.T) {
	// Test that URL constants are not empty and have expected patterns
	urls := map[string]string{
		"MetaCubeXGeoIPDatURL":    MetaCubeXGeoIPDatURL,
		"MetaCubeXGeoIPMMDBURL":   MetaCubeXGeoIPMMDBURL,
		"MetaCubeXGeoIPMetaDBURL": MetaCubeXGeoIPMetaDBURL,
		"MetaCubeXGeoSiteDatURL":  MetaCubeXGeoSiteDatURL,
		"MetaCubeXGeoSiteDBURL":   MetaCubeXGeoSiteDBURL,
	}

	for name, url := range urls {
		if url == "" {
			t.Errorf("%s is empty", name)
		}
		if len(url) < 10 {
			t.Errorf("%s = %q, seems too short", name, url)
		}
	}
}
