package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

func scanIP(ip string, wg *sync.WaitGroup) {
	defer wg.Done()
	// dialing icmp for ipv4
	conn, err := net.DialTimeout("ip4:icmp", ip, time.Second*1)
	if err != nil {
		return
	}
	defer conn.Close()
	fmt.Printf("IP %s is up\n", ip)
}

func main() {
	var wg sync.WaitGroup
	localIP := "192.168.1." // small range
	startIP := 1
	endIP := 254
	for i := startIP; i <= endIP; i++ {
		ip := fmt.Sprintf("%s%d", localIP, i)
		wg.Add(1)
		go scanIP(ip, &wg)
	}
	wg.Wait()
	fmt.Println("Scan complete")
}
