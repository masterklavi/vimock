# gRPC Descriptors

VIMock нужны protobuf descriptors, чтобы декодировать unary gRPC requests и кодировать protobuf responses.

## Что загружать

Поддерживаемые descriptor inputs:

- `.dsc` descriptor set files.
- `.desc` descriptor set files.
- `.proto` source files можно хранить и видеть в списке, но они пока не компилируются в active runtime registry.

Для runtime gRPC stubbing используйте `.dsc` или `.desc`, сгенерированные как protobuf `FileDescriptorSet`.

## Рекомендуемый способ через Admin API

Запустить VIMock:

```bash
go run ./cmd/vimock
```

Загрузить descriptor set:

```bash
curl -i -X PUT \
  --data-binary @testdata/mc_product.dsc \
  http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
```

Перезагрузить active registry:

```bash
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

Посмотреть descriptors и active services:

```bash
curl -s http://localhost:8080/__admin/ext/grpc/descriptors
```

Ожидаемая форма ответа:

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

Удалить descriptor:

```bash
curl -i -X DELETE http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

## Legacy File API Upload

Используйте этот способ, если существующие autotests или bootstrap scripts загружают descriptors через WireMock filebrowser-like TUS API.

Получить token:

```bash
curl -i -X POST http://localhost:8080/api/login
```

Ожидаемое body:

```text
vimock-file-token
```

Создать upload через nested legacy path:

```bash
curl -i -X POST 'http://localhost:8080/api/tus/grpc/mc_product.dsc?override=true' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Length: 40026' \
  -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' \
  -H 'X-Auth: vimock-file-token'
```

`Upload-Metadata: filename ...` использует hex-encoded file name. Для `mc_product.dsc` значение такое:

```text
6d635f70726f647563742e647363
```

Загрузить bytes:

```bash
curl -i -X PATCH 'http://localhost:8080/api/tus/grpc/mc_product.dsc?override=true' \
  -H 'Content-Type: application/offset+octet-stream' \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Offset: 0' \
  -H 'X-Auth: vimock-file-token' \
  --data-binary @testdata/mc_product.dsc
```

Перезагрузить descriptors:

```bash
curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset
```

VIMock принимает nested legacy paths, например `/api/tus/grpc/mc_product.dsc`, но сохраняет basename `mc_product.dsc`. Если загруженные bytes являются валидным descriptor set, они копируются в gRPC descriptor registry.

## Загрузка `.proto` файлов

Можно загрузить `.proto` files:

```bash
curl -i -X PUT \
  --data-binary @service.proto \
  http://localhost:8080/__admin/ext/grpc/descriptors/service.proto
```

Текущее поведение:

- Файл хранится и показывается в `GET /__admin/ext/grpc/descriptors`.
- `loadable` равно `false`.
- `POST /__admin/ext/grpc/reset` не компилирует `.proto` source в active registry.

Используйте descriptor sets для runtime matching, пока не реализована компиляция `.proto`.

## Troubleshooting

### `POST /api/tus/grpc/mc_product.dsc` возвращает 404

Нужна версия VIMock с поддержкой `POST /api/tus/{file...}` и `PATCH /api/tus/{file...}`. Старый route pattern принимал только один path segment.

### `registry.files` остается `0`

Проверьте:

- После upload был вызван `POST /__admin/ext/grpc/reset`.
- Расширение файла `.dsc` или `.desc`.
- Загруженные bytes являются валидным protobuf `FileDescriptorSet`.
- Вы не загрузили только `.proto` файл, ожидая его компиляцию.

### gRPC request возвращает `UNIMPLEMENTED`

Проверьте:

- Descriptor registry содержит нужный service в `registry.services`.
- Mapping URL path равен `/<fully-qualified-service>/<method>`.
- Request body матчится mapping-ом после protobuf-to-JSON conversion.

## Следующий шаг

После загрузки descriptors создайте mappings по статье [gRPC stubbing](grpc-stubbing.md).
