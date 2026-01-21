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

func Test_parseProtoPort(t *testing.T) {
	tests := []struct {
		name      string
		protoPort string
		wantProto Protocol
		wantStart uint16
		wantEnd   uint16
		wantOK    bool
	}{
		{
			name:      "empty",
			protoPort: "",
			wantProto: ProtocolBoth,
			wantStart: 0,
			wantEnd:   0,
			wantOK:    true,
		},
		{
			name:      "wildcard",
			protoPort: "*",
			wantProto: ProtocolBoth,
			wantStart: 0,
			wantEnd:   0,
			wantOK:    true,
		},
		{
			name:      "tcp only",
			protoPort: "tcp",
			wantProto: ProtocolTCP,
			wantStart: 0,
			wantEnd:   0,
			wantOK:    true,
		},
		{
			name:      "udp only",
			protoPort: "udp",
			wantProto: ProtocolUDP,
			wantStart: 0,
			wantEnd:   0,
			wantOK:    true,
		},
		{
			name:      "tcp with port",
			protoPort: "tcp/443",
			wantProto: ProtocolTCP,
			wantStart: 443,
			wantEnd:   443,
			wantOK:    true,
		},
		{
			name:      "udp with port range",
			protoPort: "udp/6881-6889",
			wantProto: ProtocolUDP,
			wantStart: 6881,
			wantEnd:   6889,
			wantOK:    true,
		},
		{
			name:      "invalid port range",
			protoPort: "tcp/9000-8000",
			wantProto: ProtocolBoth,
			wantStart: 0,
			wantEnd:   0,
			wantOK:    false,
		},
		{
			name:      "invalid protocol",
			protoPort: "icmp",
			wantProto: ProtocolBoth,
			wantStart: 0,
			wantEnd:   0,
			wantOK:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto, start, end, ok := parseProtoPort(tt.protoPort)
			assert.Equal(t, tt.wantProto, proto)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
