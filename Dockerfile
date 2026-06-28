# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/vimock ./cmd/vimock

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& addgroup -S vimock \
	&& adduser -S -D -H -G vimock vimock

COPY --from=build /out/vimock /usr/local/bin/vimock

USER vimock:vimock
EXPOSE 8080 8443
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD wget -q -T 2 -O - http://127.0.0.1:8080/__admin/health >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/vimock"]
