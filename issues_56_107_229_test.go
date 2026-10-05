package jsonparser

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// Verifies: SYS-REQ-010 (Delete)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestDeleteFound(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		keys  []string
		want  string
		found bool
	}{
		{
			name:  "object key",
			data:  `{"a":1,"b":2}`,
			keys:  []string{"a"},
			want:  `{"b":2}`,
			found: true,
		},
		{
			name:  "nested key",
			data:  `{"a":{"b":1}}`,
			keys:  []string{"a", "b"},
			want:  `{"a":{}}`,
			found: true,
		},
		{
			name:  "array element",
			data:  `[1,2,3]`,
			keys:  []string{"[1]"},
			want:  `[1,3]`,
			found: true,
		},
		{
			name:  "missing key",
			data:  `{"a":1}`,
			keys:  []string{"b"},
			want:  `{"a":1}`,
			found: false,
		},
		{
			name:  "malformed value",
			data:  `{"a":`,
			keys:  []string{"a"},
			want:  `{"a":`,
			found: false,
		},
		{
			name:  "root",
			data:  `{"a":1}`,
			want:  ``,
			found: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := []byte(test.data)
			original := append([]byte(nil), data...)

			got, found := DeleteFound(data, test.keys...)
			if found != test.found {
				t.Fatalf("DeleteFound found = %v; want %v", found, test.found)
			}
			if string(got) != test.want {
				t.Fatalf("DeleteFound result = %q; want %q", got, test.want)
			}
			if !bytes.Equal(data, original) {
				t.Fatalf("DeleteFound mutated input: got %q; want %q", data, original)
			}

			if !found && len(data) > 0 && &got[0] != &data[0] {
				t.Fatal("DeleteFound returned a copy when the key was not found")
			}
			if deleteResult := Delete(data, test.keys...); !bytes.Equal(deleteResult, got) {
				t.Fatalf("Delete result = %q; DeleteFound result = %q", deleteResult, got)
			}
		})
	}
}

// Verifies: SYS-REQ-008 (EachKey)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEachKeyPathWith100Components(t *testing.T) {
	testEachKeyPathDepth(t, 100)
}

// Verifies: SYS-REQ-008 (EachKey)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEachKeyPathBeyondStackCapacity(t *testing.T) {
	testEachKeyPathDepth(t, stackArraySize+1)
}

func testEachKeyPathDepth(t *testing.T, depth int) {
	t.Helper()
	path := make([]string, depth)
	var data strings.Builder
	for i := range path {
		path[i] = "key" + strconv.Itoa(i)
		data.WriteString(`{"`)
		data.WriteString(path[i])
		data.WriteString(`":`)
	}
	data.WriteString(`"found"`)
	for range path {
		data.WriteByte('}')
	}

	callbacks := 0
	offset := EachKey([]byte(data.String()), func(idx int, value []byte, valueType ValueType, err error) {
		callbacks++
		if err != nil {
			t.Errorf("EachKey callback returned error: %v", err)
		}
		if idx != 0 {
			t.Errorf("EachKey callback index = %d; want 0", idx)
		}
		if valueType != String {
			t.Errorf("EachKey value type = %v; want %v", valueType, String)
		}
		if string(value) != "found" {
			t.Errorf("EachKey value = %q; want %q", value, "found")
		}
	}, path)

	if offset < 0 {
		t.Fatalf("EachKey did not find the %d-component path", depth)
	}
	if callbacks != 1 {
		t.Fatalf("EachKey callbacks = %d; want 1", callbacks)
	}
}

// Verifies: SYS-REQ-009 (Set)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestCreateInsertComponentUsesCalculatedCapacity(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		setValue string
		comma    bool
		object   bool
		want     string
	}{
		{
			name:     "object key",
			keys:     []string{"key"},
			setValue: `1`,
			object:   true,
			want:     `{"key":1}`,
		},
		{
			name:     "nested object key",
			keys:     []string{"a", "b"},
			setValue: `true`,
			comma:    true,
			want:     `,"a":{"b":true}`,
		},
		{
			name:     "array root",
			keys:     []string{"[3]"},
			setValue: `"value"`,
			want:     `["value"]`,
		},
		{
			name:     "nested array",
			keys:     []string{"items", "[2]"},
			setValue: `null`,
			object:   true,
			want:     `{"items":[null]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValue := []byte(test.setValue)
			size := calcAllocateSpace(test.keys, setValue, test.comma, test.object)
			got := createInsertComponent(test.keys, setValue, test.comma, test.object)

			if string(got) != test.want {
				t.Fatalf("createInsertComponent result = %q; want %q", got, test.want)
			}
			if len(got) != size {
				t.Fatalf("createInsertComponent length = %d; calculated size = %d", len(got), size)
			}
			if cap(got) != size {
				t.Fatalf("createInsertComponent capacity = %d; calculated size = %d", cap(got), size)
			}
		})
	}
}

var benchmarkSetInsertResult []byte

// BenchmarkSet models Set's former incrementally-grown bytes.Buffer.
//
// Verifies: SYS-REQ-009 (Set)
func BenchmarkSet(b *testing.B) {
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = "key" + strconv.Itoa(i)
	}
	setValue := bytes.Repeat([]byte("value"), 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSetInsertResult = createInsertComponentBuffered(keys, setValue, false, true)
	}
}

// BenchmarkSetPreAllocated measures Set's calcAllocateSpace-backed component
// construction.
//
// Verifies: SYS-REQ-009 (Set)
func BenchmarkSetPreAllocated(b *testing.B) {
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = "key" + strconv.Itoa(i)
	}
	setValue := bytes.Repeat([]byte("value"), 100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSetInsertResult = createInsertComponent(keys, setValue, false, true)
	}
}

func createInsertComponentBuffered(keys []string, setValue []byte, comma, object bool) []byte {
	var buffer bytes.Buffer
	isIndex := len(keys[0]) > 0 && keys[0][0] == '['

	if comma {
		buffer.WriteByte(',')
	}
	if isIndex && !comma {
		buffer.WriteByte('[')
	} else {
		if object {
			buffer.WriteByte('{')
		}
		if !isIndex {
			buffer.WriteByte('"')
			buffer.WriteString(keys[0])
			buffer.WriteString(`":`)
		}
	}

	for i := 1; i < len(keys); i++ {
		if len(keys[i]) > 0 && keys[i][0] == '[' {
			buffer.WriteByte('[')
		} else {
			buffer.WriteString(`{"`)
			buffer.WriteString(keys[i])
			buffer.WriteString(`":`)
		}
	}
	buffer.Write(setValue)
	for i := len(keys) - 1; i > 0; i-- {
		if len(keys[i]) > 0 && keys[i][0] == '[' {
			buffer.WriteByte(']')
		} else {
			buffer.WriteByte('}')
		}
	}
	if isIndex && !comma {
		buffer.WriteByte(']')
	}
	if object && !isIndex {
		buffer.WriteByte('}')
	}

	return buffer.Bytes()
}

var benchmarkEachKeyResult int
var benchmarkEachKeyValue []byte

func benchDenseArrayFixture(n int) ([]byte, [][]string) {
	var buf bytes.Buffer
	buf.WriteString(`{"array":[`)
	paths := make([][]string, n)
	for i := 0; i < n; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(`{"v":`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteByte('}')
		paths[i] = []string{"array", "[" + strconv.Itoa(i) + "]", "v"}
	}
	buf.WriteString(`]}`)
	return buf.Bytes(), paths
}

func benchIdleArraysFixture(fields int) ([]byte, [][]string) {
	var buf bytes.Buffer
	buf.WriteString(`{"o":{`)
	paths := make([][]string, fields)
	for i := 0; i < fields; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(`"k`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(`":[1,2,3,4,5]`)
		paths[i] = []string{"o", "k" + strconv.Itoa(i)}
	}
	buf.WriteString(`}}`)
	return buf.Bytes(), paths
}

func benchSharedTerminalFixture(n int) ([]byte, [][]string) {
	paths := make([][]string, n)
	for i := range paths {
		paths[i] = []string{"array", "[0]"}
	}
	return []byte(`{"array":["x"]}`), paths
}

// BenchmarkEachKeyDenseArray measures bulk extraction of every element of an
// array of objects by index, the core EachKey workload.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeyDenseArray128(b *testing.B) {
	data, paths := benchDenseArrayFixture(128)
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// Verifies: SYS-REQ-008
func BenchmarkEachKeyDenseArray1024(b *testing.B) {
	data, paths := benchDenseArrayFixture(1024)
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// BenchmarkEachKeySparseArray measures two wanted indexes inside a large array.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeySparseArray(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := 0; i < 16384; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(strconv.Itoa(i))
	}
	buf.WriteByte(']')
	data := buf.Bytes()
	paths := [][]string{{"3"}, {"16380"}}
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// BenchmarkEachKeyIdleArrays measures array-heavy documents whose paths never
// target an array: every array must be skipped at minimal cost.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeyIdleArrays62(b *testing.B) {
	data, paths := benchIdleArraysFixture(62)
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// Verifies: SYS-REQ-008
func BenchmarkEachKeyIdleArrays128(b *testing.B) {
	data, paths := benchIdleArraysFixture(128)
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// BenchmarkEachKeySharedTerminal measures many identical terminal paths
// requesting one index of a one-element array, the worst case for grouping
// registration.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeySharedTerminal128(b *testing.B) {
	data, paths := benchSharedTerminalFixture(128)
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// Verifies: SYS-REQ-008
func BenchmarkEachKeySharedTerminal1024(b *testing.B) {
	data, paths := benchSharedTerminalFixture(1024)
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// BenchmarkEachKeyFlatObject measures the object branch alone, which must be
// unaffected by array bookkeeping.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeyFlatObject62(b *testing.B) {
	data, paths := benchIdleArraysFixture(0)
	var buf bytes.Buffer
	buf.WriteString(`{`)
	paths = paths[:0]
	for i := 0; i < 62; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(`"k`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(`":`)
		buf.WriteString(strconv.Itoa(i))
		paths = append(paths, []string{"k" + strconv.Itoa(i)})
	}
	buf.WriteByte('}')
	data = buf.Bytes()
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}

// BenchmarkEachKeyErrDenseArray mirrors the dense workload for the erroring
// variant.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeyErrDenseArray128(b *testing.B) {
	data, paths := benchDenseArrayFixture(128)
	cb := func(i int, v []byte, vt ValueType, e error) error { benchmarkEachKeyValue = v; return nil }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := EachKeyErr(data, cb, paths...); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEachKeySparseArrayObjects mirrors the sparse workload with object
// elements, the shape real payloads use.
//
// Verifies: SYS-REQ-008
func BenchmarkEachKeySparseArrayObjects(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := 0; i < 16384; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(`{"i":`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(`,"s":"value"}`)
	}
	buf.WriteByte(']')
	data := buf.Bytes()
	paths := [][]string{{"3", "i"}, {"16380", "i"}}
	cb := func(i int, v []byte, vt ValueType, e error) { benchmarkEachKeyValue = v }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkEachKeyResult = EachKey(data, cb, paths...)
	}
}
