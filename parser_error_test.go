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

func testIter(data []byte) (err error) {
	EachKey(data, func(idx int, value []byte, vt ValueType, iterErr error) {
		if iterErr != nil {
			err = fmt.Errorf("Error parsing json: %s", iterErr.Error())
		}
	}, testPaths...)
	return err
}

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

// check having a very deep key depth
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
