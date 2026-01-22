package outbound

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSOCKS5(t *testing.T) {
	t.Run("without auth", func(t *testing.T) {
		ob := NewSOCKS5("127.0.0.1:1080", "", "")
		require.NotNil(t, ob)

		s := ob.(*SOCKS5)
		assert.Equal(t, "127.0.0.1:1080", s.Addr)
		assert.Empty(t, s.Username)
		assert.Empty(t, s.Password)
	})

	t.Run("with auth", func(t *testing.T) {
		ob := NewSOCKS5("127.0.0.1:1080", "user", "pass")
		require.NotNil(t, ob)

		s := ob.(*SOCKS5)
		assert.Equal(t, "user", s.Username)
		assert.Equal(t, "pass", s.Password)
	})
}

func TestSOCKS5Errors(t *testing.T) {
	t.Run("auth failed", func(t *testing.T) {
		assert.Contains(t, errSOCKS5AuthFailed.Error(), "authentication failed")
	})

	t.Run("unsupported auth method", func(t *testing.T) {
		err := errSOCKS5UnsupportedAuthMethod{Method: 0x05}
		assert.Contains(t, err.Error(), "unsupported")
		assert.Contains(t, err.Error(), "5")
	})

	t.Run("request failed codes", func(t *testing.T) {
		codes := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0xFF}
		for _, code := range codes {
			err := errSOCKS5RequestFailed{Rep: code}
			assert.NotEmpty(t, err.Error())
		}
	})
}

func TestAddrToSOCKS5Addr(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		addr := &Addr{Host: "192.168.1.1", Port: 80}
		atyp, dstAddr, dstPort := addrToSOCKS5Addr(addr)
		assert.Equal(t, byte(0x01), atyp) // ATYPIPv4
		assert.Equal(t, net.ParseIP("192.168.1.1").To4(), net.IP(dstAddr))
		assert.Equal(t, uint16(80), uint16(dstPort[0])<<8|uint16(dstPort[1]))
	})

	t.Run("ipv6", func(t *testing.T) {
		addr := &Addr{Host: "2001:db8::1", Port: 443}
		atyp, dstAddr, dstPort := addrToSOCKS5Addr(addr)
		assert.Equal(t, byte(0x04), atyp) // ATYPIPv6
		assert.Equal(t, net.ParseIP("2001:db8::1").To16(), net.IP(dstAddr))
		assert.Equal(t, uint16(443), uint16(dstPort[0])<<8|uint16(dstPort[1]))
	})

	t.Run("domain", func(t *testing.T) {
		addr := &Addr{Host: "example.com", Port: 8080}
		atyp, dstAddr, dstPort := addrToSOCKS5Addr(addr)
		assert.Equal(t, byte(0x03), atyp) // ATYPDomain
		assert.Equal(t, []byte("example.com"), dstAddr)
		assert.Equal(t, uint16(8080), uint16(dstPort[0])<<8|uint16(dstPort[1]))
	})
}

func TestSocks5AddrToAddr(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		dstAddr := net.ParseIP("192.168.1.1").To4()
		dstPort := []byte{0x00, 0x50} // port 80
		addr := socks5AddrToAddr(0x01, dstAddr, dstPort)
		assert.Equal(t, "192.168.1.1", addr.Host)
		assert.Equal(t, uint16(80), addr.Port)
	})

	t.Run("ipv6", func(t *testing.T) {
		dstAddr := net.ParseIP("2001:db8::1").To16()
		dstPort := []byte{0x01, 0xBB} // port 443
		addr := socks5AddrToAddr(0x04, dstAddr, dstPort)
		assert.Equal(t, "2001:db8::1", addr.Host)
		assert.Equal(t, uint16(443), addr.Port)
	})

	t.Run("domain", func(t *testing.T) {
		// Domain format: first byte is length, followed by domain
		dstAddr := append([]byte{byte(len("example.com"))}, []byte("example.com")...)
		dstPort := []byte{0x1F, 0x90} // port 8080
		addr := socks5AddrToAddr(0x03, dstAddr, dstPort)
		assert.Equal(t, "example.com", addr.Host)
		assert.Equal(t, uint16(8080), addr.Port)
	})
}
