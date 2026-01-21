package acl

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
	"github.com/xflash-panda/acl-engine/pkg/acl/metadb"
	"github.com/xflash-panda/acl-engine/pkg/acl/mmdb"
	"github.com/xflash-panda/acl-engine/pkg/acl/singsite"
)

const (
	DefaultUpdateInterval = 7 * 24 * time.Hour // 7 days
)

var (
	ErrGeoIPFormatNotSet   = errors.New("GeoIPFormat not set and cannot be detected from file path")
	ErrGeoSiteFormatNotSet = errors.New("GeoSiteFormat not set and cannot be detected from file path")
	ErrUnsupportedFormat   = errors.New("unsupported geo format")
)

// FileGeoLoader implements GeoLoader interface by loading geo data from files.
type FileGeoLoader struct {
	GeoIPPath     string
	GeoSitePath   string
	GeoIPFormat   GeoIPFormat   // Optional, auto-detected from path if not set
	GeoSiteFormat GeoSiteFormat // Optional, auto-detected from path if not set

	geoIPOnce   sync.Once
	geoIPMap    map[string]*geodat.GeoIP
	geoIPErr    error
	geoSiteOnce sync.Once
	geoSiteMap  map[string]*geodat.GeoSite
	geoSiteErr  error
}

// NewFileGeoLoader creates a new FileGeoLoader with the given file paths.
// Format is auto-detected from file extensions.
func NewFileGeoLoader(geoIPPath, geoSitePath string) *FileGeoLoader {
	return &FileGeoLoader{
		GeoIPPath:   geoIPPath,
		GeoSitePath: geoSitePath,
	}
}

func (l *FileGeoLoader) getGeoIPFormat() GeoIPFormat {
	if l.GeoIPFormat != "" {
		return l.GeoIPFormat
	}
	return DetectGeoIPFormat(l.GeoIPPath)
}

func (l *FileGeoLoader) getGeoSiteFormat() GeoSiteFormat {
	if l.GeoSiteFormat != "" {
		return l.GeoSiteFormat
	}
	return DetectGeoSiteFormat(l.GeoSitePath)
}

// LoadGeoIP loads the GeoIP database from the configured file path.
// The result is cached after the first call.
func (l *FileGeoLoader) LoadGeoIP() (map[string]*geodat.GeoIP, error) {
	l.geoIPOnce.Do(func() {
		if l.GeoIPPath == "" {
			return
		}
		format := l.getGeoIPFormat()
		if format == "" {
			l.geoIPErr = ErrGeoIPFormatNotSet
			return
		}
		l.geoIPMap, l.geoIPErr = loadGeoIP(l.GeoIPPath, format)
	})
	return l.geoIPMap, l.geoIPErr
}

// LoadGeoSite loads the GeoSite database from the configured file path.
// The result is cached after the first call.
func (l *FileGeoLoader) LoadGeoSite() (map[string]*geodat.GeoSite, error) {
	l.geoSiteOnce.Do(func() {
		if l.GeoSitePath == "" {
			return
		}
		format := l.getGeoSiteFormat()
		if format == "" {
			l.geoSiteErr = ErrGeoSiteFormatNotSet
			return
		}
		l.geoSiteMap, l.geoSiteErr = loadGeoSite(l.GeoSitePath, format)
	})
	return l.geoSiteMap, l.geoSiteErr
}

// NilGeoLoader is a GeoLoader that always returns nil (no geo data).
// Useful when you don't need GeoIP/GeoSite matching.
type NilGeoLoader struct{}

func (l *NilGeoLoader) LoadGeoIP() (map[string]*geodat.GeoIP, error) {
	return nil, nil
}

func (l *NilGeoLoader) LoadGeoSite() (map[string]*geodat.GeoSite, error) {
	return nil, nil
}

// AutoGeoLoader implements GeoLoader with automatic download support.
// It downloads geo data files from CDN if they don't exist or are outdated.
type AutoGeoLoader struct {
	// GeoIPPath is the full path to the geoip file.
	// If empty, uses DataDir + default filename based on GeoIPFormat.
	GeoIPPath string
	// GeoSitePath is the full path to the geosite file.
	// If empty, uses DataDir + default filename based on GeoSiteFormat.
	GeoSitePath string
	// DataDir is the directory to store downloaded files.
	// Required when GeoIPPath/GeoSitePath is not set.
	DataDir string
	// GeoIPFormat specifies the GeoIP file format.
	// If empty, auto-detected from GeoIPPath extension.
	GeoIPFormat GeoIPFormat
	// GeoSiteFormat specifies the GeoSite file format.
	// If empty, auto-detected from GeoSitePath extension.
	GeoSiteFormat GeoSiteFormat
	// GeoIPURL is the download URL for the geoip file.
	// Required when auto-downloading is needed.
	GeoIPURL string
	// GeoSiteURL is the download URL for the geosite file.
	// Required when auto-downloading is needed.
	GeoSiteURL string
	// UpdateInterval is the interval to check for updates.
	// If zero, uses DefaultUpdateInterval (7 days).
	UpdateInterval time.Duration
	// Logger is called when downloading or errors occur (optional).
	Logger func(format string, args ...interface{})

	geoIPMap   map[string]*geodat.GeoIP
	geoSiteMap map[string]*geodat.GeoSite
	mu         sync.Mutex
}

func (l *AutoGeoLoader) log(format string, args ...interface{}) {
	if l.Logger != nil {
		l.Logger(format, args...)
	}
}

func (l *AutoGeoLoader) getGeoIPFormat() GeoIPFormat {
	if l.GeoIPFormat != "" {
		return l.GeoIPFormat
	}
	if l.GeoIPPath != "" {
		return DetectGeoIPFormat(l.GeoIPPath)
	}
	return ""
}

func (l *AutoGeoLoader) getGeoSiteFormat() GeoSiteFormat {
	if l.GeoSiteFormat != "" {
		return l.GeoSiteFormat
	}
	if l.GeoSitePath != "" {
		return DetectGeoSiteFormat(l.GeoSitePath)
	}
	return ""
}

func (l *AutoGeoLoader) getGeoIPPath() string {
	if l.GeoIPPath != "" {
		return l.GeoIPPath
	}
	format := l.getGeoIPFormat()
	filename := DefaultGeoIPFilename(format)
	if l.DataDir != "" {
		return filepath.Join(l.DataDir, filename)
	}
	return filename
}

func (l *AutoGeoLoader) getGeoSitePath() string {
	if l.GeoSitePath != "" {
		return l.GeoSitePath
	}
	format := l.getGeoSiteFormat()
	filename := DefaultGeoSiteFilename(format)
	if l.DataDir != "" {
		return filepath.Join(l.DataDir, filename)
	}
	return filename
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
	if url == "" {
		return fmt.Errorf("download URL not configured for %s", filename)
	}
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = os.Remove(tmpName) }()

	_, err = io.Copy(tmpFile, resp.Body)
	_ = tmpFile.Close()
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
func (l *AutoGeoLoader) LoadGeoIP() (map[string]*geodat.GeoIP, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.geoIPMap != nil {
		return l.geoIPMap, nil
	}

	format := l.getGeoIPFormat()
	if format == "" {
		return nil, ErrGeoIPFormatNotSet
	}

	filename := l.getGeoIPPath()

	// Try to download if needed
	if l.shouldDownload(filename) {
		err := l.download(filename, l.GeoIPURL, func(f string) error {
			_, err := loadGeoIP(f, format)
			return err
		})
		if err != nil {
			// If download fails but file exists, try to use it
			if _, serr := os.Stat(filename); os.IsNotExist(serr) {
				return nil, err
			}
		}
	}

	m, err := loadGeoIP(filename, format)
	if err != nil {
		return nil, err
	}
	l.geoIPMap = m
	return m, nil
}

// LoadGeoSite loads the GeoSite database, downloading if necessary.
func (l *AutoGeoLoader) LoadGeoSite() (map[string]*geodat.GeoSite, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.geoSiteMap != nil {
		return l.geoSiteMap, nil
	}

	format := l.getGeoSiteFormat()
	if format == "" {
		return nil, ErrGeoSiteFormatNotSet
	}

	filename := l.getGeoSitePath()

	// Try to download if needed
	if l.shouldDownload(filename) {
		err := l.download(filename, l.GeoSiteURL, func(f string) error {
			_, err := loadGeoSite(f, format)
			return err
		})
		if err != nil {
			// If download fails but file exists, try to use it
			if _, serr := os.Stat(filename); os.IsNotExist(serr) {
				return nil, err
			}
		}
	}

	m, err := loadGeoSite(filename, format)
	if err != nil {
		return nil, err
	}
	l.geoSiteMap = m
	return m, nil
}

// loadGeoIP loads GeoIP data from a file based on the specified format.
func loadGeoIP(filename string, format GeoIPFormat) (map[string]*geodat.GeoIP, error) {
	switch format {
	case GeoIPFormatDAT:
		return geodat.LoadGeoIP(filename)
	case GeoIPFormatMMDB:
		return mmdb.LoadGeoIP(filename)
	case GeoIPFormatMetaDB:
		return metadb.LoadGeoIP(filename)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// loadGeoSite loads GeoSite data from a file based on the specified format.
func loadGeoSite(filename string, format GeoSiteFormat) (map[string]*geodat.GeoSite, error) {
	switch format {
	case GeoSiteFormatDAT:
		return geodat.LoadGeoSite(filename)
	case GeoSiteFormatSing:
		return singsite.LoadGeoSite(filename)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}
