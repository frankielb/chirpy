package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/frankielb/chirpy/internal/auth"
	"github.com/frankielb/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	respondJSON(w, http.StatusCreated, user)
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userReq := userIn{}
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
	userOut := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	respondJSON(w, http.StatusOK, userOut)

}
