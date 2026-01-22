package outbound

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddr_String(t *testing.T) {
	tests := []struct {
		name     string
		addr     *Addr
		expected string
	}{
		{
			name:     "domain with port",
			addr:     &Addr{Host: "example.com", Port: 443},
			expected: "example.com:443",
		},
		{
			name:     "ipv4 with port",
			addr:     &Addr{Host: "192.168.1.1", Port: 80},
			expected: "192.168.1.1:80",
		},
		{
			name:     "ipv6 with port",
			addr:     &Addr{Host: "::1", Port: 8080},
			expected: "[::1]:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.addr.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddr_NetworkAddr(t *testing.T) {
	tests := []struct {
		name     string
		addr     *Addr
		expected string
	}{
		{
			name:     "no resolve info",
			addr:     &Addr{Host: "example.com", Port: 443},
			expected: "example.com:443",
		},
		{
			name: "with ipv4 resolve info",
			addr: &Addr{
				Host: "example.com",
				Port: 443,
				ResolveInfo: &ResolveInfo{
					IPv4: net.ParseIP("1.2.3.4"),
				},
			},
			expected: "1.2.3.4:443",
		},
		{
			name: "with ipv6 resolve info",
			addr: &Addr{
				Host: "example.com",
				Port: 443,
				ResolveInfo: &ResolveInfo{
					IPv6: net.ParseIP("2001:db8::1"),
				},
			},
			expected: "[2001:db8::1]:443",
		},
		{
			name: "with both ipv4 and ipv6 prefers ipv4",
			addr: &Addr{
				Host: "example.com",
				Port: 443,
				ResolveInfo: &ResolveInfo{
					IPv4: net.ParseIP("1.2.3.4"),
					IPv6: net.ParseIP("2001:db8::1"),
				},
			},
			expected: "1.2.3.4:443",
		},
		{
			name: "empty resolve info",
			addr: &Addr{
				Host:        "example.com",
				Port:        443,
				ResolveInfo: &ResolveInfo{},
			},
			expected: "example.com:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.addr.NetworkAddr()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitIPv4IPv6(t *testing.T) {
	tests := []struct {
		name         string
		ips          []net.IP
		expectedIPv4 net.IP
		expectedIPv6 net.IP
	}{
		{
			name:         "empty list",
			ips:          nil,
			expectedIPv4: nil,
			expectedIPv6: nil,
		},
		{
			name:         "only ipv4",
			ips:          []net.IP{net.ParseIP("1.2.3.4")},
			expectedIPv4: net.ParseIP("1.2.3.4"),
			expectedIPv6: nil,
		},
		{
			name:         "only ipv6",
			ips:          []net.IP{net.ParseIP("2001:db8::1")},
			expectedIPv4: nil,
			expectedIPv6: net.ParseIP("2001:db8::1"),
		},
		{
			name:         "both ipv4 and ipv6",
			ips:          []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("2001:db8::1")},
			expectedIPv4: net.ParseIP("1.2.3.4"),
			expectedIPv6: net.ParseIP("2001:db8::1"),
		},
		{
			name:         "multiple ipv4 returns first",
			ips:          []net.IP{net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8")},
			expectedIPv4: net.ParseIP("1.2.3.4"),
			expectedIPv6: nil,
		},
		{
			name:         "ipv6 before ipv4",
			ips:          []net.IP{net.ParseIP("2001:db8::1"), net.ParseIP("1.2.3.4")},
			expectedIPv4: net.ParseIP("1.2.3.4"),
			expectedIPv6: net.ParseIP("2001:db8::1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv4, ipv6 := splitIPv4IPv6(tt.ips)
			if tt.expectedIPv4 != nil {
				assert.True(t, tt.expectedIPv4.Equal(ipv4), "IPv4 mismatch")
			} else {
				assert.Nil(t, ipv4)
			}
			if tt.expectedIPv6 != nil {
				assert.True(t, tt.expectedIPv6.Equal(ipv6), "IPv6 mismatch")
			} else {
				assert.Nil(t, ipv6)
			}
		})
	}
}
