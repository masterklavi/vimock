# VIMock Documentation

This documentation is organized by user tasks, not by implementation steps.

Russian version: [ru/README.md](ru/README.md)

## Start Here

- [Getting started](getting-started.md): run VIMock, check health, create the first stub.
- [Configuration](configuration.md): CLI flags, environment variables, Docker and HTTPS.
- [Testing VIMock](testing.md): unit tests, race tests and black-box API autotests.

## Common Tasks

| I want to... | Read this |
|---|---|
| Create a basic HTTP mock | [HTTP stubbing](http-stubbing.md) |
| Match requests by URL, query, headers or body JSONPath | [HTTP stubbing](http-stubbing.md#request-matching) |
| Return JSON, text or binary response bodies | [HTTP stubbing](http-stubbing.md#responses) |
| Upload files for `bodyFileName` | [Body files and legacy upload](body-files-and-legacy-upload.md) |
| Upload gRPC `.dsc` or `.desc` descriptors | [gRPC descriptors](grpc-descriptors.md) |
| Use legacy `/api/tus/grpc/mc_product.dsc` descriptor upload | [gRPC descriptors: legacy upload](grpc-descriptors.md#legacy-file-api-upload) |
| Create a unary gRPC stub mapping | [gRPC stubbing](grpc-stubbing.md) |
| Use GraphQL semantic matching | [GraphQL matching](graphql.md) |
| Proxy unmatched or fallback requests | [Proxying](proxying.md) |
| Record upstream responses into mappings | [Recording](recording.md) |
| Run public API black-box checks | [Testing VIMock](testing.md#black-box-api-autotests) |

## Compatibility Notes

- Mapping and runtime state are in memory.
- `.dsc` and `.desc` descriptor sets are loadable by the gRPC runtime.
- `.proto` files can be uploaded and listed, but source compilation into the active runtime registry is not implemented yet.
- gRPC runtime currently supports unary calls. Streaming, reflection, gRPC proxying and gRPC recording are not implemented yet.
- Response templating is intentionally limited to the helpers used by current mocks, primarily `{{jsonPath request.body '...'}}`.

## API Reference By Area

- [HTTP stubbing](http-stubbing.md)
- [Body files and legacy upload](body-files-and-legacy-upload.md)
- [gRPC descriptors](grpc-descriptors.md)
- [gRPC stubbing](grpc-stubbing.md)
- [GraphQL matching](graphql.md)
- [Proxying](proxying.md)
- [Recording](recording.md)
- [Configuration](configuration.md)
- [Testing VIMock](testing.md)
