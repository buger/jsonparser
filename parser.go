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
func nextValue(data []byte) (offset int) {
	for true {
		if len(data) == offset {
			return -1
		}

		if data[offset] != ' ' && data[offset] != '\n' && data[offset] != ',' {
			return
		}

		offset++
	}

	return -1
}

// Tries to find the end of string
// Support if string contains escaped quote symbols.
func stringEnd(data []byte) int {
	i := 0

	for len(data) > i {
		if data[i] != '"' {
			i++
			continue
		}

		// If it just escaped \", continue
		if i >= 1 && data[i-1] == '\\' {
			i++
			continue
		} else {
			return i + 1
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

	for true {
		if i >= ln {
			return -1
		}

		c := data[i]

		// If inside string, skip it
		if c == '"' {
			//sFrom := i
			i++

			se := stringEnd(data[i:])
			if se == -1 {
				return -1
			}
			i += se - 1
		}

		if c == openSym {
			level++
		} else if c == closeSym {
			level--
		}

		i++

		if level == 0 {
			break
		}
	}

	return i
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

		// If inside string, skip it
		if data[i] == '"' {
			i++

			se := stringEnd(data[i:])
			if se == -1 {
				return -1
			}

			if ln > i+se &&
				data[i+se] == ':' && // if string is a Key, and key level match
				keyLevel == level-1 && // If key nesting level match current object nested level

				// Checks to speedup key comparsion
				len(keys[level-1]) == se-1 && // if it have same length
				data[i] == keys[level-1][0] { // If first character same
				if bytes.Equal([]byte(keys[level-1]), data[i:i+se-1]) {
					keyLevel++
					// If we found all keys in path
					if keyLevel == lk {
						return i + se + 1
					}
				}
			}

			i += se - 1
		} else if data[i] == '{' {
			level++
		} else if data[i] == '}' {
			level--
		} else if data[i] == '[' {
			// Do not search for keys inside arrays
			aOff := trailingBracket(data[i:], '[', ']')
			i += aOff
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
		endOffset = trailingBracket(data[offset:], '[', ']')

		if endOffset == -1 {
			return []byte{}, dataType, offset, errors.New("Value is array, but can't find closing ']' symbol")
		}

		endOffset += offset
	} else if data[offset] == '{' { // if object value
		dataType = Object
		// break label, for stopping nested loops
		endOffset = trailingBracket(data[offset:], '{', '}')

		if endOffset == -1 {
			return []byte{}, dataType, offset, errors.New("Value looks like object, but can't find closing '}' symbol")
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
			return []byte{}, dataType, offset, errors.New("Value looks like Number/Boolean/None, but can't find its end: ',' or '}' symbol")
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
