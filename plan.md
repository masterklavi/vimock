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
| 4 | `vimock` матчится по текущим request matchers | JSONPath, absent, query/header equalTo, equalToJson foundation | Stable matcher tests |
| 5 | `vimock` умеет response pipeline текущих моков | response-template, jsonPath helper, bodyFileName, binary body, JSON-RPC id | Моки с `{{jsonPath}}`, PDF/bin bodyFileName |
| 6 | `vimock` совместим с bootstrap автотестов по файлам | Legacy file API `/api/login`, `/api/tus`, `.dsc`, grpc reset no-op/reload hook | Legacy upload workflow проходит |
| 7 | `vimock` прошел одноразовую проверку временного набора mappings | Проверка совместимости без постоянной зависимости от временных fixture-папок | Результат зафиксирован в отчете шага |
| 8 | `vimock` умеет proxy fallback и delays | `proxyBaseUrl`, `proxyUrlPrefixToRemove`, fixed/random/chunked delays | Stub fallback через proxy, delay tests |
| 9 | `vimock` поддерживает scenarios | Scenario state engine, Started, transitions, concurrent safety | Stateful mapping меняет response |
| 10 | `vimock` проходит runtime-generated workflow автотестов | Runtime create/use/delete, name+folder update flow, PDM reset behavior | Runtime-generated workflow проходит |
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
curl -X POST http://localhost:8080/__admin/mappings -d @testdata/simple_body_mapping.json -H 'Content-Type: application/json'
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
- `curl -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @testdata/simple_body_mapping.json`
- `curl http://localhost:8080/__admin/mappings`
- `curl -X PUT http://localhost:8080/__admin/mappings/{id} -H 'Content-Type: application/json' --data-binary @mapping.json`
- `curl -X DELETE http://localhost:8080/__admin/mappings/{id}`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 78.7%.
- Ручная проверка `POST /__admin/mappings` на `testdata/simple_body_mapping.json` - HTTP 201, сгенерирован UUID, возвращены `name`, `request`, `response`.
- Ручная проверка `GET /__admin/mappings` после POST - HTTP 200, `mappings[0]` содержит загруженный mapping, `meta.total=1`.
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
Статус: DONE
Сделано:
- Добавлен runtime router для всех не-Admin HTTP requests через fallback handler `/`.
- Admin paths `/__admin` и `/__admin/*` защищены от попадания в runtime matching и для неизвестных Admin routes возвращают обычный 404.
- В `mapping.Mapping` добавлена типизированная runtime-часть: `priority`, `request.method`, `request.url`, `request.urlPath`, `request.urlPattern`, `response.status`, `response.headers`, `response.body`, `response.jsonBody`.
- Для `request.method` поддержаны `ANY`, `GET`, `POST`.
- Для `request.url` реализован exact match по path+query.
- Для `request.urlPath` реализован exact match только по path, query игнорируется.
- Для `request.urlPattern` реализован full regex match по path+query, как WireMock Java `Pattern.matcher(value).matches()`.
- Реализован выбор mapping-а по меньшему `priority`; если priority равен, используется порядок добавления mapping-а.
- Реализован fallback proxy mapping selection: mapping с `priority=10` проигрывает более приоритетным точным stubs и выбирается только если они не подошли.
- Реализован response writer для `status`, `headers`, `body`, `jsonBody`; для `jsonBody` без явного `Content-Type` выставляется `application/json`.
- Реализован WireMock-like 404 для unmatched request без mappings: text/plain body `No response could be served as there are no stub mappings in this WireMock instance.`
- Добавлен `testdata/simple_body_mapping.json` для ручной проверки команды из плана.
- Добавлены `httptest` tests на `url`, `urlPath`, `urlPattern`, `ANY`, priority, insertion-order tie-breaker, удаление mapping-а и no-mappings 404.

Измененные файлы:
- `internal/mapping/model.go`
- `internal/server/runtime.go`
- `internal/server/runtime_test.go`
- `internal/server/server.go`
- `testdata/simple_body_mapping.json`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @testdata/simple_body_mapping.json`
- `curl -i http://localhost:8080/some/path`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 71.3%.
- Ручная проверка `POST /__admin/mappings` на `testdata/simple_body_mapping.json` - HTTP 201, mapping загружен.
- Ручная проверка `GET /some/path` после загрузки mapping-а - HTTP 200, `Content-Type: text/plain`, body `hello from vimock`.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- MAP-005, MAP-006, MAP-007, MAP-008, MAP-009, RESP-001, RESP-002, RESP-003, RESP-004, RESP-010, NFR-002, NFR-005.

Known gaps:
- Body/query/header matchers не реализованы, это scope шага 4.
- `response-template`, request-based response values и body files не реализованы, это scope шага 5.
- `response.proxyBaseUrl` на этом шаге еще не выполнял upstream proxying; закрыто на шаге 8.
- WireMock near-miss diff для unmatched request пока не реализован; возвращается простой WireMock-like 404.
- Общий coverage пока 71.3%; требование 90% остается финальным quality gate.

Риски/решения:
- WireMock использует Java regex, VIMock сейчас использует Go RE2. Сейчас покрыты простые `urlPattern` формы (`/prefix/.*` или точный путь). Java-only regex конструкции нужно отдельно валидировать, если появятся в стабильных требованиях.
- Runtime matching читает immutable snapshot из store и не блокирует Admin API writes дольше, чем требуется для copy-on-write публикации нового snapshot.
- BodyPatterns пока намеренно игнорируются; это может делать matching шире для текущих сложных моков до выполнения шага 4.
- В sandbox bind/listen, curl к локальному порту и Docker daemon требуют elevated execution; в обычном локальном окружении эти команды должны выполняться без дополнительных прав.
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
- Добавить representative stable tests на matcher patterns из требований.

Не делать на этом шаге: полный JSONUnit compatibility сверх требований текущих моков.

Проверка результата:

```bash
go test ./...
go test ./internal/matcher -run TestCurrentMockPatterns
```

Отчет ИИ по шагу 4:

```text
Статус: DONE
Сделано:
- Добавлен пакет `internal/matcher` с минимальным JSONPath evaluator под формы, которые используются в зафиксированных требованиях.
- Поддержан string `bodyPatterns.matchesJsonPath`.
- Поддержан object `bodyPatterns.matchesJsonPath` с `expression` и `absent=true`.
- Поддержаны JSONPath path segments: поля, array index, wildcard `*`.
- Поддержаны JSONPath filters `?()` с equality по строкам, числам, bool, `null` и массивам.
- Поддержаны `.size()` checks для массивов, объектов и строк.
- Поддержаны `request.queryParameters.*.equalTo`.
- Поддержаны `request.headers.*.equalTo`, включая `Content-Type: application/protobuf`.
- Поддержан базовый `bodyPatterns.equalToJson` как foundation для gRPC/recording.
- Runtime matching теперь учитывает method/url/query/headers/bodyPatterns вместе.
- Body JSON parsing оптимизирован: body парсится лениво и не более одного раза на HTTP request, затем переиспользуется всеми body matchers.
- Добавлен `testdata/matcher_mapping.json` для ручной проверки body/query/header matching.
- Обновлены README и docs по шагу 4.

Измененные файлы:
- `internal/matcher/jsonpath.go`
- `internal/matcher/request.go`
- `internal/matcher/jsonpath_test.go`
- `internal/mapping/model.go`
- `internal/server/runtime.go`
- `internal/server/runtime_test.go`
- `testdata/matcher_mapping.json`
- `README.md`
- `docs/README.md`
- `docs/step-04-request-matching.md`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @testdata/matcher_mapping.json`
- `curl -i -X POST 'http://localhost:8080/matchers?date=2025-10-14' -H 'Content-Type: application/json' --data '{"params":{"providers":["provider-1"]}}'`
- `go test ./internal/matcher -run TestCurrentMockPatterns`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./internal/matcher -run TestCurrentMockPatterns` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 74.8%.
- Ручная проверка `POST /__admin/mappings` на `testdata/matcher_mapping.json` - HTTP 201.
- Ручная проверка matching request `POST /matchers?date=2025-10-14` с `Content-Type: application/json` и body `{"params":{"providers":["provider-1"]}}` - HTTP 200, body `matched by body query header`.
- Ручная negative проверка с body `{"params":{"providers":["other"]}}` - HTTP 404.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- MATCH-001, MATCH-002, MATCH-003, MATCH-004, MATCH-005, MATCH-006, MATCH-007, MATCH-008, MATCH-009, MATCH-010, MATCH-011, MATCH-012, JRPC-001, TEST-003.

Known gaps:
- Полная JSONPath compatibility сверх текущих моков не реализована.
- Полная JSONUnit compatibility для `equalToJson` не реализована.
- Response templating и helper `{{jsonPath request.body '...'}}` не реализованы, это scope шага 5.
- Body files не реализованы, это scope шага 5.
- Общий coverage пока 74.8%; требование 90% остается финальным quality gate.

Риски/решения:
- JSONPath реализован как targeted evaluator под текущие fixture patterns, чтобы не тянуть внешний runtime dependency и не расширять scope шага.
- Если в новых моках появятся Java JSONPath/JsonUnit возможности за пределами текущих паттернов, fixture parsing/matcher tests должны подсветить это и evaluator нужно будет расширить.
- Для производительности body JSON parsing выполняется лениво один раз на request и переиспользуется всеми candidate mappings.
- В sandbox bind/listen, curl к локальному порту и Docker daemon требуют elevated execution; в обычном локальном окружении эти команды должны выполняться без дополнительных прав.
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
Статус: DONE

Сделано:
- Добавлен response rendering pipeline отдельно от HTTP runtime.
- Добавлен in-memory file storage abstraction для будущих body files и upload API.
- Поддержан parsing `response.bodyFileName` и `response.transformers`.
- Реализован lookup `bodyFileName` через file store.
- Binary body files отдаются как bytes без text/json перекодирования.
- Реализован targeted `response-template` для helper `{{jsonPath request.body '...'}}`.
- Реализован JSON-RPC-style echo `id`/`requestId` из request body в `jsonBody`.
- Добавлено JSON string escaping для значений helper внутри `jsonBody`.
- Runtime теперь читает request body один раз, переиспользует его для matching и rendering.
- Добавлен пример `testdata/template_mapping.json`.
- Обновлены README и docs по шагу 5.

Измененные файлы:
- `internal/files/store.go`
- `internal/files/store_test.go`
- `internal/response/render.go`
- `internal/response/render_test.go`
- `internal/mapping/model.go`
- `internal/server/server.go`
- `internal/server/runtime.go`
- `internal/server/runtime_test.go`
- `testdata/template_mapping.json`
- `.gitignore`
- `README.md`
- `docs/README.md`
- `docs/step-04-request-matching.md`
- `docs/step-05-response-templating-and-body-files.md`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @testdata/template_mapping.json`
- `curl -i -X POST http://localhost:8080/template -H 'Content-Type: application/json' --data '{"id":"rpc-42","requestId":"req-42"}'`

Проверки и результаты:
- `go test ./...` - успешно.
- `go test -race ./...` - успешно.
- `go test ./internal/response -run TestTemplateAndBodyFiles` - успешно.
- `go test -coverprofile=coverage.out ./...` - успешно, total coverage 75.0%.
- Ручная проверка `POST /__admin/mappings` на `testdata/template_mapping.json` - HTTP 201.
- Ручная проверка `POST /template` с body `{"id":"rpc-42","requestId":"req-42"}` - HTTP 200, JSON response содержит `id=rpc-42` и `requestId=req-42`.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- RESP-005, RESP-006, RESP-007, RESP-008, RESP-009, RESP-011, JRPC-002, JRPC-003, FILE-001, FILE-009, FILE-010.

Known gaps:
- HTTP upload API файлов не реализован, это scope шага 6.
- Полная WireMock/Handlebars совместимость `response-template` не реализована; поддержан только helper `jsonPath request.body`.
- Persistent/static file loading не реализован; текущий file store in-memory.
- Template helper для JSON body рассчитан на вставку значений внутрь JSON string fields; более широкие raw JSON insertion cases потребуют отдельной совместимости.
- gRPC conversion не реализован, это последующие шаги.
- Общий coverage 75.0%; требование 90% остается финальным quality gate.

Риски/решения:
- Renderer вынесен в отдельный пакет, чтобы дальше добавить delay/proxy/recording без разрастания runtime handler.
- File store сразу copy-safe и mutex-protected, так как приложение будет обслуживать параллельные запросы.
- Body file не проходит через template pipeline, чтобы не портить binary payload.
- Request body читается один раз и передается в matcher/rendering через shared `BodyContext`, чтобы не делать повторный JSON parse на каждый candidate mapping.
```

## Шаг 6. Legacy File API для автотестов

Цель: пройти legacy workflow загрузки файлов из проанализированного bootstrap-кода автотестов.

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
curl -i -X POST 'http://localhost:8080/api/tus/mc_product.dsc?override=true' -H 'Tus-Resumable: 1.0.0' -H 'Upload-Length: 40026' -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' -H 'X-Auth: vimock-file-token'
curl -i -X PATCH 'http://localhost:8080/api/tus/mc_product.dsc?override=true' -H 'Content-Type: application/offset+octet-stream' -H 'Tus-Resumable: 1.0.0' -H 'Upload-Offset: 0' -H 'X-Auth: vimock-file-token' --data-binary @testdata/mc_product.dsc
```

Отчет ИИ по шагу 6:

```text
Статус: DONE

Сделано:
- Реализован `POST /api/login` для legacy file upload workflow.
- Реализован `POST /api/tus/{file}?override=true` для инициализации upload.
- Реализован `PATCH /api/tus/{file}?override=true` для загрузки bytes.
- Добавлена проверка `X-Auth` token на upload endpoints.
- Добавлен parsing `Upload-Metadata: filename <hex>`.
- Uploaded bytes сохраняются в общий in-memory `files.Store`.
- Проверено, что загруженный файл можно вернуть через `response.bodyFileName`.
- Добавлен `POST /__admin/ext/grpc/reset` как no-op compatibility hook с HTTP 200 и пустым телом.
- Добавлены tests на happy path из `upload_file_to_wiremock()` и validation errors.
- Добавлен реальный descriptor fixture `testdata/mc_product.dsc` для локального запуска upload-команд.
- Обновлены README и docs по шагу 6.

Измененные файлы:
- `internal/server/file_api.go`
- `internal/server/file_api_test.go`
- `internal/server/server.go`
- `internal/server/admin.go`
- `README.md`
- `docs/README.md`
- `docs/step-05-response-templating-and-body-files.md`
- `docs/step-06-legacy-file-api.md`
- `testdata/mc_product.dsc`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -i -X POST http://localhost:8080/api/login`
- `curl -i -X POST 'http://localhost:8080/api/tus/mc_product.dsc?override=true' -H 'Tus-Resumable: 1.0.0' -H 'Upload-Length: 40026' -H 'Upload-Metadata: filename 6d635f70726f647563742e647363' -H 'X-Auth: vimock-file-token'`
- `curl -i -X PATCH 'http://localhost:8080/api/tus/mc_product.dsc?override=true' -H 'Content-Type: application/offset+octet-stream' -H 'Tus-Resumable: 1.0.0' -H 'Upload-Offset: 0' -H 'X-Auth: vimock-file-token' --data-binary @testdata/mc_product.dsc`
- `curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset`

Проверки и результаты:
- `go test ./...` - успешно.
- `go test -race ./...` - успешно.
- `go test ./internal/server -run TestLegacyFileUploadWorkflow` - успешно.
- `go test ./internal/server -run 'TestLegacyFileUpload|TestUploadMetadata|TestValidateUpload'` - успешно.
- `go test -coverprofile=coverage.out ./...` - успешно, total coverage 75.9%, `internal/server` coverage 83.8%.
- Ручная проверка `POST /api/login` - HTTP 200, body `vimock-file-token`.
- Ручная проверка `POST /api/tus/mc_product.dsc?override=true` - HTTP 201, headers `Location`, `Tus-Resumable`, `Upload-Offset: 0`.
- Ручная проверка `PATCH /api/tus/mc_product.dsc?override=true` - HTTP 204, `Upload-Offset: 8`.
- Ручная проверка `POST /__admin/ext/grpc/reset` - HTTP 200, пустое тело.
- Ручная проверка uploaded file через mapping `bodyFileName=mc_product.dsc` - HTTP 200, bytes `0a 12 76 69 6d 6f 63 6b`.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- FILE-002, FILE-003, FILE-004, FILE-005, FILE-006, FILE-007, FILE-008, GRPC-018, ACC-005.

Known gaps:
- Полноценный TUS protocol не реализован; поддержан только workflow текущих автотестов.
- Non-zero upload offsets и chunk resume не поддержаны.
- Persistent/static file storage не реализован; uploaded files живут в памяти процесса.
- `/__admin/ext/grpc/reset` пока no-op hook, без descriptor registry reload.
- Native gRPC descriptor Admin API и gRPC stubbing остаются scope последующих шагов.
- Общий coverage 75.9%; требование 90% остается финальным quality gate.

Риски/решения:
- Token сделан константным, потому что текущие автотесты используют его только как bearer для `X-Auth`; полноценная auth модель не нужна для локального mock-сервиса.
- `Upload-Metadata` декодируется как hex, потому что так работает текущий legacy upload workflow; TUS base64 metadata можно добавить позже при реальной необходимости.
- `PATCH` перезаписывает файл целиком при `Upload-Offset: 0`, чтобы не моделировать resumable upload без требований.
- Upload пишет в тот же `files.Store`, который использует response renderer, чтобы этот storage дальше можно было переиспользовать для gRPC descriptors.
```

## Шаг 7. Одноразовая проверка временного набора mappings

Цель: одноразово проверить, что временный набор mappings загружается и не ломает сервис, но не оставлять в коде/документации постоянную зависимость от временных папок.

Покрываемые требования: ACC-001, ACC-002, ACC-003, ACC-004, RT-004, TEST-003.

Сделать:

- Одноразово прогнать временный набор mappings через тот же parser/model path, что Admin API.
- Проверить, что загрузка mappings не ломает сервис.
- Проверить representative runtime matching на временном наборе.
- Зафиксировать результат в отчете шага.
- Не оставлять постоянный тестовый код или документацию, завязанные на временные папки.

Не делать на этом шаге: добавлять постоянные regression tests на временные fixture-папки.

Проверка результата:

```bash
go test ./...
```

Отчет ИИ по шагу 7:

```text
Статус: DONE

Сделано:
- Одноразово прогнан временный набор mappings через Admin API load path.
- Подтверждено, что сервис принимает весь временный набор mappings без ошибок загрузки.
- Одноразово проверен representative runtime matching на временном наборе.
- После подтверждения результата удалены постоянный fixture-loader test и отдельная документация шага 7, чтобы не закреплять зависимость от временных папок.
- README и docs index не содержат ссылок на долговременные fixture checks временных папок.

Измененные файлы:
- `docs/step-02-admin-api.md`
- `docs/step-04-request-matching.md`
- `docs/step-06-legacy-file-api.md`
- `internal/mapping/fixtures_test.go` удален
- `plan.md`

Как запускать:
- `go test ./...`
- `go test -race ./...`

Проверки и результаты:
- Одноразовый fixture run - успешно, total mappings checked: 394.
- Unsupported runtime fields report: `newScenarioState=4`, `requiredScenarioState=4`, `response.proxyBaseUrl=21`, `response.proxyUrlPrefixToRemove=21`, `scenarioName=4`.
- Одноразовый representative smoke run - успешно, 11 mappings matched through HTTP handler.
- `go test ./...` - успешно.
- `go test -race ./...` - успешно.
- `go test -coverprofile=coverage.out ./...` - успешно, total coverage 75.9%, `internal/server` coverage 83.8%.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- ACC-001, ACC-002, ACC-003, ACC-004, RT-004, TEST-003.

Known gaps:
- Runtime behavior для `response.proxyBaseUrl` и `response.proxyUrlPrefixToRemove` закрыт на шаге 8.
- Scenario state fields `scenarioName`, `requiredScenarioState`, `newScenarioState` сохраняются, но runtime scenario engine не реализован.
- Постоянный regression suite на временные fixture-папки намеренно не добавлен.
- Общий coverage 75.9%; требование 90% остается финальным quality gate.

Риски/решения:
- Проверка временного набора использовалась как одноразовый quality gate, а не как постоянный источник truth.
- Unsupported fields сохранены в отчете как input для следующих шагов, но не привязаны к конкретным временным файлам.
- Для постоянного качества вместо временных папок дальше нужны стабильные fixtures в `testdata` или black-box suite из шага 17.
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
go test ./internal/proxy ./internal/delay ./internal/server -run 'TestProxy|TestDelay|TestRuntimeProxies|TestRuntimeAppliesFixedDelay'
```

Отчет ИИ по шагу 8:

```text
Статус: DONE

Сделано:
- Добавлены typed response fields для `proxyBaseUrl`, `proxyUrlPrefixToRemove`, `fixedDelayMilliseconds`, `delayDistribution`, `chunkedDribbleDelay`.
- Добавлен пакет `internal/proxy` для построения upstream URL, prefix removal, forwarding request body/headers и чтения upstream response.
- Proxy удаляет hop-by-hop headers и возвращает upstream status/headers/body.
- Runtime теперь после priority selection выполняет proxy forwarding, если у выбранного stub есть `response.proxyBaseUrl`.
- Добавлен пакет `internal/delay` для fixed delay, uniform/lognormal delay distribution и chunked dribble interval.
- Runtime применяет initial delay перед обычным response или proxy response.
- Runtime пишет body chunks с задержкой для `chunkedDribbleDelay` и flush после каждого chunk, если writer поддерживает flush.
- Добавлены tests на proxy URL rewriting, request forwarding, invalid proxy base URL.
- Добавлены tests на fixed/uniform/lognormal delay calculation, chunked interval и context cancellation.
- Добавлены runtime tests на proxy fallback после priority selection и fixed delay + chunked dribble без реального sleep.
- Обновлены README и docs по шагу 8.

Измененные файлы:
- `internal/mapping/model.go`
- `internal/mapping/model_test.go`
- `internal/delay/delay.go`
- `internal/delay/delay_test.go`
- `internal/proxy/proxy.go`
- `internal/proxy/proxy_test.go`
- `internal/server/server.go`
- `internal/server/runtime.go`
- `internal/server/runtime_test.go`
- `README.md`
- `docs/README.md`
- `docs/step-08-proxy-and-delays.md`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- Загрузить mapping с `response.proxyBaseUrl` и `response.proxyUrlPrefixToRemove`, затем выполнить matching request.
- Загрузить mapping с `fixedDelayMilliseconds`, `delayDistribution` или `chunkedDribbleDelay`, затем выполнить matching request.
- `go test ./internal/proxy ./internal/delay ./internal/server -run 'TestProxy|TestDelay|TestRuntimeProxies|TestRuntimeAppliesFixedDelay'`
- `go test ./...`
- `go test -race ./...`

Проверки и результаты:
- `go test ./internal/delay ./internal/proxy ./internal/mapping ./internal/server -run 'TestInitialDuration|TestChunked|TestSleep|TestProxy|TestTargetURL|TestParseJSONResponseProxyAndDelays|TestRuntimeProxies|TestRuntimeAppliesFixedDelay'` - успешно.
- `go test ./...` - успешно.
- `go test -race ./...` - успешно.
- `go test -coverprofile=coverage.out ./...` - успешно, total coverage 72.8%.
- `docker build -t vimock:dev .` - успешно.

Покрытые требования:
- PROXY-001, PROXY-002, PROXY-003, RESP-012.

Known gaps:
- Proxy recording generated mappings не реализован, это scope шага 14.
- Proxy streaming не реализован: response body читается целиком в память перед отдачей клиенту.
- Full WireMock delay edge cases не покрыты: реализованы основные `fixedDelayMilliseconds`, `uniform`, `lognormal`, `chunkedDribbleDelay`.
- Общий coverage 72.8%; требование 90% остается финальным quality gate.

Риски/решения:
- Proxy выполняется только после выбора stub по priority/insertion order, поэтому fallback mapping не перехватывает более приоритетные stubs.
- Для тестов proxy используется fake `http.RoundTripper`, а не `httptest.NewServer`, чтобы тесты работали в sandbox без bind/listen permissions.
- Delay sleep в runtime инъектируется через `delay.Sleeper`, поэтому runtime tests проверяют delay contract без реального ожидания.
- Chunked dribble использует flush best-effort: если writer не поддерживает `http.Flusher`, chunks всё равно пишутся с задержкой, но фактическая доставка зависит от HTTP stack.
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
Статус: DONE
Сделано:
- Добавлены typed mapping fields для `scenarioName`, `requiredScenarioState`, `newScenarioState` с сохранением raw JSON в Admin API responses.
- Добавлен пакет `internal/scenario` с in-memory state store.
- Начальное состояние любого scenario считается `Started`.
- Runtime теперь сначала отбирает request-matching candidates, затем под scenario lock выбирает stub по `requiredScenarioState` и сразу выполняет transition в `newScenarioState`.
- Операция scenario matching + transition сделана атомарной относительно параллельных HTTP-запросов.
- Scenario state хранит только состояния, отличающиеся от виртуального `Started`, чтобы не раздувать память.
- Scenario store отслеживает создание/обновление/удаление scenario mappings через Admin API и удаляет stale state при удалении последнего mapping-а scenario.
- При `PUT /__admin/mappings/{id}` внутри того же scenario текущее состояние scenario сохраняется.
- Добавлен WireMock-compatible endpoint `POST /__admin/scenarios/reset`, который сбрасывает все scenarios в `Started` и возвращает `{}`.
- Добавлены unit tests на parsing scenario fields, state transitions, reset, удаление stale state и concurrent selection.
- Добавлены runtime tests на stateful цепочку, пропуск stub-а в неправильном state и admin reset.
- Обновлены README и docs по шагу 9.

Измененные файлы:
- `internal/mapping/model.go`
- `internal/mapping/model_test.go`
- `internal/scenario/store.go`
- `internal/scenario/store_test.go`
- `internal/server/admin.go`
- `internal/server/runtime.go`
- `internal/server/runtime_test.go`
- `internal/server/server.go`
- `README.md`
- `docs/README.md`
- `docs/step-09-stateful-scenarios.md`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- Создать mappings с `scenarioName`, `requiredScenarioState`, `newScenarioState`.
- Несколько раз вызвать один и тот же stub URL и проверить последовательность responses.
- `curl -i -X POST http://localhost:8080/__admin/scenarios/reset` для сброса всех scenario states.
- `go test ./internal/scenario ./internal/server -run 'TestSelectAndTransition|TestRuntimeSupportsStatefulScenarios|TestAdminResetsScenarioState'`
- `go test -race ./internal/scenario ./internal/server`
- `go test ./...`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./internal/scenario ./internal/mapping ./internal/server` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./internal/scenario ./internal/server` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 73.1%.

Покрытые требования:
- SCN-001, SCN-002, SCN-003, SCN-004, SCN-005, SCN-006.

Known gaps:
- `GET /__admin/scenarios` не реализован, потому что GUI/state inspection не входит в шаг 9.
- `PUT /__admin/scenarios/{name}/state` не реализован, потому что шаг требует reset hook, а ручная установка state не использовалась в текущих моках/автотестах.
- Endpoint `/__admin/mappings/reset` пока не реализован; при появлении он должен также вызывать scenario reset/sync.
- Общий coverage 73.1%; требование 90% остается финальным quality gate.

Риски/решения:
- Transition выполняется сразу после выбора stub-а, до rendering/proxy response. Это соответствует WireMock-подходу `onStubServed` после matching и не зависит от успешности записи response клиенту.
- Чтобы не держать scenario lock во время JSONPath/body matching, runtime сначала собирает request candidates без scenario state, затем атомарно выбирает stateful stub и меняет state.
- Scenario state не хранит полный список возможных states и не валидирует `newScenarioState` против него; это осознанное упрощение текущего шага, так как runtime mappings принимаются динамически и полная state inspection отложена.
```

## Шаг 10. Runtime-generated workflow автотестов

Цель: runtime-generated workflow из проанализированного кода автотестов должен работать без специальных обходов.

Покрываемые требования: RT-001, RT-002, RT-003, RT-004, RT-005, ADM-014, ACC-006.

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
Статус: DONE
Сделано:
- Добавлен in-process end-to-end test `TestRuntimeGeneratedWorkflow` для runtime-generated mapping lifecycle.
- Тест проверяет `POST /__admin/mappings` для generated mappings без `id` и наличие generated `id` в response.
- Тест проверяет, что runtime mapping активен сразу после создания и матчится через реальные request matchers/runtime response pipeline.
- Тест покрывает representative generated mappings для PDM/gRPC-compatible flow, ShCat, Officer, Susanin, Vanga, Courier/Frodo и Fry.
- Для PDM/gRPC-compatible generated mapping проверен `POST /__admin/ext/grpc/reset` после создания.
- Тест проверяет cleanup через `DELETE /__admin/mappings/{id}`.
- Тест проверяет repeated delete semantics: повторный `DELETE /__admin/mappings/{id}` возвращает `404`.
- Добавлен `TestAutotestMappingLifecycle`, который имитирует reload static mapping: `GET /__admin/mappings`, поиск по `name` и `metadata.wiremock-gui.folder`, затем `PUT /__admin/mappings/{id}`.
- Проверено, что после `PUT` обновленный mapping активен сразу.
- Representative generated mappings зафиксированы inline в тестах, без зависимости от временных fixture directories.
- Обновлены README и docs по шагу 10.

Измененные файлы:
- `internal/server/autotest_workflow_test.go`
- `README.md`
- `docs/README.md`
- `docs/step-10-runtime-generated-workflow.md`
- `plan.md`

Как запускать:
- `go test ./... -run 'TestRuntimeGeneratedWorkflow|TestAutotestMappingLifecycle'`
- `go test ./internal/server -run 'TestRuntimeGeneratedWorkflow|TestAutotestMappingLifecycle'`
- `go test ./...`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./internal/server -run 'TestRuntimeGeneratedWorkflow|TestAutotestMappingLifecycle' -count=1` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./... -run 'TestRuntimeGeneratedWorkflow|TestAutotestMappingLifecycle'` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - успешно.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go tool cover -func=coverage.out` - total coverage 73.1%.

Покрытые требования:
- RT-001, RT-002, RT-003, RT-004, RT-005, ADM-014, ACC-006.

Known gaps:
- Полный запуск внешнего набора автотестов не выполнялся, потому что шаг 10 ограничен in-process contract tests.
- Black-box API автотесты остаются scope шага 17.
- Общий coverage 73.1%; требование 90% остается финальным quality gate.

Риски/решения:
- Representative generated mappings держатся прямо в Go tests, чтобы не создавать постоянную зависимость от временных fixture directories.
- Тесты используют `httptest` handler, поэтому не требуют bind/listen permissions и работают стабильно в sandbox/CI.
- PDM/gRPC-compatible flow на этом шаге проверяет HTTP mapping lifecycle и reset hook; полноценное gRPC исполнение остается scope шагов 11-12.
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
curl -X PUT --data-binary @testdata/mc_product.dsc http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc
curl http://localhost:8080/__admin/ext/grpc/descriptors
curl -X POST http://localhost:8080/__admin/ext/grpc/reset
```

Отчет ИИ по шагу 11:

```text
Статус: DONE
Сделано:
- Включена transport base конфигурация HTTP/1.1, HTTP/2 и unencrypted HTTP/2 для будущего gRPC runtime.
- Добавлен in-memory gRPC descriptor registry с отдельным namespace от body files.
- Реализована загрузка, список и удаление descriptor files через Admin API.
- `.dsc` и `.desc` валидируются как protobuf FileDescriptorSet.
- `.proto` принимаются как UTF-8 source files, но пока не компилируются в active registry.
- `POST /__admin/ext/grpc/reset` атомарно пересобирает active registry из загруженных `.dsc`/`.desc` и сохраняет совместимый HTTP 200 с пустым телом.
- Active registry предоставляет lookup service/message type для последующего protobuf JSON runtime.
- Legacy upload `.dsc`/`.desc` теперь дополнительно синхронизирует валидные descriptor sets в gRPC registry, не ломая совместимость для невалидных legacy uploads.
- Добавлены unit/in-process API tests на upload/list/delete/reset, invalid descriptors и legacy bridge.
- README и docs обновлены по шагу 11.

Измененные файлы:
- `cmd/vimock/main.go`
- `go.mod`
- `go.sum`
- `internal/grpcdesc/store.go`
- `internal/grpcdesc/store_test.go`
- `internal/server/admin.go`
- `internal/server/file_api.go`
- `internal/server/server.go`
- `internal/server/grpc_descriptor_test.go`
- `README.md`
- `docs/README.md`
- `docs/step-11-grpc-descriptor-registry.md`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -i -X PUT --data-binary @testdata/mc_product.dsc http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc`
- `curl -s http://localhost:8080/__admin/ext/grpc/descriptors`
- `curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset`
- `curl -i -X DELETE http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./internal/grpcdesc ./internal/server` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - passed, total coverage 73.6%.

Покрытые требования:
- PROTO-003, PROTO-004: descriptor/proto upload API foundation добавлен.
- GRPC-001, GRPC-002: подготовлена gRPC/HTTP2 transport base и registry foundation.
- GRPC-011, GRPC-012, GRPC-013, GRPC-014, GRPC-015, GRPC-016, GRPC-017, GRPC-018: заложен descriptor registry, reload и type registry foundation для последующего gRPC JSON/protobuf runtime.
- FILE-011: legacy `.dsc`/`.desc` upload связан с descriptor registry.

Known gaps:
- Protobuf request/response conversion не реализован, это scope шага 12.
- gRPC service/method dispatch не реализован, это scope шага 12.
- gRPC reflection пока не реализован.
- `.proto` source compilation пока не реализован; `.proto` хранится и отображается в API как source, но не попадает в active registry.
- Если после reset сервис/метод отсутствует в descriptors, фактический gRPC `UNIMPLEMENTED` будет реализован вместе с runtime handler.

Риски/решения:
- Использован `google.golang.org/protobuf` как базовая зависимость для FileDescriptorSet, protodesc и dynamicpb type registry.
- Active registry заменяется целиком под mutex, чтобы reset был атомарным для будущих runtime readers.
- Invalid legacy descriptor uploads игнорируются registry, но не ломают legacy file workflow.
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
Статус: DONE
Сделано:
- Добавлен unary gRPC runtime поверх существующего `net/http` handler.
- gRPC requests распознаются по `POST` и `Content-Type: application/grpc`; обычный HTTP pipeline не изменен.
- Реализован route lookup по `/<fully-qualified service>/<method>` через active descriptor registry.
- Unary gRPC frame декодируется в protobuf payload; compressed messages пока явно отклоняются.
- Request protobuf декодируется через dynamic descriptors и конвертируется в protobuf JSON с proto field names для существующего matcher pipeline.
- Для matching переиспользуются текущие `urlPath`, method, headers, query и bodyPatterns.
- Binary metadata headers `*-bin` для matcher-а декодируются из base64 в byte-array строку вида `[1, 2, 3]`.
- Response renderer и `response-template` переиспользуются до protobuf encoding.
- JSON response body кодируется обратно в output protobuf message и возвращается как gRPC frame.
- Поддержаны `grpc-status-name` и `grpc-status-reason` как gRPC trailers.
- Поддержан fallback mapping HTTP statuses в gRPC statuses: 400, 401, 403, 404, 429, 502, 503, 504.
- Unmatched gRPC request возвращает `UNIMPLEMENTED` с сообщением `No matching stub mapping found for gRPC request`.
- Добавлена стабильная PDM fixture `testdata/grpc_mapping.json`.
- Добавлены in-process gRPC frame tests на PDM fixture, non-OK status, unmatched request и binary metadata matching.
- README и docs обновлены по шагу 12.

Измененные файлы:
- `internal/grpcdesc/store.go`
- `internal/server/runtime.go`
- `internal/server/server.go`
- `internal/server/grpc_runtime.go`
- `internal/server/grpc_runtime_test.go`
- `testdata/grpc_mapping.json`
- `README.md`
- `docs/README.md`
- `docs/step-11-grpc-descriptor-registry.md`
- `docs/step-12-grpc-stubbing-runtime.md`
- `plan.md`

Как запускать:
- `go run ./cmd/vimock`
- `curl -i -X PUT --data-binary @testdata/mc_product.dsc http://localhost:8080/__admin/ext/grpc/descriptors/mc_product.dsc`
- `curl -i -X POST http://localhost:8080/__admin/ext/grpc/reset`
- `curl -i -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @testdata/grpc_mapping.json`
- `curl -s http://localhost:8080/__admin/mappings`
- `grpcurl -plaintext -protoset testdata/mc_product.dsc -d '{"guids":["b27ed95d-3717-4538-9be6-a7136b8ad52f"]}' localhost:8080 pdm_api_gateway.v1.MCProduct/WarehousesByNomenclature`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./internal/grpcdesc ./internal/server` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - passed, total coverage 72.8%.

Покрытые требования:
- GRPC-003: gRPC method route maps to WireMock-compatible URL path.
- GRPC-004, GRPC-005: request protobuf conversion to JSON body implemented.
- GRPC-006: existing JSON matchers and response templating reused.
- GRPC-007: JSON response conversion to protobuf implemented.
- GRPC-008, GRPC-009, GRPC-010: OK/non-OK statuses, HTTP status fallback and unmatched `UNIMPLEMENTED` implemented.
- MATCH-011: binary metadata `*-bin` converted for header matching.
- ACC-007: representative PDM descriptor + mapping fixture covered by runtime test.

Known gaps:
- Только unary calls; client-streaming и server-streaming не реализованы.
- gRPC reflection не реализован.
- gRPC proxying и recording не реализованы.
- `.proto` source compilation не реализован.
- Compressed gRPC messages не поддерживаются.
- Тесты используют in-process gRPC frame client, потому что текущий sandbox запрещает listen на loopback; wire-level format при этом проверяется тем же handler path.

Риски/решения:
- Серверная реализация не тянет `grpc-go`: используется `net/http`, gRPC framing и `google.golang.org/protobuf` dynamic/protojson runtime.
- Для совместимости с текущими PDM mappings request JSON генерируется с `UseProtoNames=true`, чтобы поля были в snake_case.
- Active descriptor registry отдает TypeResolver для `google.protobuf.Any`; dedicated Any кейс пока не покрыт отдельным тестом.
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
Статус: DONE
Сделано:
- Добавлен WireMock-compatible GraphQL custom matcher `request.customMatcher.name = "graphql-body-matcher"`.
- Поддержан JSON/Admin API формат `customMatcher.parameters.query`, `variables`, `operationName`.
- Incoming GraphQL request body парсится как JSON object с `query`, опциональными `variables` и `operationName`.
- Query matching реализован через schema-less GraphQL parser + canonical structural representation.
- Whitespace и порядок selections/arguments/directives/fragments не влияют на match.
- Aliases, arguments, fragments, inline fragments, directives, variable definitions, lists и input objects учитываются структурно.
- Variables сравниваются как строгий JSON: порядок object keys не важен, порядок array elements важен.
- Если expected `variables` или `operationName` отсутствуют, request с этими полями не матчится.
- Invalid JSON или invalid GraphQL query дают no match, процесс не падает.
- Добавлена поддержка `request.urlPathPattern` для совместимости с WireMock GraphQL extension mappings.
- Response идет через общий HTTP response pipeline.
- Добавлен пример `testdata/graphql_mapping.json`.
- Добавлены unit tests для GraphQL matcher и runtime tests через обычный HTTP handler.
- README и docs обновлены по шагу 13.

Измененные файлы:
- `internal/matcher/graphql.go`
- `internal/matcher/graphql_test.go`
- `internal/mapping/model.go`
- `internal/server/graphql_runtime_test.go`
- `testdata/graphql_mapping.json`
- `README.md`
- `docs/README.md`
- `docs/step-13-graphql-semantic-matcher.md`
- `plan.md`

Как запускать:
- `curl -i -X POST http://localhost:8080/__admin/mappings -H 'Content-Type: application/json' --data-binary @testdata/graphql_mapping.json`
- `curl -i -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' -d '{"operationName":"GetHero","variables":{"episode":"JEDI"},"query":"query GetHero($episode: Episode) { hero(episode: $episode) { friends { name } age name } }"}'`
- `curl -i -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' -d '{"operationName":"GetHero","variables":{"episode":"EMPIRE"},"query":"query GetHero($episode: Episode) { hero(episode: $episode) { friends { name } age name } }"}'`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./internal/matcher ./internal/mapping ./internal/server` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - passed, total coverage 70.8%.

Покрытые требования:
- PROTO-005: GraphQL protocol matcher foundation добавлен.
- GQL-001, GQL-002, GQL-003: semantic query matcher, whitespace/order normalization реализованы.
- GQL-004, GQL-005, GQL-006: aliases, arguments и fragments учитываются.
- GQL-007, GQL-008: variables и operationName matching реализованы.
- GQL-009: compatible JSON/Admin API `graphql-body-matcher` поддержан.
- GQL-010: invalid input дает no match без падения процесса.
- GQL-011: response идет через общий response pipeline.
- ACC-008: representative GraphQL fixture и runtime flow покрыты тестом.

Known gaps:
- Matcher schema-less: GraphQL schema validation не выполняется.
- Federation-specific behavior не реализован.
- Parser покрывает синтаксис текущих compatibility fixtures, но не заявляет полную parity с `graphql-java` для всех редких GraphQL syntax edge cases.
- Диагностика invalid GraphQL пока не отдается как WireMock sub-events, только приводит к no match.

Риски/решения:
- Внешнюю GraphQL dependency не добавлял, чтобы не утяжелять бинарник и сборку; реализован небольшой parser под нужный compatibility scope.
- `urlPathPattern` добавлен как path-only regexp matcher, чтобы extension mappings с `/graphql` работали без переписывания.
- Новый parser увеличил объем кода и снизил общий coverage до 70.8%; 90% остается финальным quality gate.
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
Статус: DONE
Сделано:
- Добавлен in-memory recording store для active recording sessions и serve events.
- Реализован `POST /__admin/recordings/start`.
- Реализован `POST /__admin/recordings/stop`.
- Реализован `POST /__admin/recordings/snapshot`.
- Active recording проксирует unmatched HTTP requests к `targetBaseUrl`.
- `stop` строит mappings из записанных proxied events и активирует их в in-memory mapping store.
- `snapshot` строит mappings из накопленных serve events и активирует их.
- Serve events пишутся для обычных HTTP stub responses, proxy responses и matched unary gRPC responses.
- Поддержан `captureHeaders` для записи выбранных request headers как `request.headers.*.equalTo`.
- JSON request bodies записываются как `bodyPatterns.equalToJson`.
- JSON response bodies записываются как `jsonBody`.
- Text response bodies записываются как `body`.
- Binary response bodies записываются как `base64Body`.
- Mapping runtime теперь умеет читать `response.base64Body`.
- Request method matcher больше не ограничен только `GET`/`POST`, чтобы recorded mappings могли использовать другие HTTP methods.
- Добавлены примеры `testdata/recording_start.json` и `testdata/recording_snapshot.json`.
- README и docs обновлены по шагу 14.

Измененные файлы:
- `internal/recording/store.go`
- `internal/recording/store_test.go`
- `internal/mapping/model.go`
- `internal/server/admin.go`
- `internal/server/runtime.go`
- `internal/server/grpc_runtime.go`
- `internal/server/server.go`
- `internal/server/recording_test.go`
- `testdata/recording_start.json`
- `testdata/recording_snapshot.json`
- `README.md`
- `docs/README.md`
- `docs/step-14-recording-and-snapshotting.md`
- `plan.md`

Как запускать:
- `curl -i -X POST http://localhost:8080/__admin/recordings/start -H 'Content-Type: application/json' --data-binary @testdata/recording_start.json`
- `curl -i http://localhost:8080/api/products/123 -H 'X-Request-Id: req-1'`
- `curl -i -X POST http://localhost:8080/__admin/recordings/stop`
- `curl -s http://localhost:8080/__admin/mappings`
- `curl -i -X POST http://localhost:8080/__admin/recordings/snapshot -H 'Content-Type: application/json' --data-binary @testdata/recording_snapshot.json`
- `curl -i -X POST http://localhost:8080/__admin/recordings/start -H 'Content-Type: application/json' -d '{"targetBaseUrl":"https://time.now"}'`
- `curl -i http://localhost:8080/developer/api/ip`
- `curl -i -X POST http://localhost:8080/__admin/recordings/stop`
- `curl -i http://localhost:8080/developer/api/ip`

Проверки и результаты:
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test ./...` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -race ./internal/recording ./internal/server ./internal/mapping` - passed.
- `GOCACHE=/Users/vseiinstrumentyru/GolandProjects/vimock/.gocache go test -coverprofile=coverage.out ./...` - passed, total coverage 69.9%.

Покрытые требования:
- REC-001: serve events хранятся in-memory.
- REC-002, REC-003: `recordings/start` и `targetBaseUrl` реализованы.
- REC-004: active recording проксирует unmatched HTTP requests.
- REC-005, REC-006: `recordings/stop` создает и активирует recorded mappings.
- REC-007: `recordings/snapshot` реализован.
- REC-008, REC-009: `captureHeaders`, `requestBodyPattern`, `persist` поддержаны в рабочем объеме; остальные spec fields принимаются как foundation.
- REC-010: binary response bodies записываются через `base64Body`.
- REC-011: generated mappings compatible с текущим Admin/runtime parser.

Known gaps:
- `outputFormat`, `extractBodyCriteria`, `repeatsAsScenarios` принимаются в spec, но не имеют полной WireMock parity.
- Persistent filesystem output не реализован; все mappings и events in-memory.
- Active recording проксирует только unmatched HTTP requests.
- gRPC upstream proxy/recording не реализован.
- Matched unary gRPC serve events попадают в snapshot foundation как JSON events, но full gRPC recording остается отдельной задачей.

Риски/решения:
- Recording не создает временный proxy mapping, а встроен в runtime no-match path; это проще и не мешает существующим mappings.
- Snapshot активирует generated mappings сразу, как stop, чтобы результат можно было проверить без перезапуска.
- `base64Body` добавлен в parser, чтобы binary recorded mappings можно было сразу проигрывать.
- Общий coverage снизился до 69.9%; 90% остается финальным quality gate.
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
- Прогнать stable fixture tests из `testdata` и black-box contract suite.
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
- Загружать stable mapping fixtures только через Admin API, не через internal packages.
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
| 10 | RT-001, RT-002, RT-003, RT-004, RT-005, ADM-014, ACC-006 |
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
