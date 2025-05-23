package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/frankielb/chirpy/internal/auth"
	"github.com/frankielb/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type userIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {

	// read it into struct
	decoder := json.NewDecoder(r.Body)
	newUser := userIn{}
	if err := decoder.Decode(&newUser); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode new user", err)

	}
	// hash the password
	hash, err := auth.HashPassword(newUser.Password)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldnt hash password", err)
		return
	}
	// create the user
	dbUser, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          newUser.Email,
		HashedPassword: hash,
	})
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't create user", err)
	}
	user := User{
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: false,
	}
	respondJSON(w, http.StatusCreated, user)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		//ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
	}
	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	userReq := loginRequest{}
	if err := decoder.Decode(&userReq); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode new user", err)
		return
	}
	user, err := cfg.DB.GetUserByEmail(r.Context(), userReq.Email)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}
	// check password
	if err := auth.CheckPasswordHash(user.HashedPassword, userReq.Password); err != nil {
		respondJSONError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	//get expire time
	expirationTime := time.Hour

	// make token
	token, err := auth.MakeJWT(user.ID,
		cfg.Secret,
		expirationTime)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "coiuldnt create auth token", err)
		return
	}

	// make refresh token
	refresh, err := auth.MakeRefreshToken()
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "coiuldnt create auth token", err)
		return
	}
	// add to db
	if _, err := cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refresh,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	}); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "coiuldnt add re token to db", err)
		return
	}
	respondJSON(w, http.StatusOK, response{
		User: User{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
		Token:        token,
		RefreshToken: refresh,
	})
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	// get token from header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "couldn't find bearer token", err)
		return
	}
	// check token is in db and good
	rTokenDB, err := cfg.DB.GetRefreshTokenFromToken(r.Context(), refreshToken)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "invalid token: nf", err)
		return
	}
	if rTokenDB.ExpiresAt.Before(time.Now()) {
		respondJSONError(w, http.StatusUnauthorized, "invalid token: exp", nil)
		return
	}
	if rTokenDB.RevokedAt.Valid {
		respondJSONError(w, http.StatusUnauthorized, "invalid token: rvkd", nil)
		return
	}
	// make new jwt
	accessToken, err := auth.MakeJWT(rTokenDB.UserID, cfg.Secret, time.Hour)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "couldn't create access token", err)
		return
	}
	// respond
	type response struct {
		Token string `json:"token"`
	}
	respondJSON(w, http.StatusOK, response{Token: accessToken})
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	// get token from header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "couldn't find bearer token", err)
		return
	}
	if err := cfg.DB.RevokeToken(r.Context(), refreshToken); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "couldn't revoke token", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) updatePswdEmlHandler(w http.ResponseWriter, r *http.Request) {
	// find user via jwt
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "couldn't find bearer token", err)
		return
	}

	userId, err := auth.ValidateJWT(refreshToken, cfg.Secret)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "bad token", err)
	}

	// read req
	decoder := json.NewDecoder(r.Body)
	newPwdEml := userIn{}
	if err := decoder.Decode(&newPwdEml); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode new user", err)
		return
	}
	// hash password
	hashedPswd, err := auth.HashPassword(newPwdEml.Password)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "couldn't hash password", err)
		return
	}
	// update in DB
	if err := cfg.DB.UpdatePswdEml(r.Context(), database.UpdatePswdEmlParams{
		HashedPassword: hashedPswd,
		Email:          newPwdEml.Email,
		ID:             userId,
	}); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "couldn't update", err)
		return
	}
	// get updated user for out
	userOut, err := cfg.DB.GetUserByEmail(r.Context(), newPwdEml.Email)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "didnt update", err)
		return
	}
	respondJSON(w, http.StatusOK, User{
		ID:          userOut.ID,
		CreatedAt:   userOut.CreatedAt,
		UpdatedAt:   userOut.UpdatedAt,
		Email:       userOut.Email,
		IsChirpyRed: userOut.IsChirpyRed,
	})

}

func (cfg *apiConfig) upgradeHandler(w http.ResponseWriter, r *http.Request) {
	// check apikey
	apikey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "couldn't get apiKey", err)
		return
	}
	if apikey != cfg.PolkaKey {
		respondJSONError(w, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	type data struct {
		UserID uuid.UUID `json:"user_id"`
	}
	type req struct {
		Event string `json:"event"`
		Data  data   `json:"data"`
	}
	// read req
	decoder := json.NewDecoder(r.Body)
	request := req{}
	if err := decoder.Decode(&request); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode request", err)
		return
	}
	if request.Event != "user.upgraded" {
		respondJSON(w, http.StatusNoContent, nil)
		return
	}
	// upgrade
	if err := cfg.DB.UpgradeRedByID(r.Context(), request.Data.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, http.StatusNotFound, "user not found", err)
			return
		}
		respondJSONError(w, http.StatusInternalServerError, "Couldn't upgrade user", err)
		return
	}
	respondJSON(w, http.StatusNoContent, nil)

}
