# gRPC Stubbing

VIMock supports unary gRPC stubs using WireMock-style JSON mappings.

## Setup Flow

1. Upload a `.dsc` or `.desc` descriptor set.
2. Call `POST /__admin/ext/grpc/reset`.
3. Create a mapping where `request.urlPath` is the gRPC method path.
4. Call VIMock with a unary gRPC client.

Descriptor setup is documented in [gRPC descriptors](grpc-descriptors.md).

## Mapping Shape

For service `pdm_api_gateway.v1.MCProduct` and method `WarehousesByNomenclature`, the URL path is:

```text
/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature
```

Example mapping:

```json
{
  "name": "WarehousesByNomenclature one warehouse",
  "persistent": true,
  "priority": 1,
  "request": {
    "method": "POST",
    "urlPath": "/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature",
    "bodyPatterns": [
      {
        "matchesJsonPath": "$[?(@.guids == ['b27ed95d-3717-4538-9be6-a7136b8ad52f'])]"
      }
    ]
  },
  "response": {
    "status": 200,
    "body": "{\"warehouses\":[{\"warehouses_guid\":[\"00000000-0000-0000-0000-050258290258\"],\"nomenclature_guid\":\"b27ed95d-3717-4538-9be6-a7136b8ad52f\"}]}"
  }
}
```

Upload it:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @testdata/grpc_mapping.json
```

## How Matching Works

Runtime flow:

```text
gRPC frame -> protobuf request -> JSON body -> mapping match -> JSON response -> protobuf response -> gRPC frame
```

Request matchers run against protobuf JSON with proto field names. For example, a request protobuf with `guids` becomes JSON like:

```json
{"guids": ["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}
```

## gRPC Status Responses

Use `grpc-status-name` and optional `grpc-status-reason` response headers:

```json
{
  "response": {
    "status": 200,
    "headers": {
      "grpc-status-name": "NOT_FOUND",
      "grpc-status-reason": "missing warehouse"
    }
  }
}
```

Supported status names:

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

## Try With grpcurl

If `grpcurl` is installed:

```bash
grpcurl \
  -plaintext \
  -protoset testdata/mc_product.dsc \
  -d '{"guids":["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}' \
  localhost:8080 \
  pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature
```

## Current Limitations

- Unary calls only.
- No gRPC reflection yet.
- No gRPC proxying or recording yet.
- No compressed gRPC messages yet.
- `.proto` files are not compiled into runtime descriptors yet.
