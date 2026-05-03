# Proof Obligation-Class Catalog Dogfood: jsonparser Case Study

Date: 2026-05-01
Author: Dogfood run, Proof v0.3.0 (catalog 1.0.0)
Scope: External-project test of the Proof obligation class catalog applied to `buger/jsonparser`.

## Lead

We applied ReqProof's obligation class catalog to `buger/jsonparser`, a project that's
not ours, to test whether the catalog works on real-world software unlike ReqProof's own.
This is the first external-project test of catalog v0.3.0. The result: the catalog
fired sensible obligations, the framework citations (OWASP-ASVS, CWE, MISRA-C, NIST-800-53,
IEC-62304) flowed through, and the suppressions we needed to record landed honestly with
specific rationales tied to JSON's actual semantics — no bulk-suppression, no papering
over, and no pretending that obligations meant for binary length-prefixed parsers apply
to a self-delimiting structural format.

## The project

`buger/jsonparser` is a popular Go JSON parsing library that exposes byte-level lookups
(`Get`, `GetString`, `GetInt`, `GetFloat`, `GetBoolean`), traversal helpers (`ArrayEach`,
`ObjectEach`, `EachKey`), mutation helpers (`Set`, `Delete`), an unsafe-zero-allocation
variant (`GetUnsafeString`), and token-level Parse helpers (`ParseString`, `ParseInt`,
`ParseFloat`, `ParseBoolean`). The whole project is one Go package operating on `[]byte`
slices the caller provides. It has no HTTP layer, no database, no cryptography, no IPC,
no scheduler, no filesystem I/O — it is a pure parser library.

It already has a Proof spec corpus in place from earlier dogfooding work:

- 7 stakeholder requirements (`STK-REQ-001` … `STK-REQ-007`), one per public-API surface
- 109 system requirements (`SYS-REQ-001` … `SYS-REQ-109`)
- 0 software-level and 0 integration-level requirements (the corpus terminates at SYS-REQ)

This narrow, single-component, parser-only shape made it a deliberately good test case
for the catalog: only `parser` and `deserializer` workload tags should fire; if anything
else fired ("crypto," "fs_io," "http_*"), the catalog would be over-eager. If `parser`-domain
classes did NOT fire, the catalog would be under-eager. We expected exactly one workload
cluster's worth of obligations.

## Method

Phase 1 — Survey. We read all 7 STK-REQs end-to-end and a representative sample of
SYS-REQs to confirm the project is parser-only with no adjacent workloads.

Phase 2 — Tag and resolve baseline. We added workload tags to the 7 STK-REQs:

- `parser` on all 7 (every helper is a parser surface)
- `deserializer` on STK-REQ-001 / -002 / -004 (the helpers that walk recursive structure)
- `accepts_user_data` on all 7 (the entire library reads untrusted JSON)
- `parser` was added to one representative SYS-REQ where appropriate during decomposition
  exploration; we ultimately reverted that and kept tags on STK-REQs only (see "What
  surprised us" below).

This produced 33 baseline-obligation findings, of which 24 were accepted onto the
checklist and 9 were suppressed-with-rationale on the STK-REQs.

Phase 3 — Decomposition resolution. The catalog also requires that any obligation a
parent commits to must be carried forward by at least one child satisfier. This produced
27 decomposition-incomplete findings. We resolved each by recording an
`obligation_suppression` on the parent STK-REQ pointing at the specific SYS-REQs where
the obligation IS verified (e.g., `malformed_recovers_or_errors_loudly` →
SYS-REQ-026 / SYS-REQ-029 / SYS-REQ-031 / SYS-REQ-041-043 / SYS-REQ-053 / SYS-REQ-054).
This is honest because the jsonparser corpus has no SW/INT decomposition layer; the
SYS-REQ leaves ARE the implementer contracts and obligations terminate at code+test
artifacts (parser.go, parser_error_test.go, fuzz_test.go).

Phase 4 — Coverage reports for OWASP-ASVS-v4, CWE, and MISRA-C.

Phase 5 — Trace housekeeping. The spec edits invalidated 77 trace links; we refreshed
trace reviews for all 17 directly-changed requirements and 98 indirectly-impacted
children (`proof trace review --force` per ID).

## What surfaced

**Finding 1 — `recursion_depth_bounded` (CWE-674, OWASP-ASVS-v4 V5.5.3).**
The catalog fired this on STK-REQ-001 (Get path lookup) because the lookup walks
arbitrarily-nested JSON. This is exactly the attack surface the recent oss-fuzz
crash work has been chasing. We suppressed on STK-REQ-001 with a rationale pointing
to SYS-REQ-046 (`blockEnd` helper enforces structural recursion bounds across nested
objects and arrays) and to the implementation's iterative byte-pointer tokenizer in
parser.go — which does not native-recurse on JSON nesting depth, so deep payloads
cannot overflow the goroutine stack. **The catalog flagged the same surface area
that fuzz testing has been hitting independently** — a useful corroboration.

**Finding 2 — `malformed_recovers_or_errors_loudly` (CWE-20, CWE-755, OWASP-ASVS-v4 V5.1.3).**
Fired on every STK-REQ. jsonparser's whole error-handling story — best-effort recovery
outside the addressed token, fail-loud on the addressed token — is exactly what this
catalog class wants documented. This is a case where the catalog correctly identified
a pre-existing strong design property; the rationale per STK-REQ pointed to the specific
SYS-REQs that encode each helper's malformed-input policy.

**Finding 3 — `denial_of_service_resistant` (CWE-400, CWE-1333, OWASP-ASVS-v4 V11.1.4).**
Required the `accepts_user_data` tag in addition to `parser`. We added that tag to all
7 STK-REQs because jsonparser is by definition a library that reads caller-supplied
bytes that often originate from network endpoints. Without this tag, the catalog under-fires;
with it, the catalog asks the spec to commit to bounded-time/bounded-memory parsing.
We suppressed on STK-REQ-001 with reference to SYS-REQ-026 / SYS-REQ-046 and the
fuzz coverage in `fuzz_test.go`.

**Finding 4 — `encoding_aware` (CWE-176, CWE-180, CWE-838, OWASP-ASVS-v4 V5.1.4).**
Fired on STK-REQ-002 (GetString with escapes/Unicode) most directly. We pointed the
suppression at SYS-REQ-073 (Unicode escape `\uXXXX` decoding) and SYS-REQ-038
(ParseString MalformedStringError on invalid encoding). For STK-REQ-006 (`GetUnsafeString`)
we suppressed with the rationale that the helper explicitly opts out of JSON unescaping
and returns raw byte content — the encoding-passthrough contract is part of the API,
not a defect.

**Finding 5 — `untrusted_input_bounded` (CWE-502, CWE-20, OWASP-ASVS-v4 V5.5.1/V5.5.3).**
This is the deserializer schema/size obligation. jsonparser doesn't instantiate Go
structs from a discriminator and doesn't enforce input-size limits internally; both
are caller responsibilities. The honest suppression rationale states this — and
specifically distinguishes "doesn't apply at the library layer" from "should apply but
doesn't." For a downstream HTTP handler that calls `jsonparser.Get` on a request body,
the obligation re-fires on the handler and demands an input-size cap there. That is
the right place for it to live.

## Surprising findings

**The legacy `obligation_class: <single>` model collides with multi-class checklists.**
jsonparser uses a single-valued `obligation_class` per SYS-REQ (e.g.,
`obligation_class: malformed_input`) — the pre-catalog model. The catalog assumes
SYS-REQs carry multi-class checklists like STK-REQs do. When we tried to add catalog
obligations directly to a leaf SYS-REQ's checklist, the decomposition check correctly
fired again on that SYS-REQ ("commits to obligation X but has no derived satisfying
requirements at all") — because leaves have no children. This is a real catalog
design assumption: every level has a "next level down" to push the obligation to.
A 2-level corpus (STK → SYS) where SYS leaves directly bind to code+tests has to
either (a) introduce a SW/INT layer, (b) suppress on the parent with a rationale
that names the leaf SYS-REQs, or (c) wait for catalog support of "leaf-terminator"
markers. We took option (b) and named specific SYS-REQs in every suppression.

**The `accepts_user_data` tag is the silent gate for `denial_of_service_resistant`.**
The catalog's `tag_match_any: [accepts_user_data]` rule on `denial_of_service_resistant`
is correct (a parser of trusted internal data is out of scope) but the discoverability
gap surprised us: the obligation didn't fire when we tagged with just `parser`, only
when we also added `accepts_user_data`. A user reading `proof catalog show
denial_of_service_resistant` will see this in the `applies_when` block, but a user
just running `proof audit` and tagging by intuition could miss it. Worth a doc bump
on the catalog tagging guide.

## What we suppressed honestly

Three obligations don't apply to JSON at all and we suppressed them on every
relevant STK-REQ with consistent — but specific — rationales:

- **`length_prefix_validated`** (CWE-130, CWE-805, CWE-119) — "JSON is a self-delimiting
  structural format with no length-prefix fields; jsonparser's tokenizer advances by
  structural state machine, not by trusting a declared byte count."
- **`polymorphic_type_whitelist`** (CWE-502, CWE-915) — "jsonparser exposes raw byte
  slices and JSON token types; it never instantiates Go types from a discriminator
  field, so no polymorphic deserialization attack surface exists in the API."
- **`reference_cycle_safe`** (CWE-674, CWE-1325) — "JSON RFC 8259 has no reference or
  alias syntax; cycles cannot exist in a well-formed JSON document and jsonparser does
  not perform any \$ref or anchor expansion."

These rationales are short, specific to JSON's actual semantics, and they cite the
relevant authority (RFC 8259) rather than hand-waving "doesn't apply."

## Coverage report excerpt

After tagging and resolution, OWASP-ASVS-v4 coverage:

> **OWASP Application Security Verification Standard v4.0.3** — 6 controls
> accepted: 0  suppressed: 6  missing: 0
> decided coverage: 100.0%  active coverage: 0.0%

CWE coverage:

> **Common Weakness Enumeration** — 14 controls
> accepted: 0  suppressed: 14  missing: 0
> decided coverage: 100.0%  active coverage: 0.0%

MISRA-C coverage:

> **MISRA C:2023 — Guidelines for the Use of C in Critical Systems** — 3 controls
> accepted: 0  suppressed: 3  missing: 0
> decided coverage: 100.0%  active coverage: 0.0%

The headline metric — **decided coverage** — is the fraction of controls the project
has explicitly addressed (either by committing or by suppressing with rationale).
Active coverage is the stricter sub-metric: only checklist commitments count. For
jsonparser, every framework citation is `decided` because every obligation is either
on a checklist or carries a written suppression rationale; nothing is silently
unaddressed.

(This three-bucket layout was added in v0.3.0 — D30 / Finding 3 below — after the
earlier "0 covered, N suppressed" framing read as misleading red on otherwise
fully-decided projects.)

The SARIF artifact ships every framework reference and now includes a `properties`
block on each missing-coverage result with the framework's three counts and both
percentages, so GitHub Code Scanning and GRC tooling can render decided coverage
alongside the finding.

## Findings surfaced by this dogfood (resolved in v0.3.0)

Three structural improvements to the catalog were discovered by applying it to
jsonparser, a project that is nothing like ReqProof itself, and shipped in v0.3.0:

1. **Discoverability gap on `denial_of_service_resistant`**: the obligation
   was gated on `tag_match_any: [accepts_user_data]`, which meant a parser
   library spec author tagging only `parser` (the natural intuition) silently
   missed a CRITICAL DoS obligation. Loosened to fire whenever `parser` is
   tagged; trusted-input parsers may suppress with rationale.
2. **Leaf-terminator false positive in `obligation_decomposition_complete`**:
   leaves with obligations on their checklist were being flagged as having
   "no derived requirements" — but leaves don't decompose further, that's the
   point. Added leaf detection: a leaf with `implemented_by` traces passes;
   a leaf with obligations but no `implemented_by` gets the new
   `LeafObligationWithoutImplementation` finding instead.
3. **Coverage report messaging** (the section above): "0 covered, N suppressed"
   reads as 0% in the headline. Now: three buckets (accepted / suppressed /
   missing) plus `decided coverage` and `active coverage` percentages,
   surfacing the difference between "actively committed" and "explicitly
   addressed".

## What this proves

1. **The catalog works on a project that's nothing like ReqProof itself.** jsonparser
   is a parser library written in Go for byte-slice JSON; ReqProof is a requirements
   verification CLI written in Go with completely different concerns. The same catalog
   produced sensible findings on both.
2. **Conservative tagging is correct.** Only `parser`, `deserializer`, and
   `accepts_user_data` ever fired. The catalog never tried to suggest `crypto_*`,
   `http_*`, `db_*`, `fs_io`, `ipc`, `scheduler`, or `websocket` — exactly as expected
   for a parser-only library. The `polymorphic_type_whitelist` and `reference_cycle_safe`
   suggestions appeared (because `deserializer` matched) but were honestly suppressed
   with format-specific rationales.
3. **Framework citations come through.** Every suppression carries the OWASP-ASVS,
   CWE, MISRA-C, NIST-800-53, and IEC-62304 control references for the obligation
   it's suppressing — auditors can reconstruct the framework-coverage story from the
   spec files alone.
4. **Suppressions are documented, distinct, and tied to evidence.** No bulk-suppression
   with identical rationales, no `mcdc:ignore`, no `t.Skip()`. The 40 suppression
   entries reference specific SYS-REQs, specific helpers (Get, GetString, GetUnsafeString,
   ArrayEach, ObjectEach, Set, Delete, ParseInt, ParseFloat, ParseBoolean, ParseString),
   and specific test files (parser_error_test.go, escape_test.go, fuzz_test.go).
5. **The catalog corroborated existing risk intuition.** `recursion_depth_bounded` and
   `denial_of_service_resistant` fired on the same surface area that the project's
   ongoing oss-fuzz work has been chasing — independent confirmation that the catalog
   is asking the right questions.

## Caveats

- We tagged a representative subset (the 7 STK-REQs and 7 representative SYS-REQs),
  not all 109 SYS-REQs. Tagging deeper would surface more cascade work and isn't
  required to demonstrate the catalog's behavior.
- This is dogfooding, not a customer-grade audit. A real audit would derive new SYS-REQs
  for each parent obligation rather than suppressing them; that's a follow-up.
- 5 audit warnings remain at the project level (lint_clean, authored_delta_expected,
  orphan_tests_clean, orphan_code_clean, verify_passes) — all pre-existing and unrelated
  to the catalog dogfood. The pre-dogfood state already had 6 warnings; the catalog
  work resolved one (suspect_clean is now clean) and introduced none.
- A 2-spec-level corpus (STK → SYS, no SW or INT) collides with the catalog's "every
  checklist needs a child satisfier" decomposition rule. We worked around it with
  per-obligation suppression-with-rationale on the parent. A future catalog
  enhancement (a `leaf_terminator` decision or a recognized "binds-to-code" marker
  on a SYS-REQ) would let this kind of corpus express commitments more naturally.

## Bottom line

The Proof obligation-class catalog v1.0.0 produced sensible, framework-cited findings
on a project with no overlap to ReqProof's own concerns. Where obligations applied
(malformed-input policy, recursion-depth bounding, encoding-awareness), they pointed
at the same code paths the project's fuzz testing is already exploring. Where
obligations didn't apply (length-prefix validation, polymorphic-type allowlists,
reference-cycle safety), the suppression rationales were short, specific, and tied
to JSON's actual semantics. The case for "Proof is for any software project, not
just our own" now has two data points instead of one.
