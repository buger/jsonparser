// Native Go fuzz wrappers around the OSS-Fuzz targets in fuzz.go so they can
// be driven by `go test -fuzz` locally. Panics are recovered and recorded
// (deduplicated by panic msg + innermost parser frame) under
// fuzz_results/crashes/, so a single fuzz run keeps exploring after the first
// crash instead of stopping. See run_fuzz_campaign.sh for batch usage.
package jsonparser

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

var nativeFuzzSeeds = []string{
	"{}",
	"[]",
	`{"test":1}`,
	`{"test":"hello","other":2}`,
	`{"a":{"test":[1,2,3]},"test":null}`,
	`{"test":{"nested":{"test":true}}}`,
	`[1,2,3,4]`,
	`{"test":}`,
	`{"x":"test"}`,
	`{"test":"\""}`,
	`{"test":1.5e10}`,
	`{"test":-0.0}`,
	`{"test":true,"a":false,"b":null}`,
	`,{"test":1{}`,
	`,""{"test":0}`,
	`{"name":"x","order":1,"nested":{"a":1,"b":2,"nested3":{"b":3}},"nested2":{"a":4},"arr":[{"b":5},{"b":6}],"arrInt":[0,1,2,3,4,5,6]}`,
	"",
}

func addSeeds(f *testing.F) {
	for _, s := range nativeFuzzSeeds {
		f.Add([]byte(s))
	}
}

var fuzzCrashDir = func() string {
	d := os.Getenv("FUZZ_CRASH_DIR")
	if d == "" {
		d = "fuzz_results/crashes"
	}
	_ = os.MkdirAll(d, 0o755)
	return d
}()

func crashSignature(panicMsg string, stack []byte) string {
	var key strings.Builder
	key.WriteString(panicMsg)
	for _, line := range strings.Split(string(stack), "\n") {
		if strings.Contains(line, "github.com/buger/jsonparser/") &&
			!strings.Contains(line, "fuzz_native_test.go") &&
			!strings.Contains(line, "/fuzz.go:") {
			key.WriteString("|")
			key.WriteString(strings.TrimSpace(line))
			break
		}
	}
	sum := sha256.Sum256([]byte(key.String()))
	return hex.EncodeToString(sum[:])[:12]
}

func recordCrash(target string, panicVal interface{}, stack, input []byte) {
	panicMsg := fmt.Sprintf("%v", panicVal)
	sig := crashSignature(panicMsg, stack)
	fname := filepath.Join(fuzzCrashDir, target+"_"+sig+".json")
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)
	_ = json.NewEncoder(f).Encode(map[string]interface{}{
		"target":    target,
		"sig":       sig,
		"panic":     panicMsg,
		"stack":     string(stack),
		"input_b64": base64.StdEncoding.EncodeToString(inputCopy),
	})
}

func runWithCapture(target string, data []byte, fn func([]byte)) {
	defer func() {
		if r := recover(); r != nil {
			recordCrash(target, r, debug.Stack(), data)
		}
	}()
	fn(data)
}

func FuzzDeleteNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzDeleteNative", data, func(d []byte) { _ = FuzzDelete(d) })
	})
}

func FuzzParseStringNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzParseStringNative", data, func(d []byte) { _ = FuzzParseString(d) })
	})
}

func FuzzEachKeyNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzEachKeyNative", data, func(d []byte) { _ = FuzzEachKey(d) })
	})
}

func FuzzSetNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzSetNative", data, func(d []byte) { _ = FuzzSet(d) })
	})
}

func FuzzObjectEachNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzObjectEachNative", data, func(d []byte) { _ = FuzzObjectEach(d) })
	})
}

func FuzzParseFloatNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzParseFloatNative", data, func(d []byte) { _ = FuzzParseFloat(d) })
	})
}

func FuzzParseIntNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzParseIntNative", data, func(d []byte) { _ = FuzzParseInt(d) })
	})
}

func FuzzParseBoolNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzParseBoolNative", data, func(d []byte) { _ = FuzzParseBool(d) })
	})
}

func FuzzTokenStartNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzTokenStartNative", data, func(d []byte) { _ = FuzzTokenStart(d) })
	})
}

func FuzzGetStringNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzGetStringNative", data, func(d []byte) { _ = FuzzGetString(d) })
	})
}

func FuzzGetFloatNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzGetFloatNative", data, func(d []byte) { _ = FuzzGetFloat(d) })
	})
}

func FuzzGetIntNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzGetIntNative", data, func(d []byte) { _ = FuzzGetInt(d) })
	})
}

func FuzzGetBooleanNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzGetBooleanNative", data, func(d []byte) { _ = FuzzGetBoolean(d) })
	})
}

func FuzzGetUnsafeStringNative(f *testing.F) {
	addSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		runWithCapture("FuzzGetUnsafeStringNative", data, func(d []byte) { _ = FuzzGetUnsafeString(d) })
	})
}
