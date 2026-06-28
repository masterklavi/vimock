# Step 6: Legacy File API

## What Is Available

- `POST /api/login`.
- `POST /api/tus/{file}?override=true`.
- `PATCH /api/tus/{file}?override=true`.
- `Upload-Metadata` filename parsing.
- `X-Auth` token check for upload requests.
- Uploaded bytes are stored in the shared in-memory file store.
- Uploaded files can be returned through `response.bodyFileName`.
- `POST /__admin/ext/grpc/reset` compatibility hook.

## Current Scope

This step implements the file workflow used by `autotests-example/utils/mock_utils.py`.

The repository includes a real PDM descriptor fixture for local checks:

```text
testdata/mc_product.dsc
```

The following features are intentionally not complete yet:

- Full TUS protocol.
- Chunked/resumable uploads with non-zero offsets.
- Persistent file storage.
- Native gRPC descriptor registry reload.
- gRPC stubbing.

## Upload Workflow

Start VIMock:

```bash
go run ./cmd/vimock
```

Get an upload token:

```bash
curl -i -X POST http://localhost:8080/api/login
```

Expected result:

```http
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
```

```text
vimock-file-token
```

Create an upload:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/mc_product.dsc?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 40026' \
  -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' \
  -H 'X-Auth: vimock-file-token'
```

Expected result:

```http
HTTP/1.1 201 Created
Location: /api/tus/mc_product.dsc
Tus-Resumable: 1.0.0
Upload-Offset: 0
```

Upload bytes:

```bash
curl -i -X PATCH 'http://localhost:8080/api/tus/mc_product.dsc?override=true' \
  -H 'Content-Type: application/offset+octet-stream' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Offset: 0' \
  -H 'X-Auth: vimock-file-token' \
  --data-binary @testdata/mc_product.dsc
```

Expected result:

```http
HTTP/1.1 204 No Content
Tus-Resumable: 1.0.0
Upload-Offset: <uploaded-bytes>
```

Reload gRPC extension compatibility hook:

```bash
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

Expected result:

```http
HTTP/1.1 200 OK
Content-Length: 0
```

## Tests

```bash
go test ./...
go test -race ./...
go test ./internal/server -run TestLegacyFileUploadWorkflow
```
