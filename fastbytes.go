package jsonparser

func BytesParseInt(bytes []byte) (v int64, ok bool) {
	if len(bytes) == 0 {
		return 0, false
	}

	var neg bool = false
	if bytes[0] == '-' {
		neg = true
		bytes = bytes[1:]
	}

	for _, c := range bytes {
		if c >= '0' && c <= '9' {
			v = (10 * v) + int64(c-'0')
		} else {
			return 0, false
		}
	}

	if neg {
		return -v, true
	} else {
		return v, true
	}
}
