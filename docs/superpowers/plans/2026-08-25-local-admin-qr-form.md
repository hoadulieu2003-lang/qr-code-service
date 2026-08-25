# Local QR Admin Form Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Add a local server-rendered /admin form that creates one traceability product and shows a QR phone cameras can scan, without changing the existing integration API.

**Architecture:** Keep Product, ProductStore, Config.TraceURL, and the JSON API as the source of truth. Extract one secure creation operation in handlers.go; the existing JSON endpoint and the new HTML form both call it. Render the administration page with html/template and embed an in-memory PNG QR as base64 so the browser never needs to call an authenticated image endpoint.

**Tech Stack:** Go 1.21.4, net/http, net/http/httptest, html/template, encoding/base64, github.com/go-chi/chi, and the existing github.com/skip2/go-qrcode dependency.

## Global Constraints

- Work only in C:\Users\game\Documents\app\qr code\.worktrees\product-traceability-mvp on branch codex/product-traceability-mvp; never modify master.
- Run Go commands with C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe.
- Preserve GET /, GET /health, POST /api/products, GET /api/products/{trace_code}/qr.png, and GET /trace/{trace_code}.
- The admin tool is local/internal only: no account system, product list, edit/delete, analytics, database, JavaScript frontend, cookie, or local storage.
- POST /admin/products accepts standard form data, compares the supplied key with crypto/subtle.ConstantTimeCompare, and never puts the key in a URL, response, cookie, log, or data/products.json.
- The result QR is 512 px with High recovery and encodes exactly Config.TraceURL(trace_code). It is an HTML base64 data URI, not a new public image endpoint.
- For phone acceptance, PUBLIC_BASE_URL is a shared-Wi-Fi LAN URL while the form is opened locally as http://localhost:<PORT>/admin.
- Do not commit .env, data/, QR files, executables, or actual product data.

---

## File structure

| File | Responsibility |
| --- | --- |
| handlers.go | Constant-time comparison and one error-classified product-creation operation shared by the JSON API and form. |
| handlers_test.go | Focused tests for the shared creation operation plus existing API regression tests. |
| admin_page.go | Escaped Go templates, form parsing, QR data-URI generation, GET and POST admin handlers. |
| admin_page_test.go | Form rendering, success, PNG embedding, invalid data, bad key, duplicate code, and invalid public URL tests. |
| app.go | Register GET /admin and POST /admin/products when ProductStore exists. |
| README.md | Browser-based local/phone test instructions. |

## Shared interfaces

Add these symbols in handlers.go before either transport uses them:

~~~go
var (
    ErrInvalidProduct            = errors.New("invalid product")
    ErrPublicTraceURLUnavailable = errors.New("public trace URL is unavailable")
)

type createProductResponse struct {
    TraceCode string `json:"trace_code"`
    TraceURL  string `json:"trace_url"`
    QRURL     string `json:"qr_url"`
}

func hasAPIKeyValue(provided, expected string) bool
func hasAPIKey(request *http.Request, expected string) bool
func createProduct(config Config, store *ProductStore, product Product) (createProductResponse, error)
~~~

hasAPIKey delegates to hasAPIKeyValue(request.Header.Get("X-API-Key"), expected). createProduct normalizes and validates the Product, validates the trace URL before writing, saves through ProductStore.Create, and returns the same trace_code, trace_url, and qr_url that the JSON API already returns. It wraps validation with ErrInvalidProduct and invalid/missing public URL with ErrPublicTraceURLUnavailable; ErrDuplicateTraceCode remains the duplicate signal.

admin_page.go defines:

~~~go
func adminFormHandler() http.HandlerFunc
func adminCreateProductHandler(config Config, store *ProductStore) http.HandlerFunc
~~~

The form names are the eight Product JSON field names plus api_key. Its view can contain Product, Error, TraceURL, and QRBase64 only; it must never contain an API-key field.

### Task 1: Extract the shared secure creation operation

**Files:**

- Modify: handlers.go
- Modify: handlers_test.go

**Interfaces:**

- Consumes: Product.normalized, Product.Validate, Config.TraceURL, ProductStore.Create, ErrDuplicateTraceCode, and crypto/subtle.ConstantTimeCompare.
- Produces: hasAPIKeyValue, a refactored hasAPIKey, and createProduct for both transports.

- [ ] **Step 1: Add failing shared-operation tests**

Append these tests to handlers_test.go:

~~~go
func TestCreateProductServiceCreatesTraceURLs(t *testing.T) {
    config := testConfig(t)
    store, err := NewProductStore(config.DataFile)
    if err != nil { t.Fatal(err) }

    result, err := createProduct(config, store, validProduct())
    if err != nil { t.Fatal(err) }
    if result.TraceURL != "http://192.168.1.20:18080/trace/SP-DEMO-001" {
        t.Fatalf("TraceURL = %q", result.TraceURL)
    }
    if result.QRURL != "http://192.168.1.20:18080/api/products/SP-DEMO-001/qr.png" {
        t.Fatalf("QRURL = %q", result.QRURL)
    }
}

func TestCreateProductServiceClassifiesFailures(t *testing.T) {
    config := testConfig(t)
    store, err := NewProductStore(config.DataFile)
    if err != nil { t.Fatal(err) }

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
~~~

- [ ] **Step 2: Confirm the focused tests fail first**

Run:

~~~powershell
$go = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
& $go test . -run 'TestCreateProductService' -v
~~~

Expected: FAIL because the service function and the two sentinel errors do not exist.

- [ ] **Step 3: Factor the constant-time key comparison**

Replace the current hasAPIKey body with this pair in handlers.go:

~~~go
func hasAPIKeyValue(provided, expected string) bool {
    if provided == "" || expected == "" || len(provided) != len(expected) {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func hasAPIKey(request *http.Request, expected string) bool {
    return hasAPIKeyValue(request.Header.Get("X-API-Key"), expected)
}
~~~

Keep all JSON handler header behavior unchanged.

- [ ] **Step 4: Implement createProduct with explicit error classes**

Move the product normalization, validation, trace URL creation, store write, and response construction from createProductHandler into createProduct. Use these mappings inside the helper:

~~~go
product = product.normalized()
if err := product.Validate(); err != nil {
    return createProductResponse{}, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
}
traceURL, err := config.TraceURL(product.TraceCode)
if err != nil {
    return createProductResponse{}, fmt.Errorf("%w: %v", ErrPublicTraceURLUnavailable, err)
}
if err := store.Create(product); err != nil {
    return createProductResponse{}, err
}
return createProductResponse{
    TraceCode: product.TraceCode,
    TraceURL: traceURL,
    QRURL: config.PublicBaseURL + "/api/products/" + product.TraceCode + "/qr.png",
}, nil
~~~

- [ ] **Step 5: Make the JSON endpoint call the helper**

Keep JSON decoding, Content-Type validation, and X-API-Key validation inside createProductHandler. Call createProduct and map errors in this order:

~~~go
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
~~~

On success, keep the current HTTP 201 JSON schema exactly.

- [ ] **Step 6: Verify service and JSON API regression**

Run:

~~~powershell
& $go test . -run 'TestCreateProductService|TestCreateProduct' -v
& $go test ./...
~~~

Expected: PASS. Existing tests continue to prove API responses 400, 401, 409, and 503.

- [ ] **Step 7: Commit this isolated refactor**

~~~powershell
git add handlers.go handlers_test.go
git commit -m "refactor: share product creation logic"
~~~

### Task 2: Add the escaped local form and embedded QR result

**Files:**

- Create: admin_page.go
- Create: admin_page_test.go
- Modify: app.go

**Interfaces:**

- Consumes: hasAPIKeyValue, createProduct, Config, ProductStore, Product, qrcode.Encode, and qrcode.High.
- Produces: GET /admin and POST /admin/products, both returning HTML with a scannable QR result on success.

- [ ] **Step 1: Add a reusable admin router test helper**

At the top of admin_page_test.go, define this helper so tests can inspect store state:

~~~go
func newAdminRouter(t *testing.T, baseURL string) (http.Handler, *ProductStore) {
    t.Helper()
    config := testConfig(t)
    config.PublicBaseURL = baseURL
    store, err := NewProductStore(config.DataFile)
    if err != nil { t.Fatal(err) }
    return NewRouter(config, store), store
}
~~~

- [ ] **Step 2: Write failing form-render and success tests**

Use newAdminRouter and add these tests:

~~~go
func TestAdminFormRendersAllInputs(t *testing.T) {
    router, _ := newAdminRouter(t, "http://192.168.1.20:18080")
    response := httptest.NewRecorder()
    router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin", nil))

    if response.Code != http.StatusOK { t.Fatalf("status = %d", response.Code) }
    body := response.Body.String()
    for _, name := range []string{
        "trace_code", "product_code", "name", "batch_code", "manufactured_at",
        "expires_at", "origin", "verification_status", "api_key",
    } {
        if !strings.Contains(body, `name="`+name+`"`) {
            t.Fatalf("missing input %q", name)
        }
    }
    if !strings.Contains(body, `name="api_key" type="password"`) {
        t.Fatal("API key is not a password input")
    }
}

func TestAdminCreatesProductAndEmbedsPNGQR(t *testing.T) {
    router, store := newAdminRouter(t, "http://192.168.1.20:18080")
    values := url.Values{
        "trace_code": {"SP-FORM-001"}, "product_code": {"FORM-001"},
        "name": {"Sản phẩm nhập form"}, "batch_code": {"LO-FORM-001"},
        "manufactured_at": {"2026-08-25"}, "expires_at": {"2027-08-25"},
        "origin": {"Đắk Lắk, Việt Nam"}, "verification_status": {"verified"},
        "api_key": {"admin-key"},
    }
    request := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(values.Encode()))
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    response := httptest.NewRecorder()
    router.ServeHTTP(response, request)

    if response.Code != http.StatusCreated { t.Fatalf("status = %d: %s", response.Code, response.Body.String()) }
    if _, err := store.Get("SP-FORM-001"); err != nil { t.Fatal(err) }
    if !strings.Contains(response.Body.String(), "http://192.168.1.20:18080/trace/SP-FORM-001") {
        t.Fatal("missing trace URL")
    }
    if !strings.Contains(response.Body.String(), "data:image/png;base64,") {
        t.Fatal("missing QR data URI")
    }
}
~~~

In the success test, extract the data after data:image/png;base64, through the next double quote, decode it with base64.StdEncoding.DecodeString, and assert bytes.HasPrefix(png, []byte{137, 80, 78, 71}).

- [ ] **Step 3: Write failing negative form tests**

Using the same valid url.Values, add four independent tests:

1. api_key wrong-key returns 401, response does not contain wrong-key, and store.Get("SP-FORM-001") returns ErrProductNotFound.
2. Blank origin returns 400, includes SP-FORM-001 in the HTML form, and creates no record.
3. Posting valid data twice returns 201 then 409 without overwriting the first record.
4. A router whose base URL is empty returns 503 and creates no record.

- [ ] **Step 4: Confirm admin tests fail before adding routes**

Run:

~~~powershell
& $go test . -run 'TestAdmin' -v
~~~

Expected: FAIL because the admin routes and handlers do not exist.

- [ ] **Step 5: Implement escaped templates and POST behavior**

Create admin_page.go with package-level template.Must templates from html/template. The form must include lang="vi", UTF-8, responsive viewport, labels for all Product fields, date inputs for manufactured_at/expires_at, a select restricted to verified/unverified, and a password input for api_key.

Define an adminView with Product, Error, TraceURL, and QRBase64 only. Implement a renderAdmin helper that always sets Content-Type: text/html; charset=utf-8 and calls the template. Form values are rendered through template fields so product values are HTML escaped. Do not bind the submitted api_key back into any view.

adminFormHandler renders a blank view with 200. adminCreateProductHandler calls request.ParseForm(), builds Product from the eight product fields, then calls hasAPIKeyValue(request.PostForm.Get("api_key"), config.APIKey). For a bad key, render the product fields plus an error with 401; for invalid, public URL, duplicate, or unknown errors, call createProduct and render respectively 400, 503, 409, or 500.

On success, encode result.TraceURL exactly once:

~~~go
png, err := qrcode.Encode(result.TraceURL, qrcode.High, 512)
if err != nil {
    renderAdmin(writer, http.StatusInternalServerError, adminView{
        Product: product, Error: "Không thể tạo mã QR. Hãy thử lại.",
    })
    return
}
renderAdmin(writer, http.StatusCreated, adminView{
    Product: product, TraceURL: result.TraceURL,
    QRBase64: base64.StdEncoding.EncodeToString(png),
})
~~~

The QR image element must use static data URI prefix plus the base64-only template field:

~~~html
<img src="data:image/png;base64,{{.QRBase64}}" alt="Mã QR truy xuất nguồn gốc">
~~~

- [ ] **Step 6: Register only the two new local routes**

In app.go, inside the existing if store != nil block, add exactly:

~~~go
router.Get("/admin", adminFormHandler())
router.Post("/admin/products", adminCreateProductHandler(config, store))
~~~

Do not change legacy, API, public trace, QR PNG, or health registrations. Admin routes remain absent for a legacy-only router with nil store.

- [ ] **Step 7: Verify all form and regression tests**

Run:

~~~powershell
& $go test . -run 'TestAdmin|TestCreateProduct|TestProductQR|TestPublicTracePage|TestLegacyQRCodeAndHealthRemainAvailable' -v
& $go test ./...
~~~

Expected: PASS. The focused output covers 200, 201, 400, 401, 409, and 503 admin behavior while the existing API suite stays green.

- [ ] **Step 8: Commit the local UI**

~~~powershell
git add admin_page.go admin_page_test.go app.go
git commit -m "feat: add local QR admin form"
~~~

### Task 3: Document and run local phone acceptance

**Files:**

- Modify: README.md

**Interfaces:**

- Consumes: GET /admin, POST /admin/products, PORT, API_KEY, and PUBLIC_BASE_URL.
- Produces: a repeatable browser-first test procedure and recorded evidence.

- [ ] **Step 1: Add the browser-first test guide**

Add a README subsection named Create a product from the local web form after the current LAN setup. Give these actions exactly:

1. Start the service with API_KEY and a shared-Wi-Fi IPv4 PUBLIC_BASE_URL.
2. On the service PC, open http://localhost:18080/admin, replacing 18080 when PORT differs.
3. Enter a fresh trace_code, all product fields, and the API key, then submit.
4. Scan the displayed QR with a same-Wi-Fi phone; it must open the LAN trace URL, not localhost.
5. Verify the six public fields; try a wrong key and repeated code to see 401 and 409.

State that the form never renders the submitted API key and that public deployment requires HTTPS plus authenticated administration.

- [ ] **Step 2: Run final automated verification**

Run:

~~~powershell
$go = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
& $go test ./...
& $go build -o bond-verify.exe .
git diff --check
~~~

Expected: tests exit 0, the build creates bond-verify.exe, and git diff --check has no output. The verification executable must be ignored or removed only if it is an ignored generated artifact.

- [ ] **Step 3: Start an instance and verify UI readiness**

Run in one PowerShell window:

~~~powershell
& $go build -o bond.exe .
.\bond.exe
~~~

Run in another:

~~~powershell
Invoke-WebRequest -UseBasicParsing http://localhost:18080/health
Start-Process 'http://localhost:18080/admin'
~~~

Expected: health is HTTP 200 and the PC browser shows a blank Vietnamese admin form.

- [ ] **Step 4: Execute the physical acceptance test**

Create a fresh product such as SP-PHONE-002 through the form and observe all four outcomes:

- The result page returns success, shows PNG QR, and reports a LAN trace URL ending in /trace/SP-PHONE-002.
- The phone scan opens that exact URL and displays name, product/batch code, manufacture date, expiry date, origin, and verification status.
- A wrong API key returns 401 and creates no record.
- Reposting SP-PHONE-002 returns 409 and preserves the first record.

Record health status, web status codes, independently decoded QR payload, and phone-scan confirmation in the handoff. Do not claim physical acceptance without the observed phone result.

- [ ] **Step 5: Commit the guide**

~~~powershell
git add README.md
git commit -m "docs: add local QR admin test guide"
~~~

## Plan self-review

| Specification requirement | Plan coverage |
| --- | --- |
| Server-rendered form with all Product fields and password key | Task 2, Steps 2 and 5 |
| JSON API and form share product creation behavior | Task 1, Steps 1-5; Task 2, Step 5 |
| Constant-time key comparison with no rendered/persisted key | Task 1, Step 3; Task 2, Steps 3 and 5 |
| 512 px High QR embeds the exact trace URL | Task 2, Steps 2 and 5 |
| Form 400, 401, 409, 503, and 500 handling | Task 2, Steps 3 and 5 |
| Existing API and public routes remain stable | Task 1, Step 6; Task 2, Steps 6-7 |
| Local browser plus shared-Wi-Fi phone acceptance | Task 3 |

Completeness review: every symbol consumed by a task appears in Shared interfaces or current source; the plan adds no account system, database, public admin route, or browser-side API call.
