package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/frankielb/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// db stuff
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	// init counter
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		DB:             dbQueries,
		Platform:       os.Getenv("PLATFORM"),
		Secret:         os.Getenv("SECRET"),
	}
	// init router
	mux := http.NewServeMux()

	// shows where files are on my mach
	fileServer := http.FileServer(http.Dir("."))
	// the /app isnt used in paths on mach, so remove
	fsHandler := http.StripPrefix("/app", fileServer)
	// setup file server with wrapper
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))

	// register handlers for various things
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	//mux.HandleFunc("POST /api/validate_chirp", validateHandler)
	mux.HandleFunc("POST /api/users", apiCfg.createUserHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpHandler)
	mux.HandleFunc("POST /api/login", apiCfg.loginHandler)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	mux.HandleFunc("PUT /api/users", apiCfg.updatePswdEmlHandler)

	// create the server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	Platform       string
	Secret         string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// takes handler and adds the count to it
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// increment count
		cfg.fileserverHits.Add(1)
		// og handler called
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(
		`<html>
		  <body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		  </body>
		</html>`,
		cfg.fileserverHits.Load())))
}
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.DB.DeleteAllUsers(r.Context())
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, "Failed to reset users database", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All users and server hits reset"))

}
