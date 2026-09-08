package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

// This test fails if a server starts without an administrator key, which would
// expose product-management endpoints without authentication.
func TestLoadConfigRequiresAPIKey(t *testing.T) {
	t.Setenv("API_KEY", "")
	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "API_KEY") {
		t.Fatalf("LoadConfig() error = %v, want API_KEY error", err)
	}
}

// This test fails if extracting the application router breaks Bond's existing
// health check or generic QR endpoint.
func TestLegacyQRCodeAndHealthRemainAvailable(t *testing.T) {
	cfg := testConfig(t)
	router := NewRouter(cfg, nil)

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", health.Code, http.StatusOK)
	}

	qr := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/?secret=legacy-secret&size=128&content=test", nil)
	router.ServeHTTP(qr, request)
	if qr.Code != http.StatusOK {
		t.Fatalf("legacy QR status = %d, want %d", qr.Code, http.StatusOK)
	}
	if qr.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("legacy QR content type = %q, want image/png", qr.Header().Get("Content-Type"))
	}
	if !bytes.HasPrefix(qr.Body.Bytes(), []byte{137, 80, 78, 71}) {
		t.Fatal("legacy QR body is not a PNG")
	}
}

func testConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		Port:          "18080",
		Secret:        "legacy-secret",
		APIKey:        "admin-key",
		PublicBaseURL: "http://192.168.1.20:18080",
		DataFile:      t.TempDir() + "/products.json",
		MaxSize:       1024,
		RecoveryLevel: qrcode.High,
		EnableLogs:    false,
	}
}
