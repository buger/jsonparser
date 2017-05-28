package jsonparser

import (
	"bytes"
	"testing"
)

// TestGetLineCol turns an offset into a line/column position.
func TestGetLineCol(t *testing.T) {
	runLineColTest(t, []byte("abc"), []int{})
	runLineColTest(t, []byte("\n"), []int{0})
	runLineColTest(t, []byte("\na\nb\n"), []int{0, 2, 4})
}

func runLineColTest(t *testing.T, input []byte, expected []int) {
	li := NewLineIndex(input)
	obs := li.NewlinePos
	if len(expected) != len(obs) {
		t.Errorf("runLineColTest failed at pos len(observed)==%v, "+
			"len(expected)=%v; obs='%#v'; expected='%#v'",
			len(obs), len(expected), obs, expected)
	} else {
		for i := range expected {
			if obs[i] != expected[i] {
				t.Errorf("runLineColTest failed at pos %v, observed='%#v', expected='%#v'",
					i, obs, expected)
			}
		}
	}
}

// TestOffsetToLineCol turns an offset into a line/column position.
func TestOffsetToLineCol(t *testing.T) {

	runOffsetToLineColTest(t, []byte(`{"a":"b"}`), []string{`a`}, []byte(`b`), 0, 5, 5, String)
	runOffsetToLineColTest(t, []byte("\n"+`{"a":"b"}`), []string{`a`}, []byte(`b`), 1, 5, 5, String)
	runOffsetToLineColTest(t, []byte("\n"+`{"a":"b"}`+"\n"), []string{`a`}, []byte(`b`), 1, 5, 5, String)
	runOffsetToLineColTest(t, []byte("\n\n"+`{"a":"b"}`+"\n"), []string{`a`}, []byte(`b`), 2, 5, 5, String)
	runOffsetToLineColTest(t, []byte("\n\n"+`{"a":"b"}`+"\n\n"), []string{`a`}, []byte(`b`), 2, 5, 5, String)
	runOffsetToLineColTest(t, []byte("\n\n"+`{"a":`+"\n"+`"b"}`+"\n\n"), []string{`a`}, []byte(`b`), 3, 0, 0, String)
	runOffsetToLineColTest(t, []byte("\n\n"+`{`+"\n"+`"a":`+"\n"+`"b"}`+"\n\n"), []string{`a`}, []byte(`b`), 4, 0, 0, String)
	runOffsetToLineColTest(t, []byte(`{`+"\n"+`"a":`+"\n"+`"b"}`), []string{`a`}, []byte(`b`), 2, 0, 0, String)
	runOffsetToLineColTest(t, []byte(`{`+"\n"+`"a":`+`"b"}`), []string{`a`}, []byte(`b`), 1, 4, 4, String)

	// multiline value
	runOffsetToLineColTest(t, []byte(`{`+"\n"+`"a":"b`+"\n"+`ye"}`), []string{`a`}, []byte(`b`+"\n"+`ye`), 1, 4, 4, String)

	// multi-byte characters
	runOffsetToLineColTest(t, []byte(`{"世界":"世界"}`), []string{`世界`}, []byte(`世界`), 0, 10, 6, String)
	runOffsetToLineColTest(t, []byte(`{"世界":`+"\n"+`"世界"}`), []string{`世界`}, []byte(`世界`), 1, 0, 0, String)

}

func runOffsetToLineColTest(t *testing.T, input []byte, searchPath []string,
	expectedValue []byte,
	expectedLine, expectedByteCol, expectedRuneCol int, expectedDataType ValueType) {

	li := NewLineIndex(input)
	obs, obsDataType, offs, err := Get(input, searchPath...)

	//fmt.Printf("\n Get(input='%s', searchPath='%#v') returned obs='%#v', obsDataType='%s', offs=%v, err=%v. len(obs)=%v\n", string(input), searchPath, string(obs), obsDataType, offs, err, len(obs))

	// account for the double quotes around strings in their position
	lenObs := len(obs)
	if obsDataType == String {
		lenObs += 2
	}

	if err != nil {
		panic(err)
	}
	if bytes.Compare(obs, expectedValue) != 0 {
		t.Errorf("runOffsetToLineColTest failed, obs != expectedValue, observed='%#v', expected='%#v'",
			obs, expectedValue)
	}
	if obsDataType != expectedDataType {
		t.Errorf("runOffsetToLineColTest failed, obsDataType != expectedDataType, observed='%#v', expected='%#v'",
			obsDataType, expectedDataType)
	}

	// the main event: the call to li.OffsetToLineCol()
	//
	// Note offs is where the key value *ends*, per the jsonparser.Get() docs.
	// Hence we subtract the len(obs) to get the byte offset of the
	// beginning of the value.
	//
	obsLine, obsByteCol, obsRuneCol := li.OffsetToLineCol(offs - lenObs)

	//fmt.Printf("li.OffsetToLineCol(offs=%#v) returned obsLine=%v, obsByteCol=%v, obsRuneCol=%v. len(obs)=%v\n", offs, obsLine, obsByteCol, obsRuneCol, len(obs))

	if obsLine != expectedLine {
		t.Errorf("runOffsetToLineColTest failed, obsLine != expectedLine, observed='%#v', expected='%#v'",
			obsLine, expectedLine)
	}
	if obsByteCol != expectedByteCol {
		t.Errorf("runOffsetToLineColTest failed, obsByteCol != expectedByteCol, observed='%#v', expected='%#v'",
			obsByteCol, expectedByteCol)
	}

	if obsRuneCol != expectedRuneCol {
		t.Errorf("runOffsetToLineColTest failed, obsRuneCol != expectedRuneCol, observed='%#v', expected='%#v'",
			obsRuneCol, expectedRuneCol)
	}

}
