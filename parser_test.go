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
	ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
		result = append(result, value)
	})

	return
}

func toStringArray(data []byte) (result []string) {
	ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
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
	GetTest{
		desc:    `escaped backslash quote`,
		json:    `{"a": "\\\""}`,
		path:    []string{"a"},
		isFound: true,
		data:    `\\\"`,
	},
	GetTest{
		desc:    `unescaped backslash quote`,
		json:    `{"a": "\\"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `\\`,
	},
	GetTest{
		desc:    `unicode in JSON`,
		json:    `{"a": "15°C"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `15°C`,
	},
	GetTest{
		desc:    `no padding + nested`,
		json:    `{"a":{"a":"1"},"b":2}`,
		path:    []string{"b"},
		isFound: true,
		data:    `2`,
	},
	GetTest{
		desc:    `no padding + nested + array`,
		json:    `{"a":{"b":[1,2]},"c":3}`,
		path:    []string{"c"},
		isFound: true,
		data:    `3`,
	},

	// Escaped key tests
	GetTest{
		desc:    `key with simple escape`,
		json:    `{"a\\b":1}`,
		path:    []string{"a\\b"},
		isFound: true,
		data:    `1`,
	},
	GetTest{
		desc:    `key with Unicode escape`,
		json:    `{"a\u00B0b":1}`,
		path:    []string{"a\u00B0b"},
		isFound: true,
		data:    `1`,
	},
	GetTest{
		desc:    `key with complex escape`,
		json:    `{"a\uD83D\uDE03b":1}`,
		path:    []string{"a\U0001F603b"},
		isFound: true,
		data:    `1`,
	},

	GetTest{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance
		desc:    `malformed with trailing whitespace`,
		json:    `{"a":1 `,
		path:    []string{"a"},
		isFound: true,
		data:    `1`,
	},
	GetTest{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance
		desc:    `malformed with wrong closing bracket`,
		json:    `{"a":1]`,
		path:    []string{"a"},
		isFound: true,
		data:    `1`,
	},

	// Not found key tests
	GetTest{
		desc:  "non-existent key 1",
		json:  `{"a":"b"}`,
		path:  []string{"c"},
		isErr: true,
	},
	GetTest{
		desc:  "non-existent key 2",
		json:  `{"a":"b"}`,
		path:  []string{"b"},
		isErr: true,
	},
	GetTest{
		desc:  "non-existent key 3",
		json:  `{"aa":"b"}`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  "apply scope of parent when search for nested key",
		json:  `{"a": { "b": 1}, "c": 2 }`,
		path:  []string{"a", "b", "c"},
		isErr: true,
	},
	GetTest{
		desc:  `apply scope to key level`,
		json:  `{"a": { "b": 1}, "c": 2 }`,
		path:  []string{"b"},
		isErr: true,
	},
	GetTest{
		desc:  `handle escaped quote in key name in JSON`,
		json:  `{"key\"key": 1}`,
		path:  []string{"key"},
		isErr: true,
	},

	// Error/invalid tests
	GetTest{
		desc:  `handle escaped quote in key name in JSON`,
		json:  `{"key\"key": 1}`,
		path:  []string{"key"},
		isErr: true,
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

	GetTest{ // This test returns not found instead of a parse error, as checking for the malformed JSON would reduce performance
		desc:  "malformed key (followed by comma followed by colon)",
		json:  `{"a",:1}`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc:    "malformed 'colon chain', lookup first string",
		json:    `{"a":"b":"c"}`,
		path:    []string{"a"},
		isFound: true,
		data:    "b",
	},
	GetTest{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc:    "malformed 'colon chain', lookup second string",
		json:    `{"a":"b":"c"}`,
		path:    []string{"b"},
		isFound: true,
		data:    "c",
	},
}

var getIntTests = []GetTest{
	GetTest{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    int64(1),
	},
	GetTest{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 1 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    int64(1),
	},
}

var getFloatTests = []GetTest{
	GetTest{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1.123}`,
		path:    []string{"c"},
		isFound: true,
		data:    float64(1.123),
	},
	GetTest{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 23.41323 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    float64(23.41323),
	},
}

var getStringTests = []GetTest{
	GetTest{
		desc:    `Translate Unicode symbols`,
		json:    `{"c": "test"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `test`,
	},
	GetTest{
		desc:    `Translate Unicode symbols`,
		json:    `{"c": "15\u00b0C"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `15°C`,
	},
	GetTest{
		desc:    `Translate supplementary Unicode symbols`,
		json:    `{"c": "\uD83D\uDE03"}`, // Smiley face (UTF16 surrogate pair)
		path:    []string{"c"},
		isFound: true,
		data:    "\U0001F603", // Smiley face
	},
	GetTest{
		desc:    `Translate escape symbols`,
		json:    `{"c": "\\\""}`,
		path:    []string{"c"},
		isFound: true,
		data:    `\"`,
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
	GetTest{
		desc:  `read fake boolean true`,
		json:  `{"a": txyz}`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:  `read fake boolean false`,
		json:  `{"a": fwxyz}`,
		path:  []string{"a"},
		isErr: true,
	},
	GetTest{
		desc:    `read boolean true with whitespace and another key`,
		json:    "{\r\t\n \"a\"\r\t\n :\r\t\n true\r\t\n ,\r\t\n \"b\": 1}",
		path:    []string{"a"},
		isFound: true,
		data:    true,
	},
}

var getArrayTests = []GetTest{
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
func getTestCheckFoundAndNoError(t *testing.T, testKind string, test GetTest, jtype ValueType, value interface{}, err error) bool {
	isFound := (jtype != NotExist) && (err != KeyPathNotFoundError)
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

func runGetTests(t *testing.T, testKind string, tests []GetTest, runner func(GetTest) (interface{}, ValueType, error), resultChecker func(GetTest, interface{}) (bool, interface{})) {
	for _, test := range tests {
		if activeTest != "" && test.desc != activeTest {
			continue
		}

		// fmt.Println("Running:", test.desc)

		value, dataType, err := runner(test)

		if getTestCheckFoundAndNoError(t, testKind, test, dataType, value, err) {
			if test.data == nil {
				t.Errorf("MALFORMED TEST: %v", test)
				continue
			}

			if ok, expected := resultChecker(test, value); !ok {
				if expectedBytes, ok := expected.([]byte); ok {
					expected = string(expectedBytes)
				}
				if valueBytes, ok := value.([]byte); ok {
					value = string(valueBytes)
				}
				t.Errorf("%s test '%s' expected to return value %v, but did returned %v instead", testKind, test.desc, expected, value)
			}
		}
	}
}

func TestGet(t *testing.T) {
	runGetTests(t, "Get()", getTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, dataType, _, err = Get([]byte(test.json), test.path...)
			return
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := []byte(test.data.(string))
			return bytes.Equal(expected, value.([]byte)), expected
		},
	)
}

func TestGetString(t *testing.T) {
	runGetTests(t, "GetString()", getStringTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, err = GetString([]byte(test.json), test.path...)
			return value, String, err
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := test.data.(string)
			return expected == value.(string), expected
		},
	)
}

func TestGetInt(t *testing.T) {
	runGetTests(t, "GetInt()", getIntTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, err = GetInt([]byte(test.json), test.path...)
			return value, Number, err
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := test.data.(int64)
			return expected == value.(int64), expected
		},
	)
}

func TestGetFloat(t *testing.T) {
	runGetTests(t, "GetFloat()", getFloatTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, err = GetFloat([]byte(test.json), test.path...)
			return value, Number, err
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := test.data.(float64)
			return expected == value.(float64), expected
		},
	)
}

func TestGetBoolean(t *testing.T) {
	runGetTests(t, "GetBoolean()", getBoolTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, err = GetBoolean([]byte(test.json), test.path...)
			return value, Boolean, err
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := test.data.(bool)
			return expected == value.(bool), expected
		},
	)
}

func TestGetSlice(t *testing.T) {
	runGetTests(t, "Get()-for-arrays", getArrayTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, dataType, _, err = Get([]byte(test.json), test.path...)
			return
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := test.data.([]string)
			return reflect.DeepEqual(expected, toStringArray(value.([]byte))), expected
		},
	)
}

func TestArrayEach(t *testing.T) {
	mock := []byte(`{"a": { "b":[{"x": 1} ,{"x":2},{ "x":3}, {"x":4} ]}}`)
	count := 0

	ArrayEach(mock, func(value []byte, dataType ValueType, offset int, err error) {
		count++

		switch count {
		case 1:
			if string(value) != `{"x": 1}` {
				t.Errorf("Wrong first item: %s", string(value))
			}
		case 2:
			if string(value) != `{"x":2}` {
				t.Errorf("Wrong second item: %s", string(value))
			}
		case 3:
			if string(value) != `{ "x":3}` {
				t.Errorf("Wrong third item: %s", string(value))
			}
		case 4:
			if string(value) != `{"x":4}` {
				t.Errorf("Wrong forth item: %s", string(value))
			}
		default:
			t.Errorf("Should process only 4 items")
		}
	}, "a", "b")
}

type ParseTest struct {
	in     string
	intype ValueType
	out    interface{}
	isErr  bool
}

var parseBoolTests = []ParseTest{
	ParseTest{
		in:     "true",
		intype: Boolean,
		out:    true,
	},
	ParseTest{
		in:     "false",
		intype: Boolean,
		out:    false,
	},
	ParseTest{
		in:     "foo",
		intype: Boolean,
		isErr:  true,
	},
	ParseTest{
		in:     "trux",
		intype: Boolean,
		isErr:  true,
	},
	ParseTest{
		in:     "truex",
		intype: Boolean,
		isErr:  true,
	},
	ParseTest{
		in:     "",
		intype: Boolean,
		isErr:  true,
	},
}

var parseFloatTest = []ParseTest{
	ParseTest{
		in:     "0",
		intype: Number,
		out:    float64(0),
	},
	ParseTest{
		in:     "0.0",
		intype: Number,
		out:    float64(0.0),
	},
	ParseTest{
		in:     "1",
		intype: Number,
		out:    float64(1),
	},
	ParseTest{
		in:     "1.234",
		intype: Number,
		out:    float64(1.234),
	},
	ParseTest{
		in:     "1.234e5",
		intype: Number,
		out:    float64(1.234e5),
	},
	ParseTest{
		in:     "-1.234e5",
		intype: Number,
		out:    float64(-1.234e5),
	},
	ParseTest{
		in:     "+1.234e5", // Note: + sign not allowed under RFC7159, but our parser accepts it since it uses strconv.ParseFloat
		intype: Number,
		out:    float64(1.234e5),
	},
	ParseTest{
		in:     "1.2.3",
		intype: Number,
		isErr:  true,
	},
	ParseTest{
		in:     "1..1",
		intype: Number,
		isErr:  true,
	},
	ParseTest{
		in:     "1a",
		intype: Number,
		isErr:  true,
	},
	ParseTest{
		in:     "",
		intype: Number,
		isErr:  true,
	},
}

// parseTestCheckNoError checks the error return from Parse*() against the test case expectations.
// Returns true the test should proceed to checking the actual data returned from Parse*(), or false if the test is finished.
func parseTestCheckNoError(t *testing.T, testKind string, test ParseTest, value interface{}, err error) bool {
	if isErr := (err != nil); test.isErr != isErr {
		// If the call didn't match the error expectation, fail
		t.Errorf("%s test '%s' isErr mismatch: expected %t, obtained %t (err %v). Obtained value: %v", testKind, test.in, test.isErr, isErr, err, value)
		return false
	} else if isErr {
		// Else, if there was an error, don't fail and don't check isFound or the value
		return false
	} else {
		// Else, there was no error and a value was found, so check the value
		return true
	}
}

func runParseTests(t *testing.T, testKind string, tests []ParseTest, runner func(ParseTest) (interface{}, error), resultChecker func(ParseTest, interface{}) (bool, interface{})) {
	for _, test := range tests {
		value, err := runner(test)

		if parseTestCheckNoError(t, testKind, test, value, err) {
			if test.out == nil {
				t.Errorf("MALFORMED TEST: %v", test)
				continue
			}

			if ok, expected := resultChecker(test, value); !ok {
				if expectedBytes, ok := expected.([]byte); ok {
					expected = string(expectedBytes)
				}
				if valueBytes, ok := value.([]byte); ok {
					value = string(valueBytes)
				}
				t.Errorf("%s test '%s' expected to return value %v, but did returned %v instead", testKind, test.in, expected, value)
			}
		}
	}
}

func TestParseBoolean(t *testing.T) {
	runParseTests(t, "ParseBoolean()", parseBoolTests,
		func(test ParseTest) (value interface{}, err error) {
			return ParseBoolean([]byte(test.in))
		},
		func(test ParseTest, obtained interface{}) (bool, interface{}) {
			expected := test.out.(bool)
			return obtained.(bool) == expected, expected
		},
	)
}

func TestParseFloat(t *testing.T) {
	runParseTests(t, "ParseFloat()", parseFloatTest,
		func(test ParseTest) (value interface{}, err error) {
			return ParseFloat([]byte(test.in))
		},
		func(test ParseTest, obtained interface{}) (bool, interface{}) {
			expected := test.out.(float64)
			return obtained.(float64) == expected, expected
		},
	)
}
