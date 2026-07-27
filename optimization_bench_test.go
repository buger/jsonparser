package jsonparser

import (
	"strings"
	"testing"
)

// =============================================================================
// Micro-benchmarks for the fast-path optimizations modeled on PR #285.
//
// Each benchmark measures one of the optimized hot paths on inputs that
// exercise (a) the common-case fast path and (b) the rare-case slow path,
// so the before/after delta is attributable to the optimization rather
// than input mix.
// =============================================================================

// --- stringEnd ---------------------------------------------------------------
// Common case: a string value with NO backslash escapes. The fast path uses
// bytes.IndexByte (SIMD on amd64/arm64) to skip the per-byte scan.

func BenchmarkStringEnd_NoEscape_Short(b *testing.B) {
	data := []byte(`hello world"`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stringEnd(data)
	}
}

func BenchmarkStringEnd_NoEscape_Long(b *testing.B) {
	// 128-byte string value, no escapes — exercises the SIMD fast path.
	data := []byte(strings.Repeat("a", 128) + `"`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stringEnd(data)
	}
}

func BenchmarkStringEnd_WithEscape(b *testing.B) {
	// Slow path: contains an escaped quote so the per-byte walker runs.
	data := []byte(`hello \"world"`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = stringEnd(data)
	}
}

// --- tokenEnd ---------------------------------------------------------------
// Common case: a bare scalar value (number, boolean, null) terminated by a
// delimiter. The fast path uses bytes.IndexAny (SIMD) to find it.

func BenchmarkTokenEnd_Number(b *testing.B) {
	// Typical short number terminated by a comma.
	data := []byte(`1234567890,`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenEnd(data)
	}
}

func BenchmarkTokenEnd_LongNumber(b *testing.B) {
	// A 64-digit number — exercises the SIMD scan.
	data := []byte(strings.Repeat("1", 64) + `,`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenEnd(data)
	}
}

func BenchmarkTokenEnd_Boolean(b *testing.B) {
	data := []byte(`true,`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenEnd(data)
	}
}

// --- end-to-end Get with a string value -------------------------------------
// Verifies that stringEnd's fast path actually shows up in a realistic call.

func BenchmarkGet_StringValue_NoEscape(b *testing.B) {
	data := []byte(`{"key":"` + strings.Repeat("a", 128) + `"}`)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = Get(data, "key")
	}
}
