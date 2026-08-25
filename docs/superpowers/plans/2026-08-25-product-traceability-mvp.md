# Product Traceability QR MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mở rộng Bond thành một API tạo sản phẩm truy xuất nguồn gốc, sinh QR PNG và trang công khai để điện thoại quét xem thông tin sản phẩm.

**Architecture:** Giữ Bond làm QR encoder và endpoint legacy `GET /`. Tạo một router có thể khởi tạo trong test, một `ProductStore` JSON có khóa ghi và ghi nguyên tử, cùng các handler tách riêng cho API quản trị và trang truy xuất công khai. QR chỉ encode `PUBLIC_BASE_URL/trace/{trace_code}`.

**Tech Stack:** Go 1.21.4, `github.com/go-chi/chi` v1.5.5, `github.com/skip2/go-qrcode`, standard library `encoding/json`, `html/template`, `net/http/httptest`.

## Global Constraints

- Chạy và test bằng `C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe` trên Windows.
- Giữ tương thích `GET /?secret=<SECRET>&size=<px>&content=<text>` và `GET /health` của Bond.
- API quản trị nhận `X-API-Key`, so sánh bằng `crypto/subtle.ConstantTimeCompare`, không dùng query string cho khóa.
- Product MVP lưu tại `data/products.json`; không thêm database, analytics, tài khoản hay endpoint update/delete.
- QR dùng error-correction `High`, kích thước 512 px, chỉ encode URL công khai từ `PUBLIC_BASE_URL`.
- Trang `/trace/{trace_code}` hiển thị sáu dòng: tên, mã sản phẩm/lô, ngày sản xuất, hạn dùng, nơi sản xuất, trạng thái xác thực.
- Không commit `.env`, `bond.exe` hoặc dữ liệu sản phẩm thực. Thêm `data/` vào `.gitignore`.

---

## File structure

| File | Responsibility |
| --- | --- |
| `main.go` | Load `.env`, validate startup config, create store/router, start HTTP server and healthcheck mode. |
| `config.go` | `Config`, environment parsing, public URL validation and `TraceURL`. |
| `app.go` | `NewRouter`, health route, legacy Bond QR route and route registration. |
| `product.go` | `Product`, public validation, date rules and trace-code validation. |
| `store.go` | In-process synchronization and atomic JSON persistence. |
| `handlers.go` | Authenticated product creation and authenticated QR PNG handler. |
| `trace_page.go` | Escaped, responsive public HTML trace page and public 404 page. |
| `config_test.go` | Config and legacy/health router regression tests. |
| `product_test.go` | Product validation tests. |
| `store_test.go` | JSON persistence, duplicate and restart tests. |
| `handlers_test.go` | API status, auth, QR PNG and public-page tests. |
| `sample.env` | Safe configuration template for a new local instance. |
| `README.md` | Product API and LAN phone-test instructions. |
| `.gitignore` | Exclude runtime product data. |

## Shared interfaces

```go
type Config struct {
    Port          string
    Secret        string
    APIKey        string
    PublicBaseURL string
    DataFile      string
    MaxSize       int
    RecoveryLevel qrcode.RecoveryLevel
    EnableLogs    bool
}

type Product struct {
    TraceCode          string `json:"trace_code"`
    ProductCode        string `json:"product_code"`
    Name               string `json:"name"`
    BatchCode          string `json:"batch_code"`
    ManufacturedAt     string `json:"manufactured_at"`
    ExpiresAt          string `json:"expires_at"`
    Origin             string `json:"origin"`
    VerificationStatus string `json:"verification_status"`
}

type ProductStore struct { /* unexported fields */ }

var ErrDuplicateTraceCode = errors.New("trace code already exists")
var ErrProductNotFound = errors.New("product not found")

func LoadConfig() (Config, error)
func (c Config) TraceURL(traceCode string) (string, error)
func (p Product) Validate() error
func NewProductStore(path string) (*ProductStore, error)
func (s *ProductStore) Create(product Product) error
func (s *ProductStore) Get(traceCode string) (Product, error)
func NewRouter(config Config, store *ProductStore) http.Handler
```

### Task 1: Extract startup configuration and preserve existing Bond routes

**Files:**

- Create: `config.go`
- Create: `app.go`
- Create: `config_test.go`
- Modify: `main.go`
- Modify: `sample.env`

**Interfaces:**

- Consumes: existing `SECRET`, `PORT`, `MAX_SIZE`, `RECOVERY_LEVEL`, `ENABLE_LOGS` environment variables and existing `skip2/go-qrcode` dependency.
- Produces: `Config`, `LoadConfig()`, `Config.TraceURL()`, and `NewRouter(Config, *ProductStore) http.Handler` for all later handlers and tests.

- [ ] **Step 1: Write failing tests for configuration and existing routes**

```go
func TestLoadConfigRequiresAPIKey(t *testing.T) {
    t.Setenv("API_KEY", "")
    _, err := LoadConfig()
    if err == nil || !strings.Contains(err.Error(), "API_KEY") {
        t.Fatalf("LoadConfig() error = %v, want API_KEY error", err)
    }
}

func TestLegacyQRCodeAndHealthRemainAvailable(t *testing.T) {
    cfg := testConfig(t)
    router := NewRouter(cfg, nil)

    health := httptest.NewRecorder()
    router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
    if health.Code != http.StatusOK { t.Fatalf("health = %d", health.Code) }

    qr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/?secret=legacy-secret&size=128&content=test", nil)
    router.ServeHTTP(qr, req)
    if qr.Code != http.StatusOK || !bytes.HasPrefix(qr.Body.Bytes(), []byte{137, 80, 78, 71}) {
        t.Fatalf("legacy QR response = %d, body is not PNG", qr.Code)
    }
}
```

- [ ] **Step 2: Run the focused tests and confirm failure**

Run:

```powershell
$go = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
& $go test . -run 'TestLoadConfigRequiresAPIKey|TestLegacyQRCodeAndHealthRemainAvailable' -v
```

Expected: FAIL because `LoadConfig`, `NewRouter`, and `testConfig` do not exist.

- [ ] **Step 3: Implement config parsing and router extraction**

Implement `Config` with strict parsing of `PORT`, `MAX_SIZE`, recovery level, and boolean logs. Require a non-empty `API_KEY`; allow empty `PUBLIC_BASE_URL` at startup because only product creation needs it. Validate `PUBLIC_BASE_URL` in `TraceURL` with `url.ParseRequestURI`, require `http` or `https`, host, and no trailing slash.

Move the current `/health` and legacy `/` behavior into `NewRouter`. Retain `secret`, `size`, and `content` semantics and set `Content-Type: image/png` before writing the PNG body. Keep `main.go` as only environment loading, `LoadConfig`, `NewProductStore`, router creation, and `http.ListenAndServe`.

Use this exact helper in `config_test.go`:

```go
func testConfig(t *testing.T) Config {
    t.Helper()
    return Config{
        Port: "18080", Secret: "legacy-secret", APIKey: "admin-key",
        PublicBaseURL: "http://192.168.1.20:18080", DataFile: t.TempDir() + "/products.json",
        MaxSize: 1024, RecoveryLevel: qrcode.High,
    }
}
```

- [ ] **Step 4: Update the safe environment template**

Set every required key in `sample.env` while keeping values non-secret:

```dotenv
PORT=18080
SSL="FALSE"
SECRET="replace-with-legacy-qr-secret"
API_KEY="replace-with-random-admin-api-key"
PUBLIC_BASE_URL="http://192.168.1.20:18080"
DATA_FILE="data/products.json"
MAX_SIZE=1024
RECOVERY_LEVEL="High"
ENABLE_LOGS="TRUE"
```

- [ ] **Step 5: Run regression tests and the full baseline**

Run:

```powershell
& $go test . -run 'TestLoadConfigRequiresAPIKey|TestLegacyQRCodeAndHealthRemainAvailable' -v
& $go test ./...
```

Expected: both commands PASS.

- [ ] **Step 6: Commit the foundation**

```powershell
git add main.go config.go app.go config_test.go sample.env
git commit -m "refactor: extract Bond configuration and router"
```

### Task 2: Add validated product domain model and atomic JSON store

**Files:**

- Create: `product.go`
- Create: `product_test.go`
- Create: `store.go`
- Create: `store_test.go`
- Modify: `.gitignore`

**Interfaces:**

- Consumes: `Product`, `ErrDuplicateTraceCode`, `ErrProductNotFound`, `ProductStore` declarations in the shared interfaces.
- Produces: `Product.Validate`, `NewProductStore`, `ProductStore.Create`, and `ProductStore.Get` for HTTP handlers.

- [ ] **Step 1: Write failing validation and persistence tests**

```go
func TestProductValidateRejectsInvalidDatesAndTraceCode(t *testing.T) {
    product := validProduct()
    product.TraceCode = "bad code!"
    if err := product.Validate(); err == nil { t.Fatal("expected trace-code error") }

    product = validProduct()
    product.ExpiresAt = "2026-08-24"
    if err := product.Validate(); err == nil { t.Fatal("expected expiry-date error") }
}

func TestProductStorePersistsAndRejectsDuplicate(t *testing.T) {
    path := filepath.Join(t.TempDir(), "products.json")
    store, err := NewProductStore(path)
    if err != nil { t.Fatal(err) }
    if err := store.Create(validProduct()); err != nil { t.Fatal(err) }
    if err := store.Create(validProduct()); !errors.Is(err, ErrDuplicateTraceCode) { t.Fatalf("duplicate = %v", err) }

    reloaded, err := NewProductStore(path)
    if err != nil { t.Fatal(err) }
    got, err := reloaded.Get("SP-DEMO-001")
    if err != nil || got.Name != "Cà phê rang xay Demo" { t.Fatalf("Get = %#v, %v", got, err) }
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run:

```powershell
& $go test . -run 'TestProductValidateRejectsInvalidDatesAndTraceCode|TestProductStorePersistsAndRejectsDuplicate' -v
```

Expected: FAIL because `Product`, `validProduct`, `NewProductStore`, and store errors do not exist.

- [ ] **Step 3: Implement `Product.Validate`**

Implement `Product` with the exact JSON tags in the shared interface. Trim every string before storing. Accept trace codes only with `^[A-Za-z0-9_-]+$`; require non-empty name, product code, batch code and origin; parse both dates using `time.DateOnly`; reject expiration before manufacture; accept only `verified` and `unverified`.

Define this reusable test fixture in `product_test.go`:

```go
func validProduct() Product {
    return Product{
        TraceCode: "SP-DEMO-001", ProductCode: "SP-001", Name: "Cà phê rang xay Demo",
        BatchCode: "LO-2026-001", ManufacturedAt: "2026-08-25", ExpiresAt: "2027-08-25",
        Origin: "Đắk Lắk, Việt Nam", VerificationStatus: "verified",
    }
}
```

- [ ] **Step 4: Implement `ProductStore` with atomic writes**

Store products in a `map[string]Product` protected by `sync.RWMutex`. `NewProductStore` creates the parent directory, reads an absent file as an empty store, and returns an error for malformed JSON. `Create` calls `Validate`, returns `ErrDuplicateTraceCode` for an existing code, writes a JSON array sorted by `TraceCode` to an `os.CreateTemp` file in the data directory, closes it, then renames it over `DATA_FILE`. `Get` returns `ErrProductNotFound` for missing codes.

- [ ] **Step 5: Ignore instance data and run focused tests**

Append this exact line to `.gitignore`:

```gitignore
data/
```

Run:

```powershell
& $go test . -run 'TestProductValidateRejectsInvalidDatesAndTraceCode|TestProductStorePersistsAndRejectsDuplicate' -v
& $go test ./...
```

Expected: PASS. Confirm the temporary test data is not written into the repository root.

- [ ] **Step 6: Commit model and persistence**

```powershell
git add product.go product_test.go store.go store_test.go .gitignore
git commit -m "feat: add validated product traceability store"
```

### Task 3: Implement authenticated product creation API

**Files:**

- Create: `handlers.go`
- Create: `handlers_test.go`
- Modify: `app.go`

**Interfaces:**

- Consumes: `Config.TraceURL`, `ProductStore.Create`, `Product`, `ErrDuplicateTraceCode` and `NewRouter`.
- Produces: `POST /api/products` returning `trace_code`, `trace_url`, and `qr_url` for Task 4 and external callers.

- [ ] **Step 1: Write failing handler tests for authorization and creation**

```go
func TestCreateProductRequiresAPIKey(t *testing.T) {
    router := newTestRouter(t, "http://192.168.1.20:18080")
    req := httptest.NewRequest(http.MethodPost, "/api/products", strings.NewReader(`{}`))
    response := httptest.NewRecorder()
    router.ServeHTTP(response, req)
    if response.Code != http.StatusUnauthorized { t.Fatalf("status = %d", response.Code) }
}

func TestCreateProductReturnsTraceAndQRURLs(t *testing.T) {
    router := newTestRouter(t, "http://192.168.1.20:18080")
    body, _ := json.Marshal(validProduct())
    req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(body))
    req.Header.Set("X-API-Key", "admin-key")
    response := httptest.NewRecorder()
    router.ServeHTTP(response, req)
    if response.Code != http.StatusCreated { t.Fatalf("status = %d, body = %s", response.Code, response.Body.String()) }
    if !strings.Contains(response.Body.String(), "/trace/SP-DEMO-001") { t.Fatal("missing trace URL") }
}
```

- [ ] **Step 2: Run creation tests and confirm failure**

Run:

```powershell
& $go test . -run 'TestCreateProductRequiresAPIKey|TestCreateProductReturnsTraceAndQRURLs' -v
```

Expected: FAIL because `POST /api/products` is not registered.

- [ ] **Step 3: Add authentication and `POST /api/products`**

Add `hasAPIKey(r *http.Request, expected string) bool` using `subtle.ConstantTimeCompare`. It returns false for empty, different-length, or nonmatching key.

Add a handler that requires `Content-Type` beginning with `application/json`, decodes exactly one JSON object with `json.Decoder.DisallowUnknownFields`, invokes `store.Create`, and emits:

```go
type createProductResponse struct {
    TraceCode string `json:"trace_code"`
    TraceURL  string `json:"trace_url"`
    QRURL     string `json:"qr_url"`
}
```

Map errors exactly: unauthorized key to `401`, malformed or invalid input to `400`, duplicate to `409`, and unavailable/invalid `TraceURL` configuration to `503`. Set `Content-Type: application/json; charset=utf-8` before `json.NewEncoder(w).Encode`.

- [ ] **Step 4: Add edge-case assertions**

Extend `handlers_test.go` with requests that assert `400` for an unknown JSON key, `409` for a second `SP-DEMO-001`, and `503` when `newTestRouter` receives an empty public base URL.

- [ ] **Step 5: Run handler and full tests**

Run:

```powershell
& $go test . -run 'TestCreateProduct' -v
& $go test ./...
```

Expected: PASS, including all four status-code cases.

- [ ] **Step 6: Commit product API**

```powershell
git add app.go handlers.go handlers_test.go
git commit -m "feat: add authenticated product creation API"
```

### Task 4: Generate printable QR PNG and render the public trace page

**Files:**

- Create: `trace_page.go`
- Modify: `handlers.go`
- Modify: `app.go`
- Modify: `handlers_test.go`

**Interfaces:**

- Consumes: `ProductStore.Get`, `Config.TraceURL`, `Product`, `hasAPIKey`, and the `qrcode.High` encoder behavior from Task 1.
- Produces: authenticated `GET /api/products/{trace_code}/qr.png` and public `GET /trace/{trace_code}`.

- [ ] **Step 1: Write failing QR and public-page tests**

```go
func TestProductQRRequiresKeyAndReturnsPNG(t *testing.T) {
    router := routerWithDemoProduct(t)
    denied := httptest.NewRecorder()
    router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/products/SP-DEMO-001/qr.png", nil))
    if denied.Code != http.StatusUnauthorized { t.Fatalf("denied = %d", denied.Code) }

    req := httptest.NewRequest(http.MethodGet, "/api/products/SP-DEMO-001/qr.png", nil)
    req.Header.Set("X-API-Key", "admin-key")
    ok := httptest.NewRecorder()
    router.ServeHTTP(ok, req)
    if ok.Code != http.StatusOK || ok.Header().Get("Content-Type") != "image/png" || !bytes.HasPrefix(ok.Body.Bytes(), []byte{137, 80, 78, 71}) {
        t.Fatalf("QR response invalid: %d %q", ok.Code, ok.Header().Get("Content-Type"))
    }
}

func TestPublicTracePageEscapesProductFields(t *testing.T) {
    router := routerWithDemoProduct(t)
    req := httptest.NewRequest(http.MethodGet, "/trace/SP-DEMO-001", nil)
    response := httptest.NewRecorder()
    router.ServeHTTP(response, req)
    if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Cà phê rang xay Demo") {
        t.Fatalf("trace page = %d, %s", response.Code, response.Body.String())
    }
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run:

```powershell
& $go test . -run 'TestProductQRRequiresKeyAndReturnsPNG|TestPublicTracePageEscapesProductFields' -v
```

Expected: FAIL because neither route is registered.

- [ ] **Step 3: Implement `GET /api/products/{trace_code}/qr.png`**

Require `X-API-Key`, load product through `store.Get`, derive the exact payload using `config.TraceURL(product.TraceCode)`, then call `qrcode.Encode(payload, qrcode.High, 512)`. Set `Content-Type: image/png` before body write. Return `401` for bad key and `404` for an absent product.

- [ ] **Step 4: Implement the responsive public page**

Use a package-level `template.Must(template.New("trace").Parse(...))` in `trace_page.go`. The template must use only `{{.Field}}` template expressions so Go performs HTML escaping. Render these exact labels in this order: `Tên sản phẩm`, `Mã sản phẩm / lô`, `Ngày sản xuất`, `Hạn dùng`, `Nơi sản xuất`, `Trạng thái xác thực`.

Build the combined displayed code before rendering:

```go
type traceView struct {
    Name, ProductAndBatch, ManufacturedAt, ExpiresAt, Origin, VerificationStatus string
}
```

Return `Content-Type: text/html; charset=utf-8`; return a simple HTML `404` page for a missing trace code. Do not render the API key, JSON path, error strings, or any request headers.

- [ ] **Step 5: Add 404 and escaping regression tests**

Add a missing code request and assert `404`. Add a product name `"<script>alert(1)</script>"`, create it in the test store, request its trace page, then assert the response contains `&lt;script&gt;` and does not contain a literal executable script tag.

- [ ] **Step 6: Run tests and manually verify QR payload**

Run:

```powershell
& $go test . -run 'TestProductQRRequiresKeyAndReturnsPNG|TestPublicTracePageEscapesProductFields' -v
& $go test ./...
```

Then start the service using a LAN `PUBLIC_BASE_URL`, create `SP-DEMO-001`, download its PNG, and use the existing OpenCV temporary decoder command to assert that the QR payload exactly equals `PUBLIC_BASE_URL + "/trace/SP-DEMO-001"`.

Expected: all Go tests PASS; decoder reports an exact payload match.

- [ ] **Step 7: Commit QR and public trace page**

```powershell
git add app.go handlers.go handlers_test.go trace_page.go
git commit -m "feat: add QR trace page and printable product code"
```

### Task 5: Document local LAN setup and execute the phone acceptance test

**Files:**

- Modify: `README.md`
- Modify: `sample.env`

**Interfaces:**

- Consumes: all routes and configuration implemented in Tasks 1–4.
- Produces: copyable local startup, product creation and QR download instructions for a developer with no prior repository context.

- [ ] **Step 1: Write the README acceptance section**

Add a `## Product traceability MVP` section with this sequence: identify IPv4 LAN address by `Get-NetIPAddress -AddressFamily IPv4`; set `PUBLIC_BASE_URL` to that address and port; start `bond.exe`; create `SP-DEMO-001` via `POST /api/products`; download QR with API key; open the PNG on a screen; scan from a phone on the same Wi-Fi; verify the six public rows.

Use this exact product request example, replacing only `$apiKey` and `$baseUrl` from `.env`:

```powershell
$product = @{
  trace_code = 'SP-DEMO-001'; product_code = 'SP-001'; name = 'Cà phê rang xay Demo'
  batch_code = 'LO-2026-001'; manufactured_at = '2026-08-25'; expires_at = '2027-08-25'
  origin = 'Đắk Lắk, Việt Nam'; verification_status = 'verified'
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri "$baseUrl/api/products" -Headers @{ 'X-API-Key' = $apiKey } -ContentType 'application/json' -Body $product
```

- [ ] **Step 2: Ensure the sample configuration explains LAN use**

Keep `PUBLIC_BASE_URL` as the explicit LAN-IP example in `sample.env`; add one adjacent comment stating that `127.0.0.1` is not valid for a QR scanned by another device.

- [ ] **Step 3: Run final automated verification**

Run:

```powershell
& $go test ./...
git diff --check
```

Expected: test command exits `0`; `git diff --check` prints no whitespace errors.

- [ ] **Step 4: Run the physical acceptance test**

Verify all six outcomes from the spec: `201` product creation, QR PNG response, decoded payload equals public trace URL, phone scan opens the page, page shows six correct fields and `verified`, and negative checks return `401`, `409`, and `404`.

- [ ] **Step 5: Commit documentation**

```powershell
git add README.md sample.env
git commit -m "docs: add QR traceability LAN test guide"
```

## Plan self-review

| Specification requirement | Plan coverage |
| --- | --- |
| JSON product persistence without database | Task 2 |
| Exact product fields and validation | Task 2 |
| `POST /api/products` with `X-API-Key` | Task 3 |
| `GET /api/products/{trace_code}/qr.png` | Task 4 |
| Public `/trace/{trace_code}` mobile page with six rows | Task 4 |
| Constant-time key check, escaped HTML and no secret logging | Tasks 3–4 |
| LAN URL, QR decoding and phone scan acceptance | Task 5 |
| Existing Bond QR and health endpoints remain usable | Task 1 |

Completeness scan result: no deferred work markers or undefined handoff references. Shared types and function signatures are defined before the tasks that consume them.
