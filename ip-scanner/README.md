# Go IP Address Scanner

A dependency-free (std-lib only) local-network device scanner. Unlike the old
ICMP-based version, this uses **TCP connect probes**, so it runs **without
`sudo`** and specifically identifies hosts that are *reachable* — routers that
respond to a port, laptops with SSH running, printers exposing IPP, cameras
serving RTSP, AirPlay speakers, etc. Phantom "route" addresses that never
respond to anything simply don't appear, so the output is just real devices.

## Features

- **Multi-interface auto-detect.** Walks `net.Interfaces()`, selects every
  active non-virtual IPv4 interface (skips loopback, `utun*`, `awdl*`, `llw*`,
  `bridge`, `gif`, `stf`), and scans each one's subnet. So if you have Wi-Fi
  (`en0`, e.g. `192.168.1.0/24`) and a wired USB-Ethernet peer (`en7`, e.g.
  `10.0.0.0/30`), both networks are scanned in one run and reported under
  separate per-interface section headers.
- **Multi-port fingerprint.** Probes a configurable TCP port set per host
  (default `22,80,443,554,5000,631,8080`). A host is reported if **any** port
  accepts a connection, and the output lists which ports answered — making it
  easy to tell SSH from a printer from a camera.
- **Per-host identification.** For every live host it prints:
  - IP address
  - which ports answered
  - reverse-DNS hostname (via `net.LookupAddr`)
  - MAC address (from the system ARP cache, refreshed by an active UDP poke
    if missing)
  - vendor guess from an embedded OUI table (Apple, TP-Link, Asus, Netgear,
    D-Link, Linksys, Amazon, Google, Sonos, Roku, Bose, Raspberry-Pi, Intel,
    Realtek, Broadcom — enough to flag your Mac mini as `VENDOR Apple`).
- **No device-type bias.** Phones, laptops, IoT, printers, routers, switches,
  Raspberry Pis — anything that accepts a TCP connection on a probed port is
  reported identically.
- **Pure stdlib, zero cgo.** No more `dyld: missing LC_UUID load command`
  crashes on current macOS; no `go-ping` dependency; builds with a plain
  `go build`, runs unprivileged.

## Prerequisites

- **Go** (any reasonably recent 1.x; tested with go1.22.2 on darwin/arm64).

That's it. No `sudo`, no third-party packages, no network privileges.

## Usage

```bash
# Auto-detect every active interface and scan its subnet with the default port set
go run main.go

# Restrict to specific subnets (skips auto-detect)
go run main.go -subnets 192.168.1.0/24,10.0.0.0/30

# Probe only SSH, faster
go run main.go -ports 22

# Tune timeout and concurrency
go run main.go -timeout 500ms -workers 128
```

Flags:

| Flag        | Default                       | Meaning                                      |
|-------------|-------------------------------|----------------------------------------------|
| `-ports`    | `22,80,443,554,5000,631,8080` | comma-separated TCP ports to probe per host  |
| `-subnets`  | (empty -> auto-detect)        | comma-separated CIDRs to scan                |
| `-timeout`  | `800ms`                       | per-port `net.DialTimeout`                   |
| `-workers`  | `64`                          | concurrent host-scan workers                 |

## Sample output

```
$ go run main.go
Discovering hosts across 3 interface(s) probing ports [22 80 443 554 5000 631 8080]

== en0    (Wi-Fi)      192.168.1.50/24 (my ip 192.168.1.50) - 254 targets ==
-- reachable (a probed port answered) --
IP 192.168.1.50      PORTS [5000]                    HOST -                          MAC a4:5e:60:11:22:33  VENDOR Apple
-- ARP-known but no open port (live on L2, SSH/other services not listening) --
IP 192.168.1.1       PORTS [none]                    HOST -                          MAC 74:da:88:aa:bb:cc  VENDOR TP-Link
IP 192.168.1.42      PORTS [none]                    HOST -                          MAC 9e:b7:46:dd:ee:ff  VENDOR Apple

== en7    (USB 10/100/1000 LAN) 10.0.0.1/30 (my ip 10.0.0.1) - 2 targets ==
-- reachable (a probed port answered) --
IP 10.0.0.1          PORTS [5000]                    HOST -                          MAC 3c:18:a0:11:22:33  VENDOR ASIX

== en7    (USB 10/100/1000 LAN) ARP-discovered peers (my ip -) - 1 targets ==
-- ARP-known but no open port (live on L2, SSH/other services not listening) --
IP 169.254.10.20     PORTS [none]                    HOST mac-mini.local             MAC 1c:f6:4c:11:22:33  VENDOR Apple

Scan complete: 2 port-reachable host(s), 3 ARP-known-only candidate(s), across 3 interface(s)
```

Two important things this output shows:

1. **The wired Mac mini at `169.254.10.20` is detected by name** (`mac-mini.local`, vendor `Apple`), even though it's on a `169.254/16` link-local address that no configured subnet reaches, and even though it's not yet accepting SSH. The scanner harvests the system ARP table to find peers that exist on layer-2 but fall outside any subnet, so you don't miss them.
2. **It's in the "ARP-known but no open port" section** — meaning the mini is alive on the wire but SSH isn't accepting connections. Enable Remote Login on the mini (System Settings → General → Sharing → Remote Login), then re-run; the mini will move into the "reachable" section and you can `ssh user@169.254.10.20` (or whichever IP it moves to).

## Why two result sections per interface

For each interface the scanner prints two groups:

- **reachable** — a probed TCP port (default `22,80,443,554,5000,631,8080`) actually accepted a connection. These are hosts you can probably connect to.
- **ARP-known but no open port** — the host was found in the system ARP table with a real MAC (so it's live at layer-2) but didn't answer any probed port. These are still real devices on your network; they're just not running anything we probed. This is how a Mac mini with Remote Login disabled, an iPhone, a smart speaker, etc. show up so you can spot them.

## What "IP is up" used to mean vs now

The old ICMP tool reported any host that replied to ping. The new tool:

- Probes TCP ports — a "reachable" entry is something you can actually connect to.
- Falls back to the ARP cache to surface layer-2-present hosts that didn't accept a TCP connection, so you don't miss sleeping or filtered devices — useful when "which IP belongs to my Mac mini" is the question.

## Why the rewrite

The original used `github.com/go-ping/ping`, which pulls in `CoreFoundation`
via cgo. Go 1.22.2 emits that linkage in a way macOS 26's `dyld` rejects
(`missing LC_UUID load command`), so `go run main.go` crashed instantly.
Even with `CGO_ENABLED=0`, go-ping's unprivileged UDP fallback fails on
current macOS (`sendto: no route to host`). Switching to stdlib TCP probes
fixes both problems and gives an SSH-aware result in one stroke.