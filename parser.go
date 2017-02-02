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

	return len(data)
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

	if lk == 0 {
		return 0
	}

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
			if level == keyLevel {
				keyLevel--
			}
		case '[':
			// If we want to get array element by index
			if keyLevel == level && keys[level][0] == '[' {
				aIdx, _ := strconv.Atoi(keys[level][1 : len(keys[level])-1])

				var curIdx int
				var valueFound []byte
				var valueOffset int

				ArrayEach(data[i:], func(value []byte, dataType ValueType, offset int, err error) {
					if curIdx == aIdx {
						valueFound = value
						valueOffset = offset
					}
					curIdx += 1
				})

				if valueFound == nil {
					return -1
				} else {
					return i + valueOffset + searchKeys(valueFound, keys[level+1:]...)
				}
			} else {
				// Do not search for keys inside arrays
				if arraySkip := blockEnd(data[i:], '[', ']'); arraySkip == -1 {
					return -1
				} else {
					i += arraySkip - 1
				}
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

func sameTree(p1, p2 []string) bool {
	minLen := len(p1)
	if len(p2) < minLen {
		minLen = len(p2)
	}

	for pi_1, p_1 := range p1[:minLen] {
		if p2[pi_1] != p_1 {
			return false
		}
	}

	return true
}

func EachKey(data []byte, cb func(int, []byte, ValueType, error), paths ...[]string) int {
	return eachKey(false, data, cb, paths...)
}

//eachKey, getfix specifies if we want the old or the new functionality of the get function
func eachKey(getfix bool, data []byte, cb func(int, []byte, ValueType, error), paths ...[]string) int {
	var pathFlags int64
	var level, pathsMatched, i int
	ln := len(data)

	var maxPath int
	for _, p := range paths {
		if len(p) > maxPath {
			maxPath = len(p)
		}
	}

	var stackbuf [unescapeStackBufSize]byte // stack-allocated array for allocation-free unescaping of small strings
	pathsBuf := make([]string, maxPath)

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
				match := -1
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

				if maxPath >= level {
					pathsBuf[level-1] = bytesToString(&keyUnesc)

					for pi, p := range paths {
						if len(p) != level || pathFlags&bitwiseFlags[pi+1] != 0 || !equalStr(&keyUnesc, p[level-1]) || !sameTree(p, pathsBuf[:level]) {
							continue
						}

						match = pi

						i++
						pathsMatched++
						pathFlags |= bitwiseFlags[pi+1]

						v, dt, of, e := get(getfix, data[i:])
						cb(pi, v, dt, e)

						if of != -1 {
							i += of
						}

						if pathsMatched == len(paths) {
							return i
						}
					}
				}

				if match == -1 {
					tokenOffset := nextToken(data[i+1:])
					i += tokenOffset

					if data[i] == '{' {
						blockSkip := blockEnd(data[i:], '{', '}')
						i += blockSkip + 1
					}
				}

				switch data[i] {
				case '{', '}', '[', '"':
					i--
				}
			} else {
				i--
			}
		case '{':
			level++
		case '}':
			level--
		case '[':
			var arrIdxFlags int64
			var pIdxFlags int64
			for pi, p := range paths {
				if len(p) < level+1 || pathFlags&bitwiseFlags[pi+1] != 0 || p[level][0] != '[' || !sameTree(p, pathsBuf[:level]) {
					continue
				}

				aIdx, _ := strconv.Atoi(p[level][1 : len(p[level])-1])
				arrIdxFlags |= bitwiseFlags[aIdx+1]
				pIdxFlags |= bitwiseFlags[pi+1]
			}

			if arrIdxFlags > 0 {
				level++

				var curIdx int
				arrOff, _ := ArrayEach(data[i:], func(value []byte, dataType ValueType, offset int, err error) {
					if arrIdxFlags&bitwiseFlags[curIdx+1] != 0 {
						for pi, p := range paths {
							if pIdxFlags&bitwiseFlags[pi+1] != 0 {
								aIdx, _ := strconv.Atoi(p[level-1][1 : len(p[level-1])-1])

								if curIdx == aIdx {
									of := searchKeys(value, p[level:]...)

									pathsMatched++
									pathFlags |= bitwiseFlags[pi+1]

									if of != -1 {
										v, dt, _, e := get(getfix, value[of:])
										cb(pi, v, dt, e)
									}
								}
							}
						}
					}

					curIdx += 1
				})

				if pathsMatched == len(paths) {
					return i
				}

				i += arrOff - 1
			} else {
				// Do not search for keys inside arrays
				if arraySkip := blockEnd(data[i:], '[', ']'); arraySkip == -1 {
					return -1
				} else {
					i += arraySkip - 1
				}
			}
		case ']':
			level--
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

func (vt ValueType) String() string {
	switch vt {
	case NotExist:
		return "non-existent"
	case String:
		return "string"
	case Number:
		return "number"
	case Object:
		return "object"
	case Array:
		return "array"
	case Boolean:
		return "boolean"
	case Null:
		return "null"
	default:
		return "unknown"
	}
}

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
//Get calls the internal get function, but will strip quotes from strings returned. (breaks abstraction, but kept for compatibility)
func Get(data []byte, keys ...string) (value []byte, dataType ValueType, offset int, err error) {
	return get(false, data, keys...)
}

func get(getfix bool, data []byte, keys ...string) (value []byte, dataType ValueType, offset int, err error) {
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

		constant := data[offset : endOffset+end]

		switch data[offset] {
		case 't', 'f': // true or false
			if bytes.Equal(constant, trueLiteral) || bytes.Equal(constant, falseLiteral) {
				dataType = Boolean
			} else {
				return nil, Unknown, offset, UnknownValueTypeError
			}
		case 'u', 'n': // undefined or null
			if bytes.Equal(constant, nullLiteral) {
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
	if !getfix && dataType == String {
		value = value[1 : len(value)-1]
	}

	if dataType == Null {
		value = []byte{}
	}

	return value, dataType, endOffset, nil
}

// ArrayEach is used when iterating arrays, accepts a callback function with the same return arguments as `Get`.
func ArrayEach(data []byte, cb func(value []byte, dataType ValueType, offset int, err error), keys ...string) (offset int, err error) {
	return arrayEach(false, data, cb, keys...)
}

func arrayEach(getfix bool, data []byte, cb func(value []byte, dataType ValueType, offset int, err error), keys ...string) (offset int, err error) {
	if len(data) == 0 {
		return -1, MalformedObjectError
	}

	offset = 1

	if len(keys) > 0 {
		if offset = searchKeys(data, keys...); offset == -1 {
			return offset, KeyPathNotFoundError
		}

		// Go to closest value
		nO := nextToken(data[offset:])
		if nO == -1 {
			return offset, MalformedJsonError
		}

		offset += nO

		if data[offset] != '[' {
			return offset, MalformedArrayError
		}

		offset++
	}

	for true {
		v, t, o, e := get(getfix, data[offset:])

		if o == 0 {
			break
		}

		if e != nil {
			return offset, e
		}

		if t != NotExist {
			cb(v, t, offset+o-len(v), e)
		}

		if e != nil {
			break
		}

		offset += o

		skipToToken := nextToken(data[offset:])
		if skipToToken == -1 {
			return offset, MalformedArrayError
		}
		offset += skipToToken

		if data[offset] == ']' {
			break
		}

		if data[offset] != ',' {
			return offset, MalformedArrayError
		}

		offset++
	}

	return offset, nil
}

// ObjectEach iterates over the key-value pairs of a JSON object, invoking a given callback for each such entry
func ObjectEach(data []byte, callback func(key []byte, value []byte, dataType ValueType, offset int) error, keys ...string) (err error) {
	return objectEach(false, data, callback, keys...)
}

func objectEach(getfix bool, data []byte, callback func(key []byte, value []byte, dataType ValueType, offset int) error, keys ...string) (err error) {
	var stackbuf [unescapeStackBufSize]byte // stack-allocated array for allocation-free unescaping of small strings
	offset := 0

	// Descend to the desired key, if requested
	if len(keys) > 0 {
		if off := searchKeys(data, keys...); off == -1 {
			return KeyPathNotFoundError
		} else {
			offset = off
		}
	}

	// Validate and skip past opening brace
	if off := nextToken(data[offset:]); off == -1 {
		return MalformedObjectError
	} else if offset += off; data[offset] != '{' {
		return MalformedObjectError
	} else {
		offset++
	}

	// Skip to the first token inside the object, or stop if we find the ending brace
	if off := nextToken(data[offset:]); off == -1 {
		return MalformedJsonError
	} else if offset += off; data[offset] == '}' {
		return nil
	}

	// Loop pre-condition: data[offset] points to what should be either the next entry's key, or the closing brace (if it's anything else, the JSON is malformed)
	for offset < len(data) {
		// Step 1: find the next key
		var key []byte

		// Check what the the next token is: start of string, end of object, or something else (error)
		switch data[offset] {
		case '"':
			offset++ // accept as string and skip opening quote
		case '}':
			return nil // we found the end of the object; stop and return success
		default:
			return MalformedObjectError
		}

		// Find the end of the key string
		var keyEscaped bool
		if off, esc := stringEnd(data[offset:]); off == -1 {
			return MalformedJsonError
		} else {
			key, keyEscaped = data[offset:offset+off-1], esc
			offset += off
		}

		// Unescape the string if needed
		if keyEscaped {
			if keyUnescaped, err := Unescape(key, stackbuf[:]); err != nil {
				return MalformedStringEscapeError
			} else {
				key = keyUnescaped
			}
		}

		// Step 2: skip the colon
		if off := nextToken(data[offset:]); off == -1 {
			return MalformedJsonError
		} else if offset += off; data[offset] != ':' {
			return MalformedJsonError
		} else {
			offset++
		}

		// Step 3: find the associated value, then invoke the callback
		if value, valueType, off, err := get(getfix, data[offset:]); err != nil {
			return err
		} else if err := callback(key, value, valueType, offset+off); err != nil { // Invoke the callback here!
			return err
		} else {
			offset += off
		}

		// Step 4: skip over the next comma to the following token, or stop if we hit the ending brace
		if off := nextToken(data[offset:]); off == -1 {
			return MalformedArrayError
		} else {
			offset += off
			switch data[offset] {
			case '}':
				return nil // Stop if we hit the close brace
			case ',':
				offset++ // Ignore the comma
			default:
				return MalformedObjectError
			}
		}

		// Skip to the next token after the comma
		if off := nextToken(data[offset:]); off == -1 {
			return MalformedArrayError
		} else {
			offset += off
		}
	}

	return MalformedObjectError // we shouldn't get here; it's expected that we will return via finding the ending brace
}

// GetUnsafeString returns the value retrieved by `Get`, use creates string without memory allocation by mapping string to slice memory. It does not handle escape symbols.
func GetUnsafeString(data []byte, keys ...string) (val string, err error) {
	v, _, _, e := get(false, data, keys...)

	if e != nil {
		return "", e
	}

	return bytesToString(&v), nil
}

// GetString returns the value retrieved by `Get`, cast to a string if possible, trying to properly handle escape and utf8 symbols
// If key data type do not match, it will return an error.
func GetString(data []byte, keys ...string) (val string, err error) {
	v, t, _, e := get(true, data, keys...)

	if e != nil {
		return "", e
	}

	if t != String {
		return "", fmt.Errorf("Value is not a number: %s", string(v))
	}

	//strip quotes
	if len(v) > 0 && v[0] == '"' {
		v = v[1 : len(v)-1]
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
	v, t, _, e := get(true, data, keys...)

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
	v, t, _, e := get(true, data, keys...)

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
	v, t, _, e := get(true, data, keys...)

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
	//strip quotes
	if len(b) > 0 && b[0] == '"' {
		b = b[1 : len(b)-1]
	}

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

type JsonValue struct {
	data []byte
	Type ValueType
	err  error
}

func (jv *JsonValue) Err() error {
	return jv.err
}

func (jv *JsonValue) Error() string {
	if jv.err != nil {
		return jv.err.Error()
	} else {
		return ""
	}
}

func ParseJson(data []byte, keys ...string) *JsonValue {
	v, t, _, e := get(true, data, keys...)
	return &JsonValue{data: v, Type: t, err: e}
}

func (jv *JsonValue) Get(keys ...string) *JsonValue {
	if jv.Err() != nil {
		return jv
	}
	v, t, _, e := get(true, jv.data, keys...)
	return &JsonValue{data: v, Type: t, err: e}
}

func (jv *JsonValue) IsObject() bool {
	return jv.Type == Object
}

func (jv *JsonValue) IsArray() bool {
	return jv.Type == Array
}

func (jv *JsonValue) Index(indices ...int) *JsonValue {
	if jv.Err() != nil {
		return jv
	}

	if jv.Type != Array {
		jv.err = fmt.Errorf("Index only supported for Array not %v", jv.Type.String())
		return jv
	}

	var keys []string
	for _, index := range indices {
		keys = append(keys, fmt.Sprintf("[%v]", index))
	}

	v, t, _, e := get(true, jv.data, keys...)
	return &JsonValue{data: v, Type: t, err: e}
}

func (jv *JsonValue) ArrayEach(cb func(value *JsonValue)) error {
	if jv.Err() != nil {
		return jv
	}

	_, err := ArrayEach(jv.data, func(value []byte, dataType ValueType, offset int, err error) {
		cb(&JsonValue{data: value, Type: dataType})
	})

	return err
}

func (jv *JsonValue) ArrayEachWithIndex(cb func(idx int, value *JsonValue)) error {
	if jv.Err() != nil {
		return jv
	}

	idx := 0
	_, err := ArrayEach(jv.data, func(value []byte, dataType ValueType, offset int, err error) {
		cb(idx, &JsonValue{data: value, Type: dataType})
		idx++
	})

	return err
}

func (jv *JsonValue) ArrayEachWithError(cb func(value *JsonValue) error) error {
	if jv.Err() != nil {
		return jv
	}

	var cbErr error
	_, err := ArrayEach(jv.data, func(value []byte, dataType ValueType, offset int, err error) {
		if cbErr != nil {
			return //TODO: rewrite this method so it does not use arrayeach, so we can escape out of this mess ...
		}

		if err != nil {
			cbErr = err
			return
		}

		cbErr = cb(&JsonValue{data: value, Type: dataType})
	})

	if cbErr != nil {
		return cbErr
	}

	return err
}

func (jv *JsonValue) ToArray() ([]*JsonValue, error) {
	if jv.Err() != nil {
		return nil, jv
	}
	var res []*JsonValue
	_, err := ArrayEach(jv.data, func(value []byte, dataType ValueType, offset int, err error) {
		res = append(res, &JsonValue{data: value, Type: dataType})
	})

	return res, err
}

func (jv *JsonValue) String() string {
	return string(jv.data)
}

func (jv *JsonValue) RawBytes() []byte {
	return jv.data
}

func isFloat(b []byte) bool {
	//TODO: faster/better implementation
	return bytes.IndexByte(b, '.') != -1 || bytes.IndexByte(b, 'e') != -1
}

func (jv *JsonValue) IsInt() bool {
	return jv.Type == Number && !isFloat(jv.data)
}

func (jv *JsonValue) IsFloat() bool {
	return jv.Type == Number && isFloat(jv.data)
}

func (jv *JsonValue) IsNumber() bool {
	return jv.Type == Number
}

func (jv *JsonValue) GetInt(keys ...string) (int64, error) {
	if len(keys) == 0 {
		return jv.parseInt()
	} else {
		return jv.Get(keys...).parseInt()
	}
}

func (jv *JsonValue) parseInt() (int64, error) {
	return ParseInt(jv.data)
}

func (jv *JsonValue) GetFloat(keys ...string) (float64, error) {
	if len(keys) == 0 {
		return jv.parseFloat()
	} else {
		return jv.Get(keys...).parseFloat()
	}
}

func (jv *JsonValue) parseFloat() (float64, error) {
	return ParseFloat(jv.data)
}

func (jv *JsonValue) IsBoolean() bool {
	return jv.Type == Boolean
}

func (jv *JsonValue) GetBool(keys ...string) (bool, error) {
	if len(keys) == 0 {
		return jv.parseBool()
	} else {
		return jv.Get(keys...).parseBool()
	}
}

func (jv *JsonValue) parseBool() (bool, error) {
	return ParseBoolean(jv.data)
}

func (jv *JsonValue) IsString() bool {
	return jv.Type == String
}

func (jv *JsonValue) GetString(keys ...string) (string, error) {
	if len(keys) == 0 {
		return jv.parseString()
	} else {
		return jv.Get(keys...).parseString()
	}
}

func (jv *JsonValue) parseString() (string, error) {
	return ParseString(jv.data)
}

func (jv *JsonValue) GetStringArray(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return jv.parseStringArray()
	} else {
		return jv.Get(keys...).parseStringArray()
	}
}

func (jv *JsonValue) parseStringArray() (res []string, err error) {
	if jv.Type != Array {
		return nil, fmt.Errorf("parseStringArray can only be executed on an Array not on a '%v'", jv.Type.String())
	}
	err = jv.ArrayEachWithError(func(value *JsonValue) error {
		if val, err := value.GetString(); err != nil {
			return err
		} else {
			res = append(res, val)
		}
		return nil
	})
	return
}

func (jv *JsonValue) GetIntArray(keys ...string) ([]int64, error) {
	if len(keys) == 0 {
		return jv.parseIntArray()
	} else {
		return jv.Get(keys...).parseIntArray()
	}
}

func (jv *JsonValue) parseIntArray() (res []int64, err error) {
	if jv.Type != Array {
		return nil, fmt.Errorf("parseIntArray can only be executed on an Array not on a '%v'", jv.Type.String())
	}
	err = jv.ArrayEachWithError(func(value *JsonValue) error {
		if val, err := value.GetInt(); err != nil {
			return err
		} else {
			res = append(res, val)
		}
		return nil
	})
	return
}

func (jv *JsonValue) GetFloatArray(keys ...string) ([]float64, error) {
	if len(keys) == 0 {
		return jv.parseFloatArray()
	} else {
		return jv.Get(keys...).parseFloatArray()
	}
}

func (jv *JsonValue) parseFloatArray() (res []float64, err error) {
	if jv.Type != Array {
		return nil, fmt.Errorf("parseFloatArray can only be executed on an Array not on a '%v'", jv.Type.String())
	}
	err = jv.ArrayEachWithError(func(value *JsonValue) error {
		if val, err := value.GetFloat(); err != nil {
			return err
		} else {
			res = append(res, val)
		}
		return nil
	})
	return
}

func (jv *JsonValue) GetBoolArray(keys ...string) ([]bool, error) {
	if len(keys) == 0 {
		return jv.parseBoolArray()
	} else {
		return jv.Get(keys...).parseBoolArray()
	}
}

func (jv *JsonValue) parseBoolArray() (res []bool, err error) {
	if jv.Type != Array {
		return nil, fmt.Errorf("parseBoolArray can only be executed on an Array not on a '%v'", jv.Type.String())
	}
	err = jv.ArrayEachWithError(func(value *JsonValue) error {
		if val, err := value.GetBool(); err != nil {
			return err
		} else {
			res = append(res, val)
		}
		return nil
	})
	return
}
