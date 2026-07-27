# Proof-Gap Root-Cause Analysis: Two Bugs That Escaped L3 Strict Proof Review

Status: blameless postmortem · Scope: `jsonparser` L3 strict proof posture
Author: proof-gap review · Date: 2026-07-26

This document maps two escaped defects — (1) `Set` array-index-beyond-length data
loss and (2) the 8th empty-key-component panic site — to the proof layer that
should have caught each, identifies the specific blind spot, and records the
concrete remediation in flight. It is written to close gaps in the proof
toolchain, not to assign fault.

---

## 1. Executive summary

Two bugs escaped the L3 strict proof posture on the same branch that filed
`KI-1` and `DEFECT-260726-QS2V`. (1) `Set([]byte("[1,2]"), []byte("9"), "[5]")`
silently overwrites the array (`[9]`) instead of appending (`[1,2,9]`); the
array-append branch at `parser.go:980-981` only fires when the array's first
element is an object, so scalar arrays fall through to the "over-write it with
a new object" `else` at `parser.go:985-988`. (2) `Set({"a":[{"x":1}]}, 9, "a", "")`
panics at `parser.go:981` on `keys[depth:][0][0]` — the *slice-expression
variant* of the empty-key dereference anti-pattern that `KI-1` claimed was
fully swept ("all seven sites" at `parser.go:410,616,722,744,755,773,796`,
`proof/known-issues/KI-1.yaml:19,23`). The common thread is narrow but
important: **the proof layers verify that documented behavior is implemented and
crash-free — they do not explore undocumented edge cases, check output
correctness against an independent reference, or recognize pattern variations
of an unsafe construct.** The hazard sweep found 7 of 8 sites; the missing 8th
site used `keys[depth:][0][0]` instead of `keys[i][0]`, a shape the sweep's
match pattern did not cover.

---

## 2. Bug 1 analysis — `Set` array-index-beyond-length data loss

**Defect.** `Set([]byte("[1,2]"), []byte("9"), "[5]")` returns `[9]`, destroying
the existing elements `[1,2]` instead of appending `9`. The root cause is the
compound condition at `parser.go:980-981`:

```go
if (data[startOffset] == '{' && data[startOffset+1+nextToken(data[startOffset+1:])] != '}') ||
    (data[startOffset] == '[' && data[startOffset+1+nextToken(data[startOffset+1:])] == '{') && keys[depth:][0][0] == 91 {
    depthOffset--          // append path: back up to before the closing bracket
    startOffset = depthOffset
} else {
    comma = false          // overwrite path: discard the existing container
    object = true
}
```

The second disjunct requires the array's first element to be an object
(`data[...] == '{'`). A scalar array `[1,2]` fails this test, the whole
condition is false, and the `else` branch rebuilds the container as a fresh
object — discarding the original elements. The code comment at `parser.go:984`
("otherwise, over-write it with a new object") documents the destructive
behavior as intentional.

### 2.1 Spec / requirements layer — GAP (this is the primary gap)

`SYS-REQ-009` (`specs/system/requirements/SYS-REQ-009.req.yaml:7-8`) says the
parser shall, for a provided path, *replace the existing addressed value,
**create a supported missing path** and return the updated document, or return
`KeyPathNotFoundError` when the requested mutation path is **not usable** for
the provided input.* Two terms are undefined for this bug:

- **"supported missing path"** — no boundary domain says whether an
  array-index-beyond-length like `[5]` on `[1,2]` is in the *append* partition,
  the *reject* partition, or the *overwrite* partition. The FRETish
  formalization (`SYS-REQ-009.req.yaml:7`) introduces only the boolean
  `set_creates_missing_path` / `set_target_exists` abstractions, which collapse
  every missing-path shape into one bucket. There is no variable distinguishing
  *append-to-array* from *replace-container*.
- **"not usable for the provided input"** — the reject boundary is undefined
  for the array-index-beyond-length case, so the audit's `solver_modeling_opportunity`
  and `partition_evidence_complete` checks have nothing to evaluate (`proof audit`
  reports both as "no z3-boundary requirements consume input data-constraint
  partitions").

The catalog already models a *nearby* scenario — `proof/catalog/scenario/negative_array_index.yaml`
— but its `description` scopes it to the **Get/arrayEach lookup side**
("nextValue/arrayEach iteration", `SYS-REQ-047`). `V-panic-negative-array-index.yaml:10`
closes the campaign explicitly on the Get side ("covered by FuzzGetNative … and
negative-index regression tests in `parser_test.go`"). The Set mutation side
has no scenario catalog entry, no SYS-REQ partition, and no vector.

**Could it have caught this?** Yes. A spec partition for
`set_array_index >= array_length` with a defined outcome (append or reject)
would have forced a test and surfaced the `else`-branch data loss.

### 2.2 MC/DC layer — did not catch (and structurally could not)

`proof audit` reports `code_mcdc_coverage` 100.0% decisions / 100.0% conditions
and `mcdc_coverage` "109 requirements checked, 369 witness rows total, 0
uncovered." The `SYS-REQ-009` witness rows are visible at
`set_spec_test.go:8-14` — they enumerate combinations of `set_creates_missing_path`,
`set_path_is_provided`, `set_target_exists`, etc. `TestSetCreatesMissingEntryInExistingArray`
(`set_spec_test.go:29-44`) exercises the TRUE branch of the `parser.go:980-981`
condition (array of objects: `{"top":[{"middle":[{"present":true}]}]}`) and pins
the append result. The FALSE branch is exercised by many rows in `setTests`
(`parser_test.go:270+`).

**Why it missed the bug.** MC/DC verifies that *each boolean sub-condition
independently affects the decision's outcome* — i.e., that the code path
*exists and is reachable*. It does **not** verify that the *output bytes* of
the path are correct. Both the TRUE branch (append) and the FALSE branch
(overwrite) were exercised; MC/DC marked the decision fully covered. The bug is
not that the overwrite path is unreachable — it is that the overwrite path
fires for a case (scalar array, index beyond length) where the *correct*
behavior is append. MC/DC has no notion of "correct"; it has only "exercised."

### 2.3 Fuzz layer — did not catch (reach gap)

`FuzzSet` (`fuzz.go:39-45`) is:

```go
func FuzzSet(data []byte) int {
    _, err := Set(data, []byte(`"new value"`), "test")
    ...
}
```

The key path is the hardcoded literal `"test"` — an object key, never an array
index, never beyond any array's length. `FuzzEachKey` (`fuzz.go:13-30`) likewise
hardcodes a fixed table of non-empty, in-bounds paths (`{"arr", "[1]", "b"}`,
`{"arrInt", "[3]"}`, …). `fuzz_native_test.go:21-39` (`nativeFuzzSeeds`) mutates
only the JSON byte input; no harness mutates the key path. The result: the
fuzzer explores the *input* space of `Set` but never the *path* space, so the
array-index-beyond-length input class is unreachable from any fuzz target.

`V-panic-no-path-mutation.yaml:14` claims the empty/no-path case is "covered by
FuzzSetNative, FuzzDeleteNative" — that claim is true only for the
zero-segment early return (`parser.go:936`), not for any path-shape mutation.

**Could it have caught this?** Only a path-mutating fuzzer could. The current
harness architecture cannot reach this bug.

### 2.4 Hazard-sweep layer — not applicable

The hazard sweep (`panic_free_input_handling` obligation class,
`DEFECT-260726-QS2V.yaml:5-7`) is a *crash* discovery pass — it looks for
unguarded dereferences on caller-controlled input. Bug 1 is silent data loss,
not a panic; the `else` branch at `parser.go:985-988` completes normally and
returns a well-formed document. No crash, no hazard-sweep signal. This layer
correctly did not fire for a non-panic defect.

### 2.5 Property-test layer — did not catch (oracle gap + reach gap)

`TestPropertySetRoundTrip` (`property_test.go:426-446`) generates a random key
via `randKey` and sets an integer value on `{}`, then round-trips via `Get`.
Two gaps:

- **Reach gap.** `randKey` (`property_test.go:82-90`) draws from
  `"abcdefghijklmnopqrstuvwx"` with length 1–6 — it never emits `""` and never
  emits an array-index form like `"[5]"`. The base document is always `{}` and
  the path is always a single object key, so `depth == 0` and control flow
  never enters the `parser.go:977` `depth != 0` block where the bug lives.
- **Oracle gap.** The test round-trips the *set value* through `Get` but never
  compares the full *output document* against an independent reference. If it
  had asked "does `encoding/json` agree that `Set([1,2], 9, "[5]")` yields
  `[1,2,9]`?", it would have caught the overwrite. The property suite uses
  `encoding/json` as an oracle for `ParseInt`/`ParseFloat`/`ParseBoolean`
  (`property_test.go:10`, `TestPropertyTypedAccessorsAgreeWithEncodingJSON`) and
  for `getType` classification (`property_test.go:589`) but **not** for `Set` /
  `Delete` output correctness.

**Could it have caught this?** Yes — a reference-oracle property test over
mutated (document, path) pairs would have caught it on the first iteration.

### 2.6 Code-signals layer — did not catch (not configured)

`proof audit` reports `code_signal_deadline_cast_unreviewed` as "no
proof/signals directory (deadline-cast signal hunt no-op)" and
`code_signal_unbindable` as "no code signals scanned (no implementation traces
or no `proposes_class` declarations)." There is no `proof/signals/` directory
and no project-specific signal rule. Even if one existed, Bug 1 is a
*correctness* defect (wrong branch chosen), not a recognizable syntactic
anti-pattern, so a code signal would not naturally fire here. This gap is
relevant mainly to Bug 2.

---

## 3. Bug 2 analysis — the 8th empty-key-component panic site

**Defect.** `Set([]byte("{\"a\":[{\"x\":1}]}"), []byte("9"), "a", "")` panics at
`parser.go:981` with `runtime error: index out of range [0] with length 0`. The
panic site is the third conjunct of the Bug 1 condition:

```go
... && keys[depth:][0][0] == 91 {
```

`keys[depth:]` = `[""]` (the trailing empty component), `[0]` selects it, and
`[0][0]` indexes the first byte of the empty string — unguarded. This is the
**same anti-pattern** `KI-1` and `DEFECT-260726-QS2V` addressed at the other
seven sites, but written in a syntactic shape the sweep did not match.

### 3.1 Spec / requirements layer — partial gap

`SYS-REQ-016` (not-found, `specs/system/requirements/SYS-REQ-016.req.yaml:7-8`)
and `SYS-REQ-035` (Delete malformed, `SYS-REQ-035.req.yaml:7`) define the
*crash-free on unusable input* contract, and `SYS-REQ-009` carries an
`idempotency` obligation (`SYS-REQ-009.req.yaml:48-52`). None of them enumerate
the **empty-string key path component** as an input partition. The closest is
`SYS-REQ-016`'s `missing_path` obligation (`SYS-REQ-016.req.yaml:66-71`), whose
worst-case is "Get returns a stale value slice … (silent data corruption)" —
a *data* hazard, not a *panic* hazard. There is no spec obligation that says
"an empty-string key component is a defined, panic-free boundary input." As a
result the bug class was discovered by a sweep, not by spec-driven partition
coverage.

**Could it have caught this?** A spec obligation naming the
empty-key-component boundary (with a defined not-found / unchanged-payload
outcome) would have required partition evidence across *every* code path that
consumes a key component, including `Set`'s `depth != 0` append branch.

### 3.2 MC/DC layer — did not catch (and structurally could not)

The `SYS-REQ-009` witness rows (`set_spec_test.go:8-14`) enumerate boolean
combinations of `set_creates_missing_path` × `set_path_is_provided` × …, but
`set_path_is_provided` is a single boolean; it does not model "path component
is the empty string." The compound condition at `parser.go:980-981` was marked
100% covered because both TRUE and FALSE outcomes were exercised by other test
rows — none of which used an empty trailing component. MC/DC measures branch
*exercisation*, not the *input partition space* of each operand, so a
never-tested value of an operand (the empty string) is invisible to it.

### 3.3 Fuzz layer — did not catch (reach gap)

`FuzzSet` (`fuzz.go:40`) hardcodes `"test"`. `FuzzEachKey` (`fuzz.go:14-27`)
hardcodes non-empty paths. `randKey` (`property_test.go:82-90`) never emits `""`.
`TestPropertyNoPanicOnArbitraryBytes` (`property_test.go:901-930`) calls
`Set(raw, []byte("x"), keys...)` with `keys = []string{randKey(r)}` — again,
non-empty. The empty-string-component partition is unreachable from every
fuzz and property harness. `V-panic-no-path-mutation.yaml:14`'s claim that the
empty-key case is "covered by FuzzSetNative" is accurate only for the
zero-segment early return, not for the empty-*component* dereference.

**Could it have caught this?** Yes — a path-mutating fuzzer that included `""`
in the path-segment corpus would panic here on the first run.

### 3.4 Hazard-sweep layer — found 7 of 8, missed the slice-expression variant

This is the central finding for Bug 2. The hazard sweep
(`DEFECT-260726-QS2V.yaml:117-127`, `KI-1.yaml:19,23`) audited "every
caller-controlled path-component dereference site" and concluded **"all seven
unguarded sites (`parser.go:410,616,722,744,755,773,796`) now carry the
`len(...) > 0` guard"** with `sibling_sweep.result: clean`
(`DEFECT-260726-QS2V.yaml:121`). The seven sites all share the shape:

```go
if len(keys[i]) > 0 && string(keys[i][0]) == "[" { ... }   // e.g. parser.go:722,796,835
```

The missed site at `parser.go:981` is:

```go
... && keys[depth:][0][0] == 91 {
```

**Why the pattern missed it.** Three structural differences defeat any sweep
anchored on the `keys[i][0]` / `p[level][0]` shape:

1. **Slice expression vs. simple index.** `keys[depth:]` uses the Go slice
   operator (colon), not a simple index `keys[i]`. A regex like
   `keys\[\w+\]\[0\]` does not match `keys\[depth:\]`.
2. **Double dereference.** `[0][0]` is a two-level index into `[][]byte`,
   whereas the swept sites are single-level `[0]` into `[]byte`.
3. **Byte literal vs. string compare.** `== 91` (the byte value of `'['`)
   is not the `string(...) == "["` or `... == '['` comparator the sweep
   associated with the array-index check.

The sweep was *conceptually* correct (find unguarded dereferences of
caller-controlled key components) but *syntactically* narrow: it matched one
concrete spelling of the anti-pattern rather than the abstract property
("a `[]` index whose operand is a caller-supplied string, without a preceding
`len(...) > 0` guard"). `KI-1.yaml:19`'s `tripwire_mutation` encodes exactly
that narrow spelling — "Revert any of the seven `len(...) > 0` guards" — so
even the tripwire cannot detect a regression at line 981.

**Could it have caught this?** Yes — a structural matcher over the AST
property "index expression whose indexed operand is (transitively) a
caller-supplied `string`/`[]byte`, not fronted by a length guard" would catch
all spellings. The current sweep is a regex over text.

### 3.5 Property-test layer — did not catch (reach gap)

`TestPropertyNoPanicOnArbitraryBytes` (`property_test.go:901-930`) asserts
panic-freedom on `Set(raw, []byte("x"), keys...)` for 3000 random byte inputs
— but `keys` is `[]string{randKey(r)}` and `randKey` never emits `""`
(`property_test.go:82-90`). The empty-key-component partition is unreachable.
`TestSetEmptyKeyPathComponent` (`empty_key_path_test.go:179-207`) *does* test
the empty component, but only on object roots (`{}`, `{"a":1}`) with `depth==0`
or `depth` reaching the top-level object insert path — never
`{"a":[{"x":1}]}` with a trailing empty component after a valid object key,
which is the only input that routes into `parser.go:977`'s `depth != 0` block
and reaches line 981.

### 3.6 Code-signals layer — did not catch (not configured)

Same as §2.6: no `proof/signals/` directory, no project-specific code-signal
rule (`proof audit`: `code_signal_unbindable` = "no code signals scanned"). A
project-specific rule of the form *"flag any unchecked `[]` dereference whose
operand is a `keys`/`paths` element, in any spelling (`x[i]`, `x[i:][j]`,
`x[i][j][k]`), not preceded by a `len(...) > 0` guard"* would have flagged
line 981 at review time. The built-in signal catalog does not encode this
project-specific anti-pattern, and no custom rule was declared.

---

## 4. The common failure mode

Across both bugs the proof posture exhibits one recurring, narrow blind spot,
which decomposes into three independent facets:

1. **Proof verifies DOCUMENTED behavior is implemented and crash-free; it does
   not explore UNDOCUMENTED edge cases.** Both bugs live in input partitions
   no spec modeled — array-index-beyond-length on a scalar array (Bug 1) and
   the empty-string key component on every API (Bug 2). Every audit check that
   passed (`coverage_met` 116/116, `mcdc_coverage` 369 rows / 0 uncovered,
   `code_mcdc_coverage` 100.0%/100.0%) is anchored on the *specified* obligation
   set; an unspecified partition is simply not in any check's denominator.
   `solver_modeling_opportunity` and `partition_evidence_complete` both
   reported "no z3-boundary requirements consume input data-constraint
   partitions" — the absence of a modeled boundary is not itself a failing
   signal.

2. **Proof checks that code paths are EXERCISED, not that their OUTPUT is
   CORRECT against a reference.** MC/DC marked the `parser.go:980-981`
   decision 100% covered because both branches were reached. The
   property suite uses `encoding/json` as an oracle for scalar parsing and
   type classification (`property_test.go:10,589`) but **not** for the
   mutation helpers (`Set`/`Delete`) — so there is no test that says "the
   document `Set` produces must equal the document `encoding/json` produces
   for the same logical mutation." Bug 1 is a pure output-correctness defect
   and is therefore invisible to every layer that lacks an oracle.

3. **Proof's pattern matchers are concrete spellings, not abstract
   properties; they miss PATTERN VARIATIONS of an unsafe construct.** The
   hazard sweep found 7 of 8 empty-key dereference sites because all 7 shared
   the exact spelling `keys[i][0]` / `p[level][0]`. The 8th site,
   `keys[depth:][0][0]`, differs by a slice operator, a second index level,
   and a byte-literal comparator — three variations the regex did not
   abstract over. The sweep's *concept* (unguarded caller-controlled index)
   was right; its *match pattern* was too literal.

In one sentence: **the proof toolchain proves that the code does what the spec
says, safely, in the shapes the spec and sweep recognize — it cannot find
behavior the spec never named, correctness no oracle checked, or unsafe
spellings the matcher never generalized.**

---

## 5. Remediation map

Each gap maps to one concrete remediation track running in parallel on this
branch.

| # | Gap (this RCA) | Remediation | Concrete artifact / change |
|---|----------------|-------------|----------------------------|
| R1 | **Spec gap** — no partition for `Set` array-index-beyond-length (§2.1) | New `SYS-REQ` with a boundary domain: `set_array_index` ∈ {`in_bounds`, `== array_length`, `> array_length`} × container element-type ∈ {`scalar`, `object`}, defining append-vs-reject semantics. | New `SYS-REQ` under `specs/system/requirements/` extending `SYS-REQ-009`'s `set_creates_missing_path` partition with the array-index boundary domain; `solver_modeling_opportunity` and `partition_evidence_complete` then have a real partition to consume. |
| R2 | **Spec gap** — no contract for the empty-string key component (§3.1) | New `SYS-REQ` defining the empty-key-component boundary contract (typed not-found / unchanged-payload across `Get`/`Set`/`Delete`/`EachKey`). | New `SYS-REQ` under `specs/system/requirements/` carrying an `empty_key_component` obligation class; back-links from `SYS-REQ-009`, `SYS-REQ-016`, `SYS-REQ-035`. |
| R3 | **Correctness gap** — no output oracle for mutations (§2.5) | Reference-oracle property tests: generate random `(document, path, value)` triples, apply `Set`/`Delete`, and assert the full output document equals the document produced by an `encoding/json`-backed reference mutation. | Extension of `property_test.go` (new `TestPropertySetAgreesWithReferenceOracle`, `TestPropertyDeleteAgreesWithReferenceOracle`); closes the `property_based_test_coverage` advisory for `Set`/`Delete`. |
| R4 | **Pattern-variation gap** — hazard sweep matched only `keys[i][0]` (§3.4) | Project-specific code-signal rule that flags any unchecked `[]` dereference whose operand is (transitively) a `keys`/`paths` element, in **all** spellings: `x[i]`, `x[i:][j]`, `x[i][j][k]`, with and without a preceding `len(...) > 0` guard. | New `proof/signals/` directory with a rule declaration; `code_signal_obligations_reviewed` / `code_signal_unbindable` then have a real signal set to enforce. `KI-1.tripwire_mutation` is generalized to all spellings. |
| R5 | **Fuzz-reach gap** — fuzzers mutate JSON bytes only, never the key path (§2.3, §3.3) | Path-mutating fuzzer: a `go test -fuzz` target whose corpus includes the key path (slice of string segments, including `""` and `"[N]"` for `N > length`), so the path input space is explored alongside the byte input space. | New `Fuzz*WithPath` harness in `fuzz_native_test.go`; `run_fuzz_campaign.sh` extended to drive it; supersedes the `V-panic-no-path-mutation.yaml:14` claim that the path side is covered by the byte-only harness. |

The fix for **Bug 1's code** is to widen the `parser.go:980-981` append
condition so scalar arrays also take the append branch (or, per R1's spec
decision, to reject the path with `KeyPathNotFoundError`). The fix for
**Bug 2's code** is to add the same `len(...) > 0` guard already present at the
seven swept sites, written for the `keys[depth:][0]` operand. Both code fixes
land via the parallel remediation work; this RCA owns only the gap analysis.

---

## 6. What proof did right

This analysis exists to *close* gaps, not to dismiss the proof toolchain.
Several layers performed exactly as designed and deserve explicit credit:

- **The hazard sweep found 7 of 8 sites.** `DEFECT-260726-QS2V.yaml` records a
  real blind-discovery pass that surfaced a panic class no fuzzer could reach,
  and `KI-1` locked the fix with a tripwire. The 8th site escaped on a spelling
  variation, not because the sweep's *concept* was wrong — R4 generalizes that
  concept rather than replacing it.
- **MC/DC delivered honest 100% coverage of the code that existed.**
  `code_mcdc_coverage` 100.0% decisions / 100.0% conditions and `mcdc_coverage`
  369 witness rows / 0 uncovered mean every branch in the shipped code is
  exercised. The gap is that MC/DC's denominator is the *code*, not the *input
  partition space* — R1/R3 add the partition and oracle that complete the
  denominator.
- **The formalization caught real invariant violations.** `behavioral_implications_verified`
  (4/4 proved), `z3_properties_verified` (16 subjects, 20/20 proved),
  `data_constraint_z3_coverage` (4/4), and `lemma_branch_coverage` (18/18) all
  pass. The OSS-Fuzz Delete leading-comma panic (`SYS-REQ-035`,
  `parser.go:907-913`) was caught and closed precisely because it *was*
  specified as a `malformed_input` obligation with a Z3-falsifiable worst case
  (`SYS-REQ-035.req.yaml:52-54`). The formalization's record on specified
  hazards is strong; R1/R2 extend it to the two unspecified boundaries.
- **The catalog scenario model is internally sound.** `negative_array_index.yaml`
  correctly scopes its coverage claim to the Get/arrayEach side and closes the
  campaign honestly. The gap is that no symmetric scenario exists for the Set
  mutation side — an omission R1 remedies.

The proof posture is not broken; it is *incomplete along three specific axes*
(undocumented partitions, missing output oracle, literal pattern matching).
Sections 2–5 name those axes precisely so the remediation in §5 can close them
without retreating from the layers that worked.
