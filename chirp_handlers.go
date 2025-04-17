package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/frankielb/chirpy/internal/auth"
	"github.com/frankielb/chirpy/internal/database"
	"github.com/google/uuid"
)

type chirpJSON struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    string    `json:"user_id"`
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	// auth
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "unauthorized: no token", err)
		return
	}
	userID, err := auth.ValidateJWT(bearerToken, cfg.Secret)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "unauthorized: wrong user", err)
		return
	}

	type chirpIn struct {
		Body string `json:"body"`
		//UserID uuid.UUID `json:"user_id"`
	}
	// read it into struct
	decoder := json.NewDecoder(r.Body)
	chirp := chirpIn{}
	if err := decoder.Decode(&chirp); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode chirp", err)
		return
	}

	// too long
	if len(chirp.Body) > 140 {
		respondJSONError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	//check swearwords
	profanes := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}
	words := strings.Split(chirp.Body, " ")
	for i, word := range words {
		lower := strings.ToLower(word)
		if profanes[lower] {
			words[i] = "****"
		}
	}
	cleanText := strings.Join(words, " ")

	// good
	chirpOut, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanText,
		UserID: userID,
	})
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't create chirp", err)
		return
	}
	response := chirpJSON{
		Id:        chirpOut.ID.String(),
		CreatedAt: chirpOut.CreatedAt,
		UpdatedAt: chirpOut.UpdatedAt,
		Body:      chirpOut.Body,
		UserID:    chirpOut.UserID.String(),
	}
	respondJSON(w, http.StatusCreated, response)

}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	authorString := r.URL.Query().Get("author_id")
	// uses different query if there was a author id
	var chirps []database.Chirp
	var err error
	if authorString != "" {
		authorID, errParse := uuid.Parse(authorString)
		if errParse != nil {
			respondJSONError(w, http.StatusInternalServerError, "dodgy id", err)
			return
		}
		chirps, err = cfg.DB.GetChirpsByUser(r.Context(), authorID)
	} else {
		chirps, err = cfg.DB.GetChirps(r.Context())
	}

	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't get chirps", err)
		return
	}

	responses := []chirpJSON{}
	for _, chirp := range chirps {
		response := chirpJSON{
			Id:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		}
		responses = append(responses, response)
	}
	respondJSON(w, http.StatusOK, responses)
}

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondJSONError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}
	chirp, err := cfg.DB.GetChirp(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSONError(w, http.StatusNotFound, "Chirp not found", err)
			return
		}
		respondJSONError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}
	response := chirpJSON{
		Id:        chirp.ID.String(),
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID.String(),
	}
	respondJSON(w, http.StatusOK, response)

}

func (cfg *apiConfig) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	// get the chirp from DB via path
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondJSONError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := cfg.DB.GetChirp(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSONError(w, http.StatusNotFound, "Chirp not found", err)
			return
		}
		respondJSONError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	// find user via jwt
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "couldn't find bearer token", err)
		return
	}
	tokenID, err := auth.ValidateJWT(refreshToken, cfg.Secret)
	if err != nil {
		respondJSONError(w, http.StatusUnauthorized, "bad token", err)
		return
	}
	if tokenID != chirp.UserID {
		respondJSONError(w, http.StatusForbidden, "unauthorized", err)
		return
	}

	// delete the chirp
	if err := cfg.DB.DeleteChirpByID(r.Context(), chirpID); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "couldn't delete chirp", err)
		return
	}
	respondJSON(w, http.StatusNoContent, nil)

}
