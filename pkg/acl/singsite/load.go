package singsite

import (
	"strings"

	"github.com/metacubex/geo/encoding/singgeo"
	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

// LoadGeoSite loads a sing-geosite db file and converts it to the geodat format.
// The keys of the map (site codes) are all normalized to lowercase.
func LoadGeoSite(filename string) (map[string]*geodat.GeoSite, error) {
	reader, codes, err := singgeo.LoadSiteFromFile(filename)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*geodat.GeoSite)
	for _, code := range codes {
		items, err := reader.Read(code)
		if err != nil {
			return nil, err
		}

		domains := make([]*geodat.Domain, 0, len(items))
		for _, item := range items {
			var domainType geodat.Domain_Type
			switch item.Type {
			case singgeo.RuleTypeDomain:
				domainType = geodat.Domain_Full
			case singgeo.RuleTypeDomainSuffix:
				domainType = geodat.Domain_RootDomain
			case singgeo.RuleTypeDomainKeyword:
				domainType = geodat.Domain_Plain
			case singgeo.RuleTypeDomainRegex:
				domainType = geodat.Domain_Regex
			default:
				continue
			}

			domains = append(domains, &geodat.Domain{
				Type:  domainType,
				Value: item.Value,
			})
		}

		lowerCode := strings.ToLower(code)
		result[lowerCode] = &geodat.GeoSite{
			CountryCode: code,
			Domain:      domains,
		}
	}

	return result, nil
}

// Verify verifies that a sing-geosite db file can be loaded successfully.
func Verify(filename string) error {
	_, _, err := singgeo.LoadSiteFromFile(filename)
	return err
}
