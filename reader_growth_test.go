package jsonparser

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Verifies: SYS-REQ-116 (incremental stream access)
// reqproof:proptest:skip deterministic retained-buffer growth witness
func TestReaderParserRetainedBufferGrowth(t *testing.T) {
	for _, chunkSize := range []int{1024, defaultReaderBufferSize} {
		t.Run(fmt.Sprint(chunkSize), func(t *testing.T) {
			// A loose bound rejects per-chunk reallocation without pinning an
			// exact geometric growth factor.
			const reads = 128
			source := bytes.NewReader(bytes.Repeat([]byte{'x'}, reads*chunkSize))
			rp := NewReaderParser(source, Config{MaxBufferSize: chunkSize})
			growths := 0
			for source.Len() > 0 {
				before := cap(rp.buf)
				if err := rp.readMore(); err != nil {
					t.Fatal(err)
				}
				if cap(rp.buf) != before {
					growths++
				}
			}
			if len(rp.buf) != reads*chunkSize {
				t.Fatalf("retained %d bytes", len(rp.buf))
			}
			if !bytes.Equal(rp.buf, bytes.Repeat([]byte{'x'}, reads*chunkSize)) {
				t.Fatal("retained data changed")
			}
			if growths > 32 {
				t.Fatalf("buffer grew %d times across %d reads; retained values need amortized growth", growths, reads)
			}
		})
	}
}

func BenchmarkReaderParserRetainedValue(b *testing.B) {
	for _, size := range []int{1 << 20, 8 << 20} {
		for _, chunkSize := range []int{1024, defaultReaderBufferSize} {
			if size == 8<<20 && chunkSize == 1024 {
				continue
			}
			data := `{"target":"` + strings.Repeat("x", size) + `"}`
			b.Run(fmt.Sprintf("bytes=%d/chunk=%d", size, chunkSize), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(data)))
				for i := 0; i < b.N; i++ {
					value, vt, err := NewReaderParser(strings.NewReader(data), Config{MaxBufferSize: chunkSize}).Get("target")
					if err != nil || vt != String || len(value) != size {
						b.Fatalf("len=%d type=%v err=%v", len(value), vt, err)
					}
				}
			})
		}
	}
}

// Verifies: SYS-REQ-116 (incremental stream access)
// reqproof:proptest:skip retained-value compatibility witness
func TestReaderParserGrowthPreservesCapturedValues(t *testing.T) {
	payload := strings.Repeat("escaped\\\"value", 32768)
	for _, window := range []int{1024, defaultReaderBufferSize} {
		t.Run(fmt.Sprint(window), func(t *testing.T) {
			for _, data := range []string{`"` + payload + `"`, `{"target":"` + payload + `"}`, `{"target":{"nested":["` + payload + `",2]}}`} {
				path := []string(nil)
				if data[0] == '{' {
					path = []string{"target"}
				}
				want, wt, _, we := Get([]byte(data), path...)
				got, gt, ge := NewReaderParser(strings.NewReader(data), Config{MaxBufferSize: window}).Get(path...)
				if we != nil || ge != nil || wt != gt || !bytes.Equal(want, got) {
					t.Fatalf("captured value differs: types %v/%v errors %v/%v lengths %d/%d", wt, gt, we, ge, len(want), len(got))
				}
			}
			key := strings.Repeat("long-key", 32768)
			value, err := NewReaderParser(strings.NewReader(`{"`+key+`":"found"}`), Config{MaxBufferSize: window}).GetString(key)
			if err != nil || value != "found" {
				t.Fatalf("long key lookup = (%q, %v)", value, err)
			}
			data := `["` + payload + `",{"second":true},3]`
			var want, got []string
			_, err = ArrayEach([]byte(data), func(v []byte, _ ValueType, _ int, e error) {
				if e != nil {
					t.Error(e)
				}
				want = append(want, string(v))
			})
			if err != nil {
				t.Fatal(err)
			}
			err = NewReaderParser(strings.NewReader(data), Config{MaxBufferSize: window}).ArrayEach(func(v []byte, _ ValueType, e error) {
				if e != nil {
					t.Error(e)
				}
				got = append(got, string(v))
			})
			if err != nil || strings.Join(want, "|") != strings.Join(got, "|") {
				t.Fatalf("array differs: err=%v elements=%d/%d", err, len(want), len(got))
			}
		})
	}
}

// Verifies: SYS-REQ-116 (stream error propagation)
// reqproof:proptest:skip injected reader-error witness
func TestReaderParserGrowthReadError(t *testing.T) {
	sentinel := fmt.Errorf("reader stopped")
	data := `{"target":"` + strings.Repeat("x", 8*1024)
	rp := NewReaderParser(io.MultiReader(strings.NewReader(data), readerGrowthError{sentinel}), Config{MaxBufferSize: 1024})
	if _, _, err := rp.Get("target"); err != sentinel {
		t.Fatalf("Get error = %v, want original reader error", err)
	}
}

type readerGrowthError struct{ err error }

func (r readerGrowthError) Read([]byte) (int, error) { return 0, r.err }
