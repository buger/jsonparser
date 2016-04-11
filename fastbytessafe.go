// +build appengine appenginevm

package jsonparser

import (
	"strconv"
)

// See fastbytes_unsafe.go for explanation on why *[]byte is used (signatures must be consistent with those in that file)

func BytesEqualStr(abytes *[]byte, bstr string) bool {
	return string(*abytes) == bstr
}

func BytesParseFloat(bytes *[]byte, prec int) (float64, error) {
	return strconv.ParseFloat(string(*bytes), prec)
}
