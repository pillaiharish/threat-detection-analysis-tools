# gcp-ip-logger

Authorized-use IP logger for security awareness training on Google Cloud
GKE. Visitor IPs are captured server-side and streamed to Google Cloud
Logging. A Bearer-token-protected `/ip-lists` endpoint returns recent
entries for quick triage.

## Authorized use

This tool is intended **only** for authorized security awareness training
within an organization that has approved its use. Do not deploy against
users without authorization.

## How it works

- Visitor loads `/` — a neutral "security awareness training" page.
- Browser POSTs to `/log-ip` with only a timestamp. The server derives
  the visitor's IP from the TCP connection (or `X-Forwarded-For` when
  behind a trusted load balancer), **not** from any client-supplied field.
- The IP is written to Google Cloud Logging and stored in an in-memory
  ring buffer (last 200 entries per pod).
- `/ip-lists` returns the ring buffer as JSON behind Bearer-token auth.
- Full history lives in Cloud Logging, not in the pod.

## Environment variables

| Name | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `ADMIN_TOKEN` | (none) | Bearer token for `/ip-lists`. **Required** — if unset, returns 503. |
| `TRUSTED_PROXY` | (none) | Comma-separated IPs/CIDRs allowed to set `X-Forwarded-For`. Typically the GCLB ranges. |
| `GCP_PROJECT_ID` | (auto-detected via GCE metadata) | Cloud Logging project ID |
| `LOG_NAME` | `ip-logger` | Cloud Logging log name |
| `RATE_LIMIT_RPS` | `1` | Per-visitor request rate |
| `RATE_LIMIT_BURST` | `3` | Per-visitor burst size |

## Deploy

```bash
# 1. Build and push (replace YOUR_GCP_PROJECT_ID)
docker build -t gcr.io/YOUR_GCP_PROJECT_ID/ip-logger:v1 .
docker push gcr.io/YOUR_GCP_PROJECT_ID/ip-logger:v1

# 2. Create the secret (edit secret.yaml with a long random token first)
kubectl apply -f secret.yaml

# 3. Deploy (edit deployment.yaml image path first)
kubectl apply -f deployment.yaml
kubectl apply -f ingress.yaml
```

The pod's service account (or node workload identity) needs
`roles/logging.logWriter` on the project.

## Retrieve logs

**Recent (in-memory, per pod):**
```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" https://<ingress-ip>/ip-lists
```

**Full history (Cloud Logging):**
```bash
gcloud logging read \
  'logName="projects/YOUR_GCP_PROJECT_ID/logs/ip-logger"' \
  --project=YOUR_GCP_PROJECT_ID \
  --format=json | jq .[]
```

## Endpoints

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET  | `/`         | none | Neutral awareness-training landing page |
| POST | `/log-ip`   | rate-limited | Body: `{"timestamp":"RFC3339"}`. IP derived from connection. |
| GET  | `/ip-lists` | Bearer `ADMIN_TOKEN` | JSON array of recent entries |
| GET  | `/healthz`  | none | `OK` — for k8s probes |

## Test

```bash
cd gcp-ip-logger
go vet ./...
go test ./...
```

## Notes

- Cloud Logging init requires `GCP_PROJECT_ID` or GCE metadata access.
  Local (non-GCP) runs will fail to start — this tool is cloud-only.
- `/ip-lists` shows only the last 200 entries on that pod. With multiple
  replicas, use Cloud Logging for full history.