package outbound

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	httpRequestTimeout = 10 * time.Second
)

var (
	errHTTPUDPNotSupported   = errors.New("UDP not supported by HTTP proxy")
	errHTTPUnsupportedScheme = errors.New("unsupported scheme for HTTP proxy (use http:// or https://)")
)

type errHTTPRequestFailed struct {
	Status int
}

func (e errHTTPRequestFailed) Error() string {
	return fmt.Sprintf("HTTP request failed: %d", e.Status)
}

// HTTP is an Outbound that connects to the target using
// an HTTP/HTTPS proxy server (that supports the CONNECT method).
// HTTP proxies don't support UDP by design, so this outbound will reject
// any UDP request with errHTTPUDPNotSupported.
// Since HTTP proxies support using either IP or domain name as the target
// address, it will ignore ResolveInfo in Addr and always only use Host.
type HTTP struct {
	Dialer     *net.Dialer
	Addr       string
	HTTPS      bool
	Insecure   bool
	ServerName string
	BasicAuth  string // Base64 encoded
}

// NewHTTP creates a new HTTP outbound from a proxy URL.
// The URL should be in the format of http://[user:pass@]host:port or https://[user:pass@]host:port
func NewHTTP(proxyURL string, insecure bool) (Outbound, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errHTTPUnsupportedScheme
	}
	addr := u.Host
	if u.Port() == "" {
		if u.Scheme == "http" {
			addr = net.JoinHostPort(u.Host, "80")
		} else {
			addr = net.JoinHostPort(u.Host, "443")
		}
	}
	var basicAuth string
	if u.User != nil {
		username := u.User.Username()
		password, _ := u.User.Password()
		basicAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	}
	return &HTTP{
		Dialer:     &net.Dialer{Timeout: defaultDialerTimeout},
		Addr:       addr,
		HTTPS:      u.Scheme == "https",
		Insecure:   insecure,
		ServerName: u.Hostname(),
		BasicAuth:  basicAuth,
	}, nil
}

func (o *HTTP) dial() (net.Conn, error) {
	conn, err := o.Dialer.Dial("tcp", o.Addr)
	if err != nil {
		return nil, err
	}
	if o.HTTPS {
		conn = tls.Client(conn, &tls.Config{
			InsecureSkipVerify: o.Insecure, //nolint:gosec // user configurable
			ServerName:         o.Addr,
		})
	}
	return conn, nil
}

func (o *HTTP) addrToRequest(addr *Addr) (*http.Request, error) {
	req := &http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Host: net.JoinHostPort(addr.Host, strconv.Itoa(int(addr.Port))),
		},
		Header: http.Header{
			"Proxy-Connection": []string{"Keep-Alive"},
		},
	}
	if o.BasicAuth != "" {
		req.Header.Add("Proxy-Authorization", o.BasicAuth)
	}
	return req, nil
}

// DialTCP establishes a TCP connection through the HTTP proxy.
func (o *HTTP) DialTCP(addr *Addr) (net.Conn, error) {
	req, err := o.addrToRequest(addr)
	if err != nil {
		return nil, err
	}
	conn, err := o.dial()
	if err != nil {
		return nil, err
	}
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(httpRequestTimeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	bufReader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(bufReader, req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, errHTTPRequestFailed{resp.StatusCode}
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if bufReader.Buffered() > 0 {
		data := make([]byte, bufReader.Buffered())
		_, err := io.ReadFull(bufReader, data)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		cachedConn := &cachedConn{
			Conn:   conn,
			Buffer: *bytes.NewBuffer(data),
		}
		return cachedConn, nil
	}
	return conn, nil
}

// DialUDP is not supported by HTTP proxy.
func (o *HTTP) DialUDP(addr *Addr) (UDPConn, error) {
	return nil, errHTTPUDPNotSupported
}

// cachedConn is a net.Conn wrapper that first Read()s from a buffer,
// and then from the underlying net.Conn when the buffer is drained.
type cachedConn struct {
	net.Conn
	Buffer bytes.Buffer
}

func (c *cachedConn) Read(b []byte) (int, error) {
	if c.Buffer.Len() > 0 {
		n, err := c.Buffer.Read(b)
		if err == io.EOF {
			err = nil
		}
		return n, err
	}
	return c.Conn.Read(b)
}
