package resolver

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
