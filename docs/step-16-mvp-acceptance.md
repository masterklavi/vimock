# Step 16: MVP Acceptance And Quality Gate

## What Was Verified

- Full unit and integration test suite for all Go packages.
- Race test suite for all Go packages.
- Stable fixture and contract tests based on `testdata`.
- HTTP Admin API, runtime mappings, matching, response rendering, proxying, delays, scenarios, recording, legacy file upload, gRPC unary runtime and GraphQL semantic matcher.
- Binary build and `--version` command.
- Coverage gate: at least 90% statement coverage.
- Current performance point after the runtime matching and response rendering optimizations.

## Run The Gate

```bash
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache go test -race ./...
GOCACHE=$(pwd)/.gocache go test -coverprofile=coverage.out ./...
GOCACHE=$(pwd)/.gocache go tool cover -func=coverage.out
GOCACHE=$(pwd)/.gocache go test -run '^$' -bench=. -benchmem ./internal/server ./internal/response
GOCACHE=$(pwd)/.gocache go build -o /tmp/vimock ./cmd/vimock
/tmp/vimock --version
```

## Current Results

- `go test ./...`: PASS.
- `go test -race ./...`: PASS.
- `go test -coverprofile=coverage.out ./...`: PASS.
- `go tool cover -func=coverage.out`: `total: (statements) 90.3%`.
- `go build -o /private/tmp/vimock-step16 ./cmd/vimock`: PASS.
- `/private/tmp/vimock-step16 --version`: `vimock dev`.

Package coverage at this point:

- `cmd/vimock`: 86.8%.
- `internal/config`: 94.4%.
- `internal/delay`: 93.9%.
- `internal/files`: 93.3%.
- `internal/grpcdesc`: 91.7%.
- `internal/mapping`: 91.4%.
- `internal/matcher`: 87.4%.
- `internal/proxy`: 93.5%.
- `internal/recording`: 94.0%.
- `internal/response`: 91.8%.
- `internal/scenario`: 94.0%.
- `internal/server`: 90.4%.
- `internal/tlsconfig`: 90.9%.

Benchmarks on `darwin/arm64`, Apple M1 Pro:

```text
BenchmarkRuntimeMatchAndRespondThousandMappings-8    5862 ns/op    1000 mappings    10837 B/op    36 allocs/op
BenchmarkRenderTemplateJSONPath-8                    3216 ns/op                    2385 B/op     42 allocs/op
```

## Acceptance Scope

The step 16 gate is in-process and repository-local. It uses Go tests and stable fixtures under `testdata`.

The separate black-box API test suite for an already running service is intentionally left for step 17. That suite must live under `autotest/`, must not import VIMock internals, and must validate the public HTTP/gRPC/GraphQL/Admin API surface.

## Added Coverage In This Step

- CLI runner tests for version, config failures, listener failures, graceful shutdown and HTTPS listener startup.
- Additional mapping parser and matcher edge cases.
- Additional Admin API success and error paths.
- Additional recording, gRPC descriptor, gRPC runtime and helper tests.
- Additional response renderer, proxy, delay, scenario, config, TLS and storage tests.

## Known Limitations After This Gate

- Full external black-box acceptance is still step 17.
- gRPC support is unary-focused; streaming, reflection, proxying and recording are not implemented yet.
- `.proto` files can be uploaded through Admin API and listed, but active schema generation currently uses descriptor sets.
- Recording supports the core start/stop/snapshot/proxy workflow, but full WireMock record spec behavior is not complete.
- GraphQL semantic matching is implemented for the supported JSON/Admin mapping form; federation-specific behavior is not implemented.
