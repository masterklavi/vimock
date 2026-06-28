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

Release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Tag pushes create GitHub Release assets for Linux and macOS on `amd64` and `arm64`.

## Available Functionality

- Service bootstrap with graceful shutdown.
- HTTP server with JSON stdout logging.
- CLI/env configuration: `--host`, `--port`, `VIMOCK_HOST`, `VIMOCK_PORT`.
- Health endpoint: `GET /__admin/health`.
- Readiness endpoint: `GET /__admin/ready`.
- Docker image build via `docker build -t vimock:dev .`.
- HTTP/1.1, HTTP/2 and unencrypted HTTP/2 server transport base.
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
- WireMock-like 404 response for unmatched requests.

## TODO

- Full JSONPath compatibility beyond patterns used by current mocks.
- Full JSONUnit compatibility for `equalToJson`.
- Full WireMock/Handlebars response template compatibility beyond `jsonPath request.body`.
- Full TUS protocol beyond the current autotest upload workflow.
- Static or persistent body file storage.
- Recording and snapshotting.
- gRPC request/response protobuf conversion and gRPC stubbing runtime.
- gRPC reflection over loaded descriptors.
- `.proto` source compilation for the gRPC descriptor registry.
- GraphQL matcher support.
- Black-box API autotests.
- Final 90% unit test coverage gate.

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

## Scope guardrails

The current implementation is incremental. It includes the service bootstrap, port configuration, stdout logging, health/readiness endpoints, Admin API CRUD for mappings, basic HTTP stubbing, request matching needed by current mocks, targeted response templating, in-memory body files, proxy fallback, delays, stateful scenarios, runtime-generated mapping lifecycle checks, the legacy file upload workflow used by current autotests, and the gRPC descriptor registry foundation.

Advanced request matching beyond current fixtures, full WireMock response templating, full TUS support, recording, gRPC runtime execution, and GraphQL are intentionally added in separate increments described in `plan.md`.

## License

VIMock is licensed under the [Apache License 2.0](LICENSE).
