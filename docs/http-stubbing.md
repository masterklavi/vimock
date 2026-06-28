# HTTP Stubbing

VIMock accepts WireMock-style JSON mappings through the Admin API.

## Create A Mapping

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @mapping.json
```

Minimal mapping:

```json
{
  "name": "hello",
  "request": {
    "method": "GET",
    "urlPath": "/hello"
  },
  "response": {
    "status": 200,
    "body": "hello"
  }
}
```

## Admin API

```bash
curl -s http://localhost:8080/__admin/mappings
curl -s http://localhost:8080/__admin/mappings/{id}
curl -i -X PUT http://localhost:8080/__admin/mappings/{id} -H 'Content-Type: application/json' -d @mapping.json
curl -i -X DELETE http://localhost:8080/__admin/mappings/{id}
```

## Request Matching

Supported method values:

- `ANY`
- `GET`
- `POST`

Supported URL matchers:

```json
{"url": "/exact?x=1"}
{"urlPath": "/path-only"}
{"urlPattern": "/api/.*"}
{"urlPathPattern": "/items/[0-9]+"}
```

Query and header examples:

```json
{
  "request": {
    "method": "POST",
    "urlPath": "/items",
    "queryParameters": {
      "source": {"equalTo": "mobile"}
    },
    "headers": {
      "Content-Type": {"equalTo": "application/json"}
    }
  }
}
```

Body matcher examples:

```json
{
  "bodyPatterns": [
    {"matchesJsonPath": "$.items[?(@ == 'one')]"},
    {"matchesJsonPath": {"expression": "$.missing", "absent": true}},
    {"equalToJson": {"id": "req-1", "items": ["one"]}}
  ]
}
```

## Responses

Static body:

```json
{
  "response": {
    "status": 200,
    "headers": {"Content-Type": "text/plain"},
    "body": "ok"
  }
}
```

JSON body:

```json
{
  "response": {
    "status": 200,
    "jsonBody": {"ok": true}
  }
}
```

Response template with request JSONPath:

```json
{
  "response": {
    "status": 200,
    "jsonBody": {
      "jsonrpc": "2.0",
      "id": "{{jsonPath request.body '$.id'}}",
      "result": "ok"
    },
    "transformers": ["response-template"]
  }
}
```

Body file response:

```json
{
  "response": {
    "status": 200,
    "bodyFileName": "payload.bin",
    "headers": {"Content-Type": "application/octet-stream"}
  }
}
```

See [Body files and legacy upload](body-files-and-legacy-upload.md) for upload commands.

## Priority

Lower `priority` values win. If priorities are equal, VIMock uses deterministic insertion order.

```json
{
  "priority": 1,
  "request": {"method": "GET", "urlPath": "/exact"},
  "response": {"status": 200, "body": "exact"}
}
```

Proxy fallback mappings usually use a larger priority, for example `10`.
