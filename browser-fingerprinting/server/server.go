package main

import (
	"log/slog"
	"net/http"
)

// Vantage is the deployment position of the server. It tags every record so
// records from an intranet-side run and an internet-side run can be compared.
type Vantage string

const (
	vantageIntranet = "intranet"
	vantageInternet = "internet"
)

func main() {
	cfg := loadConfig()
	logger := slog.New(slog.NewJSONHandler(httplogSink{}, &slog.HandlerOptions{Level: cfg.logLevel}))
	slog.SetDefault(logger)

	store := newStore(cfg.txtPath, cfg.csvPath)
	scan := newScanner(cfg.scanScriptPath, cfg.scanCacheTTL, cfg.scanConcurrency, cfg.scanTimeout)

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/store-info", storeInfoHandler(cfg, store, scan))

	var handler http.Handler = mux
	handler = securityHeaders(handler)
	handler = cors(handler, cfg.corsOrigin)
	handler = withLogger(handler)

	slog.Info("server starting", "vantage", cfg.vantage, "port", cfg.port,
		"natEgressIP", cfg.natEgressIP)
	srv := &http.Server{
		Addr:    ":" + cfg.port,
		Handler: handler,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server failed", "err", err)
	}
}
