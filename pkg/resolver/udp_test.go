package resolver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUDP(t *testing.T) {
	r := NewUDP("8.8.8.8", 5*time.Second)
	require.NotNil(t, r)

	udp, ok := r.(*UDP)
	assert.True(t, ok)
	assert.Equal(t, "8.8.8.8:53", udp.addr)
}

func TestNewUDPDefaultPort(t *testing.T) {
	r := NewUDP("8.8.8.8", 0)
	udp := r.(*UDP)
	assert.Equal(t, "8.8.8.8:53", udp.addr)

	r2 := NewUDP("8.8.8.8:5353", 0)
	udp2 := r2.(*UDP)
	assert.Equal(t, "8.8.8.8:5353", udp2.addr)
}

func TestNewTCP(t *testing.T) {
	r := NewTCP("8.8.8.8", 5*time.Second)
	require.NotNil(t, r)

	tcp, ok := r.(*TCP)
	assert.True(t, ok)
	assert.Equal(t, "8.8.8.8:53", tcp.addr)
}

func TestNewTLS(t *testing.T) {
	r := NewTLS("dns.google", 5*time.Second, "dns.google", false)
	require.NotNil(t, r)

	tls, ok := r.(*TLS)
	assert.True(t, ok)
	assert.Equal(t, "dns.google:853", tls.addr)
}

func TestNewTLSDefaultPort(t *testing.T) {
	r := NewTLS("dns.google:8853", 0, "dns.google", false)
	tls := r.(*TLS)
	assert.Equal(t, "dns.google:8853", tls.addr)
}

func TestAddDefaultPort(t *testing.T) {
	tests := []struct {
		addr        string
		defaultPort string
		expected    string
	}{
		{"8.8.8.8", "53", "8.8.8.8:53"},
		{"8.8.8.8:5353", "53", "8.8.8.8:5353"},
		{"dns.google", "853", "dns.google:853"},
		{"::1", "53", "[::1]:53"},
		{"[::1]:5353", "53", "[::1]:5353"},
	}

	for _, tt := range tests {
		result := addDefaultPort(tt.addr, tt.defaultPort)
		assert.Equal(t, tt.expected, result)
	}
}

func TestTimeoutOrDefault(t *testing.T) {
	assert.Equal(t, defaultTimeout, timeoutOrDefault(0))
	assert.Equal(t, 5*time.Second, timeoutOrDefault(5*time.Second))
}

func TestUDPResolve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	r := NewUDP("8.8.8.8", 5*time.Second)

	t.Run("resolve google.com", func(t *testing.T) {
		ipv4, ipv6, err := r.Resolve("google.com")
		if err != nil {
			// Skip if network is unreachable (e.g., VPN, firewall)
			t.Skipf("skipping due to network error: %v", err)
		}
		// Google should have at least IPv4
		assert.NotNil(t, ipv4, "google.com should have IPv4")
		// IPv6 may or may not be present
		_ = ipv6
	})

	t.Run("resolve invalid domain", func(t *testing.T) {
		_, _, err := r.Resolve("this-domain-should-not-exist-12345.invalid")
		// This might not error immediately, but should return nil IPs
		if err == nil {
			// If no error, IPs should be nil
			t.Log("No error returned for invalid domain")
		}
	})
}

func TestTCPResolve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	r := NewTCP("8.8.8.8", 5*time.Second)

	t.Run("resolve google.com", func(t *testing.T) {
		ipv4, ipv6, err := r.Resolve("google.com")
		require.NoError(t, err)
		assert.NotNil(t, ipv4, "google.com should have IPv4")
		_ = ipv6
	})
}
