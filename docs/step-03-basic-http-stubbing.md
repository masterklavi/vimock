# Step 3: Basic HTTP Stubbing

## What Is Available

- Runtime handler for non-Admin HTTP requests.
- Admin routes are excluded from runtime matching.
- `request.method`: `ANY`, `GET`, `POST`.
- `request.url`: exact match by path and query.
- `request.urlPath`: exact match by path, query ignored.
- `request.urlPattern`: full regex match by path and query.
- Priority selection: lower `priority` wins.
- Tie-breaker: insertion order wins for equal priority.
- Response `status`.
- Response `headers`.
- Response `body`.
- Response `jsonBody`.
- WireMock-like 404 for unmatched requests.

## Example Mapping

```json
{
  "name": "simple body mapping",
  "request": {
    "method": "GET",
    "urlPath": "/some/path"
  },
  "response": {
    "status": 200,
    "headers": {
      "Content-Type": "text/plain"
    },
    "body": "hello from vimock"
  },
  "priority": 1
}
```

The same example is available in:

```text
testdata/simple_body_mapping.json
```

## Run

```bash
go run ./cmd/vimock
```

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/simple_body_mapping.json
```

```bash
curl -i http://localhost:8080/some/path
```

Expected result:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
```

```text
hello from vimock
```

## Tests

```bash
go test ./...
go test -race ./...
```

## TODO

- Body matchers are not implemented yet.
- Query parameter matchers are not implemented yet.
- Header matchers are not implemented yet.
- Response templating is not implemented yet.
- Body files are not implemented yet.
- Proxying is not implemented yet.
- WireMock near-miss diff for unmatched requests is not implemented yet.
