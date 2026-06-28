# Getting Started

## Run VIMock

```bash
go run ./cmd/vimock
```

Default listener:

```text
http://localhost:8080
```

Health checks:

```bash
curl -i http://localhost:8080/__admin/health
curl -i http://localhost:8080/__admin/ready
```

## Create Your First HTTP Stub

Create a mapping:

```bash
curl -i -X POST http://localhost:8080/__admin/mappings \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "hello",
    "request": {
      "method": "GET",
      "urlPath": "/hello"
    },
    "response": {
      "status": 200,
      "headers": {
        "Content-Type": "application/json"
      },
      "jsonBody": {
        "message": "hello from VIMock"
      }
    }
  }'
```

Call it:

```bash
curl -i http://localhost:8080/hello
```

List mappings:

```bash
curl -s http://localhost:8080/__admin/mappings
```

## Next Tasks

- Upload gRPC descriptors: [gRPC descriptors](grpc-descriptors.md).
- Create a gRPC stub: [gRPC stubbing](grpc-stubbing.md).
- Upload body files: [Body files and legacy upload](body-files-and-legacy-upload.md).
- Configure HTTPS or Docker: [Configuration](configuration.md).
