package monitor

import (
	"fmt"
	"network-speed-monitor/internal/models"
	"os"
)

const (
	FullLogFile     = "logs/full_logs.txt"
	FilteredLogFile = "logs/filtered_logs.txt"
)

func WriteFullLog(stat models.Stat) {
	file, _ := os.OpenFile(FullLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	fmt.Fprintf(file, "%s | Connectivity: %v | Upload: %.2f Mbps | Download: %.2f Mbps\n",
		stat.Timestamp, stat.Connectivity, stat.UploadSpeedMBps, stat.DownloadSpeedMBps)
}

func WriteFilteredLog(stat models.Stat) {
	file, _ := os.OpenFile(FilteredLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	fmt.Fprintf(file, "%s | Low Bandwidth! Upload: %.2f Mbps, Download: %.2f Mbps\n",
		stat.Timestamp, stat.UploadSpeedMBps, stat.DownloadSpeedMBps)
}
