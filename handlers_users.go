package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/azizu06/Chirpy/internal/auth"
	"github.com/azizu06/Chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	ChirpyRed bool      `json:"is_chirpy_red"`
}

type LoginResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	ChirpyRed    bool      `json:"is_chirpy_red"`
	RefreshToken string    `json:"refresh_token"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	params := parameters{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	dbParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}
	user, err := cfg.dbQueries.CreateUser(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}
	respUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		ChirpyRed: user.IsChirpyRed,
	}
	respondWithJSON(w, http.StatusCreated, respUser)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
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
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	params := parameters{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error")
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error")
		return
	}
	dbParams := database.UpdateUserParams{
		Email:          params.Email,
		ID:             userID,
		HashedPassword: hashedPassword,
	}
	user, err := cfg.dbQueries.UpdateUser(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error")
		return
	}
	respUser := User{
		Email:     user.Email,
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ChirpyRed: user.IsChirpyRed,
	}
	respondWithJSON(w, http.StatusOK, respUser)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	params := parameters{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	matches, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	if !matches {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create token")
		return
	}
	refreshToken := auth.MakeRefreshToken()
	dbParams := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	}
	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create refresh token")
		return
	}
	respUser := LoginResponse{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		ChirpyRed:    user.IsChirpyRed,
		RefreshToken: refreshToken,
	}
	respondWithJSON(w, http.StatusOK, respUser)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	user, err := cfg.dbQueries.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	type resp struct {
		Token string `json:"token"`
	}
	respToken := resp{
		Token: token,
	}
	respondWithJSON(w, http.StatusOK, respToken)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	err = cfg.dbQueries.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not revoke token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpgradeUser(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "error")
		return
	}
	if apiKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "error")
		return
	}
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserId uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	params := parameters{}
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error")
		return
	}
	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_, err = cfg.dbQueries.UpgradeUser(r.Context(), params.Data.UserId)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Error")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
