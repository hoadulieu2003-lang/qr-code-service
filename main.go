package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
	qrcode "github.com/skip2/go-qrcode"
)

// Alias for os.GetEnv, with support for fallback value, and boolean normalization
func getEnv(key string, fallback ...string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		if len(fallback) > 0 {
			value = fallback[0]
		} else {
			value = ""
		}
	} else {
		// Quotes removal
		value = strings.Trim(value, "\"")

		// Boolean normalization
		mapping := map[string]string{
			"0":     "FALSE",
			"off":   "FALSE",
			"false": "FALSE",
			"1":     "TRUE",
			"on":    "TRUE",
			"true":  "TRUE",
		}
		normalized, isBool := mapping[strings.ToLower(value)]
		if isBool {
			value = normalized
		}
	}

	return value
}

// Validates the secret using constant-time comparison to prevent timing attacks.
// Supports:
// 1. X-Bond-Secret header
// 2. Authorization: Bearer <secret>
// 3. 'secret' query parameter (for backward compatibility)
func validateSecret(r *http.Request, expectedSecret string) bool {
	if expectedSecret == "" {
		return false
	}

	expectedBytes := []byte(expectedSecret)

	// 1. Check X-Bond-Secret HTTP Header
	if headerSecret := r.Header.Get("X-Bond-Secret"); headerSecret != "" {
		if subtle.ConstantTimeCompare([]byte(headerSecret), expectedBytes) == 1 {
			return true
		}
	}

	// 2. Check Authorization Bearer HTTP Header
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), expectedBytes) == 1 {
			return true
		}
	}

	// 3. Check 'secret' Query Parameter (backward compatible)
	if querySecret := r.URL.Query().Get("secret"); querySecret != "" {
		if subtle.ConstantTimeCompare([]byte(querySecret), expectedBytes) == 1 {
			return true
		}
	}

	return false
}

// Sends an HTTP request to the /health endpoint on the running server
// and returns an exit code. This assumes that a first "./bond" program runs,
// and this was called via "./bond --healthcheck" on the exact same machine / container
func runHealthcheck() int {
	protocol := "http"
	if getEnv("SSL") == "TRUE" {
		protocol = "https"
	}

	healthURL := fmt.Sprintf("%s://localhost:%s/health", protocol, getEnv("PORT", "80"))

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Health check failed: status %d\n", resp.StatusCode)
		return 1
	}

	fmt.Println("OK")
	return 0
}

// setupRouter constructs and configures the Chi HTTP router
func setupRouter() *chi.Mux {
	recoveryLevels := map[string]qrcode.RecoveryLevel{
		"LOW":     qrcode.Low,
		"MEDIUM":  qrcode.Medium,
		"HIGH":    qrcode.High,
		"HIGHEST": qrcode.Highest,
	}

	app := chi.NewRouter()

	// Set up basic middleware
	if getEnv("ENABLE_LOGS") == "TRUE" {
		app.Use(middleware.Logger)
	}
	app.Use(middleware.Recoverer)

	// CORS-specific for OPTIONS
	app.Options("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Bond-Secret")
		w.WriteHeader(http.StatusOK)
	})

	// GET /
	app.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Bond-Secret")

		expectedSecret := getEnv("SECRET")
		if !validateSecret(r, expectedSecret) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden: Invalid or missing secret"))
			return
		}

		content := r.URL.Query().Get("content")
		size := r.URL.Query().Get("size")

		// Ensure Size and Content are provided
		if content == "" || size == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request: Both 'content' and 'size' parameters are required"))
			return
		}

		realSize, err := strconv.Atoi(size)
		if err != nil {
			// Client sent non-integer size: Return 400 Bad Request instead of 500 Internal Server Error
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Bad Request: 'size' must be a valid integer: %v", err)))
			return
		}

		maxSize, err := strconv.Atoi(getEnv("MAX_SIZE", "1024"))
		if err != nil || maxSize <= 0 {
			maxSize = 1024
		}

		if realSize <= 0 || realSize > maxSize {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Bad Request: 'size' must be between 1 and %d", maxSize)))
			return
		}

		// Resolve recovery level safely
		configuredLevel := strings.ToUpper(getEnv("RECOVERY_LEVEL", "MEDIUM"))
		recLevel, exists := recoveryLevels[configuredLevel]
		if !exists {
			log.Printf("Warning: Unknown RECOVERY_LEVEL '%s', defaulting to MEDIUM", configuredLevel)
			recLevel = qrcode.Medium
		}

		// Generate the QR code
		png, err := qrcode.Encode(content, recLevel, realSize)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Bad Request: Failed to encode QR code: %v", err)))
			return
		}

		// Correct Header order: Write headers BEFORE body
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(png)))
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write(png); err != nil {
			log.Printf("Error writing QR code response: %v", err)
		}
	})

	// GET /health
	app.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return app
}

// Entrypoint
func main() {
	godotenv.Load("default.env")

	// Load custom settings via .env file
	err := godotenv.Overload(".env")
	if err != nil {
		log.Print("No .env file provided, will continue with system env")
	}

	// Run health check when "--healthcheck" argument is supplied
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		exitCode := runHealthcheck()
		os.Exit(exitCode)
	}

	port := getEnv("PORT", "80")
	app := setupRouter()

	// Hardened HTTP Server with explicit timeouts to prevent Slowloris attacks
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      app,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run the server in a separate goroutine
	go func() {
		log.Printf("Bond QR Service starting on port %s", port)
		var srvErr error
		if getEnv("SSL") == "TRUE" {
			srvErr = srv.ListenAndServeTLS("certificate.pem", "key.pem")
		} else {
			srvErr = srv.ListenAndServe()
		}
		if srvErr != nil && srvErr != http.ErrServerClosed {
			log.Fatalf("Bond server failure: %v", srvErr)
		}
	}()

	// Graceful Shutdown handling on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received termination signal (%v). Commencing graceful shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced shutdown with error: %v", err)
	} else {
		log.Print("Server gracefully stopped. All in-flight requests drained.")
	}
}
