package main

import (
	"log"
	"network-speed-monitor/internal/api"
	"network-speed-monitor/internal/models"
	"network-speed-monitor/internal/monitor"
	"time"
)

func main() {
	go func() {
		for {
			// Collect stats every 5 seconds
			stat := collectStats()
			monitor.UpdateCache(stat)
			time.Sleep(5 * time.Second)
		}
	}()

	router := api.SetupRouter()
	log.Println("Server running at http://localhost:8080")
	if err := router.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func collectStats() models.Stat {
	timestamp := time.Now().Format("2006-01-02 15:04:05 IST")
	connectivity := monitor.CheckConnectivity()
	upload, download := monitor.GetSpeed()
	stat := models.Stat{
		Timestamp:         timestamp,
		Connectivity:      connectivity,
		UploadSpeedMBps:   upload,
		DownloadSpeedMBps: download,
	}
	monitor.WriteFullLog(stat)
	if upload < 2 || download < 2 {
		monitor.WriteFilteredLog(stat)
	}
	return stat
}

// harish $ go run cmd/main.go
// http://localhost:8080/static/index.html
// http://localhost:8080/api/logs/filtered
