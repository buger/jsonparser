package jsonparser

import (
	"bytes"
	"testing"
)

// Verifies: SYS-REQ-115
// MCDC SYS-REQ-115: config_lenient_modes_are_opt_in_and_default_remains_strict=T => TRUE
// reqproof:proptest:skip MC/DC witness for the lenient-config positive row
func TestMCDC_SYS_REQ_115_Row2_LenientConfigParsesSingleQuotes(t *testing.T) {
	data := []byte(`{'key':'value'}`)
	val, err := Lenient.GetString(data, "key")
	if err != nil {
		t.Fatalf("Lenient.GetString error: %v", err)
	}
	if val != "value" {
		t.Fatalf("got %q, want value", val)
	}
}

// Verifies: SYS-REQ-115
// MCDC SYS-REQ-115: config_lenient_modes_are_opt_in_and_default_remains_strict=F => FALSE
// reqproof:proptest:skip MC/DC witness for the strict-default negative row
func TestMCDC_SYS_REQ_115_Row1_StrictConfigRejectsSingleQuotes(t *testing.T) {
	data := []byte(`{'key':'value'}`)
	_, err := DefaultConfig.GetString(data, "key")
	if err == nil {
		t.Fatalf("expected error on strict config with single quotes")
	}
	_, _, _, err2 := Get(data, "key")
	if err2 == nil {
		t.Fatalf("expected error on package-level Get with single quotes")
	}
}

// Verifies: SYS-REQ-116
// MCDC SYS-REQ-116: reader_parser_provides_incremental_stream_access=T => TRUE
// reqproof:proptest:skip MC/DC witness for the streaming positive row
func TestMCDC_SYS_REQ_116_Row2_ReaderParserGetsFromStream(t *testing.T) {
	data := []byte(`{"users":[{"name":"Alice"}]}`)
	rp := NewReaderParser(bytes.NewReader(data))
	val, err := rp.GetString("users", "[0]", "name")
	if err != nil {
		t.Fatalf("ReaderParser.GetString error: %v", err)
	}
	if val != "Alice" {
		t.Fatalf("got %q, want Alice", val)
	}
}

// Verifies: SYS-REQ-116
// MCDC SYS-REQ-116: reader_parser_provides_incremental_stream_access=F => FALSE
// reqproof:proptest:skip MC/DC witness for the non-streaming negative row
func TestMCDC_SYS_REQ_116_Row1_PackageGetRequiresByteSliceNotReader(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	val, _, _, err := Get(data, "key")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if string(val) != "value" {
		t.Fatalf("got %q, want value", val)
	}
}
