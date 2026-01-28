package metadb

import (
	"net"
	"strings"

	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

// extractCodesFromMetaV0 extracts country codes from MetaV0 format data.
// MetaV0 can store data as either a single string or a slice of strings.
func extractCodesFromMetaV0(data any) []string {
	switch v := data.(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []any:
		codes := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				codes = append(codes, s)
			}
		}
		if len(codes) == 0 {
			return nil
		}
		return codes
	}
	return nil
}

// LoadGeoIP loads a MetaDB file and converts it to the geodat format.
// The keys of the map (country codes) are all normalized to lowercase.
func LoadGeoIP(filename string) (map[string]*geodat.GeoIP, error) {
	db, err := OpenDatabase(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	reader := db.Reader()
	if reader == nil {
		return nil, ErrInvalidDatabase
	}

	// Map to collect CIDRs by country code
	countryNetworks := make(map[string][]*geodat.CIDR)

	networks := reader.Networks()
	for networks.Next() {
		// Use the appropriate type for unmarshaling based on database type
		var subnet *net.IPNet
		var codes []string
		var err error

		switch db.Type() {
		case TypeMetaV0:
			// Meta-geoip0 stores data as slice of strings or string
			var metaCodes any
			subnet, err = networks.Network(&metaCodes)
			if err == nil {
				codes = extractCodesFromMetaV0(metaCodes)
			}
		case TypeSing:
			// sing-geoip stores data as string
			var code string
			subnet, err = networks.Network(&code)
			if err == nil && code != "" {
				codes = []string{code}
			}
		default:
			// MaxMind stores full country data
			var country geoip2Country
			subnet, err = networks.Network(&country)
			if err == nil && country.Country.IsoCode != "" {
				codes = []string{country.Country.IsoCode}
			}
		}

		if err != nil {
			return nil, err
		}

		if len(codes) == 0 {
			continue
		}

		ones, _ := subnet.Mask.Size()
		ip := subnet.IP

		// Normalize IP to 4-byte for IPv4 or 16-byte for IPv6
		if ip4 := ip.To4(); ip4 != nil {
			ip = ip4
		} else {
			ip = ip.To16()
		}

		cidr := &geodat.CIDR{
			Ip:     ip,
			Prefix: uint32(ones), //nolint:gosec // ones is 0-128 for CIDR prefix, safe to convert
		}

		// MetaDB supports multiple country codes per IP
		for _, code := range codes {
			code = strings.ToLower(code)
			countryNetworks[code] = append(countryNetworks[code], cidr)
		}
	}

	if err := networks.Err(); err != nil {
		return nil, err
	}

	// Convert to map[string]*geodat.GeoIP
	result := make(map[string]*geodat.GeoIP)
	for code, cidrs := range countryNetworks {
		result[code] = &geodat.GeoIP{
			CountryCode: strings.ToUpper(code),
			Cidr:        cidrs,
		}
	}

	return result, nil
}

// Verify verifies that a MetaDB file can be loaded successfully.
func Verify(filename string) error {
	db, err := OpenDatabase(filename)
	if err != nil {
		return err
	}
	return db.Close()
}

// LookupIP looks up the country codes for an IP address.
// Returns empty slice if not found.
func LookupIP(filename string, ip net.IP) ([]string, error) {
	db, err := OpenDatabase(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	codes := db.LookupCode(ip)
	for i, code := range codes {
		codes[i] = strings.ToLower(code)
	}
	return codes, nil
}
