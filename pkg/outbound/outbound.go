package outbound

import (
	"net"
	"strconv"
)

// Outbound defines the interface for outbound connections.
type Outbound interface {
	DialTCP(addr *Addr) (net.Conn, error)
	DialUDP(addr *Addr) (UDPConn, error)
}

// UDPConn defines the interface for UDP connections.
type UDPConn interface {
	ReadFrom(b []byte) (int, *Addr, error)
	WriteTo(b []byte, addr *Addr) (int, error)
	Close() error
}

// Addr represents a network address with optional DNS resolution info.
type Addr struct {
	Host        string       // Hostname or IP address
	Port        uint16       // Port number
	ResolveInfo *ResolveInfo // Optional DNS resolution result
}

// String returns the address in host:port format.
func (a *Addr) String() string {
	return net.JoinHostPort(a.Host, strconv.Itoa(int(a.Port)))
}

// NetworkAddr returns the resolved IP address for dialing.
// If ResolveInfo contains an IPv4 address, it returns that.
// Otherwise, if it contains an IPv6 address, it returns that.
// If no resolved address is available, it falls back to Host.
func (a *Addr) NetworkAddr() string {
	if a.ResolveInfo != nil {
		if a.ResolveInfo.IPv4 != nil {
			return net.JoinHostPort(a.ResolveInfo.IPv4.String(), strconv.Itoa(int(a.Port)))
		}
		if a.ResolveInfo.IPv6 != nil {
			return net.JoinHostPort(a.ResolveInfo.IPv6.String(), strconv.Itoa(int(a.Port)))
		}
	}
	return a.String()
}

// ResolveInfo contains DNS resolution results.
type ResolveInfo struct {
	IPv4 net.IP // Resolved IPv4 address, if any
	IPv6 net.IP // Resolved IPv6 address, if any
	Err  error  // Error that occurred during resolution, if any
}
