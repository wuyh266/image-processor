package server

import (
	"encoding/json"
	"net/http"

	"image-processor/internal/config"
	"image-processor/internal/process"
	"image-processor/internal/stats"
)

func NewRouter(cfg config.Config, st *stats.Stats) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(process.ErrorResponse{Error: "method not allowed"})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	svc := process.NewService(
		cfg.Process.MaxConcurrent,
		cfg.Process.MaxFileSize,
		cfg.Pool.Enabled,
		st,
	)

	mux.HandleFunc("/resize", process.NewResizeHandler(svc, cfg.Process.Timeout, st))

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(process.ErrorResponse{Error: "method not allowed"})
			return
		}
		snap := st.Snapshot()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(snap)
	})

	return WithAccessLog(mux)
}
