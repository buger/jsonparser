package jsonparser

import (
	"bytes"
	"errors"
	"fmt"
	d "runtime/debug"
	"strconv"
)

// Find position of next character which is not ' ', ',', '}' or ']'
func nextValue(data []byte) (offset int) {
	for true {
		if len(data) == offset {
			return -1
		}

		if data[offset] != ' ' && data[offset] != '\n' && data[offset] != ',' && data[offset] != '}' && data[offset] != ']' {
			return
		}

		offset += 1
	}

	return -1
}

// Tries to find the end of string
// Support if string contains escaped quote symbols.
func stringEnd(data []byte) int {
	i := 0

	for true {
		sIdx := bytes.IndexByte(data[i:], '"')

		if sIdx == -1 {
			return -1
		}

		i += sIdx + 1

		// If it just escaped \", continue
		if i > 2 && data[i-2] == '\\' {
			continue
		}

		break
	}

	return i
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
			level += 1
		} else if c == closeSym {
			level -= 1
		}

		i++

		if level == 0 {
			break
		}
	}

	return i
}

const (
	NOT_EXIST = iota
	STRING
	NUMBER
	OBJECT
	ARRAY
	BOOLEAN
	NULL
)

/*
Get - Receives data structure, and key path to extract value from.

Returns:
`value` - Pointer to original data structure containing key value, or just empty slice if nothing found or error
`dataType` -    Can be: `NOT_EXIST`, `STRING`, `NUMBER`, `OBJECT`, `ARRAY`, `BOOLEAN` or `NULL`
`offset` - Offset from provided data structure where key value ends. Used mostly internally, for example for `ArrayEach` helper.
`err` - If key not found or any other parsing issue it should return error. If key not found it also sets `dataType` to `NOT_EXISTS`

Accept multiple keys to specify path to JSON value (in case of quering nested structures).
If no keys provided it will try to extract closest JSON value (simple ones or object/array), useful for reading streams or arrays, see `ArrayEach` implementation.
*/
func Get(data []byte, keys ...string) (value []byte, dataType int, offset int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("Unhandler JSON parsing error: %v, %s", r, string(d.Stack())))
		}
	}()

	ln := len(data)

	if len(keys) > 0 {
		for _, k := range keys {
			lk := len(k)

			for true {
				if idx := bytes.Index(data[offset:], []byte(k)); idx != -1 && (ln-(offset+idx+lk+2)) > 0 {
					offset += idx

					if data[offset+lk] == '"' && data[offset-1] == '"' && data[offset+lk+1] == ':' {
						offset += lk + 2
						nO := nextValue(data[offset:])

						if nO == -1 {
							return []byte{}, NOT_EXIST, -1, errors.New("Malformed JSON error")
						}

						offset += nO

						break
					} else {
						offset += 1
					}
				} else {
					return []byte{}, NOT_EXIST, -1, errors.New("Key path not found")
				}
			}
		}
	} else {
		nO := nextValue(data[offset:])

		if nO == -1 {
			return []byte{}, NOT_EXIST, -1, errors.New("Malformed JSON error")
		}

		offset = nO
	}

	endOffset := offset

	// if string value
	if data[offset] == '"' {
		dataType = STRING
		if idx := stringEnd(data[offset+1:]); idx != -1 {
			endOffset += idx + 1
		} else {
			return []byte{}, dataType, offset, errors.New("Value is string, but can't find closing '\"' symbol")
		}
	} else if data[offset] == '[' { // if array value
		dataType = ARRAY
		// break label, for stopping nested loops
		endOffset = trailingBracket(data[offset:], '[', ']')

		if endOffset == -1 {
			return []byte{}, dataType, offset, errors.New("Value is array, but can't find closing ']' symbol")
		}

		endOffset += offset
	} else if data[offset] == '{' { // if object value
		dataType = OBJECT
		// break label, for stopping nested loops
		endOffset = trailingBracket(data[offset:], '{', '}')

		if endOffset == -1 {
			return []byte{}, dataType, offset, errors.New("Value looks like object, but can't find closing '}' symbol")
		}

		endOffset += offset
	} else {
		// Number, Boolean or None
		end := bytes.IndexFunc(data[endOffset:], func(c rune) bool {
			return c == ',' || c == '}' || c == ']'
		})

		if data[offset] == 't' || data[offset] == 'f' { // true or false
			dataType = BOOLEAN
		} else if data[offset] == 'u' || data[offset] == 'n' { // undefined or null
			dataType = NULL
		} else {
			dataType = NUMBER
		}

		if end == -1 {
			return []byte{}, dataType, offset, errors.New("Value looks like Number/Boolean/None, but can't find its end: ',' or '}' symbol")
		}

		endOffset += end
	}

	value = data[offset:endOffset]

	// Strip quotes from string values
	if dataType == STRING {
		value = value[1 : len(value)-1]
	}

	if dataType == NULL {
		value = []byte{}
	}

	return value, dataType, endOffset, nil
}

// Used for iterating arrays, accepts callback function with same return arguments as `Get`.
// Expects to receive array data structure (you need to `Get` it first). See example above.
// Underneeth it just calls `Get` without arguments until it can't find next item.
func ArrayEach(data []byte, cb func(value []byte, dataType int, offset int, err error)) {
	if len(data) == 0 {
		return
	}

	offset := 1
	for true {
		v, t, o, e := Get(data[offset:])

		if t != NOT_EXIST {
			cb(v, t, o, e)
		}

		if e != nil {
			break
		}

		offset += o
	}
}

// Returns same arguments as `Get` except `dataType`.
// If key data type do not match, it will return error.
func GetNumber(data []byte, keys ...string) (val float64, offset int, err error) {
	v, t, offset, e := Get(data, keys...)

	if e != nil {
		return 0, offset, e
	}

	if t != NUMBER {
		return 0, offset, errors.New(fmt.Sprintf("Value is not a number: %s", string(v)))
	}

	val, err = strconv.ParseFloat(string(v), 64)
	return
}

// Returns same arguments as `Get` except `dataType`.
// If key data type do not match, it will return error.
func GetBoolean(data []byte, keys ...string) (val bool, offset int, err error) {
	v, t, offset, e := Get(data, keys...)

	if e != nil {
		return false, offset, e
	}

	if t != BOOLEAN {
		return false, offset, errors.New(fmt.Sprintf("Value is not a boolean: %s", string(v)))
	}

	if v['0'] == 't' {
		val = true
	} else {
		val = false
	}

	return
}
