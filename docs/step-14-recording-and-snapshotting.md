# Step 14: Recording And Snapshotting

## What Is Available

- `POST /__admin/recordings/start` starts recording mode.
- `POST /__admin/recordings/stop` stops recording mode, creates mappings from recorded proxied traffic and activates them.
- `POST /__admin/recordings/snapshot` creates mappings from stored serve events and activates them.
- Active recording proxies unmatched HTTP requests to `targetBaseUrl`.
- Recorded mappings stay in memory.
- JSON response bodies are recorded as `jsonBody`.
- Text response bodies are recorded as `body`.
- Binary response bodies are recorded as `base64Body`.
- `captureHeaders` records selected request headers as `request.headers.*.equalTo`.
- JSON request bodies are recorded as `bodyPatterns.equalToJson`.

## Example Files

```text
testdata/recording_start.json
testdata/recording_snapshot.json
```

## Start Recording

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/start \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/recording_start.json
```

After start, unmatched HTTP requests are proxied to `targetBaseUrl` from the start spec.

```bash
curl -i http://localhost:8080/api/products/123 \
  -H 'X-Request-Id: req-1'
```

## Stop Recording

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/stop
```

The response contains generated mappings:

```json
{
  "mappings": [],
  "meta": {
    "total": 0
  }
}
```

The generated mappings are active immediately after `stop`.

## Snapshot Serve Events

Snapshot uses serve events already observed by VIMock.

```bash
curl -i -X POST http://localhost:8080/__admin/recordings/snapshot \
  -H 'Content-Type: application/json' \
  --data-binary @testdata/recording_snapshot.json
```

## Current Scope

- HTTP recording is supported.
- Snapshotting works from in-memory serve events.
- Mappings are activated in memory only.
- `outputFormat`, `extractBodyCriteria` and `repeatsAsScenarios` are accepted in the spec but not fully implemented yet.
- gRPC upstream proxy recording is not implemented yet.
- gRPC serve-event snapshot foundation records matched unary gRPC JSON events, but full gRPC recording remains a known gap.
- Persistent filesystem output is not implemented yet.

## Tests

```bash
go test ./internal/recording ./internal/server -run 'TestRecording|TestBuildSnapshot|TestStoreStartStop'
go test ./...
```
