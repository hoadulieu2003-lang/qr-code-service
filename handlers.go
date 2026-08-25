package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type createProductResponse struct {
	TraceCode string `json:"trace_code"`
	TraceURL  string `json:"trace_url"`
	QRURL     string `json:"qr_url"`
}

func hasAPIKey(request *http.Request, expected string) bool {
	provided := request.Header.Get("X-API-Key")
	if provided == "" || expected == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func createProductHandler(config Config, store *ProductStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !hasAPIKey(request, config.APIKey) {
			writeJSONError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
			writeJSONError(writer, http.StatusBadRequest, "Content-Type must be application/json")
			return
		}

		var product Product
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&product); err != nil {
			writeJSONError(writer, http.StatusBadRequest, "invalid product JSON")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeJSONError(writer, http.StatusBadRequest, "request body must contain one JSON object")
			return
		}

		product = product.normalized()
		if err := product.Validate(); err != nil {
			writeJSONError(writer, http.StatusBadRequest, err.Error())
			return
		}
		traceURL, err := config.TraceURL(product.TraceCode)
		if err != nil {
			writeJSONError(writer, http.StatusServiceUnavailable, "public trace URL is unavailable")
			return
		}
		if err := store.Create(product); err != nil {
			switch {
			case errors.Is(err, ErrDuplicateTraceCode):
				writeJSONError(writer, http.StatusConflict, "trace code already exists")
			default:
				writeJSONError(writer, http.StatusInternalServerError, "could not save product")
			}
			return
		}

		writeJSON(writer, http.StatusCreated, createProductResponse{
			TraceCode: product.TraceCode,
			TraceURL:  traceURL,
			QRURL:     config.PublicBaseURL + "/api/products/" + product.TraceCode + "/qr.png",
		})
	}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeJSONError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}
