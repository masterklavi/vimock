# Тестирование VIMock

## Repository Tests

```bash
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Black-box autotest package безопасен для обычного `go test ./...`. Если target не задан, network black-box tests skip-аются.

## Black-Box API Autotests

Запуск против уже поднятого VIMock:

```bash
VIMOCK_BASE_URL=http://localhost:8080 go test ./autotest/...
```

Тесты сами соберут и запустят временный VIMock process:

```bash
VIMOCK_AUTOTEST_START=1 go test ./autotest/...
```

Использовать уже собранный binary:

```bash
go build -o ./bin/vimock ./cmd/vimock
VIMOCK_AUTOTEST_START=1 VIMOCK_BINARY=./bin/vimock go test ./autotest/...
```

Для Docker Desktop proxy и recording tests:

```bash
VIMOCK_BASE_URL=http://localhost:8080 \
VIMOCK_AUTOTEST_UPSTREAM_HOST=host.docker.internal \
go test ./autotest/...
```

Feature coverage находится в:

```text
autotest/reports/features.json
```
