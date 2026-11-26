package benchmark

import (
	"github.com/buger/jsonparser"
	"strconv"
	"testing"
)

// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkSetLarge(b *testing.B) {
	b.ReportAllocs()

	keyPath := make([]string, 20000)
	for i := range keyPath {
		keyPath[i] = "keyPath" + strconv.Itoa(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jsonparser.Set(largeFixture, largeFixture, keyPath...)
	}
}
