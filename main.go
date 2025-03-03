package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/drewheasman/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("error starting server")
	}

	dbQueries := database.New(db)

	serveMux := http.NewServeMux()
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	config := &apiConfig{dbQueries: dbQueries}
	fileHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	serveMux.Handle("/app/", config.middlewareMetricsIncrement(fileHandler))

	serveMux.HandleFunc("GET /admin/metrics", config.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", config.resetHandler)

	serveMux.HandleFunc("GET /api/healthz", healthzHandler)
	serveMux.HandleFunc("GET /api/chirps", config.getChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps/{id}", config.getChirpHandler)
	serveMux.HandleFunc("POST /api/chirps", config.createChirpHandler)
	serveMux.HandleFunc("POST /api/users", config.usersHandler)
	serveMux.HandleFunc("POST /api/login", config.loginHandler)

	server.ListenAndServe()
}

type apiConfig struct {
	dbQueries      *database.Queries
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
