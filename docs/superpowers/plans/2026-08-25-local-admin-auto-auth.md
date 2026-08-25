# Local Admin Auto-Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Let a loopback-only local tester create and scan a QR from /admin without entering an API key, while preserving API-key authentication for integration endpoints.

**Architecture:** The existing loopback check remains the security boundary for /admin. The form handler bypasses only the browser-supplied key and calls the existing createProduct service directly; POST /api/products and the printable QR endpoint keep hasAPIKey unchanged. The template and README remove the local key workflow.

**Tech Stack:** Go 1.21.4, net/http/httptest, html/template, existing go-chi router and go-qrcode.

## Global Constraints

- Work in C:\Users\game\Documents\app\qr code\.worktrees\product-traceability-mvp on codex/product-traceability-mvp.
- Use C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe for verification.
- GET /admin and POST /admin/products remain loopback-only; LAN requests return 403 before reading form data.
- API_KEY remains non-empty at startup and POST /api/products plus GET /api/products/{trace_code}/qr.png still require X-API-Key.
- The local form accepts only the eight Product fields. It never renders, reads, persists, or logs api_key.
- Preserve product validation, 400/409/503/500 form errors, QR High/512 creation before persistence, and public trace behavior.
- Do not commit .env, data/, executable output, or product test data.

---

## File structure

| File | Responsibility |
| --- | --- |
| admin_page.go | Remove browser API-key input/check while retaining loopback enforcement and shared createProduct call. |
| admin_page_test.go | Establish the no-key local form contract and keep loopback/XSS/duplicate/QR tests real. |
| README.md | Give the fast local browser test without exposing or requesting the API key. |
| handlers.go and handlers_test.go | Do not change; their existing API-key checks prove the integration API remains protected. |

### Task 1: Remove key entry from the loopback-only form

**Files:**

- Modify: admin_page.go
- Modify: admin_page_test.go

**Interfaces:**

- Consumes: isLoopbackRequest, createProduct(Config, *ProductStore, Product), adminView, ProductStore, and hasAPIKey for API routes.
- Produces: a loopback-only form with eight Product inputs that succeeds without browser-provided credentials.

- [ ] **Step 1: Write failing form tests**

In admin_page_test.go, change validAdminForm so it contains only the eight Product fields. Replace TestAdminFormRendersProductInputsAndPasswordAPIKey with a test named TestAdminFormRendersOnlyProductInputs. It must assert each Product input exists and assert the response does not contain name="api_key" or type="password".

Add this independent behavior test:

~~~go
func TestAdminCreatesProductWithoutAPIKey(t *testing.T) {
    router, store := newAdminRouter(t, "http://192.168.1.20:18080")
    response := postAdminForm(router, validAdminForm())

    if response.Code != http.StatusCreated {
        t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
    }
    if _, err := store.Get("SP-FORM-001"); err != nil {
        t.Fatal(err)
    }
}
~~~

Replace TestAdminRejectsWrongKeyWithoutPersistingOrEchoingIt with TestAdminIgnoresUnexpectedAPIKeyFormField. Post validAdminForm with api_key set to ignored-local-value, expect 201, expect no ignored-local-value in the response, and assert the product is stored. This proves the form neither depends on nor reflects browser-provided key material.

Keep TestAdminRejectsNonLoopbackRequests and change its POST to omit api_key; it must still receive 403 and write no record.

- [ ] **Step 2: Run admin tests and verify the expected failure**

Run:

~~~powershell
$go = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
& $go test . -run 'TestAdmin' -v
~~~

Expected: the no-key success test fails with 401, and the form test still finds the obsolete API-key input.

- [ ] **Step 3: Remove only browser-form key handling**

In admin_page.go, delete the API-key label/input from adminTemplate. Do not remove hasAPIKeyValue from handlers.go because JSON API routes still use it.

In adminCreateProductHandler, keep the loopback check and form parsing, then build Product only from the eight product fields. Delete this browser-only branch:

~~~go
if !hasAPIKeyValue(request.PostForm.Get("api_key"), config.APIKey) {
    renderAdmin(writer, http.StatusUnauthorized, adminView{
        Product: product,
        Error:   "API key không đúng hoặc đang thiếu.",
    })
    return
}
~~~

Call createProduct directly after building Product. Keep error mapping for ErrInvalidProduct, ErrPublicTraceURLUnavailable, ErrDuplicateTraceCode, and unknown errors unchanged.

- [ ] **Step 4: Format and verify form plus API regressions**

Run:

~~~powershell
$gofmt = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\gofmt.exe'
& $gofmt -w admin_page.go admin_page_test.go
& $go test . -run 'TestAdmin|TestCreateProductRequiresAPIKey|TestProductQRRequiresAPIKeyAndReturnsPNG' -v
& $go test ./...
~~~

Expected: all admin tests pass without any form key; TestCreateProductRequiresAPIKey and TestProductQRRequiresAPIKeyAndReturnsPNG still pass with 401 for missing key.

- [ ] **Step 5: Commit the source and tests**

~~~powershell
git add admin_page.go admin_page_test.go
git commit -m "feat: auto-authenticate local QR admin"
~~~

### Task 2: Update the fast-test guide and restart the local service

**Files:**

- Modify: README.md

**Interfaces:**

- Consumes: the loopback-only no-key /admin form from Task 1 and existing PUBLIC_BASE_URL LAN configuration.
- Produces: a copyable local phone-test flow without any API key entry.

- [ ] **Step 1: Update the README local-form subsection**

In Create a product from the local web form, replace the instruction to enter API_KEY with: enter a fresh trace_code and all product fields, then select Tạo QR. Remove the local-form wrong-key check. State that API_KEY remains configured server-side for integration APIs but is intentionally not requested by the localhost-only test form.

- [ ] **Step 2: Run final automated verification**

Run:

~~~powershell
& $go test ./...
& $go build -o bond-verify.exe .
git diff --check
~~~

Expected: tests exit 0, the executable builds, and git diff --check prints no whitespace error.

- [ ] **Step 3: Restart and test the real local boundary**

Stop any prior bond-verify.exe process, start the rebuilt executable from the worktree, then run:

~~~powershell
Invoke-WebRequest -UseBasicParsing http://localhost:18080/health
Invoke-WebRequest -UseBasicParsing http://localhost:18080/admin
try {
  Invoke-WebRequest -UseBasicParsing http://192.168.1.22:18080/admin
} catch {
  [int]$_.Exception.Response.StatusCode
}
~~~

Expected: health 200, localhost form 200 without API-key input, and the LAN admin request 403.

- [ ] **Step 4: Finish the fast phone test**

On the service PC, open http://localhost:18080/admin. Enter a new product with expiry date after manufacture date, select Tạo QR, and scan its visible code with a same-Wi-Fi phone. Confirm it opens PUBLIC_BASE_URL/trace/{trace_code} and displays six trace fields. This is the only manual step; do not claim it until the user reports the scan.

- [ ] **Step 5: Commit the guide**

~~~powershell
git add README.md
git commit -m "docs: simplify local QR test"
~~~

## Plan self-review

| Specification requirement | Plan coverage |
| --- | --- |
| Local form has no API-key field or browser key dependency | Task 1, Steps 1-3 |
| Local form remains loopback-only | Task 1, Step 1; Task 2, Step 3 |
| Integration APIs keep key authentication | Task 1, Step 4 |
| Existing validation/QR/error behavior remains | Task 1, Steps 3-4 |
| Fast no-key local/phone instructions | Task 2, Steps 1 and 4 |

Completeness review: every referenced function exists in current source, no new service/database/account is introduced, and each changed behavior has an automated test before production code changes.
