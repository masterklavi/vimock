# Step 13: GraphQL Semantic Matcher

## What Is Available

- WireMock-compatible GraphQL custom matcher:

```json
{
  "customMatcher": {
    "name": "graphql-body-matcher",
    "parameters": {
      "query": "{ hero { name } }"
    }
  }
}
```

- Incoming GraphQL body must be a JSON object with `query`.
- Incoming `variables` and `operationName` are supported.
- Query whitespace and field order are ignored.
- Aliases, arguments, fragments and directives are preserved for matching.
- Variables use strict JSON equality: object key order is ignored, array order is preserved.
- If expected `variables` are absent, request `variables` must also be absent.
- If expected `operationName` is absent, request `operationName` must also be absent.
- Invalid JSON or invalid GraphQL query results in no match, not a process error.
- `urlPathPattern` is supported for WireMock extension fixture compatibility.

## Example Mapping

The example mapping is stored at:

```text
testdata/graphql_mapping.json
```

It matches this GraphQL operation semantically:

```graphql
query GetHero($episode: Episode) {
  hero(episode: $episode) {
    name
    age
    friends {
      name
    }
  }
}
```

## Run

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/graphql_mapping.json

curl -i -X POST http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  -d '{"operationName":"GetHero","variables":{"episode":"JEDI"},"query":"query GetHero($episode: Episode) { hero(episode: $episode) { friends { name } age name } }"}'
```

The second request uses a different field order than the mapping, but it still matches.

## Current Scope

- This is a schema-less semantic matcher.
- Federation-specific GraphQL behavior is not implemented.
- The parser is implemented for the syntax used by current compatibility fixtures: operations, fields, aliases, arguments, variables, fragments, inline fragments, directives, lists and input objects.
- Diagnostics are used internally as no-match decisions and are not exposed as WireMock sub-events yet.

## Tests

```bash
go test ./internal/matcher ./internal/server -run 'TestGraphQL|TestParseCustomMatcher'
go test ./...
```
