package acl

import (
	"testing"

	"github.com/xflash-panda/acl-engine/pkg/acl/geodat"
)

// BenchmarkGeositeMatcher_SmallList benchmarks matching with a small domain list (10 domains)
func BenchmarkGeositeMatcher_SmallList(b *testing.B) {
	domains := make([]*geodat.Domain, 10)
	for i := 0; i < 10; i++ {
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+i)) + ".com",
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	host := HostInfo{Name: "sub.examplee.com"} // Middle of list

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(host)
	}
}

// BenchmarkGeositeMatcher_MediumList benchmarks matching with medium list (100 domains)
func BenchmarkGeositeMatcher_MediumList(b *testing.B) {
	domains := make([]*geodat.Domain, 100)
	for i := 0; i < 100; i++ {
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + ".com",
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	host := HostInfo{Name: "sub.examplema.com"} // Middle of list

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(host)
	}
}

// BenchmarkGeositeMatcher_LargeList benchmarks matching with large list (1000 domains)
func BenchmarkGeositeMatcher_LargeList(b *testing.B) {
	domains := make([]*geodat.Domain, 1000)
	for i := 0; i < 1000; i++ {
		c1 := i % 26
		c2 := (i / 26) % 26
		c3 := (i / 676) % 26
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+c1)) + string(rune('a'+c2)) + string(rune('a'+c3)) + ".com",
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	host := HostInfo{Name: "sub.examplemam.com"} // Middle of list

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(host)
	}
}

// BenchmarkGeositeMatcher_LargeList_Hit benchmarks matching hit in large list
func BenchmarkGeositeMatcher_LargeList_Hit(b *testing.B) {
	domains := make([]*geodat.Domain, 1000)
	for i := 0; i < 1000; i++ {
		c1 := i % 26
		c2 := (i / 26) % 26
		c3 := (i / 676) % 26
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+c1)) + string(rune('a'+c2)) + string(rune('a'+c3)) + ".com",
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	host := HostInfo{Name: "sub.exampleaaa.com"} // First in list

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(host)
	}
}

// BenchmarkGeositeMatcher_LargeList_Miss benchmarks matching miss in large list
func BenchmarkGeositeMatcher_LargeList_Miss(b *testing.B) {
	domains := make([]*geodat.Domain, 1000)
	for i := 0; i < 1000; i++ {
		c1 := i % 26
		c2 := (i / 26) % 26
		c3 := (i / 676) % 26
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+c1)) + string(rune('a'+c2)) + string(rune('a'+c3)) + ".com",
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	host := HostInfo{Name: "notinlist.com"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(host)
	}
}

// BenchmarkGeositeMatcher_MixedTypes benchmarks with mixed domain types
func BenchmarkGeositeMatcher_MixedTypes(b *testing.B) {
	domains := make([]*geodat.Domain, 300)

	// 100 Root domains
	for i := 0; i < 100; i++ {
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "root" + string(rune('a'+i%26)) + ".com",
		}
	}

	// 100 Full domains
	for i := 100; i < 200; i++ {
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_Full,
			Value: "full" + string(rune('a'+i%26)) + ".com",
		}
	}

	// 100 Plain (keyword) domains
	for i := 200; i < 300; i++ {
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_Plain,
			Value: "keyword" + string(rune('a'+i%26)),
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	host := HostInfo{Name: "sub.rootm.com"} // Should hit fast path

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(host)
	}
}

// BenchmarkGeositeMatcher_Construction benchmarks matcher construction time
func BenchmarkGeositeMatcher_Construction_Small(b *testing.B) {
	domains := make([]*geodat.Domain, 10)
	for i := 0; i < 10; i++ {
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+i)) + ".com",
		}
	}

	geosite := &geodat.GeoSite{Domain: domains}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = newGeositeMatcher(geosite, nil)
	}
}

func BenchmarkGeositeMatcher_Construction_Large(b *testing.B) {
	domains := make([]*geodat.Domain, 1000)
	for i := 0; i < 1000; i++ {
		c1 := i % 26
		c2 := (i / 26) % 26
		c3 := (i / 676) % 26
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "example" + string(rune('a'+c1)) + string(rune('a'+c2)) + string(rune('a'+c3)) + ".com",
		}
	}

	geosite := &geodat.GeoSite{Domain: domains}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = newGeositeMatcher(geosite, nil)
	}
}

// Benchmark realistic scenario: many rules like mining pool domains
func BenchmarkGeositeMatcher_MiningPools(b *testing.B) {
	// Simulating 857 mining pool domains from your config
	domains := make([]*geodat.Domain, 857)
	for i := 0; i < 857; i++ {
		c1 := i % 26
		c2 := (i / 26) % 26
		c3 := (i / 676) % 26
		domains[i] = &geodat.Domain{
			Type:  geodat.Domain_RootDomain,
			Value: "pool" + string(rune('a'+c1)) + string(rune('a'+c2)) + string(rune('a'+c3)) + ".com",
		}
	}

	matcher, err := newGeositeMatcher(&geodat.GeoSite{Domain: domains}, nil)
	if err != nil {
		b.Fatal(err)
	}

	// Test various scenarios
	b.Run("Hit_Early", func(b *testing.B) {
		host := HostInfo{Name: "sub.poolaaa.com"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matcher.Match(host)
		}
	})

	b.Run("Hit_Middle", func(b *testing.B) {
		host := HostInfo{Name: "sub.poolmam.com"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matcher.Match(host)
		}
	})

	b.Run("Miss", func(b *testing.B) {
		host := HostInfo{Name: "google.com"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matcher.Match(host)
		}
	})
}
