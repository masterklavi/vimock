# План реализации VIMock (`vimock`)

## Правила выполнения плана

- Каждый шаг должен оставлять приложение в рабочем состоянии: `go test ./...` проходит, `go run ./cmd/vimock` запускается, базовая проверка через HTTP/gRPC/GraphQL выполняется по описанию шага.
- Каждый шаг должен быть достаточно малым для ИИ-исполнителя: не смешивать несколько крупных подсистем, не переписывать уже работающие слои без необходимости.
- Каждый шаг должен завершаться заполненным отчетом ИИ в секции `Отчет ИИ по шагу N`.
- Если требование из `tz.md` не закрыто полностью, оно остается в плане следующего шага или в Known gaps отчета.
- OUT-требования не реализуются в MVP, но должны быть защищены архитектурно: неизвестные поля не ломают загрузку, а unsupported behavior возвращает понятную ошибку.

## Инкременты

| Шаг | Рабочая версия после шага | Основной результат | Проверка запуска |
|---|---|---|---|
| 1 | `vimock` запускается как пустой сервис | Go module, CLI, HTTP server, health/readiness, stdout logs, Docker skeleton, coverage gate | `go run ./cmd/vimock`, `curl /__admin/health`, `docker build` |
| 2 | `vimock` принимает mappings через Admin API | Mapping model, in-memory storage, CRUD `/__admin/mappings`, stable IDs, unknown fields | `POST/GET/PUT/DELETE /__admin/mappings` |
| 3 | `vimock` отвечает на простые HTTP stubs | Method/URL matching, priority, deterministic selection, status/headers/body/jsonBody | Добавить mapping и дернуть stub URL |
| 4 | `vimock` матчится по текущим request matchers | JSONPath, absent, query/header equalTo, equalToJson foundation | Fixture tests на representative mappings |
| 5 | `vimock` умеет response pipeline текущих моков | response-template, jsonPath helper, bodyFileName, binary body, JSON-RPC id | Моки с `{{jsonPath}}`, PDF/bin bodyFileName |
| 6 | `vimock` совместим с bootstrap автотестов по файлам | Legacy file API `/api/login`, `/api/tus`, `.dsc`, grpc reset no-op/reload hook | Запросы из `mock_utils.py` проходят |
| 7 | `vimock` загружает все текущие static mappings | Fixture loader и тесты на `examples` + `autotests-example/mocks` | Все JSON mappings парсятся и грузятся |
| 8 | `vimock` умеет proxy fallback и delays | `proxyBaseUrl`, `proxyUrlPrefixToRemove`, fixed/random/chunked delays | Stub fallback через proxy, delay tests |
| 9 | `vimock` поддерживает scenarios | Scenario state engine, Started, transitions, concurrent safety | Stateful mapping меняет response |
| 10 | `vimock` проходит runtime-generated workflow автотестов | Runtime create/use/delete, name+folder update flow, PDM reset behavior | Workflow из `generator_utils.py` |
| 11 | `vimock` имеет gRPC descriptor registry | Admin API descriptor upload/list/delete/reset, legacy bridge, HTTP/2/gRPC listener base | Upload `.dsc`, list, reset |
| 12 | `vimock` исполняет gRPC stubs | Protobuf JSON conversion, gRPC URL mapping, status headers, templating | gRPC client вызывает sample service |
| 13 | `vimock` исполняет GraphQL stubs | GraphQL semantic matcher, variables JSON matching, custom matcher JSON format | GraphQL query with reordered fields |
| 14 | `vimock` поддерживает recording/snapshotting | Proxy recording, serve events, start/stop/snapshot, generated mappings | `POST /__admin/recordings/*` |
| 15 | `vimock` готов к CI/K8s/performance baseline | HTTPS config, HTTP/2 config, Docker hardening, graceful shutdown, benchmarks | Binary, Docker, TLS, benchmarks |
| 16 | `vimock` закрывает MVP acceptance | Full fixture/contract/race suite, 90% coverage, docs, final compliance matrix | `go test -race ./...`, coverage >= 90% |
| 17 | `vimock` проверяется black-box автотестами | Отдельная папка `autotest`, запуск против поднятого сервиса, проверка фич из `current-mocks.md` и `current-autotest.md` | `cd autotest && VIMOCK_BASE_URL=http://localhost:8080 go test ./...` |

## Шаг 1. Каркас сервиса и запуск

Цель: создать минимальный, но рабочий `VIMock`/`vimock`, который можно запускать локально и в Docker.

Покрываемые требования: CON-002, CON-003, CON-004, CON-005, CON-006, PROTO-001, TEST-001, RESP-013, RESP-014, OUT-001, OUT-002, OUT-003, OUT-004, OUT-005, ACC-010.

Сделать:

- Инициализировать Go module с именованием `vimock`.
- Создать `cmd/vimock` и минимальный HTTP server.
- Добавить endpoints `GET /__admin/health` и `GET /__admin/ready`.
- Добавить конфиг порта через CLI/env.
- Настроить stdout logging.
- Добавить `Dockerfile` и `.dockerignore`.
- Добавить базовый `go test ./...` и coverage command.
- Зафиксировать out-of-scope guardrails в README или dev docs.

Не делать на этом шаге: mappings, matching, proxy, gRPC, GraphQL.

Проверка результата:

```bash
go test ./...
go run ./cmd/vimock
curl -i http://localhost:8080/__admin/health
docker build -t vimock:dev .
```

Отчет ИИ по шагу 1:

```text
Статус: DONE
Сделано:
- Инициализирован Go module `vimock`.
- Создан entrypoint `cmd/vimock` с HTTP server.
- Добавлены endpoints `GET /__admin/health` и `GET /__admin/ready`.
- Добавлен конфиг адреса запуска через CLI/env: `--host`, `--port`, `VIMOCK_HOST`, `VIMOCK_PORT`.
- Настроено JSON stdout logging для старта, остановки и HTTP requests.
- Добавлены `Dockerfile` и `.dockerignore`.
- Добавлены unit tests для config/server слоев.
- В README зафиксированы команды запуска и guardrails шага 1.

Измененные файлы:
- `go.mod`
- `.gitignore`
- `.dockerignore`
- `Dockerfile`
- `README.md`
- `cmd/vimock/main.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/server/server.go`
- `internal/server/server_test.go`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `go run ./cmd/vimock --host 127.0.0.1 --port 8080`
- `VIMOCK_PORT=8081 go run ./cmd/vimock`
- `docker build -t vimock:dev .`
- `docker run --rm -p 8080:8080 vimock:dev`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 59.1%.
- `go run ./cmd/vimock --host 127.0.0.1 --port 18080` - сервис стартует.
- `curl -i http://127.0.0.1:18080/__admin/health` - HTTP 200, `{"status":"healthy","message":"VIMock is ok","service":"vimock"}`.
- `curl -i http://127.0.0.1:18080/__admin/ready` - HTTP 200, `{"status":"ready","message":"VIMock is ready","service":"vimock"}`.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- CON-002, CON-003, CON-004, CON-005, CON-006, PROTO-001, TEST-001, RESP-013, RESP-014, OUT-001, OUT-002, OUT-003, OUT-004, OUT-005, ACC-010.

Known gaps:
- Mappings, matching, proxy, gRPC и GraphQL не реализованы, это scope следующих шагов.
- Общий coverage пока 59.1%, потому что `cmd/vimock/main.go` не покрыт unit-тестами. Требование 90% остается quality gate следующих шагов.

Риски/решения:
- В sandbox Go не может писать в системный build cache, поэтому проверки запускались с локальным `GOCACHE=.gocache`.
- В sandbox bind/listen, curl к локальному порту и Docker daemon требуют elevated execution; в обычном локальном окружении эти команды должны выполняться без дополнительных прав.
```

## Шаг 2. Mapping model и Admin API CRUD

Цель: сделать in-memory управление WireMock mappings через Admin API без исполнения stubs.

Покрываемые требования: CON-001, CON-007, CON-008, MAP-001, MAP-002, MAP-003, MAP-004, ADM-001, ADM-002, ADM-003, ADM-004, ADM-005, ADM-006, ADM-007, ADM-008, ADM-009, ADM-010, ADM-011, ADM-012, ADM-013, ADM-014, ADM-015, NFR-001, NFR-003, NFR-004, TEST-002.

Сделать:

- Описать internal model для WireMock mapping с сохранением unknown fields.
- Сделать in-memory repository с atomic snapshot или lock-minimal API.
- Реализовать `GET /__admin/mappings`.
- Реализовать `POST /__admin/mappings` с генерацией `id`.
- Реализовать `PUT /__admin/mappings/{id}`.
- Реализовать `DELETE /__admin/mappings/{id}`.
- Сохранять и возвращать `persistent`, `metadata.wiremock-gui.folder`, `name`.
- Добавить validation errors с понятным HTTP response.
- Добавить unit tests на CRUD и concurrent access.

Не делать на этом шаге: matching пользовательских запросов.

Проверка результата:

```bash
go test ./...
go run ./cmd/vimock
curl -X POST http://localhost:8080/__admin/mappings -d @examples/druz/Fry/fry_proxy.json -H 'Content-Type: application/json'
curl http://localhost:8080/__admin/mappings
```

Отчет ИИ по шагу 2:

```text
Статус: DONE
Сделано:
- Добавлена internal model WireMock mapping с top-level полями `id`, `name`, `persistent` и сохранением исходных/unknown JSON fields.
- Добавлена генерация UUID для `POST /__admin/mappings`, если `id` не передан.
- Добавлена UUID validation для Admin API path id и top-level mapping id.
- Добавлен in-memory mapping store с copy-on-write snapshot через `atomic.Value`; чтение `GET/List` идет без write-lock, изменения защищены mutex.
- Реализован `GET /__admin/mappings` с WireMock-like ответом `{ "mappings": [...], "meta": { "total": N } }`.
- Реализован `GET /__admin/mappings/{id}` как дополнительный WireMock-compatible endpoint для чтения одного mapping-а.
- Реализован `POST /__admin/mappings` с HTTP 201 и возвратом полного mapping-а с `id`.
- Реализован `PUT /__admin/mappings/{id}` с HTTP 200, проверкой существования и принудительным использованием path `id`, как в WireMock.
- Реализован `DELETE /__admin/mappings/{id}` с HTTP 200 и телом `{}`; для отсутствующего id возвращается HTTP 404.
- Добавлены JSON validation errors с HTTP 400 и телом `{ "errors": [{ "title": "..." }] }`.
- Добавлены unit tests на model parsing, preservation unknown fields, store CRUD, concurrent access и HTTP Admin API CRUD.

Измененные файлы:
- `internal/mapping/model.go`
- `internal/mapping/store.go`
- `internal/mapping/model_test.go`
- `internal/mapping/store_test.go`
- `internal/server/admin.go`
- `internal/server/admin_test.go`
- `internal/server/server.go`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @examples/druz/Fry/fry_proxy.json`
- `curl http://localhost:8080/__admin/mappings`
- `curl -X PUT http://localhost:8080/__admin/mappings/{id} -H 'Content-Type: application/json' --data-binary @mapping.json`
- `curl -X DELETE http://localhost:8080/__admin/mappings/{id}`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 78.7%.
- Ручная проверка `POST /__admin/mappings` на `examples/druz/Fry/fry_proxy.json` - HTTP 201, сгенерирован UUID, возвращены `name`, `persistent`, `metadata.wiremock-gui.folder`, `request`, `response`.
- Ручная проверка `GET /__admin/mappings` после POST - HTTP 200, `mappings[0]` содержит загруженный Fry Proxy, `meta.total=1`.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- CON-001, CON-007, CON-008, MAP-001, MAP-002, MAP-003, MAP-004, ADM-001, ADM-002, ADM-003, ADM-004, ADM-005, ADM-006, ADM-007, ADM-008, ADM-009, ADM-010, ADM-011, ADM-012, ADM-013, ADM-014, ADM-015, NFR-001, NFR-003, NFR-004, TEST-002.

Known gaps:
- Matching пользовательских HTTP/gRPC/GraphQL запросов не реализован, это scope следующих шагов.
- Физическая persistence mappings на диск не реализована и не требуется для MVP.
- WireMock endpoints `GET /__admin/mappings?limit&offset`, `/__admin/mappings/reset`, `/__admin/mappings/import` пока не реализованы, так как не входят в шаг 2.
- Общий coverage пока 78.7%; требование 90% остается финальным quality gate.

Риски/решения:
- Store публикует immutable snapshots и переиспользует raw JSON payloads как неизменяемые значения, чтобы не копировать большие mappings на каждом read.
- `PUT /__admin/mappings/{id}` сначала проверяет существование id, затем парсит body; это повторяет WireMock-поведение, где PUT отсутствующего id возвращает 404 даже при пустом body.
- В sandbox bind/listen, curl к локальному порту и Docker daemon требуют elevated execution; в обычном локальном окружении эти команды должны выполняться без дополнительных прав.
```

## Шаг 3. Базовый HTTP stubbing

Цель: после загрузки простого mapping-а сервис должен отвечать на пользовательский HTTP-запрос.

Покрываемые требования: MAP-005, MAP-006, MAP-007, MAP-008, MAP-009, RESP-001, RESP-002, RESP-003, RESP-004, RESP-010, NFR-002, NFR-005.

Сделать:

- Добавить runtime router для всех не-Admin HTTP requests.
- Поддержать `request.method=ANY|GET|POST`.
- Поддержать `request.url`, `request.urlPath`, `request.urlPattern`.
- Реализовать priority selection, включая deterministic tie-breaker по порядку добавления.
- Реализовать response `status`, `headers`, `body`, `jsonBody`.
- Вернуть WireMock-like 404 для unmatched request.
- Добавить unit tests и end-to-end tests через `httptest`.

Не делать на этом шаге: body/query/header matchers, templating, files.

Проверка результата:

```bash
go test ./...
go run ./cmd/vimock
curl -X POST http://localhost:8080/__admin/mappings -d @testdata/simple_body_mapping.json -H 'Content-Type: application/json'
curl -i http://localhost:8080/some/path
```

Отчет ИИ по шагу 3:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 4. Request matching для текущих моков

Цель: покрыть matchers, которые реально используются в `current-mocks.md` и `current-autotest.md`.

Покрываемые требования: MATCH-001, MATCH-002, MATCH-003, MATCH-004, MATCH-005, MATCH-006, MATCH-007, MATCH-008, MATCH-009, MATCH-010, MATCH-011, MATCH-012, JRPC-001, TEST-003.

Сделать:

- Подключить JSONPath engine или реализовать адаптер с нужной семантикой WireMock.
- Поддержать string `matchesJsonPath`.
- Поддержать object `matchesJsonPath` с `expression` и `absent=true`.
- Поддержать `.size()`, фильтры `?()`, массивы, вложенные поля, строки, числа, bool.
- Поддержать `queryParameters.*.equalTo`.
- Поддержать `headers.*.equalTo`, включая protobuf content-type.
- Поддержать `equalToJson` как foundation для gRPC и recording.
- Добавить representative fixture tests из `examples` и `autotests-example/mocks`.

Не делать на этом шаге: полный JSONUnit compatibility сверх требований текущих моков.

Проверка результата:

```bash
go test ./...
go test ./internal/matcher -run TestCurrentMockPatterns
```

Отчет ИИ по шагу 4:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 5. Response templating и body files

Цель: сделать response pipeline, достаточный для JSON-RPC, PDF/bin/protobuf body files и `response-template`.

Покрываемые требования: RESP-005, RESP-006, RESP-007, RESP-008, RESP-009, RESP-011, JRPC-002, JRPC-003, FILE-001, FILE-009, FILE-010.

Сделать:

- Реализовать file storage abstraction с in-memory backend.
- Поддержать lookup `response.bodyFileName`.
- Отдавать binary payload без перекодирования.
- Реализовать `response-template` pipeline.
- Поддержать helper `{{jsonPath request.body '...'}}`.
- Проверить JSON-RPC id echo на текущих моках.
- Добавить тесты на JSON body templating, string body templating, PDF/bin bodyFileName.

Не делать на этом шаге: HTTP upload API файлов, gRPC conversion.

Проверка результата:

```bash
go test ./...
go test ./internal/response -run TestTemplateAndBodyFiles
```

Отчет ИИ по шагу 5:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 6. Legacy File API для автотестов

Цель: пройти workflow загрузки файлов из `autotests-example/utils/mock_utils.py`.

Покрываемые требования: FILE-002, FILE-003, FILE-004, FILE-005, FILE-006, FILE-007, FILE-008, GRPC-018, ACC-005.

Сделать:

- Реализовать `POST /api/login`.
- Реализовать `POST /api/tus/{file}?override=true`.
- Реализовать `PATCH /api/tus/{file}?override=true`.
- Разобрать `Upload-Metadata` и сохранить имя файла.
- Сохранять uploaded bytes в file storage.
- Добавить `POST /__admin/ext/grpc/reset` как reload hook, пока без gRPC schema registry.
- Написать integration test, повторяющий `upload_file_to_wiremock()`.

Не делать на этом шаге: полноценный TUS protocol beyond текущих автотестов.

Тонкости реализации из WireMock extensions:

- WireMock gRPC extension не использует `/api/tus`: descriptor-файлы читаются из отдельного blob store `grpc`, а не из обычного `__files`.
- Legacy upload `.dsc`/`.desc` в VIMock должен писать bytes в тот же descriptor storage, который потом использует gRPC registry.
- `/__admin/ext/grpc/reset` на этом шаге может оставаться no-op/reload hook, но контракт ответа должен быть как у extension: HTTP 200 и пустое тело.
- `Upload-Metadata` нужен только для совместимости текущих автотестов; полноценный TUS state machine не нужен, пока автотесты его не требуют.

Проверка результата:

```bash
go test ./...
curl -i -X POST http://localhost:8080/api/login
curl -i -X POST 'http://localhost:8080/api/tus/mc_product.dsc?override=true' -H 'Tus-Resumable: 1.0.0' -H 'Upload-Length: 10' -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' -H 'X-Auth: token'
```

Отчет ИИ по шагу 6:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 7. Загрузка и проверка всех текущих mappings

Цель: все mappings из `examples` и `autotests-example/mocks` должны загружаться и не ломать сервис.

Покрываемые требования: ACC-001, ACC-002, ACC-003, ACC-004, RT-004, TEST-003.

Сделать:

- Добавить fixture test, который читает все `examples/**/*.json`.
- Добавить fixture test, который читает все `autotests-example/mocks/**/*.json`.
- Загружать каждый mapping через тот же parser/model, что Admin API.
- Проверить сохранение `metadata`, `persistent`, `priority`, request/response fields.
- Подготовить небольшой набор HTTP request fixtures для smoke matching по основным папкам.
- Добавить отчет теста с количеством загруженных mappings и unsupported полями.

Не делать на этом шаге: полное покрытие всех 291 моков реальными request payloads.

Проверка результата:

```bash
go test ./... -run 'TestLoadCurrentMappings|TestStaticMappingSmoke'
```

Отчет ИИ по шагу 7:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 8. Proxy fallback и delays

Цель: реализовать proxy mappings и WireMock-compatible задержки, сохраняя работоспособность обычных stubs.

Покрываемые требования: PROXY-001, PROXY-002, PROXY-003, RESP-012.

Сделать:

- Реализовать proxy request forwarding на `proxyBaseUrl`.
- Реализовать `proxyUrlPrefixToRemove`.
- Проверить, что proxy используется только после priority selection.
- Поддержать `fixedDelayMilliseconds`.
- Поддержать `delayDistribution`.
- Поддержать `chunkedDribbleDelay` настолько, насколько позволяет HTTP response writer.
- Добавить tests на fallback behavior и delays.

Не делать на этом шаге: recording generated mappings.

Проверка результата:

```bash
go test ./...
go test ./internal/proxy ./internal/response -run 'TestProxy|TestDelay'
```

Отчет ИИ по шагу 8:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 9. Stateful scenarios

Цель: реализовать scenario state machine и обеспечить безопасность при параллельных запросах.

Покрываемые требования: SCN-001, SCN-002, SCN-003, SCN-004, SCN-005, SCN-006.

Сделать:

- Добавить scenario state store.
- Начальное состояние scenario: `Started`.
- Учитывать `requiredScenarioState` при matching.
- Выполнять transition в `newScenarioState` после serve.
- Сделать операции state transition атомарными относительно matching.
- Добавить Admin/internal reset hook для scenario state.
- Добавить тесты на сценарии из `tds-api`/`tds-ui` и race tests.

Не делать на этом шаге: GUI state inspection.

Проверка результата:

```bash
go test ./...
go test -race ./internal/scenario ./internal/runtime
```

Отчет ИИ по шагу 9:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 10. Runtime-generated workflow автотестов

Цель: workflow из `autotests-example/utils/wiremock_generator/generator_utils.py` должен работать без специальных обходов.

Покрываемые требования: RT-001, RT-002, RT-003, RT-005, ADM-014, ACC-006.

Сделать:

- Проверить `POST /__admin/mappings` для generated mocks без `id`.
- Проверить response содержит `id`.
- Проверить `DELETE /__admin/mappings/{id}` и 404 semantics.
- Проверить повторную загрузку static mocks через name+folder lookup и `PUT`.
- Добавить end-to-end test, имитирующий PDM, Shcat, Officer, Susanin, Vanga, Courier/Frodo, Fry generated mocks.
- Проверить `POST /__admin/ext/grpc/reset` после PDM generated mapping.

Не делать на этом шаге: запуск реальных Python автотестов целиком, если для этого нужны внешние сервисы.

Проверка результата:

```bash
go test ./... -run 'TestRuntimeGeneratedWorkflow|TestAutotestMappingLifecycle'
```

Отчет ИИ по шагу 10:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 11. gRPC descriptor registry и transport base

Цель: подготовить gRPC слой: descriptor/proto files загружаются через Admin API и legacy file upload остается совместимым.

Покрываемые требования: PROTO-003, PROTO-004, GRPC-001, GRPC-002, GRPC-011, GRPC-012, GRPC-013, GRPC-014, GRPC-015, GRPC-016, GRPC-017, GRPC-018, FILE-011.

Сделать:

- Добавить HTTP/2/gRPC listener configuration.
- Реализовать descriptor registry.
- Реализовать `PUT /__admin/ext/grpc/descriptors/{fileName}`.
- Реализовать `GET /__admin/ext/grpc/descriptors`.
- Реализовать `DELETE /__admin/ext/grpc/descriptors/{fileName}`.
- Реализовать `POST /__admin/ext/grpc/reset` как registry reload.
- Связать legacy `.dsc` upload с descriptor registry.
- Добавить tests на upload/list/delete/reset и invalid descriptors.

Не делать на этом шаге: protobuf request/response conversion.

Тонкости реализации из WireMock gRPC extension:

- Descriptor storage должен быть отдельным namespace/store `grpc`; Java extension сканирует только ключи с `.dsc` и `.desc`.
- `.dsc`/`.desc` трактуются как protobuf `FileDescriptorSet`; при загрузке нужно построить `FileDescriptor` с учетом зависимостей между файлами.
- `TypeRegistry` должен строиться из всех message types загруженных descriptors; он нужен для JSON conversion и `google.protobuf.Any`.
- `POST /__admin/ext/grpc/reset` должен атомарно перечитывать все descriptor blobs и заменять active registry целиком.
- Если после reset сервис/метод больше не присутствует в descriptors, вызов должен завершаться gRPC `UNIMPLEMENTED`.
- gRPC handler должен включаться только для настоящих gRPC requests; обычные HTTP requests продолжают идти через стандартный HTTP stub pipeline.
- WireMock extension добавляет server reflection поверх загруженных descriptors; для максимальной совместимости стоит реализовать reflection или явно зафиксировать как known gap до закрытия MVP.

Проверка результата:

```bash
go test ./...
curl -X PUT --data-binary @autotests-example/mocks/pdm/mc_product.dsc http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
curl http://localhost:8080/__admin/ext/grpc/descriptors
curl -X POST http://localhost:8080/__admin/ext/grpc/reset
```

Отчет ИИ по шагу 11:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 12. gRPC stubbing runtime

Цель: gRPC requests должны матчиться WireMock JSON mappings и возвращать protobuf responses.

Покрываемые требования: GRPC-003, GRPC-004, GRPC-005, GRPC-006, GRPC-007, GRPC-008, GRPC-009, GRPC-010, MATCH-011, ACC-007.

Сделать:

- Маршрутизировать gRPC call в mapping по `/<fully-qualified service>/<method>`.
- Конвертировать request protobuf в JSON body для matcher pipeline.
- Использовать существующие JSON matchers и templating.
- Конвертировать JSON response body в protobuf response.
- Поддержать `grpc-status-name` и non-OK statuses.
- Добавить sample proto/descriptor и gRPC client test.
- Добавить contract-style tests по примерам WireMock gRPC docs.

Не делать на этом шаге: server reflection, если оно не нужно для MVP.

Тонкости реализации из WireMock gRPC extension:

- gRPC call адаптируется в обычный WireMock request: `method=POST`, `url=/<fully-qualified service>/<method>`, `protocol=HTTP/2`, body = protobuf JSON.
- Request protobuf конвертируется в JSON через protobuf JSON mapping, а response JSON парсится обратно в output message descriptor.
- Response `grpc-status-name=OK` не обязателен для успешного ответа, но должен поддерживаться; если header отсутствует и HTTP status = 200, response body парсится как OK message.
- Если `grpc-status-name` присутствует и не `OK`, body игнорируется, а клиент получает gRPC error с кодом из header и reason из `grpc-status-reason`.
- Если `grpc-status-name` отсутствует, HTTP statuses мапятся в gRPC errors: `400 -> INTERNAL`, `401 -> UNAUTHENTICATED`, `403 -> PERMISSION_DENIED`, `404 -> UNIMPLEMENTED`, `429/502/503/504 -> UNAVAILABLE`.
- Unmatched gRPC request должен возвращать `UNIMPLEMENTED` с сообщением `No matching stub mapping found for gRPC request`.
- gRPC metadata headers должны попадать в общий matcher/template model как request headers; binary metadata `*-bin` в Java extension превращается в строку вида byte array.
- `response-template` должен работать без отдельного gRPC templating layer, потому что gRPC response до protobuf encoding остается обычным JSON body.
- Delays применяются до отправки gRPC response; fixed и random delay должны работать так же, как для HTTP response.
- Client-streaming в extension реализован упрощенно: используется первый matching request message, unmatched 404 пропускаются до конца stream, если не найдено ни одного match - `UNIMPLEMENTED`.
- Server-streaming в extension фактически возвращает один response message; true multi-response stream не закладывать в MVP без отдельного требования.
- `google.protobuf.Any` требует `@type: "type.googleapis.com/<full-message-name>"` в JSON response.
- Для raw mappings с binary body matcher Java extension конвертирует `binaryEqualTo` в `equalToJson` при наличии gRPC response header; VIMock должен поддержать это только если появятся такие mappings.

Проверка результата:

```bash
go test ./... -run 'TestGRPC'
go run ./cmd/vimock
# Запустить sample gRPC client из test fixture
```

Отчет ИИ по шагу 12:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 13. GraphQL semantic matcher

Цель: поддержать WireMock GraphQL extension-compatible matching поверх HTTP.

Покрываемые требования: PROTO-005, GQL-001, GQL-002, GQL-003, GQL-004, GQL-005, GQL-006, GQL-007, GQL-008, GQL-009, GQL-010, GQL-011, ACC-008.

Сделать:

- Добавить GraphQL parser/normalizer.
- Реализовать semantic query matching.
- Игнорировать whitespace/formatting.
- Нормализовать порядок полей там, где это допустимо GraphQL-семантикой.
- Реализовать variables matching через JSON matching logic.
- Поддержать JSON/Admin API эквивалент `GraphqlBodyMatcher`.
- Прогнать tests с одинаковыми queries в разном формате и с variables.
- Проверить, что response идет через общий response pipeline.

Не делать на этом шаге: GraphQL federation-specific features, если они не появятся в требованиях.

Тонкости реализации из WireMock GraphQL extension:

- Совместимый JSON API использует `request.customMatcher.name = "graphql-body-matcher"`.
- `customMatcher.parameters.query` обязателен; `variables` и `operationName` опциональны.
- Incoming request body должен быть JSON object с полем `query` и опциональными `variables`, `operationName`.
- Если expected `variables` не заданы, request с любым `variables` должен не матчиться; то же правило для `operationName`.
- Query matching делается через parse GraphQL AST, сортировку AST и structural comparison; порядок полей и whitespace игнорируются.
- Missing/additional/different fields, отличающиеся aliases и отличающиеся arguments должны давать no match.
- Variables matching использует WireMock `EqualToJsonPattern` с `ignoreArrayOrder=false` и `ignoreExtraElements=false`: порядок object keys не важен, порядок array elements важен.
- Invalid JSON или invalid GraphQL query при matching должны давать no match с диагностикой, а не падение процесса.
- GraphQL matcher должен оставаться дополнительным matcher-ом после стандартных `method/url` checks, как WireMock `customMatcher`, а не заменять HTTP routing.
- Fixtures из `wiremock-graphql-extension/e2e/fixtures` стоит использовать как acceptance cases: order, aliases, arguments, fragments, variables, array order, invalid input.

Проверка результата:

```bash
go test ./... -run 'TestGraphQL'
```

Отчет ИИ по шагу 13:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 14. Recording и snapshotting

Цель: реализовать WireMock-compatible record/playback API и snapshot API.

Покрываемые требования: REC-001, REC-002, REC-003, REC-004, REC-005, REC-006, REC-007, REC-008, REC-009, REC-010, REC-011.

Сделать:

- Хранить serve events для snapshotting.
- Реализовать `POST /__admin/recordings/start`.
- Поддержать `targetBaseUrl` и record spec fields.
- Проксировать записываемые запросы к target service.
- Реализовать `POST /__admin/recordings/stop`.
- Создавать recorded mappings и активировать их после stop.
- Реализовать `POST /__admin/recordings/snapshot`.
- Поддержать `outputFormat`, `extractBodyCriteria`, `captureHeaders`, `requestBodyPattern`, `repeatsAsScenarios`, `persist`.
- Поддержать binary bodies через `base64Body` или body files.

Не делать на этом шаге: UI recorder page.

Тонкости реализации из WireMock gRPC extension:

- gRPC proxy/recording должен записывать уже сконвертированный JSON, а не raw protobuf bytes.
- Recorded gRPC mapping должен выглядеть как обычный WireMock mapping: `method=POST`, `urlPath=/<service>/<method>`, `bodyPatterns.equalToJson`, response body JSON, header `grpc-status-name=OK`.
- Для proxy к upstream gRPC сервису нужен gRPC-aware client path: при `Content-Type: application/grpc` делать реальный gRPC call по descriptor context, затем вернуть в recorder JSON body и gRPC status headers.
- Ошибки upstream gRPC должны попадать в recorded/proxied response как `grpc-status-name` и `grpc-status-reason`, с HTTP status по reverse mapping там, где он задан.
- Recording gRPC в MVP можно ограничить unary calls, если streaming recording явно не нужен.

Проверка результата:

```bash
go test ./... -run 'TestRecording|TestSnapshot'
curl -X POST http://localhost:8080/__admin/recordings/start -d '{"targetBaseUrl":"http://127.0.0.1:9000"}' -H 'Content-Type: application/json'
curl -X POST http://localhost:8080/__admin/recordings/stop
```

Отчет ИИ по шагу 14:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 15. HTTPS, HTTP/2, Docker/CI/Kubernetes readiness и performance baseline

Цель: подготовить сервис к реальной эксплуатации 100+ процессов и CI/Kubernetes запуску.

Покрываемые требования: PROTO-002, CON-004, CON-005, CON-006, NFR-001, NFR-002, NFR-005, NFR-006, TEST-006.

Сделать:

- Добавить TLS config для HTTPS.
- Проверить HTTP/2 поверх TLS.
- Добавить graceful shutdown.
- Уточнить Docker image: non-root user, healthcheck, minimal image.
- Добавить Kubernetes-ready probes documentation.
- Добавить benchmark tests для matching и response pipeline.
- Проверить memory behavior на большом наборе mappings.
- Устранить избыточные копирования payloads на hot path.

Не делать на этом шаге: Kubernetes manifests, если они не нужны для MVP acceptance.

Проверка результата:

```bash
go test ./...
go test -bench=. ./...
docker build -t vimock:dev .
```

Отчет ИИ по шагу 15:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 16. Финальная приемка MVP и quality gate

Цель: закрыть все MUST requirements, подтвердить 90% coverage и подготовить итоговый compliance report.

Покрываемые требования: TEST-001, TEST-002, TEST-003, TEST-004, TEST-005, ACC-001, ACC-002, ACC-003, ACC-004, ACC-005, ACC-006, ACC-007, ACC-008, ACC-009, ACC-010.

Сделать:

- Прогнать все unit tests.
- Прогнать race tests.
- Прогнать fixture tests на `examples` и `autotests-example/mocks`.
- Прогнать contract tests по WireMock-compatible фичам.
- Прогнать gRPC docs compatibility tests.
- Прогнать GraphQL docs compatibility tests.
- Проверить coverage >= 90%.
- Создать финальный compliance report по всем requirement IDs из `tz.md`.
- Обновить README с командами запуска, Docker, API examples и known limitations.

Не делать на этом шаге: добавлять новые функциональные фичи, кроме исправления блокеров acceptance.

Тонкости приемки из WireMock extensions:

- gRPC acceptance должен покрыть descriptor upload/list/delete/reset, reload с заменой набора сервисов, unary JSON match/response, templated JSON response, non-OK statuses, HTTP-to-gRPC status mapping, request metadata headers, `Any`, delays и gRPC recording.
- GraphQL acceptance должен покрыть `graphql-body-matcher` JSON format, semantic query order, nested order, missing/additional/different fields, aliases, arguments, fragments, variables object order, variables array order mismatch, absent variables/operationName и invalid input.
- Для gRPC reflection принять решение до закрытия MVP: либо реализовать как extension, либо явно отметить known limitation и проверить, что текущие автотесты не зависят от reflection.
- Для streaming gRPC зафиксировать поддерживаемый минимум: client-streaming first-match и server-streaming single-message совместимость с extension, либо documented gap если это не требуется текущими моками.

Проверка результата:

```bash
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Отчет ИИ по шагу 16:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Known gaps:
Риски/решения:
```

## Шаг 17. Black-box автотесты API запущенного сервиса

Цель: добавить отдельный набор автотестов, который проверяет уже запущенный `vimock` через HTTP/gRPC/GraphQL/Admin API и подтверждает все фичи, найденные в `current-mocks.md` и `current-autotest.md`.

Покрываемые требования: TEST-003, TEST-004, TEST-005, ACC-001, ACC-002, ACC-003, ACC-004, ACC-005, ACC-006, ACC-007, ACC-008, ACC-010.

Сделать:

- Создать отдельную папку `autotest/`, независимую от unit-тестов приложения.
- Сделать запуск против уже поднятого сервиса через `VIMOCK_BASE_URL`.
- Добавить режим запуска сервиса из binary для локальной проверки, если `VIMOCK_BASE_URL` не задан.
- Загружать mappings из `examples/**/*.json` и `autotests-example/mocks/**/*.json` только через Admin API, не через internal packages.
- Проверять feature matrix из `current-mocks.md`: URL/method matching, priority, headers/query/body matchers, JSONPath, `equalToJson`, response templating, `bodyFileName`, binary bodies, proxy, delays, scenarios, recording.
- Проверять workflow из `current-autotest.md`: bootstrap mappings, legacy `/api/login`, `/api/tus`, `.dsc` upload, `POST /__admin/ext/grpc/reset`, runtime create/update/delete mappings, name+folder lookup, generated mappings.
- Добавить gRPC black-box tests: descriptor upload через Admin API, reset registry, unary request/response, JSON matching, templated response, status errors.
- Добавить GraphQL black-box tests: `graphql-body-matcher`, variables, operationName, semantic query order, negative cases.
- Добавить проверку request journal/Admin API там, где автотесты используют WireMock verification-like behavior.
- Сделать тестовые fixtures явными: каждый testcase должен ссылаться на источник из `current-mocks.md` или `current-autotest.md`.
- Добавить machine-readable отчет, например `autotest/reports/features.json` или JUnit XML, чтобы видеть покрытие фич в CI.
- Добавить README в `autotest/` с командами запуска локально, против Docker и против Kubernetes/CI endpoint.

Не делать на этом шаге: запускать реальные продуктовые автотесты целиком и ходить во внешние сервисы; этот набор должен проверять только VIMock API и его WireMock-compatible поведение.

Тонкости реализации:

- Эти автотесты должны быть black-box: запрещено импортировать internal Go packages VIMock.
- Тесты должны уметь работать против `go run ./cmd/vimock`, собранного binary, Docker container и удаленного CI/Kubernetes URL.
- Набор должен быть feature-driven, а не file-count-driven: не обязательно дергать каждый из 291 mappings реальным payload, но каждая уникальная WireMock-фича из `current-mocks.md` должна иметь хотя бы один исполняемый testcase.
- Для mappings без очевидного request payload нужно фиксировать статус `covered-by-load-only` или `requires-fixture`, чтобы не создавать ложное ощущение полного behavioral coverage.
- Для proxy/recording использовать локальные stub upstream servers, запускаемые самими автотестами.
- Для gRPC/GraphQL использовать минимальные test descriptors/schemas и дополнительно проверять совместимость JSON mapping syntax из extension docs.
- В CI эти тесты должны запускаться после unit/contract suite и после сборки binary/container.

Проверка результата:

```bash
go test ./...
go build -o ./bin/vimock ./cmd/vimock
./bin/vimock --port 8080
cd autotest
VIMOCK_BASE_URL=http://localhost:8080 go test ./...
```

Отчет ИИ по шагу 17:

```text
Статус: TODO
Сделано:
Измененные файлы:
Как запускать:
Проверки и результаты:
Покрытые требования:
Feature coverage из current-mocks.md:
Feature coverage из current-autotest.md:
Known gaps:
Риски/решения:
```

## Полное покрытие требований по шагам

| Шаг | Requirement IDs |
|---|---|
| 1 | CON-002, CON-003, CON-004, CON-005, CON-006, PROTO-001, TEST-001, RESP-013, RESP-014, OUT-001, OUT-002, OUT-003, OUT-004, OUT-005, ACC-010 |
| 2 | CON-001, CON-007, CON-008, MAP-001, MAP-002, MAP-003, MAP-004, ADM-001, ADM-002, ADM-003, ADM-004, ADM-005, ADM-006, ADM-007, ADM-008, ADM-009, ADM-010, ADM-011, ADM-012, ADM-013, ADM-014, ADM-015, NFR-001, NFR-003, NFR-004, TEST-002 |
| 3 | MAP-005, MAP-006, MAP-007, MAP-008, MAP-009, RESP-001, RESP-002, RESP-003, RESP-004, RESP-010, NFR-002, NFR-005 |
| 4 | MATCH-001, MATCH-002, MATCH-003, MATCH-004, MATCH-005, MATCH-006, MATCH-007, MATCH-008, MATCH-009, MATCH-010, MATCH-011, MATCH-012, JRPC-001, TEST-003 |
| 5 | RESP-005, RESP-006, RESP-007, RESP-008, RESP-009, RESP-011, JRPC-002, JRPC-003, FILE-001, FILE-009, FILE-010 |
| 6 | FILE-002, FILE-003, FILE-004, FILE-005, FILE-006, FILE-007, FILE-008, GRPC-018, ACC-005 |
| 7 | ACC-001, ACC-002, ACC-003, ACC-004, RT-004, TEST-003 |
| 8 | PROXY-001, PROXY-002, PROXY-003, RESP-012 |
| 9 | SCN-001, SCN-002, SCN-003, SCN-004, SCN-005, SCN-006 |
| 10 | RT-001, RT-002, RT-003, RT-005, ADM-014, ACC-006 |
| 11 | PROTO-003, PROTO-004, GRPC-001, GRPC-002, GRPC-011, GRPC-012, GRPC-013, GRPC-014, GRPC-015, GRPC-016, GRPC-017, GRPC-018, FILE-011 |
| 12 | GRPC-003, GRPC-004, GRPC-005, GRPC-006, GRPC-007, GRPC-008, GRPC-009, GRPC-010, MATCH-011, ACC-007 |
| 13 | PROTO-005, GQL-001, GQL-002, GQL-003, GQL-004, GQL-005, GQL-006, GQL-007, GQL-008, GQL-009, GQL-010, GQL-011, ACC-008 |
| 14 | REC-001, REC-002, REC-003, REC-004, REC-005, REC-006, REC-007, REC-008, REC-009, REC-010, REC-011 |
| 15 | PROTO-002, CON-004, CON-005, CON-006, NFR-001, NFR-002, NFR-005, NFR-006, TEST-006 |
| 16 | TEST-001, TEST-002, TEST-003, TEST-004, TEST-005, ACC-001, ACC-002, ACC-003, ACC-004, ACC-005, ACC-006, ACC-007, ACC-008, ACC-009, ACC-010 |
| 17 | TEST-003, TEST-004, TEST-005, ACC-001, ACC-002, ACC-003, ACC-004, ACC-005, ACC-006, ACC-007, ACC-008, ACC-010 |

## Контрольные вопросы для ИИ перед закрытием каждого шага

- Можно ли запустить `vimock` после этого шага?
- Есть ли команда, которой пользователь может пощупать результат вручную?
- Не сломаны ли уже реализованные требования предыдущих шагов?
- Добавлены ли unit/integration tests на новую функциональность?
- Не добавлены ли фичи вне scope текущего шага?
- Заполнен ли отчет ИИ по шагу?
- Если шаг касается приемки, есть ли black-box проверка через публичный API без импорта internal packages?
