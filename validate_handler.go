package main

import (
	"encoding/json"
	"net/http"
	"strings"
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

	//check swearwords
	profanes := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}
	words := strings.Split(parameter.Body, " ")
	for i, word := range words {
		lower := strings.ToLower(word)
		if profanes[lower] {
			words[i] = "****"
		}
	}
	cleanText := strings.Join(words, " ")

	// good
	type cleanOut struct {
		CleanBody string `json:"cleaned_body"`
	}
	respondJSON(w, http.StatusOK, cleanOut{
		CleanBody: cleanText,
	})
}
