package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	// init counter
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	// init router
	mux := http.NewServeMux()

	// shows where files are on my mach
	fileServer := http.FileServer(http.Dir("."))
	// the /app isnt used in paths on mach, so remove
	handler := http.StripPrefix("/app", fileServer)
	// setup file server with wrapper
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	// register handlers for various things
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/reset", apiCfg.metricsReset)

	// create the server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}

type apiConfig struct {
	fileserverHits atomic.Int32
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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Hits: %d", cfg.fileserverHits.Load())
}
func (cfg *apiConfig) metricsReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
}
