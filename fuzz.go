package jsonparser

func FuzzParseString(data []byte) int {
	r, err := ParseString(data)
	if err != nil || r == "" {
		return 0
	}
	return 1
}
