package main

import "net/http"

func getHealthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) // This is the default anyway
	_, _ = w.Write([]byte("OK"))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)

	err := cfg.dbQueries.DeleteAllUsers(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to delete all users")
	}

	_, _ = w.Write([]byte("OK"))
}
