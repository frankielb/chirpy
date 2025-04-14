package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	type chirpIn struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	// read it into struct
	decoder := json.NewDecoder(r.Body)
	chirp := chirpIn{}
	if err := decoder.Decode(&chirp); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode chirp", err)
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
		UserID: chirp.UserID,
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
		UserID:    chirp.UserID.String(),
	}
	respondJSON(w, http.StatusCreated, response)

}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.DB.GetChirps(r.Context())
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
