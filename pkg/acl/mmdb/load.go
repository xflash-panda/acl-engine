package mmdb

import (
	"net"
	"strings"

	"github.com/oschwald/maxminddb-golang"
	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

// mmdbRecord represents a record in the MMDB file.
type mmdbRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

// LoadGeoIP loads a MMDB file and converts it to the geodat format.
// The keys of the map (country codes) are all normalized to lowercase.
func LoadGeoIP(filename string) (map[string]*geodat.GeoIP, error) {
	db, err := maxminddb.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	// Map to collect CIDRs by country code
	countryNetworks := make(map[string][]*geodat.CIDR)

	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		var record mmdbRecord
		subnet, err := networks.Network(&record)
		if err != nil {
			return nil, err
		}

		countryCode := strings.ToLower(record.Country.ISOCode)
		if countryCode == "" {
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

		countryNetworks[countryCode] = append(countryNetworks[countryCode], cidr)
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

// Verify verifies that a MMDB file can be loaded successfully.
func Verify(filename string) error {
	db, err := maxminddb.Open(filename)
	if err != nil {
		return err
	}
	_ = db.Close()
	return nil
}

// LookupIP looks up the country code for an IP address.
// Returns empty string if not found.
func LookupIP(filename string, ip net.IP) (string, error) {
	db, err := maxminddb.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() { _ = db.Close() }()

	var record mmdbRecord
	err = db.Lookup(ip, &record)
	if err != nil {
		return "", err
	}

	return strings.ToLower(record.Country.ISOCode), nil
}
