package acl

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xflash-panda/acl-engine/pkg/acl/v2geo"
)

const (
	DefaultGeoIPFilename   = "geoip.dat"
	DefaultGeoSiteFilename = "geosite.dat"
	DefaultGeoIPURL        = "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/geoip.dat"
	DefaultGeoSiteURL      = "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/geosite.dat"
	DefaultUpdateInterval  = 7 * 24 * time.Hour // 7 days
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

// AutoGeoLoader implements GeoLoader with automatic download support.
// It downloads geo data files from CDN if they don't exist or are outdated.
type AutoGeoLoader struct {
	// GeoIPPath is the path to geoip.dat file.
	// If empty, uses DefaultGeoIPFilename in DataDir.
	GeoIPPath string
	// GeoSitePath is the path to geosite.dat file.
	// If empty, uses DefaultGeoSiteFilename in DataDir.
	GeoSitePath string
	// DataDir is the directory to store downloaded files.
	// If empty, uses current directory.
	DataDir string
	// GeoIPURL is the download URL for geoip.dat.
	// If empty, uses DefaultGeoIPURL.
	GeoIPURL string
	// GeoSiteURL is the download URL for geosite.dat.
	// If empty, uses DefaultGeoSiteURL.
	GeoSiteURL string
	// UpdateInterval is the interval to check for updates.
	// If zero, uses DefaultUpdateInterval (7 days).
	UpdateInterval time.Duration
	// Logger is called when downloading or errors occur (optional).
	Logger func(format string, args ...interface{})

	geoIPMap   map[string]*v2geo.GeoIP
	geoSiteMap map[string]*v2geo.GeoSite
	mu         sync.Mutex
}

// NewAutoGeoLoader creates an AutoGeoLoader with default settings.
// dataDir is the directory to store downloaded geo data files.
func NewAutoGeoLoader(dataDir string) *AutoGeoLoader {
	return &AutoGeoLoader{
		DataDir: dataDir,
	}
}

func (l *AutoGeoLoader) log(format string, args ...interface{}) {
	if l.Logger != nil {
		l.Logger(format, args...)
	}
}

func (l *AutoGeoLoader) getGeoIPPath() string {
	if l.GeoIPPath != "" {
		return l.GeoIPPath
	}
	if l.DataDir != "" {
		return filepath.Join(l.DataDir, DefaultGeoIPFilename)
	}
	return DefaultGeoIPFilename
}

func (l *AutoGeoLoader) getGeoSitePath() string {
	if l.GeoSitePath != "" {
		return l.GeoSitePath
	}
	if l.DataDir != "" {
		return filepath.Join(l.DataDir, DefaultGeoSiteFilename)
	}
	return DefaultGeoSiteFilename
}

func (l *AutoGeoLoader) getGeoIPURL() string {
	if l.GeoIPURL != "" {
		return l.GeoIPURL
	}
	return DefaultGeoIPURL
}

func (l *AutoGeoLoader) getGeoSiteURL() string {
	if l.GeoSiteURL != "" {
		return l.GeoSiteURL
	}
	return DefaultGeoSiteURL
}

func (l *AutoGeoLoader) getUpdateInterval() time.Duration {
	if l.UpdateInterval > 0 {
		return l.UpdateInterval
	}
	return DefaultUpdateInterval
}

func (l *AutoGeoLoader) shouldDownload(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return true
	}
	if info.Size() == 0 {
		return true
	}
	return time.Since(info.ModTime()) > l.getUpdateInterval()
}

func (l *AutoGeoLoader) download(filename, url string, checkFunc func(string) error) error {
	l.log("Downloading %s from %s", filename, url)

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory failed: %w", err)
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		l.log("Download failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		l.log("Download failed: %v", err)
		return err
	}

	// Write to temp file first
	tmpFile, err := os.CreateTemp(dir, ".geoloader.tmp.*")
	if err != nil {
		l.log("Create temp file failed: %v", err)
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		l.log("Write file failed: %v", err)
		return err
	}

	// Verify the downloaded file
	if err := checkFunc(tmpName); err != nil {
		l.log("Integrity check failed: %v", err)
		return fmt.Errorf("integrity check failed: %w", err)
	}

	// Move to final location
	if err := os.Rename(tmpName, filename); err != nil {
		l.log("Rename failed: %v", err)
		return fmt.Errorf("rename failed: %w", err)
	}

	l.log("Downloaded %s successfully", filename)
	return nil
}

// LoadGeoIP loads the GeoIP database, downloading if necessary.
func (l *AutoGeoLoader) LoadGeoIP() (map[string]*v2geo.GeoIP, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.geoIPMap != nil {
		return l.geoIPMap, nil
	}

	filename := l.getGeoIPPath()

	// Try to download if needed
	if l.shouldDownload(filename) {
		err := l.download(filename, l.getGeoIPURL(), func(f string) error {
			_, err := v2geo.LoadGeoIP(f)
			return err
		})
		if err != nil {
			// If download fails but file exists, try to use it
			if _, serr := os.Stat(filename); os.IsNotExist(serr) {
				return nil, err
			}
		}
	}

	m, err := v2geo.LoadGeoIP(filename)
	if err != nil {
		return nil, err
	}
	l.geoIPMap = m
	return m, nil
}

// LoadGeoSite loads the GeoSite database, downloading if necessary.
func (l *AutoGeoLoader) LoadGeoSite() (map[string]*v2geo.GeoSite, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.geoSiteMap != nil {
		return l.geoSiteMap, nil
	}

	filename := l.getGeoSitePath()

	// Try to download if needed
	if l.shouldDownload(filename) {
		err := l.download(filename, l.getGeoSiteURL(), func(f string) error {
			_, err := v2geo.LoadGeoSite(f)
			return err
		})
		if err != nil {
			// If download fails but file exists, try to use it
			if _, serr := os.Stat(filename); os.IsNotExist(serr) {
				return nil, err
			}
		}
	}

	m, err := v2geo.LoadGeoSite(filename)
	if err != nil {
		return nil, err
	}
	l.geoSiteMap = m
	return m, nil
}
