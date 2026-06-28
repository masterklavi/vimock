# Step 5: Response Templating And Body Files

## What Is Available

- `response.bodyFileName` parsing.
- In-memory file storage abstraction.
- Binary body file responses without text recoding.
- Targeted `response-template` support.
- Template helper `{{jsonPath request.body '...'}}`.
- JSON-RPC-style id echo from request body to response body.
- JSON string escaping when a helper value is inserted into `jsonBody`.

## Current Scope

This step implements only the response pipeline needed by current mocks.

The following WireMock features are not complete yet:

- Full Handlebars response template compatibility.
- Persistent or static body file loading from disk.
- gRPC descriptor conversion.

## Example Mapping

```json
{
  "name": "template json rpc id echo",
  "request": {
    "method": "POST",
    "urlPath": "/template"
  },
  "response": {
    "status": 200,
    "headers": {
      "Content-Type": "application/json"
    },
    "jsonBody": {
      "jsonrpc": "2.0",
      "id": "{{jsonPath request.body '$.id'}}",
      "requestId": "{{jsonPath request.body '$.requestId'}}"
    },
    "transformers": ["response-template"]
  },
  "priority": 1
}
```

The same example is available in:

```text
testdata/template_mapping.json
```

## Run

```bash
go run ./cmd/vimock
```

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/template_mapping.json
```

```bash
curl -i -X POST http://localhost:8080/template \
  -H 'Content-Type: application/json' \
  --data '{"id":"rpc-42","requestId":"req-42"}'
```

Expected result:

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "jsonrpc": "2.0",
  "id": "rpc-42",
  "requestId": "req-42"
}
```

## Body Files

`bodyFileName` is resolved through the internal file store:

```json
{
  "response": {
    "status": 200,
    "bodyFileName": "document.pdf"
  }
}
```

At this step files can be provided by code through the in-memory store. Legacy HTTP upload APIs are covered by Step 6.

## Tests

```bash
go test ./...
go test -race ./...
go test ./internal/response -run TestTemplateAndBodyFiles
```
