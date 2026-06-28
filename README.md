# ViMock

ВИ.Мок для локальной быстрой работы с моками в стиле Wiremock

## Запуск

```bash
go run ./cmd/vimock
```

По умолчанию сервис слушает `0.0.0.0:8080`.

Настройки:

- `--host` или `VIMOCK_HOST`
- `--port` или `VIMOCK_PORT`

Проверка:

```bash
curl -i http://localhost:8080/__admin/health
curl -i http://localhost:8080/__admin/ready
```

Docker:

```bash
docker build -t vimock:dev .
docker run --rm -p 8080:8080 vimock:dev
```

## Scope guardrails

На первом шаге реализован только каркас сервиса: запуск, конфигурация порта, stdout logging и health/readiness endpoints.

Пока намеренно не реализованы mappings, request matching, response templating, proxying, recording, gRPC и GraphQL. Эти фичи добавляются отдельными инкрементами из `plan.md`.
