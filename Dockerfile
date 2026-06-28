# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/vimock ./cmd/vimock

FROM scratch

COPY --from=build /out/vimock /vimock

EXPOSE 8080
ENTRYPOINT ["/vimock"]
