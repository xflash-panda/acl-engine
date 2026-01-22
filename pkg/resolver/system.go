package resolver

import "net"

// System is a Resolver that uses the system's default DNS resolver.
type System struct{}

// NewSystem creates a new System resolver.
func NewSystem() Resolver {
	return &System{}
}

// Resolve resolves the hostname using the system's default DNS resolver.
func (r *System) Resolve(host string) (ipv4, ipv6 net.IP, err error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, nil, err
	}
	ipv4, ipv6 = splitIPv4IPv6(ips)
	return ipv4, ipv6, nil
}
