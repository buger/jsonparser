package jsonparser

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

// Verifies: SYS-REQ-116 (incremental path lookup)
// SYS-REQ-116:nominal:nominal
// reqproof:proptest:skip targeted streaming witness; not a property-test subject
func TestReaderParserGet(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		keys     []string
		want     string
		wantType ValueType
	}{
		{
			name:     "nested number",
			data:     `{"person":{"name":"Ada","age":37}}`,
			keys:     []string{"person", "age"},
			want:     "37",
			wantType: Number,
		},
		{
			name:     "object",
			data:     `{"person":{"name":"Ada","age":37}}`,
			keys:     []string{"person"},
			want:     `{"name":"Ada","age":37}`,
			wantType: Object,
		},
		{
			name:     "array index",
			data:     `{"people":[{"name":"Ada"},{"name":"Grace"}]}`,
			keys:     []string{"people", "[1]", "name"},
			want:     "Grace",
			wantType: String,
		},
		{
			name:     "root value",
			data:     ` [1,{"ok":true}] trailing bytes are not needed`,
			want:     `[1,{"ok":true}]`,
			wantType: Array,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := NewReaderParser(bytes.NewReader([]byte(tt.data)))
			value, vt, err := rp.Get(tt.keys...)
			if err != nil {
				t.Fatalf("Get returned error: %v", err)
			}
			if vt != tt.wantType {
				t.Fatalf("Get type = %v, want %v", vt, tt.wantType)
			}
			if got := string(value); got != tt.want {
				t.Fatalf("Get value = %q, want %q", got, tt.want)
			}
		})
	}

	rp := NewReaderParser(strings.NewReader(`{"present":1}`))
	if _, vt, err := rp.Get("missing"); !errors.Is(err, KeyPathNotFoundError) || vt != NotExist {
		t.Fatalf("missing Get = type %v, error %v; want NotExist, KeyPathNotFoundError", vt, err)
	}
}

// Verifies: SYS-REQ-116 (streaming string decoding)
// SYS-REQ-116:nominal:nominal
// reqproof:proptest:skip targeted streaming witness; not a property-test subject
func TestReaderParserGetString(t *testing.T) {
	rp := NewReaderParser(strings.NewReader(`{"message":"hello\n\u263a"}`))
	got, err := rp.GetString("message")
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if want := "hello\n☺"; got != want {
		t.Fatalf("GetString = %q, want %q", got, want)
	}

	rp = NewReaderParser(strings.NewReader(`{"message":42}`))
	if _, err := rp.GetString("message"); err == nil {
		t.Fatal("GetString accepted a non-string value")
	}
}

// Verifies: SYS-REQ-116 (incremental array iteration)
// SYS-REQ-116:nominal:nominal
// reqproof:proptest:skip targeted streaming witness; not a property-test subject
func TestReaderParserArrayEach(t *testing.T) {
	const elements = 20000
	var source strings.Builder
	source.WriteByte('[')
	for i := 0; i < elements; i++ {
		if i > 0 {
			source.WriteByte(',')
		}
		source.WriteString(strconv.Itoa(i))
	}
	source.WriteByte(']')

	rp := NewReaderParser(strings.NewReader(source.String()))
	count := 0
	err := rp.ArrayEach(func(value []byte, vt ValueType, err error) {
		if err != nil {
			t.Errorf("callback %d returned error: %v", count, err)
			return
		}
		if vt != Number {
			t.Errorf("callback %d type = %v, want Number", count, vt)
			return
		}
		if got, want := string(value), strconv.Itoa(count); got != want {
			t.Errorf("callback %d value = %q, want %q", count, got, want)
		}
		count++
	})
	if err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
	if count != elements {
		t.Fatalf("ArrayEach callbacks = %d, want %d", count, elements)
	}
	if len(rp.buf) > defaultReaderBufferSize {
		t.Fatalf("sliding buffer retained %d bytes after iteration, want at most %d", len(rp.buf), defaultReaderBufferSize)
	}
}

// Verifies: SYS-REQ-116 (bounded window over a large stream)
// SYS-REQ-116:boundary:nominal
// reqproof:proptest:skip deterministic 100 MiB streaming boundary witness
func TestReaderParserLargeFile(t *testing.T) {
	const paddingSize = int64(100 * 1024 * 1024)
	const maxBufferSize = 32 * 1024

	source := io.MultiReader(
		strings.NewReader(`{"padding":"`),
		&repeatedByteReader{remaining: paddingSize, value: 'x'},
		strings.NewReader(`","target":"found"}`),
	)
	rp := NewReaderParser(source, Config{MaxBufferSize: maxBufferSize})

	value, vt, err := rp.Get("target")
	if err != nil {
		t.Fatalf("Get after 100 MiB field returned error: %v", err)
	}
	if vt != String || string(value) != "found" {
		t.Fatalf("Get after 100 MiB field = (%q, %v), want (found, String)", value, vt)
	}
	if rp.bufStart < paddingSize {
		t.Fatalf("sliding buffer discarded through offset %d, want at least %d", rp.bufStart, paddingSize)
	}
	if cap(rp.buf) > 2*maxBufferSize {
		t.Fatalf("sliding buffer capacity = %d, want <= %d", cap(rp.buf), 2*maxBufferSize)
	}
}

// Verifies: SYS-REQ-116 (malformed and truncated streams)
// SYS-REQ-116:malformed_input:nominal
// reqproof:proptest:skip targeted malformed-stream witness
func TestReaderParserMalformed(t *testing.T) {
	rp := NewReaderParser(strings.NewReader(`{"message":"unterminated`))
	if _, _, err := rp.Get("message"); !errors.Is(err, MalformedStringError) {
		t.Fatalf("truncated string error = %v, want MalformedStringError", err)
	}

	rp = NewReaderParser(strings.NewReader(`[1,{"open":true}`))
	if err := rp.ArrayEach(func([]byte, ValueType, error) {}); !errors.Is(err, MalformedArrayError) {
		t.Fatalf("truncated array error = %v, want MalformedArrayError", err)
	}
}

// Verifies: SYS-REQ-116 (tokens crossing read boundaries)
// SYS-REQ-116:boundary:nominal
// reqproof:proptest:skip targeted chunk-boundary witness
func TestReaderParserChunkBoundary(t *testing.T) {
	source := &fixedChunkReader{
		data:  []byte(`{"prefix":0,"message":"a value crossing many tiny chunks","suffix":1}`),
		chunk: 3,
	}
	rp := NewReaderParser(source)
	value, vt, err := rp.Get("message")
	if err != nil {
		t.Fatalf("chunked Get returned error: %v", err)
	}
	if vt != String || string(value) != "a value crossing many tiny chunks" {
		t.Fatalf("chunked Get = (%q, %v)", value, vt)
	}
}

// Verifies: SYS-REQ-116 (config-aware streaming)
// SYS-REQ-116:nominal:nominal
// reqproof:proptest:skip targeted config witness
func TestReaderParserConfig(t *testing.T) {
	rp := NewReaderParser(
		strings.NewReader(`{'message':'it\'s\q'}`),
		Config{
			AllowSingleQuotes:   true,
			AllowUnknownEscapes: true,
			MaxBufferSize:       8,
		},
	)

	got, err := rp.GetString("message")
	if err != nil {
		t.Fatalf("lenient GetString returned error: %v", err)
	}
	if want := "it'sq"; got != want {
		t.Fatalf("lenient GetString = %q, want %q", got, want)
	}
}

// Verifies: SYS-REQ-116 (empty input)
// SYS-REQ-116:empty_input:nominal
// reqproof:proptest:skip targeted empty-stream witness
func TestReaderParserEmpty(t *testing.T) {
	rp := NewReaderParser(bytes.NewReader(nil))
	if _, vt, err := rp.Get("key"); !errors.Is(err, KeyPathNotFoundError) || vt != NotExist {
		t.Fatalf("empty Get = type %v, error %v; want NotExist, KeyPathNotFoundError", vt, err)
	}
}

type repeatedByteReader struct {
	remaining int64
	value     byte
}

func (r *repeatedByteReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = r.value
	}
	r.remaining -= int64(len(p))
	return len(p), nil
}

type fixedChunkReader struct {
	data  []byte
	pos   int
	chunk int
}

func (r *fixedChunkReader) Read(p []byte) (int, error) {
	if r.pos == len(r.data) {
		return 0, io.EOF
	}
	if len(p) > r.chunk {
		p = p[:r.chunk]
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
