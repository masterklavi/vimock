# Step 17: Black-Box API Autotests

## What Is Available

- Separate `autotest/` package for black-box API checks.
- Tests use public HTTP/gRPC/GraphQL/Admin API only and do not import `vimock/internal` packages.
- Safe default for release jobs: if no target is configured, network tests are skipped instead of failing.
- External target mode through `VIMOCK_BASE_URL`.
- Self-start mode through `VIMOCK_AUTOTEST_START=1`.
- Optional prebuilt binary mode through `VIMOCK_BINARY`.
- Machine-readable feature coverage report in `autotest/reports/features.json`.

## Safe Default

This command is safe in GitHub release jobs even when `vimock` is not running:

```bash
go test ./...
```

In this mode `vimock/autotest` passes because the report validation test runs and network black-box tests are skipped.

## Run Against An Existing Service

```bash
VIMOCK_BASE_URL=http://localhost:8080 go test ./autotest/...
```

## Run With Self-Started VIMock

```bash
VIMOCK_AUTOTEST_START=1 go test ./autotest/...
```

Use a prebuilt binary:

```bash
go build -o ./bin/vimock ./cmd/vimock
VIMOCK_AUTOTEST_START=1 VIMOCK_BINARY=./bin/vimock go test ./autotest/...
```

## Docker Mode

Proxy and recording tests start a local upstream server. VIMock must be able to reach that upstream.

Docker Desktop example:

```bash
VIMOCK_BASE_URL=http://localhost:8080 \
VIMOCK_AUTOTEST_UPSTREAM_HOST=host.docker.internal \
go test ./autotest/...
```

Linux Docker usually needs host networking or another upstream host reachable from the container.

## Covered Feature Groups

- Admin API health/readiness and mapping CRUD.
- Mapping lookup by `name` and `metadata.wiremock-gui.folder`.
- HTTP matching: method, `urlPath`, `urlPattern`, `urlPathPattern`, query, headers, JSONPath, absent JSONPath and `equalToJson`.
- Response generation: status, headers, `jsonBody`, `body`, response-template and body files.
- Legacy file API: `/api/login`, TUS create, TUS patch and binary `bodyFileName` playback.
- Priority and fallback mapping selection.
- Delays: `fixedDelayMilliseconds`, `delayDistribution`, `chunkedDribbleDelay`.
- Stateful scenarios and scenario reset.
- Proxying with `proxyBaseUrl` and `proxyUrlPrefixToRemove`.
- Recording start, proxy, stop, snapshot and playback from recorded mappings.
- gRPC descriptor/proto upload, list, reset, unary JSON matching, protobuf response and non-OK status mapping.
- GraphQL semantic matcher with variables, operationName and negative cases.

## Current Results

```bash
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache VIMOCK_AUTOTEST_START=1 go test -count=1 -v ./autotest
```

Latest local results:

- `go test ./...`: PASS with `vimock/autotest` safe skip behavior.
- `VIMOCK_AUTOTEST_START=1 go test -count=1 -v ./autotest`: PASS.

## Known Limitations

- The suite validates feature coverage, not every historical mapping file one by one.
- Request journal / WireMock verification API is not tested because the analyzed autotest workflow does not use it and VIMock does not implement it yet.
- gRPC streaming, reflection, proxying and recording remain known gaps.
- GraphQL federation-specific matching remains a known gap.
