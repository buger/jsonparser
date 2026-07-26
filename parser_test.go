package jsonparser

import (
	"bytes"
	"errors"
	"fmt"
	_ "fmt"
	"reflect"
	"testing"
)

// Set it to non-empty value if want to run only specific test
var activeTest = ""

// Test helper for SYS-REQ-006.
func toArray(data []byte) (result [][]byte) {
	ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
		result = append(result, value)
	})

	return
}

// Test helper for SYS-REQ-006 and SYS-REQ-008.
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

type SetTest struct {
	desc    string
	json    string
	setData string
	path    []string

	isErr   bool
	isFound bool

	data interface{}
}

type DeleteTest struct {
	desc string
	json string
	path []string

	data interface{}
}

var deleteTests = []DeleteTest{
	{
		desc: "Delete test key",
		json: `{"test":"input"}`,
		path: []string{"test"},
		data: `{}`,
	},
	{
		desc: "Delete object",
		json: `{"test":"input"}`,
		path: []string{},
		data: ``,
	},
	{
		desc: "Delete a nested object",
		json: `{"test":"input","new.field":{"key": "new object"}}`,
		path: []string{"new.field", "key"},
		data: `{"test":"input","new.field":{}}`,
	},
	{
		desc: "Deleting a key that doesn't exist should return the same object",
		json: `{"test":"input"}`,
		path: []string{"test2"},
		data: `{"test":"input"}`,
	},
	{
		desc: "Delete object in an array",
		json: `{"test":[{"key":"val-obj1"}]}`,
		path: []string{"test", "[0]"},
		data: `{"test":[]}`,
	},
	{
		desc: "Deleting a object in an array that doesn't exists should return the same object",
		json: `{"test":[{"key":"val-obj1"}]}`,
		path: []string{"test", "[1]"},
		data: `{"test":[{"key":"val-obj1"}]}`,
	},
	{
		desc: "Delete a complex object in a nested array",
		json: `{"test":[{"key":[{"innerKey":"innerKeyValue"}]}]}`,
		path: []string{"test", "[0]", "key", "[0]"},
		data: `{"test":[{"key":[]}]}`,
	},
	{
		desc: "Delete known key (simple type within nested array)",
		json: `{"test":[{"key":["innerKey"]}]}`,
		path: []string{"test", "[0]", "key", "[0]"},
		data: `{"test":[{"key":[]}]}`,
	},
	{
		desc: "Delete in empty json",
		json: `{}`,
		path: []string{},
		data: ``,
	},
	{
		desc: "Delete empty array",
		json: `[]`,
		path: []string{},
		data: ``,
	},
	{
		desc: "Deleting non json should return the same value",
		json: `1.323`,
		path: []string{"foo"},
		data: `1.323`,
	},
	{
		desc: "Delete known key (top level array)",
		json: `[{"key":"val-obj1"}]`,
		path: []string{"[0]"},
		data: `[]`,
	},
	{ // This test deletes the key instead of returning a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc: `malformed with trailing whitespace`,
		json: `{"a":1 `,
		path: []string{"a"},
		data: `{ `,
	},
	{ // This test dels the key instead of returning a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc: "malformed 'colon chain', delete b",
		json: `{"a":"b":"c"}`,
		path: []string{"b"},
		data: `{"a":}`,
	},
	{
		desc: "Delete object without inner array",
		json: `{"a": {"b": 1}, "b": 2}`,
		path: []string{"b"},
		data: `{"a": {"b": 1}}`,
	},
	{
		desc: "Delete object without inner array",
		json: `{"a": [{"b": 1}], "b": 2}`,
		path: []string{"b"},
		data: `{"a": [{"b": 1}]}`,
	},
	{
		desc: "Delete object without inner array",
		json: `{"a": {"c": {"b": 3}, "b": 1}, "b": 2}`,
		path: []string{"a", "b"},
		data: `{"a": {"c": {"b": 3}}, "b": 2}`,
	},
	{
		desc: "Delete object without inner array",
		json: `{"a": [{"c": {"b": 3}, "b": 1}], "b": 2}`,
		path: []string{"a", "[0]", "b"},
		data: `{"a": [{"c": {"b": 3}}], "b": 2}`,
	},
	{
		desc: "Remove trailing comma if last object is deleted",
		json: `{"a": "1", "b": "2"}`,
		path: []string{"b"},
		data: `{"a": "1"}`,
	},
	{
		desc: "Correctly delete first element with space-comma",
		json: `{"a": "1" ,"b": "2" }`,
		path: []string{"a"},
		data: `{"b": "2" }`,
	},
	{
		desc: "Correctly delete middle element with space-comma",
		json: `{"a": "1" ,"b": "2" , "c": 3}`,
		path: []string{"b"},
		data: `{"a": "1" , "c": 3}`,
	},
	{
		desc: "Delete non-last key",
		json: `{"test":"input","test1":"input1"}`,
		path: []string{"test"},
		data: `{"test1":"input1"}`,
	},
	{
		desc: "Delete non-exist key",
		json: `{"test:":"input"}`,
		path: []string{"test", "test1"},
		data: `{"test:":"input"}`,
	},
	{
		desc: "Delete non-last object in an array",
		json: `[{"key":"val-obj1"},{"key2":"val-obj2"}]`,
		path: []string{"[0]"},
		data: `[{"key2":"val-obj2"}]`,
	},
	{
		desc: "Delete non-first object in an array",
		json: `[{"key":"val-obj1"},{"key2":"val-obj2"}]`,
		path: []string{"[1]"},
		data: `[{"key":"val-obj1"}]`,
	},
	{
		desc: "Issue #188: infinite loop in Delete",
		json: `^_ï¿½^C^A^@[`,
		path: []string{""},
		data: `^_ï¿½^C^A^@[`,
	},
	{
		desc: "Issue #188: infinite loop in Delete",
		json: `^_ï¿½^C^A^@{`,
		path: []string{""},
		data: `^_ï¿½^C^A^@{`,
	},
	{
		desc: "Issue #150: leading space",
		json: `   {"test":"input"}`,
		path: []string{"test"},
		data: `   {}`,
	},
	{
		desc: "GO-2026-4514: malformed JSON without enclosing braces should not panic",
		json: `"0":"0":`,
		path: []string{"0"},
		data: `"0":"0":`,
	},
	{
		desc: "GO-2026-4514: malformed JSON with key but truncated value should not panic",
		json: `{"a":  `,
		path: []string{"a"},
		data: `{"a":  `,
	},
	{
		desc: "GO-2026-4514: malformed nested JSON with truncated value should not panic",
		json: `{"a":{"b":  `,
		path: []string{"a", "b"},
		data: `{"a":{"b":  `,
	},
	{
		// OSS-Fuzz testcase 4649128545288192: leading garbage comma
		// caused findTokenStart to return offset 0, which then made the
		// trailing-comma cleanup branch reassign keyOffset=0, which made
		// lastToken(data[:0]) return -1, which made data[prevTok] panic
		// with "index out of range [-1]" at parser.go.
		desc: "OSS-Fuzz: leading-comma malformed input must not panic in Delete",
		json: `,{"test":1{}`,
		path: []string{"test"},
		data: `}`,
	},
	{
		desc: "OSS-Fuzz variant: leading comma + empty string then object must not panic in Delete",
		json: `,""{"test":0}`,
		path: []string{"test"},
		data: `}`,
	},
}

var setTests = []SetTest{
	{
		desc:    "set unknown key (string)",
		json:    `{"test":"input"}`,
		isFound: true,
		path:    []string{"new.field"},
		setData: `"new value"`,
		data:    `{"test":"input","new.field":"new value"}`,
	},
	{
		desc:    "set known key (string)",
		json:    `{"test":"input"}`,
		isFound: true,
		path:    []string{"test"},
		setData: `"new value"`,
		data:    `{"test":"new value"}`,
	},
	{
		desc:    "set unknown key (object)",
		json:    `{"test":"input"}`,
		isFound: true,
		path:    []string{"new.field"},
		setData: `{"key": "new object"}`,
		data:    `{"test":"input","new.field":{"key": "new object"}}`,
	},
	{
		desc:    "set known key (object)",
		json:    `{"test":"input"}`,
		isFound: true,
		path:    []string{"test"},
		setData: `{"key": "new object"}`,
		data:    `{"test":{"key": "new object"}}`,
	},
	{
		desc:    "set known key (object within array)",
		json:    `{"test":[{"key":"val-obj1"}]}`,
		isFound: true,
		path:    []string{"test", "[0]"},
		setData: `{"key":"new object"}`,
		data:    `{"test":[{"key":"new object"}]}`,
	},
	{
		desc:    "set unknown key (replace object)",
		json:    `{"test":[{"key":"val-obj1"}]}`,
		isFound: true,
		path:    []string{"test", "newKey"},
		setData: `"new object"`,
		data:    `{"test":{"newKey":"new object"}}`,
	},
	{
		desc:    "set unknown key (complex object within nested array)",
		json:    `{"test":[{"key":[{"innerKey":"innerKeyValue"}]}]}`,
		isFound: true,
		path:    []string{"test", "[0]", "key", "[0]", "newInnerKey"},
		setData: `{"key":"new object"}`,
		data:    `{"test":[{"key":[{"innerKey":"innerKeyValue","newInnerKey":{"key":"new object"}}]}]}`,
	},
	{
		desc:    "set known key (complex object within nested array)",
		json:    `{"test":[{"key":[{"innerKey":"innerKeyValue"}]}]}`,
		isFound: true,
		path:    []string{"test", "[0]", "key", "[0]", "innerKey"},
		setData: `{"key":"new object"}`,
		data:    `{"test":[{"key":[{"innerKey":{"key":"new object"}}]}]}`,
	},
	{
		desc:    "set unknown key (object, partial subtree exists)",
		json:    `{"test":{"input":"output"}}`,
		isFound: true,
		path:    []string{"test", "new.field"},
		setData: `{"key":"new object"}`,
		data:    `{"test":{"input":"output","new.field":{"key":"new object"}}}`,
	},
	{
		desc:    "set unknown key (object, empty partial subtree exists)",
		json:    `{"test":{}}`,
		isFound: true,
		path:    []string{"test", "new.field"},
		setData: `{"key":"new object"}`,
		data:    `{"test":{"new.field":{"key":"new object"}}}`,
	},
	{
		desc:    "set unknown key (object, no subtree exists)",
		json:    `{"test":"input"}`,
		isFound: true,
		path:    []string{"new.field", "nested", "value"},
		setData: `{"key": "new object"}`,
		data:    `{"test":"input","new.field":{"nested":{"value":{"key": "new object"}}}}`,
	},
	{
		desc:    "set in empty json",
		json:    `{}`,
		isFound: true,
		path:    []string{"foo"},
		setData: `"null"`,
		data:    `{"foo":"null"}`,
	},
	{
		desc:    "set subtree in empty json",
		json:    `{}`,
		isFound: true,
		path:    []string{"foo", "bar"},
		setData: `"null"`,
		data:    `{"foo":{"bar":"null"}}`,
	},
	{
		desc:    "set in empty string - not found",
		json:    ``,
		isFound: false,
		path:    []string{"foo"},
		setData: `"null"`,
		data:    ``,
	},
	{
		desc:    "set in Number - not found",
		json:    `1.323`,
		isFound: false,
		path:    []string{"foo"},
		setData: `"null"`,
		data:    `1.323`,
	},
	{
		desc:    "set known key (top level array)",
		json:    `[{"key":"val-obj1"}]`,
		isFound: true,
		path:    []string{"[0]", "key"},
		setData: `"new object"`,
		data:    `[{"key":"new object"}]`,
	},
	{
		desc:    "set unknown key (trailing whitespace)",
		json:    `{"key":"val-obj1"}  `,
		isFound: true,
		path:    []string{"alt-key"},
		setData: `"new object"`,
		data:    `{"key":"val-obj1","alt-key":"new object"}  `,
	},
	{ // This test sets the key instead of returning a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc:    `malformed with trailing whitespace`,
		json:    `{"a":1 `,
		path:    []string{"a"},
		setData: `2`,
		isFound: true,
		data:    `{"a":2 `,
	},
	{ // This test sets the key instead of returning a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc:    "malformed 'colon chain', set second string",
		json:    `{"a":"b":"c"}`,
		path:    []string{"b"},
		setData: `"d"`,
		isFound: true,
		data:    `{"a":"b":"d"}`,
	},
	{
		desc:    "set indexed path to object on empty JSON",
		json:    `{}`,
		path:    []string{"top", "[0]", "middle", "[0]", "bottom"},
		setData: `"value"`,
		isFound: true,
		data:    `{"top":[{"middle":[{"bottom":"value"}]}]}`,
	},
	{
		desc:    "set indexed path on existing object with object",
		json:    `{"top":[{"middle":[]}]}`,
		path:    []string{"top", "[0]", "middle", "[0]", "bottom"},
		setData: `"value"`,
		isFound: true,
		data:    `{"top":[{"middle":[{"bottom":"value"}]}]}`,
	},
	{
		desc:    "set indexed path on existing object with value",
		json:    `{"top":[{"middle":[]}]}`,
		path:    []string{"top", "[0]", "middle", "[0]"},
		setData: `"value"`,
		isFound: true,
		data:    `{"top":[{"middle":["value"]}]}`,
	},
	{
		desc:    "set indexed path on empty object with value",
		json:    `{}`,
		path:    []string{"top", "[0]", "middle", "[0]"},
		setData: `"value"`,
		isFound: true,
		data:    `{"top":[{"middle":["value"]}]}`,
	},
	{
		desc:    "set indexed path on object with existing array",
		json:    `{"top":["one", "two", "three"]}`,
		path:    []string{"top", "[2]"},
		setData: `"value"`,
		isFound: true,
		data:    `{"top":["one", "two", "value"]}`,
	},
	{
		desc:    "set non-exist key",
		json:    `{"test":"input"}`,
		setData: `"new value"`,
		isFound: false,
	},
	{
		desc:    "set key in invalid json",
		json:    `{"test"::"input"}`,
		path:    []string{"test"},
		setData: "new value",
		isErr:   true,
	},
	{
		desc:    "set unknown key (simple object within nested array)",
		json:    `{"test":{"key":[{"innerKey":"innerKeyValue", "innerKey2":"innerKeyValue2"}]}}`,
		isFound: true,
		path:    []string{"test", "key", "[1]", "newInnerKey"},
		setData: `"new object"`,
		data:    `{"test":{"key":[{"innerKey":"innerKeyValue", "innerKey2":"innerKeyValue2"},{"newInnerKey":"new object"}]}}`,
	},
}

var getTests = []GetTest{
	// Trivial tests
	{
		desc:    "read string",
		json:    `""`,
		isFound: true,
		data:    ``,
	},
	{
		desc:    "read number",
		json:    `0`,
		isFound: true,
		data:    `0`,
	},
	{
		desc:    "read object",
		json:    `{}`,
		isFound: true,
		data:    `{}`,
	},
	{
		desc:    "read array",
		json:    `[]`,
		isFound: true,
		data:    `[]`,
	},
	{
		desc:    "read boolean",
		json:    `true`,
		isFound: true,
		data:    `true`,
	},

	// Found key tests
	{
		desc:    "handling multiple nested keys with same name",
		json:    `{"a":[{"b":1},{"b":2},3],"c":{"c":[1,2]}} }`,
		path:    []string{"c", "c"},
		isFound: true,
		data:    `[1,2]`,
	},
	{
		desc:    "read basic key",
		json:    `{"a":"b"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	{
		desc:    "read basic key with space",
		json:    `{"a": "b"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	{
		desc:    "read composite key",
		json:    `{"a": { "b":{"c":"d" }}}`,
		path:    []string{"a", "b", "c"},
		isFound: true,
		data:    `d`,
	},
	{
		desc:    `read numberic value as string`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    `1`,
	},
	{
		desc:    `handle multiple nested keys with same name`,
		json:    `{"a":[{"b":1},{"b":2},3],"c":{"c":[1,2]}} }`,
		path:    []string{"c", "c"},
		isFound: true,
		data:    `[1,2]`,
	},
	{
		desc:    `read string values with quotes`,
		json:    `{"a": "string\"with\"quotes"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `string\"with\"quotes`,
	},
	{
		desc:    `read object`,
		json:    `{"a": { "b":{"c":"d" }}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    `{"c":"d" }`,
	},
	{
		desc:    `empty path`,
		json:    `{"c":"d" }`,
		path:    []string{},
		isFound: true,
		data:    `{"c":"d" }`,
	},
	{
		desc:    `formatted JSON value`,
		json:    "{\n  \"a\": \"b\"\n}",
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	{
		desc:    `formatted JSON value 2`,
		json:    "{\n  \"a\":\n    {\n\"b\":\n   {\"c\":\"d\",\n\"e\": \"f\"}\n}\n}",
		path:    []string{"a", "b"},
		isFound: true,
		data:    "{\"c\":\"d\",\n\"e\": \"f\"}",
	},
	{
		desc:    `whitespace`,
		json:    " \n\r\t{ \n\r\t\"whitespace\" \n\r\t: \n\r\t333 \n\r\t} \n\r\t",
		path:    []string{"whitespace"},
		isFound: true,
		data:    "333",
	},
	{
		desc:    `escaped backslash quote`,
		json:    `{"a": "\\\""}`,
		path:    []string{"a"},
		isFound: true,
		data:    `\\\"`,
	},
	{
		desc:    `unescaped backslash quote`,
		json:    `{"a": "\\"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `\\`,
	},
	{
		desc:    `unicode in JSON`,
		json:    `{"a": "15°C"}`,
		path:    []string{"a"},
		isFound: true,
		data:    `15°C`,
	},
	{
		desc:    `no padding + nested`,
		json:    `{"a":{"a":"1"},"b":2}`,
		path:    []string{"b"},
		isFound: true,
		data:    `2`,
	},
	{
		desc:    `no padding + nested + array`,
		json:    `{"a":{"b":[1,2]},"c":3}`,
		path:    []string{"c"},
		isFound: true,
		data:    `3`,
	},
	{
		desc:    `empty key`,
		json:    `{"":{"":{"":true}}}`,
		path:    []string{"", "", ""},
		isFound: true,
		data:    `true`,
	},

	// Escaped key tests
	{
		desc:    `key with simple escape`,
		json:    `{"a\\b":1}`,
		path:    []string{"a\\b"},
		isFound: true,
		data:    `1`,
	},
	{
		desc:    `key and value with whitespace escapes`,
		json:    `{"key\b\f\n\r\tkey":"value\b\f\n\r\tvalue"}`,
		path:    []string{"key\b\f\n\r\tkey"},
		isFound: true,
		data:    `value\b\f\n\r\tvalue`, // value is not unescaped since this is Get(), but the key should work correctly
	},
	{
		desc:    `key with Unicode escape`,
		json:    `{"a\u00B0b":1}`,
		path:    []string{"a\u00B0b"},
		isFound: true,
		data:    `1`,
	},
	{
		desc:    `key with complex escape`,
		json:    `{"a\uD83D\uDE03b":1}`,
		path:    []string{"a\U0001F603b"},
		isFound: true,
		data:    `1`,
	},

	{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance
		desc:    `malformed with trailing whitespace`,
		json:    `{"a":1 `,
		path:    []string{"a"},
		isFound: true,
		data:    `1`,
	},
	{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance
		desc:    `malformed with wrong closing bracket`,
		json:    `{"a":1]`,
		path:    []string{"a"},
		isFound: true,
		data:    `1`,
	},

	// Not found key tests
	{
		desc:    `empty input`,
		json:    ``,
		path:    []string{"a"},
		isFound: false,
	},
	{
		desc:    "non-existent key 1",
		json:    `{"a":"b"}`,
		path:    []string{"c"},
		isFound: false,
	},
	{
		desc:    "non-existent key 2",
		json:    `{"a":"b"}`,
		path:    []string{"b"},
		isFound: false,
	},
	{
		desc:    "non-existent key 3",
		json:    `{"aa":"b"}`,
		path:    []string{"a"},
		isFound: false,
	},
	{
		desc:    "apply scope of parent when search for nested key",
		json:    `{"a": { "b": 1}, "c": 2 }`,
		path:    []string{"a", "b", "c"},
		isFound: false,
	},
	{
		desc:    `apply scope to key level`,
		json:    `{"a": { "b": 1}, "c": 2 }`,
		path:    []string{"b"},
		isFound: false,
	},
	{
		desc:    `handle escaped quote in key name in JSON`,
		json:    `{"key\"key": 1}`,
		path:    []string{"key"},
		isFound: false,
	},
	{
		desc:    "handling multiple keys with different name",
		json:    `{"a":{"a":1},"b":{"a":3,"c":[1,2]}}`,
		path:    []string{"a", "c"},
		isFound: false,
	},
	{
		desc:    "handling nested json",
		json:    `{"a":{"b":{"c":1},"d":4}}`,
		path:    []string{"a", "d"},
		isFound: true,
		data:    `4`,
	},
	{ // Issue #148
		desc:    `missing key in different key same level`,
		json:    `{"s":"s","ic":2,"r":{"o":"invalid"}}`,
		path:    []string{"ic", "o"},
		isFound: false,
	},

	// Error/invalid tests
	{
		desc:    `handle escaped quote in key name in JSON`,
		json:    `{"key\"key": 1}`,
		path:    []string{"key"},
		isFound: false,
	},
	{
		desc:    `missing closing brace, but can still find key`,
		json:    `{"a":"b"`,
		path:    []string{"a"},
		isFound: true,
		data:    `b`,
	},
	{
		desc:  `missing value closing quote`,
		json:  `{"a":"b`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:  `missing value closing curly brace`,
		json:  `{"a": { "b": "c"`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:  `missing value closing square bracket`,
		json:  `{"a": [1, 2, 3 }`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:  `missing value 1`,
		json:  `{"a":`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:  `missing value 2`,
		json:  `{"a": `,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:  `missing value 3`,
		json:  `{"a":}`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:    `malformed array (no closing brace)`,
		json:    `{"a":[, "b":123}`,
		path:    []string{"b"},
		isFound: false,
	},
	{ // Issue #81
		desc:    `missing key in object in array`,
		json:    `{"p":{"a":[{"u":"abc","t":"th"}]}}`,
		path:    []string{"p", "a", "[0]", "x"},
		isFound: false,
	},
	{ // Issue #81 counter test
		desc:    `existing key in object in array`,
		json:    `{"p":{"a":[{"u":"abc","t":"th"}]}}`,
		path:    []string{"p", "a", "[0]", "u"},
		isFound: true,
		data:    "abc",
	},
	{ // This test returns not found instead of a parse error, as checking for the malformed JSON would reduce performance
		desc:    "malformed key (followed by comma followed by colon)",
		json:    `{"a",:1}`,
		path:    []string{"a"},
		isFound: false,
	},
	{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc:    "malformed 'colon chain', lookup first string",
		json:    `{"a":"b":"c"}`,
		path:    []string{"a"},
		isFound: true,
		data:    "b",
	},
	{ // This test returns a match instead of a parse error, as checking for the malformed JSON would reduce performance (this is not ideal)
		desc:    "malformed 'colon chain', lookup second string",
		json:    `{"a":"b":"c"}`,
		path:    []string{"b"},
		isFound: true,
		data:    "c",
	},
	// Array index paths
	{
		desc:    "last key in path is index",
		json:    `{"a":[{"b":1},{"b":"2"}, 3],"c":{"c":[1,2]}}`,
		path:    []string{"a", "[1]"},
		isFound: true,
		data:    `{"b":"2"}`,
	},
	{
		desc:    "get string from array",
		json:    `{"a":[{"b":1},"foo", 3],"c":{"c":[1,2]}}`,
		path:    []string{"a", "[1]"},
		isFound: true,
		data:    "foo",
	},
	{
		desc:    "key in path is index",
		json:    `{"a":[{"b":"1"},{"b":"2"},3],"c":{"c":[1,2]}}`,
		path:    []string{"a", "[0]", "b"},
		isFound: true,
		data:    `1`,
	},
	{
		desc: "last key in path is an index to value in array (formatted json)",
		json: `{
		    "a": [
			{
			    "b": 1
			},
			{"b":"2"},
			3
		    ],
		    "c": {
			"c": [
			    1,
			    2
			]
		    }
		}`,
		path:    []string{"a", "[1]"},
		isFound: true,
		data:    `{"b":"2"}`,
	},
	{
		desc: "key in path is index (formatted json)",
		json: `{
		    "a": [
			{"b": 1},
			{"b": "2"},
			3
		    ],
		    "c": {
			"c": [
			    1,
			    2
			]
		    }
		}`,
		path:    []string{"a", "[0]", "b"},
		isFound: true,
		data:    `1`,
	},
	{
		// Issue #178: Crash in searchKeys
		desc:    `invalid json`,
		json:    `{{{"":`,
		path:    []string{"a", "b"},
		isFound: false,
	},
	{
		desc:    `opening brace instead of closing and without key`,
		json:    `{"a":1{`,
		path:    []string{"b"},
		isFound: false,
	},
}

var getIntTests = []GetTest{
	{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1}`,
		path:    []string{"c"},
		isFound: true,
		data:    int64(1),
	},
	{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 1 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    int64(1),
	},
	{ // Issue #138: overflow detection
		desc:  `Fails because of overflow`,
		json:  `{"p":9223372036854775808}`,
		path:  []string{"p"},
		isErr: true,
	},
	{ // Issue #138: overflow detection
		desc:  `Fails because of underflow`,
		json:  `{"p":-9223372036854775809}`,
		path:  []string{"p"},
		isErr: true,
	},
	{
		desc:  `read non-numeric value as integer`,
		json:  `{"a": "b", "c": "d"}`,
		path:  []string{"c"},
		isErr: true,
	},
	{
		desc:  `null test`,
		json:  `{"a": "b", "c": null}`,
		path:  []string{"c"},
		isErr: true,
	},
}

var getFloatTests = []GetTest{
	{
		desc:    `read numeric value as number`,
		json:    `{"a": "b", "c": 1.123}`,
		path:    []string{"c"},
		isFound: true,
		data:    float64(1.123),
	},
	{
		desc:    `read numeric value as number in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": 23.41323 \n}",
		path:    []string{"c"},
		isFound: true,
		data:    float64(23.41323),
	},
	{
		desc:  `read non-numeric value as float`,
		json:  `{"a": "b", "c": "d"}`,
		path:  []string{"c"},
		isErr: true,
	},
	{
		desc:  `null test`,
		json:  `{"a": "b", "c": null}`,
		path:  []string{"c"},
		isErr: true,
	},
}

var getStringTests = []GetTest{
	{
		desc:    `Translate Unicode symbols`,
		json:    `{"c": "test"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `test`,
	},
	{
		desc:    `Translate Unicode symbols`,
		json:    `{"c": "15\u00b0C"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `15°C`,
	},
	{
		desc:    `Translate supplementary Unicode symbols`,
		json:    `{"c": "\uD83D\uDE03"}`, // Smiley face (UTF16 surrogate pair)
		path:    []string{"c"},
		isFound: true,
		data:    "\U0001F603", // Smiley face
	},
	{
		desc:    `Translate escape symbols`,
		json:    `{"c": "\\\""}`,
		path:    []string{"c"},
		isFound: true,
		data:    `\"`,
	},
	{
		desc:    `key and value with whitespace escapes`,
		json:    `{"key\b\f\n\r\tkey":"value\b\f\n\r\tvalue"}`,
		path:    []string{"key\b\f\n\r\tkey"},
		isFound: true,
		data:    "value\b\f\n\r\tvalue", // value is unescaped since this is GetString()
	},
	{ // This test checks we avoid an infinite loop for certain malformed JSON. We don't check for all malformed JSON as it would reduce performance.
		desc:    `malformed with double quotes`,
		json:    `{"a"":1}`,
		path:    []string{"a"},
		isFound: false,
		data:    ``,
	},
	{ // More malformed JSON testing, to be sure we avoid an infinite loop.
		desc:    `malformed with double quotes, and path does not exist`,
		json:    `{"z":123,"y":{"x":7,"w":0},"v":{"u":"t","s":"r","q":0,"p":1558051800},"a":"b","c":"2016-11-02T20:10:11Z","d":"e","f":"g","h":{"i":"j""},"k":{"l":"m"}}`,
		path:    []string{"o"},
		isFound: false,
		data:    ``,
	},
	{
		desc:  `read non-string as string`,
		json:  `{"c": true}`,
		path:  []string{"c"},
		isErr: true,
	},
	{
		desc:    `empty array index`,
		json:    `[""]`,
		path:    []string{"[]"},
		isFound: false,
	},
	{
		desc:    `malformed array index`,
		json:    `[""]`,
		path:    []string{"["},
		isFound: false,
	},
	{
		desc:  `null test`,
		json:  `{"c": null}`,
		path:  []string{"c"},
		isErr: true,
	},
}

var getUnsafeStringTests = []GetTest{
	{
		desc:    `read empty string as unsafe string`,
		json:    `{"c": ""}`,
		path:    []string{"c"},
		isFound: true,
		data:    ``,
	},
	{
		desc:    `Do not translate Unicode symbols`,
		json:    `{"c": "test"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `test`,
	},
	{
		desc:    `Do not translate Unicode symbols`,
		json:    `{"c": "15\u00b0C"}`,
		path:    []string{"c"},
		isFound: true,
		data:    `15\u00b0C`,
	},
	{
		desc:    `Do not translate supplementary Unicode symbols`,
		json:    `{"c": "\uD83D\uDE03"}`, // Smiley face (UTF16 surrogate pair)
		path:    []string{"c"},
		isFound: true,
		data:    `\uD83D\uDE03`, // Smiley face
	},
	{
		desc:    `Do not translate escape symbols`,
		json:    `{"c": "\\\""}`,
		path:    []string{"c"},
		isFound: true,
		data:    `\\\"`,
	},
	{
		desc:    `read boolean token as unsafe string`,
		json:    `{"c": true}`,
		path:    []string{"c"},
		isFound: true,
		data:    `true`,
	},
	{
		desc:    `missing key returns not found for unsafe string`,
		json:    `{"c": "test"}`,
		path:    []string{"missing"},
		isFound: false,
	},
}

var getBoolTests = []GetTest{
	{
		desc:    `read boolean true as boolean`,
		json:    `{"a": "b", "c": true}`,
		path:    []string{"c"},
		isFound: true,
		data:    true,
	},
	{
		desc:    `boolean true in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": true \n}",
		path:    []string{"c"},
		isFound: true,
		data:    true,
	},
	{
		desc:    `read boolean false as boolean`,
		json:    `{"a": "b", "c": false}`,
		path:    []string{"c"},
		isFound: true,
		data:    false,
	},
	{
		desc:    `boolean true in formatted JSON`,
		json:    "{\"a\": \"b\", \"c\": false \n}",
		path:    []string{"c"},
		isFound: true,
		data:    false,
	},
	{
		desc:  `read fake boolean true`,
		json:  `{"a": txyz}`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:  `read fake boolean false`,
		json:  `{"a": fwxyz}`,
		path:  []string{"a"},
		isErr: true,
	},
	{
		desc:    `read boolean true with whitespace and another key`,
		json:    "{\r\t\n \"a\"\r\t\n :\r\t\n true\r\t\n ,\r\t\n \"b\": 1}",
		path:    []string{"a"},
		isFound: true,
		data:    true,
	},
	{
		desc:    `null test`,
		json:    `{"a": "b", "c": null}`,
		path:    []string{"c"},
		isFound: false,
		isErr:   true,
	},
}

var getArrayTests = []GetTest{
	{
		desc:    `read array of simple values`,
		json:    `{"a": { "b":[1,2,3,4]}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    []string{`1`, `2`, `3`, `4`},
	},
	{
		desc:    `read array via empty path`,
		json:    `[1,2,3,4]`,
		path:    []string{},
		isFound: true,
		data:    []string{`1`, `2`, `3`, `4`},
	},
	{
		desc:    `read array of objects`,
		json:    `{"a": { "b":[{"x":1},{"x":2},{"x":3},{"x":4}]}}`,
		path:    []string{"a", "b"},
		isFound: true,
		data:    []string{`{"x":1}`, `{"x":2}`, `{"x":3}`, `{"x":4}`},
	},
	{
		desc:    `read nested array`,
		json:    `{"a": [[[1]],[[2]]]}`,
		path:    []string{"a"},
		isFound: true,
		data:    []string{`[[1]]`, `[[2]]`},
	},
}

// checkFoundAndNoError checks the dataType and error return from Get*() against the test case expectations.
// Returns true the test should proceed to checking the actual data returned from Get*(), or false if the test is finished.
// Test helper for SYS-REQ-001, SYS-REQ-002, SYS-REQ-003, SYS-REQ-004, SYS-REQ-005, and SYS-REQ-011.
func getTestCheckFoundAndNoError(t *testing.T, testKind string, test GetTest, jtype ValueType, value interface{}, err error) bool {
	isFound := (err != KeyPathNotFoundError)
	isErr := (err != nil && err != KeyPathNotFoundError)

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

// Test helper for SYS-REQ-001, SYS-REQ-002, SYS-REQ-003, SYS-REQ-004, SYS-REQ-005, and SYS-REQ-011.
func runGetTests(t *testing.T, testKind string, tests []GetTest, runner func(GetTest) (interface{}, ValueType, error), resultChecker func(GetTest, interface{}) (bool, interface{})) {
	for _, test := range tests {
		if activeTest != "" && test.desc != activeTest {
			continue
		}

		fmt.Println("Running:", test.desc)

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

// Test helper for SYS-REQ-009.
func setTestCheckFoundAndNoError(t *testing.T, testKind string, test SetTest, value interface{}, err error) bool {
	isFound := (err != KeyPathNotFoundError)
	isErr := (err != nil && err != KeyPathNotFoundError)

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

// Test helper for SYS-REQ-009.
func runSetTests(t *testing.T, testKind string, tests []SetTest, runner func(SetTest) (interface{}, ValueType, error), resultChecker func(SetTest, interface{}) (bool, interface{})) {
	for _, test := range tests {
		if activeTest != "" && test.desc != activeTest {
			continue
		}

		fmt.Println("Running:", test.desc)

		value, _, err := runner(test)

		if setTestCheckFoundAndNoError(t, testKind, test, value, err) {
			if test.data == nil {
				t.Errorf("MALFORMED TEST: %v", test)
				continue
			}

			if string(value.([]byte)) != test.data {
				t.Errorf("Unexpected result on %s test '%s'", testKind, test.desc)
				t.Log("Got:     ", string(value.([]byte)))
				t.Log("Expected:", test.data)
				t.Log("Error:   ", err)
			}
		}
	}
}

// Test helper for SYS-REQ-010.
func runDeleteTests(t *testing.T, testKind string, tests []DeleteTest, runner func(DeleteTest) (interface{}, []byte), resultChecker func(DeleteTest, interface{}) (bool, interface{})) {
	for _, test := range tests {
		if activeTest != "" && test.desc != activeTest {
			continue
		}

		original := make([]byte, len(test.json))
		copy(original, test.json)

		fmt.Println("Running:", test.desc)

		value, bytes := runner(test)

		if string(original) != string(bytes) {
			t.Errorf("ORIGINAL DATA MALFORMED: %v, %v", string(original), string(bytes))
			continue
		}

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

// Verifies: SYS-REQ-010 [example]
// STK-REQ-005:AC-2:acceptance
// MCDC SYS-REQ-010: delete_path_is_provided=F, delete_returns_empty_document_without_path=T => TRUE
// Verifies: SYS-REQ-033 [example]
// MCDC SYS-REQ-033: delete_path_is_provided=T, delete_target_exists=T, delete_returns_document_without_target=T => TRUE
// Verifies: SYS-REQ-034 [example]
// MCDC SYS-REQ-034: delete_path_is_provided=T, delete_target_exists=F, delete_input_is_unusable_for_requested_path=F, delete_preserves_input_when_target_missing=T => TRUE
func TestDelete(t *testing.T) {
	runDeleteTests(t, "Delete()", deleteTests,
		func(test DeleteTest) (interface{}, []byte) {
			ba := []byte(test.json)
			return Delete(ba, test.path...), ba
		},
		func(test DeleteTest, value interface{}) (bool, interface{}) {
			expected := []byte(test.data.(string))
			return bytes.Equal(expected, value.([]byte)), expected
		},
	)
}

// Verifies: SYS-REQ-001 [example]
// STK-REQ-001:AC-1:acceptance
// MCDC SYS-REQ-001: addressed_path_exists=F, json_input_is_well_formed=T, key_path_is_provided=T, returns_existing_path_lookup_result=F => TRUE
// MCDC SYS-REQ-001: addressed_path_exists=T, json_input_is_well_formed=F, key_path_is_provided=T, returns_existing_path_lookup_result=F => TRUE
// MCDC SYS-REQ-001: addressed_path_exists=T, json_input_is_well_formed=T, key_path_is_provided=F, returns_existing_path_lookup_result=F => TRUE
// MCDC SYS-REQ-001: addressed_path_exists=T, json_input_is_well_formed=T, key_path_is_provided=T, returns_existing_path_lookup_result=F => FALSE
// MCDC SYS-REQ-001: addressed_path_exists=T, json_input_is_well_formed=T, key_path_is_provided=T, returns_existing_path_lookup_result=T => TRUE
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

// Verifies: SYS-REQ-016 [boundary]
// MCDC SYS-REQ-016: N/A
// Verifies: SYS-REQ-017 [boundary]
// MCDC SYS-REQ-017: N/A
// Verifies: SYS-REQ-018 [boundary]
// MCDC SYS-REQ-018: N/A
// Verifies: SYS-REQ-019 [boundary]
// MCDC SYS-REQ-019: N/A
// Verifies: SYS-REQ-020 [boundary]
// MCDC SYS-REQ-020: N/A
// Verifies: SYS-REQ-021 [boundary]
// MCDC SYS-REQ-021: N/A
// Verifies: SYS-REQ-022 [boundary]
// MCDC SYS-REQ-022: N/A
// Verifies: SYS-REQ-023 [boundary]
// MCDC SYS-REQ-023: N/A
// Verifies: SYS-REQ-024 [boundary]
// MCDC SYS-REQ-024: N/A
// Verifies: SYS-REQ-025 [boundary]
// MCDC SYS-REQ-025: N/A
// Verifies: SYS-REQ-026 [boundary]
// MCDC SYS-REQ-026: N/A
// Verifies: SYS-REQ-027 [boundary]
// MCDC SYS-REQ-027: N/A
func TestGetRequirementSlices(t *testing.T) {
	t.Run("well formed missing path returns not found", func(t *testing.T) {
		value, dataType, offset, err := Get([]byte(`{"a":"b"}`), "missing")
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("expected KeyPathNotFoundError, got %v", err)
		}
		if dataType != NotExist || offset != -1 || value != nil {
			t.Fatalf("expected not-found tuple, got value=%v type=%v offset=%d", value, dataType, offset)
		}
	})

	t.Run("incomplete input returns parse error", func(t *testing.T) {
		if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
			t.Fatal("expected parse-related error for incomplete input")
		}
	})

	t.Run("no key path returns closest root value", func(t *testing.T) {
		value, dataType, _, err := Get([]byte(`{"a":1}`))
		if err != nil {
			t.Fatalf("Get without key path returned error: %v", err)
		}
		if dataType != Object || string(value) != `{"a":1}` {
			t.Fatalf("unexpected root value result: value=%s type=%v", string(value), dataType)
		}
	})

	t.Run("empty input with key path returns not found", func(t *testing.T) {
		_, dataType, offset, err := Get([]byte(""), "a")
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("expected KeyPathNotFoundError, got %v", err)
		}
		if dataType != NotExist || offset != -1 {
			t.Fatalf("expected empty-input not-found tuple, got type=%v offset=%d", dataType, offset)
		}
	})

	t.Run("object key lookup respects current scope", func(t *testing.T) {
		value, dataType, _, err := Get([]byte(`{"a":{"b":1},"b":2}`), "a", "b")
		if err != nil {
			t.Fatalf("nested object lookup returned error: %v", err)
		}
		if dataType != Number || string(value) != "1" {
			t.Fatalf("unexpected nested object lookup result: value=%s type=%v", string(value), dataType)
		}
	})

	t.Run("array index lookup returns in-bounds element", func(t *testing.T) {
		value, dataType, _, err := Get([]byte(`{"a":[{"b":1},"foo",3]}`), "a", "[1]")
		if err != nil {
			t.Fatalf("array index lookup returned error: %v", err)
		}
		if dataType != String || string(value) != "foo" {
			t.Fatalf("unexpected array lookup result: value=%s type=%v", string(value), dataType)
		}
	})

	t.Run("invalid array index syntax returns not found", func(t *testing.T) {
		if _, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "["); !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("expected KeyPathNotFoundError for malformed array index, got %v", err)
		}
	})

	t.Run("out of bounds array index returns not found", func(t *testing.T) {
		if _, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[9]"); !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("expected KeyPathNotFoundError for out-of-bounds array index, got %v", err)
		}
	})

	t.Run("escaped keys are matched after decoding", func(t *testing.T) {
		value, dataType, _, err := Get([]byte(`{"a\u00B0b":1}`), "a°b")
		if err != nil {
			t.Fatalf("escaped-key lookup returned error: %v", err)
		}
		if dataType != Number || string(value) != "1" {
			t.Fatalf("unexpected escaped-key lookup result: value=%s type=%v", string(value), dataType)
		}
	})

	t.Run("string results are unquoted but not unescaped", func(t *testing.T) {
		value, dataType, _, err := Get([]byte(`{"a":"line\nbreak"}`), "a")
		if err != nil {
			t.Fatalf("string lookup returned error: %v", err)
		}
		if dataType != String || string(value) != `line\nbreak` {
			t.Fatalf("unexpected raw string result: value=%s type=%v", string(value), dataType)
		}
	})

	t.Run("best effort lookup succeeds when malformed data is outside addressed token", func(t *testing.T) {
		value, dataType, _, err := Get([]byte(`{"a":1]`), "a")
		if err != nil {
			t.Fatalf("best-effort lookup returned error: %v", err)
		}
		if dataType != Number || string(value) != "1" {
			t.Fatalf("unexpected best-effort lookup result: value=%s type=%v", string(value), dataType)
		}
	})

	t.Run("invalid addressed token shape returns value type error", func(t *testing.T) {
		if _, _, _, err := Get([]byte(`{"a":u}`), "a"); !errors.Is(err, UnknownValueTypeError) {
			t.Fatalf("expected UnknownValueTypeError, got %v", err)
		}
	})
}

// Verifies: SYS-REQ-002 [example]
// STK-REQ-002:AC-1:acceptance
// MCDC SYS-REQ-002: addressed_value_is_string=F, raw_string_token_is_well_formed=T, returns_getstring_decoded_value=F => TRUE
// MCDC SYS-REQ-002: addressed_value_is_string=T, raw_string_token_is_well_formed=F, returns_getstring_decoded_value=F => TRUE
// MCDC SYS-REQ-002: addressed_value_is_string=T, raw_string_token_is_well_formed=T, returns_getstring_decoded_value=F => FALSE
// MCDC SYS-REQ-002: addressed_value_is_string=T, raw_string_token_is_well_formed=T, returns_getstring_decoded_value=T => TRUE
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

// Verifies: SYS-REQ-011 [example]
// STK-REQ-006:AC-1:acceptance
// MCDC SYS-REQ-011: addressed_value_is_string=F, returns_unsafe_string_view=F => TRUE
// MCDC SYS-REQ-011: addressed_value_is_string=T, returns_unsafe_string_view=F => FALSE
// MCDC SYS-REQ-011: addressed_value_is_string=T, returns_unsafe_string_view=T => TRUE
func TestGetUnsafeString(t *testing.T) {
	runGetTests(t, "GetUnsafeString()", getUnsafeStringTests,
		func(test GetTest) (value interface{}, dataType ValueType, err error) {
			value, err = GetUnsafeString([]byte(test.json), test.path...)
			return value, String, err
		},
		func(test GetTest, value interface{}) (bool, interface{}) {
			expected := test.data.(string)
			return expected == value.(string), expected
		},
	)
}

// Verifies: SYS-REQ-003 [example]
// STK-REQ-003:AC-1:acceptance
// MCDC SYS-REQ-003: addressed_value_is_number=F, raw_number_token_is_integer_parseable=T, returns_getint_value=F => TRUE
// MCDC SYS-REQ-003: addressed_value_is_number=T, raw_number_token_is_integer_parseable=F, returns_getint_value=F => TRUE
// MCDC SYS-REQ-003: addressed_value_is_number=T, raw_number_token_is_integer_parseable=T, returns_getint_value=F => FALSE
// MCDC SYS-REQ-003: addressed_value_is_number=T, raw_number_token_is_integer_parseable=T, returns_getint_value=T => TRUE
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

// Verifies: SYS-REQ-004 [example]
// STK-REQ-003:AC-2:acceptance
// MCDC SYS-REQ-004: addressed_value_is_number=F, raw_number_token_is_float_parseable=T, returns_getfloat_value=F => TRUE
// MCDC SYS-REQ-004: addressed_value_is_number=T, raw_number_token_is_float_parseable=F, returns_getfloat_value=F => TRUE
// MCDC SYS-REQ-004: addressed_value_is_number=T, raw_number_token_is_float_parseable=T, returns_getfloat_value=F => FALSE
// MCDC SYS-REQ-004: addressed_value_is_number=T, raw_number_token_is_float_parseable=T, returns_getfloat_value=T => TRUE
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

// Verifies: SYS-REQ-005 [example]
// STK-REQ-003:AC-3:acceptance
// MCDC SYS-REQ-005: addressed_value_is_boolean=F, raw_boolean_token_is_well_formed=T, returns_getboolean_value=F => TRUE
// MCDC SYS-REQ-005: addressed_value_is_boolean=T, raw_boolean_token_is_well_formed=F, returns_getboolean_value=F => TRUE
// MCDC SYS-REQ-005: addressed_value_is_boolean=T, raw_boolean_token_is_well_formed=T, returns_getboolean_value=F => FALSE
// MCDC SYS-REQ-005: addressed_value_is_boolean=T, raw_boolean_token_is_well_formed=T, returns_getboolean_value=T => TRUE
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

// Verifies: SYS-REQ-001 [example]
// MCDC SYS-REQ-001: N/A
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

// Verifies: SYS-REQ-006 [example]
// STK-REQ-004:AC-1:acceptance
// MCDC SYS-REQ-006: addressed_array_is_empty=F, addressed_array_is_well_formed=T, array_callback_receives_elements_in_order=F => FALSE
// MCDC SYS-REQ-006: addressed_array_is_empty=F, addressed_array_is_well_formed=T, array_callback_receives_elements_in_order=T => TRUE
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

// Verifies: SYS-REQ-029 [boundary]
// MCDC SYS-REQ-029: addressed_array_is_well_formed=F, malformed_array_input_returns_error=T => TRUE
func TestArrayEachWithWhiteSpace(t *testing.T) {
	// Issue #159
	count := 0
	funcError := func([]byte, ValueType, int, error) { t.Errorf("Run func not allow") }
	funcSuccess := func(value []byte, dataType ValueType, index int, err error) {
		count++

		switch count {
		case 1:
			if string(value) != `AAA` {
				t.Errorf("Wrong first item: %s", string(value))
			}
		case 2:
			if string(value) != `BBB` {
				t.Errorf("Wrong second item: %s", string(value))
			}
		case 3:
			if string(value) != `CCC` {
				t.Errorf("Wrong third item: %s", string(value))
			}
		default:
			t.Errorf("Should process only 3 items")
		}
	}

	type args struct {
		data []byte
		cb   func(value []byte, dataType ValueType, offset int, err error)
		keys []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Array with white space", args{[]byte(`    ["AAA", "BBB", "CCC"]`), funcSuccess, []string{}}, false},
		{"Array with only one character after white space", args{[]byte(`    1`), funcError, []string{}}, true},
		{"Only white space", args{[]byte(`    `), funcError, []string{}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ArrayEach(tt.args.data, tt.args.cb, tt.args.keys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArrayEach() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// Verifies: SYS-REQ-028 [boundary]
// MCDC SYS-REQ-028: addressed_array_is_empty=T, addressed_array_is_well_formed=T, empty_array_produces_no_callbacks=T => TRUE
func TestArrayEachEmpty(t *testing.T) {
	funcError := func([]byte, ValueType, int, error) { t.Errorf("Run func not allow") }

	type args struct {
		data []byte
		cb   func(value []byte, dataType ValueType, offset int, err error)
		keys []string
	}
	tests := []struct {
		name       string
		args       args
		wantOffset int
		wantErr    bool
	}{
		{"Empty array", args{[]byte("[]"), funcError, []string{}}, 1, false},
		{"Empty array with space", args{[]byte("[ ]"), funcError, []string{}}, 2, false},
		{"Empty array with \n", args{[]byte("[\n]"), funcError, []string{}}, 2, false},
		{"Empty field array", args{[]byte("{\"data\": []}"), funcError, []string{"data"}}, 10, false},
		{"Empty field array with space", args{[]byte("{\"data\": [ ]}"), funcError, []string{"data"}}, 11, false},
		{"Empty field array with \n", args{[]byte("{\"data\": [\n]}"), funcError, []string{"data"}}, 11, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOffset, err := ArrayEach(tt.args.data, tt.args.cb, tt.args.keys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArrayEach() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("ArrayEach() = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}

type keyValueEntry struct {
	key       string
	value     string
	valueType ValueType
}

// Test helper for SYS-REQ-007.
func (kv keyValueEntry) String() string {
	return fmt.Sprintf("[%s: %s (%s)]", kv.key, kv.value, kv.valueType)
}

type ObjectEachTest struct {
	desc string
	json string

	isErr   bool
	entries []keyValueEntry
}

var objectEachTests = []ObjectEachTest{
	{
		desc:    "empty object",
		json:    `{}`,
		entries: []keyValueEntry{},
	},
	{
		desc: "single key-value object",
		json: `{"key": "value"}`,
		entries: []keyValueEntry{
			{"key", "value", String},
		},
	},
	{
		desc: "multiple key-value object with many value types",
		json: `{
		  "key1": null,
		  "key2": true,
		  "key3": 1.23,
		  "key4": "string value",
		  "key5": [1,2,3],
		  "key6": {"a":"b"}
		}`,
		entries: []keyValueEntry{
			{"key1", "null", Null},
			{"key2", "true", Boolean},
			{"key3", "1.23", Number},
			{"key4", "string value", String},
			{"key5", "[1,2,3]", Array},
			{"key6", `{"a":"b"}`, Object},
		},
	},
	{
		desc: "escaped key",
		json: `{"key\"\\\/\b\f\n\r\t\u00B0": "value"}`,
		entries: []keyValueEntry{
			{"key\"\\/\b\f\n\r\t\u00B0", "value", String},
		},
	},
	// Error cases
	{
		desc:  "no object present",
		json:  ` \t\n\r`,
		isErr: true,
	},
	{
		desc:  "unmatched braces 1",
		json:  `{`,
		isErr: true,
	},
	{
		desc:  "unmatched braces 2",
		json:  `}`,
		isErr: true,
	},
	{
		desc:  "unmatched braces 3",
		json:  `}{}`,
		isErr: true,
	},
	{
		desc:  "bad key (number)",
		json:  `{123: "value"}`,
		isErr: true,
	},
	{
		desc:  "bad key (unclosed quote)",
		json:  `{"key: 123}`,
		isErr: true,
	},
	{
		desc:  "bad value (no value)",
		json:  `{"key":}`,
		isErr: true,
	},
	{
		desc:  "bad value (bogus value)",
		json:  `{"key": notavalue}`,
		isErr: true,
	},
	{
		desc:  "bad entry (missing colon)",
		json:  `{"key" "value"}`,
		isErr: true,
	},
	{
		desc:  "bad entry (no trailing comma)",
		json:  `{"key": "value" "key2": "value2"}`,
		isErr: true,
	},
	{
		desc:  "bad entry (two commas)",
		json:  `{"key": "value",, "key2": "value2"}`,
		isErr: true,
	},
}

// Verifies: SYS-REQ-031 [example]
// MCDC SYS-REQ-031: addressed_object_is_well_formed=F, malformed_object_input_returns_error=T => TRUE
// Verifies: SYS-REQ-030 [example]
// MCDC SYS-REQ-030: addressed_object_is_empty=T, addressed_object_is_well_formed=T, empty_object_produces_no_entries=T => TRUE
// Verifies: SYS-REQ-031 [example]
// MCDC SYS-REQ-031: addressed_object_is_well_formed=F, malformed_object_input_returns_error=T => TRUE
// Verifies: SYS-REQ-030 [example]
// MCDC SYS-REQ-030: addressed_object_is_empty=T, addressed_object_is_well_formed=T, empty_object_produces_no_entries=T => TRUE
// Verifies: SYS-REQ-007 [example]
// STK-REQ-004:AC-2:acceptance
// MCDC SYS-REQ-007: addressed_object_is_empty=F, addressed_object_is_well_formed=T, object_callback_receives_entries=F => FALSE
// MCDC SYS-REQ-007: addressed_object_is_empty=F, addressed_object_is_well_formed=T, object_callback_receives_entries=T => TRUE
func TestObjectEach(t *testing.T) {
	for _, test := range objectEachTests {
		if activeTest != "" && test.desc != activeTest {
			continue
		}

		// Execute ObjectEach and capture all of the entries visited, in order
		var entries []keyValueEntry
		err := ObjectEach([]byte(test.json), func(key, value []byte, valueType ValueType, off int) error {
			entries = append(entries, keyValueEntry{
				key:       string(key),
				value:     string(value),
				valueType: valueType,
			})
			return nil
		})

		// Check the correctness of the result
		isErr := (err != nil)
		if test.isErr != isErr {
			// If the call didn't match the error expectation, fail
			t.Errorf("ObjectEach test '%s' isErr mismatch: expected %t, obtained %t (err %v)", test.desc, test.isErr, isErr, err)
		} else if isErr {
			// Else, if there was an expected error, don't fail and don't check anything further
		} else if len(test.entries) != len(entries) {
			t.Errorf("ObjectEach test '%s' mismatch in number of key-value entries: expected %d, obtained %d (entries found: %s)", test.desc, len(test.entries), len(entries), entries)
		} else {
			for i, entry := range entries {
				expectedEntry := test.entries[i]
				if expectedEntry.key != entry.key {
					t.Errorf("ObjectEach test '%s' key mismatch at entry %d: expected %s, obtained %s", test.desc, i, expectedEntry.key, entry.key)
					break
				} else if expectedEntry.value != entry.value {
					t.Errorf("ObjectEach test '%s' value mismatch at entry %d: expected %s, obtained %s", test.desc, i, expectedEntry.value, entry.value)
					break
				} else if expectedEntry.valueType != entry.valueType {
					t.Errorf("ObjectEach test '%s' value type mismatch at entry %d: expected %s, obtained %s", test.desc, i, expectedEntry.valueType, entry.valueType)
					break
				} else {
					// Success for this entry
				}
			}
		}
	}
}

// Verifies: SYS-REQ-032 [boundary]
// MCDC SYS-REQ-032: addressed_object_is_well_formed=T, object_callback_returns_error=T, object_callback_error_is_returned=T => TRUE
func TestObjectEachNestedPathAndCallbackError(t *testing.T) {
	t.Run("nested object path", func(t *testing.T) {
		var entries []keyValueEntry
		err := ObjectEach([]byte(`{"outer":{"a":1,"b":true}}`), func(key, value []byte, valueType ValueType, off int) error {
			entries = append(entries, keyValueEntry{
				key:       string(key),
				value:     string(value),
				valueType: valueType,
			})
			return nil
		}, "outer")
		if err != nil {
			t.Fatalf("ObjectEach nested path returned error: %v", err)
		}
		expected := []keyValueEntry{
			{key: "a", value: "1", valueType: Number},
			{key: "b", value: "true", valueType: Boolean},
		}
		if !reflect.DeepEqual(expected, entries) {
			t.Fatalf("ObjectEach nested path entries mismatch: expected %#v, got %#v", expected, entries)
		}
	})

	t.Run("callback error is returned", func(t *testing.T) {
		sentinel := errors.New("stop iteration")
		err := ObjectEach([]byte(`{"a":1}`), func(key, value []byte, valueType ValueType, off int) error {
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("ObjectEach callback error mismatch: expected %v, got %v", sentinel, err)
		}
	})
}

var testJson = []byte(`{
	"name": "Name", 
	"order": "Order", 
	"sum": 100, 
	"len": 12, 
	"isPaid": true, 
	"nested": {"a":"test", "b":2, "nested3":{"a":"test3","b":4}, "c": "unknown"}, 
	"nested2": {
		"a":"test2", 
		"b":3
	}, 
	"arr": [
		{
			"a":"zxc", 
			"b": 1
		}, 
		{
			"a":"123", 
			"b":2
		}
	], 
	"arrInt": [1,2,3,4], 
	"intPtr": 10, 
	"a\n":{
		"b\n":99
	}
}`)

// Verifies: SYS-REQ-008 [example]
// STK-REQ-004:AC-3:acceptance
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=F, eachkey_completes_requested_scan=F, eachkey_malformed_input_returns_error=F, missing_multipath_request_does_not_emit_callback=F, multipath_requests_are_provided=T => FALSE
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=F, eachkey_completes_requested_scan=F, eachkey_malformed_input_returns_error=F, missing_multipath_request_does_not_emit_callback=T, multipath_requests_are_provided=T => TRUE
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=F, eachkey_completes_requested_scan=T, eachkey_malformed_input_returns_error=F, missing_multipath_request_does_not_emit_callback=F, multipath_requests_are_provided=T => TRUE
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=T, eachkey_completes_requested_scan=F, eachkey_malformed_input_returns_error=F, missing_multipath_request_does_not_emit_callback=F, multipath_requests_are_provided=T => TRUE
func TestEachKey(t *testing.T) {
	paths := [][]string{
		{"name"},
		{"order"},
		{"nested", "a"},
		{"nested", "b"},
		{"nested2", "a"},
		{"nested", "nested3", "b"},
		{"arr", "[1]", "b"},
		{"arrInt", "[3]"},
		{"arrInt", "[5]"}, // Should not find last key
		{"nested"},
		{"arr", "["},    // issue#177 Invalid arguments
		{"a\n", "b\n"},  // issue#165
		{"nested", "b"}, // Should find repeated key
	}

	keysFound := 0

	EachKey(testJson, func(idx int, value []byte, vt ValueType, err error) {
		keysFound++

		switch idx {
		case 0:
			if string(value) != "Name" {
				t.Error("Should find 1 key", string(value))
			}
		case 1:
			if string(value) != "Order" {
				t.Errorf("Should find 2 key")
			}
		case 2:
			if string(value) != "test" {
				t.Errorf("Should find 3 key")
			}
		case 3:
			if string(value) != "2" {
				t.Errorf("Should find 4 key")
			}
		case 4:
			if string(value) != "test2" {
				t.Error("Should find 5 key", string(value))
			}
		case 5:
			if string(value) != "4" {
				t.Errorf("Should find 6 key")
			}
		case 6:
			if string(value) != "2" {
				t.Errorf("Should find 7 key")
			}
		case 7:
			if string(value) != "4" {
				t.Error("Should find 8 key", string(value))
			}
		case 8:
			t.Errorf("Found key #8 that should not be found")
		case 9:
			if string(value) != `{"a":"test", "b":2, "nested3":{"a":"test3","b":4}, "c": "unknown"}` {
				t.Error("Should find 9 key", string(value))
			}
		case 10:
			t.Errorf("Found key #10 that should not be found")
		case 11:
			if string(value) != "99" {
				t.Error("Should find 10 key", string(value))
			}
		case 12:
			if string(value) != "2" {
				t.Errorf("Should find 11 key")
			}
		default:
			t.Errorf("Should find only 10 keys, got %v key", idx)
		}
	}, paths...)

	if keysFound != 11 {
		t.Errorf("Should find 11 keys: %d", keysFound)
	}
}

type ParseTest struct {
	in     string
	intype ValueType
	out    interface{}
	isErr  bool
}

var parseBoolTests = []ParseTest{
	{
		in:     "true",
		intype: Boolean,
		out:    true,
	},
	{
		in:     "false",
		intype: Boolean,
		out:    false,
	},
	{
		in:     "foo",
		intype: Boolean,
		isErr:  true,
	},
	{
		in:     "trux",
		intype: Boolean,
		isErr:  true,
	},
	{
		in:     "truex",
		intype: Boolean,
		isErr:  true,
	},
	{
		in:     "",
		intype: Boolean,
		isErr:  true,
	},
}

var parseFloatTest = []ParseTest{
	{
		in:     "0",
		intype: Number,
		out:    float64(0),
	},
	{
		in:     "0.0",
		intype: Number,
		out:    float64(0.0),
	},
	{
		in:     "1",
		intype: Number,
		out:    float64(1),
	},
	{
		in:     "1.234",
		intype: Number,
		out:    float64(1.234),
	},
	{
		in:     "1.234e5",
		intype: Number,
		out:    float64(1.234e5),
	},
	{
		in:     "-1.234e5",
		intype: Number,
		out:    float64(-1.234e5),
	},
	{
		in:     "+1.234e5", // Note: + sign not allowed under RFC7159, but our parser accepts it since it uses strconv.ParseFloat
		intype: Number,
		out:    float64(1.234e5),
	},
	{
		in:     "1.2.3",
		intype: Number,
		isErr:  true,
	},
	{
		in:     "1..1",
		intype: Number,
		isErr:  true,
	},
	{
		in:     "1a",
		intype: Number,
		isErr:  true,
	},
	{
		in:     "",
		intype: Number,
		isErr:  true,
	},
}

// parseTestCheckNoError checks the error return from Parse*() against the test case expectations.
// Returns true the test should proceed to checking the actual data returned from Parse*(), or false if the test is finished.
// Test helper for SYS-REQ-012, SYS-REQ-013, SYS-REQ-014, and SYS-REQ-015.
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

// Test helper for SYS-REQ-012, SYS-REQ-013, SYS-REQ-014, and SYS-REQ-015.
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

// Verifies: SYS-REQ-036 [example]
// MCDC SYS-REQ-036: raw_boolean_literal_is_valid=F, returns_parseboolean_error=T => TRUE
// Verifies: SYS-REQ-012 [example]
// STK-REQ-007:AC-1:acceptance
// MCDC SYS-REQ-012: raw_boolean_literal_is_valid=F, returns_parseboolean_value=F => TRUE
// MCDC SYS-REQ-012: raw_boolean_literal_is_valid=T, returns_parseboolean_value=F => FALSE
// MCDC SYS-REQ-012: raw_boolean_literal_is_valid=T, returns_parseboolean_value=T => TRUE
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

// Verifies: SYS-REQ-037 [example]
// MCDC SYS-REQ-037: raw_float_token_is_well_formed=F, returns_parsefloat_error=T => TRUE
// Verifies: SYS-REQ-013 [example]
// STK-REQ-007:AC-2:acceptance
// MCDC SYS-REQ-013: raw_float_token_is_well_formed=F, returns_parsefloat_value=F => TRUE
// MCDC SYS-REQ-013: raw_float_token_is_well_formed=T, returns_parsefloat_value=F => FALSE
// MCDC SYS-REQ-013: raw_float_token_is_well_formed=T, returns_parsefloat_value=T => TRUE
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

// Verifies: SYS-REQ-013 [fuzz]
// MCDC SYS-REQ-013: N/A
func TestFuzzParseFloatHarnessCoverage(t *testing.T) {
	if got := FuzzParseFloat([]byte(`1.25`)); got != 1 {
		t.Fatalf("expected FuzzParseFloat success path to return 1, got %d", got)
	}
	if got := FuzzParseFloat([]byte(`1.2.3`)); got != 0 {
		t.Fatalf("expected FuzzParseFloat failure path to return 0, got %d", got)
	}
}

// Verifies: STK-REQ-001 [boundary]
// MCDC STK-REQ-001: N/A
func TestValueTypeString(t *testing.T) {
	cases := []struct {
		value    ValueType
		expected string
	}{
		{NotExist, "non-existent"},
		{String, "string"},
		{Number, "number"},
		{Object, "object"},
		{Array, "array"},
		{Boolean, "boolean"},
		{Null, "null"},
		{Unknown, "unknown"},
		{ValueType(255), "unknown"},
	}

	for _, tc := range cases {
		if got := tc.value.String(); got != tc.expected {
			t.Fatalf("ValueType(%d).String() = %q, want %q", tc.value, got, tc.expected)
		}
	}
}

// Verifies: STK-REQ-001 [boundary]
// MCDC STK-REQ-001: N/A
func TestTokenStart(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "comma separator", input: `{"a":1,"b":2`, expected: 6},
		{name: "array separator", input: `[1,2`, expected: 2},
		{name: "opening object", input: `{"a":1`, expected: 0},
		{name: "no separator", input: `value`, expected: 0},
	}

	for _, tc := range cases {
		if got := tokenStart([]byte(tc.input)); got != tc.expected {
			t.Fatalf("%s: tokenStart(%q) = %d, want %d", tc.name, tc.input, got, tc.expected)
		}
	}
}

var parseStringTest = []ParseTest{
	{
		in:     ``,
		intype: String,
		out:    "",
	},
	{
		in:     `\uFF11`,
		intype: String,
		out:    "\uFF11",
	},
	{
		in:     `line\nbreak`,
		intype: String,
		out:    "line\nbreak",
	},
	{
		in:     `\uFFFF`,
		intype: String,
		out:    "\uFFFF",
	},
	{
		in:     `\uDF00`,
		intype: String,
		isErr:  true,
	},
}

// Verifies: SYS-REQ-038 [example]
// MCDC SYS-REQ-038: raw_string_literal_is_well_formed=F, returns_parsestring_error=T => TRUE
// Verifies: SYS-REQ-014 [example]
// STK-REQ-007:AC-3:acceptance
// MCDC SYS-REQ-014: raw_string_literal_is_well_formed=F, returns_parsestring_value=F => TRUE
// MCDC SYS-REQ-014: raw_string_literal_is_well_formed=T, returns_parsestring_value=F => FALSE
// MCDC SYS-REQ-014: raw_string_literal_is_well_formed=T, returns_parsestring_value=T => TRUE
func TestParseString(t *testing.T) {
	runParseTests(t, "ParseString()", parseStringTest,
		func(test ParseTest) (value interface{}, err error) {
			return ParseString([]byte(test.in))
		},
		func(test ParseTest, obtained interface{}) (bool, interface{}) {
			expected := test.out.(string)
			return obtained.(string) == expected, expected
		},
	)
}

// Verifies: SYS-REQ-040 [example]
// MCDC SYS-REQ-040: raw_int_token_is_well_formed=F, raw_int_token_overflows_int64=F, returns_parseint_malformed_error=T => TRUE
// Verifies: SYS-REQ-039 [example]
// MCDC SYS-REQ-039: raw_int_token_overflows_int64=T, returns_parseint_overflow_error=T => TRUE
// Verifies: SYS-REQ-015 [example]
// STK-REQ-007:AC-4:acceptance
// MCDC SYS-REQ-015: raw_int_token_is_well_formed=F, returns_parseint_value=F => TRUE
// MCDC SYS-REQ-015: raw_int_token_is_well_formed=T, returns_parseint_value=F => FALSE
// MCDC SYS-REQ-015: raw_int_token_is_well_formed=T, returns_parseint_value=T => TRUE
func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int64
		wantErr error
	}{
		{
			name: "zero",
			in:   "0",
			want: 0,
		},
		{
			name: "negative integer",
			in:   "-12345",
			want: -12345,
		},
		{
			name: "max int64",
			in:   "9223372036854775807",
			want: 9223372036854775807,
		},
		{
			name:    "empty input",
			in:      "",
			wantErr: MalformedValueError,
		},
		{
			name:    "fractional token",
			in:      "1.2",
			wantErr: MalformedValueError,
		},
		{
			name:    "alpha suffix",
			in:      "123x",
			wantErr: MalformedValueError,
		},
		{
			name:    "overflow",
			in:      "9223372036854775808",
			wantErr: OverflowIntegerError,
		},
		{
			name:    "underflow",
			in:      "-9223372036854775809",
			wantErr: OverflowIntegerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseInt([]byte(test.in))
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("ParseInt(%q) error mismatch: expected %v, got %v", test.in, test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseInt(%q) returned unexpected error: %v", test.in, err)
			}
			if got != test.want {
				t.Fatalf("ParseInt(%q) value mismatch: expected %d, got %d", test.in, test.want, got)
			}
		})
	}
}
