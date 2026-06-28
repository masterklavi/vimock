# Testing VIMock

## Repository Tests

```bash
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

The black-box autotest package is safe in normal `go test ./...` runs. If no target is configured, network black-box tests are skipped.

## Black-Box API Autotests

Run against an already running VIMock:

```bash
VIMOCK_BASE_URL=http://localhost:8080 go test ./autotest/...
```

Let tests build and start a temporary VIMock process:

```bash
VIMOCK_AUTOTEST_START=1 go test ./autotest/...
```

Use an already built binary:

```bash
go build -o ./bin/vimock ./cmd/vimock
VIMOCK_AUTOTEST_START=1 VIMOCK_BINARY=./bin/vimock go test ./autotest/...
```

For Docker Desktop proxy and recording tests:

```bash
VIMOCK_BASE_URL=http://localhost:8080 \
VIMOCK_AUTOTEST_UPSTREAM_HOST=host.docker.internal \
go test ./autotest/...
```

Feature coverage is stored in:

```text
autotest/reports/features.json
```
