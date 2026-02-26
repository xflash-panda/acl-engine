package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xflash-panda/acl-engine/pkg/outbound"
)

func TestParseDirectMode(t *testing.T) {
	tests := []struct {
		input string
		want  outbound.DirectMode
	}{
		{"", outbound.DirectModeAuto},
		{"auto", outbound.DirectModeAuto},
		{"Auto", outbound.DirectModeAuto},
		{"64", outbound.DirectMode64},
		{"46", outbound.DirectMode46},
		{"6", outbound.DirectMode6},
		{"4", outbound.DirectMode4},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDirectMode(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := parseDirectMode("invalid")
	assert.Error(t, err)
}

func TestParseMinimalConfig(t *testing.T) {
	yaml := `
acl:
  inline:
    - direct(all)
`
	r, err := Parse([]byte(yaml), nil)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestParseFullOutbounds(t *testing.T) {
	yaml := `
outbounds:
  - name: mydirect
    type: direct
    direct:
      mode: "46"
      bindIPv4: 1.2.3.4
      fastOpen: true

  - name: mysocks
    type: socks5
    socks5:
      addr: 127.0.0.1:1080
      username: user
      password: pass

  - name: myhttp
    type: http
    http:
      url: http://proxy.example.com:8080
      insecure: false

  - name: myreject
    type: reject

acl:
  inline:
    - mydirect(all)
`
	r, err := Parse([]byte(yaml), nil)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestParseWithResolver(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "system",
			yaml: `
resolver:
  type: system
acl:
  inline:
    - direct(all)
`,
		},
		{
			name: "udp",
			yaml: `
resolver:
  type: udp
  udp:
    addr: 8.8.8.8:53
    timeout: 5s
acl:
  inline:
    - direct(all)
`,
		},
		{
			name: "tcp",
			yaml: `
resolver:
  type: tcp
  tcp:
    addr: 8.8.8.8:53
acl:
  inline:
    - direct(all)
`,
		},
		{
			name: "tls",
			yaml: `
resolver:
  type: tls
  tls:
    addr: 1.1.1.1:853
    sni: cloudflare-dns.com
    insecure: false
acl:
  inline:
    - direct(all)
`,
		},
		{
			name: "https",
			yaml: `
resolver:
  type: https
  https:
    addr: https://dns.google/dns-query
    timeout: 5s
acl:
  inline:
    - direct(all)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Parse([]byte(tt.yaml), nil)
			require.NoError(t, err)
			assert.NotNil(t, r)
		})
	}
}

func TestBuildWithCacheSize(t *testing.T) {
	yaml := `
acl:
  inline:
    - direct(all)
`
	r, err := Parse([]byte(yaml), &BuildOptions{CacheSize: 2048})
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "missing outbound name",
			yaml: `
outbounds:
  - type: direct
acl:
  inline:
    - direct(all)
`,
			wantErr: "name is required",
		},
		{
			name: "unknown outbound type",
			yaml: `
outbounds:
  - name: foo
    type: unknown
acl:
  inline:
    - direct(all)
`,
			wantErr: "unknown outbound type",
		},
		{
			name: "invalid direct mode",
			yaml: `
outbounds:
  - name: foo
    type: direct
    direct:
      mode: invalid
acl:
  inline:
    - direct(all)
`,
			wantErr: "unknown direct mode",
		},
		{
			name: "bindDevice with bindIPv4",
			yaml: `
outbounds:
  - name: foo
    type: direct
    direct:
      bindIPv4: 1.2.3.4
      bindDevice: eth0
acl:
  inline:
    - direct(all)
`,
			wantErr: "bindDevice is mutually exclusive with bindIPv4/bindIPv6",
		},
		{
			name: "bindDevice with bindIPv6",
			yaml: `
outbounds:
  - name: foo
    type: direct
    direct:
      bindIPv6: "::1"
      bindDevice: eth0
acl:
  inline:
    - direct(all)
`,
			wantErr: "bindDevice is mutually exclusive with bindIPv4/bindIPv6",
		},
		{
			name: "invalid bindIPv4",
			yaml: `
outbounds:
  - name: foo
    type: direct
    direct:
      bindIPv4: not-an-ip
acl:
  inline:
    - direct(all)
`,
			wantErr: "invalid bindIPv4",
		},
		{
			name: "socks5 missing addr",
			yaml: `
outbounds:
  - name: foo
    type: socks5
    socks5:
      username: user
acl:
  inline:
    - direct(all)
`,
			wantErr: "socks5 addr is required",
		},
		{
			name: "http missing url",
			yaml: `
outbounds:
  - name: foo
    type: http
    http:
      insecure: true
acl:
  inline:
    - direct(all)
`,
			wantErr: "http url is required",
		},
		{
			name: "unknown resolver type",
			yaml: `
resolver:
  type: unknown
acl:
  inline:
    - direct(all)
`,
			wantErr: "unknown resolver type",
		},
		{
			name: "both file and inline",
			yaml: `
acl:
  file: /tmp/rules.acl
  inline:
    - direct(all)
`,
			wantErr: "cannot specify both",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
