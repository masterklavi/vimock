# Step 10: Runtime-Generated Workflow

## What Is Available

- `POST /__admin/mappings` accepts runtime-generated mappings without `id`.
- Created mappings are active immediately after the Admin API response.
- `POST /__admin/mappings` response contains generated `id`.
- `DELETE /__admin/mappings/{id}` removes a runtime-generated mapping.
- Repeated delete returns `404`, which can be safely treated as already deleted by test code.
- Static mappings can be reloaded by listing mappings, finding by `name` and `metadata.wiremock-gui.folder`, then updating by `PUT /__admin/mappings/{id}`.
- `POST /__admin/ext/grpc/reset` remains available for generated PDM/gRPC-compatible mappings.

## Lifecycle

The expected lifecycle is:

```text
create mapping -> use mapping -> cleanup mapping
```

If cleanup is retried after a previous successful cleanup, VIMock returns `404`.

## Create And Use

Create a runtime mapping:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  --data-binary @- <<'JSON'
{
    "name": "runtime generated example",
    "request": {
      "method": "POST",
      "urlPath": "/runtime/example",
      "bodyPatterns": [
        {
          "matchesJsonPath": "$[?(@.method == 'example.run')]"
        }
      ]
    },
    "response": {
      "status": 200,
      "jsonBody": {
        "jsonrpc": "2.0",
        "id": "{{jsonPath request.body '$.id'}}",
        "result": {
          "ok": true
        }
      },
      "transformers": [
        "response-template"
      ]
    },
    "priority": 1,
    "metadata": {
      "wiremock-gui": {
        "folder": "/runtime/generated"
      }
    }
  }
JSON
```

Use it immediately:

```bash
curl -i -X POST http://localhost:8080/runtime/example \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"example.run","params":{},"id":"req-1"}'
```

Expected response body contains:

```json
{
  "id": "req-1"
}
```

## Reload Existing Static Mapping

1. Call `GET /__admin/mappings`.
2. Find mapping with matching `name` and `metadata.wiremock-gui.folder`.
3. Call `PUT /__admin/mappings/{id}` with the new mapping body.

This keeps the same mapping id and makes the updated response active immediately.

## Generated Contract Coverage

The step 10 tests cover representative generated mappings for:

- PDM/gRPC-compatible HTTP mapping with `POST /__admin/ext/grpc/reset`.
- ShCat JSON-RPC mapping.
- Officer JSON-RPC mapping.
- Susanin JSON-RPC mapping.
- Vanga JSON-RPC mapping.
- Courier/Frodo JSON-RPC mapping.
- Fry JSON-RPC mapping.

## Tests

```bash
go test ./internal/server -run 'TestRuntimeGeneratedWorkflow|TestAutotestMappingLifecycle'
go test ./...
```

## Current Scope

- The tests are in-process `httptest` tests, not external black-box API tests.
- Running the full external autotest suite is out of scope for this step.
