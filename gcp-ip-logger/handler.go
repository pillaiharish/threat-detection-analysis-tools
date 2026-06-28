package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type App struct {
	cfg    config
	logger *cloudLogger

	mu      sync.Mutex
	visitors map[string]*rate.Limiter // per-IP rate limiters
	logs    []IPLog                    // ring buffer for /ip-lists
}

type IPLog struct {
	IP        string `json:"ip"`
	Timestamp string `json:"timestamp"`
	UserAgent string `json:"user_agent"`
	Path      string `json:"path"`
}

const ringBufferSize = 200

func (a *App) limiterFor(ip string) *rate.Limiter {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.visitors == nil {
		a.visitors = make(map[string]*rate.Limiter)
	}
	lim, ok := a.visitors[ip]
	if !ok {
		lim = rate.NewLimiter(rate.Limit(a.cfg.RateLimitRPS), a.cfg.RateLimitBurst)
		a.visitors[ip] = lim
	}
	return lim
}

func (a *App) recordLog(entry IPLog) {
	a.mu.Lock()
	a.logs = append(a.logs, entry)
	if len(a.logs) > ringBufferSize {
		a.logs = a.logs[len(a.logs)-ringBufferSize:]
	}
	a.mu.Unlock()
	a.logger.log(context.Background(), entry)
}

func (a *App) logIPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := clientIP(r, a.cfg)

	if !validIP(ip) {
		http.Error(w, "could not determine client IP", http.StatusBadRequest)
		return
	}

	lim := a.limiterFor(ip)
	if !lim.Allow() {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, a.cfg.MaxBody)

	var body struct {
		Timestamp string `json:"timestamp"`
		IP        string `json:"ip"` // accepted but ignored; server-derived IP wins
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, a.cfg.MaxBody)).Decode(&body); err != nil {
		if err == io.EOF {
			body.Timestamp = time.Now().UTC().Format(time.RFC3339)
		} else {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
	}

	ts := body.Timestamp
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	} else if !validTimestamp(ts) {
		http.Error(w, "invalid timestamp", http.StatusBadRequest)
		return
	}

	ua := sanitizeUserAgent(r.UserAgent())
	path := sanitizePath(r.Header.Get("Referer"))

	entry := IPLog{
		IP:        ip,
		Timestamp: ts,
		UserAgent: ua,
		Path:      path,
	}

	a.recordLog(entry)

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) ipListsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.cfg.AdminToken == "" {
		http.Error(w, "ADMIN_TOKEN not configured", http.StatusServiceUnavailable)
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if !constantTimeEqual(token, a.cfg.AdminToken) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	a.mu.Lock()
	logsCopy := make([]IPLog, len(a.logs))
	copy(logsCopy, a.logs)
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logsCopy); err != nil {
		log.Printf("ip-lists encode error: %v", err)
	}
}

func validIP(ip string) bool {
	if ip == "" || len(ip) > 45 {
		return false
	}
	if strings.ContainsAny(ip, " \t\r\n\"'<>\\") {
		return false
	}
	return net.ParseIP(ip) != nil
}

func validTimestamp(ts string) bool {
	if len(ts) > 64 || strings.ContainsAny(ts, "\r\n\"\\") {
		return false
	}
	if _, err := time.Parse(time.RFC3339, ts); err == nil {
		return true
	}
	_, err := time.Parse("2006-01-02T15:04:05Z", ts)
	return err == nil
}

func sanitizeUserAgent(ua string) string {
	if len(ua) > 256 {
		ua = ua[:256]
	}
	ua = strings.TrimSpace(ua)
	ua = strings.ReplaceAll(ua, "\r", "")
	ua = strings.ReplaceAll(ua, "\n", "")
	return ua
}

func sanitizePath(p string) string {
	if len(p) > 256 {
		p = p[:256]
	}
	return strings.TrimSpace(p)
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}