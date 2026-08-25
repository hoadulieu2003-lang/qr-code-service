package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// This test fails if an unauthenticated caller can create public trace records.
func TestCreateProductRequiresAPIKey(t *testing.T) {
	router := newTestRouter(t, "http://192.168.1.20:18080")
	request := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

// This test fails if the API creates a product but returns a QR URL unrelated
// to the public trace URL that phones must scan.
func TestCreateProductReturnsTraceAndQRURLs(t *testing.T) {
	baseURL := "http://192.168.1.20:18080"
	router := newTestRouter(t, baseURL)
	body, err := json.Marshal(validProduct())
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(body))
	request.Header.Set("X-API-Key", "admin-key")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var payload struct {
		TraceCode string `json:"trace_code"`
		TraceURL  string `json:"trace_url"`
		QRURL     string `json:"qr_url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.TraceCode != "SP-DEMO-001" {
		t.Fatalf("trace_code = %q", payload.TraceCode)
	}
	if payload.TraceURL != baseURL+"/trace/SP-DEMO-001" {
		t.Fatalf("trace_url = %q", payload.TraceURL)
	}
	if payload.QRURL != baseURL+"/api/products/SP-DEMO-001/qr.png" {
		t.Fatalf("qr_url = %q", payload.QRURL)
	}
}

func TestCreateProductRejectsUnknownJSONField(t *testing.T) {
	router := newTestRouter(t, "http://192.168.1.20:18080")
	request := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBufferString(`{"unexpected":true}`))
	request.Header.Set("X-API-Key", "admin-key")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateProductRejectsDuplicateTraceCode(t *testing.T) {
	router := newTestRouter(t, "http://192.168.1.20:18080")
	body, err := json.Marshal(validProduct())
	if err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(body))
		request.Header.Set("X-API-Key", "admin-key")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		want := http.StatusCreated
		if attempt == 2 {
			want = http.StatusConflict
		}
		if response.Code != want {
			t.Fatalf("attempt %d status = %d, want %d", attempt, response.Code, want)
		}
	}
}

func TestCreateProductReturnsServiceUnavailableWithoutPublicBaseURL(t *testing.T) {
	router := newTestRouter(t, "")
	body, err := json.Marshal(validProduct())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(body))
	request.Header.Set("X-API-Key", "admin-key")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func newTestRouter(t *testing.T, publicBaseURL string) http.Handler {
	t.Helper()
	config := testConfig(t)
	config.PublicBaseURL = publicBaseURL
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(config, store)
}
