package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const chirpMaxLength int = 140

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	htmlString := `
<html>
    <body>
        <h1>Welcome, Chirpy Admin</h1>
        <p>Chirpy has been visited %d times!</p>
    </body>
</html>`
	_, _ = w.Write([]byte(fmt.Sprintf(htmlString, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, _ *http.Request) {
	cfg.fileserverHits.Store(0)
	_, _ = w.Write([]byte("OK"))
}

type error struct {
	Error string `json:"error"`
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	fmt.Println(message)
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
		respondWithError(w, 400, "error marshalling valid response")
		return
	}

	w.Write([]byte(resp))
}

func validateChirpHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	type expectedRequest struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	var decoded expectedRequest
	if err := decoder.Decode(&decoded); err != nil || decoded.Body == "" {
		respondWithError(w, 400, "error unmarshalling request body")
		return
	}

	if len(decoded.Body) > chirpMaxLength {
		fmt.Println("request body too long (max "+strconv.Itoa(chirpMaxLength)+" characters):", decoded.Body)
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	decodedWords := strings.Fields(decoded.Body)
	cleanedWords := []string{}
	for _, word := range decodedWords {
		lowerWord := strings.ToLower(word)
		for _, p := range profaneWords {
			if lowerWord == p {
				word = "****"
			}
		}
		fmt.Println(word)
		cleanedWords = append(cleanedWords, word)
	}

	type cleanedResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}
	respondWithJson(w, 200, cleanedResponse{CleanedBody: strings.Join(cleanedWords, " ")})
}
