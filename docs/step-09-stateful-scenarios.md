# Step 9: Stateful Scenarios

## What Is Available

- `scenarioName`.
- `requiredScenarioState`.
- `newScenarioState`.
- Initial scenario state is `Started`.
- Atomic scenario selection and transition for concurrent requests.
- Scenario reset endpoint: `POST /__admin/scenarios/reset`.

## How It Works

Several mappings can match the same request, but only the mapping whose `requiredScenarioState` equals the current scenario state is served.

After the mapping is selected, `newScenarioState` changes the scenario state for the next request.

If a mapping has `scenarioName` but no `requiredScenarioState`, it is independent of the current state. If it has `newScenarioState`, it still transitions the scenario after serving.

## Example

First response starts a job:

```json
{
  "scenarioName": "job",
  "requiredScenarioState": "Started",
  "newScenarioState": "Running",
  "request": {
    "method": "GET",
    "urlPath": "/job"
  },
  "response": {
    "status": 202,
    "body": "started"
  },
  "priority": 1
}
```

Second response sees the `Running` state and finishes the job:

```json
{
  "scenarioName": "job",
  "requiredScenarioState": "Running",
  "newScenarioState": "Done",
  "request": {
    "method": "GET",
    "urlPath": "/job"
  },
  "response": {
    "status": 202,
    "body": "running"
  },
  "priority": 1
}
```

Third response keeps returning `done` while the scenario remains in `Done`:

```json
{
  "scenarioName": "job",
  "requiredScenarioState": "Done",
  "request": {
    "method": "GET",
    "urlPath": "/job"
  },
  "response": {
    "status": 200,
    "body": "done"
  },
  "priority": 1
}
```

## Run

Start VIMock:

```bash
go run ./cmd/vimock
```

Create the mappings:

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d '{
    "scenarioName": "job",
    "requiredScenarioState": "Started",
    "newScenarioState": "Running",
    "request": {
      "method": "GET",
      "urlPath": "/job"
    },
    "response": {
      "status": 202,
      "body": "started"
    },
    "priority": 1
  }'
```

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d '{
    "scenarioName": "job",
    "requiredScenarioState": "Running",
    "newScenarioState": "Done",
    "request": {
      "method": "GET",
      "urlPath": "/job"
    },
    "response": {
      "status": 202,
      "body": "running"
    },
    "priority": 1
  }'
```

```bash
curl -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d '{
    "scenarioName": "job",
    "requiredScenarioState": "Done",
    "request": {
      "method": "GET",
      "urlPath": "/job"
    },
    "response": {
      "status": 200,
      "body": "done"
    },
    "priority": 1
  }'
```

Call the same URL several times:

```bash
curl -i http://localhost:8080/job
curl -i http://localhost:8080/job
curl -i http://localhost:8080/job
curl -i http://localhost:8080/job
```

Expected bodies:

```text
started
running
done
done
```

Reset all scenarios:

```bash
curl -i -X POST http://localhost:8080/__admin/scenarios/reset
curl -i http://localhost:8080/job
```

Expected body after reset:

```text
started
```

## Tests

```bash
go test ./...
go test ./internal/scenario ./internal/server -run 'TestSelectAndTransition|TestRuntimeSupportsStatefulScenarios|TestAdminResetsScenarioState'
go test -race ./internal/scenario ./internal/server
```

## Current Scope

- Scenario state is in-memory only.
- `GET /__admin/scenarios` is not implemented in this step.
- `PUT /__admin/scenarios/{name}/state` is not implemented in this step.
