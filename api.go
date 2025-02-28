package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

func validateChirpHandler(w http.ResponseWriter, req *http.Request) {
	type expectedRequest struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	var decoded expectedRequest
	if err := decoder.Decode(&decoded); err != nil || decoded.Body == "" {
		respondWithError(w, http.StatusBadRequest, "error unmarshalling request body")
		return
	}

	if len(decoded.Body) > chirpMaxLength {
		fmt.Println("request body too long (max "+strconv.Itoa(chirpMaxLength)+" characters):", decoded.Body)
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
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
	respondWithJson(w, http.StatusOK, cleanedResponse{CleanedBody: strings.Join(cleanedWords, " ")})
}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, req *http.Request) {
	type expectedRequest struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	var decoded expectedRequest
	if err := decoder.Decode(&decoded); err != nil || decoded.Email == "" {
		respondWithError(w, http.StatusBadRequest, "error unmarshalling request body")
		return
	}

	userRecord, err := cfg.dbQueries.CreateUser(req.Context(), decoded.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating user")
		return
	}

	fmt.Println("user created")

	type usersHandlerResponse struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	usersResponse := usersHandlerResponse{
		Id:        userRecord.ID,
		CreatedAt: userRecord.CreatedAt,
		UpdatedAt: userRecord.UpdatedAt,
		Email:     userRecord.Email,
	}

	fmt.Println("trying to respond with", http.StatusCreated)
	respondWithJson(w, http.StatusCreated, usersResponse)
}
