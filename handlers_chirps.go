package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/azizu06/Chirpy/internal/auth"
	"github.com/azizu06/Chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.dbQueries.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating chirp")
		return
	}
	chirps := []Chirp{}
	for _, chirp := range dbChirps {
		chirps = append(chirps, Chirp(chirp))
	}
	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	rawChirpID := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(rawChirpID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}
	dbChirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	respChirp := Chirp(dbChirp)
	respondWithJSON(w, http.StatusOK, respChirp)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	userID, err := auth.ValidateJWT(tokenString, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	type parameters struct {
		Body string `json:"body"`
	}
	params := parameters{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
	dbParams := database.CreateChirpParams{
		Body:   cleanChirp(params.Body),
		UserID: userID,
	}
	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating chirp")
		return
	}
	respChirp := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	respondWithJSON(w, http.StatusCreated, respChirp)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "error")
		return
	}
	userID, err := auth.ValidateJWT(tokenString, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "error")
		return
	}
	rawID := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(rawID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error")
		return
	}
	chirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "error")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error")
		return
	}
	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "error")
		return
	}

	err = cfg.dbQueries.DeleteChirp(r.Context(), chirp.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func cleanChirp(body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		lowerWord := strings.ToLower(word)
		if lowerWord == "kerfuffle" || lowerWord == "sharbert" || lowerWord == "fornax" {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}
