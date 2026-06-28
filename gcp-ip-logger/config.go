package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type config struct {
	Port            string
	AdminToken      string
	TrustedProxy    string
	RateLimitRPS    float64
	RateLimitBurst  int
	GCPProjectID    string
	LogName         string
	MaxBody         int64
}

const maxBodyBytes = 4 * 1024

func loadConfig() config {
	cfg := config{
		Port:           envOr("PORT", "8080"),
		AdminToken:     os.Getenv("ADMIN_TOKEN"),
		TrustedProxy:   os.Getenv("TRUSTED_PROXY"),
		GCPProjectID:   os.Getenv("GCP_PROJECT_ID"),
		LogName:        envOr("LOG_NAME", "ip-logger"),
		RateLimitRPS:   envFloat("RATE_LIMIT_RPS", 1.0),
		RateLimitBurst: envInt("RATE_LIMIT_BURST", 3),
		MaxBody:        maxBodyBytes,
	}

	if cfg.GCPProjectID == "" {
		if pid, err := gcpProjectFromMetadata(); err == nil && pid != "" {
			cfg.GCPProjectID = pid
		}
	}

	if cfg.AdminToken == "" {
		logf("WARNING: ADMIN_TOKEN unset — /ip-lists will return 503")
	}

	return cfg
}

func (c config) trustedPeer(raddr string) bool {
	if c.TrustedProxy == "" {
		return false
	}
	host, _, err := net.SplitHostPort(raddr)
	if err != nil {
		host = raddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, member := range strings.Split(c.TrustedProxy, ",") {
		member = strings.TrimSpace(member)
		if member == "" {
			continue
		}
		if strings.Contains(member, "/") {
			if _, network, err := net.ParseCIDR(member); err == nil && network.Contains(ip) {
				return true
			}
		} else {
			if member == host {
				return true
			}
		}
	}
	return false
}

func envOr(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}

func envFloat(key string, dflt float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return dflt
}

func envInt(key string, dflt int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return dflt
}

func gcpProjectFromMetadata() (string, error) {
	// Lazy import to avoid hard dependency at compile time on platforms
	// where the metadata server isn't available.
	return fetchMetadataProjectID()
}

func logf(format string, args ...any) {
	fmt.Printf("[config] "+format+"\n", args...)
}