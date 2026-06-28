# GraphQL Matching

VIMock поддерживает semantic GraphQL body matcher через `request.customMatcher.name = graphql-body-matcher`.

## Пример Mapping

```json
{
  "name": "GraphQL hero",
  "request": {
    "method": "POST",
    "urlPath": "/graphql",
    "customMatcher": {
      "name": "graphql-body-matcher",
      "parameters": {
        "query": "query GetHero($episode: Episode) { hero(episode: $episode) { name age friends { name } } }",
        "variables": {
          "episode": "JEDI"
        },
        "operationName": "GetHero"
      }
    }
  },
  "response": {
    "status": 200,
    "jsonBody": {
      "data": {
        "hero": {
          "name": "Luke Skywalker"
        }
      }
    }
  }
}
```

Создать mapping:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @graphql-mapping.json
```

Вызвать с другим порядком полей и форматированием:

```bash
curl -i -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{
    "operationName": "GetHero",
    "variables": {"episode": "JEDI"},
    "query": "query GetHero($episode: Episode) { hero(episode: $episode) { friends { name } age name } }"
  }'
```

## Matching Rules

- Игнорирует whitespace и formatting.
- Игнорирует порядок fields там, где это допустимо GraphQL semantics.
- Учитывает aliases, arguments, fragments, variables и operation name semantics.
- Variables сравниваются как normalized JSON. Порядок object keys игнорируется, порядок arrays сохраняется.

## Текущие ограничения

- Federation-specific matching не реализован.
- Поддерживаемая runtime mapping form - JSON/Admin API custom matcher из примера выше.
