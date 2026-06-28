# Step 15: HTTPS, Docker, Kubernetes Readiness And Performance

## What Is Available

- HTTP listener stays enabled on `--port` / `VIMOCK_PORT`.
- Optional HTTPS listener is enabled by `--https-port` / `VIMOCK_HTTPS_PORT`.
- HTTPS supports file-based certificates via `--tls-cert-file` and `--tls-key-file`.
- HTTPS supports local/CI self-signed certificates via `--tls-self-signed`.
- TLS config advertises `h2` and `http/1.1` through ALPN.
- Both HTTP and HTTPS listeners are stopped by the same graceful shutdown path.
- Docker image runs as a non-root user on Alpine with CA certificates.
- Docker image has a healthcheck against `GET /__admin/health`.
- Runtime matching iterates the immutable mapping snapshot without copying the whole mapping list per request.
- Benchmarks cover large mapping matching and response templating.

## Run HTTP Only

```bash
go run ./cmd/vimock
curl -i http://localhost:8080/__admin/health
```

## Run HTTPS With Self-Signed Certificate

```bash
go run ./cmd/vimock --https-port 8443 --tls-self-signed
curl -k --http2 -i https://localhost:8443/__admin/health
```

## Run HTTPS With Certificate Files

```bash
go run ./cmd/vimock \
  --https-port 8443 \
  --tls-cert-file ./cert.pem \
  --tls-key-file ./key.pem

curl --http2 -i https://localhost:8443/__admin/health
```

## Environment Variables

```bash
VIMOCK_HTTPS_PORT=8443 \
VIMOCK_TLS_SELF_SIGNED=true \
go run ./cmd/vimock
```

```bash
VIMOCK_HTTPS_PORT=8443 \
VIMOCK_TLS_CERT_FILE=/tls/tls.crt \
VIMOCK_TLS_KEY_FILE=/tls/tls.key \
go run ./cmd/vimock
```

## Docker

```bash
docker build -t vimock:dev .
docker run --rm -p 8080:8080 vimock:dev
```

```bash
docker run --rm \
  -p 8080:8080 \
  -p 8443:8443 \
  vimock:dev \
  --https-port 8443 \
  --tls-self-signed
```

The image exposes ports `8080` and `8443` and defines this healthcheck:

```bash
wget -q -T 2 -O - http://127.0.0.1:8080/__admin/health >/dev/null || exit 1
```

## Kubernetes Probes

Use HTTP probes unless the Pod policy requires probing HTTPS.

Readiness:

```yaml
readinessProbe:
  httpGet:
    path: /__admin/ready
    port: 8080
  periodSeconds: 5
  timeoutSeconds: 2
```

Liveness:

```yaml
livenessProbe:
  httpGet:
    path: /__admin/health
    port: 8080
  periodSeconds: 10
  timeoutSeconds: 2
```

HTTPS with a mounted Kubernetes TLS Secret:

```bash
vimock \
  --https-port 8443 \
  --tls-cert-file /tls/tls.crt \
  --tls-key-file /tls/tls.key
```

## Benchmarks

```bash
go test -run '^$' -bench=. -benchmem ./internal/server ./internal/response
```

Current benchmark targets:

- `BenchmarkRuntimeMatchAndRespondThousandMappings`: full HTTP runtime with 1000 mappings.
- `BenchmarkRenderTemplateJSONPath`: response-template rendering with JSONPath helpers.

## Tests

```bash
go test ./...
go test -race ./internal/mapping ./internal/server ./internal/tlsconfig ./internal/config
go test -run '^$' -bench=. -benchmem ./internal/server ./internal/response
docker build -t vimock:dev .
```

## Current Scope

- HTTPS is process-local configuration; certificate files are read from the container or host filesystem.
- Self-signed certificates are generated in memory on startup and are intended for local/CI smoke checks.
- Kubernetes manifests are not included in this step.
- Docker healthcheck probes the HTTP listener; if HTTP is disabled in a future step, the healthcheck must be adjusted.
