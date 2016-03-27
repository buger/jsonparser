// +build appengine appenginevm

package strbytes

import (
	"strconv"
)

func Equal(astr string, bbytes []byte) bool {
	return astr == string(bbytes)
}

func ParseFloat(bytes []byte, prec int) (float64, error) {
	return strconv.ParseFloat(string(bytes), prec)
}

func ParseInt(bytes []byte, base int, bitSize int) (int64, error) {
	return strconv.ParseInt(string(bytes), base, bitSize)
}
