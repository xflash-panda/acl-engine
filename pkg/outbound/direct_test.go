package outbound

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDirect(t *testing.T) {
	modes := []DirectMode{
		DirectModeAuto,
		DirectMode64,
		DirectMode46,
		DirectMode6,
		DirectMode4,
	}

	for _, mode := range modes {
		ob := NewDirect(mode)
		require.NotNil(t, ob)
		d, ok := ob.(*Direct)
		require.True(t, ok)
		assert.Equal(t, mode, d.Mode)
		assert.NotNil(t, d.DialFunc4)
		assert.NotNil(t, d.DialFunc6)
	}
}

func TestNewDirectWithOptions(t *testing.T) {
	t.Run("basic options", func(t *testing.T) {
		ob, err := NewDirectWithOptions(DirectOptions{
			Mode: DirectModeAuto,
		})
		require.NoError(t, err)
		require.NotNil(t, ob)
	})

	t.Run("with bind ipv4", func(t *testing.T) {
		ob, err := NewDirectWithOptions(DirectOptions{
			Mode:    DirectModeAuto,
			BindIP4: net.ParseIP("127.0.0.1"),
		})
		require.NoError(t, err)
		require.NotNil(t, ob)
	})

	t.Run("with bind ipv6", func(t *testing.T) {
		ob, err := NewDirectWithOptions(DirectOptions{
			Mode:    DirectModeAuto,
			BindIP6: net.ParseIP("::1"),
		})
		require.NoError(t, err)
		require.NotNil(t, ob)
	})

	t.Run("invalid bind ipv4", func(t *testing.T) {
		_, err := NewDirectWithOptions(DirectOptions{
			Mode:    DirectModeAuto,
			BindIP4: net.ParseIP("::1"), // IPv6 address for IPv4 field
		})
		require.Error(t, err)
	})

	t.Run("invalid bind ipv6", func(t *testing.T) {
		_, err := NewDirectWithOptions(DirectOptions{
			Mode:    DirectModeAuto,
			BindIP6: net.ParseIP("127.0.0.1"), // IPv4 address for IPv6 field
		})
		require.Error(t, err)
	})
}

func TestNewDirectBindToIPs(t *testing.T) {
	t.Run("valid ips", func(t *testing.T) {
		ob, err := NewDirectBindToIPs(DirectModeAuto, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))
		require.NoError(t, err)
		require.NotNil(t, ob)
	})

	t.Run("nil ips", func(t *testing.T) {
		ob, err := NewDirectBindToIPs(DirectModeAuto, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, ob)
	})
}

func TestDirectDialTCP(t *testing.T) {
	// Create a local TCP server for testing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			_ = conn.Close()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	t.Run("dial with resolved ip", func(t *testing.T) {
		ob := NewDirect(DirectModeAuto)
		conn, err := ob.DialTCP(&Addr{
			Host: addr.IP.String(),
			Port: uint16(addr.Port), //nolint:gosec // test code
			ResolveInfo: &ResolveInfo{
				IPv4: addr.IP,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, conn)
		_ = conn.Close()
	})

	t.Run("dial without resolve info", func(t *testing.T) {
		ob := NewDirect(DirectModeAuto)
		conn, err := ob.DialTCP(&Addr{
			Host: "127.0.0.1",
			Port: uint16(addr.Port), //nolint:gosec // test code
		})
		require.NoError(t, err)
		require.NotNil(t, conn)
		_ = conn.Close()
	})
}

func TestDirectDialUDP(t *testing.T) {
	t.Run("create udp conn", func(t *testing.T) {
		ob := NewDirect(DirectModeAuto)
		conn, err := ob.DialUDP(&Addr{
			Host: "127.0.0.1",
			Port: 53,
		})
		require.NoError(t, err)
		require.NotNil(t, conn)
		_ = conn.Close()
	})
}

func TestDirectModes(t *testing.T) {
	// Create local servers
	listener4, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener4.Close() }()

	go func() {
		for {
			conn, err := listener4.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	addr4 := listener4.Addr().(*net.TCPAddr)

	t.Run("mode 4 with ipv4", func(t *testing.T) {
		ob := NewDirect(DirectMode4)
		conn, err := ob.DialTCP(&Addr{
			Host: addr4.IP.String(),
			Port: uint16(addr4.Port), //nolint:gosec // test code
			ResolveInfo: &ResolveInfo{
				IPv4: addr4.IP,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, conn)
		_ = conn.Close()
	})

	t.Run("mode 4 without ipv4", func(t *testing.T) {
		ob := NewDirect(DirectMode4)
		_, err := ob.DialTCP(&Addr{
			Host: "example.com",
			Port: 443,
			ResolveInfo: &ResolveInfo{
				IPv6: net.ParseIP("2001:db8::1"),
			},
		})
		require.Error(t, err)
	})

	t.Run("mode 46 prefers ipv4", func(t *testing.T) {
		ob := NewDirect(DirectMode46)
		conn, err := ob.DialTCP(&Addr{
			Host: addr4.IP.String(),
			Port: uint16(addr4.Port), //nolint:gosec // test code
			ResolveInfo: &ResolveInfo{
				IPv4: addr4.IP,
				IPv6: net.ParseIP("2001:db8::1"),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, conn)
		_ = conn.Close()
	})
}

func TestErrorTypes(t *testing.T) {
	t.Run("noAddressError", func(t *testing.T) {
		err := noAddressError{IPv4: true, IPv6: true}
		assert.Contains(t, err.Error(), "no IPv4 or IPv6")

		err = noAddressError{IPv4: true}
		assert.Contains(t, err.Error(), "no IPv4")

		err = noAddressError{IPv6: true}
		assert.Contains(t, err.Error(), "no IPv6")

		err = noAddressError{}
		assert.Contains(t, err.Error(), "no address")
	})

	t.Run("invalidModeError", func(t *testing.T) {
		err := invalidModeError{}
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("resolveError", func(t *testing.T) {
		err := resolveError{Err: nil}
		assert.Contains(t, err.Error(), "resolve error")

		innerErr := net.UnknownNetworkError("test")
		err = resolveError{Err: innerErr}
		assert.Contains(t, err.Error(), "test")
		assert.Equal(t, innerErr, err.Unwrap())
	})
}
