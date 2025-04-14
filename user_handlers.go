package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	type emailIn struct {
		Email string `json:"email"`
	}
	// read it into struct
	decoder := json.NewDecoder(r.Body)
	email := emailIn{}
	if err := decoder.Decode(&email); err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't decode email", err)

	}
	// create the user
	dbUser, err := cfg.DB.CreateUser(r.Context(), email.Email)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Couldn't create user", err)
	}
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	respondJSON(w, http.StatusCreated, user)
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
