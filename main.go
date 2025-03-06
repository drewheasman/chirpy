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
	serverSecret := os.Getenv("SERVER_SECRET")

	dbQueries := database.New(db)

	serveMux := http.NewServeMux()
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	config := &apiConfig{
		dbQueries:    dbQueries,
		serverSecret: serverSecret,
	}
	fileHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	serveMux.Handle("/app/", config.middlewareMetricsIncrement(fileHandler))

	serveMux.HandleFunc("GET /admin/metrics", config.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", config.resetHandler)

	serveMux.HandleFunc("GET /api/healthz", getHealthzHandler)
	serveMux.HandleFunc("GET /api/chirps", config.getChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps/{id}", config.getChirpHandler)
	serveMux.HandleFunc("DELETE /api/chirps/{id}", config.deleteChirpHandler)
	serveMux.HandleFunc("POST /api/chirps", config.createChirpHandler)
	serveMux.HandleFunc("POST /api/users", config.createUsersHandler)
	serveMux.HandleFunc("PUT /api/users", config.updateUserHandler)
	serveMux.HandleFunc("POST /api/login", config.loginHandler)
	serveMux.HandleFunc("POST /api/refresh", config.refreshHandler)
	serveMux.HandleFunc("POST /api/revoke", config.revokeHandler)

	server.ListenAndServe()
}

type apiConfig struct {
	dbQueries      *database.Queries
	fileserverHits atomic.Int32
	serverSecret   string
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
