# Ten years of jsonparser: from "10x faster" to a six-release comeback

Ten years ago, I was building a company dashboard. It needed to analyze gigabytes of JSON data — server logs, API responses, analytics events — extract insights, and visualize them in real time.

I had a choice. I could set up a complex database pipeline: ingest, transform, index, query. I could build aggregation layers and ETL jobs and wait minutes for each dashboard refresh. That's what everyone did. That's what you're supposed to do.

I didn't want to do that.

I wanted real-time analytics from plain files. No database. No ingestion pipeline. No infrastructure to maintain. Just read the JSON, pull out the fields I needed, and show the numbers. The problem was that Go's standard `encoding/json` was too slow for that — deserializing gigabytes of JSON into structs or `map[string]interface{}` took too long and allocated too much memory.

**Performance is a feature.** That's the conviction that started everything.

I didn't need a full parser. I needed three fields out of a 10KB JSON document. Why deserialize the entire thing? Why build a tree? Why allocate? If I just walked the bytes and extracted the path I wanted, I could skip 90% of the work.

So on March 20, 2016, I released [buger/jsonparser](https://github.com/buger/jsonparser) — a path-based, zero-allocation JSON parser that worked directly on bytes. The original README made a wonderfully restrained claim:

> "Alternative JSON parser for Go (so far fastest)"

It was 10x faster than `encoding/json`. It allocated zero bytes. And it was enough. The real-time dashboard worked — reading gigabytes of JSON from plain files, extracting insights on the fly, no database required. People like to overengineer. I like plain files and fast code.

The library caught on. Over the next decade it accumulated 5,600+ stars and 450+ forks. But the ecosystem moved. Generated-code parsers like easyjson and ffjson became serious competitors. Go's own `encoding/json` improved. The claim gradually softened from "the fastest" to "one of the fastest."

The library kept working, though. And it spread further than anyone tracked — including the author.

## It's inside Docker. And Istio. And Grafana.

When I started looking at who actually depends on jsonparser, I expected to find a few API-gateway projects and some CLI tools. What I found was more interesting.

The **direct dependents** — projects whose `go.mod` explicitly lists `github.com/buger/jsonparser` — include:

| Project | Stars | What it is |
|---|---:|---|
| [lux](https://github.com/iawia002/lux) | 31,570 | Video download library and CLI |
| [Grafana Loki](https://github.com/grafana/loki) | 28,637 | Log aggregation ("like Prometheus, but for logs") |
| [Tyk](https://github.com/TykTechnologies/tyk) | 10,777 | API Gateway |
| [Keybase](https://github.com/keybase/client) | 9,232 | End-to-end encryption platform |
| [Coroot](https://github.com/coroot/coroot) | 7,848 | Open-source APM |
| [Containernetworking/plugins](https://github.com/containernetworking/plugins) | 2,559 | Kubernetes CNI networking plugins |
| [Solana Go SDK](https://github.com/solana-foundation/solana-go) | 1,571 | Solana blockchain SDK |
| [Sentry Go SDK](https://github.com/getsentry/sentry-go) | 1,102 | Official Sentry error tracking |
| [Wundergraph graphql-go-tools](https://github.com/wundergraph/graphql-go-tools) | 828 | GraphQL Router / Federation |

Twenty-one confirmed projects, over 120,000 combined stars.

But the interesting part is the **transitive chain**. Some of those direct dependents are themselves infrastructure libraries that the biggest projects in the world build on:

```
jsonparser
  └── containernetworking/plugins (CNI)
        ├── istio/istio (36,000+ ⭐) — the service mesh
        ├── moby/moby (69,000+ ⭐) — Docker itself
        └── aws/eks-distro — Amazon's Kubernetes distribution
```

That's right. **Docker and Istio transitively ship jsonparser** through the CNI plugin chain. Every Kubernetes cluster running Istio CNI networking is executing jsonparser code on every network configuration read. Most of the engineers running those clusters have no idea.

The Solana chain reaches cross-chain:

```
jsonparser
  └── solana-foundation/solana-go
        └── wormhole-foundation/wormhole (1,893 ⭐)
              — the blockchain interoperability protocol
```

And the GraphQL Federation world:

```
jsonparser
  └── wundergraph/graphql-go-tools
        └── wundergraph/cosmo (1,242 ⭐)
              — the open-source Apollo Studio alternative
```

Adding it up — direct plus transitive — jsonparser's code reaches projects with over **230,000 combined GitHub stars**. That's not a library people talk about at conferences. It's a library that's *already there*, quietly running, when you deploy a container, route traffic through a service mesh, query a log database, or bridge a blockchain.

## In July 2026, I went back in

Six releases shipped: v1.3.0 through v1.6.0. The scoreboard:

- 50 open GitHub issues → zero
- 12 open pull requests → zero
- 4 formally tracked known issues → zero
- 12 real bugs found and fixed
- 25+ new backward-compatible APIs added
- Zero breaking changes

And after a decade of competition, jsonparser was once again the fastest parser in the large-payload benchmark.

But the interesting part is not the scoreboard. It is what we found while getting there.

## The bugs hiding inside mature code

The comforting myth about mature libraries is that their remaining bugs must be obscure. Some of these were obscure. Others were embarrassingly small.

Passing an empty string as a key-path component could panic. Not in one place, but across **eight separate indexing sites**. The first sweep found seven. Then a root-cause review found an eighth instance written in a slightly different syntactic shape: `keys[depth:][0][0]`. That is a useful lesson in miniature: a search for one unsafe pattern can report "all clear" while the same assumption survives behind different syntax.

Another bug failed silently. Setting an out-of-range index on a scalar array could **destroy the original elements** — `Set([1,2], "9", "[5]")` could produce `[9]` instead of `[1,2,9]`. No panic. Valid JSON. Just data loss. This bug lived inside Grafana Loki's parser. Inside Tyk's API gateway. Inside every project that trusted `Set` with array paths.

`Delete` had paths that could leave a trailing comma, producing malformed output such as `{"a":1,}`. Cross-type `Set` operations could generate invalid JSON when an array-style path met an object — or vice versa.

Unicode contributed its traditional parser misery. Lone UTF-16 surrogates were being synthesized into bogus non-BMP code points instead of following the behavior expected from Go's standard library. `ParseInt("-")`, meanwhile, had the opposite personality: it confidently returned `(0, nil)` for a minus sign with no digits.

Then there was input-buffer aliasing. `Set` and `Delete` could corrupt the caller's input via `append` on a slice with spare capacity. The returned slice looked fine, but the original buffer's backing array could be quietly overwritten. This class of bug is particularly nasty in Go because slices share backing arrays — and the caller has no way to know their input was mutated.

There were semantic inconsistencies too. `Get(data, "key", "[0]")` could find a value while `EachKey` failed on the same path. `ArrayEach` could invoke its callback on a non-array root using stale offsets. `EachKey` could panic when given more than 64 path components.

None of these are glamorous bugs. That is precisely why they matter. Parser failures live at transitions: empty to non-empty, scalar to container, object path to array path, high surrogate to missing low surrogate, slice length to slice capacity. Happy-path tests exercise values. The bugs live between categories.

## Formal verification — and what it did not magically solve

The comeback began with making jsonparser the **first Go library proven to L3 assurance** by [ReqProof](https://reqproof.com).

The resulting proof case contains 123 formal requirements, full traceability across the public API, 100% Modified Condition/Decision Coverage, and an audit result of zero errors and zero warnings. The proof artifacts live with the code rather than in a detached report.

MC/DC is stronger than ordinary line or branch coverage. It requires evidence that each condition in a decision can independently affect that decision's outcome. It is used in safety-critical software because "we executed this line once" says very little about a compound boolean expression.

But 100% MC/DC does not mean "there are no bugs." We proved that distinction almost immediately.

The scalar-array corruption path was covered. Both sides of its main decision were exercised. The problem was that the wrong input category entered the wrong branch. MC/DC knew the branches were reachable; it did not know which output was semantically correct.

That led to a [blameless proof-gap postmortem](https://github.com/buger/jsonparser/blob/master/docs/proof-gap-root-cause.md). Every escaped defect forced the assurance model to get stronger. For the aliasing bug, we added an explicit "must not mutate the input buffer" obligation. For the `EachKey` inconsistency, we added a cross-API consistency gate. For the benchmark mistake, we added a benchmark-honesty lint.

## Why ordinary fuzzing missed it

jsonparser had already been through OSS-Fuzz. It found a real `Delete` panic. It still missed several of the bug classes above.

The reason was reachability. The fuzz harness mutated JSON bytes, but used fixed key paths like `"test"`. No amount of byte mutation can discover a panic that requires an empty path component if the path is never mutated. Likewise, a fuzzer cannot reach an out-of-range array-index mutation when it only generates ordinary object keys.

So I built [probelabs/json-fuzz](https://github.com/probelabs/json-fuzz) — a structure-aware JSON fuzzer. It generates valid JSON from a grammar, then applies mutations at meaningful structural boundaries: after a colon, inside a Unicode escape, before a closing delimiter, or in an adversarial key path. It runs at roughly 250,000 inputs per second and checks more than "did it crash?" — its gates include output validity, round trips, numeric differential tests against `encoding/json`, offset bounds, determinism, aliasing, and input preservation.

That fuzzer found bugs years of normal use and blind mutation had missed.

The approach generalized. [probelabs/graphql-fuzz](https://github.com/probelabs/graphql-fuzz) uses grammar-aware generation for GraphQL operations, schemas, and Apollo Federation directives. Full-pipeline testing found a **production-reachable `todo!()` panic** in [Apollo Hive Router's schema transformer](https://github.com/graphql-hive/router) when processing Federation 1 type extensions — a pattern pervasive in real supergraphs.

That was a satisfying side quest: work intended to harden a ten-year-old JSON parser produced a reusable method that found a new bug in an unrelated Rust GraphQL router.

## The benchmark had been lying since 2017

The funniest — and most painful — discovery was sitting in [issue #126](https://github.com/buger/jsonparser/issues/126), opened in October 2017.

The benchmark's payload types had ffjson-generated `MarshalJSON` and `UnmarshalJSON` methods. When the supposed `encoding/json` benchmark received those types, Go correctly called their custom methods. So the "encoding/json" column was not really measuring `encoding/json`. It was measuring ffjson-generated code through the standard-library interface.

The issue was right. It remained open for almost nine years.

We fixed the benchmark by introducing plain payload types, updated all comparison libraries to their latest versions, recorded the hardware and Go version, and took results as the median of five runs.

Then the honest benchmark exposed something else: jsonparser itself had become unexpectedly slow on the large payload.

## Five frames, 30 ideas, 15 worktrees

For the performance investigation, I used an "ADHD" skill: a structured process that looks at the same problem through multiple independent cognitive frames — hardware engineer, competitor, speedrunner, assumption-remover, biologist — deliberately producing divergent hypotheses before converging.

Five frames generated 30 optimization ideas. Fifteen of those became isolated worktree experiments. Most did not win. That is important: performance work is not the art of having one clever idea. It is the discipline of making many claims cheap to falsify.

The biggest win came from an almost absurd piece of repeated work. `stringEndConfig` found the first quote, then searched the **entire remaining parent document** for a backslash. On a 24KB payload, it could walk tens of kilobytes for every string — looking for a character that wasn't there.

Bounding that scan to the actual string body took the large benchmark from roughly 128µs to 22µs — a **5.8x improvement** from a one-line fix.

The second winning experiment replaced two separate SIMD scans (one for `"`, one for `\`) with a single inline 8-byte SWAR (SIMD-Within-A-Register) loop that tests for both characters simultaneously. That shaved off another eight percent.

We also evaluated [StringZilla](https://github.com/ashvardanian/StringZilla) — the CGo SIMD library. Go's built-in `bytes.IndexByte` on ARM64 uses hand-tuned NEON assembly at 192 GB/s. StringZilla's NEON path topped out at 109 GB/s. The CGo call overhead alone (20ns) was 13x the cost of `bytes.IndexByte` on an 8-byte string. No crossover point at any size. The standard library won.

The final large-payload results on an Apple M4 Max with Go 1.26.3, all libraries at latest versions:

| Parser | Time | Allocations |
|---|---:|---:|
| **jsonparser** | **20,788 ns** | **0** |
| easyjson | 32,765 ns | 134 |
| ffjson | 60,129 ns | 144 |
| `encoding/json` | 134,123 ns | 147 |

**6.4x faster than encoding/json. 1.6x faster than easyjson. The only zero-allocation parser.** Ten years later, the "fastest" claim came back — but this time on a corrected benchmark with a guard against repeating the old mistake.

## Ten years of feature requests, without breaking ten years of users

Closing the backlog did not mean mechanically closing old tickets. Many represented legitimate gaps in the API.

The new additions include error-returning iteration (`ArrayEachErr`, `EachKeyErr`), safe string construction (`Escape`, `SetString`), `GetArrayLen`, `GetObjectLen`, `GetUint64`, `DeleteFound`, wildcard paths (`SetWildcard`, `EachKeyWildcard`), a JSONPath compiler (`ParsePath`, `CompilePath`), a `Config` type with opt-in single-quote and lenient-escape support, a streaming `ReaderParser` for 10GB+ inputs, unified naming aliases (`EachArray`, `EachObject`), and `Append` for clean array-appending without knowing the length.

That compatibility policy was non-negotiable. Six releases is aggressive enough; forcing thousands of downstream users — including Docker, Istio, and Grafana — through a migration at the same time would have been irresponsible.

## What a comeback really means

The repository is now at [v1.6.0](https://github.com/buger/jsonparser/releases/tag/v1.6.0). Zero open issues. Zero open PRs. Zero known issues. 123 formal requirements, zero audit findings.

"Zero known issues" does not mean "zero bugs forever." If this session proved anything, it is that mature software can hide incorrect assumptions behind green tests, full coverage, widespread production use, and confident benchmark labels.

What changed is that jsonparser now has better ways to turn the next surprise into a permanent improvement.

In 2016, the project was an experiment in whether performance alone could replace an entire infrastructure category. Could you skip the database and just read files fast enough?

Ten years later, the answer is still yes — and the experiment has expanded into something broader: how far one maintainer can take an old, widely used library — one that quietly runs inside Docker and Istio and Grafana — by combining compatibility discipline, formal requirements, structure-aware fuzzing, parallel experimentation, AI-assisted implementation, and a willingness to reopen every assumption.

Including the flattering ones.

The dashboard I originally built is long gone. But the library it required is now inside more software than I can track, proven correct to a level most projects never attempt, and faster than it has ever been.

That feels like a better anniversary than a cake.

---

*jsonparser is on [GitHub](https://github.com/buger/jsonparser). The formal verification tooling is [ReqProof](https://reqproof.com). The fuzzer is [probelabs/json-fuzz](https://github.com/probelabs/json-fuzz). The GraphQL fuzzer is [probelabs/graphql-fuzz](https://github.com/probelabs/graphql-fuzz).*
