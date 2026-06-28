# Body Files и Legacy Upload

VIMock хранит загруженные файлы in-memory. Mappings могут возвращать bytes через `response.bodyFileName`.

## Загрузка через Legacy File API

Получить token:

```bash
curl -i -X POST http://localhost:8080/api/login
```

Создать upload:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/payload.bin?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 6' \
  -H 'Upload-Metadata: filename 7061796c6f61642e62696e' \
  -H 'X-Auth: vimock-file-token'
```

Загрузить bytes:

```bash
curl -i -X PATCH 'http://localhost:8080/api/tus/payload.bin?override=true' \
  -H 'Content-Type: application/offset+octet-stream' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Offset: 0' \
  -H 'X-Auth: vimock-file-token' \
  --data-binary @payload.bin
```

Nested paths принимаются для совместимости:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/grpc/mc_product.dsc?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 40026' \
  -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' \
  -H 'X-Auth: vimock-file-token'
```

VIMock сохраняет только basename, например `mc_product.dsc`.

## Вернуть Body File

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

Создать и вызвать:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @download-mapping.json

curl -i http://localhost:8080/download
```

## Scope

- Upload storage in-memory.
- Поддержан только полный upload с `Upload-Offset: 0`.
- Full TUS resumable protocol не реализован.
- `.dsc` и `.desc` uploads также попадают в gRPC descriptor registry, если bytes являются валидным descriptor set.
