# Performance target: at least 1.5 times Gin throughput

## Acceptance contract

For the same validated work, define the observed speedup as:

```text
speedup = median(Gin ns/op) / median(Routix ns/op)
pass = speedup >= 1.5
```

At this threshold, Routix uses at most two thirds of Gin's ns/op. Every required cell must pass. A favorable average or a large-route win cannot compensate for a failing wildcard or small-route cell.

The fixed matrix has **48 cells**:

| Dimension | Required values |
|---|---|
| Workload | Noop204, StaticText, PathParameter, Wildcard, JSONStdlib, JSONNative |
| Registered route count | 1, 10, 100, 1000 |
| Execution mode | BenchmarkHTTP with CPU=1; BenchmarkHTTPParallel with CPU=8 |
| Minimum observations | Six per framework per cell, including both framework orderings |
| Compared frameworks | Routix and Gin |

The runtime, toolchain, payload, route distribution, status, headers, body, error behavior, and middleware semantics must remain equivalent. API compatibility, arbitrary public Params mutations, timeout isolation, and post-request cleanup are mandatory. Cached responses, skipped handlers/middleware, unvalidated wire changes, retained request values, or an API that silently stops materializing public maps cannot be counted as feature-preserving speedups.

## Target checker

The repository includes a Python 3 standard-library analysis tool:

```sh
python3 tools/benchmark-target.py \
    matrix-serial-forward.txt matrix-serial-reverse.txt \
    matrix-parallel-forward.txt matrix-parallel-reverse.txt \
    --speedup 1.5 --min-samples 6 > target-report.json
```

The tool reads Go benchmark text and emits medians, ranges, sample counts, per-cell ratios, and the complete matrix verdict as JSON. Exit 0 means all 48 cells meet the configured observed-median threshold; exit 1 means a missing cell, insufficient observations, or a ratio below the target; exit 2 means invalid input or options. Duplicate input paths are rejected to prevent accidentally counting one run twice. Explicit Go test/benchmark failure markers also invalidate the input.

Sequential CPU=8 results do not fill a parallel CPU=8 cell. Output from a different benchmark or a different route count cannot fill a required cell. Other frameworks and RoutixBefore may appear in the same input files; they are not substituted for the requested Routix/Gin pair.

This is an acceptance check on supplied measurements, not a statistical significance test. It cannot establish semantic equivalence or authenticate benchmark provenance by parsing timings. Keep source hashes, pinned dependencies, correctness logs, and the unedited raw results with each report.

The comparative harness remains a separate local fixture under the test-file exclusion policy. The checker does not run that harness, and a clean clone still does not contain the full comparative test suite. See [performance methodology](performance.md) and [verification](verification.md).

## Current outcome

The frozen-source run on 2026-11-02 used Go 1.26.10, darwin/arm64, Apple M2, Gin v1.12.0, and the production Go tree of main commit `e36988b6689787d683e2720589a0f061e94c80ed`. Six observations per framework per cell were collected: three forward and three reverse. Serial observations were 500ms each; parallel observations were 200ms each. Scheduling, thermal state, frequency, and other applications were uncontrolled. CPU settings are GOMAXPROCS values, not OS affinity guarantees.

**Observed result: 2/48 cells pass; the complete target is not met.** All required cells and repetitions are present. These figures are observational medians, not statistical proof or a production guarantee.

| Mode | Workload | Routes | Routix ns/op | Gin ns/op | Observed throughput ratio | Target |
|---|---|---:|---:|---:|---:|---|
| HTTP CPU=1 | Noop204 | 1 | 35.32 | 29.95 | 0.848x | below_target |
| HTTP CPU=1 | Noop204 | 10 | 35.58 | 38.67 | 1.087x | below_target |
| HTTP CPU=1 | Noop204 | 100 | 38.03 | 55.45 | 1.458x | below_target |
| HTTP CPU=1 | Noop204 | 1000 | 50.09 | 74.50 | 1.487x | below_target |
| HTTP CPU=1 | StaticText | 1 | 133.60 | 130.95 | 0.980x | below_target |
| HTTP CPU=1 | StaticText | 10 | 130.30 | 138.40 | 1.062x | below_target |
| HTTP CPU=1 | StaticText | 100 | 132.35 | 145.95 | 1.103x | below_target |
| HTTP CPU=1 | StaticText | 1000 | 145.70 | 165.30 | 1.135x | below_target |
| HTTP CPU=1 | PathParameter | 1 | 183.00 | 147.85 | 0.808x | below_target |
| HTTP CPU=1 | PathParameter | 10 | 192.25 | 163.40 | 0.850x | below_target |
| HTTP CPU=1 | PathParameter | 100 | 189.85 | 178.45 | 0.940x | below_target |
| HTTP CPU=1 | PathParameter | 1000 | 187.25 | 196.35 | 1.049x | below_target |
| HTTP CPU=1 | Wildcard | 1 | 189.25 | 155.85 | 0.824x | below_target |
| HTTP CPU=1 | Wildcard | 10 | 193.60 | 169.15 | 0.874x | below_target |
| HTTP CPU=1 | Wildcard | 100 | 196.05 | 188.90 | 0.964x | below_target |
| HTTP CPU=1 | Wildcard | 1000 | 203.30 | 202.75 | 0.997x | below_target |
| HTTP CPU=1 | JSONStdlib | 1 | 215.45 | 214.80 | 0.997x | below_target |
| HTTP CPU=1 | JSONStdlib | 10 | 214.95 | 227.75 | 1.060x | below_target |
| HTTP CPU=1 | JSONStdlib | 100 | 223.50 | 245.30 | 1.098x | below_target |
| HTTP CPU=1 | JSONStdlib | 1000 | 220.10 | 256.95 | 1.167x | below_target |
| HTTP CPU=1 | JSONNative | 1 | 221.55 | 268.05 | 1.210x | below_target |
| HTTP CPU=1 | JSONNative | 10 | 250.80 | 258.55 | 1.031x | below_target |
| HTTP CPU=1 | JSONNative | 100 | 250.65 | 268.15 | 1.070x | below_target |
| HTTP CPU=1 | JSONNative | 1000 | 276.50 | 324.55 | 1.174x | below_target |
| HTTPParallel CPU=8 | Noop204 | 1 | 11.06 | 12.71 | 1.148x | below_target |
| HTTPParallel CPU=8 | Noop204 | 10 | 11.02 | 15.65 | 1.421x | below_target |
| HTTPParallel CPU=8 | Noop204 | 100 | 13.52 | 22.67 | 1.677x | pass |
| HTTPParallel CPU=8 | Noop204 | 1000 | 14.47 | 22.25 | 1.537x | pass |
| HTTPParallel CPU=8 | StaticText | 1 | 45.81 | 46.88 | 1.023x | below_target |
| HTTPParallel CPU=8 | StaticText | 10 | 45.91 | 44.83 | 0.976x | below_target |
| HTTPParallel CPU=8 | StaticText | 100 | 49.03 | 51.52 | 1.051x | below_target |
| HTTPParallel CPU=8 | StaticText | 1000 | 55.31 | 52.87 | 0.956x | below_target |
| HTTPParallel CPU=8 | PathParameter | 1 | 72.06 | 63.77 | 0.885x | below_target |
| HTTPParallel CPU=8 | PathParameter | 10 | 77.08 | 63.05 | 0.818x | below_target |
| HTTPParallel CPU=8 | PathParameter | 100 | 75.11 | 74.78 | 0.996x | below_target |
| HTTPParallel CPU=8 | PathParameter | 1000 | 64.75 | 71.59 | 1.106x | below_target |
| HTTPParallel CPU=8 | Wildcard | 1 | 75.38 | 63.28 | 0.839x | below_target |
| HTTPParallel CPU=8 | Wildcard | 10 | 71.75 | 67.56 | 0.942x | below_target |
| HTTPParallel CPU=8 | Wildcard | 100 | 75.79 | 73.80 | 0.974x | below_target |
| HTTPParallel CPU=8 | Wildcard | 1000 | 69.48 | 73.88 | 1.063x | below_target |
| HTTPParallel CPU=8 | JSONStdlib | 1 | 79.81 | 73.77 | 0.924x | below_target |
| HTTPParallel CPU=8 | JSONStdlib | 10 | 74.54 | 74.40 | 0.998x | below_target |
| HTTPParallel CPU=8 | JSONStdlib | 100 | 82.58 | 87.44 | 1.059x | below_target |
| HTTPParallel CPU=8 | JSONStdlib | 1000 | 69.69 | 82.45 | 1.183x | below_target |
| HTTPParallel CPU=8 | JSONNative | 1 | 79.71 | 90.10 | 1.130x | below_target |
| HTTPParallel CPU=8 | JSONNative | 10 | 79.47 | 86.21 | 1.085x | below_target |
| HTTPParallel CPU=8 | JSONNative | 100 | 76.72 | 95.77 | 1.248x | below_target |
| HTTPParallel CPU=8 | JSONNative | 1000 | 79.02 | 89.56 | 1.133x | below_target |

## Cost investigation and rejected experiments

The investigation compared the existing router against an adaptive single-context cache, immutable matched-node metadata, known-key deletion during parameter-map cleanup, and whole-context reset. Targeted race/lifecycle tests passed, but the gains did not hold across workloads; some candidates regressed other cells. Those runtime changes were rejected and were not merged.

A direct common-response diagnostic measured about 92 ns for the parameter response alone and about 123 ns for filling, reading, and clearing a public parameter map plus writing the response. Those reference runs omit routing, context acquisition, and framework error/middleware machinery. They indicate that map and response work are significant parts of the budget; they are not formal lower bounds or proof that another implementation cannot improve them.

This is why reducing trie lookup alone is insufficient evidence for the full target. Public map behavior must stay observable. A new accessor representation that only wins by silently making the public Params map nil would violate the current API contract. JSONStdlib also contains shared application encoding work that cannot be attributed entirely to the router.

## Next optimization requirements

Use CPU/allocation profiles to choose one concrete change per experiment. Keep benchmark inputs and the target checker fixed. Accept an implementation only after API/behavioral tests and the complete matrix show a reproducible improvement; preserve regressions and rejected raw outputs in the experiment ledger. Native JSON and shared-stdlib JSON need separate conclusions. Optimize application-owned work only with equivalent wire behavior and equal comparison conditions.

Do not use the target as a production latency guarantee. Route shapes, payload sizes, network/TLS, logging, auth, tracing, GC settings, CPU scheduling, and user handler work change the result. A microbenchmark result cannot establish a universal fixed ratio for arbitrary applications.
