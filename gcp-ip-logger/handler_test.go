package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestApp(token string) *App {
	sink := &stubSink{}
	return &App{
		cfg: config{
			AdminToken:     token,
			RateLimitRPS:   100,
			RateLimitBurst: 10,
			MaxBody:        maxBodyBytes,
		},
		logger: &cloudLogger{sink: sink},
	}
}

func TestLogIP_RejectsNonPOST(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodGet, "/log-ip", nil)
	rec := httptest.NewRecorder()
	app.logIPHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if rec.Header().Get("Allow") != "POST" {
		t.Fatalf("expected Allow header, got %q", rec.Header().Get("Allow"))
	}
}

func TestLogIP_AcceptsValidPOST(t *testing.T) {
	app := newTestApp("secret")
	body := `{"timestamp":"2024-01-02T15:04:05Z"}`
	req := httptest.NewRequest(http.MethodPost, "/log-ip", strings.NewReader(body))
	req.RemoteAddr = "203.0.113.5:5678"
	rec := httptest.NewRecorder()
	app.logIPHandler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestLogIP_RejectsInvalidJSON(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodPost, "/log-ip", strings.NewReader("{bad json"))
	req.RemoteAddr = "203.0.113.5:5678"
	rec := httptest.NewRecorder()
	app.logIPHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLogIP_RejectsBadTimestamp(t *testing.T) {
	app := newTestApp("secret")
	body := `{"timestamp":"not-a-time"}`
	req := httptest.NewRequest(http.MethodPost, "/log-ip", strings.NewReader(body))
	req.RemoteAddr = "203.0.113.5:5678"
	rec := httptest.NewRecorder()
	app.logIPHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLogIP_IgnoresClientSentIP(t *testing.T) {
	app := newTestApp("secret")
	sink := app.logger.sink.(*stubSink)
	body := `{"ip":"1.2.3.999","timestamp":"2024-01-02T15:04:05Z"}`
	req := httptest.NewRequest(http.MethodPost, "/log-ip", strings.NewReader(body))
	req.RemoteAddr = "203.0.113.5:5678"
	rec := httptest.NewRecorder()
	app.logIPHandler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if len(sink.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(sink.entries))
	}
	if sink.entries[0].IP != "203.0.113.5" {
		t.Fatalf("server should derive IP, got %q", sink.entries[0].IP)
	}
}

func TestLogIP_RateLimited(t *testing.T) {
	app := newTestApp("secret")
	app.cfg.RateLimitRPS = 0.001
	app.cfg.RateLimitBurst = 1
	body := `{"timestamp":"2024-01-02T15:04:05Z"}`
	// First request passes.
	req := httptest.NewRequest(http.MethodPost, "/log-ip", strings.NewReader(body))
	req.RemoteAddr = "203.0.113.5:5678"
	rec := httptest.NewRecorder()
	app.logIPHandler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("first request should succeed, got %d", rec.Code)
	}
	// Second request triggers rate limit.
	req2 := httptest.NewRequest(http.MethodPost, "/log-ip", strings.NewReader(body))
	req2.RemoteAddr = "203.0.113.5:5678"
	rec2 := httptest.NewRecorder()
	app.logIPHandler(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}
}

func TestIPLists_RequiresBearer(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodGet, "/ip-lists", nil)
	rec := httptest.NewRecorder()
	app.ipListsHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestIPLists_RejectsWrongToken(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodGet, "/ip-lists", nil)
	req.Header.Set("Authorization", "Bearer wrongtoken")
	rec := httptest.NewRecorder()
	app.ipListsHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestIPLists_ReturnsEntriesWithValidToken(t *testing.T) {
	app := newTestApp("sekrit")
	sink := app.logger.sink.(*stubSink)
	// record one entry directly
	app.recordLog(IPLog{IP: "198.51.100.1", Timestamp: "2024-01-02T15:04:05Z"})
	_ = sink

	req := httptest.NewRequest(http.MethodGet, "/ip-lists", nil)
	req.Header.Set("Authorization", "Bearer sekrit")
	rec := httptest.NewRecorder()
	app.ipListsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var entries []IPLog
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(entries) != 1 || entries[0].IP != "198.51.100.1" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestIPLists_AdminTokenUnsetReturns503(t *testing.T) {
	app := newTestApp("")
	req := httptest.NewRequest(http.MethodGet, "/ip-lists", nil)
	req.Header.Set("Authorization", "Bearer whatever")
	rec := httptest.NewRecorder()
	app.ipListsHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestIPLists_RejectsNonGET(t *testing.T) {
	app := newTestApp("secret")
	req := httptest.NewRequest(http.MethodPost, "/ip-lists", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	app.ipListsHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !constantTimeEqual("abc", "abc") {
		t.Fatal("identical strings should match")
	}
	if constantTimeEqual("abc", "abd") {
		t.Fatal("different strings should not match")
	}
	if constantTimeEqual("abc", "abcd") {
		t.Fatal("different lengths should not match")
	}
}

func TestSanitizeUserAgent(t *testing.T) {
	out := sanitizeUserAgent("Mozilla/5.0\r\nEvil: header\r\n")
	if strings.Contains(out, "\r") || strings.Contains(out, "\n") {
		t.Fatalf("CRLF not stripped: %q", out)
	}
	if len(out) > 256 {
		t.Fatalf("UA too long: %d", len(out))
	}
}

func TestValidIP(t *testing.T) {
	cases := map[string]bool{
		"203.0.113.5":    true,
		"::1":            true,
		"":               false,
		"not an ip":      false,
		"1.2.3.4\n":      false,
		strings.Repeat("a", 50): false,
	}
	for ip, want := range cases {
		if got := validIP(ip); got != want {
			t.Errorf("validIP(%q) = %v, want %v", ip, got, want)
		}
	}
}

func TestValidTimestamp(t *testing.T) {
	cases := map[string]bool{
		"2024-01-02T15:04:05Z":       true,
		"2024-01-02T15:04:05+07:00":  true,
		"not-a-time":                  false,
		"":                            false,
		"2024-01-02T15:04:05Z\r\nEvil": false,
	}
	for ts, want := range cases {
		if got := validTimestamp(ts); got != want {
			t.Errorf("validTimestamp(%q) = %v, want %v", ts, got, want)
		}
	}
}

