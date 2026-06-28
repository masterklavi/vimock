# ТЗ: WireMock-подобный сервис на Go

## 1. Назначение

Нужно разработать сервис `VIMock` на Go, который работает как WireMock для HTTP/gRPC/GraphQL мокирования и совместим с используемыми в проекте WireMock mappings, Admin API и автотестами.

Название сервиса: `VIMock`.

Техническое имя сервиса для бинарника, Docker image, package/module naming и CLI: `vimock`.

Глобальная цель: совместимость с WireMock по поведению, JSON mapping format и Admin API.

MVP scope: все требования с приоритетом `MUST` в этом документе.

## 2. Источники требований

| Источник | Назначение |
|---|---|
| `current-mocks.md` | Анализ 291 WireMock JSON mapping из `examples`. |
| `current-autotest.md` | Анализ работы автотестов из `autotests-example` с WireMock. |
| https://wiremock.org/docs/grpc/ | Целевой API-синтаксис для gRPC. |
| https://wiremock.org/docs/graphql/ | Целевой API-синтаксис для GraphQL. |
| https://wiremock.org/docs/record-playback/ | Целевой API-синтаксис для recording и snapshotting. |
| Ответы заказчика от 2026-06-28 | Базовые продуктовые и эксплуатационные требования. |

## 3. Термины

| Термин | Значение |
|---|---|
| Mapping | WireMock JSON stub mapping. |
| VIMock | Разрабатываемый WireMock-подобный сервис. |
| vimock | Техническое имя сервиса для binary/container/module/CLI. |
| Admin API | HTTP API управления mappings, файлами, recording и расширениями. |
| Body file | Файл, на который ссылается `response.bodyFileName`. |
| Descriptor file | `.dsc` или `.proto` файл, необходимый для gRPC обработки. |
| Runtime mapping | Mapping, созданный через Admin API во время выполнения автотеста. |
| Static mapping | Mapping, загружаемый из каталога моков при старте или bootstrap-е тестов. |
| MUST | Обязательное требование MVP. |
| SHOULD | Желательное требование, допускается после MVP. |
| OUT | Не входит в MVP, пока не появится отдельное требование. |

## 4. Ограничения и допущения

| ID | Приоритет | Требование |
|---|---|---|
| CON-001 | MUST | Mappings и runtime state должны храниться in-memory в рамках процесса. |
| CON-002 | MUST | Логи должны писаться в stdout. |
| CON-003 | MUST | Сервис должен запускаться как локальный бинарник. |
| CON-004 | MUST | Сервис должен запускаться в Docker. |
| CON-005 | MUST | Сервис должен быть пригоден для запуска в CI и Kubernetes. |
| CON-006 | MUST | Ожидается 100+ независимых процессов сервиса, поэтому реализация должна избегать избыточного потребления памяти. |
| CON-007 | MUST | Все runtime изменения mappings, files и descriptors должны становиться активными без рестарта процесса. |
| CON-008 | SHOULD | Неизвестные поля WireMock mapping должны сохраняться в модели и API-ответах, если они не мешают исполнению поддерживаемых фич. |

## 5. Поддерживаемые протоколы

| ID | Приоритет | Требование |
|---|---|---|
| PROTO-001 | MUST | Поддержать HTTP/1.1. |
| PROTO-002 | MUST | Поддержать HTTPS. |
| PROTO-003 | MUST | Поддержать HTTP/2. |
| PROTO-004 | MUST | Поддержать gRPC поверх HTTP/2. |
| PROTO-005 | MUST | Поддержать GraphQL-over-HTTP. |

## 6. WireMock Mapping Format

| ID | Приоритет | Требование |
|---|---|---|
| MAP-001 | MUST | Сервис должен принимать WireMock JSON mappings с полями `id`, `name`, `persistent`, `request`, `response`, `priority`, `metadata`. |
| MAP-002 | MUST | Если `id` не передан, сервис должен сгенерировать уникальный `id` и вернуть его в Admin API response. |
| MAP-003 | MUST | Поле `persistent=true` должно приниматься, храниться и возвращаться через Admin API. Физическая персистентность на диск не требуется для MVP. |
| MAP-004 | MUST | Поле `metadata.wiremock-gui.folder` должно приниматься, храниться и возвращаться через Admin API. Реализация GUI не требуется. |
| MAP-005 | MUST | Поддержать HTTP methods в mappings: `ANY`, `GET`, `POST`. |
| MAP-006 | MUST | Поддержать URL matchers: `request.url`, `request.urlPath`, `request.urlPattern`. |
| MAP-007 | MUST | Поддержать `priority`. Mapping с меньшим числом priority должен выбираться раньше mapping-а с большим числом. |
| MAP-008 | MUST | Если несколько matching mappings имеют одинаковый priority, порядок выбора должен быть детерминированным. Для MVP используется порядок добавления mapping-а. |
| MAP-009 | MUST | Proxy mappings с `priority=10` должны работать как fallback после более приоритетных точных mappings. |

## 7. Request Matching

| ID | Приоритет | Требование |
|---|---|---|
| MATCH-001 | MUST | Поддержать `bodyPatterns.matchesJsonPath` в строковом формате. |
| MATCH-002 | MUST | Поддержать `bodyPatterns.matchesJsonPath` в объектном формате с `expression`. |
| MATCH-003 | MUST | Поддержать `matchesJsonPath.absent=true`. |
| MATCH-004 | MUST | JSONPath engine должен поддерживать фильтры `?()`. |
| MATCH-005 | MUST | JSONPath engine должен поддерживать сравнение строк, чисел и boolean. |
| MATCH-006 | MUST | JSONPath engine должен поддерживать проверку размера массивов через `.size()`. |
| MATCH-007 | MUST | JSONPath engine должен поддерживать проверку элементов массивов и вложенных полей. |
| MATCH-008 | MUST | Поддержать `request.queryParameters.*.equalTo`. |
| MATCH-009 | MUST | Поддержать `request.headers.*.equalTo`. |
| MATCH-010 | MUST | Поддержать matching header `Content-Type: application/protobuf`. |
| MATCH-011 | MUST | Поддержать `bodyPatterns.equalToJson` для gRPC-compatible mappings. |
| MATCH-012 | SHOULD | Расширять matcher engine по мере появления новых matcher types в моках. |

## 8. Response Generation

| ID | Приоритет | Требование |
|---|---|---|
| RESP-001 | MUST | Поддержать `response.status`. |
| RESP-002 | MUST | Поддержать `response.headers`. |
| RESP-003 | MUST | Поддержать `response.jsonBody`. |
| RESP-004 | MUST | Поддержать `response.body`. |
| RESP-005 | MUST | Поддержать `response.bodyFileName`. |
| RESP-006 | MUST | `bodyFileName` должен уметь отдавать JSON, PDF, binary/protobuf и другие byte payloads без изменения содержимого. |
| RESP-007 | MUST | Поддержать transformer `response-template`. |
| RESP-008 | MUST | Поддержать template helper `{{jsonPath request.body '...'}}` в `jsonBody` и `body`. |
| RESP-009 | MUST | Поддержать request-based response values, необходимые для JSON-RPC `id`. |
| RESP-010 | MUST | Поддержать static response body из mapping-а. |
| RESP-011 | MUST | Поддержать file response body из storage сервиса. |
| RESP-012 | MUST | Поддержать delays в WireMock-compatible формате: `fixedDelayMilliseconds`, `delayDistribution`, `chunkedDribbleDelay`. |
| RESP-013 | OUT | Fault simulation не входит в MVP, так как текущие моки и автотесты его не используют. |
| RESP-014 | OUT | Webhooks и `postServeActions` не входят в MVP, так как текущие моки и автотесты их не используют. |

## 9. Proxy и Recording

| ID | Приоритет | Требование |
|---|---|---|
| PROXY-001 | MUST | Поддержать proxy mappings через `response.proxyBaseUrl`. |
| PROXY-002 | MUST | Поддержать `response.proxyUrlPrefixToRemove`. |
| PROXY-003 | MUST | Proxy должен использоваться только если не найден более приоритетный stub mapping. |
| REC-001 | MUST | Поддержать WireMock-compatible recording mode. |
| REC-002 | MUST | Поддержать `POST /__admin/recordings/start`. |
| REC-003 | MUST | `POST /__admin/recordings/start` должен принимать payload с `targetBaseUrl`. |
| REC-004 | MUST | `POST /__admin/recordings/start` должен поддерживать record spec fields: `filters`, `captureHeaders`, `requestBodyPattern`, `extractBodyCriteria`, `persist`, `repeatsAsScenarios`, `transformers`, `transformerParameters`. |
| REC-005 | MUST | Поддержать `POST /__admin/recordings/stop`. |
| REC-006 | MUST | `POST /__admin/recordings/stop` должен создавать recorded mappings и делать их активными сразу после stop. |
| REC-007 | MUST | Поддержать `POST /__admin/recordings/snapshot`. |
| REC-008 | MUST | `POST /__admin/recordings/snapshot` должен уметь создавать mappings из уже полученных serve events. |
| REC-009 | MUST | `POST /__admin/recordings/snapshot` должен поддерживать snapshot spec fields: `filters`, `captureHeaders`, `requestBodyPattern`, `extractBodyCriteria`, `outputFormat`, `persist`, `repeatsAsScenarios`, `transformers`, `transformerParameters`. |
| REC-010 | MUST | Recorded mappings должны сохраняться в in-memory storage. |
| REC-011 | MUST | Для binary response bodies recording должен поддерживать `base64Body` или extraction в body files согласно `extractBodyCriteria`. |

## 10. Stateful Scenarios

| ID | Приоритет | Требование |
|---|---|---|
| SCN-001 | MUST | Поддержать `scenarioName`. |
| SCN-002 | MUST | Поддержать `requiredScenarioState`. |
| SCN-003 | MUST | Поддержать `newScenarioState`. |
| SCN-004 | MUST | Начальное состояние scenario должно быть `Started`. |
| SCN-005 | MUST | State scenarios должны работать корректно при параллельных HTTP-запросах. |
| SCN-006 | MUST | Admin API reset mappings/state не должен оставлять scenario state в неконсистентном состоянии. |

## 11. JSON-RPC поверх HTTP

| ID | Приоритет | Требование |
|---|---|---|
| JRPC-001 | MUST | Сервис должен корректно матчить JSON-RPC requests по полям `method`, `params`, `id`. |
| JRPC-002 | MUST | Response templating должен уметь возвращать исходный `id` из request body. |
| JRPC-003 | MUST | Поддерживаемый JSON-RPC transport для текущих моков: HTTP `POST` с JSON body. |

## 12. gRPC

| ID | Приоритет | Требование |
|---|---|---|
| GRPC-001 | MUST | gRPC API-синтаксис mappings должен быть совместим с https://wiremock.org/docs/grpc/. |
| GRPC-002 | MUST | gRPC stubs должны задаваться WireMock JSON mappings. |
| GRPC-003 | MUST | gRPC request mapping должен использовать `request.method=POST`. |
| GRPC-004 | MUST | gRPC request mapping должен использовать `request.urlPath=/<fully-qualified service name>/<method name>`. |
| GRPC-005 | MUST | Сервис должен конвертировать входящее protobuf-сообщение в JSON-представление перед matching. |
| GRPC-006 | MUST | Сервис должен применять JSON matchers к JSON-представлению protobuf-сообщения. |
| GRPC-007 | MUST | Сервис должен конвертировать JSON response body обратно в protobuf response. |
| GRPC-008 | MUST | Поддержать gRPC response templating, включая `{{jsonPath request.body '...'}}`. |
| GRPC-009 | MUST | Поддержать gRPC status headers, включая `grpc-status-name`. |
| GRPC-010 | MUST | Поддержать non-OK gRPC responses через WireMock-compatible status mapping. |
| GRPC-011 | MUST | Поддержать descriptor files `.dsc`. |
| GRPC-012 | MUST | Поддержать proto files `.proto`, если они нужны для descriptor/runtime schema generation. |
| GRPC-013 | MUST | В отличие от WireMock extension, descriptor/proto files должны загружаться через Admin API, а не через ручное размещение в файловой системе. |
| GRPC-014 | MUST | Поддержать `PUT /__admin/ext/grpc/descriptors/{fileName}` для загрузки или замены descriptor/proto файла raw bytes payload-ом. |
| GRPC-015 | MUST | Поддержать `GET /__admin/ext/grpc/descriptors` для списка загруженных descriptor/proto файлов. |
| GRPC-016 | MUST | Поддержать `DELETE /__admin/ext/grpc/descriptors/{fileName}` для удаления descriptor/proto файла. |
| GRPC-017 | MUST | Поддержать `POST /__admin/ext/grpc/reset` для reload descriptor/proto files. |
| GRPC-018 | MUST | Legacy file upload из текущих автотестов должен оставаться совместимым до миграции на Admin API descriptor upload. |

## 13. GraphQL

| ID | Приоритет | Требование |
|---|---|---|
| GQL-001 | MUST | GraphQL API-синтаксис должен быть совместим с https://wiremock.org/docs/graphql/. |
| GQL-002 | MUST | Поддержать semantic query matching. |
| GQL-003 | MUST | GraphQL matcher должен нормализовать query перед сравнением. |
| GQL-004 | MUST | GraphQL matcher должен игнорировать whitespace и форматирование query. |
| GQL-005 | MUST | GraphQL matcher должен не зависеть от порядка полей там, где это допустимо GraphQL-семантикой. |
| GQL-006 | MUST | GraphQL matcher должен сравнивать variables через JSON matching logic. |
| GQL-007 | MUST | Поддержать синтаксис `GraphqlBodyMatcher.extensionName`. |
| GQL-008 | MUST | Поддержать параметры `GraphqlBodyMatcher.parameters(query)`. |
| GQL-009 | MUST | Поддержать параметры `GraphqlBodyMatcher.parameters(query, variables)`. |
| GQL-010 | MUST | Поддержать JSON/Admin API эквивалент GraphQL custom matcher. Минимальный формат: `request.customMatcher.name=graphql`, `request.customMatcher.parameters.query`, `request.customMatcher.parameters.variables`. |
| GQL-011 | MUST | GraphQL responses должны использовать обычный WireMock response pipeline: `status`, `headers`, `jsonBody`, `body`, templating. |

## 14. Admin API для mappings

| ID | Приоритет | Требование |
|---|---|---|
| ADM-001 | MUST | Admin API должен быть доступен под `/__admin`. |
| ADM-002 | MUST | Поддержать `GET /__admin/mappings`. |
| ADM-003 | MUST | `GET /__admin/mappings` должен возвращать JSON с массивом `mappings`. |
| ADM-004 | MUST | Каждый элемент `mappings` должен содержать `id`, `name`, `metadata.wiremock-gui.folder` и исходное содержимое mapping-а. |
| ADM-005 | MUST | Поддержать `POST /__admin/mappings`. |
| ADM-006 | MUST | Успешный `POST /__admin/mappings` должен возвращать HTTP 201 и JSON с `id`. |
| ADM-007 | MUST | Поддержать `PUT /__admin/mappings/{id}`. |
| ADM-008 | MUST | Успешный `PUT /__admin/mappings/{id}` должен возвращать HTTP 200 или 201. |
| ADM-009 | MUST | Поддержать `DELETE /__admin/mappings/{id}`. |
| ADM-010 | MUST | Успешный `DELETE /__admin/mappings/{id}` должен возвращать HTTP 200. |
| ADM-011 | MUST | `DELETE /__admin/mappings/{id}` для отсутствующего mapping-а должен возвращать HTTP 404. |
| ADM-012 | MUST | Mapping должен становиться активным сразу после успешного `POST` или `PUT`. |
| ADM-013 | MUST | Удаленный mapping должен перестать участвовать в matching сразу после успешного `DELETE`. |
| ADM-014 | MUST | Повторная загрузка статических моков должна поддерживаться через поиск по `name` и `metadata.wiremock-gui.folder`, затем `PUT` по найденному `id`. |
| ADM-015 | MUST | Одновременные операции `GET`, `POST`, `PUT`, `DELETE` не должны приводить к data race или неконсистентному matching. |

## 15. File API

| ID | Приоритет | Требование |
|---|---|---|
| FILE-001 | MUST | Поддержать загрузку файлов, используемых `bodyFileName`. |
| FILE-002 | MUST | Поддержать загрузку `.dsc` файлов для текущих автотестов. |
| FILE-003 | MUST | Поддержать legacy endpoint `POST /api/login`. |
| FILE-004 | MUST | `POST /api/login` должен возвращать HTTP 200 и token/body, пригодный для передачи в header `X-Auth`. |
| FILE-005 | MUST | Поддержать legacy endpoint `POST /api/tus/{file}?override=true`. |
| FILE-006 | MUST | `POST /api/tus/{file}?override=true` должен принимать headers `Tus-Resumable`, `Upload-Length`, `Upload-Metadata`, `X-Auth` и возвращать HTTP 201. |
| FILE-007 | MUST | Поддержать legacy endpoint `PATCH /api/tus/{file}?override=true`. |
| FILE-008 | MUST | `PATCH /api/tus/{file}?override=true` должен принимать binary content с headers `Content-Type: application/offset+octet-stream`, `Tus-Resumable`, `Upload-Offset`, `X-Auth` и возвращать HTTP 204. |
| FILE-009 | MUST | Загруженные файлы должны быть доступны для mappings без рестарта процесса. |
| FILE-010 | MUST | Для MVP file storage может быть in-memory. |
| FILE-011 | SHOULD | Добавить WireMock-compatible или vimock-native Admin API для загрузки обычных body files, если legacy TUS API будет удаляться из автотестов. |

## 16. Runtime-generated mappings в автотестах

| ID | Приоритет | Требование |
|---|---|---|
| RT-001 | MUST | Сервис должен поддерживать создание mapping-а перед тестом через `POST /__admin/mappings`. |
| RT-002 | MUST | Сервис должен поддерживать использование runtime mapping-а сразу после создания. |
| RT-003 | MUST | Сервис должен поддерживать удаление runtime mapping-а после теста через `DELETE /__admin/mappings/{id}`. |
| RT-004 | MUST | Runtime-generated mappings должны поддерживать PDM, Shcat, Officer, Susanin, Vanga, Courier/Frodo и Fry сценарии из автотестов. |
| RT-005 | MUST | Runtime-generated PDM mappings должны корректно работать после `POST /__admin/ext/grpc/reset`. |

## 17. Производительность и надежность

| ID | Приоритет | Требование |
|---|---|---|
| NFR-001 | MUST | Сервис должен быть безопасен для конкурентных запросов и операций Admin API. |
| NFR-002 | MUST | Matching должен быть оптимизирован под большое число mappings без полного лишнего копирования request/response payloads. |
| NFR-003 | MUST | Runtime storage должен поддерживать атомарную замену набора mappings или отдельного mapping-а. |
| NFR-004 | MUST | Ошибки загрузки invalid mapping должны возвращать понятный HTTP error с текстом причины. |
| NFR-005 | MUST | Сервис не должен завершать процесс из-за malformed request, malformed mapping или ошибки proxy target. |
| NFR-006 | SHOULD | Для чтения mappings использовать immutable snapshot или эквивалентную lock-minimal схему. |

## 18. Тестирование

| ID | Приоритет | Требование |
|---|---|---|
| TEST-001 | MUST | Не менее 90% кода должно быть покрыто unit-тестами. |
| TEST-002 | MUST | Unit-тесты должны покрывать matcher engine, priority selection, response templating, Admin API handlers, file API, gRPC descriptor registry, GraphQL matcher. |
| TEST-003 | MUST | Должны быть fixture-based tests на mappings из `examples` и `autotests-example/mocks`. |
| TEST-004 | MUST | Должны быть контрактные тесты совместимости с WireMock JSON mapping behavior для поддерживаемых фич. |
| TEST-005 | MUST | Должны быть race/concurrency tests для параллельного matching и Admin API операций. |
| TEST-006 | SHOULD | Должны быть benchmark-тесты matching pipeline и response pipeline. |

## 19. Out of scope для MVP

| ID | Статус | Пояснение |
|---|---|---|
| OUT-001 | OUT | WireMock GUI не требуется. Нужно только хранить `metadata.wiremock-gui.folder`. |
| OUT-002 | OUT | Fault simulation не требуется, пока не появятся моки или тесты с `fault`. |
| OUT-003 | OUT | Webhooks и `postServeActions` не требуются, пока не появятся моки или тесты с этими полями. |
| OUT-004 | OUT | WebSocket mocking не требуется. |
| OUT-005 | OUT | Физическая персистентность mappings на диск не требуется для MVP. |

## 20. Acceptance criteria MVP

| ID | Критерий приемки |
|---|---|
| ACC-001 | Все JSON mappings из `examples` должны успешно загружаться без потери поддерживаемых полей. |
| ACC-002 | Все JSON mappings из `autotests-example/mocks` должны успешно загружаться без потери поддерживаемых полей. |
| ACC-003 | Static mocks должны матчиться по тем WireMock-фичам, которые перечислены в `current-mocks.md`. |
| ACC-004 | Автотестовый bootstrap должен успешно выполнить загрузку JSON mappings через Admin API. |
| ACC-005 | Автотестовый bootstrap должен успешно загрузить `.dsc` через legacy file API и выполнить `POST /__admin/ext/grpc/reset`. |
| ACC-006 | Runtime-generated mappings должны создаваться, использоваться и удаляться по workflow из `current-autotest.md`. |
| ACC-007 | gRPC mappings должны проходить контрактные тесты по синтаксису из WireMock gRPC docs. |
| ACC-008 | GraphQL matcher должен проходить контрактные тесты по синтаксису из WireMock GraphQL docs. |
| ACC-009 | Unit test coverage должен быть не ниже 90%. |
| ACC-010 | Сервис должен запускаться как бинарник и Docker container. |
