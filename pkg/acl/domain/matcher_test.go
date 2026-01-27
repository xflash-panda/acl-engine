package domain

import (
	"testing"
)

func TestMatcher_BasicMatching(t *testing.T) {
	tests := []struct {
		name          string
		domains       []string
		domainSuffix  []string
		testDomain    string
		shouldMatch   bool
	}{
		{
			name:         "exact match",
			domains:      []string{"google.com"},
			domainSuffix: nil,
			testDomain:   "google.com",
			shouldMatch:  true,
		},
		{
			name:         "exact no match",
			domains:      []string{"google.com"},
			domainSuffix: nil,
			testDomain:   "mail.google.com",
			shouldMatch:  false,
		},
		{
			name:         "suffix match - subdomain",
			domains:      nil,
			domainSuffix: []string{"google.com"},
			testDomain:   "mail.google.com",
			shouldMatch:  true,
		},
		{
			name:         "suffix match - exact",
			domains:      nil,
			domainSuffix: []string{"google.com"},
			testDomain:   "google.com",
			shouldMatch:  true,
		},
		{
			name:         "suffix with dot - subdomain only",
			domains:      nil,
			domainSuffix: []string{".google.com"},
			testDomain:   "mail.google.com",
			shouldMatch:  true,
		},
		{
			name:         "suffix with dot - not exact",
			domains:      nil,
			domainSuffix: []string{".google.com"},
			testDomain:   "google.com",
			shouldMatch:  false,
		},
		{
			name:         "no match",
			domains:      []string{"google.com"},
			domainSuffix: []string{"baidu.com"},
			testDomain:   "bing.com",
			shouldMatch:  false,
		},
		{
			name:         "case insensitive",
			domains:      []string{"Google.COM"},
			domainSuffix: nil,
			testDomain:   "google.com",
			shouldMatch:  true,
		},
		{
			name:         "multiple levels subdomain",
			domains:      nil,
			domainSuffix: []string{"google.com"},
			testDomain:   "a.b.c.google.com",
			shouldMatch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewMatcher(tt.domains, tt.domainSuffix)
			result := matcher.Match(tt.testDomain)
			if result != tt.shouldMatch {
				t.Errorf("Match(%q) = %v, want %v", tt.testDomain, result, tt.shouldMatch)
			}
		})
	}
}

func TestMatcher_EmptyInput(t *testing.T) {
	matcher := NewMatcher(nil, nil)
	if matcher.Match("google.com") {
		t.Error("Empty matcher should not match anything")
	}
}

func TestMatcher_LargeDomainList(t *testing.T) {
	// Test with many domains (simulating geosite scenario)
	suffixes := []string{
		"examplea.com",
		"exampleb.com",
		"examplec.com",
	}

	matcher := NewMatcher(nil, suffixes)

	// Should match
	if !matcher.Match("examplea.com") {
		t.Error("Should match exact domain")
	}

	if !matcher.Match("sub.examplea.com") {
		t.Error("Should match subdomain of first suffix")
	}

	// Should not match
	if matcher.Match("notinlist.com") {
		t.Error("Should not match domain not in list")
	}
}

func TestMatcher_SpecialCharacters(t *testing.T) {
	matcher := NewMatcher(nil, []string{"example-test.com", "example_test.com"})

	tests := []struct {
		domain      string
		shouldMatch bool
	}{
		{"example-test.com", true},
		{"sub.example-test.com", true},
		{"example_test.com", true},
		{"sub.example_test.com", true},
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := matcher.Match(tt.domain)
			if result != tt.shouldMatch {
				t.Errorf("Match(%q) = %v, want %v", tt.domain, result, tt.shouldMatch)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMatcher_Match_Hit_First(b *testing.B) {
	// Benchmark matching first domain in list
	suffixes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		suffixes[i] = "example" + string(rune('a'+i%26)) + ".com"
	}
	matcher := NewMatcher(nil, suffixes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match("sub.examplea.com")
	}
}

func BenchmarkMatcher_Match_Hit_Middle(b *testing.B) {
	// Benchmark matching middle domain in list
	suffixes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		suffixes[i] = "example" + string(rune('a'+i%26)) + ".com"
	}
	matcher := NewMatcher(nil, suffixes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match("sub.examplem.com")
	}
}

func BenchmarkMatcher_Match_Miss(b *testing.B) {
	// Benchmark no match scenario
	suffixes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		suffixes[i] = "example" + string(rune('a'+i%26)) + ".com"
	}
	matcher := NewMatcher(nil, suffixes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match("notfound.com")
	}
}

func BenchmarkMatcher_Construction(b *testing.B) {
	// Benchmark matcher construction time
	suffixes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		suffixes[i] = "example" + string(rune('a'+i%26)) + ".com"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMatcher(nil, suffixes)
	}
}
