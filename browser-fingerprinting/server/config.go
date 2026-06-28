package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	port            string
	vantage         Vantage
	natEgressIP     string
	txtPath         string
	csvPath         string
	scanScriptPath  string
	scanCacheTTL    time.Duration
	scanConcurrency int
	scanTimeout     time.Duration
	corsOrigin      string
	trustedProxy    string
	logLevel        slog.Level
	maxBodyBytes    int64
	rateRefill      int
	rateBurst       int
}

// loadConfig reads configuration from environment variables. Defaults are
// tuned for an authorized intranet assessment scenario.
func loadConfig() config {
	c := config{
		port:            getenv("PORT", "8080"),
		vantage:         Vantage(getenv("VANTAGE", vantageIntranet)),
		txtPath:         getenv("TXT_PATH", "client_info.txt"),
		csvPath:         getenv("CSV_PATH", "index.csv"),
		scanScriptPath:  getenv("SCAN_SCRIPT", "scripts/fingerprint.sh"),
		scanCacheTTL:    getenvDuration("SCAN_CACHE_TTL", 30*time.Minute),
		scanConcurrency: getenvInt("SCAN_CONCURRENCY", 5),
		scanTimeout:     getenvDuration("SCAN_TIMEOUT", 3*time.Minute),
		corsOrigin:      getenv("CORS_ORIGIN", "*"),
		trustedProxy:    getenv("TRUSTED_PROXY", ""),
		logLevel:        slog.LevelInfo,
		maxBodyBytes:    8 << 10,
		rateRefill:      1,
		rateBurst:       3,
	}
	if c.vantage != vantageIntranet && c.vantage != vantageInternet {
		slog.Warn("invalid VANTAGE, defaulting to intranet", "value", c.vantage)
		c.vantage = vantageIntranet
	}

	// Determine the public NAT egress IP we should test as the perimeter.
	// From the internet vantage it equals the visitor's IP at request time,
	// so we don't need a reflector; we defer and compute per-request.
	// From the intranet vantage we need an external reflector (the container's
	// outbound IP == the school's public egress). Try reflectors in order.
	if c.vantage == vantageIntranet {
		ip, err := detectNATEgressIP()
		if err != nil {
			slog.Error("could not detect NAT egress IP; intranet scan of perimeter will be skipped",
				"err", err)
		} else {
			c.natEgressIP = ip
			slog.Info("detected NAT egress IP", "ip", ip)
		}
	}
	return c
}

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}
func getenvInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func getenvDuration(k string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// httplogSink routes slog output to os.Stderr so container platforms capture
// structured logs. Used to avoid the deprecated log package and to keep stdout
// free of per-request PII noise.
type httplogSink struct{}

func (httplogSink) Write(p []byte) (int, error) { return os.Stderr.Write(p) }

// splitTrim is a small testable helper used in proxy header parsing.
func splitTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
