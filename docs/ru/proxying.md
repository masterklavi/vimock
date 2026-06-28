# Proxying

Используйте proxy mappings, когда VIMock должен отправлять request в upstream service.

## Fallback Proxy Mapping

```json
{
  "name": "api proxy fallback",
  "priority": 10,
  "request": {
    "method": "ANY",
    "urlPattern": "/api/.*"
  },
  "response": {
    "status": 200,
    "proxyBaseUrl": "https://example.com",
    "proxyUrlPrefixToRemove": "/api"
  }
}
```

Создать mapping:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @proxy-mapping.json
```

Вызвать VIMock:

```bash
curl -i http://localhost:8080/api/get
```

С `proxyUrlPrefixToRemove=/api` VIMock отправит запрос в:

```text
https://example.com/get
```

## Priority

Используйте меньшее priority для точных stubs и большее значение для fallback proxy mappings. Пример:

- Exact stub: `priority: 1`.
- Proxy fallback: `priority: 10`.

## Текущие ограничения

- HTTP proxying поддержан.
- gRPC proxying пока не реализован.
