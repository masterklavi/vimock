# gRPC Stubbing

VIMock поддерживает unary gRPC stubs через WireMock-style JSON mappings.

## Setup Flow

1. Загрузить `.dsc` или `.desc` descriptor set.
2. Вызвать `POST /__admin/ext/grpc/reset`.
3. Создать mapping, где `request.urlPath` равен gRPC method path.
4. Вызвать VIMock unary gRPC client-ом.

Настройка descriptors описана в [gRPC descriptors](grpc-descriptors.md).

## Mapping Shape

Для service `pdm_api_gateway.v1.MCProduct` и method `WarehousesByNomenclature` URL path такой:

```text
/pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature
```

Пример mapping:

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

Загрузить mapping:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @testdata/grpc_mapping.json
```

## Как работает matching

Runtime flow:

```text
gRPC frame -> protobuf request -> JSON body -> mapping match -> JSON response -> protobuf response -> gRPC frame
```

Request matchers работают с protobuf JSON и proto field names. Например request protobuf с `guids` становится JSON:

```json
{"guids": ["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}
```

## gRPC Status Responses

Используйте `grpc-status-name` и optional `grpc-status-reason` response headers:

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

Поддерживаемые status names:

- `OK`
- `CANCELED` или `CANCELLED`
- `UNKNOWN`
- `INVALID_ARGUMENT`
- `NOT_FOUND`
- `PERMISSION_DENIED`
- `UNIMPLEMENTED`
- `INTERNAL`
- `UNAVAILABLE`
- `UNAUTHENTICATED`

## Проверка через grpcurl

Если установлен `grpcurl`:

```bash
grpcurl \
  -plaintext \
  -protoset testdata/mc_product.dsc \
  -d '{"guids":["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}' \
  localhost:8080 \
  pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature
```

## Текущие ограничения

- Только unary calls.
- gRPC reflection пока нет.
- gRPC proxying и recording пока нет.
- Compressed gRPC messages пока не поддерживаются.
- `.proto` файлы пока не компилируются в runtime descriptors.
