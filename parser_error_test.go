package jsonparser

import (
	"fmt"
	"strings"
	"testing"
)

var testPaths = [][]string{
	[]string{"test"},
	[]string{"these"},
	[]string{"keys"},
	[]string{"please"},
}

// Test helper for SYS-REQ-008.
func testIter(data []byte) (err error) {
	EachKey(data, func(idx int, value []byte, vt ValueType, iterErr error) {
		if iterErr != nil {
			err = fmt.Errorf("Error parsing json: %s", iterErr.Error())
		}
	}, testPaths...)
	return err
}

// Verifies: SYS-REQ-001 [malformed]
// MCDC SYS-REQ-001: N/A
// Verifies: SYS-REQ-008 [malformed]
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=F, eachkey_completes_requested_scan=F, eachkey_malformed_input_returns_error=T, missing_multipath_request_does_not_emit_callback=F, multipath_requests_are_provided=T => TRUE
func TestPanickingErrors(t *testing.T) {
	if err := testIter([]byte(`{"test":`)); err == nil {
		t.Error("Expected error...")
	}

	if err := testIter([]byte(`{"test":0}some":[{"these":[{"keys":"some"}]}]}some"}]}],"please":"some"}`)); err == nil {
		t.Error("Expected error...")
	}

	if _, _, _, err := Get([]byte(`{"test":`), "test"); err == nil {
		t.Error("Expected error...")
	}

	if _, _, _, err := Get([]byte(`{"some":0}some":[{"some":[{"some":"some"}]}]}some"}]}],"some":"some"}`), "x"); err == nil {
		t.Error("Expected error...")
	}
}

// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=F, eachkey_completes_requested_scan=F, eachkey_malformed_input_returns_error=F, missing_multipath_request_does_not_emit_callback=F, multipath_requests_are_provided=F => TRUE
func TestEachKeyNoRequests(t *testing.T) {
	called := false
	EachKey([]byte(`{"a":1}`), func(idx int, value []byte, vt ValueType, err error) {
		called = true
	})
	if called {
		t.Fatal("EachKey should not invoke the callback when no paths are requested")
	}
}

// check having a very deep key depth
// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestKeyDepth(t *testing.T) {
	var sb strings.Builder
	var keys []string
	//build data
	sb.WriteString("{")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&sb, `"key%d": %dx,`, i, i)
		keys = append(keys, fmt.Sprintf("key%d", i))
	}
	sb.WriteString("}")

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys)
}

// check having a bunch of keys in a call to EachKey
// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestKeyCount(t *testing.T) {
	var sb strings.Builder
	var keys [][]string
	//build data
	sb.WriteString("{")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&sb, `"key%d":"%d"`, i, i)
		if i < 127 {
			sb.WriteString(",")
		}
		keys = append(keys, []string{fmt.Sprintf("key%d", i)})
	}
	sb.WriteString("}")

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys...)
}

// try pulling lots of keys out of a big array
// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestKeyDepthArray(t *testing.T) {
	var sb strings.Builder
	var keys []string
	//build data
	sb.WriteString("[")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&sb, `{"key": %d},`, i)
		keys = append(keys, fmt.Sprintf("[%d].key", i))
	}
	sb.WriteString("]")

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys)
}

// check having a bunch of keys
// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestKeyCountArray(t *testing.T) {
	var sb strings.Builder
	var keys [][]string
	//build data
	sb.WriteString("[")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&sb, `{"key":"%d"}`, i)
		if i < 127 {
			sb.WriteString(",")
		}
		keys = append(keys, []string{fmt.Sprintf("[%d].key", i)})
	}
	sb.WriteString("]")

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys...)
}

// check having a bunch of keys in a super deep array
// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestEachKeyArray(t *testing.T) {
	var sb strings.Builder
	var keys [][]string
	//build data
	sb.WriteString(`[`)
	for i := 0; i < 127; i++ {
		fmt.Fprintf(&sb, `%d`, i)
		if i < 127 {
			sb.WriteString(",")
		}
		if i < 32 {
			keys = append(keys, []string{fmt.Sprintf("[%d]", 128+i)})
		}
	}
	sb.WriteString(`]`)

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys...)
}

// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestLargeArray(t *testing.T) {
	var sb strings.Builder
	//build data
	sb.WriteString(`[`)
	for i := 0; i < 127; i++ {
		fmt.Fprintf(&sb, `%d`, i)
		if i < 127 {
			sb.WriteString(",")
		}
	}
	sb.WriteString(`]`)
	keys := [][]string{[]string{`[1]`}}

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys...)
}

// Verifies: SYS-REQ-008 [boundary]
// MCDC SYS-REQ-008: N/A
func TestArrayOutOfBounds(t *testing.T) {
	var sb strings.Builder
	//build data
	sb.WriteString(`[`)
	for i := 0; i < 61; i++ {
		fmt.Fprintf(&sb, `%d`, i)
		if i < 61 {
			sb.WriteString(",")
		}
	}
	sb.WriteString(`]`)
	keys := [][]string{[]string{`[128]`}}

	data := []byte(sb.String())
	EachKey(data, func(offset int, value []byte, dt ValueType, err error) {
		return
	}, keys...)
}
