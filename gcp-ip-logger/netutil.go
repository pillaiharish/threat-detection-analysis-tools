package main

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// clientIP derives the visitor's IP from the request.
// When TRUSTED_PROXY is set and the immediate peer matches it, the
// leftmost address in X-Forwarded-For is used. Otherwise the peer
// address from r.RemoteAddr is used.
func clientIP(r *http.Request, cfg config) string {
	if cfg.trustedPeer(r.RemoteAddr) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			for _, part := range strings.Split(xff, ",") {
				candidate := strings.TrimSpace(part)
				if candidate != "" && net.ParseIP(candidate) != nil {
					return candidate
				}
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if net.ParseIP(host) == nil {
		return ""
	}
	return host
}

// fetchMetadataProjectID reads the project ID from the GCE metadata server.
// Returns empty string if unavailable (non-GCP hosts).
func fetchMetadataProjectID() (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET",
		"http://metadata.google.internal/computeMetadata/v1/project/project-id",
		nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}