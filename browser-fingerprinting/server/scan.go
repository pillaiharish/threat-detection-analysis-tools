package main

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"log/slog"
)

// scanner owns a bounded worker pool that runs fingerprint.sh against an IP.
// Each unique (IP) is scanned at most once per scanCacheTTL to avoid hammering
// the school network during a lab.
type scanner struct {
	scriptPath string
	cacheTTL   time.Duration
	timeout    time.Duration
	jobs       chan scanJob
	wg         sync.WaitGroup

	mu    sync.Mutex
	cache map[string]time.Time
}

type scanJob struct {
	ip        string
	role      scanRole
	stamp     time.Time
	visitorIP string
	vantage   string
	shortHash string
	cb        func(scanResult)
}

type scanResult struct {
	IP         string
	Status     scanStatus
	OSGuess    string
	RawOutput  string
	DurationMs int64
}

func newScanner(scriptPath string, cacheTTL time.Duration, concurrency int, timeout time.Duration) *scanner {
	s := &scanner{
		scriptPath: scriptPath,
		cacheTTL:   cacheTTL,
		timeout:    timeout,
		jobs:       make(chan scanJob, 64),
		cache:      make(map[string]time.Time),
	}
	for i := 0; i < concurrency; i++ {
		s.wg.Add(1)
		go s.runWorker()
	}
	return s
}

func (s *scanner) enqueue(j scanJob, cb func(scanResult)) {
	if j.cb == nil {
		j.cb = cb
	} else if cb != nil {
		// Compose callbacks so stampScan runs alongside any caller-supplied one.
		prev := j.cb
		j.cb = func(r scanResult) { prev(r); cb(r) }
	}
	// Cache check: skip if we scanned this IP recently.
	s.mu.Lock()
	if last, ok := s.cache[j.ip]; ok && time.Since(last) < s.cacheTTL {
		s.mu.Unlock()
		slog.Info("scan skipped (cached)", "ip", j.ip)
		return
	}
	s.cache[j.ip] = time.Now()
	s.mu.Unlock()
	select {
	case s.jobs <- j:
	default:
		slog.Warn("scan queue full, dropping", "ip", j.ip)
	}
}

func (s *scanner) runWorker() {
	defer s.wg.Done()
	for j := range s.jobs {
		res := s.runScript(j.ip)
		if j.cb != nil {
			j.cb(res)
		}
	}
}

// runScript executes fingerprint.sh with the IP as a single argv element. The
// script is responsible for shell-escaping; we only pass one arg so there's
// no shell interpretation. The IP is validated by sanitizeIPLiteral upstream.
func (s *scanner) runScript(ip string) scanResult {
	if ip == "" {
		return scanResult{Status: scanError, RawOutput: "empty ip"}
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.scriptPath, ip)
	out, err := cmd.CombinedOutput()
	dur := time.Since(start).Milliseconds()
	raw := string(out)
	if err != nil {
		slog.Info("nmap returned error", "ip", ip, "err", err.Error(), "rc", cmd.ProcessState.ExitCode())
	}
	return scanResult{
		IP:         ip,
		Status:     parseScanStatus(raw, err),
		OSGuess:    parseOSGuess(raw),
		RawOutput:  raw,
		DurationMs: dur,
	}
}

// parseScanStatus classifies nmap -O output into one of the enumerated statuses.
// This is what powers "which laptop's OS was hidden by NAT/local firewall".
var reHostUp = regexp.MustCompile(`(?m)^\s*Host is up`)
var reNoOS = regexp.MustCompile(`(?m)^Too many fingerprints`)
var reOSDetails = regexp.MustCompile(`(?m)^OS details: (.+)$`)
var re0Hosts = regexp.MustCompile(`(?m)0 hosts up`)

func parseScanStatus(raw string, runErr error) scanStatus {
	// Order matters: explicit "requires root" surfaces a scanner misconfig.
	if strings.Contains(raw, "requires root") {
		return scanError
	}
	// 0 hosts up → host unreachable / opaque boundary. Valid negative result.
	if re0Hosts.MatchString(raw) {
		return scanHostDown
	}
	// Host is up but nmap couldn't guess an OS → NAT/firewall defeated us.
	if reNoOS.MatchString(raw) || (reHostUp.MatchString(raw) && !reOSDetails.MatchString(raw)) {
		return scanOSHidden
	}
	// OS line present → success.
	if reOSDetails.MatchString(raw) {
		return scanSuccess
	}
	// nmap exited nonzero and nothing else matched.
	if runErr != nil {
		return scanError
	}
	return scanError
}

func parseOSGuess(raw string) string {
	m := reOSDetails.FindStringSubmatch(raw)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
