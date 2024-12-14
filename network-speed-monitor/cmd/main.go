package main

import (
	"fmt"
	"log"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

func main() {
	// Fetch user information
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		log.Printf("Failed to fetch user info: %v", err)
		return
	}
	fmt.Println("User Info:", user)

	// Fetch server list
	serverList, err := speedtest.FetchServers()
	if err != nil {
		log.Printf("Failed to fetch server list: %v", err)
		return
	}
	fmt.Printf("Available Servers: %d\n", len(*serverList.Available()))

	// Find target server (select closest by default)
	target, err := serverList.FindServer([]int{}) // Pass an empty slice to select the closest server
	if err != nil {
		log.Printf("Failed to find server: %v", err)
		return
	}
	server := target[0] // Use the first server from the result
	fmt.Printf("Testing against server: %s (%s)\n", server.Name, server.Country)

	// Run ping test with a callback function
	err = server.PingTest(func(latency time.Duration) {
		fmt.Printf("Ping: %.2f ms\n", latency.Seconds()*1000) // Convert latency from seconds to milliseconds
	})
	if err != nil {
		log.Printf("Ping test failed: %v", err)
		return
	}

	// Run download speed test
	if err := server.DownloadTest(); err != nil {
		log.Printf("Download test failed: %v", err)
		return
	}
	fmt.Printf("Download Speed: %.2f Mbps\n", server.DLSpeed)

	// Run upload speed test
	if err := server.UploadTest(); err != nil {
		log.Printf("Upload test failed: %v", err)
		return
	}
	fmt.Printf("Upload Speed: %.2f Mbps\n", server.ULSpeed)
}
