# browser-fingerprinting

Authorized-assessment browser + OS fingerprinting tool. For testing whether
your own school network's NAT boundary defeats OS fingerprinting, and for
per-laptop triage ("which laptop has problems"). Use only on networks you own
or are explicitly authorized to test.

## What it does

- Serves a single page (`web/index.html`) that computes a high-entropy browser
  fingerprint (Canvas, WebGL, AudioContext, fonts, screen set, UA client hints,
  timezone, hardware concurrency, etc.) and a stable SHA-256 `fingerprintHash`.
- The page POSTs the hash + raw signals to `/store-info` on the same origin.
- The server:
  - Derives the visitor's IP from the TCP peer (not from client-supplied data),
    honoring `X-Forwarded-For` only when `TRUSTED_PROXY` is set.
  - Tags each record with a `vantage` (`intranet` or `internet`) so a side-by-side
    run from inside and outside the network can be compared.
  - Writes a human-readable `client_info.txt` block per visit and a grep-able
    `index.csv` row.
  - Enqueues a `nmap -O -T3 --osscan-guess` scan of both the visitor IP and the
    school's public NAT egress IP via `scripts/fingerprint.sh`, with a 30-min
    per-IP cache and a bounded worker pool so repeat reloads don't hammer the
    network. Scan results are appended to both files.
- `index.csv` has a normalized `scanStatus` column with values
  `success`/`os_hidden`/`host_down`/`error`/`pending` so the operator can grep
  for laptops whose OS was hidden by NAT or local firewall.

## Why the container runs as root

`nmap -O` requires raw-socket access (`CAP_NET_RAW`). The original code
attempted this without root and silently failed (`client_info.txt` showed
`requires root privileges. QUITTING!`), which made the entire NAT test
meaningless — you couldn't distinguish "NAT defeated fingerprinting" from
"scanner couldn't run." Running as root is intentional and scoped to your
authorized assessment. Do not run this image on untrusted networks.

## Vantages

Run **two** deployments for a full assessment:

| Vantage     | Where the server runs | What the visitor IP is     | NAT egress IP source |
|-------------|-----------------------|-----------------------------|----------------------|
| `intranet`  | inside the school LAN | laptop's internal LAN IP    | reflector (ipify → ifconfig.me → icanhazip.com) |
| `internet`  | cloud/internet host   | school's public NAT egress  | == visitor IP, no reflector needed |

Correlate records across the two runs by `fingerprintHash` — the same laptop
visiting both servers produces the same hash because the fingerprint is
computed in the browser from the same signals.

## Configure

All env vars optional unless noted.

| Variable           | Default             | Purpose |
|--------------------|---------------------|---------|
| `PORT`             | `8080`              | HTTP port |
| `VANTAGE`          | `intranet`          | `intranet` or `internet` |
| `TXT_PATH`         | `client_info.txt`   | Human-readable audit file |
| `CSV_PATH`         | `index.csv`         | Machine-readable index |
| `SCAN_SCRIPT`      | `scripts/fingerprint.sh` | Path to the nmap wrapper |
| `SCAN_CACHE_TTL`   | `30m`               | Skip re-scanning same IP within this window |
| `SCAN_CONCURRENCY` | `5`                 | Max simultaneous nmap runs |
| `SCAN_TIMEOUT`     | `3m`                | Per-scan wall-clock cap |
| `CORS_ORIGIN`      | `*`                 | `Access-Control-Allow-Origin` value; empty disables CORS |
| `TRUSTED_PROXY`    | (empty)             | If set, XFF only honored when immediate peer matches this |

## Run

```sh
make docker                     # build
make docker-intranet             # run from inside the school LAN
make docker-internet             # run from an internet host
```

Or locally:

```sh
cd browser-fingerprinting
VANTAGE=intranet go run ./server
```

Then have laptops open `http://<server>:8080/` in their browser. Each visit
will be recorded and (if not recently scanned) OS-fingerprinted.

## Triage

After both runs, copy `client_info.txt` and `index.csv` from both deployments
to one directory. Then:

```sh
# Find every visit by a specific laptop across both vantages:
grep <fingerprintHash> index.csv

# Which laptops had their OS hidden from inside the LAN?
awk -F, '$8=="visitor" && $9=="os_hidden"' index.csv

# Did the perimeter block OS fingerprinting from the outside?
awk -F, '$8=="nat" && $9=="os_hidden"' index.csv
```

The human-readable `client_info.txt` is for eyeballing in the lab; the
`[scan visitor]` / `[scan nat]` sub-blocks match back to a visit by its
timestamp.

## Cleanup

After the assessment, delete the audit files:

```sh
make clean   # or: rm -f browser-fingerprinting/client_info.txt browser-fingerprinting/index.csv
```

You are responsible for securely erasing any copies you made of the audit
files. The records contain visitor IPs, user agents, and OS guesses.

## Security notes

- All client-supplied strings are sanitized (CR/LF stripped, length bounded)
  before being written to disk or passed as the nmap argv.
- nmap is invoked with the IP as a single argv element via `exec.Command` — no
  shell interpretation. The script also validates the argument matches an
  IP/host charset.
- Per-IP rate limiting (1 request / 10 s / IP) prevents reload-spam.
- Body size capped at 8 KB; `json.Decoder.DisallowUnknownFields` rejects
  malformed payloads.
- Container logs use `log/slog` structured output and never contain raw PII —
  only short fingerprint hashes and scan statuses.
- The static page sets `Content-Security-Policy`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`.

## Known limitations

- The browser-fingerprint hash relies on `crypto.subtle`; the page must be
  served over HTTPS or `localhost`. On plain HTTP from a non-localhost address,
  `SubtleCrypto` is unavailable → the page reports an error. For the intranet
  vantage you can either serve over HTTPS or have browsers access via
  `http://localhost:8080`-style loopback, or relax CSP/disable the hash.
  Subnet-internal `http://<server-host>:8080` will not compute the hash.
- nmap `-O` against a host that drops all our probes returns `0 hosts up` →
  `scanHostDown`. That's a valid negative result (the boundary is opaque) and is
  reported as such in the CSV.