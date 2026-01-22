package resolver

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultTimeout     = 2 * time.Second
	defaultRetryTimes  = 2
	defaultDNSPort     = "53"
	defaultDNSTLSPort  = "853"
)

// UDP is a Resolver that uses a UDP DNS server.
type UDP struct {
	addr       string
	client     *dns.Client
	retryTimes int
}

// NewUDP creates a new UDP DNS resolver.
func NewUDP(addr string, timeout time.Duration) Resolver {
	return &UDP{
		addr: addDefaultPort(addr, defaultDNSPort),
		client: &dns.Client{
			Timeout: timeoutOrDefault(timeout),
		},
		retryTimes: defaultRetryTimes,
	}
}

// TCP is a Resolver that uses a TCP DNS server.
type TCP struct {
	addr       string
	client     *dns.Client
	retryTimes int
}

// NewTCP creates a new TCP DNS resolver.
func NewTCP(addr string, timeout time.Duration) Resolver {
	return &TCP{
		addr: addDefaultPort(addr, defaultDNSPort),
		client: &dns.Client{
			Net:     "tcp",
			Timeout: timeoutOrDefault(timeout),
		},
		retryTimes: defaultRetryTimes,
	}
}

// TLS is a Resolver that uses a DNS-over-TLS server.
type TLS struct {
	addr       string
	client     *dns.Client
	retryTimes int
}

// NewTLS creates a new DNS-over-TLS resolver.
func NewTLS(addr string, timeout time.Duration, sni string, insecure bool) Resolver {
	return &TLS{
		addr: addDefaultPort(addr, defaultDNSTLSPort),
		client: &dns.Client{
			Net:     "tcp-tls",
			Timeout: timeoutOrDefault(timeout),
			TLSConfig: &tls.Config{
				ServerName:         sni,
				InsecureSkipVerify: insecure, //nolint:gosec // user configurable
			},
		},
		retryTimes: defaultRetryTimes,
	}
}

func addDefaultPort(addr, defaultPort string) string {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return net.JoinHostPort(addr, defaultPort)
	}
	return addr
}

func timeoutOrDefault(timeout time.Duration) time.Duration {
	if timeout == 0 {
		return defaultTimeout
	}
	return timeout
}

// skipCNAMEChain skips the CNAME chain and returns the last CNAME target.
func skipCNAMEChain(answers []dns.RR) string {
	var lastCNAME string
	for _, a := range answers {
		if cname, ok := a.(*dns.CNAME); ok {
			if lastCNAME == "" {
				lastCNAME = cname.Target
			} else if cname.Hdr.Name == lastCNAME {
				lastCNAME = cname.Target
			} else {
				return lastCNAME
			}
		}
	}
	return lastCNAME
}

type dnsResolver interface {
	exchange(m *dns.Msg) (*dns.Msg, error)
	getRetryTimes() int
}

func (r *UDP) exchange(m *dns.Msg) (*dns.Msg, error) {
	resp, _, err := r.client.Exchange(m, r.addr)
	return resp, err
}

func (r *UDP) getRetryTimes() int {
	return r.retryTimes
}

func (r *TCP) exchange(m *dns.Msg) (*dns.Msg, error) {
	resp, _, err := r.client.Exchange(m, r.addr)
	return resp, err
}

func (r *TCP) getRetryTimes() int {
	return r.retryTimes
}

func (r *TLS) exchange(m *dns.Msg) (*dns.Msg, error) {
	resp, _, err := r.client.Exchange(m, r.addr)
	return resp, err
}

func (r *TLS) getRetryTimes() int {
	return r.retryTimes
}

func lookup4(resolver dnsResolver, host string) (net.IP, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)
	m.RecursionDesired = true
	resp, err := resolver.exchange(m)
	if err != nil {
		return nil, err
	}
	if len(resp.Answer) == 0 {
		return nil, nil
	}
	hasCNAME := false
	for _, a := range resp.Answer {
		if aa, ok := a.(*dns.A); ok {
			return aa.A.To4(), nil
		} else if _, ok := a.(*dns.CNAME); ok {
			hasCNAME = true
		}
	}
	if hasCNAME {
		return lookup4(resolver, skipCNAMEChain(resp.Answer))
	}
	return nil, nil
}

func lookup6(resolver dnsResolver, host string) (net.IP, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeAAAA)
	m.RecursionDesired = true
	resp, err := resolver.exchange(m)
	if err != nil {
		return nil, err
	}
	if len(resp.Answer) == 0 {
		return nil, nil
	}
	hasCNAME := false
	for _, a := range resp.Answer {
		if aa, ok := a.(*dns.AAAA); ok {
			return aa.AAAA.To16(), nil
		} else if _, ok := a.(*dns.CNAME); ok {
			hasCNAME = true
		}
	}
	if hasCNAME {
		return lookup6(resolver, skipCNAMEChain(resp.Answer))
	}
	return nil, nil
}

func resolve(resolver dnsResolver, host string) (ipv4, ipv6 net.IP, err error) {
	type lookupResult struct {
		ip  net.IP
		err error
	}
	ch4, ch6 := make(chan lookupResult, 1), make(chan lookupResult, 1)
	go func() {
		var ip net.IP
		var lookupErr error
		for i := 0; i < resolver.getRetryTimes(); i++ {
			ip, lookupErr = lookup4(resolver, host)
			if lookupErr == nil {
				break
			}
		}
		ch4 <- lookupResult{ip, lookupErr}
	}()
	go func() {
		var ip net.IP
		var lookupErr error
		for i := 0; i < resolver.getRetryTimes(); i++ {
			ip, lookupErr = lookup6(resolver, host)
			if lookupErr == nil {
				break
			}
		}
		ch6 <- lookupResult{ip, lookupErr}
	}()
	result4, result6 := <-ch4, <-ch6
	ipv4, ipv6 = result4.ip, result6.ip
	if result4.err != nil {
		err = result4.err
	} else if result6.err != nil {
		err = result6.err
	}
	return ipv4, ipv6, err
}

// Resolve resolves the hostname using the UDP DNS server.
func (r *UDP) Resolve(host string) (ipv4, ipv6 net.IP, err error) {
	return resolve(r, host)
}

// Resolve resolves the hostname using the TCP DNS server.
func (r *TCP) Resolve(host string) (ipv4, ipv6 net.IP, err error) {
	return resolve(r, host)
}

// Resolve resolves the hostname using the DNS-over-TLS server.
func (r *TLS) Resolve(host string) (ipv4, ipv6 net.IP, err error) {
	return resolve(r, host)
}
