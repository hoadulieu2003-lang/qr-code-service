package main

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	qrcode "github.com/skip2/go-qrcode"
)

// ProductStore is intentionally empty in Task 1. Task 2 adds its persistence
// fields and methods while preserving the router dependency boundary.
type ProductStore struct{}

func NewRouter(config Config, _ *ProductStore) http.Handler {
	router := chi.NewRouter()
	if config.EnableLogs {
		router.Use(middleware.Logger)
	}
	router.Use(middleware.Recoverer)

	router.Options("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", http.MethodGet)
		w.WriteHeader(http.StatusNoContent)
	})

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", http.MethodGet)

		if r.URL.Query().Get("secret") != config.Secret {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		content := r.URL.Query().Get("content")
		size := r.URL.Query().Get("size")
		if content == "" || size == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		realSize, err := strconv.Atoi(size)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		if realSize <= 0 || realSize > config.MaxSize {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		png, err := qrcode.Encode(content, config.RecoveryLevel, realSize)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", http.MethodGet)
		w.WriteHeader(http.StatusOK)
	})

	return router
}
