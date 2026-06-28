# Proxying

Use proxy mappings when VIMock should forward a request to an upstream service.

## Fallback Proxy Mapping

```json
{
  "name": "api proxy fallback",
  "priority": 10,
  "request": {
    "method": "ANY",
    "urlPattern": "/api/.*"
  },
  "response": {
    "status": 200,
    "proxyBaseUrl": "https://example.com",
    "proxyUrlPrefixToRemove": "/api"
  }
}
```

Create it:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d @proxy-mapping.json
```

Call VIMock:

```bash
curl -i http://localhost:8080/api/get
```

With `proxyUrlPrefixToRemove=/api`, VIMock forwards to:

```text
https://example.com/get
```

## Priority

Use a lower priority value for exact stubs and a higher value for fallback proxy mappings. Example:

- Exact stub: `priority: 1`.
- Proxy fallback: `priority: 10`.

## Current Limitations

- HTTP proxying is supported.
- gRPC proxying is not implemented yet.
