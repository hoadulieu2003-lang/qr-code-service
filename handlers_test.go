package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// This test fails if the shared product-creation path returns URLs different
// from the public trace URL and printable QR URL required by API consumers.
func TestCreateProductServiceCreatesTraceURLs(t *testing.T) {
	config := testConfig(t)
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}

	result, err := createProduct(config, store, validProduct())
	if err != nil {
		t.Fatal(err)
	}
	if result.TraceURL != "http://192.168.1.20:18080/trace/SP-DEMO-001" {
		t.Fatalf("TraceURL = %q", result.TraceURL)
	}
	if result.QRURL != "http://192.168.1.20:18080/api/products/SP-DEMO-001/qr.png" {
		t.Fatalf("QRURL = %q", result.QRURL)
	}
}

// This test fails if invalid product data or an unusable public URL can be
// confused with storage errors by either the JSON API or the admin form.
func TestCreateProductServiceClassifiesFailures(t *testing.T) {
	config := testConfig(t)
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}

	invalid := validProduct()
	invalid.Origin = ""
	if _, err := createProduct(config, store, invalid); !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("invalid product error = %v", err)
	}

	config.PublicBaseURL = ""
	if _, err := createProduct(config, store, validProduct()); !errors.Is(err, ErrPublicTraceURLUnavailable) {
		t.Fatalf("public URL error = %v", err)
	}
}

// This test fails if extracting shared creation logic changes the validation
// error body relied on by existing JSON API clients.
func TestCreateProductPreservesValidationErrorBody(t *testing.T) {
	router := newTestRouter(t, "http://192.168.1.20:18080")
	product := validProduct()
	product.Origin = ""
	body, err := json.Marshal(product)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(body))
	request.Header.Set("X-API-Key", "admin-key")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error != "origin must not be empty" {
		t.Fatalf("error = %q, want original validation message", payload.Error)
	}
}

// This test fails if a trace code that cannot fit a High-recovery QR can be
// saved even though neither the admin page nor the printable PNG can use it.
func TestCreateProductServiceRejectsUnencodableQRCodeWithoutPersisting(t *testing.T) {
	config := testConfig(t)
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}
	product := validProduct()
	product.TraceCode = strings.Repeat("A", 3000)

	if _, err := createProduct(config, store, product); !errors.Is(err, ErrQRCodeUnavailable) {
		t.Fatalf("createProduct() error = %v, want ErrQRCodeUnavailable", err)
	}
	if _, err := store.Get(product.TraceCode); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("store.Get() error = %v, want ErrProductNotFound", err)
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

// This test fails if printable trace QR files can be downloaded without an
// administrator key or if the download stops being a valid PNG.
func TestProductQRRequiresAPIKeyAndReturnsPNG(t *testing.T) {
	router := routerWithDemoProduct(t)

	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/products/SP-DEMO-001/qr.png", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated QR status = %d, want %d", denied.Code, http.StatusUnauthorized)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/products/SP-DEMO-001/qr.png", nil)
	request.Header.Set("X-API-Key", "admin-key")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("QR status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("QR content type = %q, want image/png", response.Header().Get("Content-Type"))
	}
	if !bytes.HasPrefix(response.Body.Bytes(), []byte{137, 80, 78, 71}) {
		t.Fatal("QR body is not a PNG")
	}
}

func TestProductQRReturnsNotFoundForUnknownTraceCode(t *testing.T) {
	router := routerWithDemoProduct(t)
	request := httptest.NewRequest(http.MethodGet, "/api/products/SP-MISSING-001/qr.png", nil)
	request.Header.Set("X-API-Key", "admin-key")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q, want JSON error", response.Header().Get("Content-Type"))
	}
}

// This test fails if the public page loses essential product data or escapes
// are removed and product text can execute as page markup.
func TestPublicTracePageShowsFieldsAndEscapesProductText(t *testing.T) {
	config := testConfig(t)
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}
	product := validProduct()
	product.Name = "<script>alert(1)</script>"
	product.TraceCode = "SP-ESCAPE-001"
	if err := store.Create(product); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(config, store)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/trace/SP-ESCAPE-001", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, label := range []string{"Tên sản phẩm", "Mã sản phẩm / lô", "Ngày sản xuất", "Hạn dùng", "Nơi sản xuất", "Trạng thái xác thực"} {
		if !strings.Contains(body, label) {
			t.Fatalf("page does not contain %q", label)
		}
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("page did not escape product name: %s", body)
	}
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatal("page contains an executable script tag from product text")
	}
}

func TestPublicTracePageReturnsNotFoundForUnknownCode(t *testing.T) {
	router := routerWithDemoProduct(t)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/trace/SP-MISSING-001", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q, want HTML error", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Body.String(), "Không tìm thấy sản phẩm") {
		t.Fatalf("missing public not-found message: %s", response.Body.String())
	}
}

func routerWithDemoProduct(t *testing.T) http.Handler {
	t.Helper()
	config := testConfig(t)
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(validProduct()); err != nil {
		t.Fatal(err)
	}
	return NewRouter(config, store)
}
