package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

func newAdminRouter(t *testing.T, baseURL string) (http.Handler, *ProductStore) {
	t.Helper()
	config := testConfig(t)
	config.PublicBaseURL = baseURL
	store, err := NewProductStore(config.DataFile)
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(config, store), store
}

func validAdminForm() url.Values {
	return url.Values{
		"trace_code":          {"SP-FORM-001"},
		"product_code":        {"FORM-001"},
		"name":                {"Sản phẩm nhập form"},
		"batch_code":          {"LO-FORM-001"},
		"manufactured_at":     {"2026-08-25"},
		"expires_at":          {"2027-08-25"},
		"origin":              {"Đắk Lắk, Việt Nam"},
		"verification_status": {"verified"},
		"api_key":             {"admin-key"},
	}
}

func postAdminForm(router http.Handler, values url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// This test fails if a tester cannot enter every Product field through the
// local page or the administrator key would be exposed as ordinary text.
func TestAdminFormRendersProductInputsAndPasswordAPIKey(t *testing.T) {
	router, _ := newAdminRouter(t, "http://192.168.1.20:18080")
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, name := range []string{
		"trace_code", "product_code", "name", "batch_code", "manufactured_at",
		"expires_at", "origin", "verification_status",
	} {
		if !strings.Contains(body, `name="`+name+`"`) {
			t.Fatalf("form is missing product input %q", name)
		}
	}
	if !strings.Contains(body, `<input type="password" name="api_key"`) {
		t.Fatalf("form does not render API key as a password input: %s", body)
	}
}

// This test fails if a valid form submission does not persist the product or
// does not return a real PNG QR that points to the public trace URL.
func TestAdminCreatesProductAndEmbedsPNGQR(t *testing.T) {
	router, store := newAdminRouter(t, "http://192.168.1.20:18080")
	response := postAdminForm(router, validAdminForm())

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if _, err := store.Get("SP-FORM-001"); err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	const prefix = `data:image/png;base64,`
	start := strings.Index(body, prefix)
	if start == -1 {
		t.Fatalf("response does not contain a QR data URI: %s", body)
	}
	encoded := body[start+len(prefix):]
	end := strings.Index(encoded, `"`)
	if end == -1 {
		t.Fatalf("QR data URI is not closed: %s", body)
	}
	png, err := base64.StdEncoding.DecodeString(html.UnescapeString(encoded[:end]))
	if err != nil {
		t.Fatalf("decode QR data URI: %v", err)
	}
	if !bytes.HasPrefix(png, []byte{137, 80, 78, 71}) {
		t.Fatal("QR data URI does not contain PNG bytes")
	}
	wantPNG, err := qrcode.Encode("http://192.168.1.20:18080/trace/SP-FORM-001", qrcode.High, 512)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(png, wantPNG) {
		t.Fatal("QR data URI is not the expected High-recovery 512px trace QR")
	}
	if !strings.Contains(body, "http://192.168.1.20:18080/trace/SP-FORM-001") {
		t.Fatalf("response does not show the public trace URL: %s", body)
	}
}

// This test fails if a rejected key can create a public trace record or if
// the local page reflects the secret back into its HTML response.
func TestAdminRejectsWrongKeyWithoutPersistingOrEchoingIt(t *testing.T) {
	router, store := newAdminRouter(t, "http://192.168.1.20:18080")
	values := validAdminForm()
	values.Set("api_key", "wrong-key")
	response := postAdminForm(router, values)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if strings.Contains(response.Body.String(), "wrong-key") {
		t.Fatal("response echoes the submitted API key")
	}
	if _, err := store.Get("SP-FORM-001"); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("store.Get() error = %v, want ErrProductNotFound", err)
	}
}

// This test fails if invalid form data is saved or product text can return as
// executable markup while the user is correcting a form error.
func TestAdminRejectsInvalidFormWithoutPersistingAndEscapesInput(t *testing.T) {
	router, store := newAdminRouter(t, "http://192.168.1.20:18080")
	values := validAdminForm()
	values.Set("name", "<script>alert(1)</script>")
	values.Set("origin", "")
	response := postAdminForm(router, values)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	body := response.Body.String()
	if !strings.Contains(body, "SP-FORM-001") {
		t.Fatalf("response did not preserve the trace code: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") || strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("response did not safely escape submitted product text: %s", body)
	}
	if _, err := store.Get("SP-FORM-001"); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("store.Get() error = %v, want ErrProductNotFound", err)
	}
}

// This test fails if a form can overwrite a trace record already created by
// the same local tester.
func TestAdminRejectsDuplicateTraceCode(t *testing.T) {
	router, store := newAdminRouter(t, "http://192.168.1.20:18080")
	if response := postAdminForm(router, validAdminForm()); response.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d", response.Code, http.StatusCreated)
	}
	second := validAdminForm()
	second.Set("name", "Sản phẩm không được ghi đè")
	if response := postAdminForm(router, second); response.Code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", response.Code, http.StatusConflict)
	}
	stored, err := store.Get("SP-FORM-001")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "Sản phẩm nhập form" {
		t.Fatalf("stored name = %q, want first product name", stored.Name)
	}
}

// This test fails if the form stores a product despite having no LAN URL that
// a phone can open after scanning the QR code.
func TestAdminRejectsUnavailablePublicTraceURL(t *testing.T) {
	router, store := newAdminRouter(t, "")
	response := postAdminForm(router, validAdminForm())

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if _, err := store.Get("SP-FORM-001"); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("store.Get() error = %v, want ErrProductNotFound", err)
	}
}

// This test fails if a LAN client can access the local-only administration
// form or save a product after observing the unauthenticated form page.
func TestAdminRejectsNonLoopbackRequests(t *testing.T) {
	router, store := newAdminRouter(t, "http://192.168.1.20:18080")
	getRequest := httptest.NewRequest(http.MethodGet, "/admin", nil)
	getRequest.RemoteAddr = "192.168.1.50:12345"
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusForbidden {
		t.Fatalf("GET status = %d, want %d", getResponse.Code, http.StatusForbidden)
	}

	values := validAdminForm()
	postRequest := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(values.Encode()))
	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRequest.RemoteAddr = "192.168.1.50:12345"
	postResponse := httptest.NewRecorder()
	router.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusForbidden {
		t.Fatalf("POST status = %d, want %d", postResponse.Code, http.StatusForbidden)
	}
	if _, err := store.Get("SP-FORM-001"); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("store.Get() error = %v, want ErrProductNotFound", err)
	}
}
