// +build reqproof_proof

// Phase Q snippet extraction for the OSS-Fuzz Delete panic
// (parser.go pre-fix line 813-820, fixed in commit a6c5ed3).
//
// Slice expressions `data[:keyOffset]` are an E_SLICE_EXPR in the
// translator, so we model the buggy block ABSTRACTLY by taking the
// values lastToken/nextToken would have returned as integer params.
// The lemmas express the dereference-safety obligation directly.
//
// Gated behind `reqproof_proof` build tag.

package jsonparser

// deleteCleanupBuggyDereferenceSafe encodes the implicit obligation
// of the pre-fix block (parser.go pre-a6c5ed3, lines 813-820): the
// data[prevTok] dereference happens whenever remainedTok > -1, so
// safety requires prevTok >= 0 in that case.
//
// Pre-fix branch shape (production code):
//
//   if remainedTok > -1 && remainedValue[remainedTok] == '}' && data[prevTok] == ',' { newOffset = prevTok }
//   else                                                                              { newOffset = prevTok + 1 }
//
// The data[prevTok] read short-circuits, but is reached for every
// remainedTok > -1 case where the previous two operands are true.
// On malformed input where keyOffset == 0, lastToken returns -1 and
// the access panics.
//
// The lemma below claims the obligation always holds; on the buggy
// model it MUST yield a counterexample at (prevTok = -1, remainedTok = 0).
//
// reqproof:lemma deleteCleanupBuggy_prevTok_nonneg_falsifiable func(prevTok, remainedTok int) bool {
//   return deleteCleanupBuggyDereferenceObligation(prevTok, remainedTok)
// }
func deleteCleanupBuggyDereferenceObligation(prevTok, remainedTok int) bool {
	if remainedTok >= 0 {
		return prevTok >= 0
	}
	return true
}

// deleteCleanupFixedDereferenceObligation encodes the post-fix block
// (parser.go HEAD a6c5ed3, lines 815-822). The new prevTok > -1
// guard fronts every dereference, so the obligation holds.
//
// Post-fix branch shape (production code):
//
//   if prevTok > -1 && remainedTok > -1 && remainedValue[remainedTok] == '}' && data[prevTok] == ',' { newOffset = prevTok }
//   else if prevTok > -1                                                                              { newOffset = prevTok + 1 }
//   else                                                                                              { newOffset = 0 }
//
// data[prevTok] is now dereferenced only when prevTok > -1 AND
// remainedTok > -1.
//
// reqproof:lemma deleteCleanupFixed_prevTok_nonneg func(prevTok, remainedTok int) bool {
//   return !(prevTok >= 0 && remainedTok >= 0) || prevTok >= 0
// }
func deleteCleanupFixedDereferenceObligation(prevTok, remainedTok int) bool {
	return !(prevTok >= 0 && remainedTok >= 0) || prevTok >= 0
}

// deleteCleanupBuggyFalsifyingWitness documents the falsifying input
// the COUNTEREXAMPLE verdict surfaces: prevTok = -1, remainedTok = 0
// satisfies the buggy block's branch condition `remainedTok > -1`
// while violating prevTok >= 0. Any input that makes lastToken
// return -1 (i.e. an empty prefix, which happens for the OSS-Fuzz
// testcase `,{"test":1{}` where keyOffset == 0) triggers this.
//
// Plain Go function (no lemma). The machine-checked counterexample
// on deleteCleanupBuggy_prevTok_nonneg_falsifiable already provides
// the proof; this helper exists for documentation only.
func deleteCleanupBuggyFalsifyingWitness() bool {
	return !deleteCleanupBuggyDereferenceObligation(-1, 0)
}
