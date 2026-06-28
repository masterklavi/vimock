# VIMock Black-Box Autotests

This package tests VIMock through its public HTTP/gRPC/GraphQL/Admin API only. It does not import `vimock/internal` packages.

## Safe Default

A plain repository test run is safe:

```bash
go test ./...
```

If neither `VIMOCK_BASE_URL` nor `VIMOCK_AUTOTEST_START=1` is set, network black-box tests are skipped. This prevents GitHub release jobs from failing just because a VIMock process is not running.

## Run Against An Existing Service

```bash
VIMOCK_BASE_URL=http://localhost:8080 go test ./autotest/...
```

## Run With Self-Started Binary

Build automatically from source:

```bash
VIMOCK_AUTOTEST_START=1 go test ./autotest/...
```

Use an already built binary:

```bash
go build -o ./bin/vimock ./cmd/vimock
VIMOCK_AUTOTEST_START=1 VIMOCK_BINARY=./bin/vimock go test ./autotest/...
```

## Docker Notes

For proxy and recording tests, VIMock must be able to reach the local upstream server started by the test process.

With Docker Desktop, expose the upstream host explicitly:

```bash
VIMOCK_BASE_URL=http://localhost:8080 \
VIMOCK_AUTOTEST_UPSTREAM_HOST=host.docker.internal \
go test ./autotest/...
```

On Linux Docker, run the container with host networking or provide another upstream host that is reachable from the container.

## Feature Report

Machine-readable coverage is stored in:

```bash
autotest/reports/features.json
```
