// +build !appengine,!appenginevm

package strbytes

import (
	"reflect"
	"strconv"
	"unsafe"
)

func equalSafe(astr string, bbytes []byte) bool {
	return astr == string(bbytes)
}

func equalUnsafe(astr string, bbytes []byte) bool {
	bslicehdr := (*reflect.SliceHeader)(unsafe.Pointer(&bbytes))
	bstrhdr := reflect.StringHeader{Data: bslicehdr.Data, Len: bslicehdr.Len}
	bstr := *(*string)(unsafe.Pointer(&bstrhdr))
	return astr == bstr
}
func equalMoreUnsafe(astr string, bbytes []byte) bool {
	bstr := *(*string)(unsafe.Pointer(&bbytes))
	return astr == bstr
}

func Equal(astr string, bbytes []byte) bool {
	return equalMoreUnsafe(astr, bbytes)
}

func ParseFloat(bytes []byte, bitSize int) (float64, error) {
	str := *(*string)(unsafe.Pointer(&bytes))
	return strconv.ParseFloat(str, bitSize)
}

func ParseInt(bytes []byte, base int, bitSize int) (int64, error) {
	str := *(*string)(unsafe.Pointer(&bytes))
	return strconv.ParseInt(str, base, bitSize)
}
