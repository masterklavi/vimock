# Step 12: gRPC Stubbing Runtime

## What Is Available

- Unary gRPC requests are detected by `POST` and `Content-Type: application/grpc`.
- gRPC method paths use WireMock-compatible URLs: `/<fully-qualified-service>/<method>`.
- Request protobuf messages are decoded with the active descriptor registry.
- Decoded requests are converted to protobuf JSON with proto field names.
- Existing WireMock request matchers are reused against the converted JSON body.
- Existing response templating is reused before protobuf response encoding.
- Response JSON bodies are encoded back to the method output protobuf message.
- `grpc-status-name` and `grpc-status-reason` response headers are converted to gRPC trailers.
- Selected HTTP response statuses are converted to gRPC statuses when `grpc-status-name` is absent.

## Runtime Flow

```text
gRPC frame -> protobuf request -> JSON body -> mapping match -> JSON response -> protobuf response -> gRPC frame
```

For a method `pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature`, the mapping request URL is:

```text
/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature
```

## Setup Flow

1. Upload a descriptor set.
2. Reload the descriptor registry.
3. Create or load WireMock mappings that use gRPC method paths.
4. Call the service with a unary gRPC client.

Descriptor setup example:

```bash
curl -i -X PUT \
  --data-binary @testdata/mc_product.dsc \
  http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc

curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

The representative PDM mapping fixture used by tests is stored at:

```text
testdata/grpc_mapping.json
```

## Status Mapping

If `grpc-status-name` is present and not `OK`, VIMock returns a gRPC error and ignores the response body.

Supported status names in this increment:

- `OK`
- `CANCELED` or `CANCELLED`
- `UNKNOWN`
- `INVALID_ARGUMENT`
- `NOT_FOUND`
- `PERMISSION_DENIED`
- `UNIMPLEMENTED`
- `INTERNAL`
- `UNAVAILABLE`
- `UNAUTHENTICATED`

If `grpc-status-name` is absent, these HTTP response statuses are mapped to gRPC errors:

- `400 -> INTERNAL`
- `401 -> UNAUTHENTICATED`
- `403 -> PERMISSION_DENIED`
- `404 -> UNIMPLEMENTED`
- `429 -> UNAVAILABLE`
- `502 -> UNAVAILABLE`
- `503 -> UNAVAILABLE`
- `504 -> UNAVAILABLE`

Unmatched gRPC requests return `UNIMPLEMENTED` with:

```text
No matching stub mapping found for gRPC request
```

## Current Scope

- Unary gRPC calls are supported.
- Client-streaming and server-streaming are not implemented yet.
- gRPC reflection is not implemented yet.
- gRPC proxying and recording are not implemented yet.
- `.proto` source compilation is not implemented yet.
- Compressed gRPC messages are not supported yet.

## Tests

```bash
go test ./internal/server -run 'TestGRPCRuntime'
go test ./...
```
