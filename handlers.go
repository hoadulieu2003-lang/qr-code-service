package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	qrcode "github.com/skip2/go-qrcode"
)

type createProductResponse struct {
	TraceCode string `json:"trace_code"`
	TraceURL  string `json:"trace_url"`
	QRURL     string `json:"qr_url"`
	qrPNG     []byte
}

var (
	ErrInvalidProduct            = errors.New("invalid product")
	ErrPublicTraceURLUnavailable = errors.New("public trace URL is unavailable")
	ErrQRCodeUnavailable         = errors.New("QR code is unavailable")
)

type invalidProductError struct {
	cause error
}

func (e invalidProductError) Error() string {
	return e.cause.Error()
}

func (e invalidProductError) Unwrap() error {
	return ErrInvalidProduct
}

func hasAPIKeyValue(provided, expected string) bool {
	if provided == "" || expected == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func hasAPIKey(request *http.Request, expected string) bool {
	return hasAPIKeyValue(request.Header.Get("X-API-Key"), expected)
}

func createProduct(config Config, store *ProductStore, product Product) (createProductResponse, error) {
	product = product.normalized()
	if err := product.Validate(); err != nil {
		return createProductResponse{}, invalidProductError{cause: err}
	}
	traceURL, err := config.TraceURL(product.TraceCode)
	if err != nil {
		return createProductResponse{}, fmt.Errorf("%w: %v", ErrPublicTraceURLUnavailable, err)
	}
	png, err := qrcode.Encode(traceURL, qrcode.High, 512)
	if err != nil {
		return createProductResponse{}, fmt.Errorf("%w: %v", ErrQRCodeUnavailable, err)
	}
	if err := store.Create(product); err != nil {
		return createProductResponse{}, err
	}

	return createProductResponse{
		TraceCode: product.TraceCode,
		TraceURL:  traceURL,
		QRURL:     config.PublicBaseURL + "/api/products/" + product.TraceCode + "/qr.png",
		qrPNG:     png,
	}, nil
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

		result, err := createProduct(config, store, product)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidProduct):
				writeJSONError(writer, http.StatusBadRequest, err.Error())
			case errors.Is(err, ErrPublicTraceURLUnavailable):
				writeJSONError(writer, http.StatusServiceUnavailable, "public trace URL is unavailable")
			case errors.Is(err, ErrDuplicateTraceCode):
				writeJSONError(writer, http.StatusConflict, "trace code already exists")
			default:
				writeJSONError(writer, http.StatusInternalServerError, "could not save product")
			}
			return
		}

		writeJSON(writer, http.StatusCreated, result)
	}
}

func productQRHandler(config Config, store *ProductStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !hasAPIKey(request, config.APIKey) {
			writeJSONError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}

		product, err := store.Get(chi.URLParam(request, "traceCode"))
		if errors.Is(err, ErrProductNotFound) {
			writeJSONError(writer, http.StatusNotFound, "product not found")
			return
		}
		if err != nil {
			writeJSONError(writer, http.StatusInternalServerError, "could not load product")
			return
		}

		traceURL, err := config.TraceURL(product.TraceCode)
		if err != nil {
			writeJSONError(writer, http.StatusServiceUnavailable, "public trace URL is unavailable")
			return
		}
		png, err := qrcode.Encode(traceURL, qrcode.High, 512)
		if err != nil {
			writeJSONError(writer, http.StatusInternalServerError, "could not generate QR code")
			return
		}

		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write(png)
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
