# Step 2: Mapping Admin API

## What Is Available

- In-memory WireMock mapping storage.
- Mapping `id` generation when `id` is missing.
- UUID validation for mapping ids.
- Unknown JSON fields are preserved.
- `persistent`, `metadata.wiremock-gui.folder` and `name` are stored and returned.
- Lock-minimal storage with immutable snapshots for reads.
- JSON validation errors.

## Admin API

```http
GET /__admin/mappings
```

Returns all mappings:

```json
{
  "mappings": [],
  "meta": {
    "total": 0
  }
}
```

```http
GET /__admin/mappings/{id}
```

Returns one mapping by id.

```http
POST /__admin/mappings
```

Creates a mapping. Returns HTTP 201 and the full mapping with `id`.

```http
PUT /__admin/mappings/{id}
```

Updates an existing mapping. The path id is authoritative.

```http
DELETE /__admin/mappings/{id}
```

Deletes an existing mapping. Returns HTTP 200 and `{}`.

## Example

```bash
go run ./cmd/vimock
```

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/simple_body_mapping.json
```

```bash
curl http://localhost:8080/__admin/mappings
```

## Tests

```bash
go test ./...
go test -race ./...
```

## TODO

- Physical persistence to disk is not implemented and is not required for MVP.
- `GET /__admin/mappings?limit&offset` is not implemented yet.
- `/__admin/mappings/reset` is not implemented yet.
- `/__admin/mappings/import` is not implemented yet.
- Runtime matching was added in step 3, not step 2.
