# MVP Compliance Report

Date: 2026-06-28
Service: VIMock (`vimock`)

## Summary

Step 16 quality gate is passed for the current in-repository MVP implementation:

- Unit/integration suite: PASS.
- Race suite: PASS.
- Statement coverage: PASS, `90.3%`.
- Binary build and `--version`: PASS.
- Stable fixture/contract tests from `testdata`: PASS.
- Performance smoke benchmarks: PASS.

Overall product compliance status: PARTIAL.

Reason: the implemented service covers the main HTTP/WireMock-compatible runtime, Admin API, matching, response rendering, proxying, delays, scenarios, recording basics, legacy file upload, unary gRPC runtime and GraphQL semantic matching. Some broader `tz.md` requirements remain intentionally open or partial, mostly around full WireMock recording spec parity, `.proto` compilation, advanced gRPC behavior and the separate black-box API suite planned for step 17.

## Quality Gate Evidence

```bash
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache go test -race ./...
GOCACHE=$(pwd)/.gocache go test -coverprofile=coverage.out ./...
GOCACHE=$(pwd)/.gocache go tool cover -func=coverage.out
GOCACHE=$(pwd)/.gocache go test -run '^$' -bench=. -benchmem ./internal/server ./internal/response
GOCACHE=$(pwd)/.gocache go build -o /tmp/vimock ./cmd/vimock
/tmp/vimock --version
```

Latest results:

- `go test ./...`: PASS.
- `go test -race ./...`: PASS.
- Coverage: `total: (statements) 90.3%`.
- `BenchmarkRuntimeMatchAndRespondThousandMappings`: `5862 ns/op`, `10837 B/op`, `36 allocs/op`.
- `BenchmarkRenderTemplateJSONPath`: `3216 ns/op`, `2385 B/op`, `42 allocs/op`.
- `--version`: `vimock dev`.

## Requirement Matrix

| Requirement group | Status | Evidence / Notes |
|---|---|---|
| `CON-001..CON-008` | PASS | Runtime state is in-memory; logs go to stdout; binary, Docker and CI/Kubernetes-oriented modes are documented; unknown mapping fields are preserved. |
| `PROTO-001..PROTO-005` | PASS / PARTIAL | HTTP/1.1, HTTPS, HTTP/2, unary gRPC over HTTP/2 and GraphQL-over-HTTP are implemented. Advanced gRPC streaming remains open. |
| `MAP-001..MAP-009` | PASS | WireMock mapping fields, generated IDs, metadata, methods, URL matchers, priority and fallback proxy ordering are covered by tests. |
| `MATCH-001..MATCH-012` | PASS / PARTIAL | JSONPath, absent checks, filters, array size, query/header matchers and `equalToJson` are implemented for current fixtures. Full WireMock/JSONUnit parity remains a future expansion. |
| `RESP-001..RESP-012` | PASS / PARTIAL | Status, headers, `jsonBody`, `body`, `bodyFileName`, binary bodies, targeted response-template and delays are implemented. Full Handlebars/WireMock templating is not complete. |
| `RESP-013..RESP-014` | PASS | Fault simulation, webhooks and `postServeActions` stay out of MVP scope. |
| `PROXY-001..PROXY-003` | PASS | Proxy mappings, prefix removal and lower-priority fallback behavior are implemented and tested. |
| `REC-001..REC-011` | PARTIAL | Start/stop/snapshot, proxy recording and in-memory recorded mappings work. Full record spec behavior and body-file extraction are incomplete. |
| `SCN-001..SCN-006` | PASS | Stateful scenarios and reset are implemented with concurrency tests. |
| `JRPC-001..JRPC-003` | PASS | JSON-RPC over HTTP is covered through JSONPath request matching and request-id response templating. |
| `GRPC-001..GRPC-018` | PARTIAL | Admin descriptor API, legacy `.dsc` upload, reset, unary protobuf JSON matching, templated responses and status mapping work. `.proto` upload is stored/listed but not compiled into the active registry. Streaming, reflection, gRPC proxying and recording are open. |
| `GQL-001..GQL-011` | PASS / PARTIAL | GraphQL semantic matcher, variables, operationName and response pipeline are implemented for the supported JSON/Admin mapping form. Federation-specific behavior and any non-JSON client DSL concerns are open. |
| `ADM-001..ADM-015` | PASS | Mapping Admin API CRUD, validation, immediate activation/deactivation and concurrent access are tested. |
| `FILE-001..FILE-010` | PASS | In-memory body file storage and legacy `/api/login` + TUS-like upload workflow are implemented. |
| `FILE-011` | OPEN | Native body-file upload Admin API is still a SHOULD requirement. |
| `RT-001..RT-005` | PASS | Runtime-generated mapping create/use/delete and PDM reset workflow are covered by contract tests. |
| `NFR-001..NFR-006` | PASS | Race tests pass; matching uses immutable snapshots and indexed candidates; runtime avoids unnecessary mapping/response copies. |
| `TEST-001` | PASS | Statement coverage is `90.3%`. |
| `TEST-002` | PASS | Tests cover matcher engine, priority, templating, Admin API, File API, gRPC descriptor registry and GraphQL matcher. |
| `TEST-003..TEST-004` | PASS / PARTIAL | Stable `testdata` fixture and contract tests are present. Full external black-box API suite is step 17. Temporary source folders are not referenced. |
| `TEST-005` | PASS | `go test -race ./...` passes. |
| `TEST-006` | PASS | Benchmark tests exist for matching and response rendering. |
| `OUT-001..OUT-005` | PASS | Out-of-scope features remain unimplemented by design. |
| `ACC-001..ACC-010` | PASS / PARTIAL | In-repository acceptance passes. Black-box API acceptance, full gRPC docs compatibility and full recording spec parity remain open. |

## Open Or Partial Requirement IDs

- `REC-004`: fields such as `filters`, `transformers` and `transformerParameters` are not behaviorally applied yet.
- `REC-009`: snapshot spec has the same advanced-field limitation as recording start.
- `REC-011`: binary bodies are recorded as `base64Body`; extraction into body files via `extractBodyCriteria` is not implemented.
- `GRPC-001`: unary mapping syntax is supported, but full docs-level behavior is not complete.
- `GRPC-012`: `.proto` files can be uploaded and listed, but they are not compiled into active runtime descriptors.
- `GQL-001`: core JSON/Admin mapping behavior is supported; non-runtime client DSL parity is not directly represented by the service API.
- `GQL-007..GQL-010`: JSON custom matcher compatibility is implemented for `graphql-body-matcher`; exact client DSL syntax should be verified in step 17 black-box tests.
- `FILE-011`: native body-file Admin API is not implemented.
- `TEST-003..TEST-004`: covered by stable in-process fixtures/contracts, but not yet by a separate black-box suite.
- `ACC-007..ACC-008`: covered by in-process gRPC/GraphQL contract tests; black-box API confirmation remains step 17.

## Decision Notes

- The old temporary source fixture directories are not referenced by tests or docs. Stable examples live under `testdata`.
- Step 16 does not introduce new functional features beyond acceptance blockers and coverage; it mainly hardens tests and runner seams.
- Step 17 should add public API black-box tests without importing VIMock internal packages.
