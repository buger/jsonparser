package jsonparser

func FuzzParseString(data []byte) int {
	r, err := ParseString(data)
	if err != nil || r == nil {
		return 0
	}
	return 1
}
