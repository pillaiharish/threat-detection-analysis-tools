package api

import (
	"net/http"
)

func SetupRouter() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", StatsHandler)
	mux.HandleFunc("/api/logs/full", FullLogsHandler)
	mux.HandleFunc("/api/logs/filtered", FilteredLogsHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}
