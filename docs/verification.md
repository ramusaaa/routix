# Verification and test availability

## Baseline commands

Run from the repository root using a compatible Go toolchain:

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

The module declares Go 1.25.0; the recent verification toolchain is Go 1.26.10. Record `go version`, `go env`, and the revision for every report. Do not silently claim support based only on the module directive.

## Published checkout versus local suite

Previous work intentionally excluded test and Markdown files from commits. The published checkout therefore does not contain all regressions used during development. The original rate-limit test expects 400 after the quota is exceeded, while the implementation correctly returns 429. A clean checkout's `go test ./...` can fail on that stale expectation. The local copy uses the corrected expectation.

Do not change the production status to satisfy a stale test. Do not report a clean-checkout pass based on an uncommitted local suite. Publishing the corrected and expanded test suite requires changing the current commit exclusion policy.

Locally available suites cover routing, deadline/body/timeout ownership, cache identity, JSON errors, proxy trust, contracts, telemetry, hot reload, route inspection, and generated auth/migrations. Confirm that a file exists before attempting the commands below; these tests may be absent in a clone.

## Routing and concurrency

Useful local test families include `TestFoundation`, `TestDispatch`, `TestSingleCPU`, `TestFlatPrefix`, `TestPrivateState`, `TestAllocation`, `TestRegression`, and `TestFollowup`.

```sh
go test -race -run 'Test(Foundation|Dispatch|SingleCPU|FlatPrefix|PrivateState|Allocation|Regression|Followup)' -count=3 .
```

Route tests use an independent linear reference matcher, rather than copying the optimization. Cover registration order and replacement, shared parameter shapes, static/parameter/wildcard fallback, empty wildcard, slash variants, Unicode and arbitrary bytes, HEAD/405, snapshot invalidation, concurrent registrations, and cross-request data isolation.

Race checks must include concurrent registration/serving, proxy changes, pooling configuration, shared middleware state, and late timeout workers. They are not replaced by a fast benchmark.

## Fuzzing

Run one fuzz target per command when the local fixture exists:

```sh
go test -run '^$' -fuzz '^FuzzFoundationRoutingReference$' -fuzztime=60s -parallel=2 .
go test -run '^$' -fuzz '^FuzzDispatchOptimizedReference$' -fuzztime=60s -parallel=2 .
go test -run '^$' -fuzz '^FuzzSingleCPUPrefixReference$' -fuzztime=60s -parallel=2 .
go test -run '^$' -fuzz '^FuzzFlatPrefixDenseReference$' -fuzztime=60s -parallel=2 .
```

Save failing corpus inputs and reproduce them as deterministic regressions. Record execution counts and elapsed time; neither establishes exhaustive correctness.

## Middleware and HTTP lifetime

Use real HTTP connections for shutdown, response-controller, slow upload, nested timeout, and keep-alive behavior. httptest alone cannot establish that socket read deadlines work. Verify both successful completion and late/error/panic paths, and check that compressed responses close during unwinding.

Test private-cache exclusions for authorization, cookies, mTLS client identity, request cache policy, response cache policy, host, and Vary. Compare actual status/header/body, not only handler invocation counts.

## Generated projects

The local command tests create isolated application modules, point them at the checkout under test, build/run the emitted code, and test SQLite auth/migrations. PostgreSQL and MySQL tests require configured live servers and must be reported as skipped when those are absent. The generated-project fixture selects a live backend using `ROUTIX_TEST_DRIVER` and `ROUTIX_TEST_DSN`. Do not infer successful cross-database coverage from a SQLite run.

Test auth registration/login/revocation/refresh and concurrent refresh. Test migrations apply/rollback/history/checksums/dirty repair/concurrent locking and driver-specific DDL behavior. Run CLI build/hot reload/route inspection against emitted applications, not only mocked Go subprocesses.

## Public API compatibility

Compare exported declarations and method sets for the root and telemetry packages, including exported fields, constants, interfaces, signatures, and generic constraints. The last completed comparison before this work matched 794 exported API records. That count is historical evidence, not an automated gate stored in the published repository.

API compatibility also needs behavioral checks: mutable/replaced public maps, cleanup after return, response wire format, route metadata, error propagation, and request ownership. An unchanged symbol list cannot prove those behaviors.

## Follow-up regressions

`review_followup_local_test.go` adds explicit tests for same-shape replacement after snapshot publication, replacement parameter/middleware/route metadata, HEAD/Allow behavior, plain and typed errors under single/nested timeout, completed-response status preservation, inherited header addition/deletion, late-worker header isolation, malformed numeric constraints, and concurrent validator-cache use.

The inherited-header and numeric-configuration tests fail against the preceding main sources and pass after the fixes. Error-without-response timeout status already passes against those preceding sources; keep that regression rather than treating an unconfirmed risk as a newly fixed bug.

## Documentation verification

Compile complete Go examples in isolated modules with a replace pointing to the checkout. Type-check contextual snippets with their documented surrounding declarations. Check relative links and source paths from a clean portable copy. CLI examples must match argument parsing, not only help text.

Documentation links must not depend on a maintainer's Desktop or temporary artifact directories. If benchmark fixtures remain local-only, state that limitation and include enough methodology and results for review.

## Gin target analysis tool

`tools/benchmark-target.py` checks the fixed 48-cell target described in [the acceptance contract](performance-target.md). Its local regression file is `tools/benchmark_target_local_test.py`; it remains excluded from commits under the current policy. Twelve checks cover the exact 1.5 throughput boundary, a single failing/missing cell, insufficient samples, wrong CPU/mode, invalid/nonfinite timings and options, duplicate input paths, explicit failed runs, scientific notation, and combining forward/reverse inputs. This tool checks measured data; it does not replace adapter correctness, race, API, or provenance checks.
