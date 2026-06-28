package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"

	"log/slog"
)

// ----- Sanitizers -----------------------------------------------------------

var hexRe = regexp.MustCompile(`^[a-f0-9]+$`)

// sanitizeHash enforces that the client-supplied fingerprint hash is a hex
// string of expected length. We rely on this hash as a correlation key, so a
// bogus client value must not corrupt the audit files via injection.
func sanitizeHash(s string) string {
	s = strings.TrimSpace(s)
	// Strip any embedded CR/LF and control chars defensively.
	s = strings.Map(func(r rune) rune {
		if r < 0x20 {
			return -1
		}
		return r
	}, s)
	if len(s) != 64 || !hexRe.MatchString(strings.ToLower(s)) {
		return sha256Hex("invalid:" + s)
	}
	return strings.ToLower(s)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ----- Middleware ----------------------------------------------------------

// securityHeaders adds basic defensive headers. note: index.html is served
// from the same origin, so CSP allows 'self' for script/style.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// cors adds permissive CORS handling, gated by CORS_ORIGIN. The assessment
// pages are same-origin by default; this is here per operator request so the
// endpoint can also be probed from a different origin if desired.
func cors(next http.Handler, origin string) http.Handler {
	if origin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		h.Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withLogger records per-request metadata to slog at info level. It logs only
// method, path, status, duration, and a remote peer prefix — never raw PII.
func withLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"durationMs", time.Since(start).Milliseconds())
	})
}
