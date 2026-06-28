# Step 1: Service Bootstrap

## What Is Available

- Go module: `vimock`.
- Entrypoint: `cmd/vimock`.
- HTTP server.
- Graceful shutdown on `SIGINT` and `SIGTERM`.
- JSON logs to stdout.
- Health endpoint: `GET /__admin/health`.
- Readiness endpoint: `GET /__admin/ready`.
- CLI config: `--host`, `--port`.
- Env config: `VIMOCK_HOST`, `VIMOCK_PORT`.
- Docker build through `Dockerfile`.

## Run Locally

```bash
go run ./cmd/vimock
```

```bash
go run ./cmd/vimock --host 127.0.0.1 --port 8080
```

```bash
VIMOCK_PORT=8081 go run ./cmd/vimock
```

## Check

```bash
curl -i http://localhost:8080/__admin/health
curl -i http://localhost:8080/__admin/ready
```

Expected result:

- `GET /__admin/health` returns HTTP 200.
- `GET /__admin/ready` returns HTTP 200.

## Docker

```bash
docker build -t vimock:dev .
docker run --rm -p 8080:8080 vimock:dev
```

## Tests

```bash
go test ./...
```

## TODO

- Mapping Admin API was not part of step 1.
- Runtime request matching was not part of step 1.
- Proxying, recording, gRPC and GraphQL were not part of step 1.
