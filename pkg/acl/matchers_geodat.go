package acl

import (
	"bytes"
	"errors"
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/xflash-panda/acl-engine/pkg/acl/domain"
	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

var _ hostMatcher = (*geoipMatcher)(nil)

type geoipMatcher struct {
	N4      []*net.IPNet // sorted
	N6      []*net.IPNet // sorted
	Inverse bool
}

// matchIP tries to match the given IP address with the corresponding IPNets.
// Note that this function does NOT handle the Inverse flag.
func (m *geoipMatcher) matchIP(ip net.IP) bool {
	var n []*net.IPNet
	if ip4 := ip.To4(); ip4 != nil {
		// N4 stores IPv4 addresses in 4-byte form.
		// Make sure we use it here too, otherwise bytes.Compare will fail.
		ip = ip4
		n = m.N4
	} else {
		n = m.N6
	}
	left, right := 0, len(n)-1
	for left <= right {
		mid := (left + right) / 2
		if n[mid].Contains(ip) {
			return true
		} else if bytes.Compare(n[mid].IP, ip) < 0 {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return false
}

func (m *geoipMatcher) Match(host HostInfo) bool {
	if host.IPv4 != nil {
		if m.matchIP(host.IPv4) {
			return !m.Inverse
		}
	}
	if host.IPv6 != nil {
		if m.matchIP(host.IPv6) {
			return !m.Inverse
		}
	}
	return m.Inverse
}

func newGeoIPMatcher(list *geodat.GeoIP) (*geoipMatcher, error) {
	n4 := make([]*net.IPNet, 0)
	n6 := make([]*net.IPNet, 0)
	for _, cidr := range list.Cidr {
		if len(cidr.Ip) == 4 {
			// IPv4
			n4 = append(n4, &net.IPNet{
				IP:   cidr.Ip,
				Mask: net.CIDRMask(int(cidr.Prefix), 32),
			})
		} else if len(cidr.Ip) == 16 {
			// IPv6
			n6 = append(n6, &net.IPNet{
				IP:   cidr.Ip,
				Mask: net.CIDRMask(int(cidr.Prefix), 128),
			})
		} else {
			return nil, errors.New("invalid IP length")
		}
	}
	// Sort the IPNets, so we can do binary search later.
	sort.Slice(n4, func(i, j int) bool {
		return bytes.Compare(n4[i].IP, n4[j].IP) < 0
	})
	sort.Slice(n6, func(i, j int) bool {
		return bytes.Compare(n6[i].IP, n6[j].IP) < 0
	})
	return &geoipMatcher{
		N4:      n4,
		N6:      n6,
		Inverse: list.InverseMatch,
	}, nil
}

var _ hostMatcher = (*geositeMatcher)(nil)

type geositeDomainType int

const (
	geositeDomainPlain geositeDomainType = iota
	geositeDomainRegex
	geositeDomainRoot
	geositeDomainFull
)

type geositeDomain struct {
	Type  geositeDomainType
	Value string
	Regex *regexp.Regexp
	Attrs map[string]bool
}

type geositeMatcher struct {
	// Fast matchers using succinct trie (for domains without attributes or with matching attributes)
	domainMatcher *domain.Matcher // Full/Root domains that match attributes

	// Slow matchers for special cases
	plainDomains []geositeDomain // Plain (keyword) matches
	regexDomains []geositeDomain // Regex matches
	attrDomains  []geositeDomain // Domains with attributes that need checking

	// Attributes are matched using "and" logic
	Attrs []string
}

func (m *geositeMatcher) matchDomainWithAttrs(d geositeDomain, host HostInfo) bool {
	// Match attributes first
	if len(m.Attrs) > 0 {
		if len(d.Attrs) == 0 {
			return false
		}
		for _, attr := range m.Attrs {
			if !d.Attrs[attr] {
				return false
			}
		}
	}

	switch d.Type {
	case geositeDomainPlain:
		return strings.Contains(host.Name, d.Value)
	case geositeDomainRegex:
		if d.Regex != nil {
			return d.Regex.MatchString(host.Name)
		}
	case geositeDomainFull:
		return host.Name == d.Value
	case geositeDomainRoot:
		if host.Name == d.Value {
			return true
		}
		return strings.HasSuffix(host.Name, "."+d.Value)
	default:
		return false
	}
	return false
}

func (m *geositeMatcher) Match(host HostInfo) bool {
	// Fast path: use succinct trie for Full/Root domains without special attribute requirements
	if m.domainMatcher != nil && m.domainMatcher.Match(host.Name) {
		return true
	}

	// Check plain (keyword) matches
	for _, d := range m.plainDomains {
		if m.matchDomainWithAttrs(d, host) {
			return true
		}
	}

	// Check regex matches
	for _, d := range m.regexDomains {
		if m.matchDomainWithAttrs(d, host) {
			return true
		}
	}

	// Check domains with special attributes
	for _, d := range m.attrDomains {
		if m.matchDomainWithAttrs(d, host) {
			return true
		}
	}

	return false
}

func newGeositeMatcher(list *geodat.GeoSite, attrs []string) (*geositeMatcher, error) {
	// Separate domains by type and attribute requirements
	var fullDomains []string // For exact matches
	var rootDomains []string // For suffix matches
	var plainDomains []geositeDomain
	var regexDomains []geositeDomain
	var attrDomains []geositeDomain

	needsAttrCheck := len(attrs) > 0

	for _, d := range list.Domain {
		attrMap := domainAttributeToMap(d.Attribute)

		// Check if this domain matches required attributes
		matchesAttrs := true
		if needsAttrCheck {
			if len(attrMap) == 0 {
				matchesAttrs = false
			} else {
				for _, attr := range attrs {
					if !attrMap[attr] {
						matchesAttrs = false
						break
					}
				}
			}
		}

		switch d.Type {
		case geodat.Domain_Plain:
			// Plain always needs linear scan
			plainDomains = append(plainDomains, geositeDomain{
				Type:  geositeDomainPlain,
				Value: d.Value,
				Attrs: attrMap,
			})

		case geodat.Domain_Regex:
			regex, err := regexp.Compile(d.Value)
			if err != nil {
				return nil, err
			}
			regexDomains = append(regexDomains, geositeDomain{
				Type:  geositeDomainRegex,
				Regex: regex,
				Attrs: attrMap,
			})

		case geodat.Domain_Full:
			if matchesAttrs && (len(attrMap) == 0 || !needsAttrCheck) {
				// Can use fast matcher
				fullDomains = append(fullDomains, d.Value)
			} else {
				// Needs attribute checking
				attrDomains = append(attrDomains, geositeDomain{
					Type:  geositeDomainFull,
					Value: d.Value,
					Attrs: attrMap,
				})
			}

		case geodat.Domain_RootDomain:
			if matchesAttrs && (len(attrMap) == 0 || !needsAttrCheck) {
				// Can use fast matcher
				rootDomains = append(rootDomains, d.Value)
			} else {
				// Needs attribute checking
				attrDomains = append(attrDomains, geositeDomain{
					Type:  geositeDomainRoot,
					Value: d.Value,
					Attrs: attrMap,
				})
			}

		default:
			return nil, errors.New("unsupported domain type")
		}
	}

	// Build fast domain matcher if we have domains that can use it
	var domainMatcher *domain.Matcher
	if len(fullDomains) > 0 || len(rootDomains) > 0 {
		domainMatcher = domain.NewMatcher(fullDomains, rootDomains)
	}

	return &geositeMatcher{
		domainMatcher: domainMatcher,
		plainDomains:  plainDomains,
		regexDomains:  regexDomains,
		attrDomains:   attrDomains,
		Attrs:         attrs,
	}, nil
}

func domainAttributeToMap(attrs []*geodat.Domain_Attribute) map[string]bool {
	m := make(map[string]bool)
	for _, attr := range attrs {
		// Supposedly there are also int attributes,
		// but nobody seems to use them, so we treat everything as boolean for now.
		m[attr.Key] = true
	}
	return m
}
