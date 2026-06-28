# Recording

Recording проксирует requests в upstream service и превращает наблюдаемый traffic в in-memory mappings.

## Start Recording

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/start \
  -H 'Content-Type: application/json' \
  -d '{
    "targetBaseUrl": "https://example.com",
    "captureHeaders": {
      "X-Request-Id": {}
    },
    "requestBodyPattern": "equalToJson",
    "persist": true
  }'
```

После этого отправляйте requests через VIMock:

```bash
curl -i http://localhost:8080/api/items?sku=1 -H 'X-Request-Id: req-1'
```

## Stop And Activate Recorded Mappings

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/stop
```

Response содержит generated mappings, и VIMock сразу активирует их in-memory.

## Snapshot Existing Serve Events

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/snapshot \
  -H 'Content-Type: application/json' \
  -d '{
    "captureHeaders": {
      "X-Request-Id": {}
    },
    "requestBodyPattern": "equalToJson",
    "persist": true
  }'
```

## Текущие ограничения

- Recorded mappings хранятся in-memory.
- Full WireMock recording spec parity пока не завершен.
- Binary response bodies записываются как `base64Body`; body file extraction через `extractBodyCriteria` пока не реализован.
- gRPC recording пока не реализован.
