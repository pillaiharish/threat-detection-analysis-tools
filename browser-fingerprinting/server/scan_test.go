package main

import (
	"errors"
	"strings"
	"testing"
)

func TestParseScanStatus_success(t *testing.T) {
	out := `Nmap scan report for 10.0.0.5
Host is up (0.001s latency).
OS details: Linux 3.0 - 4.0
`
	s := parseScanStatus(out, nil)
	if s != scanSuccess {
		t.Errorf("expected success, got %s", s)
	}
	if g := parseOSGuess(out); !strings.Contains(g, "Linux") {
		t.Errorf("expected Linux guess, got %q", g)
	}
}

func TestParseScanStatus_osHidden(t *testing.T) {
	out := `Nmap scan report for 10.0.0.5
Host is up (0.001s latency).
Too many fingerprints match this host
`
	s := parseScanStatus(out, errors.New("exit 1"))
	if s != scanOSHidden {
		t.Errorf("expected os_hidden, got %s", s)
	}
}

func TestParseScanStatus_hostDown(t *testing.T) {
	out := `Starting Nmap
0 hosts up
`
	s := parseScanStatus(out, errors.New("exit 1"))
	if s != scanHostDown {
		t.Errorf("expected host_down, got %s", s)
	}
}

func TestParseScanStatus_requiresRoot(t *testing.T) {
	out := "TCP/IP fingerprinting (OS scan) requires root privileges.\nQUITTING!\n"
	s := parseScanStatus(out, errors.New("exit 1"))
	if s != scanError {
		t.Errorf("expected error, got %s", s)
	}
}

func TestScannerCache(t *testing.T) {
	s := newScanner("/nonexistent/fingerprint.sh", 1<<30, 1, 0)
	called := 0
	cb := func(scanResult) { called++ }
	s.enqueue(scanJob{ip: "1.1.1.1", cb: cb}, nil)
	s.enqueue(scanJob{ip: "1.1.1.1", cb: cb}, nil)
	s.mu.Lock()
	_, ok := s.cache["1.1.1.1"]
	s.mu.Unlock()
	if !ok {
		t.Errorf("expected cache entry for 1.1.1.1")
	}
}