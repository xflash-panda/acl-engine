package resolver

import "net"

// Resolver defines the interface for DNS resolution.
type Resolver interface {
	// Resolve resolves the hostname to IPv4 and IPv6 addresses.
	// Either or both of the returned IPs can be nil if no address is found.
	// An error is returned if the resolution fails completely.
	Resolve(host string) (ipv4, ipv6 net.IP, err error)
}

// splitIPv4IPv6 gets the first IPv4 and IPv6 address from a list of IP addresses.
func splitIPv4IPv6(ips []net.IP) (ipv4, ipv6 net.IP) {
	for _, ip := range ips {
		if ip.To4() != nil {
			if ipv4 == nil {
				ipv4 = ip
			}
		} else {
			if ipv6 == nil {
				ipv6 = ip
			}
		}
		if ipv4 != nil && ipv6 != nil {
			break
		}
	}
	return ipv4, ipv6
}
