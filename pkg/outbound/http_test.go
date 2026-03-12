package outbound

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTP(t *testing.T) {
	t.Run("http url", func(t *testing.T) {
		ob, err := NewHTTP("http://proxy.example.com:8080", false)
		require.NoError(t, err)
		require.NotNil(t, ob)

		h := ob.(*HTTP)
		assert.Equal(t, "proxy.example.com:8080", h.Addr)
		assert.False(t, h.HTTPS)
	})

	t.Run("http url without port", func(t *testing.T) {
		ob, err := NewHTTP("http://proxy.example.com", false)
		require.NoError(t, err)
		require.NotNil(t, ob)

		h := ob.(*HTTP)
		assert.Equal(t, "proxy.example.com:80", h.Addr)
	})

	t.Run("https url", func(t *testing.T) {
		ob, err := NewHTTP("https://proxy.example.com:8443", false)
		require.NoError(t, err)
		require.NotNil(t, ob)

		h := ob.(*HTTP)
		assert.Equal(t, "proxy.example.com:8443", h.Addr)
		assert.True(t, h.HTTPS)
	})

	t.Run("https url without port", func(t *testing.T) {
		ob, err := NewHTTP("https://proxy.example.com", false)
		require.NoError(t, err)
		require.NotNil(t, ob)

		h := ob.(*HTTP)
		assert.Equal(t, "proxy.example.com:443", h.Addr)
	})

	t.Run("with auth", func(t *testing.T) {
		ob, err := NewHTTP("http://user:pass@proxy.example.com:8080", false)
		require.NoError(t, err)
		require.NotNil(t, ob)

		h := ob.(*HTTP)
		assert.Contains(t, h.BasicAuth, "Basic ")
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		_, err := NewHTTP("socks5://proxy.example.com:1080", false)
		require.Error(t, err)
		assert.Equal(t, errHTTPUnsupportedScheme, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		_, err := NewHTTP("://invalid", false)
		require.Error(t, err)
	})
}

func TestHTTP_DialUDP(t *testing.T) {
	ob, err := NewHTTP("http://proxy.example.com:8080", false)
	require.NoError(t, err)

	conn, err := ob.DialUDP(&Addr{Host: "example.com", Port: 53})
	assert.Nil(t, conn)
	assert.Error(t, err)
	assert.Equal(t, errHTTPUDPNotSupported, err)
}

func TestHTTP_DialTCP(t *testing.T) {
	// Create a mock HTTP proxy server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		// Read the CONNECT request
		reader := bufio.NewReader(conn)
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}

		// Send success response
		if req.Method == http.MethodConnect {
			_, _ = conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		}
	}()

	addr := listener.Addr().String()
	ob, err := NewHTTP("http://"+addr, false)
	require.NoError(t, err)

	conn, err := ob.DialTCP(&Addr{Host: "example.com", Port: 443})
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()
}

func TestHTTP_TLSServerName(t *testing.T) {
	// Bug: TLS ServerName was set to o.Addr (host:port) instead of o.ServerName (host only).
	// This causes TLS handshake failures for HTTPS proxies because SNI includes the port.
	t.Run("ServerName should not contain port", func(t *testing.T) {
		ob, err := NewHTTP("https://proxy.example.com:8443", false)
		require.NoError(t, err)

		h := ob.(*HTTP)
		// ServerName must be hostname only, without port
		assert.Equal(t, "proxy.example.com", h.ServerName)

		// Verify that dial() actually uses ServerName (not Addr) for TLS
		// by capturing the SNI from a real TLS handshake.
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		sniCh := make(chan string, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer func() { _ = conn.Close() }()
			// Use tls.Server with GetConfigForClient to capture SNI
			tlsConn := tls.Server(conn, &tls.Config{ //nolint:gosec // test code, TLS version irrelevant
				GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
					sniCh <- hello.ServerName
					return nil, nil
				},
			})
			// Trigger handshake (will fail, but we only need the ClientHello)
			_ = tlsConn.Handshake()
		}()

		testH := &HTTP{
			Dialer:     &net.Dialer{Timeout: defaultDialerTimeout},
			Addr:       listener.Addr().String(),
			HTTPS:      true,
			Insecure:   true,
			ServerName: "proxy.example.com",
		}
		conn, _ := testH.dial()
		if conn != nil {
			// Trigger TLS handshake
			_ = conn.(*tls.Conn).Handshake()
			_ = conn.Close()
		}

		sni := <-sniCh
		assert.Equal(t, "proxy.example.com", sni, "TLS SNI should be hostname without port")
	})
}

func TestHTTPRequestFailed(t *testing.T) {
	err := errHTTPRequestFailed{Status: 403}
	assert.Contains(t, err.Error(), "403")
}

func TestCachedConn(t *testing.T) {
	// Create a mock connection
	server, client := net.Pipe()
	defer func() { _ = server.Close() }()
	defer func() { _ = client.Close() }()

	// Write some data to server side
	go func() {
		_, _ = server.Write([]byte("hello from server"))
		_ = server.Close()
	}()

	// Create cached conn with buffered data
	cached := &cachedConn{
		Conn:   client,
		Buffer: *bytes.NewBuffer([]byte("buffered data")),
	}

	// First read should come from buffer
	buf := make([]byte, 100)
	n, err := cached.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "buffered data", string(buf[:n]))

	// Second read should come from underlying connection
	n, err = cached.Read(buf)
	if err != nil && err != io.EOF {
		require.NoError(t, err)
	}
	if n > 0 {
		assert.Equal(t, "hello from server", string(buf[:n]))
	}
}
