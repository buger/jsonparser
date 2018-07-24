package jsonparser

import (
	"strconv"
	"testing"
	"unsafe"
)

type ParseIntTest struct {
	in         string
	out        int64
	isErr      bool
	isOverflow bool
}

var parseIntTests = []ParseIntTest{
	{
		in:  "0",
		out: 0,
	},
	{
		in:  "1",
		out: 1,
	},
	{
		in:  "-1",
		out: -1,
	},
	{
		in:  "12345",
		out: 12345,
	},
	{
		in:  "-12345",
		out: -12345,
	},
	{
		in:  "9223372036854775807", // = math.MaxInt64
		out: 9223372036854775807,
	},
	{
		in:  "-9223372036854775808", // = math.MinInt64
		out: -9223372036854775808,
	},
	{
		in:         "-92233720368547758081",
		out:        0,
		isErr:      true,
		isOverflow: true,
	},
	{
		in:         "18446744073709551616", // = 2^64
		out:        0,
		isErr:      true,
		isOverflow: true,
	},
	{
		in:         "9223372036854775808", // = math.MaxInt64 - 1
		out:        0,
		isErr:      true,
		isOverflow: true,
	},
	{
		in:         "-9223372036854775809", // = math.MaxInt64 - 1
		out:        0,
		isErr:      true,
		isOverflow: true,
	},
	{
		in:    "",
		isErr: true,
	},
	{
		in:    "abc",
		isErr: true,
	},
	{
		in:    "12345x",
		isErr: true,
	},
	{
		in:    "123e5",
		isErr: true,
	},
	{
		in:    "9223372036854775807x",
		isErr: true,
	},
}

func TestBytesParseInt(t *testing.T) {
	for _, test := range parseIntTests {
		out, ok, overflow := parseInt([]byte(test.in))
		if overflow != test.isOverflow {
			t.Errorf("Test '%s' error return did not overflow expectation (obtained %t, expected %t)", test.in, overflow, test.isOverflow)
		}
		if ok != !test.isErr {
			t.Errorf("Test '%s' error return did not match expectation (obtained %t, expected %t)", test.in, !ok, test.isErr)
		} else if ok && out != test.out {
			t.Errorf("Test '%s' did not return the expected value (obtained %d, expected %d)", test.in, out, test.out)
		}
	}
}

func BenchmarkParseInt(b *testing.B) {
	bytes := []byte("123")
	for i := 0; i < b.N; i++ {
		parseInt(bytes)
	}
}

// Alternative implementation using unsafe and delegating to strconv.ParseInt
func BenchmarkParseIntUnsafeSlower(b *testing.B) {
	bytes := []byte("123")
	for i := 0; i < b.N; i++ {
		strconv.ParseInt(*(*string)(unsafe.Pointer(&bytes)), 10, 64)
	}
}

// Old implementation that did not check for overflows.
func BenchmarkParseIntOverflows(b *testing.B) {
	bytes := []byte("123")
	for i := 0; i < b.N; i++ {
		parseIntOverflows(bytes)
	}
}

func parseIntOverflows(bytes []byte) (v int64, ok bool) {
	if len(bytes) == 0 {
		return 0, false
	}

	var neg bool = false
	if bytes[0] == '-' {
		neg = true
		bytes = bytes[1:]
	}

	for _, c := range bytes {
		if c >= '0' && c <= '9' {
			v = (10 * v) + int64(c-'0')
		} else {
			return 0, false
		}
	}

	if neg {
		return -v, true
	} else {
		return v, true
	}
}
