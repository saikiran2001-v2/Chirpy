// Package main implements a simple HTTP server that serves static files
// under the "/app" prefix and provides basic health, metrics, and reset endpoints.
package main

// Import the standard net/http package, which provides utilities for building HTTP servers
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/saikiran2001-v2/Chirpy/internal/auth"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/saikiran2001-v2/Chirpy/internal/database"
)

type chirp struct {
	db *database.Queries
}

type apiConfig struct {
	// fileServerHits tracks how many times the static file server has been hit.
	// We use an atomic counter because the handler may be called concurrently
	// from multiple goroutines handling HTTP requests.
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtsecret      string
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
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}
	cfg.db.DeleteAllUsers(r.Context())
	cfg.fileServerHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
		Error string `json:"error"`
	}

	respBody := returnVals{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

// main is the entry point of the program. Every Go executable starts execution here.
func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	dbQueries := database.New(db)

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

	c := apiConfig{
		db:        dbQueries,
		platform:  platform,
		jwtsecret: jwtSecret,
	}

	chir := chirp{
		db: dbQueries,
	}

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

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		type response struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}
		hashedPass, err := auth.HashPassword(params.Password)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}
		ctx := r.Context()
		user, err := c.db.CreateUser(ctx, database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hashedPass,
		})
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		respondWithJson(w, 201, response{
			ID:        user.ID.String(),
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		})
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		type returnVals struct {
			Id           string `json:"id"`
			CreatedAt    string `json:"created_at"`
			UpdatedAt    string `json:"updated_at"`
			Email        string `json:"email"`
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		ctx := r.Context()
		user, err := chir.db.GetUser(ctx, params.Email)

		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		if user.HashedPassword == "" {
			respondWithError(w, 500, "no such user exists in our database")
			return
		}

		checkMatch, err := auth.CheckPaswordHash(params.Password, user.HashedPassword)
		if err != nil || !checkMatch {
			respondWithError(w, 401, "Incorrect email or password")
			return
		}

		token, err := auth.MakeJWT(user.ID, c.jwtsecret, time.Hour)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		refreshTokenStr := auth.MakeRefreshToken()
		if refreshTokenStr == "" {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		_, err = c.db.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{Token: refreshTokenStr, UserID: user.ID})
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		resp := returnVals{
			Id:           user.ID.String(),
			CreatedAt:    user.CreatedAt.String(),
			UpdatedAt:    user.UpdatedAt.String(),
			Email:        user.Email,
			Token:        token,
			RefreshToken: refreshTokenStr,
		}

		respondWithJson(w, 200, resp)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type returnVals struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    string    `json:"user_id"`
		}

		ctx := r.Context()
		chirps, err := chir.db.GetChirps(ctx)
		if err != nil {
			respondWithError(w, 500, "Something went wrong at get chirps")
			return
		}

		resp := make([]returnVals, len(chirps))
		for i, chirp := range chirps {
			resp[i] = returnVals{
				ID:        chirp.ID.String(),
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID.String(),
			}
		}
		respondWithJson(w, 200, resp)
	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}

		ctx := r.Context()

		refreshTokenstr, err := c.db.GetRefreshToken(ctx, token)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}
		if refreshTokenstr.ExpiresAt.Time.Before(time.Now()) {
			respondWithError(w, 401, "Token expired")
			return
		}
		if refreshTokenstr.RevokedAt.Valid {
			respondWithError(w, 401, "Token revoked")
			return
		}

		type returnVals struct {
			Token string `json:"token"`
		}

		user, err := c.db.GetUserFromRefreshToken(ctx, refreshTokenstr.Token)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}

		accessToken, err := auth.MakeJWT(user.ID, c.jwtsecret, time.Hour)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		response := returnVals{
			Token: accessToken,
		}

		respondWithJson(w, 200, response)
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, 500, err.Error())
			return
		}

		ctx := r.Context()

		refreshTokenStr, err := c.db.GetRefreshToken(ctx, refreshToken)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}

		err = c.db.RevokeRefreshToken(ctx, refreshTokenStr.Token)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(204)
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		chirpID := r.PathValue("chirpID")

		type returnVals struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    string    `json:"user_id"`
		}

		ctx := r.Context()
		chirpIDConv, err := uuid.Parse(chirpID)
		if err != nil {
			respondWithError(w, 500, "Something went wrong while converting chirp id into UUID")
			return
		}

		chirp, err := chir.db.GetChirp(ctx, chirpIDConv)

		if err == sql.ErrNoRows {
			respondWithError(w, 404, "Chirp not found")
			return
		}

		if err != nil {
			respondWithError(w, 500, "Something went wrong when calling GetChirp")
			return
		}

		response := returnVals{
			ID:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		}

		respondWithJson(w, 200, response)
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}

		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, 401, "Missing bearer token")
			return
		}

		userID, err := auth.ValidateJWT(token, c.jwtsecret)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}

		params := parameters{}
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 500, "Something went wrong0")
			return
		}

		if len(params.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}

		newSplit := strings.Split(params.Body, " ")
		for i, word := range newSplit {
			if strings.ToLower(word) == "kerfuffle" {
				newSplit[i] = "****"
			}
			if strings.ToLower(word) == "sharbert" {
				newSplit[i] = "****"
			}
			if strings.ToLower(word) == "fornax" {
				newSplit[i] = "****"
			}
		}

		cleanedBody := strings.Join(newSplit, " ")
		type returnVals struct {
			ID          string    `json:"id"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
			CleanedBody string    `json:"body"`
			UserID      string    `json:"user_id"`
		}

		ctx := r.Context()
		chirp, err := chir.db.CreateChirp(ctx, database.CreateChirpParams{
			Body:   cleanedBody,
			UserID: userID,
		})

		if err != nil {
			respondWithError(w, 500, "Something went wrong2")
			return
		}

		respBody := returnVals{
			ID:          chirp.ID.String(),
			CreatedAt:   chirp.CreatedAt,
			UpdatedAt:   chirp.UpdatedAt,
			CleanedBody: cleanedBody,
			UserID:      chirp.UserID.String(),
		}

		respondWithJson(w, 201, respBody)
	})

	// ---------------------------------------------------------------------
	// STEP 4: Start the server and block until it stops
	// ---------------------------------------------------------------------
	// ListenAndServe starts the HTTP server on the configured address and blocks
	// until the program is terminated or an error occurs.
	server.ListenAndServe()
}
