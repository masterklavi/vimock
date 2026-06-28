# ViMock

VIMock is a WireMock-like mock server for fast local work with HTTP/gRPC/GraphQL stubs.

## Run

```bash
go run ./cmd/vimock
```

By default, the service listens on `0.0.0.0:8080`.

Configuration:

- `--host` or `VIMOCK_HOST`
- `--port` or `VIMOCK_PORT`

Health checks:

```bash
curl -i http://localhost:8080/__admin/health
curl -i http://localhost:8080/__admin/ready
```

Docker:

```bash
docker build -t vimock:dev .
docker run --rm -p 8080:8080 vimock:dev
```

## Available Functionality

- Service bootstrap with graceful shutdown.
- HTTP server with JSON stdout logging.
- CLI/env configuration: `--host`, `--port`, `VIMOCK_HOST`, `VIMOCK_PORT`.
- Health endpoint: `GET /__admin/health`.
- Readiness endpoint: `GET /__admin/ready`.
- Docker image build via `docker build -t vimock:dev .`.
- In-memory WireMock mapping storage.
- Admin API: `GET /__admin/mappings`.
- Admin API: `GET /__admin/mappings/{id}`.
- Admin API: `POST /__admin/mappings`.
- Admin API: `PUT /__admin/mappings/{id}`.
- Admin API: `DELETE /__admin/mappings/{id}`.
- Mapping fields: `id`, `name`, `persistent`, `priority`, `request`, `response`, `metadata`.
- Unknown mapping fields are preserved in Admin API responses.
- Basic HTTP stubbing for non-Admin requests.
- Request methods: `ANY`, `GET`, `POST`.
- URL matchers: `request.url`, `request.urlPath`, `request.urlPattern`.
- Priority selection with deterministic insertion-order tie-breaker.
- Response fields: `status`, `headers`, `body`, `jsonBody`.
- WireMock-like 404 response for unmatched requests.

## TODO

- Request body matchers, including JSONPath.
- Query parameter matchers.
- Header matchers.
- `equalToJson`.
- Response templating.
- Body files.
- Delays.
- Proxying via `response.proxyBaseUrl`.
- Recording and snapshotting.
- Scenario state.
- File API.
- gRPC descriptor upload API and gRPC stubbing.
- GraphQL matcher support.
- Black-box API autotests.
- Final 90% unit test coverage gate.

## Documentation

- [Docs index](docs/README.md)
- [Step 1: Service bootstrap](docs/step-01-bootstrap.md)
- [Step 2: Mapping Admin API](docs/step-02-admin-api.md)
- [Step 3: Basic HTTP stubbing](docs/step-03-basic-http-stubbing.md)

## Scope guardrails

The current implementation is incremental. It includes the service bootstrap, port configuration, stdout logging, health/readiness endpoints, Admin API CRUD for mappings, and basic HTTP stubbing.

Advanced request matchers, response templating, body files, proxying, recording, gRPC, and GraphQL are intentionally added in separate increments described in `plan.md`.
