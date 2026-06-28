# port-scanner

A concurrent TCP port scanner written in Go. Scans one or more hosts
(IPs, hostnames, or CIDR ranges) for open TCP ports, with optional
banner grabbing and JSON/CSV output for pipeline integration.

## Authorized use

This tool is intended **only** for scanning systems you own or have
explicit written authorization to test. Unauthorized port scanning may
violate law and acceptable-use policies.

## Features

- Concurrent TCP connect scan with configurable concurrency
- Multiple targets: comma-separated IPs, hostnames, or CIDR ranges
- Mixed port specs: ranges + lists (`1-1000,22,80,443,8080-9000`)
- Optional banner grab on open ports (`-banner`)
- Output formats: text (default), JSON (`-json`), CSV (`-csv`)
- De-duplication: duplicate hosts/ports in input are scanned once
- CIDR safety limit: refuses ranges exceeding 65536 hosts

## Usage

```bash
# Basic scan — default ports 1-1024
go run main.go -target 192.168.1.5

# Multiple hosts + CIDR + custom ports + banner grab
go run main.go -target 192.168.1.0/30,10.0.0.5 -ports 22,80,443,1000-2000 -banner

# JSON output for pipeline
go run main.go -target scanme.nmap.org -ports 1-1024 -json | jq .

# CSV output
go run main.go -target 10.0.0.0/29 -ports 1-100 -csv > scan.csv
```

## Flags

| Flag | Type | Default | Purpose |
|---|---|---|---|
| `-target` | string | (required) | comma-separated IPs, hostnames, or CIDR ranges |
| `-ports` | string | `1-1024` | port spec: ranges + lists (e.g. `1-1000,22,80,443,8080-9000`) |
| `-timeout` | duration | `2s` | per-port TCP connect timeout |
| `-concurrency` | int | `100` | max concurrent port probes |
| `-banner` | bool | `false` | grab banners on open ports |
| `-json` | bool | `false` | JSON output to stdout |
| `-csv` | bool | `false` | CSV output to stdout (column header included) |
| `-version` | bool | `false` | print version and exit |

`-json` and `-csv` are mutually exclusive.

## Build

```bash
cd port-scanner
go build -o port-scanner .
./port-scanner -target 127.0.0.1 -ports 1-100
```

## Test

```bash
cd port-scanner
go vet ./...
go test ./...
```

## Output formats

**Text (default):**
```
192.168.1.5:
  22     open  1ms
  80     open  1ms  (banner: SSH-2.0-OpenSSH_8.9)
  443    open  2ms
10.0.0.5: no open ports
```

**JSON:** Array of `hostResult` objects containing `host` and `ports[]`
with `port`, `state`, `banner` (if grabbed), and `latency_ms`.

**CSV:** Columns: `host,port,state,latency_ms,banner`. Hosts with no open
ports emit one row with empty port and state `closed`.

## Notes

- Port 0 is invalid and rejected. Valid range: 1-65535.
- CIDR expansion limit: 65536 hosts. `/16` OK; `/8` rejected.
- Banner reading uses a 500ms read deadline — protocols that don't send
  an immediate banner (e.g. HTTP) yield an empty banner string.
- Hostname resolution: only IPv4 results are used.