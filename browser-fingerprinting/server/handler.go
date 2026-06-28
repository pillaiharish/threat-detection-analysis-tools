package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// storeInfoHandler builds the POST /store-info handler. It closes over config
// and the shared store/scanner so it can validate input, persist a record, and
// enqueue an OS scan of both the visitor's IP and the school's public NAT
// egress IP (skipping duplicates when they coincide, e.g. vantage=internet).
func storeInfoHandler(cfg config, store *store, scan *scanner) http.HandlerFunc {
	type payload struct {
		FingerprintHash string          `json:"fingerprintHash"`
		Signals         json.RawMessage `json:"signals"`
	}
	type visitor struct {
		IP              string
		ProxyUsed       bool
		FingerprintHash string
		Signals         json.RawMessage
		ReceivedAt      time.Time
	}

	var limiter = NewVisitorLimiter(rate.Limit(cfg.rateRefill)/rate.Limit(10), cfg.rateBurst)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodOptions {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Identify the visitor from the TCP-layer peer, honoring X-Forwarded-For
		// only when a trusted proxy is configured. This replaces the previous
		// client-supplied ipAddress field which was trivially spoofable.
		visitorIP, usedProxy := extractVisitorIP(r, cfg.trustedProxy)
		visitorIP = sanitizeIPLiteral(visitorIP)

		if !limiter.Allow(visitorIP) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}

		// Bound the body size; the payload is small.
		r.Body = http.MaxBytesReader(w, r.Body, cfg.maxBodyBytes)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var p payload
		if err := dec.Decode(&p); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		p.FingerprintHash = sanitizeHash(p.FingerprintHash)
		// Keep raw signals but bound their length; they're echoed to disk for
		// triage analysis. Reject if absurdly large.
		if int64(len(p.Signals)) > cfg.maxBodyBytes {
			http.Error(w, "signals too large", http.StatusRequestEntityTooLarge)
			return
		}

		short := shortHash(p.FingerprintHash)

		// Persist the human-readable record + the CSV index row under one lock.
		rec := record{
			Timestamp:         time.Now().UTC(),
			Vantage:           string(cfg.vantage),
			VisitorIP:         visitorIP,
			ProxyUsed:         usedProxy,
			NatEgressIP:       cfg.natEgressIP,
			FingerprintHash:   p.FingerprintHash,
			ShortHash:         short,
			Signals:           p.Signals,
			ScanStatusVisitor: scanPending,
			ScanStatusNAT:     scanPending,
		}
		store.append(rec)

		// Enqueue scans asynchronously. Each scans returns via a callback that
		// stamps the result back onto the index + the human file.
		scanTargetVisitor := visitorIP
		scanTargetNAT := cfg.natEgressIP

		// Vantage=internet: visitor IP == NAT egress; scan once and reuse result.
		if cfg.vantage == vantageInternet {
			scanTargetNAT = ""
		}

		// For the intranet case, do not nmap-empty (if NAT egress unknown).
		if scanTargetVisitor != "" {
			scan.enqueue(scanJob{
				ip:        scanTargetVisitor,
				role:      scanRoleVisitor,
				stamp:     rec.Timestamp,
				visitorIP: visitorIP,
				vantage:   string(cfg.vantage),
				shortHash: short,
			}, func(res scanResult) {
				store.stampScan(rec.Timestamp, scanRoleVisitor, res)
				slog.Info("scan complete", "role", "visitor", "ip", res.IP,
					"status", res.Status, "durationMs", res.DurationMs)
			})
		}
		if scanTargetNAT != "" && scanTargetNAT != scanTargetVisitor {
			scan.enqueue(scanJob{
				ip:        scanTargetNAT,
				role:      scanRoleNAT,
				stamp:     rec.Timestamp,
				visitorIP: visitorIP,
				vantage:   string(cfg.vantage),
				shortHash: short,
			}, func(res scanResult) {
				store.stampScan(rec.Timestamp, scanRoleNAT, res)
				slog.Info("scan complete", "role", "nat", "ip", res.IP,
					"status", res.Status, "durationMs", res.DurationMs)
			})
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// extractVisitorIP returns the visitor's IP and whether a proxy header was
// consulted. When trustedProxy is empty, X-Forwarded-For is ignored and we
// use the TCP peer directly (most secure default). When trustedProxy is set,
// we consult X-Forwarded-For only if the immediate peer matches it.
func extractVisitorIP(r *http.Request, trustedProxy string) (ip string, usedProxy bool) {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	peer = strings.TrimSpace(peer)
	if trustedProxy == "" {
		return peer, false
	}
	// Only honor XFF if the immediate peer is our trusted proxy.
	if peer != trustedProxy {
		return peer, false
	}
	xff := r.Header.Get("X-Forwarded-For")
	parts := splitTrim(xff, ",")
	if len(parts) > 0 {
		return parts[0], true
	}
	return peer, false
}

// sanitizeIPLiteral validates that the input looks like an IPv4/IPv6 literal.
// Returns "" if invalid. We rely primarily on the TCP peer for this value, but
// defense in depth: never trust a header blindly.
func sanitizeIPLiteral(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if net.ParseIP(s) != nil {
		return s
	}
	return ""
}

// shortHash returns the first 16 hex chars of a SHA-256 hex string for
// log/triage display.
func shortHash(h string) string {
	if len(h) < 16 {
		return h
	}
	return h[:16]
}

// VisitorLimiter is a per-IP token bucket limiter. Used to stop a single
// laptop reloading the page from spamming scans and the audit log.
type VisitorLimiter struct {
	buckets map[string]*rate.Limiter
	per     rate.Limit
	burst   int
}

func NewVisitorLimiter(per rate.Limit, burst int) *VisitorLimiter {
	return &VisitorLimiter{buckets: make(map[string]*rate.Limiter), per: per, burst: burst}
}

func (vl *VisitorLimiter) Allow(ip string) bool {
	if ip == "" {
		return true
	}
	l, ok := vl.buckets[ip]
	if !ok {
		l = rate.NewLimiter(vl.per, vl.burst)
		vl.buckets[ip] = l
	}
	return l.Allow()
}
