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

func toStringArray(data []byte) (result []string) {
	ArrayEach(data, func(value []byte, dataType int, offset int, err error) {
		result = append(result, string(value))
	})

	return
}

type GetTest struct {
	desc string
	json string
	path []string

	isErr   bool
	isFound bool

	data interface{}
}

var getTests = []GetTest{
	// Found key tests
	GetTest{
		desc:    "handling multiple nested keys with same name",
		json:    `{"a":[{"b":1},{"b":2},3],"c":{"c":[1,2]}} }`,
		path:    []string{"c", "c"},
		isFound: true,
		data:    `[1,2]`,
	},
	GetTest{
		desc:    "read basic key",
		json:    `{"a":"b"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	GetTest{
		desc:    "read basic key with space",
		json:    `{"a": "b"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	GetTest{
		desc:    "read composite key",
		json:    `{"a": { "b":{"c":"d" }}}`,
		path:    []string{"a", "b", "c"},
		isFound: true,
		data:    `d`,
	},
	GetTest{
		desc:    `read numberic value as string`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    `1`,
	},
	GetTest{
		desc:    `handle multiple nested keys with same name`,
		json:    `{"a":[{"b":1},{"b":2},3],"c":{"c":[1,2]}} }`,
		path:    []string{"c", "c"},
		isFound: true,
		data:    `[1,2]`,
	},
	GetTest{
		desc:    `read string values with quotes`,
		json:    `{"a": "string\"with\"quotes"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `string\"with\"quotes`,
	},
	GetTest{
		desc:    `read object`,
		json:    `{"a": { "b":{"c":"d" }}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    `{"c":"d" }`,
	},
	GetTest{
		desc:    `empty path`,
		json:    `{"c":"d" }`,
		path:    []string{},
		isFound: true,
		data:    `{"c":"d" }`,
	},
	GetTest{
		desc:    `formatted JSON value`,
		json:    "{\n  \"a\": \"b\"\n}",
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	GetTest{
		desc:    `formatted JSON value 2`,
		json:    "{\n  \"a\":\n    {\n\"b\":\n   {\"c\":\"d\",\n\"e\": \"f\"}\n}\n}",
		path:    []string{"a", "b"},
		isFound: true,
		data:    "{\"c\":\"d\",\n\"e\": \"f\"}",
	},
	GetTest{
		desc:    `whitespace`,
		json:    " \n\r\t{ \n\r\t\"whitespace\" \n\r\t: \n\r\t333 \n\r\t} \n\r\t",
		path:    []string{"whitespace"},
		isFound: true,
		data:    "333",
	},

	// Not found key tests
	GetTest{
		desc:    "non-existent key 1",
		json:    `{"a":"b"}`,
		path:    []string{"c"},
		isFound: false,
	},
	GetTest{
		desc:    "non-existent key 2",
		json:    `{"a":"b"}`,
		path:    []string{"b"},
		isFound: false,
	},
	GetTest{
		desc:    "non-existent key 3",
		json:    `{"aa":"b"}`,
		path:    []string{"a"},
		isFound: false,
	},
	GetTest{
		desc:    "apply scope of parent when search for nested key",
		json:    `{"a": { "b": 1}, "c": 2 }`,
		path:    []string{"a", "b", "c"},
		isFound: false,
	},
	GetTest{
		desc:    `apply scope to key level`,
		json:    `{"a": { "b": 1}, "c": 2 }`,
		path:    []string{"b"},
		isFound: false,
	},
	GetTest{
		desc:    `handle escaped quote in key name in JSON`,
		json:    `{"key\"key": 1}`,
		path:    []string{"key"},
		isFound: false,
	},

	// Error/invalid tests
	GetTest{
		desc:    `handle escaped quote in key name in JSON`,
		json:    `{"key\"key": 1}`,
		path:    []string{"key"},
		isFound: false,
	},
	GetTest{
		desc:    `missing closing brace, but can still find key`,
		json:    `{"a":"b"`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	GetTest{
		desc:  `missing value closing quote`,
		json:  `{"a":"b`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  `missing value closing curly brace`,
		json:  `{"a": { "b": "c"`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  `missing value closing square bracket`,
		json:  `{"a": [1, 2, 3 }`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  `missing value 1`,
		json:  `{"a":`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  `missing value 2`,
		json:  `{"a": `,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  `missing value 3`,
		json:  `{"a":}`,
		path:  []string{"a"},
		isErr: true,
	},
}

var getNumberTests = []GetTest{
	GetTest{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    float64(1),
	},
	GetTest{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 1 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    float64(1),
	},
	GetTest{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 1 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    float64(1),
	},
}

var getBoolTests = []GetTest{
	GetTest{
		desc:    `read boolean true as boolean`,
		json:    `{"a": "b", "c": true}`,
		path:    []string{"c"},
		isFound: true,
		data:    true,
	},
	GetTest{
		desc:    `boolean true in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": true \n}",
		path:    []string{"c"},
		isFound: true,
		data:    true,
	},
	GetTest{
		desc:    `read boolean false as boolean`,
		json:    `{"a": "b", "c": false}`,
		path:    []string{"c"},
		isFound: true,
		data:    false,
	},
	GetTest{
		desc:    `boolean true in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": false \n}",
		path:    []string{"c"},
		isFound: true,
		data:    false,
	},
}

var getSliceTests = []GetTest{
	GetTest{
		desc:    `read array of simple values`,
		json:    `{"a": { "b":[1,2,3,4]}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    []string{`1`, `2`, `3`, `4`},
	},
	GetTest{
		desc:    `read array via empty path`,
		json:    `[1,2,3,4]`,
		path:    []string{},
		isFound: true,
		data:    []string{`1`, `2`, `3`, `4`},
	},
	GetTest{
		desc:    `read array of objects`,
		json:    `{"a": { "b":[{"x":1},{"x":2},{"x":3},{"x":4}]}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    []string{`{"x":1}`, `{"x":2}`, `{"x":3}`, `{"x":4}`},
	},
	GetTest{
		desc:    `read nested array`,
		json:    `{"a": [[[1]],[[2]]]}`,
		path:    []string{"a"},
		isFound: true,
		data:    []string{`[[1]]`, `[[2]]`},
	},
}

// checkFoundAndNoError checks the dataType and error return from Get*() against the test case expectations.
// Returns true the test should proceed to checking the actual data returned from Get*(), or false if the test is finished.
func checkFoundAndNoError(t *testing.T, testKind string, test GetTest, jtype int, err error) bool {
	isFound := (jtype != NotExist)
	isErr := (err != nil)

	if test.isErr != isErr {
		// If the call didn't match the error expectation, fail
		t.Errorf("%s test '%s' isErr mismatch: expected %t, obtained %t (err %v)", testKind, test.desc, test.isErr, isErr, err)
		return false
	} else if isErr {
		// Else, if there was an error, don't fail and don't check isFound or the value
		return false
	} else if test.isFound != isFound {
		// Else, if the call didn't match the is-found expectation, fail
		t.Errorf("%s test '%s' isFound mismatch: expected %t, obtained %t", testKind, test.desc, test.isFound, isFound)
		return false
	} else if !isFound {
		// Else, if no value was found, don't fail and don't check the value
		return false
	} else {
		// Else, there was no error and a value was found, so check the value
		return true
	}
}

func TestGet(t *testing.T) {
	for _, gt := range getTests {
		v, jtype, _, err := Get([]byte(gt.json), gt.path...)

		if checkFoundAndNoError(t, "Get()", gt, jtype, err) {
			if gt.data == nil {
				t.Errorf("MALFORMED TEST: %v", gt)
				continue
			}

			expectedData := []byte(gt.data.(string))
			if !bytes.Equal(expectedData, v) {
				t.Errorf("Get() test '%s' expected to return value %v, but did returned %v instead", string(expectedData), string(v))
			}
		}
	}
}

func TestGetNumber(t *testing.T) {
	for _, gnt := range getNumberTests {
		v, _, err := GetNumber([]byte(gnt.json), gnt.path...)

		if checkFoundAndNoError(t, "GetNumber()", gnt, Number, err) {
			if gnt.data == nil {
				t.Errorf("MALFORMED TEST: %v", gnt)
				continue
			} else if _, ok := gnt.data.(float64); !ok {
				t.Errorf("MALFORMED TEST: %v", gnt)
				continue
			}

			expectedData := gnt.data.(float64)
			if expectedData != v {
				t.Errorf("GetNumber() test '%s' expected to return value %v, but did returned %v instead", expectedData, v)
			}
		}
	}
}

func TestGetBool(t *testing.T) {
	for _, gnt := range getBoolTests {
		v, _, err := GetBoolean([]byte(gnt.json), gnt.path...)

		if checkFoundAndNoError(t, "GetBoolean()", gnt, Number, err) {
			if gnt.data == nil {
				t.Errorf("MALFORMED TEST: %v", gnt)
				continue
			} else if _, ok := gnt.data.(bool); !ok {
				t.Errorf("MALFORMED TEST: %v", gnt)
				continue
			}

			expectedData := gnt.data.(bool)
			if expectedData != v {
				t.Errorf("GetBoolean() test '%s' expected to return value %v, but did returned %v instead", expectedData, v)
			}
		}
	}
}

func TestGetSlice(t *testing.T) {
	for _, gst := range getSliceTests {
		v, jtype, _, err := Get([]byte(gst.json), gst.path...)

		if checkFoundAndNoError(t, "Get()", gst, jtype, err) {
			if gst.data == nil {
				t.Errorf("MALFORMED TEST: %v", gst)
				continue
			} else if _, ok := gst.data.([]string); !ok {
				t.Errorf("MALFORMED TEST: %v", gst)
				continue
			}

			expectedData := gst.data.([]string)
			vslice := toStringArray(v)
			if !reflect.DeepEqual(expectedData, vslice) {
				t.Errorf("Get() test '%s' expected to return value %v, but did returned %v instead", expectedData, v)
			}
		}
	}
}
