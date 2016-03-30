package jsonparser

import (
	"bytes"
	_ "fmt"
	"reflect"
	"testing"
)

// Set it to non-empty value if want to run only specific test
var activeTest = ""

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

type Test struct {
	desc string
	json string
	path []string

	isErr   bool
	isFound bool

	data interface{}
}

var getTests = []Test{
	// Found key tests
	Test{
		desc:    "handling multiple nested keys with same name",
		json:    `{"a":[{"b":1},{"b":2},3],"c":{"c":[1,2]}} }`,
		path:    []string{"c", "c"},
		isFound: true,
		data:    `[1,2]`,
	},
	Test{
		desc:    "read basic key",
		json:    `{"a":"b"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	Test{
		desc:    "read basic key with space",
		json:    `{"a": "b"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	Test{
		desc:    "read composite key",
		json:    `{"a": { "b":{"c":"d" }}}`,
		path:    []string{"a", "b", "c"},
		isFound: true,
		data:    `d`,
	},
	Test{
		desc:    `read numberic value as string`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    `1`,
	},
	Test{
		desc:    `handle multiple nested keys with same name`,
		json:    `{"a":[{"b":1},{"b":2},3],"c":{"c":[1,2]}} }`,
		path:    []string{"c", "c"},
		isFound: true,
		data:    `[1,2]`,
	},
	Test{
		desc:    `read string values with quotes`,
		json:    `{"a": "string\"with\"quotes"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `string\"with\"quotes`,
	},
	Test{
		desc:    `read object`,
		json:    `{"a": { "b":{"c":"d" }}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    `{"c":"d" }`,
	},
	Test{
		desc:    `empty path`,
		json:    `{"c":"d" }`,
		path:    []string{},
		isFound: true,
		data:    `{"c":"d" }`,
	},
	Test{
		desc:    `formatted JSON value`,
		json:    "{\n  \"a\": \"b\"\n}",
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	Test{
		desc:    `formatted JSON value 2`,
		json:    "{\n  \"a\":\n    {\n\"b\":\n   {\"c\":\"d\",\n\"e\": \"f\"}\n}\n}",
		path:    []string{"a", "b"},
		isFound: true,
		data:    "{\"c\":\"d\",\n\"e\": \"f\"}",
	},
	Test{
		desc:    `whitespace`,
		json:    " \n\r\t{ \n\r\t\"whitespace\" \n\r\t: \n\r\t333 \n\r\t} \n\r\t",
		path:    []string{"whitespace"},
		isFound: true,
		data:    "333",
	},
	Test{
		desc:    `escaped backslash quote`,
		json:    `{"a": "\\\""}`,
		path:    []string{"a"},
		isFound: true,
		data:    `\\\"`,
	},
	Test{
		desc:    `unescaped backslash quote`,
		json:    `{"a": "\\"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `\\`,
	},
	Test{
		desc:    `unicode in JSON`,
		json:    `{"a": "15°C"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `15°C`,
	},
	Test{
		desc:    `no padding + nested`,
		json:    `{"a":{"a":"1"},"b":2}`,
		path:    []string{"b"},
		isFound: true,
		data:    `2`,
	},
	Test{
		desc:    `no padding + nested + array`,
		json:    `{"a":{"b":[1,2]},"c":3}`,
		path:    []string{"c"},
		isFound: true,
		data:    `3`,
	},

	// Not found key tests
	Test{
		desc:    "non-existent key 1",
		json:    `{"a":"b"}`,
		path:    []string{"c"},
		isFound: false,
		isErr:   true,
	},
	Test{
		desc:    "non-existent key 2",
		json:    `{"a":"b"}`,
		path:    []string{"b"},
		isFound: false,
		isErr:   true,
	},
	Test{
		desc:    "non-existent key 3",
		json:    `{"aa":"b"}`,
		path:    []string{"a"},
		isFound: false,
		isErr:   true,
	},
	Test{
		desc:    "apply scope of parent when search for nested key",
		json:    `{"a": { "b": 1}, "c": 2 }`,
		path:    []string{"a", "b", "c"},
		isFound: false,
		isErr:   true,
	},
	Test{
		desc:    `apply scope to key level`,
		json:    `{"a": { "b": 1}, "c": 2 }`,
		path:    []string{"b"},
		isFound: false,
		isErr:   true,
	},
	Test{
		desc:    `handle escaped quote in key name in JSON`,
		json:    `{"key\"key": 1}`,
		path:    []string{"key"},
		isFound: false,
		isErr:   true,
	},

	// Error/invalid tests
	Test{
		desc:    `handle escaped quote in key name in JSON`,
		json:    `{"key\"key": 1}`,
		path:    []string{"key"},
		isFound: false,
		isErr:   true,
	},
	Test{
		desc:    `missing closing brace, but can still find key`,
		json:    `{"a":"b"`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	Test{
		desc:  `missing value closing quote`,
		json:  `{"a":"b`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  `missing value closing curly brace`,
		json:  `{"a": { "b": "c"`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  `missing value closing square bracket`,
		json:  `{"a": [1, 2, 3 }`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  `missing value 1`,
		json:  `{"a":`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  `missing value 2`,
		json:  `{"a": `,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  `missing value 3`,
		json:  `{"a":}`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  "malformed key (followed by comma followed by colon)",
		json:  `{"a",:1}`,
		path:  []string{"a"},
		isErr: true,
	},
}

var getIntTests = []Test{
	Test{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    int64(1),
	},
	Test{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 1 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    int64(1),
	},
}

var getFloatTests = []Test{
	Test{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1.123}`,
		path:    []string{"c"},
		isFound: true,
		data:    float64(1.123),
	},
	Test{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 23.41323 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    float64(23.41323),
	},
}

var getStringTests = []Test{
	Test{
		desc:    `Translate unicode symbols`,
		json:    `{"c": "test"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `test`,
	},
	Test{
		desc:    `Translate unicode symbols`,
		json:    `{"c": "15\u00b0C"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `15°C`,
	},
	Test{
		desc:    `Translate escape symbols`,
		json:    `{"c": "\\\""}`,
		path:    []string{"c"},
		isFound: true,
		data:    `\"`,
	},
}

var getBoolTests = []Test{
	Test{
		desc:    `read boolean true as boolean`,
		json:    `{"a": "b", "c": true}`,
		path:    []string{"c"},
		isFound: true,
		data:    true,
	},
	Test{
		desc:    `boolean true in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": true \n}",
		path:    []string{"c"},
		isFound: true,
		data:    true,
	},
	Test{
		desc:    `read boolean false as boolean`,
		json:    `{"a": "b", "c": false}`,
		path:    []string{"c"},
		isFound: true,
		data:    false,
	},
	Test{
		desc:    `boolean true in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": false \n}",
		path:    []string{"c"},
		isFound: true,
		data:    false,
	},
	Test{
		desc:  `read fake boolean true`,
		json:  `{"a": txyz}`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:  `read fake boolean false`,
		json:  `{"a": fwxyz}`,
		path:  []string{"a"},
		isErr: true,
	},
	Test{
		desc:    `read boolean true with whitespace and another key`,
		json:    "{\r\t\n \"a\"\r\t\n :\r\t\n true\r\t\n ,\r\t\n \"b\": 1}",
		path:    []string{"a"},
		isFound: true,
		data:    true,
	},
}

var getArrayTests = []Test{
	Test{
		desc:    `read array of simple values`,
		json:    `{"a": { "b":[1,2,3,4]}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    []string{`1`, `2`, `3`, `4`},
	},
	Test{
		desc:    `read array via empty path`,
		json:    `[1,2,3,4]`,
		path:    []string{},
		isFound: true,
		data:    []string{`1`, `2`, `3`, `4`},
	},
	Test{
		desc:    `read array of objects`,
		json:    `{"a": { "b":[{"x":1},{"x":2},{"x":3},{"x":4}]}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    []string{`{"x":1}`, `{"x":2}`, `{"x":3}`, `{"x":4}`},
	},
	Test{
		desc:    `read nested array`,
		json:    `{"a": [[[1]],[[2]]]}`,
		path:    []string{"a"},
		isFound: true,
		data:    []string{`[[1]]`, `[[2]]`},
	},
}

// checkFoundAndNoError checks the dataType and error return from Get*() against the test case expectations.
// Returns true the test should proceed to checking the actual data returned from Get*(), or false if the test is finished.
func checkFoundAndNoError(t *testing.T, testKind string, test Test, jtype int, value interface{}, err error) bool {
	isFound := (jtype != NotExist)
	isErr := (err != nil)

	if test.isErr != isErr {
		// If the call didn't match the error expectation, fail
		t.Errorf("%s test '%s' isErr mismatch: expected %t, obtained %t (err %v). Value: %v", testKind, test.desc, test.isErr, isErr, err, value)
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

func runTests(t *testing.T, tests []Test, runner func(Test) (interface{}, int, error), typeChecker func(Test, interface{}) (bool, interface{})) {
	for _, test := range tests {
		if activeTest != "" && test.desc != activeTest {
			continue
		}

		// fmt.Println("Running:", test.desc)

		value, dataType, err := runner(test)

		if checkFoundAndNoError(t, "Get()", test, dataType, value, err) {
			if test.data == nil {
				t.Errorf("MALFORMED TEST: %v", test)
				continue
			}

			if ok, expected := typeChecker(test, value); !ok {
				t.Errorf("Test '%s' expected to return value %v, but did returned %v instead", test.desc, expected, value)
			}
		}
	}
}

func TestGet(t *testing.T) {
	runTests(t, getTests,
		func(test Test) (value interface{}, dataType int, err error) {
			value, dataType, _, err = Get([]byte(test.json), test.path...)
			return
		},
		func(test Test, value interface{}) (bool, interface{}) {
			expected := []byte(test.data.(string))
			return bytes.Equal(expected, value.([]byte)), expected
		},
	)
}

func TestGetString(t *testing.T) {
	runTests(t, getStringTests,
		func(test Test) (value interface{}, dataType int, err error) {
			value, err = GetString([]byte(test.json), test.path...)
			return value, String, err
		},
		func(test Test, value interface{}) (bool, interface{}) {
			expected := test.data.(string)
			return expected == value.(string), expected
		},
	)
}

func TestGetInt(t *testing.T) {
	runTests(t, getIntTests,
		func(test Test) (value interface{}, dataType int, err error) {
			value, err = GetInt([]byte(test.json), test.path...)
			return value, Number, err
		},
		func(test Test, value interface{}) (bool, interface{}) {
			expected := test.data.(int64)
			return expected == value.(int64), expected
		},
	)
}

func TestGetFloat(t *testing.T) {
	runTests(t, getFloatTests,
		func(test Test) (value interface{}, dataType int, err error) {
			value, err = GetFloat([]byte(test.json), test.path...)
			return value, Number, err
		},
		func(test Test, value interface{}) (bool, interface{}) {
			expected := test.data.(float64)
			return expected == value.(float64), expected
		},
	)
}

func TestGetBoolean(t *testing.T) {
	runTests(t, getBoolTests,
		func(test Test) (value interface{}, dataType int, err error) {
			value, err = GetBoolean([]byte(test.json), test.path...)
			return value, Boolean, err
		},
		func(test Test, value interface{}) (bool, interface{}) {
			expected := test.data.(bool)
			return expected == value.(bool), expected
		},
	)
}

func TestGetSlice(t *testing.T) {
	runTests(t, getArrayTests,
		func(test Test) (value interface{}, dataType int, err error) {
			value, dataType, _, err = Get([]byte(test.json), test.path...)
			return
		},
		func(test Test, value interface{}) (bool, interface{}) {
			expected := test.data.([]string)
			return reflect.DeepEqual(expected, toStringArray(value.([]byte))), expected
		},
	)
}
