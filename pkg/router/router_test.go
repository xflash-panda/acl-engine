package router

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xflash-panda/acl-engine/pkg/acl"
	"github.com/xflash-panda/acl-engine/pkg/outbound"
)

// mockResolver is a simple resolver for testing
type mockResolver struct {
	ipv4 net.IP
	ipv6 net.IP
	err  error
}

func (m *mockResolver) Resolve(host string) (net.IP, net.IP, error) {
	return m.ipv4, m.ipv6, m.err
}

func TestNew(t *testing.T) {
	rules := `
direct(192.168.0.0/16)
reject(10.0.0.0/8)
direct(all)
`
	outbounds := []OutboundEntry{}
	geoLoader := &acl.NilGeoLoader{}

	r, err := New(rules, outbounds, geoLoader)
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestNewWithOptions(t *testing.T) {
	rules := `direct(all)`
	outbounds := []OutboundEntry{}
	geoLoader := &acl.NilGeoLoader{}

	t.Run("with cache size", func(t *testing.T) {
		r, err := New(rules, outbounds, geoLoader, WithCacheSize(2048))
		require.NoError(t, err)
		require.NotNil(t, r)
	})

	t.Run("with resolver", func(t *testing.T) {
		resolver := &mockResolver{ipv4: net.ParseIP("1.2.3.4")}
		r, err := New(rules, outbounds, geoLoader, WithResolver(resolver))
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.NotNil(t, r.resolver)
	})
}

func TestNewWithCustomOutbounds(t *testing.T) {
	rules := `
proxy(*.example.com)
direct(all)
`
	// Create a custom outbound
	proxyOb := outbound.NewReject() // Use reject as mock proxy

	outbounds := []OutboundEntry{
		{"proxy", proxyOb},
	}
	geoLoader := &acl.NilGeoLoader{}

	r, err := New(rules, outbounds, geoLoader)
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestNewInvalidRules(t *testing.T) {
	t.Run("invalid syntax", func(t *testing.T) {
		rules := `invalid syntax here`
		_, err := New(rules, nil, &acl.NilGeoLoader{})
		require.Error(t, err)
	})

	t.Run("unknown outbound", func(t *testing.T) {
		rules := `unknown_outbound(all)`
		_, err := New(rules, nil, &acl.NilGeoLoader{})
		require.Error(t, err)
	})
}

func TestBuiltInOutbounds(t *testing.T) {
	rules := `
direct(1.1.1.1)
reject(2.2.2.2)
direct(all)
`
	// No custom outbounds, should use built-in direct and reject
	r, err := New(rules, nil, &acl.NilGeoLoader{})
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestOutboundsToMap(t *testing.T) {
	t.Run("empty outbounds", func(t *testing.T) {
		m := outboundsToMap(nil)
		assert.NotNil(t, m["direct"])
		assert.NotNil(t, m["reject"])
		assert.NotNil(t, m["default"])
		// default should equal direct when no outbounds provided
		assert.Equal(t, m["direct"], m["default"])
	})

	t.Run("with custom outbounds", func(t *testing.T) {
		proxy := outbound.NewReject()
		outbounds := []OutboundEntry{
			{"proxy", proxy},
		}
		m := outboundsToMap(outbounds)
		assert.Equal(t, proxy, m["proxy"])
		// default should be first outbound
		assert.Equal(t, proxy, m["default"])
	})

	t.Run("override built-in", func(t *testing.T) {
		customDirect := outbound.NewReject() // use reject as custom direct
		outbounds := []OutboundEntry{
			{"direct", customDirect},
		}
		m := outboundsToMap(outbounds)
		assert.Equal(t, customDirect, m["direct"])
	})

	t.Run("case insensitive", func(t *testing.T) {
		proxy := outbound.NewReject()
		outbounds := []OutboundEntry{
			{"PROXY", proxy},
		}
		m := outboundsToMap(outbounds)
		assert.Equal(t, proxy, m["proxy"])
	})
}

func TestRouterDialTCP(t *testing.T) {
	// Create a local TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	rules := `direct(all)`
	r, err := New(rules, nil, &acl.NilGeoLoader{})
	require.NoError(t, err)

	conn, err := r.DialTCP(&outbound.Addr{
		Host: addr.IP.String(),
		Port: uint16(addr.Port), //nolint:gosec // test code
	})
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

func TestRouterDialTCPWithResolver(t *testing.T) {
	// Create a local TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	rules := `direct(all)`
	resolver := &mockResolver{ipv4: addr.IP}

	r, err := New(rules, nil, &acl.NilGeoLoader{}, WithResolver(resolver))
	require.NoError(t, err)

	conn, err := r.DialTCP(&outbound.Addr{
		Host: "test.local",
		Port: uint16(addr.Port), //nolint:gosec // test code
	})
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

func TestRouterDialUDP(t *testing.T) {
	rules := `direct(all)`
	r, err := New(rules, nil, &acl.NilGeoLoader{})
	require.NoError(t, err)

	conn, err := r.DialUDP(&outbound.Addr{
		Host: "127.0.0.1",
		Port: 53,
	})
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

func TestRouterReject(t *testing.T) {
	rules := `reject(all)`
	r, err := New(rules, nil, &acl.NilGeoLoader{})
	require.NoError(t, err)

	_, err = r.DialTCP(&outbound.Addr{
		Host: "example.com",
		Port: 443,
	})
	require.Error(t, err)
}

func TestRouterIPMatching(t *testing.T) {
	rules := `
reject(192.168.0.0/16)
direct(all)
`
	// Use a resolver that returns the IP
	resolver := &mockResolver{ipv4: net.ParseIP("192.168.1.1")}
	r, err := New(rules, nil, &acl.NilGeoLoader{}, WithResolver(resolver))
	require.NoError(t, err)

	// Should be rejected because 192.168.1.1 matches 192.168.0.0/16
	_, err = r.DialTCP(&outbound.Addr{
		Host: "test.local",
		Port: 80,
	})
	require.Error(t, err)
}

func TestRouterResolveWithIPHost(t *testing.T) {
	rules := `direct(all)`
	resolver := &mockResolver{ipv4: net.ParseIP("1.2.3.4")}

	r, err := New(rules, nil, &acl.NilGeoLoader{}, WithResolver(resolver))
	require.NoError(t, err)

	// When host is already an IP, should not call resolver
	addr := &outbound.Addr{Host: "192.168.1.1", Port: 80}
	r.resolve(addr)

	assert.NotNil(t, addr.ResolveInfo)
	assert.NotNil(t, addr.ResolveInfo.IPv4)
	// Should parse the IP directly, not use resolver's IP
	assert.True(t, net.ParseIP("192.168.1.1").Equal(addr.ResolveInfo.IPv4))
}

func TestRouterResolveIPv6(t *testing.T) {
	rules := `direct(all)`
	resolver := &mockResolver{}

	r, err := New(rules, nil, &acl.NilGeoLoader{}, WithResolver(resolver))
	require.NoError(t, err)

	addr := &outbound.Addr{Host: "2001:db8::1", Port: 80}
	r.resolve(addr)

	assert.NotNil(t, addr.ResolveInfo)
	assert.NotNil(t, addr.ResolveInfo.IPv6)
}
