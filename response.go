package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	fmt.Println(message)

	type error struct {
		Error string `json:"error"`
	}

	errorJsonString, err := json.Marshal(error{Error: message})
	if err != nil {
		fmt.Println("error marshalling error json")
	}

	w.WriteHeader(statusCode)
	w.Write([]byte(errorJsonString))
}

func respondWithJson(w http.ResponseWriter, statusCode int, payload interface{}) {
	resp, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error marshalling valid response")
		return
	}

	w.WriteHeader(statusCode)
	w.Write([]byte(resp))
}
