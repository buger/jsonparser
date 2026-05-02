// +build reqproof_proof

// Item #3 dogfood lemmas — exercise translator features landed in items
// #1 (path-conditions) and #2 (variadic + multi-return + multi-target
// assigns at the producer side).
//
// These lemmas live in a build-tag-guarded file so the production
// jsonparser binary is unaffected.
//
// Each lemma is attached to a tiny anchor predicate (the parser
// requires reqproof:lemma directives to be attached to a declaration
// with a return value).

package jsonparser

// --- Item #1 (path-conditions) lemmas ---------------------------------
//
// The lemmas below state guarded indexing-safety obligations on the
// existing token-locator helpers. Item #1 propagates the if-guard
// premise into the SMT goal so the body becomes an implication
// rather than a raw conjunction.

// reqproof:lemma tokenEnd_path_indexable_implies_nonneg func(data []byte) bool {
//   r := tokenEnd(data)
//   if r < len(data) {
//     return r >= 0
//   }
//   return true
// }
func anchor_tokenEnd_path() bool { return true }

// reqproof:lemma nextToken_path_indexable_implies_lt_len func(data []byte) bool {
//   r := nextToken(data)
//   if r >= 0 {
//     return r < len(data)
//   }
//   return true
// }
func anchor_nextToken_path() bool { return true }

// reqproof:lemma lastToken_path_indexable_implies_lt_len func(data []byte) bool {
//   r := lastToken(data)
//   if r >= 0 {
//     return r < len(data)
//   }
//   return true
// }
func anchor_lastToken_path() bool { return true }

// reqproof:lemma tokenStart_path_indexable_when_nonempty func(data []byte) bool {
//   r := tokenStart(data)
//   if len(data) > 0 {
//     return r >= 0 && r < len(data)
//   }
//   return r == 0
// }
func anchor_tokenStart_path() bool { return true }

// --- Item #1 path-conditions over the arithmetic helper h2I ---------
//
// h2I returns -1 (badHex) for non-hex bytes and 0..15 otherwise. These
// guarded lemmas exercise path-conditions where the antecedent is on
// the *output*, not the input.

// reqproof:lemma h2I_nonneg_implies_le_15 func(c byte) bool {
//   r := h2I(c)
//   if r >= 0 {
//     return r <= 15
//   }
//   return true
// }
func anchor_h2I_nonneg() bool { return true }

// --- Item #2 (variadic) lemma -----------------------------------------
//
// `keysCount` exercises a callee-side variadic ...string parameter
// with a *single* int return (avoiding the tuple-sort issue we hit
// when we tried (int, bool) — that's a real translator follow-up,
// see docs/reqproof-item3-application.md).
//
// Item #2 is what allows this signature to translate at all.

func keysCount(keys ...string) int {
	return len(keys)
}

// reqproof:lemma keysCount_matches_len func(keys []string) bool {
//   return keysCount(keys...) == len(keys)
// }
func anchor_keysCount_len() bool { return true }

// reqproof:lemma keysCount_nonneg func(keys []string) bool {
//   return keysCount(keys...) >= 0
// }
func anchor_keysCount_nonneg() bool { return true }
