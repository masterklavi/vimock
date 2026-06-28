# Step 4: Request Matching

## What Is Available

- `bodyPatterns.matchesJsonPath` as a string.
- `bodyPatterns.matchesJsonPath` as an object with `expression` and `absent=true`.
- JSONPath path navigation with fields, array indexes and wildcard `*`.
- JSONPath filters with `?()`.
- Equality checks for strings, numbers, booleans, `null` and arrays.
- `.size()` checks for arrays, objects and strings.
- `request.queryParameters.*.equalTo`.
- `request.headers.*.equalTo`.
- `bodyPatterns.equalToJson` as a basic JSON equality matcher.
- Fixture parsing test for all JSON mappings in `examples` and `autotests-example/mocks`.

## Supported JSONPath Examples

```text
$[?(@.method == 'rests.get')]
$.guids[?(@ == 'b27ed95d-3717-4538-9be6-a7136b8ad52f')]
$.params.providers[?(@ == 'provider-1')]
$.params[?(@.destinations.size() == 2)]
$.params.chains[?(@ == ['source','middle','destination'])]
$.params.chains.*[?(@.chain_nodes == ['a','b','c'])]
$.params.filter[?(@.tripId == 42)]
$.params.seals[0][?(@.numbers == ['11111111', '222222222', '3333333333'])]
```

## Example Mapping

```json
{
  "name": "body query header matcher mapping",
  "request": {
    "method": "POST",
    "urlPath": "/matchers",
    "queryParameters": {
      "date": {
        "equalTo": "2025-10-14"
      }
    },
    "headers": {
      "Content-Type": {
        "equalTo": "application/json"
      }
    },
    "bodyPatterns": [
      {
        "matchesJsonPath": "$.params.providers[?(@ == 'provider-1')]"
      },
      {
        "matchesJsonPath": {
          "expression": "$.params.missing",
          "absent": true
        }
      }
    ]
  },
  "response": {
    "status": 200,
    "headers": {
      "Content-Type": "text/plain"
    },
    "body": "matched by body query header"
  },
  "priority": 1
}
```

The same example is available in:

```text
testdata/matcher_mapping.json
```

## Run

```bash
go run ./cmd/vimock
```

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/matcher_mapping.json
```

```bash
curl -i -X POST 'http://localhost:8080/matchers?date=2025-10-14' \
  -H 'Content-Type: application/json' \
  --data '{"params":{"providers":["provider-1"]}}'
```

Expected result:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
```

```text
matched by body query header
```

## Tests

```bash
go test ./...
go test -race ./...
go test ./internal/matcher -run TestJSONPathCurrentMockPatterns
```

## TODO

- Full JSONPath compatibility beyond the patterns used by current mocks is not implemented.
- Full JSONUnit compatibility for `equalToJson` is not implemented.
- JSONPath template helper for responses is not implemented yet.
- Response templating is not implemented yet.
- Body files are not implemented yet.
