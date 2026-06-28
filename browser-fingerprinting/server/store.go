package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// scanStatus is a normalized enum for the nmap OS-probe outcome. The triage
// operator greps the CSV index for these to find "which laptops had their OS
// hidden by NAT/local firewall" vs "which OSes were successfully guessed".
type scanStatus string

const (
	scanPending  scanStatus = "pending"
	scanSuccess  scanStatus = "success"   // OS guessed
	scanOSHidden scanStatus = "os_hidden" // host up but OS not guessable
	scanHostDown scanStatus = "host_down" // 0 hosts up / unreachable
	scanError    scanStatus = "error"     // nmap itself errored
)

type scanRole string

const (
	scanRoleVisitor scanRole = "visitor"
	scanRoleNAT     scanRole = "nat"
)

// record is the canonical per-visit record. It is written to client_info.txt
// in human-readable form and to index.csv in machine-readable form.
type record struct {
	Timestamp         time.Time
	Vantage           string
	VisitorIP         string
	ProxyUsed         bool
	NatEgressIP       string
	FingerprintHash   string
	ShortHash         string
	Signals           json.RawMessage
	ScanStatusVisitor scanStatus
	OSGuessVisitor    string
	ScanStatusNAT     scanStatus
	OSGuessNAT        string
	ScanDurationMsV   int64
	ScanDurationMsN   int64
}

// store is the on-disk audit trail. Two files under one mutex:
//   - client_info.txt  : human-readable blocks (operator eyeballs this in the lab)
//   - index.csv        : machine-readable rows (operator greps this by hash/IP)
type store struct {
	mu      sync.Mutex
	txtPath string
	csvPath string
}

func newStore(txt, csv string) *store {
	return &store{txtPath: txt, csvPath: csv}
}

// append writes one visit record to both files atomically (single mutex) and
// creates the files with header rows on first use.
func (s *store) append(r record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writeTxt(r)
	s.writeCSV(r)
}

// stampScan fills in the scan result for a previously-recorded visit and
// appends a "Scan results for <IP>" sub-block to the human-readable file so
// the operator can match it back to the original visit by timestamp.
func (s *store) stampScan(visitStamp time.Time, role scanRole, res scanResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	header := fmt.Sprintf(
		"\n[scan %s] for visit@%s ip=%s\n", role, visitStamp.Format(time.RFC3339Nano), res.IP)
	block := fmt.Sprintf("  status=%s osGuess=%q durationMs=%d\n  nmap_output:\n%s  ----end scan----\n",
		res.Status, res.OSGuess, res.DurationMs, indent(res.RawOutput, "    "))
	appendString(s.txtPath, header+block)

	// Update CSV with a trailing scan-result line keyed by visit timestamp.
	row := []string{
		visitStamp.Format(time.RFC3339Nano),
		"", // vantage (already on prior row; blank here)
		res.IP,
		"", "", "", // natIP, fpHash, shortHash
		"",           // signals
		string(role), // scanRole
		string(res.Status),
		res.OSGuess,
		strconv.FormatInt(res.DurationMs, 10),
	}
	appendCSVRow(s.csvPath, row)
}

func (s *store) writeTxt(r record) {
	var b strings.Builder
	b.WriteString("\n========================================\n")
	b.WriteString(fmt.Sprintf("Timestamp:   %s\n", r.Timestamp.Format(time.RFC3339Nano)))
	b.WriteString(fmt.Sprintf("Vantage:      %s\n", r.Vantage))
	b.WriteString(fmt.Sprintf("VisitorIP:    %s\n", r.VisitorIP))
	b.WriteString(fmt.Sprintf("ProxyUsed:    %v\n", r.ProxyUsed))
	b.WriteString(fmt.Sprintf("NatEgressIP:  %s\n", r.NatEgressIP))
	b.WriteString(fmt.Sprintf("Fingerprint:  %s\n", r.FingerprintHash))
	b.WriteString(fmt.Sprintf("ShortHash:    %s\n", r.ShortHash))
	b.WriteString(fmt.Sprintf("Signals:      %s\n", truncate(string(r.Signals), 4000)))
	b.WriteString("ScanVisitor:  pending\n")
	b.WriteString("ScanNAT:       pending\n")
	b.WriteString("========================================\n")
	appendString(s.txtPath, b.String())
}

func (s *store) writeCSV(r record) {
	header := []string{"timestamp", "vantage", "visitorIP", "natEgressIP",
		"fingerprintHash", "shortHash", "signals",
		"scanRole", "scanStatus", "osGuess", "scanDurationMs"}
	ensureCSVHeader(s.csvPath, header)

	// One row for the visitor-scan placeholder + one for NAT placeholder
	row := []string{
		r.Timestamp.Format(time.RFC3339Nano),
		r.Vantage,
		r.VisitorIP,
		r.NatEgressIP,
		r.FingerprintHash,
		r.ShortHash,
		truncate(string(r.Signals), 4000),
		string(scanRoleVisitor), string(scanPending), "", "0",
	}
	appendCSVRow(s.csvPath, row)
	if r.NatEgressIP != "" && r.NatEgressIP != r.VisitorIP {
		row2 := []string{
			r.Timestamp.Format(time.RFC3339Nano),
			r.Vantage,
			r.VisitorIP,
			r.NatEgressIP,
			r.FingerprintHash,
			r.ShortHash,
			"", // signals not repeated on NAT scan row
			string(scanRoleNAT), string(scanPending), "", "0",
		}
		appendCSVRow(s.csvPath, row2)
	}
}

// --- low-level file helpers ---

func appendString(path, s string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(s)
}

func appendCSVRow(path string, row []string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write(row)
	w.Flush()
}

func ensureCSVHeader(path string, header []string) {
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return
	}
	appendCSVRow(path, header)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func indent(s, prefix string) string {
	out := ""
	for _, line := range splitLines(s) {
		out += prefix + line + "\n"
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else if r != '\r' {
			cur += string(r)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
