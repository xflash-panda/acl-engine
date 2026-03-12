package acl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseGeoSiteName(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		want  string
		want1 []string
	}{
		{
			name:  "no attrs",
			s:     "pornhub",
			want:  "pornhub",
			want1: []string{},
		},
		{
			name:  "one attr 1",
			s:     "xiaomi@cn",
			want:  "xiaomi",
			want1: []string{"cn"},
		},
		{
			name:  "one attr 2",
			s:     " google @jp ",
			want:  "google",
			want1: []string{"jp"},
		},
		{
			name:  "two attrs 1",
			s:     "netflix@jp@kr",
			want:  "netflix",
			want1: []string{"jp", "kr"},
		},
		{
			name:  "two attrs 2",
			s:     "netflix @xixi    @haha ",
			want:  "netflix",
			want1: []string{"xixi", "haha"},
		},
		{
			name:  "empty",
			s:     "",
			want:  "",
			want1: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := parseGeoSiteName(tt.s)
			assert.Equalf(t, tt.want, got, "parseGeoSiteName(%v)", tt.s)
			assert.Equalf(t, tt.want1, got1, "parseGeoSiteName(%v)", tt.s)
		})
	}
}

func Test_compiledRule_MatchPort0(t *testing.T) {
	// Bug: A rule with port 0 (e.g. "tcp/0") should only match port 0,
	// but StartPort==0 causes the port filter to be skipped entirely,
	// matching ALL ports instead.
	rule := compiledRule[string]{
		Outbound:      "test",
		HostMatcher:   &allMatcher{},
		Protocol:      ProtocolTCP,
		HasPortFilter: true,
		StartPort:     0,
		EndPort:       0,
	}

	// Port 0 should match
	assert.True(t, rule.Match(HostInfo{Name: "example.com"}, ProtocolTCP, 0),
		"tcp/0 rule should match port 0")

	// Port 443 should NOT match a port-0-only rule
	assert.False(t, rule.Match(HostInfo{Name: "example.com"}, ProtocolTCP, 443),
		"tcp/0 rule should NOT match port 443")

	// Port 80 should NOT match a port-0-only rule
	assert.False(t, rule.Match(HostInfo{Name: "example.com"}, ProtocolTCP, 80),
		"tcp/0 rule should NOT match port 80")
}

func Test_parseProtoPort(t *testing.T) {
	tests := []struct {
		name           string
		protoPort      string
		wantProto      Protocol
		wantPortFilter bool
		wantStart      uint16
		wantEnd        uint16
		wantOK         bool
	}{
		{
			name:      "empty",
			protoPort: "",
			wantProto: ProtocolBoth,
			wantOK:    true,
		},
		{
			name:      "wildcard",
			protoPort: "*",
			wantProto: ProtocolBoth,
			wantOK:    true,
		},
		{
			name:      "tcp only",
			protoPort: "tcp",
			wantProto: ProtocolTCP,
			wantOK:    true,
		},
		{
			name:      "udp only",
			protoPort: "udp",
			wantProto: ProtocolUDP,
			wantOK:    true,
		},
		{
			name:           "tcp with port",
			protoPort:      "tcp/443",
			wantProto:      ProtocolTCP,
			wantPortFilter: true,
			wantStart:      443,
			wantEnd:        443,
			wantOK:         true,
		},
		{
			name:           "udp with port range",
			protoPort:      "udp/6881-6889",
			wantProto:      ProtocolUDP,
			wantPortFilter: true,
			wantStart:      6881,
			wantEnd:        6889,
			wantOK:         true,
		},
		{
			name:           "tcp with port 0",
			protoPort:      "tcp/0",
			wantProto:      ProtocolTCP,
			wantPortFilter: true,
			wantStart:      0,
			wantEnd:        0,
			wantOK:         true,
		},
		{
			name:      "tcp with wildcard port",
			protoPort: "tcp/*",
			wantProto: ProtocolTCP,
			wantOK:    true,
		},
		{
			name:      "invalid port range",
			protoPort: "tcp/9000-8000",
			wantOK:    false,
		},
		{
			name:      "invalid protocol",
			protoPort: "icmp",
			wantOK:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto, hasPortFilter, start, end, ok := parseProtoPort(tt.protoPort)
			assert.Equal(t, tt.wantProto, proto)
			assert.Equal(t, tt.wantPortFilter, hasPortFilter)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
