package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/frankielb/chirpy/internal/database"
	"github.com/google/uuid"
)

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
	type cleanOut struct {
		Id        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	chirpOut, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanText,
		UserID: chirp.UserID,
	})
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't create chirp", err)
		return
	}
	response := cleanOut{
		Id:        chirpOut.ID.String(),
		CreatedAt: chirpOut.CreatedAt,
		UpdatedAt: chirpOut.UpdatedAt,
		Body:      chirpOut.Body,
		UserID:    chirp.UserID,
	}
	respondJSON(w, http.StatusCreated, response)

}
