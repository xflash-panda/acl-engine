package router

import (
	"net"
	"os"
	"strings"

	"github.com/xflash-panda/acl-engine/pkg/acl"
	"github.com/xflash-panda/acl-engine/pkg/outbound"
	"github.com/xflash-panda/acl-engine/pkg/resolver"
)

const (
	defaultCacheSize = 1024
)

// Router routes connections to different outbounds based on ACL rules.
// It implements the outbound.Outbound interface.
type Router struct {
	ruleSet  acl.CompiledRuleSet[outbound.Outbound]
	default_ outbound.Outbound
	resolver resolver.Resolver
}

// Option configures the Router.
type Option func(*routerOptions)

type routerOptions struct {
	cacheSize int
	resolver  resolver.Resolver
}

// WithCacheSize sets the LRU cache size for rule matching results.
func WithCacheSize(size int) Option {
	return func(o *routerOptions) {
		o.cacheSize = size
	}
}

// WithResolver sets a custom DNS resolver for the router.
// If set, the router will resolve hostnames before matching ACL rules.
func WithResolver(r resolver.Resolver) Option {
	return func(o *routerOptions) {
		o.resolver = r
	}
}

// OutboundEntry represents an outbound with a name.
type OutboundEntry struct {
	Name     string
	Outbound outbound.Outbound
}

// New creates a new Router from ACL rules string.
// The rules parameter contains ACL rules in text format.
// The outbounds parameter is a list of named outbounds.
// The geoLoader is used to load GeoIP/GeoSite databases on demand.
func New(rules string, outbounds []OutboundEntry, geoLoader acl.GeoLoader, opts ...Option) (*Router, error) {
	options := &routerOptions{
		cacheSize: defaultCacheSize,
	}
	for _, opt := range opts {
		opt(options)
	}

	trs, err := acl.ParseTextRules(rules)
	if err != nil {
		return nil, err
	}
	obMap := outboundsToMap(outbounds)
	rs, err := acl.Compile[outbound.Outbound](trs, obMap, options.cacheSize, geoLoader)
	if err != nil {
		return nil, err
	}
	return &Router{
		ruleSet:  rs,
		default_: obMap["default"],
		resolver: options.resolver,
	}, nil
}

// NewFromFile creates a new Router from an ACL rules file.
func NewFromFile(filename string, outbounds []OutboundEntry, geoLoader acl.GeoLoader, opts ...Option) (*Router, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return New(string(bs), outbounds, geoLoader, opts...)
}

func outboundsToMap(outbounds []OutboundEntry) map[string]outbound.Outbound {
	obMap := make(map[string]outbound.Outbound)
	for _, ob := range outbounds {
		obMap[strings.ToLower(ob.Name)] = ob.Outbound
	}
	// Add built-in outbounds if not overridden
	if _, ok := obMap["direct"]; !ok {
		obMap["direct"] = outbound.NewDirect(outbound.DirectModeAuto)
	}
	if _, ok := obMap["reject"]; !ok {
		obMap["reject"] = outbound.NewReject()
	}
	if _, ok := obMap["default"]; !ok {
		if len(outbounds) > 0 {
			obMap["default"] = outbounds[0].Outbound
		} else {
			obMap["default"] = obMap["direct"]
		}
	}
	return obMap
}

func (r *Router) resolve(addr *outbound.Addr) {
	if r.resolver == nil {
		return
	}
	// Check if the host is already an IP address
	if ip := net.ParseIP(addr.Host); ip != nil {
		addr.ResolveInfo = &outbound.ResolveInfo{}
		if ip.To4() != nil {
			addr.ResolveInfo.IPv4 = ip
		} else {
			addr.ResolveInfo.IPv6 = ip
		}
		return
	}
	ipv4, ipv6, err := r.resolver.Resolve(addr.Host)
	addr.ResolveInfo = &outbound.ResolveInfo{
		IPv4: ipv4,
		IPv6: ipv6,
		Err:  err,
	}
}

func (r *Router) match(addr *outbound.Addr, proto acl.Protocol) outbound.Outbound {
	hostInfo := acl.HostInfo{Name: addr.Host}
	if addr.ResolveInfo != nil {
		hostInfo.IPv4 = addr.ResolveInfo.IPv4
		hostInfo.IPv6 = addr.ResolveInfo.IPv6
	}
	ob, hijackIP := r.ruleSet.Match(hostInfo, proto, addr.Port)
	if ob == nil {
		return r.default_
	}
	if hijackIP != nil {
		// Rewrite both Host & ResolveInfo
		addr.Host = hijackIP.String()
		if ip4 := hijackIP.To4(); ip4 != nil {
			addr.ResolveInfo = &outbound.ResolveInfo{IPv4: ip4}
		} else {
			addr.ResolveInfo = &outbound.ResolveInfo{IPv6: hijackIP}
		}
	}
	return ob
}

// DialTCP routes and establishes a TCP connection based on ACL rules.
func (r *Router) DialTCP(addr *outbound.Addr) (net.Conn, error) {
	r.resolve(addr)
	ob := r.match(addr, acl.ProtocolTCP)
	return ob.DialTCP(addr)
}

// DialUDP routes and creates a UDP connection based on ACL rules.
func (r *Router) DialUDP(addr *outbound.Addr) (outbound.UDPConn, error) {
	r.resolve(addr)
	ob := r.match(addr, acl.ProtocolUDP)
	return ob.DialUDP(addr)
}
