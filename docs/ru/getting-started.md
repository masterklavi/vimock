# Быстрый старт

## Запустить VIMock

```bash
go run ./cmd/vimock
```

Адрес по умолчанию:

```text
http://localhost:8080
```

Health checks:

```bash
curl -i http://localhost:8080/__admin/health
curl -i http://localhost:8080/__admin/ready
```

## Создать первый HTTP stub

Создать mapping:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "hello",
    "request": {
      "method": "GET",
      "urlPath": "/hello"
    },
    "response": {
      "status": 200,
      "headers": {
        "Content-Type": "application/json"
      },
      "jsonBody": {
        "message": "hello from VIMock"
      }
    }
  }'
```

Вызвать stub:

```bash
curl -i http://localhost:8080/hello
```

Посмотреть mappings:

```bash
curl -s http://localhost:8080/__admin/mappings
```

## Следующие задачи

- Загрузить gRPC descriptors: [gRPC descriptors](grpc-descriptors.md).
- Создать gRPC stub: [gRPC stubbing](grpc-stubbing.md).
- Загрузить body files: [Body files и legacy upload](body-files-and-legacy-upload.md).
- Настроить HTTPS или Docker: [Конфигурация](configuration.md).
