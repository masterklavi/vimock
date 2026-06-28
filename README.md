# ViMock

VIMock is a WireMock-like mock server for fast local work with HTTP/gRPC/GraphQL stubs.

## Run

```bash
go run ./cmd/vimock
```

By default, the service listens on `0.0.0.0:8080`.

Version:

```bash
vimock --version
```

Configuration:

- `--host` or `VIMOCK_HOST`
- `--port` or `VIMOCK_PORT`
- `--https-port` or `VIMOCK_HTTPS_PORT`
- `--tls-cert-file` or `VIMOCK_TLS_CERT_FILE`
- `--tls-key-file` or `VIMOCK_TLS_KEY_FILE`
- `--tls-self-signed` or `VIMOCK_TLS_SELF_SIGNED`

HTTPS with an in-memory self-signed certificate:

```bash
go run ./cmd/vimock --https-port 8443 --tls-self-signed
curl -k --http2 https://localhost:8443/__admin/health
```

Health checks:

```bash
curl -i http://localhost:8080/__admin/health
curl -i http://localhost:8080/__admin/ready
```

Docker:

```bash
docker build -t vimock:dev .
docker run --rm -p 8080:8080 vimock:dev
docker run --rm -p 8080:8080 -p 8443:8443 vimock:dev --https-port 8443 --tls-self-signed
```

Release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Tag pushes create GitHub Release assets for Linux and macOS on `amd64` and `arm64`.

## Available Functionality

- Service bootstrap with graceful shutdown.
- HTTP server with JSON stdout logging.
- CLI/env configuration: `--host`, `--port`, `--https-port`, `--tls-cert-file`, `--tls-key-file`, `--tls-self-signed`, `VIMOCK_HOST`, `VIMOCK_PORT`, `VIMOCK_HTTPS_PORT`, `VIMOCK_TLS_CERT_FILE`, `VIMOCK_TLS_KEY_FILE`, `VIMOCK_TLS_SELF_SIGNED`.
- Health endpoint: `GET /__admin/health`.
- Readiness endpoint: `GET /__admin/ready`.
- Docker image build via `docker build -t vimock:dev .`.
- Docker runtime image uses Alpine, CA certificates, non-root user and a container healthcheck.
- HTTP/1.1, HTTP/2, unencrypted HTTP/2 server transport base and HTTP/2 over TLS.
- HTTPS listener with file-based certificates or generated in-memory self-signed certificates.
- In-memory WireMock mapping storage.
- Admin API: `GET /__admin/mappings`.
- Admin API: `GET /__admin/mappings/{id}`.
- Admin API: `POST /__admin/mappings`.
- Admin API: `PUT /__admin/mappings/{id}`.
- Admin API: `DELETE /__admin/mappings/{id}`.
- Runtime-generated mapping lifecycle: create, use immediately, delete, tolerate repeated delete as `404`.
- Static mapping reload workflow: list by `name` and `metadata.wiremock-gui.folder`, then update through `PUT /__admin/mappings/{id}`.
- Mapping fields: `id`, `name`, `persistent`, `priority`, `request`, `response`, `metadata`.
- Unknown mapping fields are preserved in Admin API responses.
- Basic HTTP stubbing for non-Admin requests.
- Request methods: `ANY`, `GET`, `POST`.
- URL matchers: `request.url`, `request.urlPath`, `request.urlPattern`.
- Request body matcher: `bodyPatterns.matchesJsonPath`.
- Request body matcher: `bodyPatterns.matchesJsonPath.expression` with `absent=true`.
- JSONPath filters with `?()`, arrays, nested fields, scalar equality and `.size()`.
- Query parameter matcher: `request.queryParameters.*.equalTo`.
- Header matcher: `request.headers.*.equalTo`.
- Request body matcher: `bodyPatterns.equalToJson`.
- Priority selection with deterministic insertion-order tie-breaker.
- Response fields: `status`, `headers`, `body`, `jsonBody`.
- Response field: `bodyFileName` backed by in-memory file storage.
- Response transformer: targeted `response-template`.
- Response template helper: `{{jsonPath request.body '...'}}`.
- JSON-RPC-style request id echo through response templating.
- Binary response body files are returned without text recoding.
- Proxy mappings via `response.proxyBaseUrl`.
- Proxy prefix rewriting via `response.proxyUrlPrefixToRemove`.
- Recording API: `POST /__admin/recordings/start`.
- Recording API: `POST /__admin/recordings/stop`.
- Snapshot API: `POST /__admin/recordings/snapshot`.
- Recorded mappings are created in memory and activated after `stop` or `snapshot`.
- Recorded binary response bodies use `base64Body`.
- Response delays: `fixedDelayMilliseconds`, `delayDistribution`, `chunkedDribbleDelay`.
- Stateful scenarios via `scenarioName`, `requiredScenarioState`, `newScenarioState`.
- Scenario reset endpoint: `POST /__admin/scenarios/reset`.
- Legacy file auth endpoint: `POST /api/login`.
- Legacy file upload create endpoint: `POST /api/tus/{file}?override=true`.
- Legacy file upload bytes endpoint: `PATCH /api/tus/{file}?override=true`.
- `Upload-Metadata` filename parsing for current autotest file upload workflow.
- gRPC descriptor Admin API: `GET /__admin/ext/grpc/descriptors`.
- gRPC descriptor Admin API: `PUT /__admin/ext/grpc/descriptors/{fileName}`.
- gRPC descriptor Admin API: `DELETE /__admin/ext/grpc/descriptors/{fileName}`.
- gRPC descriptor registry reload: `POST /__admin/ext/grpc/reset`.
- Legacy `.dsc` and `.desc` file uploads feed the gRPC descriptor registry when the uploaded bytes are valid `FileDescriptorSet` data.
- Unary gRPC stubbing runtime for loaded descriptors.
- gRPC request protobuf is adapted to protobuf JSON and matched by existing WireMock request matchers.
- gRPC JSON response bodies are encoded back to protobuf responses.
- gRPC status mapping via `grpc-status-name`, `grpc-status-reason`, and selected HTTP statuses.
- GraphQL semantic body matcher via `request.customMatcher.name = graphql-body-matcher`.
- GraphQL query matching ignores whitespace and field order while preserving aliases, arguments, fragments, variables and operation name semantics.
- URL path regex matcher: `request.urlPathPattern`.
- WireMock-like 404 response for unmatched requests.
- Unit, race and contract test quality gate with 90%+ statement coverage.
- Black-box API autotest suite under `autotest/` with safe skip behavior when no VIMock target is configured.

## TODO

- Full JSONPath compatibility beyond patterns used by current mocks.
- Full JSONUnit compatibility for `equalToJson`.
- Full WireMock/Handlebars response template compatibility beyond `jsonPath request.body`.
- Full TUS protocol beyond the current autotest upload workflow.
- Static or persistent body file storage.
- Full WireMock recording spec parity beyond current `targetBaseUrl`, `captureHeaders`, `requestBodyPattern`, `persist` fields.
- gRPC streaming support.
- gRPC proxying and recording.
- gRPC reflection over loaded descriptors.
- `.proto` source compilation for the gRPC descriptor registry.
- GraphQL federation-specific matching.

## Quality Gate

```bash
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache go test -race ./...
GOCACHE=$(pwd)/.gocache go test -coverprofile=coverage.out ./...
GOCACHE=$(pwd)/.gocache go tool cover -func=coverage.out
```

Current statement coverage: `90.3%`.

Black-box API suite:

```bash
VIMOCK_BASE_URL=http://localhost:8080 go test ./autotest/...
VIMOCK_AUTOTEST_START=1 go test ./autotest/...
```

## Documentation

- [Docs index](docs/README.md)
- [Step 1: Service bootstrap](docs/step-01-bootstrap.md)
- [Step 2: Mapping Admin API](docs/step-02-admin-api.md)
- [Step 3: Basic HTTP stubbing](docs/step-03-basic-http-stubbing.md)
- [Step 4: Request matching](docs/step-04-request-matching.md)
- [Step 5: Response templating and body files](docs/step-05-response-templating-and-body-files.md)
- [Step 6: Legacy File API](docs/step-06-legacy-file-api.md)
- [Step 8: Proxy and delays](docs/step-08-proxy-and-delays.md)
- [Step 9: Stateful scenarios](docs/step-09-stateful-scenarios.md)
- [Step 10: Runtime-generated workflow](docs/step-10-runtime-generated-workflow.md)
- [Step 11: gRPC descriptor registry](docs/step-11-grpc-descriptor-registry.md)
- [Step 12: gRPC stubbing runtime](docs/step-12-grpc-stubbing-runtime.md)
- [Step 13: GraphQL semantic matcher](docs/step-13-graphql-semantic-matcher.md)
- [Step 14: Recording and snapshotting](docs/step-14-recording-and-snapshotting.md)
- [Step 15: HTTPS, Docker and performance baseline](docs/step-15-https-docker-kubernetes-performance.md)
- [Step 16: MVP acceptance and quality gate](docs/step-16-mvp-acceptance.md)
- [Step 17: Black-box API autotests](docs/step-17-black-box-autotests.md)
- [MVP compliance report](docs/mvp-compliance-report.md)

## Scope guardrails

The current implementation is incremental. It includes the service bootstrap, HTTP/HTTPS port configuration, stdout logging, health/readiness endpoints, Admin API CRUD for mappings, basic HTTP stubbing, request matching needed by current mocks, targeted response templating, in-memory body files, proxy fallback, recording/snapshotting, delays, stateful scenarios, runtime-generated mapping lifecycle checks, the legacy file upload workflow used by current autotests, the gRPC descriptor registry foundation, unary gRPC stubbing runtime, GraphQL semantic matching, Docker hardening, performance benchmarks, a 90%+ coverage quality gate and black-box API autotests.

Advanced request matching beyond current fixtures, full WireMock response templating, full TUS support, advanced recording modes, advanced gRPC features, and GraphQL federation-specific behavior are intentionally added in separate increments described in `plan.md`.

## License

VIMock is licensed under the [Apache License 2.0](LICENSE).
