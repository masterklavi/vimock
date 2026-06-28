# Документация VIMock

Эта документация организована по задачам пользователя, а не по шагам реализации.

English version: [../README.md](../README.md)

## С чего начать

- [Быстрый старт](getting-started.md): запустить VIMock, проверить health, создать первый stub.
- [Конфигурация](configuration.md): CLI flags, переменные окружения, Docker и HTTPS.
- [Тестирование VIMock](testing.md): unit tests, race tests и black-box API autotests.

## Частые задачи

| Что нужно сделать | Читать |
|---|---|
| Создать простой HTTP mock | [HTTP stubbing](http-stubbing.md) |
| Матчить запросы по URL, query, headers или body JSONPath | [HTTP stubbing](http-stubbing.md#request-matching) |
| Вернуть JSON, текст или binary response body | [HTTP stubbing](http-stubbing.md#responses) |
| Загрузить файлы для `bodyFileName` | [Body files и legacy upload](body-files-and-legacy-upload.md) |
| Загрузить gRPC `.dsc` или `.desc` descriptors | [gRPC descriptors](grpc-descriptors.md) |
| Использовать legacy `/api/tus/grpc/mc_product.dsc` upload | [gRPC descriptors: legacy upload](grpc-descriptors.md#legacy-file-api-upload) |
| Создать unary gRPC stub mapping | [gRPC stubbing](grpc-stubbing.md) |
| Использовать semantic GraphQL matching | [GraphQL matching](graphql.md) |
| Проксировать fallback/unmatched запросы | [Proxying](proxying.md) |
| Записать upstream responses в mappings | [Recording](recording.md) |
| Запустить public API black-box checks | [Тестирование VIMock](testing.md#black-box-api-autotests) |

## Compatibility notes

- Mappings и runtime state хранятся in-memory.
- `.dsc` и `.desc` descriptor sets используются gRPC runtime.
- `.proto` файлы можно загрузить и увидеть в списке, но компиляция `.proto` в active runtime registry пока не реализована.
- gRPC runtime сейчас поддерживает unary calls. Streaming, reflection, gRPC proxying и gRPC recording пока не реализованы.
- Response templating ограничен helper-ами, которые нужны текущим mocks, в первую очередь `{{jsonPath request.body '...'}}`.

## API reference по областям

- [HTTP stubbing](http-stubbing.md)
- [Body files и legacy upload](body-files-and-legacy-upload.md)
- [gRPC descriptors](grpc-descriptors.md)
- [gRPC stubbing](grpc-stubbing.md)
- [GraphQL matching](graphql.md)
- [Proxying](proxying.md)
- [Recording](recording.md)
- [Configuration](configuration.md)
- [Тестирование VIMock](testing.md)
