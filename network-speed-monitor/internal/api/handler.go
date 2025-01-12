package api

import (
	"encoding/json"
	"net/http"
	"network-speed-monitor/internal/monitor"
)

func StatsHandler(w http.ResponseWriter, r *http.Request) {
	stat := monitor.GetCache()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stat)
}

// FullLogsHandler serves the full logs file
func FullLogsHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, monitor.FullLogFile)
}

// FilteredLogsHandler serves the filtered logs file
func FilteredLogsHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, monitor.FilteredLogFile)
}
