# Body Files And Legacy Upload

VIMock stores uploaded files in memory. Mappings can return uploaded bytes with `response.bodyFileName`.

## Upload With Legacy File API

Get token:

```bash
curl -i -X POST http://localhost:8080/api/login
```

Create upload:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/payload.bin?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 6' \
  -H 'Upload-Metadata: filename 7061796c6f61642e62696e' \
  -H 'X-Auth: vimock-file-token'
```

Upload bytes:

```bash
curl -i -X PATCH 'http://localhost:8080/api/tus/payload.bin?override=true' \
  -H 'Content-Type: application/offset+octet-stream' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Offset: 0' \
  -H 'X-Auth: vimock-file-token' \
  --data-binary @payload.bin
```

Nested paths are accepted for compatibility:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/grpc/mc_product.dsc?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 40026' \
  -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' \
  -H 'X-Auth: vimock-file-token'
```

VIMock stores only the basename, for example `mc_product.dsc`.

## Return A Body File

Mapping:

```json
{
  "request": {
    "method": "GET",
    "urlPath": "/download"
  },
  "response": {
    "status": 200,
    "headers": {
      "Content-Type": "application/octet-stream"
    },
    "bodyFileName": "payload.bin"
  }
}
```

Create and call:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @download-mapping.json

curl -i http://localhost:8080/download
```

## Scope

- Upload storage is in memory.
- Only full upload with `Upload-Offset: 0` is supported.
- Full TUS resumable protocol is not implemented.
- `.dsc` and `.desc` uploads also feed the gRPC descriptor registry when bytes are valid descriptor sets.
