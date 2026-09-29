FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o discovery ./cmd/discovery

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/discovery .

EXPOSE 8080

CMD ["./discovery"]