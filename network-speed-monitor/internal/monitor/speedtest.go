package monitor

import (
	"log"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// GetSpeed measures upload and download speeds using speedtest-go
func GetSpeed() (float64, float64) {
	_, err := speedtest.FetchUserInfo()
	if err != nil {
		log.Printf("Failed to fetch user info: %v", err)
		return 0, 0
	}

	serverList, err := speedtest.FetchServers()
	if err != nil {
		log.Printf("Failed to fetch server list: %v", err)
		return 0, 0
	}

	// Find the best server
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		log.Printf("Failed to find server: %v", err)
		return 0, 0
	}

	server := targets[0]

	// Measure latency
	server.PingTest(func(latency time.Duration) {
		log.Printf("Latency: %v", latency)
	})

	// Measure download speed
	if err := server.DownloadTest(); err != nil {
		log.Printf("Download test failed: %v", err)
		return 0, 0
	}

	// Measure upload speed
	if err := server.UploadTest(); err != nil {
		log.Printf("Upload test failed: %v", err)
		return 0, 0
	}

	// Convert ByteRate (bytes/sec) to Mbps
	download := float64(server.DLSpeed) / 1_000_000 * 8
	upload := float64(server.ULSpeed) / 1_000_000 * 8

	return upload, download
}
