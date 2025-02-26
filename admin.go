package main

import "net/http"

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) // This is the default anyway
	_, _ = w.Write([]byte("OK"))
}
