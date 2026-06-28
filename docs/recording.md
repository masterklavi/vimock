# Recording

Recording proxies requests to an upstream service and turns observed traffic into in-memory mappings.

## Start Recording

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/start \
  -H 'Content-Type: application/json' \
  -d '{
    "targetBaseUrl": "https://example.com",
    "captureHeaders": {
      "X-Request-Id": {}
    },
    "requestBodyPattern": "equalToJson",
    "persist": true
  }'
```

Then send requests through VIMock:

```bash
curl -i http://localhost:8080/api/items?sku=1 -H 'X-Request-Id: req-1'
```

## Stop And Activate Recorded Mappings

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/stop
```

The response contains generated mappings, and VIMock activates them immediately in memory.

## Snapshot Existing Serve Events

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/snapshot \
  -H 'Content-Type: application/json' \
  -d '{
    "captureHeaders": {
      "X-Request-Id": {}
    },
    "requestBodyPattern": "equalToJson",
    "persist": true
  }'
```

## Current Limitations

- Recorded mappings are stored in memory.
- Full WireMock recording spec parity is not complete yet.
- Binary response bodies are recorded as `base64Body`; body file extraction through `extractBodyCriteria` is not implemented yet.
- gRPC recording is not implemented yet.
