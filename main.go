// Package main implements a simple HTTP server that serves static files
// under the "/app" prefix and provides basic health, metrics, and reset endpoints.
package main

// Import the standard net/http package, which provides utilities for building HTTP servers
import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	// fileServerHits tracks how many times the static file server has been hit.
	// We use an atomic counter because the handler may be called concurrently
	// from multiple goroutines handling HTTP requests.
	fileServerHits atomic.Int32
}

// middlewareMetricInc returns a middleware that increments the file server hit counter
// before delegating to the next handler. This is applied to the static file handler
// so that each request served from the "/app" prefix is counted.
func (cfg *apiConfig) middlewareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// handlerMetrics writes the current number of file server hits as plain text.
// It is exposed at the "/metrics" endpoint and is useful for debugging or simple
// monitoring of how many static files have been requested.
func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, _ *http.Request) {
	hits := cfg.fileServerHits.Load()
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	html := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
	w.Write([]byte(html))
}

// handlerReset resets the file server hit counter back to zero.
// It is mapped to the "/reset" endpoint using the POST method.
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, _ *http.Request) {
	cfg.fileServerHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

// main is the entry point of the program. Every Go executable starts execution here.
func main() {
	// ---------------------------------------------------------------------
	// STEP 1: Create a request multiplexer (router)
	// ---------------------------------------------------------------------
	// A ServeMux matches incoming request URLs to registered handler functions.
	// We’ll use it to tell the server how to handle different paths.
	mux := http.NewServeMux()

	// ---------------------------------------------------------------------
	// STEP 2: Configure the HTTP server
	// ---------------------------------------------------------------------
	// The Server struct holds configuration for the HTTP server.
	//   • Addr – the network address to listen on. ":8080" means “all interfaces on port 8080”.
	//   • Handler – the object that will actually process each request. We give it our mux.
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	} // server is now ready, but not started yet.

	c := apiConfig{}

	// ---------------------------------------------------------------------
	// STEP 3: Register a handler for the root path ("/")
	// ---------------------------------------------------------------------
	// http.FileServer serves static files from a directory on disk.
	// http.Dir(".") tells it to use the current working directory as the root.
	// By handling the "/" pattern, every request that doesn’t match a more specific
	// route will fall back to serving a file from this directory.
	// Serve static files under the "/app" URL prefix. The StripPrefix middleware removes the "/app"
	// prefix before the request reaches the file server so that the file server sees the correct
	// relative file paths. The custom middlewareMetricInc wrapper increments the hit counter for each
	// request served from this handler.
	mux.Handle("/app/", c.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// healthz is a simple liveness endpoint used by the platform to verify the service is up.
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Expose the current hit count at "/metrics".
	mux.HandleFunc("GET /admin/metrics", c.handlerMetrics)

	// Reset the hit counter via a POST request to "/reset".
	mux.HandleFunc("POST /admin/reset", c.handlerReset)

	// ---------------------------------------------------------------------
	// STEP 4: Start the server and block until it stops
	// ---------------------------------------------------------------------
	// ListenAndServe starts the HTTP server on the configured address and blocks
	// until the program is terminated or an error occurs.
	server.ListenAndServe()
}
