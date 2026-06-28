# Step 11: gRPC Descriptor Registry

## What Is Available

- The HTTP server enables HTTP/1.1, HTTP/2 and unencrypted HTTP/2.
- Descriptor files are stored in a dedicated in-memory gRPC descriptor registry.
- `.dsc` and `.desc` uploads are validated as protobuf `FileDescriptorSet` data.
- `.proto` uploads are accepted as UTF-8 source files, but source compilation is not implemented yet.
- `POST /__admin/ext/grpc/reset` atomically rebuilds the active registry from uploaded descriptor sets.
- Legacy `.dsc` and `.desc` file uploads also feed the descriptor registry when the uploaded bytes are valid descriptor sets.

## Admin API

Upload or replace a descriptor set:

```bash
curl -i -X PUT \
  --data-binary @testdata/mc_product.dsc \
  http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
```

List stored descriptors and the currently active registry:

```bash
curl -s http://localhost:8080/__admin/ext/grpc/descriptors
```

Reload the active registry:

```bash
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

Delete a descriptor:

```bash
curl -i -X DELETE http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
```

## Response Shape

Descriptor list responses contain:

```json
{
  "descriptors": [
    {
      "name": "mc_product.dsc",
      "kind": "descriptor-set",
      "size": 1234,
      "loadable": true,
      "updatedAt": "2026-06-28T00:00:00Z"
    }
  ],
  "registry": {
    "generation": 1,
    "files": 1,
    "services": [
      "package.Service"
    ],
    "messages": [
      "package.Request"
    ]
  },
  "meta": {
    "total": 1
  }
}
```

Before `POST /__admin/ext/grpc/reset`, uploaded descriptor files are listed, but `registry.files` can still be `0`.

## Legacy Upload Bridge

The legacy upload API continues to write bytes into the in-memory file store. If the uploaded file name ends with `.dsc` or `.desc` and the body is a valid `FileDescriptorSet`, the same bytes are also copied into the gRPC descriptor registry.

Invalid legacy `.dsc` and `.desc` uploads are still accepted by the legacy file API for compatibility, but they are ignored by the descriptor registry.

## Current Scope

- Protobuf request decoding is not implemented yet.
- Protobuf response encoding is not implemented yet.
- gRPC service/method dispatch is not implemented yet.
- gRPC reflection is not implemented yet.
- `.proto` source compilation is not implemented yet.

## Tests

```bash
go test ./...
go test -race ./internal/grpcdesc ./internal/server
```
