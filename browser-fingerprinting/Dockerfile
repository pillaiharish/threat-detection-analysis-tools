# Step 1: build the Go binary for Linux
FROM golang:1.22-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /app/bin
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/server ./server

# Step 2: runtime. NOTE: runs as root INTENTIONALLY. nmap -O requires
# CAP_NET_RAW for raw-socket OS probing. The container only needs to be run on
# networks you are authorized to assess (e.g. your own school intranet for a
# NAT-defeat test). Do NOT expose this image to untrusted networks.
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        nmap bash ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/bin/server /app/server
RUN chmod +x /app/server
COPY web /app/web
COPY scripts /app/scripts

EXPOSE 8080
CMD ["/app/server"]