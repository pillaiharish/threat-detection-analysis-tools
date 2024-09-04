package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

func scanIP(ip string, wg *sync.WaitGroup) {
	defer wg.Done()

	pinger, err := ping.NewPinger(ip)
	if err != nil {
		fmt.Printf("Failed to ping IP %s: %v\n", ip, err)
		return
	}
	// send only one ping
	pinger.Count = 1
	// set timeout for the ping
	pinger.Timeout = time.Second * 3

	err = pinger.Run() // start the ping
	if err != nil {
		fmt.Printf("Failed to ping IP %s: %v\n", ip, err)
		return
	}

	stats := pinger.Statistics() // get ping statistics
	if stats.PacketsRecv > 0 {
		fmt.Printf("IP %s is up\n", ip)
	}
}

func main() {
	var wg sync.WaitGroup

	localIP := "192.168.0." // small range
	startIP := 1
	endIP := 254

	for i := startIP; i <= endIP; i++ {
		ip := fmt.Sprintf("%s%d", localIP, i)
		wg.Add(1)
		go scanIP(ip, &wg)
	}

	wg.Wait()
	fmt.Println("Scan complete")

	// the IPs displayed in console are only available
	fmt.Println("Note: If no IPs are show in output then no IPs found")
}
