package metadb

import (
	"errors"
	"fmt"
	"net"

	"github.com/oschwald/maxminddb-golang"
)

// DatabaseType represents the type of GeoIP database.
type DatabaseType string

const (
	// TypeMaxmind is the standard MaxMind GeoIP2/GeoLite2 format.
	TypeMaxmind DatabaseType = "MaxMind"
	// TypeSing is the sing-geoip format.
	TypeSing DatabaseType = "sing-geoip"
	// TypeMetaV0 is the Meta-geoip0 format.
	TypeMetaV0 DatabaseType = "Meta-geoip0"
)

// ErrInvalidDatabase is returned when the database format is not recognized.
var ErrInvalidDatabase = errors.New("invalid GeoIP database")

// Database wraps a MaxMind DB reader with type information.
type Database struct {
	reader   *maxminddb.Reader
	dbType   DatabaseType
}

// geoip2Country is used for parsing MaxMind GeoIP2 format.
type geoip2Country struct {
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

// OpenDatabase opens a GeoIP database file.
// Supports MaxMind GeoIP2/GeoLite2, sing-geoip, and Meta-geoip0 formats.
func OpenDatabase(filename string) (*Database, error) {
	reader, err := maxminddb.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db := &Database{
		reader: reader,
		dbType: detectDatabaseType(reader.Metadata.DatabaseType),
	}

	return db, nil
}

// OpenDatabaseFromBytes opens a GeoIP database from bytes.
func OpenDatabaseFromBytes(data []byte) (*Database, error) {
	reader, err := maxminddb.FromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("open database from bytes: %w", err)
	}

	db := &Database{
		reader: reader,
		dbType: detectDatabaseType(reader.Metadata.DatabaseType),
	}

	return db, nil
}

// detectDatabaseType determines the database type from metadata.
func detectDatabaseType(dbType string) DatabaseType {
	switch DatabaseType(dbType) {
	case TypeSing:
		return TypeSing
	case TypeMetaV0:
		return TypeMetaV0
	default:
		return TypeMaxmind
	}
}

// Reader returns the underlying MaxMind DB reader.
func (db *Database) Reader() *maxminddb.Reader {
	return db.reader
}

// Type returns the database type.
func (db *Database) Type() DatabaseType {
	return db.dbType
}

// LookupCode looks up country codes for an IP address.
// Returns nil if not found.
func (db *Database) LookupCode(ip net.IP) []string {
	switch db.dbType {
	case TypeMaxmind:
		var country geoip2Country
		_ = db.reader.Lookup(ip, &country)
		if country.Country.IsoCode == "" {
			return nil
		}
		return []string{country.Country.IsoCode}

	case TypeSing:
		var code string
		_ = db.reader.Lookup(ip, &code)
		if code == "" {
			return nil
		}
		return []string{code}

	case TypeMetaV0:
		var codes any
		_ = db.reader.Lookup(ip, &codes)
		switch c := codes.(type) {
		case string:
			return []string{c}
		case []any:
			result := make([]string, 0, len(c))
			for _, v := range c {
				if s, ok := v.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
		return nil

	default:
		return nil
	}
}

// Close closes the database.
func (db *Database) Close() error {
	return db.reader.Close()
}
