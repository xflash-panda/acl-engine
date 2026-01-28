package metadb

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadGeoIP(b *testing.B) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("testdata/geoip.metadb not found, skipping benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := LoadGeoIP(testFile)
		if err != nil {
			b.Fatal(err)
		}
		// Verify we got data
		if len(result) == 0 {
			b.Fatal("expected non-empty result")
		}
	}
}

func BenchmarkLoadGeoIP_Parallel(b *testing.B) {
	testFile := filepath.Join(getTestDataDir(), "geoip.metadb")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("testdata/geoip.metadb not found, skipping benchmark")
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result, err := LoadGeoIP(testFile)
			if err != nil {
				b.Fatal(err)
			}
			if len(result) == 0 {
				b.Fatal("expected non-empty result")
			}
		}
	})
}
