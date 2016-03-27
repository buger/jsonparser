// +build !appengine,!appenginevm

package jsonparser

import (
	"strconv"
	"unsafe"
)

//
// The reason for using *[]byte rather than []byte in parameters is an optimization. As of Go 1.6,
// the compiler cannot perfectly inline the function when using a non-pointer slice. That is,
// the non-pointer []byte parameter version is slower than if its function body is manually
// inlined, whereas the pointer []byte version is equally fast to the manually inlined
// version. Instruction count in assembly taken from "go tool compile" confirms this difference.
//

func BytesEqualStr(abytesptr *[]byte, bstr string) bool {
	return *(*string)(unsafe.Pointer(abytesptr)) == bstr
}

func BytesParseFloat(bytesptr *[]byte, bitSize int) (float64, error) {
	return strconv.ParseFloat(*(*string)(unsafe.Pointer(bytesptr)), bitSize)
}
