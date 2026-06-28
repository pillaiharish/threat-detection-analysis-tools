package main

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"log/slog"
)

// statusRecorder captures the response status for access logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) { s.status = code; s.ResponseWriter.WriteHeader(code) }
func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = 200
	}
	return s.ResponseWriter.Write(b)
}

// detectNATEgressIP determines the school's public NAT egress IP by asking a
// third-party reflector from inside the intranet vantage. Tries reflectors in
// order; first success wins. Used to populate natEgressIP for the intranet
// vantage so the perimeter nmap -O scan targets the correct host.
var natReflectors = []string{
	"https://api64.ipify.org",
	"https://ifconfig.me",
	"https://icanhazip.com",
}

func detectNATEgressIP() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, url := range natReflectors {
		ip, err := requestReflectIP(client, url)
		if err != nil {
			slog.Warn("reflector failed", "url", url, "err", err)
			continue
		}
		if ip != "" {
			return ip, nil
		}
	}
	return "", errNoReflector
}

var errNoReflector = &reflectErr{"all reflectors failed"}

type reflectErr struct{ msg string }

func (e *reflectErr) Error() string { return e.msg }

// requestReflectIP returns the IP reported by a single reflector endpoint.
// It validates that the response is a parsable IP literal to defend against
// weird/malicious responses from blocked egress (captive portals etc.).
func requestReflectIP(c *http.Client, url string) (string, error) {
	resp, err := c.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", &reflectErr{"not an IP literal: " + ip}
	}
	return ip, nil
}
