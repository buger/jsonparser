package jsonparser

import (
	"bytes"
	"errors"
	"fmt"
	tm "github.com/buger/goterm"
	_ "runtime/debug"
	"strconv"
)

var debug = false

// In case if JSON is having non standard format (like prettifying)
// and value start on new line or after more then 1 space
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

func trailingBracket(data []byte, openSym byte, closeSym byte) int {
	level := 0
	i := 0
	ln := len(data)

	for true {
		if i >= ln {
			if debug {
				fmt.Println("can't find matching bracket", level, tm.Highlight(tm.Highlight(tm.Context(string(data), i, 500), string(openSym), tm.RED), string(closeSym), tm.BLUE))
			}
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
			//fmt.Println("Found string:", level, tm.HighlightRegion(data, sFrom, i, tm.GREEN))
		}

		if c == openSym {
			level += 1

			if debug {
				fmt.Println("Found open sym:", level, tm.HighlightRegion(string(data), i, i+1, tm.RED))
			}
		} else if c == closeSym {
			level -= 1

			if debug {
				fmt.Println("Found close sym:", level, tm.HighlightRegion(string(data), i, i+1, tm.BLUE))
			}
		}

		i++

		if level == 0 {
			break
		}
	}

	if debug {
		fmt.Println("Found matching brackets:", tm.HighlightRegion(string(data), 0, i, tm.YELLOW))
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

func Get(data []byte, keys ...string) (value []byte, dataType int, offset int, err error) {
	// defer func() {
	//     if r := recover(); r != nil {
	//         err = errors.New(fmt.Sprintf("Unhandler JSON parsing error: %v, %s", r, string(d.Stack())))
	//     }
	// }()

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

func GetBoolean(data []byte, keys ...string) (val bool, err error) {
	v, t, _, e := Get(data, keys...)

	if e != nil {
		return false, e
	}

	if t != BOOLEAN {
		return false, errors.New(fmt.Sprintf("Value is not a boolean: %s", string(v)))
	}

	if v['0'] == 't' {
		val = true
	} else {
		val = false
	}

	return
}
