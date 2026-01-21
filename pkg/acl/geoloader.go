package acl

import (
	"sync"

	"github.com/xflash-panda/acl-engine/pkg/acl/v2geo"
)

// FileGeoLoader implements GeoLoader interface by loading geo data from files.
type FileGeoLoader struct {
	GeoIPPath   string
	GeoSitePath string

	geoIPOnce   sync.Once
	geoIPMap    map[string]*v2geo.GeoIP
	geoIPErr    error
	geoSiteOnce sync.Once
	geoSiteMap  map[string]*v2geo.GeoSite
	geoSiteErr  error
}

// NewFileGeoLoader creates a new FileGeoLoader with the given file paths.
func NewFileGeoLoader(geoIPPath, geoSitePath string) *FileGeoLoader {
	return &FileGeoLoader{
		GeoIPPath:   geoIPPath,
		GeoSitePath: geoSitePath,
	}
}

// LoadGeoIP loads the GeoIP database from the configured file path.
// The result is cached after the first call.
func (l *FileGeoLoader) LoadGeoIP() (map[string]*v2geo.GeoIP, error) {
	l.geoIPOnce.Do(func() {
		if l.GeoIPPath == "" {
			return
		}
		l.geoIPMap, l.geoIPErr = v2geo.LoadGeoIP(l.GeoIPPath)
	})
	return l.geoIPMap, l.geoIPErr
}

// LoadGeoSite loads the GeoSite database from the configured file path.
// The result is cached after the first call.
func (l *FileGeoLoader) LoadGeoSite() (map[string]*v2geo.GeoSite, error) {
	l.geoSiteOnce.Do(func() {
		if l.GeoSitePath == "" {
			return
		}
		l.geoSiteMap, l.geoSiteErr = v2geo.LoadGeoSite(l.GeoSitePath)
	})
	return l.geoSiteMap, l.geoSiteErr
}

// NilGeoLoader is a GeoLoader that always returns nil (no geo data).
// Useful when you don't need GeoIP/GeoSite matching.
type NilGeoLoader struct{}

func (l *NilGeoLoader) LoadGeoIP() (map[string]*v2geo.GeoIP, error) {
	return nil, nil
}

func (l *NilGeoLoader) LoadGeoSite() (map[string]*v2geo.GeoSite, error) {
	return nil, nil
}
