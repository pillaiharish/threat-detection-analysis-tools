FROM golang:1.22-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /ip-logger .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN groupadd -r app && useradd -r -g app -s /sbin/nologin app
COPY --from=builder /ip-logger /app/ip-logger
COPY static /app/static

ENV PORT=8080

USER app

EXPOSE 8080

ENTRYPOINT ["/app/ip-logger"]