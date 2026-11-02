# Known limitations and unresolved findings

This ledger separates implemented features from unresolved source findings. It is not a claim that every entry has been re-tested on every environment. Routing/lifecycle changes already completed are documented in architecture and verification; they are not listed as still missing.

## Priority: correctness and safe project generation

| Area | Current limitation | Source to inspect |
|---|---|---|
| Validation | Unknown rules and some formats can be ignored; nested traversal and pointer behavior are incomplete | `validator.go` |
| Validation tags | Comma splitting conflicts with regex quantifiers such as `{2,5}`; malformed numeric options are now rejected by the runtime validator | `validator.go`, `contract_schema.go` |
| Install/release consistency | Release/latest and generated fallback dependency can lag main; static CLI version is not a source identity | install scripts, generators/project, commands/help |
| Generator paths | Some make generators overwrite existing files and do not sufficiently constrain output names/paths | generators/make |
| Docker scaffolding | Hardcoded database credentials, host port publication, and development host/database mismatches need review | generators/docker |
| Build output | Output `app` collides with the generated `app/` directory | commands/build |
| Proxy helper | Legacy `GetRealIP` trusts forwarding headers differently from `Context.ClientIP` | `utils.go`, `proxy.go` |

These are implementation issues, not instructions to rely on the affected behavior. Avoid using unsupported validators for security decisions and inspect generated output before deployment.

## Contract and helper gaps

Numeric enum representation, pointer constraints, UTF-8 length, and required boolean semantics need runtime/OpenAPI/TypeScript consistency tests. A valid OpenAPI document alone does not prove that runtime validation implements each constraint identically.

Some CLI make/install/unknown-command failures can still exit successfully. Package installation followed immediately by tidy can remove an unused dependency. The generic Clone helper and reflect service-container lifecycle/nil handling have unresolved correctness findings. Inspect `utils.go`, `module.go`, commands/install, and the CLI entry point before extending those APIs.

## Product and verification gaps

- Rate limiting and cache are process-local; distributed coordination is not built in.
- Optional generated integrations and CRUD include application scaffolding, not complete implementations.
- The public regression suite lags local validation because tests have been excluded from commits. A stale published rate-limit expectation needs a test-only correction when publication is authorized.
- Fresh release/tag publication and end-to-end installation verification remain separate release work.
- Performance superiority is workload-specific. Small-route routing and some parallel measurements have shown remaining gaps. See performance results rather than assuming `FastRouter` guarantees lower latency.
- No real-network p95/p99, sustained production load, or peak-RSS claim follows from in-process ServeHTTP microbenchmarks.

Reproduce a ledger entry before fixing it. Add an explicit regression, update this document when the issue is resolved, and keep the evidence scope clear.
