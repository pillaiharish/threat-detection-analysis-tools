package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeHash_validHex(t *testing.T) {
	in := strings.Repeat("a", 64)
	got := sanitizeHash(in)
	if got != in {
		t.Errorf("expected unchanged, got %s", got)
	}
}

func TestSanitizeHash_uppercaseNormalized(t *testing.T) {
	in := strings.Repeat("A", 64)
	got := sanitizeHash(in)
	if got != strings.ToLower(in) {
		t.Errorf("expected lowercased, got %s", got)
	}
}

func TestSanitizeHash_injectNewline(t *testing.T) {
	in := "a\nb\n" + strings.Repeat("c", 60)
	got := sanitizeHash(in)
	// Must not contain CR/LF and must be 64 hex chars (it's a digest fallback).
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("contains newline: %q", got)
	}
	if len(got) != 64 {
		t.Errorf("expected 64 chars, got %d (%q)", len(got), got)
	}
}

func TestSanitizeHash_wrongLength(t *testing.T) {
	got := sanitizeHash("abc")
	if len(got) != 64 {
		t.Errorf("expected digest fallback of length 64, got %d", len(got))
	}
}

func TestShortHash(t *testing.T) {
	if got := shortHash(strings.Repeat("z", 64)); got != strings.Repeat("z", 16) {
		t.Errorf("expected first 16 chars, got %s", got)
	}
	if got := shortHash("short"); got != "short" {
		t.Errorf("expected unchanged, got %s", got)
	}
}

func TestExtractVisitorIP_noProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/store-info", strings.NewReader("{}"))
	r.RemoteAddr = "10.0.0.5:54321"
	ip, used := extractVisitorIP(r, "")
	if ip != "10.0.0.5" || used {
		t.Errorf("got ip=%q used=%v", ip, used)
	}
}

func TestExtractVisitorIP_untrustedPeerIgnoresXFF(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/store-info", strings.NewReader("{}"))
	r.RemoteAddr = "8.8.8.8:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	ip, used := extractVisitorIP(r, "10.0.0.1")
	if ip != "8.8.8.8" || used {
		t.Errorf("expected peer used, got ip=%q used=%v", ip, used)
	}
}

func TestExtractVisitorIP_honorsXFFFromTrustedProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/store-info", strings.NewReader("{}"))
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	ip, used := extractVisitorIP(r, "10.0.0.1")
	if ip != "203.0.113.7" || !used {
		t.Errorf("expected first XFF hop used, got ip=%q used=%v", ip, used)
	}
}

func TestSanitizeIPLiteral(t *testing.T) {
	cases := map[string]string{
		"1.2.3.4":                  "1.2.3.4",
		"::1":                      "::1",
		"2601:dead::beef":          "2601:dead::beef",
		"1.2.3.4; rm -rf /":        "",
		"\n1.2.3.4":                "1.2.3.4",
		"not-an-ip":                "",
		"":                        "",
	}
	for in, want := range cases {
		if got := sanitizeIPLiteral(in); got != want {
			t.Errorf("sanitizeIPLiteral(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitTrim(t *testing.T) {
	got := splitTrim("a, b ,, c", ",")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("idx %d: got %q want %q", i, got[i], want[i])
		}
	}
}