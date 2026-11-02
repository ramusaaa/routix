# Performance methodology and results

Routix performance is workload-specific. Public map mutability, request-state cleanup, middleware ordering, and HTTP output are correctness requirements, not optional benchmark modes. `NewFastRouter` uses the same router and does not guarantee faster results than every competing framework.

## Comparable workloads

The comparison uses bare routers through standard `net/http.Handler.ServeHTTP`, with no network, TLS, logging, auth, compression, or instrumentation. All frameworks register the same number of routes and visit every route round-robin. Route registration, snapshot warmup, and request/writer preparation are outside the timer. Every route's status, headers, and body are checked before timing, and status is checked per iteration.

| Workload | Pattern/payload | Shared behavior |
|---|---|---|
| Noop204 | `/rNNNN/noop` | 204 without body |
| StaticText | `/rNNNN/static` | text/plain and fixed `ok` bytes |
| PathParameter | `/rNNNN/users/:id` | matched parameter returned as text |
| Wildcard | `/rNNNN/files/*` | matched suffix returned as text |
| JSONStdlib | `/rNNNN/json` | same encoding/json encoder and payload |
| JSONNative | `/rNNNN/json` | each framework's JSON helper; codec/wire policies may differ |

Gin uses its named wildcard syntax; Chi uses `{id}` for parameters. Wildcard outputs are normalized equivalently before being written. The capture writer and request set are preallocated consistently. Report benchmark-side allocations as part of B/op and allocs/op, not as zero-allocation routing merely because setup is excluded.

CPU=1 tests use Go's `-cpu=1`; they are not an operating-system affinity guarantee. CPU=8 uses RunParallel, whose ns/op is aggregate throughput normalization, not single-request latency. Shared-host thermal state, frequency, and other application activity are uncontrolled.

## Latest measurement

Date: 2026-11-02. Baseline production tree: `9624918188589e9a463a9f47e4a04c635e2df1d4`. Go 1.26.10, darwin/arm64, Apple M2. Gin v1.12.0, Echo v5.4.4, Chi v5.5.0. Before is the preceding main implementation; after also includes context preparation/layout changes, strict numeric validator configuration, and timeout header propagation fixes.

Routing cells are **eight observations' medians in ns/op**: four 500ms repetitions in each framework ordering. Lower is better. Controls use three 200ms repetitions; parallel cells use four 300ms repetitions. These are in-process measurements on an uncontrolled shared host, not production latency or proof of universal superiority.

| Routes | Workload | Routix before | Routix after | Gin | Echo | Chi |
|---|---|---:|---:|---:|---:|---:|
| 1 | PathParameter | 177.15 | 173.50 | 144.25 | 147.90 | 439.45 |
| 1 | Wildcard | 178.05 | 176.40 | 146.10 | 148.80 | 435.50 |
| 10 | PathParameter | 181.70 | 182.15 | 155.45 | 164.15 | 470.10 |
| 10 | Wildcard | 184.95 | 182.70 | 163.95 | 170.15 | 492.75 |
| 100 | PathParameter | 188.10 | 186.50 | 170.10 | 181.60 | 512.05 |
| 100 | Wildcard | 184.65 | 184.75 | 179.75 | 180.55 | 510.70 |
| 1000 | PathParameter | 172.25 | 172.15 | 185.30 | 197.45 | 608.00 |
| 1000 | Wildcard | 190.60 | 192.75 | 188.20 | 197.55 | 613.35 |

The median is lower than both Gin and Echo in 1 of the eight routing cells. This is a per-cell observation, not a statistical significance test.

### Other workloads, CPU=1, 1000 routes

| Workload | Routix before | Routix after | Gin | Echo | Chi |
|---|---:|---:|---:|---:|---:|
| Noop204 | 48.32 | 48.20 | 66.60 | 65.02 | 314.00 |
| StaticText | 203.30 | 165.80 | 162.90 | 293.90 | 598.20 |
| JSONStdlib | 235.60 | 243.20 | 260.10 | 246.40 | 513.00 |
| JSONNative | 258.50 | 253.70 | 278.10 | 308.20 | 523.10 |

### Parallel control, CPU=8, 1000 routes

| Workload | Routix before | Routix after | Gin | Echo | Chi |
|---|---:|---:|---:|---:|---:|
| Noop204 | 10.41 | 10.15 | 18.55 | 23.35 | 146.35 |
| StaticText | 28.60 | 38.62 | 39.94 | 44.12 | 144.60 |
| PathParameter | 49.45 | 47.18 | 59.20 | 55.61 | 204.00 |
| Wildcard | 45.12 | 53.77 | 52.72 | 65.05 | 206.25 |
| JSONStdlib | 52.73 | 51.59 | 61.83 | 62.06 | 149.05 |
| JSONNative | 51.11 | 53.06 | 71.72 | 65.62 | 151.15 |

### Allocations, CPU=1, 1000 routes

| Workload | Framework | B/op | allocs/op |
|---|---|---:|---:|
| PathParameter | RoutixBefore | 24 | 2 |
| PathParameter | Routix | 24 | 2 |
| PathParameter | Gin | 24 | 2 |
| PathParameter | Echo | 24 | 2 |
| PathParameter | Chi | 728 | 6 |
| Wildcard | RoutixBefore | 32 | 2 |
| Wildcard | Routix | 32 | 2 |
| Wildcard | Gin | 32 | 2 |
| Wildcard | Echo | 32 | 2 |
| Wildcard | Chi | 736 | 6 |

Raw ranges and all 150 summary rows are preserved in the measurement artifact. The portable fixture contains raw outputs and pinned sources; it is not currently a committed repository benchmark suite.

### Follow-up of variable parallel results

The primary parallel series showed regressions in StaticText and Wildcard. A separate six-by-700ms CPU=8 paired check was run without source edits; it does not replace or erase the primary series. StaticText medians were 28.82 ns before and 30.45 ns after; Wildcard medians were 44.39 ns before and 43.665 ns after. Ranges overlapped. These follow-up results show variability, not a general latency improvement claim.

On this arm64 build, reflect-reported Context field size is 248 bytes before and 240 bytes after. This is type size, not heap allocation-class size or resident-memory savings. Per-request allocations above did not decrease.

## Reproduction

The comparative harness is a separate development fixture, not part of the published test suite under the current test-file exclusion policy. A clean clone does not currently contain an executable Gin/Echo/Chi comparison. Do not imply that `go test -bench .` alone reproduces the tables.

The measurement artifact includes the harness, pinned go.mod/go.sum, before/after sources, raw outputs, source hashes, and runner. In that portable fixture the key commands are:

```sh
go test -run '^$' -bench '^BenchmarkHTTP$/(PathParameter|Wildcard)$' -benchmem -benchtime=500ms -count=4 -cpu=1
BENCH_FRAMEWORK_ORDER=Chi,Echo,Gin,Routix,RoutixBefore go test -run '^$' -bench '^BenchmarkHTTP$/(PathParameter|Wildcard)$' -benchmem -benchtime=500ms -count=4 -cpu=1
go test -run '^$' -bench '^BenchmarkHTTPParallel$/./routes=1000$' -benchmem -benchtime=300ms -count=4 -cpu=8
```

Both orderings contribute to the final median. Before/after versions are loaded in the same executable; only the old module's path is renamed to allow both versions to compile. Do not replace a disappointing final median with a favorable candidate run.

## Interpreting results

- A small median lead with overlapping ranges is not established statistical superiority.
- Keep small-route and large-route results separate. A dense prefix index can help large suitable route sets without helping a one-route application.
- Keep native JSON separate from a shared stdlib encoder comparison.
- Bare Router results do not measure `NewAPI`, which installs shared process-metrics middleware.
- Fast dispatch can trade snapshot preparation time and memory for request throughput. Include cold-start/refresh measurements when changing index construction.
- In-process ns/op cannot be labeled real-network RPS, p95/p99 latency, or production capacity.

For a production comparison, add representative middleware, body sizes, parameter shapes, invalid routes, request distributions, a real HTTP server/client, sustained concurrency, and latency/error percentiles. Preserve matched response semantics and publish the full matrix.

## Fixed Gin target

The new target is at least 1.5 times Gin throughput in every required cell, with API and behavioral compatibility preserved. [The target contract and checker](performance-target.md) require all 48 cells; no aggregate or selected winner substitutes for that verdict. This target is not yet met.
