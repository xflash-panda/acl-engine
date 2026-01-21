package acl

import (
	"path/filepath"
	"strings"
)

// GeoIPFormat represents the format of GeoIP data files.
type GeoIPFormat string

const (
	GeoIPFormatDAT    GeoIPFormat = "dat"
	GeoIPFormatMMDB   GeoIPFormat = "mmdb"
	GeoIPFormatMetaDB GeoIPFormat = "metadb"
)

// GeoSiteFormat represents the format of GeoSite data files.
type GeoSiteFormat string

const (
	GeoSiteFormatDAT    GeoSiteFormat = "dat"
	GeoSiteFormatSing GeoSiteFormat = "db"
)

// DetectGeoIPFormat detects the GeoIP format from a file path based on extension.
// Returns empty string if the format cannot be detected.
func DetectGeoIPFormat(path string) GeoIPFormat {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mmdb":
		return GeoIPFormatMMDB
	case ".metadb":
		return GeoIPFormatMetaDB
	case ".dat":
		return GeoIPFormatDAT
	default:
		return ""
	}
}

// DetectGeoSiteFormat detects the GeoSite format from a file path based on extension.
// Returns empty string if the format cannot be detected.
func DetectGeoSiteFormat(path string) GeoSiteFormat {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".db":
		return GeoSiteFormatSing
	case ".dat":
		return GeoSiteFormatDAT
	default:
		return ""
	}
}

// DefaultGeoIPFilename returns the default filename for the given GeoIP format.
func DefaultGeoIPFilename(format GeoIPFormat) string {
	switch format {
	case GeoIPFormatMMDB:
		return "geoip.mmdb"
	case GeoIPFormatMetaDB:
		return "geoip.metadb"
	default:
		return "geoip.dat"
	}
}

// DefaultGeoSiteFilename returns the default filename for the given GeoSite format.
func DefaultGeoSiteFilename(format GeoSiteFormat) string {
	switch format {
	case GeoSiteFormatSing:
		return "geosite.db"
	default:
		return "geosite.dat"
	}
}

// MetaCubeX CDN URLs for various geo data formats.
const (
	MetaCubeXGeoIPDatURL    = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat"
	MetaCubeXGeoIPMMDBURL   = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/country.mmdb"
	MetaCubeXGeoIPMetaDBURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
	MetaCubeXGeoSiteDatURL  = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"
	MetaCubeXGeoSiteDBURL   = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.db"
)
