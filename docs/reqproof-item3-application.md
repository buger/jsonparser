# Reqproof Item #3 — Application to jsonparser

This doc records the dogfooding result of applying reqproof's
`feat/z3-roadmap` items #1 (path-conditions), #2 (variadic +
multi-return), and Phase AA verify-safety to the jsonparser package.

- Reqproof revision: `feat/z3-roadmap` HEAD `d17622a9`.
- jsonparser revision: `fix-oss-fuzz-delete-leading-comma` HEAD `80cf537`.
- Date: 2026-05-01.

## Pass 1 — lemma corpus re-run with the new toolchain

Command:

```sh
go run ./cmd/proof verify-lemma --solver z3 --solver z3,cvc5 \
    --tags reqproof_proof --no-cache /Users/leonidbugaev/go/src/jsonparser/...
```

Result (pre-item-#3 corpus, 23 lemmas):

| verdict        | count |
| -------------- | ----- |
| PROVED         | 22    |
| COUNTEREXAMPLE | 1     |
| TIMEOUT        | 0     |
| UNKNOWN        | 0     |
| TRANSLATION    | 0     |

The single COUNTEREXAMPLE is the deliberately-falsifiable
`deleteCleanupBuggy_prevTok_nonneg_falsifiable` lemma in
`parser_delete_snippet_proof.go` (the OSS-Fuzz Delete pre-fix
witness). All 22 PROVED verdicts preserved verbatim under the
new translator. Baseline retained.

## Pass 2 — new lemmas exercising items #1 and #2

Added in `parser_item3_lemmas_proof.go` (build tag
`reqproof_proof`, anchored to no-op predicates because lemma
directives must attach to a declaration with a return slot):

### Item #1 (path-conditions) lemmas — 5 new, all PROVED

| lemma                                              | shape                                                   | verdict |
| -------------------------------------------------- | ------------------------------------------------------- | ------- |
| `tokenEnd_path_indexable_implies_nonneg`           | `if r < len(data) { r >= 0 }` over `tokenEnd`           | PROVED  |
| `nextToken_path_indexable_implies_lt_len`          | `if r >= 0 { r < len(data) }` over `nextToken`          | PROVED  |
| `lastToken_path_indexable_implies_lt_len`          | `if r >= 0 { r < len(data) }` over `lastToken`          | PROVED  |
| `tokenStart_path_indexable_when_nonempty`          | `if len(data)>0 { 0<=r<len(data) } else r==0`           | PROVED  |
| `h2I_nonneg_implies_le_15`                         | `if r >= 0 { r <= 15 }` over `h2I`                      | PROVED  |

These claims could be re-stated as plain conjunctions before item
#1, but the natural "if-guard then conclusion" shape is what item
#1 is supposed to make first-class — and it does.

### Item #2 (variadic) lemmas — 2 new, all PROVED

A small helper `keysCount(keys ...string) int` was added (not used
by production code, build-tag-gated). Item #2's variadic-callee
support is what allows this signature to translate at all.

| lemma                          | claim                              | verdict |
| ------------------------------ | ---------------------------------- | ------- |
| `keysCount_matches_len`        | `keysCount(keys...) == len(keys)`  | PROVED  |
| `keysCount_nonneg`             | `keysCount(keys...) >= 0`          | PROVED  |

### Lemma corpus growth

23 → 30 lemmas. 22 PROVED → 29 PROVED. The single COUNTEREXAMPLE
is preserved at the same site. Net `+7` new PROVED lemmas, zero
regressions, zero translation errors after working around the
tuple-sort issue described below.

### Translator gap surfaced while authoring

The first attempt at the item-#2 demonstration was a helper with
*two* return slots:

```go
func keysSummary(keys ...string) (int, bool) { ... }
```

With this declaration in the package, every other lemma
(including the 22-lemma baseline) regressed from PROVED to
UNKNOWN with `error "Invalid function definition: unknown sort
'gosmt_tuple_644eb3b2'"`. The synthesized SMT tuple sort for
`(int, bool)` was emitted into the package-wide preamble but
not declared, poisoning every adjacent lemma's solver context.

**Workaround**: collapse the helper to a single-return value.

**Follow-up for reqproof**: tuple sorts produced for
multi-return helpers must be declared (or scoped per-lemma) so
that adding one such helper to a package does not break every
other lemma in that package. Item #2 lands the *callee* side of
multi-return cleanly when the lemma directly invokes the helper,
but a *passive* helper sitting in the same package is enough to
trip the preamble.

### Translator gaps still blocking — bigger candidates

| function    | gap                                                       | items #1/#2 unblock? |
| ----------- | --------------------------------------------------------- | -------------------- |
| `Delete`    | E_EARLY_RETURN_NO_ELSE at line 800 (`if !array { ... }`)  | no (separate gap)    |
| `Get`       | unsupported type `ValueType` (named alias)                | no                   |
| `GetString` | `_, _, := ...` multi-target assign at consumer site       | partial (#2 fixes producer side; consumer side is a distinct E_MULTI_TARGET_ASSIGN) |
| `GetInt`/`GetFloat`/`GetBoolean`/`GetUnsafeString` | same as `GetString` | partial |
| `searchKeys` | E_RECURSION_NO_DECREASES                                 | no                   |
| `stringEnd`, `blockEnd`, `tokenStart` (loops in body), `nextToken` (loops) | E_FOR_LOOP_NO_INVARIANT | no |
| `Set`       | E_TYPE_MISMATCH on Seq Int return                         | no                   |
| `bytes_safe.parseFloat` | E_MULTIPLE_RETURN_VALUES (return single from 2-return) | partial |

The two highest-leverage candidate functions for the next
roadmap item — `Delete` and `GetString` — are *both* still
blocked, but on gaps distinct from #1/#2. Item #2 lands the
producer side of multi-return; the consumer side
(`a, b, c, _ := f(...)`) is a separate E_MULTI_TARGET_ASSIGN.

## Pass 3 — verify-safety

Command:

```sh
go run ./cmd/proof verify-safety /Users/leonidbugaev/go/src/jsonparser/
```

| metric                | value |
| --------------------- | ----- |
| functions scanned     | 11    |
| functions skipped     | 62    |
| findings (E_INDEX_OOB) | 0    |
| findings (E_DIV_BY_ZERO) | 0  |

Zero safety findings. The 11 scannable functions are the small
arithmetic helpers (`h2I`, `isUTF16EncodedRune`,
`isUTF16EncodedRuneNot`, the `anchor_*` no-op predicates, the two
`deleteCleanup*Obligation` helpers, `keysCount`,
`deleteCleanupBuggyFalsifyingWitness`). All are pure arithmetic
or pure boolean and have no slice indexing or division.

### Why the OSS-Fuzz Delete bug does NOT surface here

verify-safety would surface the pre-fix `data[prevTok]` panic
*if `Delete` translated*. It does not — the function is rejected
at translation time with E_EARLY_RETURN_NO_ELSE on the `if
!array { ... }` block. The bug surfaces instead through the
purpose-built `parser_delete_snippet_proof.go` lemma
(`deleteCleanupBuggy_prevTok_nonneg_falsifiable`), which abstracts
the panic site as a parameterized integer obligation. That
lemma's COUNTEREXAMPLE verdict (`prevTok=-1, remainedTok=0`) is
the same machine-checked witness the OSS-Fuzz testcase would
produce, encoded against the pre-fix branch shape.

### Triage

Nothing to triage — zero findings. No new real bugs found.

## Headline

- Pre-existing 22 PROVED + 1 COUNTEREXAMPLE corpus survives
  unchanged under the new translator.
- 7 new PROVED lemmas, 5 of them path-condition-shaped (item #1)
  and 2 of them variadic-callee-shaped (item #2).
- One new translator follow-up identified: tuple-sort preamble
  scoping when a multi-return helper sits passively in the
  package.
- verify-safety continues to be useful only on functions that
  pass the dispatcher; jsonparser's loop-heavy core (Delete,
  Get-family, searchKeys) remains out of reach until
  loop-recursion translation or for-loop invariant inference
  lands. None of the items in this milestone moved that frontier.
- No new real bugs found beyond the already-known OSS-Fuzz Delete
  pre-fix witness, which continues to surface as a
  COUNTEREXAMPLE on its dedicated snippet lemma.
