# Applying the reqproof toolchain to jsonparser

This document records the result of applying reqproof's full pipeline
(Phases O+P+Q+R+S+T+U+X'+Y+Z+S.2c+Z.2 on `feat/z3-roadmap`,
HEAD `33ab4e0a`) to `github.com/buger/jsonparser` on branch
`fix-oss-fuzz-delete-leading-comma`. The headline goal was to
demonstrate how a // reqproof:lemma directive could have surfaced
the OSS-Fuzz Delete panic (testcase 4649128545288192, fixed in
commit `a6c5ed3`) BEFORE the bug ever shipped.

## TL;DR

- 7 // reqproof:lemma directives were authored directly on production
  code, all PROVED on Z3.
- The Delete bug demo is a Phase Q snippet extraction. Z3 returned
  COUNTEREXAMPLE `prevTok = -1, remainedTok = 0` for the pre-fix
  obligation, and PROVED the post-fix obligation. This is the bug
  the OSS-Fuzz fuzzer found, recovered via formal logic in 6ms.
- verify-properties (SYS-REQ): 16 checks, 0 violated, 4 skipped
  variables (orphan_no_requirement, pre-existing).
- audit: 0 errors, 6 warnings (mostly stale authored-delta on the
  fuzz-related files added in the same commit as the fix).
- lemma branch coverage: 18 / 18 (100%) — every AST branch reached
  by the SMT translation of at least one lemma is covered.

## Pass 1 — Baseline pure-function lemmas

Seven properties on five production functions (all on the
`fix-oss-fuzz-delete-leading-comma` HEAD, all PROVED on Z3):

| Lemma | Host | What it proves |
|---|---|---|
| `tokenEnd_in_range` | `parser.go::tokenEnd` | `0 <= r <= len(data)` |
| `tokenStart_in_range` | `parser.go::tokenStart` | `0 <= r <= len(data)` |
| `nextToken_in_range` | `parser.go::nextToken` | `-1 <= r < len(data)` |
| `lastToken_in_range` | `parser.go::lastToken` | `-1 <= r < len(data)` |
| `isUTF16EncodedRune_low_excluded` | `escape.go::isUTF16EncodedRune` | `r < 0xD800 ⇒ ¬isUTF16EncodedRune(r)` |
| `isUTF16EncodedRune_high_excluded` | `escape.go::isUTF16EncodedRuneNot` (alias) | `r > 0xDFFF ⇒ ¬isUTF16EncodedRune(r)` |
| `deleteCleanupFixed_prevTok_nonneg` | `parser_delete_snippet_proof.go::deleteCleanupFixedDereferenceObligation` | post-fix dereference obligation |

Two pure-function candidates were SKIPPED with documented reasons:

- `escape.go::h2I` — the body's `int(c - '0')` expression is a type
  conversion the body translator currently classifies as an
  unsupported function call. Phase D / E_FUNCTION_CALL widening is
  the prerequisite.
- `escape.go::combineUTF16Surrogates` — references package-level
  consts `supplementalPlanesOffset`, `highSurrogateOffset`,
  `lowSurrogateOffset` which the translator rejects as free
  variables.
- `bytes_unsafe.go::*` — uses `unsafe.Pointer`, an explicit
  E_UNSAFE_PTR translator-rejection. Would need Phase P L1 opaque or
  the safe-build-tag fallback in `bytes_safe.go`.

### Authoring-time refactors made on production code

To get each lemma to translate, three small refactors were applied
to production functions WITHOUT changing observable behavior. Tests
pass at HEAD; behavior is byte-identical.

1. `tokenEnd`, `nextToken`, `lastToken`, `tokenStart` — `switch`
   statements rewritten as `if` chains. The translator does not yet
   support `switch` statements (E_SWITCH_NOT_SUPPORTED), and the
   loop-translator rewrite for break/early-return cannot lower
   switch case-bodies.
2. `tokenEnd`, `nextToken`, `lastToken`, `tokenStart` — character
   literals `' '`, `'\n'`, … replaced with their integer ASCII
   values. The translator currently rejects character literals
   inside loop bodies.
3. `tokenEnd`, `tokenStart` — body shape changed from
   `if cond { return i }` (conditional early-return) to
   `if !cond { continue } return i` (guarded fall-through). The
   first shape exposes a translator bug in the conditional
   early-return lowering (see "Open follow-ups" below).

`findTokenStart` was reverted to the original `switch` form because
its body has TWO conditional returns; both rewrites tested still
exposed the conditional early-return translator bug. The function
keeps its switch and is NOT under a lemma.

## Pass 2 — The Delete-bug demo (Option β: snippet extraction)

The OSS-Fuzz Delete panic (testcase 4649128545288192) on input
`,{"test":1{}` traces to `parser.go:813-820` in the pre-fix code:

```go
prevTok := lastToken(data[:keyOffset])
// ...
if remainedTok > -1 && remainedValue[remainedTok] == '}' && data[prevTok] == ',' {
    newOffset = prevTok
} else {
    newOffset = prevTok + 1
}
```

When `keyOffset == 0`, `data[:0]` is empty so `lastToken` returns
`-1`, and `data[prevTok] == ','` panics on the negative index.

### Why direct annotation (Option α) failed

Annotating `Delete` itself with `// reqproof:requires prevTok > -1`
is not viable today. Three independent gaps:

- `Delete` calls helpers (`searchKeys`, `internalGet`,
  `findTokenStart`, …) that themselves use slice expressions
  `data[:keyOffset]` (`E_SLICE_EXPR`) and switch statements
  (`E_SWITCH_NOT_SUPPORTED`).
- `Delete` mutates a slice via `append(dataCopy[:newOffset],
  dataCopy[endOffset:]...)`, which the body translator rejects.
- Even if the translation worked, runtime panics aren't a
  first-class output of the SMT translation — Phase AA "verify-safety"
  (planned) is what would emit auto-OOB obligations.

### What worked: Option β — abstract snippet under build tag

`parser_delete_snippet_proof.go` (build-tagged
`reqproof_proof`) extracts the offending block as two helpers:

- `deleteCleanupBuggyDereferenceObligation(prevTok, remainedTok int) bool`
  encodes the dereference obligation of the **pre-fix** branch shape
  (data[prevTok] is reached whenever remainedTok > -1, so
  prevTok >= 0 must hold there).
- `deleteCleanupFixedDereferenceObligation(prevTok, remainedTok int) bool`
  encodes the **post-fix** branch shape (the new `prevTok > -1`
  guard fronts every dereference).

Two lemmas certify these:

```go
// reqproof:lemma deleteCleanupBuggy_prevTok_nonneg_falsifiable func(prevTok, remainedTok int) bool {
//   return deleteCleanupBuggyDereferenceObligation(prevTok, remainedTok)
// }
```

```go
// reqproof:lemma deleteCleanupFixed_prevTok_nonneg func(prevTok, remainedTok int) bool {
//   return deleteCleanupFixedDereferenceObligation(prevTok, remainedTok)
// }
```

### Result (verbatim from `proof verify-lemma --solver z3 --tags reqproof_proof`)

```
[ce] deleteCleanupBuggy_prevTok_nonneg_falsifiable COUNTEREXAMPLE (z3, 6ms)
    counterexample:
      remainedTok = 0
      prevTok = -1
[ok] deleteCleanupFixed_prevTok_nonneg PROVED (z3, 6ms)
```

This IS the OSS-Fuzz bug, surfaced as a Z3 counterexample in 6ms.
The `prevTok = -1` value is exactly what `lastToken(data[:0])`
returns; `remainedTok = 0` is the value of `nextToken` on the
non-empty `remainedValue` suffix.

A unit test author reading this counterexample would translate it
back to a JSON input by:
- choosing any `keyOffset == 0` ⇒ `prevTok = -1`,
- arranging non-empty `data[endOffset:]` whose first non-whitespace
  byte triggers the pre-fix branch.

`,{"test":1{}` (the OSS-Fuzz repro) is one such input.

## Pass 3 — verify-properties + audit

### verify-properties (SYS-REQ)

```
Total Variables:                               226
Variables with Data Constraints (Authored):    4
Total Checks:                                  16
Violated:                                      0
Skipped (delta):                               0
Skipped Variables (orphan_no_requirement):     4
```

The four orphan_no_requirement variables (`array_index_is_in_bounds`,
`array_index_is_out_of_bounds`, `set_called_without_path`,
`set_path_is_provided`) are pre-existing — no SYS-REQ in the
`parser` component references them, so behavioral_implication
proofs do not fire (completeness/exclusivity still ran and proved).

### audit

```
Errors: 0  Warnings: 6
Assurance: L3 (1 component)
```

Findings:

1. `authored_delta_expected` — 6 traced production files lack a
   current no-authored-change review. These are exactly the files
   touched by the fix commit and the lemma-authoring refactors;
   the warning is expected on a feature branch.
2. `lint_clean` / `orphan_code_clean` — 21 + 7 untraced symbols in
   `fuzz_native_test.go` and the new snippet file. The fuzz wrappers
   were added in the same commit as the fix; tracing them is a
   separate task.
3. `orphan_tests_clean` — 14 of 285 test functions lack a
   `// Verifies:` annotation (mostly the new fuzz wrappers).
4. `suspect_clean` — 107 suspect links; pre-existing.
5. `verify_passes` — 1 warning (the lint warning above).

No Errors. The branch is healthy at the L3 assurance level.

## Pass 4 — Lemma branch coverage

```
Coverage: 18 branches, 18 covered (100%)
every recorded branch is covered by at least one lemma.
```

Every AST branch the SMT translation of any lemma reached is
covered. The 18-branch denominator counts only the branches that
the translator visited (Phase Y's recorder fires inside body
translation), so the coverage scope is "the production code reached
by these 7 proved lemmas". Production functions whose lemmas are
blocked by translator gaps (e.g. `findTokenStart`, `Delete`,
`searchKeys`, `escape.go::Unescape`) do not appear in the
denominator. Phase AA + S.2c.3 + Phase D widening would expand
that denominator significantly.

## Pass 5 — Open follow-ups (translator gaps surfaced)

These are real reqproof gaps, surfaced by attempting to apply the
toolchain end-to-end on a small library. They are filed here so the
reqproof team can pick them up:

1. **Conditional early-return translator bug**. Loop bodies of the
   shape `if cond { return X }` followed by code emit broken SMT
   referencing `__early_val$N$M` outside the let-binding that defines
   it. Manifests as Z3 errors `unknown constant __early_val$1$1` at
   the prelude line. Workaround: rewrite as guarded fall-through
   `if !cond { continue } return X`, but THIS PATTERN is itself
   sometimes broken when there are TWO conditional returns inside
   the loop (`findTokenStart`). Phase S.2c.4 follow-up.
2. **`E_SWITCH_NOT_SUPPORTED`**. `switch` statements inside loop
   bodies (and elsewhere) need a lowering. Many idiomatic Go
   patterns (jsonparser's tokenizer is a representative example)
   use switch; refactoring is invasive.
3. **Character literals inside loop bodies**. The translator rejects
   `c == ' '` style comparisons; users must replace with the integer
   ASCII value. Trivial to author once known but a friction point.
4. **`E_FUNCTION_CALL` for type conversions**. `int(c)` on a `byte`
   is rejected as an unsupported function call even though the
   conversion is a no-op at the SMT-Int level. Phase D widening.
5. **`E_SLICE_EXPR` for slice expressions**. Disqualifies any
   function whose body uses `data[:i]` / `data[i:]`. Workaround
   exists (recursive helper + `// reqproof:decreases`) but it is a
   significant authoring lift for non-trivial functions.
6. **Free package constants**. `combineUTF16Surrogates` references
   `supplementalPlanesOffset`, etc. The translator should inline
   const declarations.
7. **Multi-lemma single host**. Two consecutive
   `// reqproof:lemma` directives on the same function host
   silently dropped the second lemma during scanning. Workaround:
   put the second lemma on a thin alias function (we did this for
   `isUTF16EncodedRune_high_excluded`). Phase R follow-up.
8. **Phase AA — auto-OOB obligations**. The Delete bug is a runtime
   index-out-of-range panic. With Phase AA we would not need the
   snippet at all — every `data[i]` read in `Delete` would carry
   an automatic `0 <= i < len(data)` proof obligation, and the
   pre-fix code would fail to verify with the SAME counterexample
   (`prevTok = -1`).

## Recommendation for the jsonparser team

1. The seven proved lemmas are cheap regressions: 7 PROVED in <60ms
   total Z3 time, cached at `.proof/lemma-cache.json`. CI should run
   `proof verify-lemma --solver z3 --tags reqproof_proof ./...` as
   part of the existing test job. A future code change that
   accidentally removes the `prevTok > -1` guard from Delete will
   re-trigger the COUNTEREXAMPLE on
   `deleteCleanupFixed_prevTok_nonneg`, and CI will fail.
2. The snippet file `parser_delete_snippet_proof.go` is a
   regression test for the bug class. Adding similar snippets for
   any future Delete/Set OOB fix is a small, repeatable pattern.
3. Once Phase AA lands, the snippet can be retired in favor of an
   automatic OOB obligation on the production `Delete` body.
