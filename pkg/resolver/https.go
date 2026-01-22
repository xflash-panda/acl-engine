package resolver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// HTTPS is a Resolver that uses a DNS-over-HTTPS server.
type HTTPS struct {
	url        string
	httpClient *http.Client
}

// NewHTTPS creates a new DNS-over-HTTPS resolver.
func NewHTTPS(addr string, timeout time.Duration, sni string, insecure bool) Resolver {
	url := addr
	if !strings.HasPrefix(addr, "https://") {
		url = fmt.Sprintf("https://%s/dns-query", addr)
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: insecure, //nolint:gosec // user configurable
	}
	return &HTTPS{
		url: url,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   timeoutOrDefault(timeout),
		},
	}
}

func (r *HTTPS) lookup(dnsType dnsmessage.Type, host string) ([]dnsmessage.Resource, error) {
	if r.url == "" {
		return nil, errors.New("no DoH URL provided")
	}
	client := r.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	if !strings.HasSuffix(host, ".") {
		host += "."
	}
	name, err := dnsmessage.NewName(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host %s: %w", host, err)
	}

	reqBuilder := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		RecursionDesired: true,
	})
	reqBuilder.EnableCompression()
	err = reqBuilder.StartQuestions()
	if err != nil {
		return nil, fmt.Errorf("failed to start dns questions for host %s: %w", host, err)
	}
	err = reqBuilder.Question(dnsmessage.Question{
		Name:  name,
		Type:  dnsType,
		Class: dnsmessage.ClassINET,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build dns question for host %s: %w", host, err)
	}
	reqMsg, err := reqBuilder.Finish()
	if err != nil {
		return nil, fmt.Errorf("failed to finish dns message for host %s: %w", host, err)
	}
	httpReq, err := http.NewRequest("POST", r.url, strings.NewReader(string(reqMsg)))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request for host %s: %w", host, err)
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform http request for host %s: %w", host, err)
	}
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status-code=%d for host %s", httpResp.StatusCode, host)
	}
	if httpResp.Header.Get("Content-Type") != "application/dns-message" {
		return nil, fmt.Errorf("unexpected content-type=%s for host %s", httpResp.Header.Get("Content-Type"), host)
	}

	limitedBody := io.LimitReader(httpResp.Body, 65536)
	respMsg, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read http response body for host %s: %w", host, err)
	}
	parser := dnsmessage.Parser{}
	header, err := parser.Start(respMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dns message header for host %s: %w", host, err)
	}
	if header.RCode != dnsmessage.RCodeSuccess {
		return nil, fmt.Errorf("dns query failed with %s for host %s", header.RCode, host)
	}
	err = parser.SkipAllQuestions()
	if err != nil {
		return nil, fmt.Errorf("failed to skip dns questions for host %s: %w", host, err)
	}
	answers, err := parser.AllAnswers()
	if err != nil {
		return nil, fmt.Errorf("failed to parse dns answers for host %s: %w", host, err)
	}
	return answers, nil
}

func (r *HTTPS) lookupA(host string) ([]net.IP, error) {
	answers, err := r.lookup(dnsmessage.TypeA, host)
	if err != nil {
		return nil, err
	}
	var results []net.IP
	for _, rr := range answers {
		if rr.Header.Type == dnsmessage.TypeA {
			a := rr.Body.(*dnsmessage.AResource)
			results = append(results, a.A[:])
		}
	}
	return results, nil
}

func (r *HTTPS) lookupAAAA(host string) ([]net.IP, error) {
	answers, err := r.lookup(dnsmessage.TypeAAAA, host)
	if err != nil {
		return nil, err
	}
	var results []net.IP
	for _, rr := range answers {
		if rr.Header.Type == dnsmessage.TypeAAAA {
			aaaa := rr.Body.(*dnsmessage.AAAAResource)
			results = append(results, aaaa.AAAA[:])
		}
	}
	return results, nil
}

// Resolve resolves the hostname using the DNS-over-HTTPS server.
func (r *HTTPS) Resolve(host string) (ipv4, ipv6 net.IP, err error) {
	type lookupResult struct {
		ip  net.IP
		err error
	}
	ch4, ch6 := make(chan lookupResult, 1), make(chan lookupResult, 1)
	go func() {
		ips, lookupErr := r.lookupA(host)
		var ip net.IP
		if lookupErr == nil && len(ips) > 0 {
			ip = ips[0]
		}
		ch4 <- lookupResult{ip, lookupErr}
	}()
	go func() {
		ips, lookupErr := r.lookupAAAA(host)
		var ip net.IP
		if lookupErr == nil && len(ips) > 0 {
			ip = ips[0]
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
