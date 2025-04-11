package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondJSONError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	// server errors
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errResponse struct {
		Err string `json:"error"`
	}
	respondJSON(w, code, errResponse{
		Err: msg,
	})
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	//interface{} means anything, so any struct

	// metadata, tells the client its json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(jsonData)
}
