# Step 8: Proxy And Delays

## What Is Available

- `response.proxyBaseUrl`.
- `response.proxyUrlPrefixToRemove`.
- Proxy fallback after normal priority selection.
- `response.fixedDelayMilliseconds`.
- `response.delayDistribution` with `uniform` and `lognormal`.
- `response.chunkedDribbleDelay` for delayed body chunks.

## Proxy Example

```json
{
  "priority": 10,
  "request": {
    "method": "ANY",
    "urlPattern": "/api-proxy/.*"
  },
  "response": {
    "proxyBaseUrl": "https://upstream.example",
    "proxyUrlPrefixToRemove": "/api-proxy"
  }
}
```

Request:

```text
GET /api-proxy/v1/items?debug=true
```

Upstream target:

```text
https://upstream.example/v1/items?debug=true
```

If a higher-priority non-proxy stub matches the same request, that stub wins and proxy is not called.

## Delay Examples

Fixed delay:

```json
{
  "response": {
    "status": 200,
    "body": "delayed",
    "fixedDelayMilliseconds": 100
  }
}
```

Uniform delay:

```json
{
  "response": {
    "status": 200,
    "body": "delayed",
    "delayDistribution": {
      "type": "uniform",
      "lower": 50,
      "upper": 150
    }
  }
}
```

Lognormal delay:

```json
{
  "response": {
    "status": 200,
    "body": "delayed",
    "delayDistribution": {
      "type": "lognormal",
      "median": 80,
      "sigma": 0.4
    }
  }
}
```

Chunked dribble delay:

```json
{
  "response": {
    "status": 200,
    "body": "abcdef",
    "chunkedDribbleDelay": {
      "numberOfChunks": 3,
      "totalDuration": 30
    }
  }
}
```

## Tests

```bash
go test ./...
go test ./internal/proxy ./internal/delay ./internal/server -run 'TestProxy|TestDelay|TestRuntimeProxies|TestRuntimeAppliesFixedDelay'
go test -race ./...
```

## Current Scope

Proxy recording is not implemented in this step. It is planned separately with recording/snapshotting.
