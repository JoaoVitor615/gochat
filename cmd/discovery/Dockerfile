# syntax=docker/dockerfile:1

FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY cmd/discovery ./cmd/discovery
COPY internal/discovery ./internal/discovery

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o discovery ./cmd/discovery

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/discovery .

EXPOSE 8080
EXPOSE 4001/udp

CMD ["./discovery"]
