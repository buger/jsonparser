package jsonparser

import (
	"bytes"
	"errors"
	"fmt"
	d "runtime/debug"
	"strconv"
)

// Find position of next character which is not whitespace
func skipWhitespace(data []byte) int {
	for i, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return i
		}
	}
	return -1
}

// Find position of next character which is not whitespace, ',', '}' or ']'
func nextValue(data []byte) int {
	for i, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t', ',', '}', ']':
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
	for i, c := range data {
		if c == '"' {
			if i >= 1 && data[i-1] == '\\' {
				continue
			} else {
				return i + 1
			}
		}
	}

	return -1
}

// Find end of the data structure, array or object.
// For array openSym and closeSym will be '[' and ']', for object '{' and '}'
// Know about nested structures
func trailingBracket(data []byte, openSym byte, closeSym byte) int {
	level := 0
	i := 0
	ln := len(data)

	for i < ln {
		c := data[i]

		// If a string is encountered, skip everything in it
		switch c {
		case '"':
			se := stringEnd(data[i+1:])
			if se == -1 {
				return -1
			}

			i += 1 + se // skip the initial quote plus the rest of the string
		case openSym:
			level++
			i++
		case closeSym:
			level--
			i++
		default:
			i++
		}

		if level == 0 {
			return i
		}
	}

	return -1
}

func fastStringBytesEqual(a string, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i, c := range b {
		if a[i] != c {
			return false
		}
	}

	return true
}

func searchKeys(data []byte, keys ...string) (int, error) {
	curKey := keys[0]
	keyLevel := 0
	level := 0
	i := 0
	ln := len(data)
	lk := len(keys)

	for i < ln {
		switch data[i] {
		// If a key string is encountered, check if it matches our key; if so, increase our key level.
		// In any case, also skip it.
		case '"':
			i++

			se := stringEnd(data[i:])
			if se == -1 {
				return -1, errors.New("unterminated key string")
			}

			wsSkip := skipWhitespace(data[i+se:])
			if wsSkip == -1 {
				return -1, errors.New("key string with no following colon")
			}

			if data[i+se+wsSkip] == ':' && // if string is a key, and key level match
				keyLevel == level-1 && // If key nesting level match current object nested level
				fastStringBytesEqual(curKey, data[i:i+se-1]) {
				keyLevel++
				// If we found all keys in path
				if keyLevel == lk {
					return i + se + wsSkip + 1, nil
				} else {
					curKey = keys[keyLevel]
				}
			}

			i += se + wsSkip
		case '{':
			level++
			i++
		case '}':
			level--
			i++
		case '[':
			// Do not search for keys inside arrays
			aOff := trailingBracket(data[i:], '[', ']')
			if aOff == -1 {
				return -1, errors.New("unterminated array")
			}

			i += aOff + 1
		default:
			i++
		}
	}

	return -1, nil // not found
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
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Unhandler JSON parsing error: %v, %s", r, string(d.Stack()))
		}
	}()

	if len(keys) > 0 {
		if offset, err = searchKeys(data, keys...); offset == -1 {
			return nil, NotExist, -1, err
		}
	}

	// Go to closest value
	nO := nextValue(data[offset:])

	if nO == -1 {
		return nil, NotExist, -1, errors.New("Malformed JSON error")
	}

	offset += nO

	endOffset := offset

	// if string value
	if data[offset] == '"' {
		dataType = String
		if idx := stringEnd(data[offset+1:]); idx != -1 {
			endOffset += idx + 1
		} else {
			return nil, dataType, offset, errors.New("Value is string, but can't find closing '\"' symbol")
		}
	} else if data[offset] == '[' { // if array value
		dataType = Array
		// break label, for stopping nested loops
		endOffset = trailingBracket(data[offset:], '[', ']')

		if endOffset == -1 {
			return nil, dataType, offset, errors.New("Value is array, but can't find closing ']' symbol")
		}

		endOffset += offset
	} else if data[offset] == '{' { // if object value
		dataType = Object
		// break label, for stopping nested loops
		endOffset = trailingBracket(data[offset:], '{', '}')

		if endOffset == -1 {
			return nil, dataType, offset, errors.New("Value looks like object, but can't find closing '}' symbol")
		}

		endOffset += offset
	} else {
		// Number, Boolean or None
		end := bytes.IndexFunc(data[endOffset:], func(c rune) bool {
			return c == ' ' || c == '\n' || c == ',' || c == '}' || c == ']'
		})

		if data[offset] == 't' || data[offset] == 'f' { // true or false
			dataType = Boolean
		} else if data[offset] == 'u' || data[offset] == 'n' { // undefined or null
			dataType = Null
		} else {
			dataType = Number
		}

		if end == -1 {
			return nil, dataType, offset, errors.New("Value looks like Number/Boolean/None, but can't find its end: ',' or '}' symbol")
		}

		endOffset += end
	}

	value = data[offset:endOffset]

	// Strip quotes from string values
	if dataType == String {
		value = value[1 : len(value)-1]
	}

	if dataType == Null {
		value = nil
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
		if offset, err = searchKeys(data, keys...); offset == -1 {
			return errors.New("Key path not found: " + err.Error())
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

// GetNumber returns the value retrieved by `Get`, cast to a float64 if possible.
// The offset is the same as in `Get`.
// If key data type do not match, it will return an error.
func GetNumber(data []byte, keys ...string) (val float64, offset int, err error) {
	v, t, offset, e := Get(data, keys...)

	if e != nil {
		return 0, offset, e
	}

	if t != Number {
		return 0, offset, fmt.Errorf("Value is not a number: %s", string(v))
	}

	val, err = strconv.ParseFloat(string(v), 64)
	return
}

// GetBoolean returns the value retrieved by `Get`, cast to a bool if possible.
// The offset is the same as in `Get`.
// If key data type do not match, it will return error.
func GetBoolean(data []byte, keys ...string) (val bool, offset int, err error) {
	v, t, offset, e := Get(data, keys...)

	if e != nil {
		return false, offset, e
	}

	if t != Boolean {
		return false, offset, fmt.Errorf("Value is not a boolean: %s", string(v))
	}

	if v[0] == 't' {
		val = true
	} else {
		val = false
	}

	return
}
