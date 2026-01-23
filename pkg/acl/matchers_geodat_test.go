package acl

import (
	"testing"

	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

func Test_geositeMatcher_matchDomain_RootDomain(t *testing.T) {
	tests := []struct {
		name        string
		domainValue string
		host        string
		want        bool
	}{
		// Basic matching
		{
			name:        "exact match",
			domainValue: "google.com",
			host:        "google.com",
			want:        true,
		},
		{
			name:        "subdomain match",
			domainValue: "google.com",
			host:        "www.google.com",
			want:        true,
		},
		{
			name:        "deep subdomain match",
			domainValue: "google.com",
			host:        "mail.corp.google.com",
			want:        true,
		},

		// No match cases
		{
			name:        "prefix but not subdomain",
			domainValue: "news.com",
			host:        "fakenews.com",
			want:        false,
		},
		{
			name:        "different domain",
			domainValue: "google.com",
			host:        "google.org",
			want:        false,
		},
		{
			name:        "partial match in middle",
			domainValue: "google.com",
			host:        "notgoogle.com.fake",
			want:        false,
		},

		// Edge cases
		{
			name:        "empty host",
			domainValue: "google.com",
			host:        "",
			want:        false,
		},
		{
			name:        "host shorter than domain",
			domainValue: "google.com",
			host:        "com",
			want:        false,
		},

		// Values that might come from sing-geosite (after TrimPrefix)
		{
			name:        "value without leading dot - exact",
			domainValue: "0emm.com",
			host:        "0emm.com",
			want:        true,
		},
		{
			name:        "value without leading dot - subdomain",
			domainValue: "0emm.com",
			host:        "www.0emm.com",
			want:        true,
		},

		// Make sure we don't incorrectly match when host ends with domain but isn't a subdomain
		{
			name:        "suffix but not boundary - no dot before",
			domainValue: "gle.com",
			host:        "google.com",
			want:        false,
		},

		// Tricky cases with dots
		{
			name:        "domain with subdomain pattern",
			domainValue: "ads.youtube.com",
			host:        "ads.youtube.com",
			want:        true,
		},
		{
			name:        "domain with subdomain pattern - subdomain",
			domainValue: "ads.youtube.com",
			host:        "video.ads.youtube.com",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &geositeMatcher{
				Domains: []geositeDomain{
					{
						Type:  geositeDomainRoot,
						Value: tt.domainValue,
					},
				},
			}
			host := HostInfo{Name: tt.host}
			if got := m.Match(host); got != tt.want {
				t.Errorf("Match() = %v, want %v (domainValue=%q, host=%q)",
					got, tt.want, tt.domainValue, tt.host)
			}
		})
	}
}

func Test_geositeMatcher_matchDomain_Full(t *testing.T) {
	tests := []struct {
		name        string
		domainValue string
		host        string
		want        bool
	}{
		{
			name:        "exact match",
			domainValue: "www.google.com",
			host:        "www.google.com",
			want:        true,
		},
		{
			name:        "no match - different",
			domainValue: "www.google.com",
			host:        "mail.google.com",
			want:        false,
		},
		{
			name:        "no match - subdomain of pattern",
			domainValue: "google.com",
			host:        "www.google.com",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &geositeMatcher{
				Domains: []geositeDomain{
					{
						Type:  geositeDomainFull,
						Value: tt.domainValue,
					},
				},
			}
			host := HostInfo{Name: tt.host}
			if got := m.Match(host); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_geositeMatcher_matchDomain_Plain(t *testing.T) {
	tests := []struct {
		name        string
		domainValue string
		host        string
		want        bool
	}{
		{
			name:        "keyword in domain",
			domainValue: "google",
			host:        "www.google.com",
			want:        true,
		},
		{
			name:        "keyword at start",
			domainValue: "www",
			host:        "www.example.com",
			want:        true,
		},
		{
			name:        "keyword at end",
			domainValue: ".com",
			host:        "example.com",
			want:        true,
		},
		{
			name:        "keyword not found",
			domainValue: "facebook",
			host:        "www.google.com",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &geositeMatcher{
				Domains: []geositeDomain{
					{
						Type:  geositeDomainPlain,
						Value: tt.domainValue,
					},
				},
			}
			host := HostInfo{Name: tt.host}
			if got := m.Match(host); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_geositeMatcher_matchDomain_Regex(t *testing.T) {
	tests := []struct {
		name        string
		domainRegex string
		host        string
		want        bool
	}{
		{
			name:        "simple regex match",
			domainRegex: `^www\..*\.com$`,
			host:        "www.example.com",
			want:        true,
		},
		{
			name:        "regex no match",
			domainRegex: `^www\..*\.com$`,
			host:        "mail.example.com",
			want:        false,
		},
		{
			name:        "number pattern",
			domainRegex: `\d+-\d+\.example\.com`,
			host:        "123-456.example.com",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := newGeositeMatcher(&geodat.GeoSite{
				Domain: []*geodat.Domain{
					{
						Type:  geodat.Domain_Regex,
						Value: tt.domainRegex,
					},
				},
			}, nil)
			if err != nil {
				t.Fatalf("newGeositeMatcher() error = %v", err)
			}

			host := HostInfo{Name: tt.host}
			if got := matcher.Match(host); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_geositeMatcher_attributes(t *testing.T) {
	geosite := &geodat.GeoSite{
		Domain: []*geodat.Domain{
			{
				Type:  geodat.Domain_RootDomain,
				Value: "google.com",
				Attribute: []*geodat.Domain_Attribute{
					{Key: "cn"},
				},
			},
			{
				Type:  geodat.Domain_RootDomain,
				Value: "youtube.com",
				Attribute: []*geodat.Domain_Attribute{
					{Key: "cn"},
					{Key: "ads"},
				},
			},
			{
				Type:  geodat.Domain_RootDomain,
				Value: "facebook.com",
			},
		},
	}

	tests := []struct {
		name  string
		attrs []string
		host  string
		want  bool
	}{
		{
			name:  "no attribute filter - matches all",
			attrs: nil,
			host:  "www.google.com",
			want:  true,
		},
		{
			name:  "no attribute filter - matches domain without attrs",
			attrs: nil,
			host:  "www.facebook.com",
			want:  true,
		},
		{
			name:  "single attr filter - matches",
			attrs: []string{"cn"},
			host:  "www.google.com",
			want:  true,
		},
		{
			name:  "single attr filter - no match (domain has no attrs)",
			attrs: []string{"cn"},
			host:  "www.facebook.com",
			want:  false,
		},
		{
			name:  "multiple attr filter - all present",
			attrs: []string{"cn", "ads"},
			host:  "www.youtube.com",
			want:  true,
		},
		{
			name:  "multiple attr filter - partial match",
			attrs: []string{"cn", "ads"},
			host:  "www.google.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := newGeositeMatcher(geosite, tt.attrs)
			if err != nil {
				t.Fatalf("newGeositeMatcher() error = %v", err)
			}

			host := HostInfo{Name: tt.host}
			if got := matcher.Match(host); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newGeositeMatcher_invalidRegex(t *testing.T) {
	geosite := &geodat.GeoSite{
		Domain: []*geodat.Domain{
			{
				Type:  geodat.Domain_Regex,
				Value: "[invalid(regex",
			},
		},
	}

	_, err := newGeositeMatcher(geosite, nil)
	if err == nil {
		t.Error("newGeositeMatcher() should return error for invalid regex")
	}
}

func Test_geositeMatcher_multipleDomains(t *testing.T) {
	geosite := &geodat.GeoSite{
		Domain: []*geodat.Domain{
			{Type: geodat.Domain_Full, Value: "exact.example.com"},
			{Type: geodat.Domain_RootDomain, Value: "google.com"},
			{Type: geodat.Domain_Plain, Value: "facebook"},
		},
	}

	matcher, err := newGeositeMatcher(geosite, nil)
	if err != nil {
		t.Fatalf("newGeositeMatcher() error = %v", err)
	}

	tests := []struct {
		host string
		want bool
	}{
		{"exact.example.com", true},
		{"www.example.com", false},
		{"google.com", true},
		{"www.google.com", true},
		{"www.facebook.com", true},
		{"twitter.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			host := HostInfo{Name: tt.host}
			if got := matcher.Match(host); got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

// Test to verify the leading dot bug is fixed
func Test_geositeMatcher_leadingDotBugFix(t *testing.T) {
	// This test verifies that domain values WITHOUT leading dots work correctly
	// The sing-geosite loader should strip the leading dot before creating the matcher
	tests := []struct {
		name        string
		domainValue string // This should be WITHOUT leading dot (after TrimPrefix in load.go)
		host        string
		want        bool
	}{
		{
			name:        "google.com exact",
			domainValue: "google.com",
			host:        "google.com",
			want:        true,
		},
		{
			name:        "google.com subdomain",
			domainValue: "google.com",
			host:        "www.google.com",
			want:        true,
		},
		{
			name:        "google.com deep subdomain",
			domainValue: "google.com",
			host:        "mail.corp.google.com",
			want:        true,
		},
		{
			name:        "should not match suffix without dot boundary",
			domainValue: "google.com",
			host:        "fakegoogle.com",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &geositeMatcher{
				Domains: []geositeDomain{
					{
						Type:  geositeDomainRoot,
						Value: tt.domainValue,
					},
				},
			}
			host := HostInfo{Name: tt.host}
			if got := m.Match(host); got != tt.want {
				t.Errorf("Match() = %v, want %v (this may indicate the leading dot bug)",
					got, tt.want)
			}
		})
	}
}
