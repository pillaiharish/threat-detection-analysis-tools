package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_NoXFF_UsesRemoteAddr(t *testing.T) {
	cfg := config{TrustedProxy: ""}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:5678"
	if ip := clientIP(req, cfg); ip != "203.0.113.5" {
		t.Fatalf("expected 203.0.113.5, got %q", ip)
	}
}

func TestClientIP_XFFUntrustedPeer_UsesRemoteAddr(t *testing.T) {
	cfg := config{TrustedProxy: "10.0.0.1"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:5678"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	if ip := clientIP(req, cfg); ip != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1 (peer not trusted), got %q", ip)
	}
}

func TestClientIP_XFFTrustedPeer_UsesLeftmost(t *testing.T) {
	cfg := config{TrustedProxy: "10.0.0.1"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.2")
	if ip := clientIP(req, cfg); ip != "198.51.100.1" {
		t.Fatalf("expected leftmost XFF 198.51.100.1, got %q", ip)
	}
}

func TestClientIP_XFFTrustedPeer_GarbageXFF_FallsBack(t *testing.T) {
	cfg := config{TrustedProxy: "10.0.0.1"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	req.Header.Set("X-Forwarded-For", "not-an-ip, garbage")
	if ip := clientIP(req, cfg); ip != "10.0.0.1" {
		t.Fatalf("expected fallback to peer 10.0.0.1, got %q", ip)
	}
}

func TestClientIP_IPv6Peer(t *testing.T) {
	cfg := config{TrustedProxy: ""}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8::1]:5678"
	if ip := clientIP(req, cfg); ip != "2001:db8::1" {
		t.Fatalf("expected IPv6 2001:db8::1, got %q", ip)
	}
}

func TestClientIP_MalformedRemoteAddr(t *testing.T) {
	cfg := config{TrustedProxy: ""}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "garbage"
	if ip := clientIP(req, cfg); ip != "" {
		t.Fatalf("expected empty for malformed peer, got %q", ip)
	}
}

func TestClientIP_XFFTrustedCIDR(t *testing.T) {
	cfg := config{TrustedProxy: "10.0.0.0/8"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.5.5.5:5678"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	if ip := clientIP(req, cfg); ip != "203.0.113.9" {
		t.Fatalf("expected 203.0.113.9 from XFF within trusted CIDR, got %q", ip)
	}
}

func TestClientIP_XFFTrustedCIDR_OutsideCIDR(t *testing.T) {
	cfg := config{TrustedProxy: "10.0.0.0/8"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:5678"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	if ip := clientIP(req, cfg); ip != "192.168.1.1" {
		t.Fatalf("expected peer (outside trusted CIDR), got %q", ip)
	}
}

func TestTrustedPeer_CIDRMatching(t *testing.T) {
	cfg := config{TrustedProxy: "130.211.0.0/22"}
	if !cfg.trustedPeer("130.211.0.5:1234") {
		t.Fatal("IP within trusted CIDR should be trusted")
	}
	if cfg.trustedPeer("8.8.8.8:1234") {
		t.Fatal("IP outside trusted CIDR should not be trusted")
	}
}

func TestTrustedPeer_CSVMatching(t *testing.T) {
	cfg := config{TrustedProxy: "10.0.0.1,35.191.0.0/16"}
	if !cfg.trustedPeer("10.0.0.1:1234") {
		t.Fatal("CSV member should be trusted")
	}
	if !cfg.trustedPeer("35.191.1.2:1234") {
		t.Fatal("CSV CIDR member should be trusted")
	}
	if cfg.trustedPeer("8.8.8.8:1234") {
		t.Fatal("non-member should not be trusted")
	}
}