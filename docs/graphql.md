# GraphQL Matching

VIMock supports a semantic GraphQL body matcher through `request.customMatcher.name = graphql-body-matcher`.

## Mapping Example

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

Create it:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @graphql-mapping.json
```

Call it with different field order and formatting:

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

- Ignores whitespace and formatting.
- Ignores field order where GraphQL semantics allow it.
- Preserves alias, argument, fragment, variable and operation name semantics.
- Variables are compared as normalized JSON. Object key order is ignored, array order is preserved.

## Current Limitations

- Federation-specific matching is not implemented.
- The supported runtime mapping form is the JSON/Admin API custom matcher shown above.
