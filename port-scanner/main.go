package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "1.0.0"

const authorizedUseNotice = `port-scanner: Authorized use only. Scan systems you own or have explicit written permission to test.`

const maxCIDRHosts = 65536

type config struct {
	targets     string
	ports       string
	timeout     time.Duration
	concurrency int
	banner      bool
	jsonOut     bool
	csvOut      bool
}

type portResult struct {
	Port    int           `json:"port"`
	State   string        `json:"state"`
	Banner  string        `json:"banner,omitempty"`
	Latency time.Duration `json:"latency_ms"`
}

type hostResult struct {
	Host  string       `json:"host"`
	Ports []portResult `json:"ports"`
}

func main() {
	fmt.Fprintln(os.Stderr, authorizedUseNotice)

	var cfg config
	flag.StringVar(&cfg.targets, "target", "", "comma-separated IPs, hostnames, or CIDR ranges (e.g. \"192.168.1.5,10.0.0.0/30,scanme.nmap.org\")")
	flag.StringVar(&cfg.ports, "ports", "1-1024", "port spec: ranges + lists (e.g. \"1-1000,22,80,443,8080-9000\")")
	flag.DurationVar(&cfg.timeout, "timeout", 2*time.Second, "per-port TCP connect timeout")
	flag.IntVar(&cfg.concurrency, "concurrency", 100, "max concurrent port probes")
	flag.BoolVar(&cfg.banner, "banner", false, "grab banners on open ports")
	flag.BoolVar(&cfg.jsonOut, "json", false, "JSON output to stdout")
	flag.BoolVar(&cfg.csvOut, "csv", false, "CSV output to stdout")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "port-scanner v%s\n\n%s\n\nUsage:\n", version, authorizedUseNotice)
		fmt.Fprintf(os.Stderr, "  port-scanner -target HOSTS [-ports SPEC] [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("port-scanner version", version)
		return
	}

	if cfg.targets == "" {
		fmt.Fprintln(os.Stderr, "error: -target is required")
		flag.Usage()
		os.Exit(2)
	}

	if cfg.concurrency < 1 {
		fmt.Fprintln(os.Stderr, "error: -concurrency must be >= 1")
		os.Exit(2)
	}

	if cfg.jsonOut && cfg.csvOut {
		fmt.Fprintln(os.Stderr, "error: -json and -csv are mutually exclusive")
		os.Exit(2)
	}

	hosts, err := parseTargets(cfg.targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ports, err := parsePorts(cfg.ports)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	results := runScan(hosts, ports, cfg)

	switch {
	case cfg.jsonOut:
		outputJSON(results)
	case cfg.csvOut:
		outputCSV(results)
	default:
		outputText(results)
	}
}

func parseTargets(spec string) ([]string, error) {
	tokens := strings.Split(spec, ",")
	var hosts []string
	seen := make(map[string]bool)

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}

		var resolved []string
		if strings.Contains(tok, "/") {
			ip, ipnet, err := net.ParseCIDR(tok)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", tok, err)
			}
			if ip.To4() == nil {
				return nil, fmt.Errorf("CIDR %q: only IPv4 supported", tok)
			}
			ones, bits := ipnet.Mask.Size()
			hostCount := 1 << (uint(bits) - uint(ones))
			if hostCount > maxCIDRHosts {
				return nil, fmt.Errorf("CIDR %q: %d hosts exceeds max %d", tok, hostCount, maxCIDRHosts)
			}
			for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
				resolved = append(resolved, ip.String())
			}
		} else if net.ParseIP(tok) != nil {
			resolved = []string{tok}
		} else {
			addrs, err := net.LookupHost(tok)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %q: %w", tok, err)
			}
			for _, a := range addrs {
				if net.ParseIP(a).To4() != nil {
					resolved = append(resolved, a)
				}
			}
			if len(resolved) == 0 {
				return nil, fmt.Errorf("no IPv4 addresses for %q", tok)
			}
		}

		for _, h := range resolved {
			if !seen[h] {
				seen[h] = true
				hosts = append(hosts, h)
			}
		}
	}

	sort.Strings(hosts)
	return hosts, nil
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func parsePorts(spec string) ([]int, error) {
	tokens := strings.Split(spec, ",")
	var ports []int
	seen := make(map[int]bool)

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}

		if strings.Contains(tok, "-") {
			parts := strings.SplitN(tok, "-", 2)
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port range %q: %v", tok, err)
			}
			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port range %q: %v", tok, err)
			}
			if start > end {
				return nil, fmt.Errorf("port range %q: start > end", tok)
			}
			for p := start; p <= end; p++ {
				if !validPort(p) {
					return nil, fmt.Errorf("port %d out of range (1-65535)", p)
				}
				if !seen[p] {
					seen[p] = true
					ports = append(ports, p)
				}
			}
		} else {
			p, err := strconv.Atoi(tok)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q: %v", tok, err)
			}
			if !validPort(p) {
				return nil, fmt.Errorf("port %d out of range (1-65535)", p)
			}
			if !seen[p] {
				seen[p] = true
				ports = append(ports, p)
			}
		}
	}

	sort.Ints(ports)
	return ports, nil
}

func validPort(p int) bool {
	return p >= 1 && p <= 65535
}

func runScan(hosts []string, ports []int, cfg config) []hostResult {
	results := make([]hostResult, len(hosts))
	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.concurrency)

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			hr := hostResult{Host: h}
			var portWg sync.WaitGroup
			var mu sync.Mutex

			for _, port := range ports {
				portWg.Add(1)
				go func(p int) {
					defer portWg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					r := scanPort(h, p, cfg.timeout, cfg.banner)
					if r.State == "open" {
						mu.Lock()
						hr.Ports = append(hr.Ports, r)
						mu.Unlock()
					}
				}(port)
			}
			portWg.Wait()

			sort.Slice(hr.Ports, func(a, b int) bool {
				return hr.Ports[a].Port < hr.Ports[b].Port
			})
			results[idx] = hr
		}(i, host)
	}

	wg.Wait()
	return results
}

func scanPort(host string, port int, timeout time.Duration, grabBanner bool) portResult {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	latency := time.Since(start)

	if err != nil {
		return portResult{Port: port, State: "closed", Latency: latency}
	}
	defer conn.Close()

	r := portResult{Port: port, State: "open", Latency: latency}

	if grabBanner {
		if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err == nil {
			scanner := bufio.NewScanner(conn)
			scanner.Buffer(make([]byte, 256), 256)
			if scanner.Scan() {
				r.Banner = strings.TrimSpace(scanner.Text())
			}
		}
	}

	return r
}

func outputText(results []hostResult) {
	for _, hr := range results {
		if len(hr.Ports) == 0 {
			fmt.Printf("%s: no open ports\n", hr.Host)
			continue
		}
		fmt.Printf("%s:\n", hr.Host)
		for _, pr := range hr.Ports {
			if pr.Banner != "" {
				fmt.Printf("  %-6d open  %s  (banner: %s)\n", pr.Port, pr.Latency.Round(time.Millisecond), pr.Banner)
			} else {
				fmt.Printf("  %-6d open  %s\n", pr.Port, pr.Latency.Round(time.Millisecond))
			}
		}
	}
}

func outputJSON(results []hostResult) {
	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: json marshal: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func outputCSV(results []hostResult) {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	if err := w.Write([]string{"host", "port", "state", "latency_ms", "banner"}); err != nil {
		fmt.Fprintf(os.Stderr, "error: csv write: %v\n", err)
		os.Exit(1)
	}

	for _, hr := range results {
		if len(hr.Ports) == 0 {
			_ = w.Write([]string{hr.Host, "", "closed", "", ""})
			continue
		}
		for _, pr := range hr.Ports {
			_ = w.Write([]string{hr.Host, strconv.Itoa(pr.Port), pr.State, strconv.FormatInt(pr.Latency.Milliseconds(), 10), pr.Banner})
		}
	}
}