package main

import (
	"encoding/json"
	"net/http"
)

func validateHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	// read it into struct
	decoder := json.NewDecoder(r.Body)
	parameter := parameters{}
	if err := decoder.Decode(&parameter); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)

	}
	// too long
	if len(parameter.Body) > 140 {
		respondJSONError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	// good
	type validOut struct {
		Valid bool `json:"valid"`
	}
	respondJSON(w, http.StatusOK, validOut{
		Valid: true,
	})
}
