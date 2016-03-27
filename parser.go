package jsonparser

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

// Find position of next character which is not ' ', ',', '}' or ']'
func nextValue(data []byte) int {
	for i, c := range data {
		switch c {
		case ' ', '\n', '\r', '\t', ',':
			continue
		default:
			return i
		}
	}

	return -1
}

// Tries to find the end of string
// Support if string contains escaped quote symbols.
func stringEnd(data []byte) int {
	escaped := false
	for i, c := range data {
		switch c {
		case '\\':
			escaped = !escaped
		case '"':
			if !escaped {
				return i + 1
			}
			escaped = false
		default:
			escaped = false
		}
	}

	return -1
}

// Find end of the data structure, array or object.
// For array openSym and closeSym will be '[' and ']', for object '{' and '}'
func blockEnd(data []byte, openSym byte, closeSym byte) int {
	level := 0
	i := 0
	ln := len(data)

	for i < ln {
		switch data[i] {
		case '"': // If inside string, skip it
			se := stringEnd(data[i+1:])
			if se == -1 {
				return -1
			}
			i += se
		case openSym: // If open symbol, increase level
			level++
		case closeSym: // If close symbol, increase level
			level--

			// If we have returned to the original level, we're done
			if level == 0 {
				return i + 1
			}
		}
		i++
	}

	return -1
}

func searchKeys(data []byte, keys ...string) int {
	keyLevel := 0
	level := 0
	i := 0
	ln := len(data)
	lk := len(keys)

	for true {
		if i >= ln {
			return -1
		}

		switch data[i] {
		case '"':
			i++

			se := stringEnd(data[i:])
			if se == -1 {
				return -1
			}

			nO := nextValue(data[i+se:])

			if nO == -1 {
				return -1
			}

			if ln > i+se+nO &&
				data[i+se+nO] == ':' && // if string is a Key, and key level match
				keyLevel == level-1 && // If key nesting level match current object nested level

				// Checks to speedup key comparsion
				len(keys[level-1]) == se-1 && // if it have same length
				data[i] == keys[level-1][0] { // If first character same
				if keys[level-1] == UnsafeBytesToString(data[i:i+se-1]) {
					keyLevel++
					// If we found all keys in path
					if keyLevel == lk {
						return i + se + nO + 1
					}
				}
			}

			i += se + nO - 1
		case '{':
			level++
		case '}':
			level--
		case '[':
			// Do not search for keys inside arrays
			arraySkip := blockEnd(data[i:], '[', ']')
			i += arraySkip
		}

		i++
	}

	return -1
}

// Data types available in valid JSON data.
const (
	NotExist = iota
	String
	Number
	Object
	Array
	Boolean
	Null
	Unknown
)

/*
Get - Receives data structure, and key path to extract value from.

Returns:
`value` - Pointer to original data structure containing key value, or just empty slice if nothing found or error
`dataType` -    Can be: `NotExist`, `String`, `Number`, `Object`, `Array`, `Boolean` or `Null`
`offset` - Offset from provided data structure where key value ends. Used mostly internally, for example for `ArrayEach` helper.
`err` - If key not found or any other parsing issue it should return error. If key not found it also sets `dataType` to `NotExist`

Accept multiple keys to specify path to JSON value (in case of quering nested structures).
If no keys provided it will try to extract closest JSON value (simple ones or object/array), useful for reading streams or arrays, see `ArrayEach` implementation.
*/
func Get(data []byte, keys ...string) (value []byte, dataType int, offset int, err error) {
	if len(keys) > 0 {
		if offset = searchKeys(data, keys...); offset == -1 {
			return []byte{}, NotExist, -1, errors.New("Key path not found")
		}
	}

	// Go to closest value
	nO := nextValue(data[offset:])

	if nO == -1 {
		return []byte{}, NotExist, -1, errors.New("Malformed JSON error")
	}

	offset += nO

	endOffset := offset

	// if string value
	if data[offset] == '"' {
		dataType = String
		if idx := stringEnd(data[offset+1:]); idx != -1 {
			endOffset += idx + 1
		} else {
			return []byte{}, dataType, offset, errors.New("Value is string, but can't find closing '\"' symbol")
		}
	} else if data[offset] == '[' { // if array value
		dataType = Array
		// break label, for stopping nested loops
		endOffset = blockEnd(data[offset:], '[', ']')

		if endOffset == -1 {
			return []byte{}, dataType, offset, errors.New("Value is array, but can't find closing ']' symbol")
		}

		endOffset += offset
	} else if data[offset] == '{' { // if object value
		dataType = Object
		// break label, for stopping nested loops
		endOffset = blockEnd(data[offset:], '{', '}')

		if endOffset == -1 {
			return []byte{}, dataType, offset, errors.New("Value looks like object, but can't find closing '}' symbol")
		}

		endOffset += offset
	} else {
		// Number, Boolean or None
		end := bytes.IndexFunc(data[endOffset:], func(c rune) bool {
			return c == ' ' || c == '\n' || c == ',' || c == '}' || c == ']'
		})

		if end == -1 {
			return nil, dataType, offset, errors.New("Value looks like Number/Boolean/None, but can't find its end: ',' or '}' symbol")
		}

		value := UnsafeBytesToString(data[offset : endOffset+end])

		switch data[offset] {
		case 't', 'f': // true or false
			if (len(value) == 4 && value == "true") || (len(value) == 5 && value == "false") {
				dataType = Boolean
			} else {
				return nil, Unknown, offset, errors.New("Unknown value type")
			}
		case 'u', 'n': // undefined or null
			if len(value) == 4 && value == "null" {
				dataType = Null
			} else {
				return nil, Unknown, offset, errors.New("Unknown value type")
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			dataType = Number
		default:
			return nil, Unknown, offset, errors.New("Unknown value type")
		}

		endOffset += end
	}

	value = data[offset:endOffset]

	// Strip quotes from string values
	if dataType == String {
		value = value[1 : len(value)-1]
	}

	if dataType == Null {
		value = []byte{}
	}

	return value, dataType, endOffset, nil
}

// ArrayEach is used when iterating arrays, accepts a callback function with the same return arguments as `Get`.
func ArrayEach(data []byte, cb func(value []byte, dataType int, offset int, err error), keys ...string) (err error) {
	if len(data) == 0 {
		return errors.New("Object is empty")
	}

	offset := 1

	if len(keys) > 0 {
		if offset = searchKeys(data, keys...); offset == -1 {
			return errors.New("Key path not found")
		}

		// Go to closest value
		nO := nextValue(data[offset:])

		if nO == -1 {
			return errors.New("Malformed JSON")
		}

		offset += nO

		if data[offset] != '[' {
			return errors.New("Value is not array")
		}

		offset++
	}

	for true {
		v, t, o, e := Get(data[offset:])

		if o == 0 {
			break
		}

		if t != NotExist {
			cb(v, t, o, e)
		}

		if e != nil {
			break
		}

		offset += o
	}

	return nil
}

// GetString returns the value retrieved by `Get`, cast to a string if possible, trying to properly handle escape and utf8 symbols
// If key data type do not match, it will return an error.
func GetString(data []byte, keys ...string) (val string, err error) {
	v, t, _, e := Get(data, keys...)

	if e != nil {
		return "", e
	}

	if t != String {
		return "", fmt.Errorf("Value is not a number: %s", string(v))
	}

	// If no escapes return raw conten
	if bytes.IndexByte(v, '\\') == -1 {
		return string(v), nil
	}

	s, err := strconv.Unquote(`"` + UnsafeBytesToString(v) + `"`)

	return s, err
}

// GetFloat returns the value retrieved by `Get`, cast to a float64 if possible.
// The offset is the same as in `Get`.
// If key data type do not match, it will return an error.
func GetFloat(data []byte, keys ...string) (val float64, err error) {
	v, t, _, e := Get(data, keys...)

	if e != nil {
		return 0, e
	}

	if t != Number {
		return 0, fmt.Errorf("Value is not a number: %s", string(v))
	}

	val, err = strconv.ParseFloat(UnsafeBytesToString(v), 64)
	return
}

// GetInt returns the value retrieved by `Get`, cast to a float64 if possible.
// If key data type do not match, it will return an error.
func GetInt(data []byte, keys ...string) (val int64, err error) {
	v, t, _, e := Get(data, keys...)

	if e != nil {
		return 0, e
	}

	if t != Number {
		return 0, fmt.Errorf("Value is not a number: %s", string(v))
	}

	val, err = strconv.ParseInt(UnsafeBytesToString(v), 10, 64)
	return
}

// GetBoolean returns the value retrieved by `Get`, cast to a bool if possible.
// The offset is the same as in `Get`.
// If key data type do not match, it will return error.
func GetBoolean(data []byte, keys ...string) (val bool, err error) {
	v, t, _, e := Get(data, keys...)

	if e != nil {
		return false, e
	}

	if t != Boolean {
		return false, fmt.Errorf("Value is not a boolean: %s", string(v))
	}

	if v[0] == 't' {
		val = true
	} else {
		val = false
	}

	return
}

// A hack until issue golang/go#2632 is fixed.
// See: https://github.com/golang/go/issues/2632
func UnsafeBytesToString(data []byte) string {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	sh := reflect.StringHeader{Data: h.Data, Len: h.Len}
	return *(*string)(unsafe.Pointer(&sh))
}
