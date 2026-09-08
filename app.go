package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	qrcode "github.com/skip2/go-qrcode"
)

// validateSecret performs constant-time comparison to prevent timing attacks.
// Supports:
// 1. X-Bond-Secret header
// 2. Authorization: Bearer <secret>
// 3. 'secret' query parameter (backward compatible)
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

func NewRouter(config Config, store *ProductStore) http.Handler {
	router := chi.NewRouter()
	if config.EnableLogs {
		router.Use(middleware.Logger)
	}
	router.Use(middleware.Recoverer)

	router.Options("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Bond-Secret")
		w.WriteHeader(http.StatusNoContent)
	})

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Bond-Secret")

		if !validateSecret(r, config.Secret) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden: Invalid or missing secret"))
			return
		}

		content := r.URL.Query().Get("content")
		size := r.URL.Query().Get("size")
		if content == "" || size == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request: Both 'content' and 'size' parameters are required"))
			return
		}

		realSize, err := strconv.Atoi(size)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Bad Request: 'size' must be a valid integer: %v", err)))
			return
		}
		if realSize <= 0 || realSize > config.MaxSize {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Bad Request: 'size' must be between 1 and %d", config.MaxSize)))
			return
		}

		png, err := qrcode.Encode(content, config.RecoveryLevel, realSize)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Bad Request: Failed to encode QR code: %v", err)))
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(png)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(png)
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	if store != nil {
		router.Post("/api/products", createProductHandler(config, store))
		router.Get("/api/products/{traceCode}/qr.png", productQRHandler(config, store))
		router.Get("/trace/{traceCode}", publicTracePageHandler(store))
		router.Get("/admin", adminFormHandler())
		router.Post("/admin/products", adminCreateProductHandler(config, store))
	}

	return router
}
