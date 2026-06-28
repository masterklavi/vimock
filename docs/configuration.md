# Configuration

## CLI

```bash
vimock --host 0.0.0.0 --port 8080
```

Available options:

- `--host`
- `--port`
- `--https-port`
- `--tls-cert-file`
- `--tls-key-file`
- `--tls-self-signed`
- `--version`

## Environment Variables

- `VIMOCK_HOST`
- `VIMOCK_PORT`
- `VIMOCK_HTTPS_PORT`
- `VIMOCK_TLS_CERT_FILE`
- `VIMOCK_TLS_KEY_FILE`
- `VIMOCK_TLS_SELF_SIGNED`

Example:

```bash
VIMOCK_HOST=127.0.0.1 VIMOCK_PORT=9090 go run ./cmd/vimock
```

## HTTPS

Self-signed certificate for local or CI checks:

```bash
go run ./cmd/vimock --https-port 8443 --tls-self-signed
curl -k --http2 https://localhost:8443/__admin/health
```

Certificate files:

```bash
vimock \
  --https-port 8443 \
  --tls-cert-file ./cert.pem \
  --tls-key-file ./key.pem
```

## Docker

```bash
docker build -t vimock:dev .
docker run --rm -p 8080:8080 vimock:dev
```

HTTPS in Docker:

```bash
docker run --rm \
  -p 8080:8080 \
  -p 8443:8443 \
  vimock:dev \
  --https-port 8443 \
  --tls-self-signed
```

## Kubernetes Probes

Readiness:

```yaml
readinessProbe:
  httpGet:
    path: /__admin/ready
    port: 8080
```

Liveness:

```yaml
livenessProbe:
  httpGet:
    path: /__admin/health
    port: 8080
```
