package resolver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPS(t *testing.T) {
	t.Run("with full url", func(t *testing.T) {
		r := NewHTTPS("https://dns.google/dns-query", 5*time.Second, "dns.google", false)
		require.NotNil(t, r)

		https, ok := r.(*HTTPS)
		assert.True(t, ok)
		assert.Equal(t, "https://dns.google/dns-query", https.url)
	})

	t.Run("with hostname only", func(t *testing.T) {
		r := NewHTTPS("dns.google", 5*time.Second, "dns.google", false)
		require.NotNil(t, r)

		https, ok := r.(*HTTPS)
		assert.True(t, ok)
		assert.Equal(t, "https://dns.google/dns-query", https.url)
	})

	t.Run("with ip address", func(t *testing.T) {
		r := NewHTTPS("8.8.8.8", 5*time.Second, "dns.google", false)
		require.NotNil(t, r)

		https, ok := r.(*HTTPS)
		assert.True(t, ok)
		assert.Equal(t, "https://8.8.8.8/dns-query", https.url)
	})
}

func TestHTTPSResolve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	r := NewHTTPS("https://dns.google/dns-query", 10*time.Second, "dns.google", false)

	t.Run("resolve google.com", func(t *testing.T) {
		ipv4, ipv6, err := r.Resolve("google.com")
		require.NoError(t, err)
		assert.NotNil(t, ipv4, "google.com should have IPv4")
		_ = ipv6
	})
}
