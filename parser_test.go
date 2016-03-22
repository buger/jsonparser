package jsonparser

import (
	"bytes"
	_ "fmt"
	"reflect"
	"testing"
)

func toArray(data []byte) (result [][]byte) {
	ArrayEach(data, func(value []byte, dataType int, offset int, err error) {
		result = append(result, value)
	})

	return
}

func TestValidJSON(t *testing.T) {
	if v, _, _, err := Get([]byte(`{"a": { "b": 1}, "c": 2 }`), "a", "b", "c"); err == nil {
		t.Errorf("Should apply scope of parent when search for nested key: %s, %v", string(v), err)
	}

	if v, _, _, _ := Get([]byte(`{"a": { "b":{"c":"d" }}}`), "a", "b", "c"); !bytes.Equal(v, []byte("d")) {
		t.Errorf("Should read composite key", v)
	}

	if v, _, _, err := Get([]byte(`{"a": { "b": 1}, "c": 2 }`), "a", "b", "c"); err == nil {
		t.Errorf("Should apply scope of parent when search for nested key: %s, %v", string(v), err)
	}

	if v, _, _, e := Get([]byte(`{"a":"b"}`), "a"); !bytes.Equal(v, []byte("b")) {
		t.Errorf("Should read basic key %s %v", string(v), e)
	}

	if v, _, _, _ := Get([]byte(`{"a": "b"}`), "a"); !bytes.Equal(v, []byte("b")) {
		t.Errorf("Should read basic key with space", v)
	}

	if v, _, _, _ := Get([]byte(`{"a": "b", "c": 1}`), "c"); !bytes.Equal(v, []byte("1")) {
		t.Errorf("Should read numberic value as string", v)
	}

	if v, _, _, _ := Get([]byte(`{"a": "string\"with\"quotes"}`), "a"); !bytes.Equal(v, []byte(`string\"with\"quotes`)) {
		t.Errorf("Should read string values with quotes", string(v))
	}

	if v, _, _ := GetNumber([]byte(`{"a": "b", "c": 1}`), "c"); v != 1 {
		t.Errorf("Should read numberic value as number", v)
	}

	if v, _, _, _ := Get([]byte(`{"a": { "b":{"c":"d" }}}`), "a", "b", "c"); !bytes.Equal(v, []byte("d")) {
		t.Errorf("Should read composite key", v)
	}

	if v, _, _, _ := Get([]byte(`{"a": { "b":{"c":"d" }}}`), "a", "b"); !bytes.Equal(v, []byte(`{"c":"d" }`)) {
		t.Errorf("Should read object", v)
	}

	if v, _, _, _ := Get([]byte(`, "a": { "b":{"c":"d" }}}`), "a", "b"); !bytes.Equal(v, []byte(`{"c":"d" }`)) {
		t.Errorf("Should handle some JSON issues", v)
	}

	if v, _, _, _ := Get([]byte(`{"c":"d" }`)); !bytes.Equal(v, []byte(`{"c":"d" }`)) {
		t.Errorf("Should handle empty path", v)
	}

	if v, _, _, _ := Get([]byte(`{"a": { "b":[1,2,3,4]}}`), "a", "b"); !reflect.DeepEqual(toArray(v), [][]byte{[]byte("1"), []byte("2"), []byte("3"), []byte(`4`)}) {
		t.Errorf("Should read array of simple values: %s, %v", string(v), toArray(v))
	}

	if v, _, _, _ := Get([]byte(`[1,2,3,4]`)); !reflect.DeepEqual(toArray(v), [][]byte{[]byte("1"), []byte("2"), []byte("3"), []byte(`4`)}) {
		t.Errorf("Should parse array without specifying path: %s %v", string(v), toArray(v))
	}

	if v, _, _, _ := Get([]byte(`{"a": { "b":[{"x":1},{"x":2},{"x":3},{"x":4}]}}`), "a", "b"); !reflect.DeepEqual(toArray(v), [][]byte{[]byte(`{"x":1}`), []byte(`{"x":2}`), []byte(`{"x":3}`), []byte(`{"x":4}`)}) {
		t.Errorf("Should read array of objects", v)
	}

	if v, _, _, _ := Get([]byte(`{"a": [[[1]],[[2]]]}`), "a"); !reflect.DeepEqual(toArray(v), [][]byte{[]byte("[[1]]"), []byte("[[2]]")}) {
		t.Errorf("Should parse nested array", v)
	}

	if v, _, _, _ := Get([]byte("{\n  \"a\": \"b\"\n}"), "a"); !bytes.Equal(v, []byte("b")) {
		t.Errorf("Should read formated json value %s", string(v))
	}

	if v, _, _, e := Get([]byte("{\n  \"a\":\n    {\n\"b\":\n   {\"c\":\"d\",\n\"e\": \"f\"}\n}\n}"), "a", "b"); !bytes.Equal(v, []byte("{\"c\":\"d\",\n\"e\": \"f\"}")) {
		t.Errorf("Should read formated json object", string(v), e)
	}
}

func TestInvalidJSON(t *testing.T) {
	if _, _, _, e := Get([]byte(`{"a":"b"`), "c"); e == nil || e.Error() != "Key path not found" {
		t.Errorf("Should not found key: ", e)
	}

	if v, _, _, e := Get([]byte(`{"a":"b"`), "a"); !bytes.Equal(v, []byte("b")) || e != nil {
		t.Errorf("Should not found missing bracket, because key still found: %s", string(v))
	}

	if _, _, _, e := Get([]byte(`{"a":"b`), "a"); e == nil || e.Error() != "Value is string, but can't find closing '\"' symbol" {
		t.Errorf("Should raise error since end of string not found: ", e)
	}

	if v, _, _, e := Get([]byte(`{"a": { "b": "c"`), "a"); e == nil || e.Error() != "Value looks like object, but can't find closing '}' symbol" {
		t.Errorf("Should raise error if closing brace not found: ", e, string(v))
	}

	if v, _, _, e := Get([]byte(`{"a": [1, 2, 3 }`), "a"); e == nil || e.Error() != "Value is array, but can't find closing ']' symbol" {
		t.Errorf("Should raise error if closing bracket not found: ", e, string(v))
	}

	if _, _, _, e := Get([]byte(`{"a": `), "a"); e == nil || e.Error() != "Malformed JSON error" {
		t.Errorf("Should raise malformed json error: ", e)
	}
}
