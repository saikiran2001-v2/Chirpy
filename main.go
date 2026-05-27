package main

// Import the standard net/http package, which provides utilities for building HTTP servers
import (
	"net/http"
)

// The entry point of the program. Every Go executable starts execution here.
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

	// ---------------------------------------------------------------------
	// STEP 3: Register a handler for the root path ("/")
	// ---------------------------------------------------------------------
	// http.FileServer serves static files from a directory on disk.
	// http.Dir(".") tells it to use the current working directory as the root.
	// By handling the "/" pattern, every request that doesn’t match a more specific
	// route will fall back to serving a file from this directory.
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// ---------------------------------------------------------------------
	// STEP 4: Start the server and block until it stops
	// ---------------------------------------------------------------------
	// ListenAndServe starts listening on the address we specified.
	// It blocks the main goroutine, handling requests until the process is stopped
	// (e.g., with Ctrl+C) or an error occurs.
	server.ListenAndServe()
}
