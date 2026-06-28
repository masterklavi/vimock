# gRPC Descriptors

VIMock needs protobuf descriptors to decode unary gRPC requests and encode protobuf responses.

## What To Upload

Supported descriptor inputs:

- `.dsc` descriptor set files.
- `.desc` descriptor set files.
- `.proto` source files can be stored and listed, but they are not compiled into the active runtime registry yet.

For runtime gRPC stubbing, use `.dsc` or `.desc` generated as a protobuf `FileDescriptorSet`.

## Recommended Admin API Upload

Start VIMock:

```bash
go run ./cmd/vimock
```

Upload a descriptor set:

```bash
curl -i -X PUT \
  --data-binary @testdata/mc_product.dsc \
  http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
```

Reload the active registry:

```bash
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

List descriptors and active services:

```bash
curl -s http://localhost:8080/__admin/ext/grpc/descriptors
```

Expected response shape:

```json
{
  "descriptors": [
    {
      "name": "mc_product.dsc",
      "kind": "descriptor-set",
      "size": 40026,
      "loadable": true,
      "updatedAt": "2026-06-29T00:00:00Z"
    }
  ],
  "registry": {
    "generation": 1,
    "files": 1,
    "services": ["pdm_api_gateway.v1.MCProduct"],
    "messages": ["..."]
  },
  "meta": {"total": 1}
}
```

Delete a descriptor:

```bash
curl -i -X DELETE http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

## Legacy File API Upload

Use this when existing autotests or bootstrap scripts upload descriptors through a WireMock filebrowser-like TUS API.

Get token:

```bash
curl -i -X POST http://localhost:8080/api/login
```

Expected body:

```text
vimock-file-token
```

Create upload using a nested legacy path:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/grpc/mc_product.dsc?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 40026' \
  -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' \
  -H 'X-Auth: vimock-file-token'
```

`Upload-Metadata: filename ...` uses the hex-encoded file name. For `mc_product.dsc`, the value is:

```text
6d635f70726f647563742e647363
```

Upload bytes:

```bash
curl -i -X PATCH 'http://localhost:8080/api/tus/grpc/mc_product.dsc?override=true' \
  -H 'Content-Type: application/offset+octet-stream' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Offset: 0' \
  -H 'X-Auth: vimock-file-token' \
  --data-binary @testdata/mc_product.dsc
```

Reload descriptors:

```bash
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

VIMock accepts nested legacy paths such as `/api/tus/grpc/mc_product.dsc`, but stores the basename `mc_product.dsc`. If the uploaded bytes are a valid descriptor set, they are copied into the gRPC descriptor registry.

## Uploading `.proto` Files

You can upload `.proto` files:

```bash
curl -i -X PUT \
  --data-binary @service.proto \
  http://localhost:8080/__admin/ext/grpc/descriptors/service.proto
```

Current behavior:

- The file is stored and shown by `GET /__admin/ext/grpc/descriptors`.
- `loadable` is `false`.
- `POST /__admin/ext/grpc/reset` does not compile `.proto` source into the active registry.

Use descriptor sets for runtime matching until `.proto` compilation is implemented.

## Troubleshooting

### `POST /api/tus/grpc/mc_product.dsc` returns 404

Use a VIMock version with support for `POST /api/tus/{file...}` and `PATCH /api/tus/{file...}`. Older route patterns only accepted one path segment.

### `registry.files` stays `0`

Check these points:

- `POST /__admin/ext/grpc/reset` was called after upload.
- The uploaded file extension is `.dsc` or `.desc`.
- The uploaded bytes are a valid protobuf `FileDescriptorSet`.
- You did not upload only a `.proto` file and expect it to compile.

### gRPC request returns `UNIMPLEMENTED`

Check these points:

- The descriptor registry lists the expected service under `registry.services`.
- The mapping URL path is `/<fully-qualified-service>/<method>`.
- The request body matches the mapping after protobuf-to-JSON conversion.

## Next Step

After descriptors are loaded, create mappings as described in [gRPC stubbing](grpc-stubbing.md).
