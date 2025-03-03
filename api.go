package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/drewheasman/chirpy/internal/auth"
	"github.com/drewheasman/chirpy/internal/database"
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

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type User struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error getting chirps")
		return
	}

	var chirpsResponse []Chirp
	for _, c := range chirps {
		chirpsResponse = append(chirpsResponse, Chirp(c))
	}

	respondWithJson(w, http.StatusOK, chirpsResponse)
}

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing uuid from given id path param")
		return
	}

	chirp, err := cfg.dbQueries.GetChirp(req.Context(), id)
	fmt.Println("chirp", chirp)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "error getting chirps")
	}
	respondWithJson(w, http.StatusOK, Chirp(chirp))
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, req *http.Request) {
	type expectedRequest struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
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
		cleanedWords = append(cleanedWords, word)
	}

	userRecord, err := cfg.dbQueries.GetUser(req.Context(), decoded.UserId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user_id not found")
	}

	chirpRecord, err := cfg.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   strings.Join(cleanedWords, " "),
		UserID: userRecord.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "failed to create chirp")
	}

	respondWithJson(w, http.StatusCreated, Chirp(chirpRecord))
}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, req *http.Request) {
	type userCreateRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	var decoded userCreateRequest
	if err := decoder.Decode(&decoded); err != nil || decoded.Email == "" || decoded.Password == "" {
		respondWithError(w, http.StatusBadRequest, "error unmarshalling request body")
		return
	}

	hashedPassword, err := auth.HashPassword(decoded.Password)
	if err != nil {
		fmt.Print(err.Error())
		respondWithError(w, http.StatusInternalServerError, "failed to hash password")
	}

	userRecord, err := cfg.dbQueries.CreateUser(req.Context(), database.CreateUserParams{
		Email:          decoded.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating user")
		return
	}

	fmt.Println("user created")

	usersResponse := User{
		Id:        userRecord.ID,
		CreatedAt: userRecord.CreatedAt,
		UpdatedAt: userRecord.UpdatedAt,
		Email:     userRecord.Email,
	}

	fmt.Println("trying to respond with", http.StatusCreated)
	respondWithJson(w, http.StatusCreated, usersResponse)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, req *http.Request) {
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	var decoded loginRequest
	if err := decoder.Decode(&decoded); err != nil {
		respondWithError(w, http.StatusBadRequest, "error unmarshalling request body")
		return
	}

	userRecord, err := cfg.dbQueries.GetUserByEmail(req.Context(), decoded.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "incorrect email or password")
	}

	usersResponse := User{
		Id:        userRecord.ID,
		CreatedAt: userRecord.CreatedAt,
		UpdatedAt: userRecord.UpdatedAt,
		Email:     userRecord.Email,
	}

	if err := auth.CheckPasswordHash(decoded.Password, userRecord.HashedPassword); err != nil {
		respondWithError(w, http.StatusUnauthorized, "incorrect email or password")
		return
	}

	respondWithJson(w, http.StatusOK, usersResponse)
}
