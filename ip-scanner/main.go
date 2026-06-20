package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPorts      = "22,80,443,554,5000,631,8080"
	defaultTimeoutMs  = 800
	defaultWorkers    = 64
	minSubnetPrefixV4 = 30
	arpPeersSentinel  = "<arp-peers>"
)

var ouiTable = map[string]string{
	// Apple - common OUI blocks across Wi-Fi/Ethernet (incl. Mac mini hosts)
	"a4:5e:60": "Apple",
	"6c:40:08": "Apple",
	"3c:22:fb": "Apple",
	"94:94:26": "Apple",
	"70:73:cb": "Apple",
	"68:5b:35": "Apple",
	"f4:06:69": "Apple",
	"ec:46:97": "Apple",
	"d4:ca:6e": "Apple",
	"a6:b2:ec": "Apple",
	"9e:b7:46": "Apple",
	"50:a6:d8": "Apple",
	"a0:99:9b": "Apple",
	"5c:f5:1e": "Apple",
	"3c:ab:8e": "Apple",
	"b8:c7:5d": "Apple",
	"7c:c1:81": "Apple",
	"3c:15:39": "Apple",
	"fc:25:3f": "Apple",
	"fc:e1:57": "Apple",
	"7c:c3:0a": "Apple",
	"84:fc:fe": "Apple",
	"ac:bc:32": "Apple",
	"90:8d:0e": "Apple",
	"9c:20:7b": "Apple",
	"d8:30:62": "Apple",
	// TP-Link
	"74:da:88": "TP-Link",
	"e0:cb:4e": "TP-Link",
	"30:b5:c2": "TP-Link",
	"ac:e0:10": "TP-Link",
	"1c:bf:ce": "TP-Link",
	"fc:db:8":  "TP-Link",
	"c0:c9:e3": "TP-Link",
	"f4:ec:38": "TP-Link",
	"f8:1a:67": "TP-Link",
	"60:32:b1": "TP-Link",
	// Asus
	"1c:87:2c": "Asus",
	"ac:9e:17": "Asus",
	"30:5a:0a": "Asus",
	"74:d0:2b": "Asus",
	"38:d5:47": "Asus",
	"2c:56:dc": "Asus",
	"1c:6f:65": "Asus",
	// Netgear
	"b8:5c:78": "Netgear",
	"00:09:5b": "Netgear",
	"00:13:46": "Netgear",
	"00:1b:2f": "Netgear",
	"00:1f:33": "Netgear",
	"00:26:f2": "Netgear",
	"20:3d:66": "Netgear",
	"44:94:fc": "Netgear",
	"fc:a1:83": "Netgear",
	"6c:b0:ce": "Netgear",
	// D-Link
	"00:0f:3d": "D-Link",
	"00:0d:88": "D-Link",
	"1c:7e:e5": "D-Link",
	"34:62:aa": "D-Link",
	"78:e7:51": "D-Link",
	// Linksys
	"00:1a:70": "Linksys",
	"00:1b:11": "Linksys",
	"00:21:29": "Linksys",
	"00:1c:10": "Linksys",
	"00:14:6c": "Linksys",
	// Amazon (Echo, Kindle, FireTV)
	"fc:75:16": "Amazon",
	"f0:4f:7c": "Amazon",
	"78:e1:37": "Amazon",
	"44:65:0d": "Amazon",
	"7c:1c:4e": "Amazon",
	"f0:54:1b": "Amazon",
	"34:d2:70": "Amazon",
	"3c:9d:7":  "Amazon",
	// Google (Home, Nest)
	"3c:37:86": "Google",
	"f4:f5:e8": "Google",
	"fc:fe:f6": "Google",
	"64:16:66": "Google",
	// Sonos
	"78:28:ca": "Sonos",
	"00:0e:58": "Sonos",
	// Roku
	"dc:3a:5e": "Roku",
	"b0:ee:45": "Roku",
	"00:0d:4f": "Roku",
	"b8:3e:59": "Roku",
	// Bose
	"00:0c:8a": "Bose",
	"50:04:bd": "Bose",
	// Raspberry Pi
	"b8:27:eb": "Raspberry-Pi",
	"d0:be:1e": "Raspberry-Pi",
	"e4:5f:1":  "Raspberry-Pi",
	// Intel (NICs, Wi-Fi on many laptops)
	"5c:5f:67": "Intel",
	"00:1b:21": "Intel",
	"00:13:1f": "Intel",
	"dc:37:14": "Intel",
	"6c:8d:a1": "Intel",
	// Realtek
	"00:e0:4c": "Realtek",
	"52:54:00": "Realtek",
	// Broadcom (common in routers / APs)
	"d4:ca:6":  "Broadcom",
	"00:1a:1f": "Broadcom",
	"00:24:1d": "Broadcom",
	// Additional OUIs - common in Apple/Intel laptops, IoT, USB-Ethernet dongles
	"1c:f6:4c": "Apple",     // Apple (various hosts)
	"28:87:ba": "Espressif", // ESP8266/ESP32 IoT devices
	"fe:f7:79": "Apple",     // locally-administered bit set - often Apple private Wi-Fi
	"3c:18:a0": "ASIX",      // common USB-Ethernet adapter OUI
}


type ifaceInfo struct {
	name   string
	hwname string
	cidr   string
	myIP   string
	extras []string // ARP-discovered peers on this iface not part of cidr
}

type hostResult struct {
	ip        string
	openPorts []int
	hostname  string
	mac       string
	vendor    string
}

func main() {
	portsFlag := flag.String("ports", defaultPorts, "comma-separated TCP ports to probe (e.g. 22,80,443)")
	subnetFlag := flag.String("subnets", "", "comma-separated CIDRs to scan (auto-detect if empty, e.g. 192.168.1.0/24,10.0.0.0/30)")
	timeoutFlag := flag.Duration("timeout", time.Duration(defaultTimeoutMs)*time.Millisecond, "per-port TCP dial timeout")
	workersFlag := flag.Int("workers", defaultWorkers, "concurrent scan workers")
	flag.Parse()

	ports := parsePorts(*portsFlag)
	if len(ports) == 0 {
		fmt.Fprintln(os.Stderr, "no valid ports specified")
		os.Exit(2)
	}

	ifaces, err := detectInterfaces(*subnetFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(ifaces) == 0 {
		fmt.Fprintln(os.Stderr, "no active IPv4 interfaces found to scan")
		os.Exit(1)
	}

	totalHosts := 0
	totalCandidates := 0
	seen := make(map[string]bool)
	fmt.Printf("Discovering hosts across %d interface(s) probing ports %v\n\n", len(ifaces), ports)
	for _, ifc := range ifaces {
		var hosts []string
		if ifc.cidr == arpPeersSentinel {
			hosts = append(hosts, ifc.extras...)
		} else {
			h, err := enumerateCIDR(ifc.cidr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skip %s: %v\n", ifc.cidr, err)
				continue
			}
			hosts = h
		}
		// merge in ARP-discovered peers that aren't already enumerated from the CIDR
		hostSet := map[string]bool{}
		for _, h := range hosts {
			hostSet[h] = true
		}
		for _, h := range ifc.extras {
			if ifc.cidr != arpPeersSentinel && !hostSet[h] {
				hostSet[h] = true
				hosts = append(hosts, h)
			}
		}
		if len(hosts) == 0 {
			fmt.Printf("== %-6s %-12s %s (my ip %s) - no targets ==\n", ifc.name, "("+ifc.hwname+")", ifc.cidr, ifc.myIP)
			continue
		}
		label := ifc.cidr
		if label == arpPeersSentinel {
			label = "ARP-discovered peers"
		}
		fmt.Printf("== %-6s %-12s %s (my ip %s) - %d targets ==\n", ifc.name, "("+ifc.hwname+")", label, ifc.myIP, len(hosts))
		results := scanHosts(hosts, ports, *timeoutFlag, *workersFlag)
		probedDead := map[string]bool{}
		for _, h := range hosts {
			probedDead[h] = true
		}
		sort.Slice(results, func(i, j int) bool { return results[i].ip < results[j].ip })
		fmt.Println("-- reachable (a probed port answered) --")
		for _, r := range results {
			delete(probedDead, r.ip)
			if seen[r.ip] {
				continue
			}
			seen[r.ip] = true
			r.mac = lookupMAC(r.ip)
			r.hostname = lookupHostname(r.ip)
			r.vendor = vendorFor(r.mac)
			printResult(r)
			totalHosts++
		}
		// ARP-known peers on this iface that didn't answer any port. These are
		// live at L2 but not running anything we probed - still useful for
		// "which IP is on the wire" (e.g. a Mac mini with SSH off).
		arpPeersOnIface := arpPeerRecords(ifc.name)
		var candidates []hostResult
		for _, rec := range arpPeersOnIface {
			if !probedDead[rec.ip] {
				continue
			}
			if !isValidMAC(rec.mac) {
				continue
			}
			if seen[rec.ip] {
				continue
			}
			candidates = append(candidates, hostResult{
				ip:        rec.ip,
				openPorts: nil,
				hostname:  lookupHostname(rec.ip),
				mac:       rec.mac,
				vendor:    vendorFor(rec.mac),
			})
		}
		if len(candidates) > 0 {
			sort.Slice(candidates, func(i, j int) bool { return candidates[i].ip < candidates[j].ip })
			fmt.Println("-- ARP-known but no open port (live on L2, SSH/other services not listening) --")
			for _, r := range candidates {
				seen[r.ip] = true
				printResult(r)
				totalCandidates++
			}
		}
		fmt.Println()
	}
	fmt.Printf("Scan complete: %d port-reachable host(s), %d ARP-known-only candidate(s), across %d interface(s)\n",
		totalHosts, totalCandidates, len(ifaces))
	if totalHosts == 0 && totalCandidates == 0 {
		fmt.Println("Note: no hosts responded and no ARP-known peers - devices may be off or network unreachable")
	}
}

func parsePorts(s string) []int {
	out := []int{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			continue
		}
		out = append(out, n)
	}
	return dedupInts(out)
}

func dedupInts(in []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func detectInterfaces(override string) ([]ifaceInfo, error) {
	if override != "" {
		return parseOverride(override)
	}
	all, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	var out []ifaceInfo
	for _, ifi := range all {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualIface(ifi.Name) {
			continue
		}
		// Collect all IPv4 addresses on this iface (incl. link-local 169.254).
		var v4addrs []net.IP
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			if ip4[0] == 127 {
				continue
			}
			v4addrs = append(v4addrs, ip4)
		}
		if len(v4addrs) == 0 {
			continue
		}
		// ARP-discovered peers (with a non-incomplete MAC) on this iface.
		// Critical: catches link-local 169.254.x peers (e.g. a host using
		// auto-IP on a USB-Ethernet link) that wouldn't be enumerated from
		// any configured subnet.
		peers := arpPeers(ifi.Name)
		peersBySubnet := map[int][]string{} // bucket by which local address's subnet they fall into
		for _, ip := range peers {
			if ip.IsLoopback() {
				continue
			}
			matched := -1
			for i, lip := range v4addrs {
				if sameSubnet(ip, lip) {
					matched = i
					break
				}
			}
			peersBySubnet[matched] = append(peersBySubnet[matched], ip.String())
		}
	// For each local addr, emit one ifaceInfo. Skip tiny self-only /30s that
	// only contain our own IP (no peers). Skip the 169.254 self-addr (it
	// doesn't define a scan range; we'll pick up 169.254 peers via ARP).
	// Also skip a self-only /24 with no ARP peers (e.g. macOS Internet
	// Sharing emitting 192.168.2.1 with nothing on it) - pure noise.
	anyPeer := len(peers) > 0
	for i, lip := range v4addrs {
		if lip[0] == 169 {
			// link-local self-addr: skip as scan range, but harvest its peers
			// into the interface's extras via the peersBySubnet[-1] bucket.
			continue
		}
		ones := 24
		// honor the real mask if present
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && ipn.IP.To4().Equal(lip) {
				if o, _ := ipn.Mask.Size(); o >= 16 && o <= 30 {
					ones = o
				}
			}
		}
		if !anyPeer && lip[3] == 1 && ones >= 24 {
			// self-only .1 /24-or-smaller with nobody else ARP-known on this
			// iface - pure noise (e.g. macOS Internet Sharing 192.168.2.1).
			continue
		}
		cidr := fmt.Sprintf("%s/%d", lip.String(), ones)
		// gather peers that belong to this subnet
		var extras []string
		if p, ok := peersBySubnet[i]; ok {
			extras = append(extras, p...)
		}
		// also include 169.254 peers as extras on this iface if we have no
		// other way of reaching them (matched == -1)
		if p, ok := peersBySubnet[-1]; ok && len(extras) == 0 {
			extras = append(extras, p...)
		}
		out = append(out, ifaceInfo{
				name:   ifi.Name,
				hwname: portLabel(ifi.Name),
				cidr:   cidr,
				myIP:   lip.String(),
				extras: extras,
			})
		}
		// If this iface has any 169.254 peers we haven't already attached to a
		// subnet (e.g. an iface that holds only a 10.55/30 self-IP but the Mac
		// mini sits at 169.254.x on the same wire), emit a synthetic "ARP peers"
		// section so they still get scanned.
		if p, ok := peersBySubnet[-1]; ok && len(p) > 0 {
			dedupExtras := map[string]bool{}
			attached := false
			for _, e := range p {
				dedupExtras[e] = true
			}
			for _, e := range out {
				if e.name == ifi.Name {
					for _, x := range e.extras {
						dedupExtras[x] = false
					}
				}
			}
			var leftover []string
			for ip, fresh := range dedupExtras {
				if fresh {
					leftover = append(leftover, ip)
				}
			}
			if len(leftover) > 0 {
				attached = true
				out = append(out, ifaceInfo{
					name:   ifi.Name,
					hwname: portLabel(ifi.Name),
					cidr:   arpPeersSentinel,
					myIP:   "-",
					extras: leftover,
				})
			}
			_ = attached
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].name != out[j].name {
			return out[i].name < out[j].name
		}
		return out[i].cidr < out[j].cidr
	})
	return out, nil
}

func sameSubnet(a, b net.IP) bool {
	// treat both as /24 for the purpose of "same L2 segment" matching, which is
	// fine for home networks. Real masks would be better but /24 covers the
	// practical case.
	return a.To4() != nil && b.To4() != nil &&
		a.To4()[0] == b.To4()[0] && a.To4()[1] == b.To4()[1] &&
		a.To4()[2] == b.To4()[2]
}

// arpPeers runs `arp -an` and returns IPs that have a real MAC and are
// associated with the given interface (by parsing the "on <iface>" field).
func arpPeers(ifaceName string) []net.IP {
	recs := arpPeerRecords(ifaceName)
	ips := make([]net.IP, 0, len(recs))
	for _, r := range recs {
		if ip := net.ParseIP(r.ip); ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}

type arpRecord struct {
	ip  string
	mac string
}

// arpPeerRecords parses `arp -an` and returns one entry per row whose MAC is a
// real (non-incomplete, non-broadcast) address on the given interface.
func arpPeerRecords(ifaceName string) []arpRecord {
	out, err := exec.Command("arp", "-an").Output()
	if err != nil {
		return nil
	}
	var recs []arpRecord
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if !strings.Contains(ln, " on "+ifaceName+" ") && !strings.HasSuffix(ln, " on "+ifaceName) {
			continue
		}
		ipStr := extractArpIP(ln)
		if ipStr == "" {
			continue
		}
		mac := extractArpMAC(ln)
		if !isValidMAC(mac) {
			continue
		}
		recs = append(recs, arpRecord{ip: ipStr, mac: mac})
	}
	return recs
}

func extractArpIP(line string) string {
	// prefer the (...) form
	if o := strings.IndexByte(line, '('); o >= 0 {
		if c := strings.IndexByte(line[o+1:], ')'); c >= 0 {
			return strings.TrimSpace(line[o+1 : o+1+c])
		}
	}
	// fallback: first whitespace-delimited token that looks like an IP
	for _, tok := range strings.Fields(line) {
		if net.ParseIP(tok) != nil {
			return tok
		}
	}
	return ""
}

func extractArpMAC(line string) string {
	if atIdx := strings.Index(line, " at "); atIdx >= 0 {
		rest := strings.TrimSpace(line[atIdx+4:])
		fields := strings.Fields(rest)
		if len(fields) >= 1 {
			return normaliseMAC(fields[0])
		}
	}
	return ""
}

func isVirtualIface(name string) bool {
	switch name {
	case "lo0", "gif0", "stf0", "bridge0", "awdl0", "llw0", "utun0", "utun1", "utun2", "utun3", "utun4", "utun5", "utun6", "utun7", "utun8", "utun9", "anpi0", "anpi1", "anpi2":
		return true
	}
	if strings.HasPrefix(name, "utun") || strings.HasPrefix(name, "awdl") || strings.HasPrefix(name, "llw") || strings.HasPrefix(name, "gif") || strings.HasPrefix(name, "stf") {
		return true
	}
	return false
}

func portLabel(dev string) string {
	out, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return dev
	}
	lines := strings.Split(string(out), "\n")
	current := ""
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "Hardware Port:") {
			current = strings.TrimPrefix(ln, "Hardware Port:")
			current = strings.TrimSpace(current)
		} else if strings.HasPrefix(ln, "Device:") {
			devName := strings.TrimSpace(strings.TrimPrefix(ln, "Device:"))
			if devName == dev {
				rest := strings.SplitN(current, " (", 2)
				return rest[0]
			}
		}
	}
	return dev
}

func parseOverride(s string) ([]ifaceInfo, error) {
	out := []ifaceInfo{}
	for i, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, ipnet, err := net.ParseCIDR(part)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", part, err)
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("only IPv4 subnets supported: %s", part)
		}
		ones, _ := ipnet.Mask.Size()
		_ = i
		out = append(out, ifaceInfo{name: "manual", hwname: "manual", cidr: fmt.Sprintf("%s/%d", ip4.String(), ones), myIP: "-"})
	}
	return out, nil
}

func enumerateCIDR(cidr string) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("not ipv4")
	}
	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil, fmt.Errorf("not ipv4 mask")
	}
	hostBits := bits - ones
	total := 1 << uint(hostBits)
	hosts := make([]string, 0, total)
	switch {
	case hostBits <= 1:
		return nil, fmt.Errorf("subnet too small: %s", cidr)
	case hostBits == 2:
		// /30: 4 addresses, 2 usable
		hosts = append(hosts, incIP(ip4, 1).String(), incIP(ip4, 2).String())
	default:
		netStart := ip4
		broadcast := incIP(ip4, total-1)
		for i := 1; i < total-1; i++ {
			c := incIP(netStart, i)
			if c.Equal(broadcast) {
				continue
			}
			hosts = append(hosts, c.String())
		}
	}
	return hosts, nil
}

func incIP(ip net.IP, n int) net.IP {
	out := make(net.IP, 4)
	copy(out, ip.To4())
	for i := 3; i >= 0 && n > 0; i-- {
		v := int(out[i]) + n
		out[i] = byte(v & 0xFF)
		n = v >> 8
	}
	return out
}

func scanHosts(hosts []string, ports []int, timeout time.Duration, workers int) []hostResult {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string, len(hosts))
	results := make(chan hostResult, len(hosts))
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				open := probePorts(ip, ports, timeout)
				if len(open) > 0 {
					results <- hostResult{ip: ip, openPorts: open}
				}
			}
		}()
	}
	for _, h := range hosts {
		jobs <- h
	}
	close(jobs)
	wg.Wait()
	close(results)
	out := make([]hostResult, 0, len(results))
	for r := range results {
		out = append(out, r)
	}
	return out
}

func probePorts(ip string, ports []int, timeout time.Duration) []int {
	open := []int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			addr := net.JoinHostPort(ip, strconv.Itoa(port))
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err == nil {
				conn.Close()
				mu.Lock()
				open = append(open, port)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	sort.Ints(open)
	return open
}

func lookupHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return "-"
	}
	sort.Strings(names)
	return strings.TrimSuffix(names[0], ".")
}

func lookupMAC(ip string) string {
	mac := arpNext(ip)
	if mac != "" {
		return mac
	}
	pokeARP(ip)
	return arpNext(ip)
}

func arpNext(ip string) string {
	out, err := exec.Command("arp", "-n", ip).Output()
	if err != nil {
		return ""
	}
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// macOS: "ip (1.2.3.4) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]"
		if atIdx := strings.Index(ln, " at "); atIdx >= 0 {
			rest := strings.TrimSpace(ln[atIdx+4:])
			fields := strings.Fields(rest)
			if len(fields) >= 1 {
				mac := normaliseMAC(fields[0])
				if mac != "" && isValidMAC(mac) {
					return mac
				}
			}
		}
	}
	return ""
}

func pokeARP(ip string) {
	c, err := net.DialTimeout("udp", net.JoinHostPort(ip, "9"), 80*time.Millisecond)
	if err == nil {
		c.Write([]byte{0})
		c.Close()
	}
}

func isValidMAC(s string) bool {
	norm := normaliseMAC(s)
	if norm == "" {
		return false
	}
	return norm != "ff:ff:ff:ff:ff:ff" && norm != "00:00:00:00:00:00" &&
		!strings.HasPrefix(norm, "01:00:5e") && !strings.HasPrefix(norm, "33:33")
}

// normaliseMAC returns a canonical lowercased, zero-padded aa:bb:cc:dd:ee:ff,
// or "" if the input isn't a real MAC. Accepts single-digit octets as the
// macOS `arp` tool sometimes prints (e.g. "de:ad:be:ef:f:d").
func normaliseMAC(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return ""
	}
	for i, p := range parts {
		_, err := strconv.ParseUint(p, 16, 8)
		if err != nil {
			return ""
		}
		parts[i] = fmt.Sprintf("%02s", p)
	}
	return strings.Join(parts, ":")
}

func vendorFor(mac string) string {
	mac = normaliseMAC(mac)
	if mac == "" || !isValidMAC(mac) {
		return "unknown"
	}
	pfx := strings.ToLower(strings.Join(strings.Split(mac, ":")[:3], ":"))
	if v, ok := ouiTable[pfx]; ok {
		return v
	}
	return "other"
}

func printResult(r hostResult) {
	portsStr := intsToStr(r.openPorts)
	if len(r.openPorts) == 0 {
		portsStr = "none"
	}
	fmt.Printf("IP %-16s  PORTS %-24s  HOST %-25s  MAC %-17s  VENDOR %s\n",
		r.ip, "["+portsStr+"]", r.hostname, r.mac, r.vendor)
}

func intsToStr(in []int) string {
	parts := make([]string, len(in))
	for i, v := range in {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}