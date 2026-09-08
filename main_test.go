package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthcheckEndpoint(t *testing.T) {
	router := NewRouter(testConfig(t), nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "OK"
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestSecretValidation(t *testing.T) {
	cfg := testConfig(t)
	router := NewRouter(cfg, nil)

	tests := []struct {
		name           string
		headerSecret   string
		bearerToken    string
		querySecret    string
		expectedStatus int
	}{
		{
			name:           "Valid secret in query",
			querySecret:    cfg.Secret,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid secret in X-Bond-Secret header",
			headerSecret:   cfg.Secret,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid secret in Authorization Bearer header",
			bearerToken:    cfg.Secret,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid secret",
			querySecret:    "wrong-secret",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Missing secret",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/?size=128&content=https://example.com"
			if tt.querySecret != "" {
				url += "&secret=" + tt.querySecret
			}

			req, _ := http.NewRequest("GET", url, nil)
			if tt.headerSecret != "" {
				req.Header.Set("X-Bond-Secret", tt.headerSecret)
			}
			if tt.bearerToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.bearerToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Test '%s': expected status %d, got %d", tt.name, tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				contentType := rr.Header().Get("Content-Type")
				if contentType != "image/png" {
					t.Errorf("Test '%s': expected Content-Type 'image/png', got '%s'", tt.name, contentType)
				}
				// Verify PNG header
				pngHeader := []byte("\x89PNG\r\n\x1a\n")
				if !bytes.HasPrefix(rr.Body.Bytes(), pngHeader) {
					t.Errorf("Test '%s': body does not start with valid PNG header", tt.name)
				}
			}
		})
	}
}

func TestInputValidation(t *testing.T) {
	cfg := testConfig(t)
	cfg.MaxSize = 512
	router := NewRouter(cfg, nil)

	tests := []struct {
		name           string
		size           string
		content        string
		expectedStatus int
	}{
		{
			name:           "Missing content",
			size:           "128",
			content:        "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing size",
			size:           "",
			content:        "ValidContent",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Size is string (not an integer)",
			size:           "abc",
			content:        "ValidContent",
			expectedStatus: http.StatusBadRequest, // Must be 400, not 500!
		},
		{
			name:           "Size is zero",
			size:           "0",
			content:        "ValidContent",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Size is negative",
			size:           "-50",
			content:        "ValidContent",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Size exceeds MAX_SIZE (512)",
			size:           "1024",
			content:        "ValidContent",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/?secret=" + cfg.Secret
			if tt.size != "" {
				url += "&size=" + tt.size
			}
			if tt.content != "" {
				url += "&content=" + tt.content
			}

			req, _ := http.NewRequest("GET", url, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Test '%s': expected status %d, got %d (body: %s)", tt.name, tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCORSOptions(t *testing.T) {
	router := NewRouter(testConfig(t), nil)

	req, _ := http.NewRequest("OPTIONS", "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Errorf("OPTIONS returned wrong status: got %d want %d or %d", rr.Code, http.StatusNoContent, http.StatusOK)
	}

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("OPTIONS missing Access-Control-Allow-Origin: got '%s'", origin)
	}
}
