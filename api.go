package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	Id           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()
	authorId := queryParams.Get("author_id")
	sortOrder := queryParams.Get("sort")
	if sortOrder == "" {
		sortOrder = "asc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		respondWithError(w, http.StatusBadRequest, "sort must be asc or desc")
		return
	}

	var chirps []database.Chirp
	var err error
	if authorId != "" {
		authorUUID, err := uuid.Parse(authorId)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "failed to parse author_id")
			return
		}
		chirps, err = cfg.dbQueries.GetChirpsByUser(req.Context(), authorUUID)
	} else {
		chirps, err = cfg.dbQueries.GetChirps(req.Context())
	}
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusBadRequest, "error getting chirps")
		return
	}

	var chirpsResponse []Chirp
	for _, c := range chirps {
		chirpsResponse = append(chirpsResponse, Chirp(c))
	}

	sort.Slice(chirpsResponse, func(i, j int) bool {
		if sortOrder == "asc" {
			return chirpsResponse[i].CreatedAt.Before(chirpsResponse[j].CreatedAt)
		}
		return chirpsResponse[i].CreatedAt.After(chirpsResponse[j].CreatedAt)
	})

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
		Body string `json:"body"`
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	id, err := auth.ValidateJWT(token, cfg.serverSecret)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
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

	userRecord, err := cfg.dbQueries.GetUser(req.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user_id not found")
		return
	}

	chirpRecord, err := cfg.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   strings.Join(cleanedWords, " "),
		UserID: userRecord.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "failed to create chirp")
		return
	}

	respondWithJson(w, http.StatusCreated, Chirp(chirpRecord))
}

type createUpdateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUsersHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var decoded createUpdateUserRequest
	if err := decoder.Decode(&decoded); err != nil || decoded.Email == "" || decoded.Password == "" {
		respondWithError(w, http.StatusBadRequest, "error unmarshalling request body")
		return
	}

	hashedPassword, err := auth.HashPassword(decoded.Password)
	if err != nil {
		fmt.Print(err.Error())
		respondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
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
		Id:          userRecord.ID,
		CreatedAt:   userRecord.CreatedAt,
		UpdatedAt:   userRecord.UpdatedAt,
		Email:       userRecord.Email,
		IsChirpyRed: userRecord.IsChirpyRed,
	}

	fmt.Println("trying to respond with", http.StatusCreated)
	respondWithJson(w, http.StatusCreated, usersResponse)
}

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	id, err := auth.ValidateJWT(token, cfg.serverSecret)

	decoder := json.NewDecoder(req.Body)
	var decoded createUpdateUserRequest
	if err := decoder.Decode(&decoded); err != nil || decoded.Email == "" || decoded.Password == "" {
		respondWithError(w, http.StatusBadRequest, "error unmarshalling request body")
		return
	}

	hashedPassword, err := auth.HashPassword(decoded.Password)
	if err != nil {
		fmt.Print(err.Error())
		respondWithError(w, http.StatusUnauthorized, "failed to hash password")
		return
	}

	userRecord, err := cfg.dbQueries.UpdateUser(req.Context(), database.UpdateUserParams{
		ID:             id,
		Email:          decoded.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "error updating user")
		return
	}

	fmt.Println("user updated")

	usersResponse := User{
		Id:          userRecord.ID,
		CreatedAt:   userRecord.CreatedAt,
		UpdatedAt:   userRecord.UpdatedAt,
		Email:       userRecord.Email,
		IsChirpyRed: userRecord.IsChirpyRed,
	}

	respondWithJson(w, http.StatusOK, usersResponse)
}

func (cfg *apiConfig) deleteChirpHandler(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing uuid from given id path param")
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.serverSecret)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	chirp, err := cfg.dbQueries.GetChirp(req.Context(), id)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}
	if chirp.UserID != userId {
		fmt.Println(err)
		respondWithError(w, http.StatusForbidden, "Can't delete chirp for a different user")
		return
	}

	err = cfg.dbQueries.DeleteChirp(req.Context(), id)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp")
		return
	}

	fmt.Println("chirp deleted")

	respondNoContent(w, http.StatusNoContent)
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
		return
	}

	usersResponse := User{
		Id:          userRecord.ID,
		CreatedAt:   userRecord.CreatedAt,
		UpdatedAt:   userRecord.UpdatedAt,
		Email:       userRecord.Email,
		IsChirpyRed: userRecord.IsChirpyRed,
	}

	if err := auth.CheckPasswordHash(decoded.Password, userRecord.HashedPassword); err != nil {
		respondWithError(w, http.StatusUnauthorized, "incorrect email or password")
		return
	}

	jwt, err := auth.MakeJWT(usersResponse.Id, cfg.serverSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}
	usersResponse.Token = jwt

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to make refresh token")
		return
	}
	err = cfg.dbQueries.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    userRecord.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save refresh token")
		return
	}
	usersResponse.RefreshToken = refreshToken

	fmt.Println("user logged in")

	respondWithJson(w, http.StatusOK, usersResponse)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	userId, err := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), token)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	jwt, err := auth.MakeJWT(userId, cfg.serverSecret, time.Hour)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	type tokenResponse struct {
		Token string `json:"token"`
	}

	respondWithJson(w, http.StatusOK, tokenResponse{Token: jwt})
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	_, err = cfg.dbQueries.GetUserFromRefreshToken(req.Context(), token)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "Not authorized")
		return
	}

	err = cfg.dbQueries.RevokeToken(req.Context(), token)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Error revoking token")
		return
	}

	fmt.Println("token revoked")

	respondNoContent(w, http.StatusNoContent)
}

func (cfg *apiConfig) polkaWebhooksHandler(w http.ResponseWriter, req *http.Request) {
	type polkaWebhookRequest struct {
		Event string `json:"event"`
		Data  struct {
			UserId string `json:"user_id"`
		} `json:"data"`
	}

	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil || apiKey != cfg.polkaKey {
		fmt.Println("invalid ApiKey")
		respondNoContent(w, http.StatusUnauthorized)
		return
	}

	var decoded polkaWebhookRequest
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&decoded); err != nil {
		fmt.Println("error decoding request")
		respondNoContent(w, http.StatusNoContent)
		return
	}

	if decoded.Event != "user.upgraded" {
		fmt.Println("unknown event")
		respondNoContent(w, http.StatusNoContent)
		return
	}

	userId, err := uuid.Parse(decoded.Data.UserId)
	if err != nil {
		fmt.Println("error parsing user_id")
		respondNoContent(w, http.StatusNoContent)
		return
	}

	err = cfg.dbQueries.SetChirpyRed(req.Context(), userId)
	if err != nil {
		fmt.Println("error setting chirpy red")
		respondNoContent(w, http.StatusNotFound)
		return
	}

	fmt.Println("set chirpy red")
	respondNoContent(w, http.StatusNoContent)
}
