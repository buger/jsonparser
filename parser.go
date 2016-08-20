package jsonparser

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Errors
var (
	KeyPathNotFoundError       = errors.New("Key path not found")
	UnknownValueTypeError      = errors.New("Unknown value type")
	MalformedJsonError         = errors.New("Malformed JSON error")
	MalformedStringError       = errors.New("Value is string, but can't find closing '\"' symbol")
	MalformedArrayError        = errors.New("Value is array, but can't find closing ']' symbol")
	MalformedObjectError       = errors.New("Value looks like object, but can't find closing '}' symbol")
	MalformedValueError        = errors.New("Value looks like Number/Boolean/None, but can't find its end: ',' or '}' symbol")
	MalformedStringEscapeError = errors.New("Encountered an invalid escape sequence in a string")
)

// How much stack space to allocate for unescaping JSON strings; if a string longer
// than this needs to be escaped, it will result in a heap allocation
const unescapeStackBufSize = 64

func tokenEnd(data []byte) int {
	for i, c := range data {
		switch c {
		case ' ', '\n', '\r', '\t', ',', '}', ']':
			return i
		}
	}

	return -1
}

// Find position of next character which is not whitespace
func nextToken(data []byte) int {
	for i, c := range data {
		switch c {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return i
		}
	}

	return -1
}

// Tries to find the end of string
// Support if string contains escaped quote symbols.
func stringEnd(data []byte) (int, bool) {
	escaped := false
	for i, c := range data {
		if c == '"' {
			if !escaped {
				return i + 1, false
			} else {
				j := i - 1
				for {
					if j < 0 || data[j] != '\\' {
						return i + 1, true // even number of backslashes
					}
					j--
					if j < 0 || data[j] != '\\' {
						break // odd number of backslashes
					}
					j--

				}
			}
		} else if c == '\\' {
			escaped = true
		}
	}

	return -1, escaped
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
			se, _ := stringEnd(data[i+1:])
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

	var stackbuf [unescapeStackBufSize]byte // stack-allocated array for allocation-free unescaping of small strings

	for i < ln {
		switch data[i] {
		case '"':
			i++
			keyBegin := i

			strEnd, keyEscaped := stringEnd(data[i:])
			if strEnd == -1 {
				return -1
			}
			i += strEnd
			keyEnd := i - 1

			valueOffset := nextToken(data[i:])
			if valueOffset == -1 {
				return -1
			}

			i += valueOffset

			// if string is a key, and key level match
			if data[i] == ':' && keyLevel == level-1 {
				key := data[keyBegin:keyEnd]

				// for unescape: if there are no escape sequences, this is cheap; if there are, it is a
				// bit more expensive, but causes no allocations unless len(key) > unescapeStackBufSize
				var keyUnesc []byte
				if !keyEscaped {
					keyUnesc = key
				} else if ku, err := Unescape(key, stackbuf[:]); err != nil {
					return -1
				} else {
					keyUnesc = ku
				}
				if equalStr(&keyUnesc, keys[level-1]) {
					keyLevel++
					// If we found all keys in path
					if keyLevel == lk {
						return i + 1
					}
				}
			} else {
				i--
			}
		case '{':
			level++
		case '}':
			level--
		case '[':
			curKey := keys[keyLevel]
			// valid key to array value follows this struture: [<int>]
			// ex: [2] would extract 3 from the following slice [1, 2, 3]
			if curKey[0] == '[' && len(curKey) > 2 && curKey[len(curKey)-1] == ']' {
				if index, err := strconv.Atoi(curKey[1 : len(curKey)-1]); err == nil {
					i++
					level++
					keyLevel++
					for index > 0 {
						_, _, o, e := Get(data[i:])
						if o == 0 {
							break
						} else if e != nil {
							return -1
						}
						i += o

						if skipToToken := nextToken(data[i:]); skipToToken != -1 {
							i += skipToToken
							if data[i] == ']' {
								break
							} else if data[i] != ',' {
								return -1
							}
							i++
							index--
						} else {
							return -1
						}
					}
					i += nextToken(data[i:])
					if index != 0 { // if keys have been looped through and array is smaller then expected return
						return -1
					} else if keyLevel == lk {
						return i
					} else {
						i--
					}
				} else {
					return -1
				}
			} else {
				// Do not search for keys inside arrays unless key dictates otherwise
				arraySkip := blockEnd(data[i:], '[', ']')
				i += arraySkip - 1
			}
		}

		i++
	}

	return -1
}

var bitwiseFlags []int64

func init() {
	for i := 0; i < 63; i++ {
		bitwiseFlags = append(bitwiseFlags, int64(math.Pow(2, float64(i))))
	}
}

func EachKey(data []byte, cb func(int, []byte, ValueType, error), paths ...[]string) int {
	var pathFlags int64
	var level, pathsMatched, i int
	ln := len(data)

	var stackbuf [unescapeStackBufSize]byte // stack-allocated array for allocation-free unescaping of small strings

	for i < ln {
		switch data[i] {
		case '"':
			i++
			keyBegin := i

			strEnd, keyEscaped := stringEnd(data[i:])
			if strEnd == -1 {
				return -1
			}
			i += strEnd

			keyEnd := i - 1

			valueOffset := nextToken(data[i:])
			if valueOffset == -1 {
				return -1
			}

			i += valueOffset

			// if string is a key, and key level match
			if data[i] == ':' {
				match := false
				key := data[keyBegin:keyEnd]

				// for unescape: if there are no escape sequences, this is cheap; if there are, it is a
				// bit more expensive, but causes no allocations unless len(key) > unescapeStackBufSize
				var keyUnesc []byte
				if !keyEscaped {
					keyUnesc = key
				} else if ku, err := Unescape(key, stackbuf[:]); err != nil {
					return -1
				} else {
					keyUnesc = ku
				}

				for pi, p := range paths {
					if len(p) < level || (pathFlags&bitwiseFlags[pi]) != 0 {
						continue
					}

					if equalStr(&keyUnesc, p[level-1]) {
						match = true

						if len(p) == level {
							i++
							pathsMatched++
							pathFlags |= bitwiseFlags[pi]

							v, dt, of, e := Get(data[i:])
							cb(pi, v, dt, e)

							if of != -1 {
								i += of
							}

							if pathsMatched == len(paths) {
								return i
							}
						}
					}
				}

				if !match {
					tokenOffset := nextToken(data[i+1:])
					i += tokenOffset

					if data[i] == '{' {
						blockSkip := blockEnd(data[i:], '{', '}')
						i += blockSkip + 1
					}
				}
			} else {
				i--
			}
		case '{':
			level++
		case '}':
			level--
		case '[':
			// Do not search for keys inside arrays
			arraySkip := blockEnd(data[i:], '[', ']')
			i += arraySkip - 1
		}

		i++
	}

	return -1
}

// Data types available in valid JSON data.
type ValueType int

const (
	NotExist = ValueType(iota)
	String
	Number
	Object
	Array
	Boolean
	Null
	Unknown
)

var (
	trueLiteral  = []byte("true")
	falseLiteral = []byte("false")
	nullLiteral  = []byte("null")
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
func Get(data []byte, keys ...string) (value []byte, dataType ValueType, offset int, err error) {
	if len(keys) > 0 {
		if offset = searchKeys(data, keys...); offset == -1 {
			return []byte{}, NotExist, -1, KeyPathNotFoundError
		}
	}

	// Go to closest value
	nO := nextToken(data[offset:])
	if nO == -1 {
		return []byte{}, NotExist, -1, MalformedJsonError
	}

	offset += nO

	endOffset := offset
	// if string value
	if data[offset] == '"' {
		dataType = String
		if idx, _ := stringEnd(data[offset+1:]); idx != -1 {
			endOffset += idx + 1
		} else {
			return []byte{}, dataType, offset, MalformedStringError
		}
	} else if data[offset] == '[' { // if array value
		dataType = Array
		// break label, for stopping nested loops
		endOffset = blockEnd(data[offset:], '[', ']')

		if endOffset == -1 {
			return []byte{}, dataType, offset, MalformedArrayError
		}

		endOffset += offset
	} else if data[offset] == '{' { // if object value
		dataType = Object
		// break label, for stopping nested loops
		endOffset = blockEnd(data[offset:], '{', '}')

		if endOffset == -1 {
			return []byte{}, dataType, offset, MalformedObjectError
		}

		endOffset += offset
	} else {
		// Number, Boolean or None
		end := tokenEnd(data[endOffset:])

		if end == -1 {
			return nil, dataType, offset, MalformedValueError
		}

		value := data[offset : endOffset+end]

		switch data[offset] {
		case 't', 'f': // true or false
			if bytes.Equal(value, trueLiteral) || bytes.Equal(value, falseLiteral) {
				dataType = Boolean
			} else {
				return nil, Unknown, offset, UnknownValueTypeError
			}
		case 'u', 'n': // undefined or null
			if bytes.Equal(value, nullLiteral) {
				dataType = Null
			} else {
				return nil, Unknown, offset, UnknownValueTypeError
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			dataType = Number
		default:
			return nil, Unknown, offset, UnknownValueTypeError
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
func ArrayEach(data []byte, cb func(value []byte, dataType ValueType, offset int, err error), keys ...string) (err error) {
	if len(data) == 0 {
		return MalformedObjectError
	}

	offset := 1

	if len(keys) > 0 {
		if offset = searchKeys(data, keys...); offset == -1 {
			return KeyPathNotFoundError
		}

		// Go to closest value
		nO := nextToken(data[offset:])
		if nO == -1 {
			return MalformedJsonError
		}

		offset += nO

		if data[offset] != '[' {
			return MalformedArrayError
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

		skipToToken := nextToken(data[offset:])
		if skipToToken == -1 {
			return MalformedArrayError
		}
		offset += skipToToken

		if data[offset] == ']' {
			break
		}

		if data[offset] != ',' {
			return MalformedArrayError
		}

		offset++
	}

	return nil
}

// GetUnsafeString returns the value retrieved by `Get`, use creates string without memory allocation by mapping string to slice memory. It does not handle escape symbols.
func GetUnsafeString(data []byte, keys ...string) (val string, err error) {
	v, _, _, e := Get(data, keys...)

	if e != nil {
		return "", e
	}

	return bytesToString(&v), nil
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

	return ParseString(v)
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

	return ParseFloat(v)
}

// GetInt returns the value retrieved by `Get`, cast to a int64 if possible.
// If key data type do not match, it will return an error.
func GetInt(data []byte, keys ...string) (val int64, err error) {
	v, t, _, e := Get(data, keys...)

	if e != nil {
		return 0, e
	}

	if t != Number {
		return 0, fmt.Errorf("Value is not a number: %s", string(v))
	}

	return ParseInt(v)
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

	return ParseBoolean(v)
}

// ParseBoolean parses a Boolean ValueType into a Go bool (not particularly useful, but here for completeness)
func ParseBoolean(b []byte) (bool, error) {
	switch {
	case bytes.Equal(b, trueLiteral):
		return true, nil
	case bytes.Equal(b, falseLiteral):
		return false, nil
	default:
		return false, MalformedValueError
	}
}

// ParseString parses a String ValueType into a Go string (the main parsing work is unescaping the JSON string)
func ParseString(b []byte) (string, error) {
	var stackbuf [unescapeStackBufSize]byte // stack-allocated array for allocation-free unescaping of small strings
	if bU, err := Unescape(b, stackbuf[:]); err != nil {
		return "", nil
	} else {
		return string(bU), nil
	}
}

// ParseNumber parses a Number ValueType into a Go float64
func ParseFloat(b []byte) (float64, error) {
	if v, err := parseFloat(&b); err != nil {
		return 0, MalformedValueError
	} else {
		return v, nil
	}
}

// ParseInt parses a Number ValueType into a Go int64
func ParseInt(b []byte) (int64, error) {
	if v, ok := parseInt(b); !ok {
		return 0, MalformedValueError
	} else {
		return v, nil
	}
}
