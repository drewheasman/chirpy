package main

import (
	"net/http"
	"sync/atomic"
)

func main() {
	serveMux := http.NewServeMux()
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	config := &apiConfig{}
	fileHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	serveMux.Handle("/app/", config.middlewareMetricsIncrement(fileHandler))

	serveMux.HandleFunc("GET /admin/metrics", config.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", config.resetHandler)

	serveMux.HandleFunc("GET /api/healthz", healthzHandler)
	serveMux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

	server.ListenAndServe()
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
