package jsonparser

func FuzzParseString(data []byte) int {
	r, err := ParseString(data)
	if err != nil {
		return 0
	}
	_ = r
	return 1
}
