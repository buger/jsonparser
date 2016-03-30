// +build !appengine,!appenginevm

package strbytes

import (
	"strings"
	"testing"
)

type eqFn func(string, []byte) bool

func TestEqual(t *testing.T) {
	eqs := map[string]eqFn{
		"safeEqual":       equalSafe,
		"unsafeEqual":     equalUnsafe,
		"moreUnsafeEqual": equalMoreUnsafe,
	}

	longstr := strings.Repeat("a", 1000)

	for eqName, eq := range eqs {
		if !eq("", []byte("")) {
			t.Errorf(`%s("", ""): expected true, obtained false`, eqName)
			break
		}

		for i := 0; i < len(longstr); i++ {
			s1, s2 := longstr[:i]+"1", longstr[:i]+"2"
			b1 := []byte(s1)

			if !eq(s1, b1) {
				t.Errorf(`%s("a"*%d + "1", "a"*%d + "1"): expected true, obtained false`, eqName, i, i)
				break
			}
			if eq(s2, b1) {
				t.Errorf(`%s("a"*%d + "1", "a"*%d + "2"): expected false, obtained true`, eqName, i, i)
				break
			}
		}
	}
}

var (
	// short string/[]byte sequences, as the difference between these
	// three methods is a constant overhead
	benchmarkString = "0123456789x"
	benchmarkBytes  = []byte("0123456789y")
)

func BenchmarkSafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		equalSafe(benchmarkString, benchmarkBytes)
	}
}

func BenchmarkUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		equalUnsafe(benchmarkString, benchmarkBytes)
	}
}

func BenchmarkMoreUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		equalMoreUnsafe(benchmarkString, benchmarkBytes)
	}
}
