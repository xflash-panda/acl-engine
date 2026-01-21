package metadb

import (
	"net"
	"strings"

	"github.com/metacubex/geo/geoip"
	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

// LoadGeoIP loads a MetaDB file and converts it to the geodat format.
// The keys of the map (country codes) are all normalized to lowercase.
func LoadGeoIP(filename string) (map[string]*geodat.GeoIP, error) {
	db, err := geoip.FromFile(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	reader := db.Reader()
	if reader == nil {
		return nil, geoip.ErrInvalidDatabase
	}

	// Map to collect CIDRs by country code
	countryNetworks := make(map[string][]*geodat.CIDR)

	networks := reader.Networks()
	for networks.Next() {
		subnet, err := networks.Network(&struct{}{})
		if err != nil {
			return nil, err
		}

		codes := db.LookupCode(subnet.IP)
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
	db, err := geoip.FromFile(filename)
	if err != nil {
		return err
	}
	_ = db.Close()
	return nil
}

// LookupIP looks up the country codes for an IP address.
// Returns empty slice if not found.
func LookupIP(filename string, ip net.IP) ([]string, error) {
	db, err := geoip.FromFile(filename)
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
