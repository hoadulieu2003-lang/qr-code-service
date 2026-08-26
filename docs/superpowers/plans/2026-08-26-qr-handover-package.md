# QR Module Handover Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Create a clean handover package with runnable Windows scripts and detailed operating, API, and test guides.

**Architecture:** Preserve the flat Go root that currently builds. Add isolated PowerShell scripts under scripts, handover documents under docs/handover, and a compact README index. Instance files remain ignored.

**Tech Stack:** Go 1.21.4, PowerShell, Markdown, existing HTTP API.

## Global Constraints

- Work only in C:\Users\game\Documents\app\qr code\.worktrees\product-traceability-mvp on codex/product-traceability-mvp.
- Do not move Go source or change existing route behavior.
- Do not stage or print .env, API_KEY, data/products.json, bond.exe, or bond-verify.exe.
- The start script builds then runs in the foreground. It must not start a hidden or detached process.
- The local UI remains loopback-only and has no browser API key. Integration API requests retain X-API-Key.
- Use C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe for final verification.

---

## File structure

| File | Responsibility |
| --- | --- |
| scripts/start-local.ps1 | Validate .env and Go, build bond.exe, run in foreground. |
| scripts/check-health.ps1 | Check health and local form contract without secrets. |
| scripts/test-api.ps1 | Create a caller-provided trace code through the API and verify its public trace page. |
| docs/handover/01-quick-start.md | Configuration, foreground lifecycle, URLs, and Wi-Fi requirements. |
| docs/handover/02-api-integration.md | API contract, examples, response fields, and status codes. |
| docs/handover/03-test-guide.md | UI, phone, API, and ten-case test procedure. |
| docs/handover/04-handover-checklist.md | Operational acceptance and production boundary checklist. |
| README.md | Handover entrypoint and folder map. |

### Task 1: Add and exercise Windows operational scripts

**Files:**

- Create: scripts/start-local.ps1
- Create: scripts/check-health.ps1
- Create: scripts/test-api.ps1

**Interfaces:**

- Consumes: .env, Go from PATH, and the existing health/admin/product/trace routes.
- Produces: foreground start, safe health proof, and repeatable API caller proof.

- [ ] **Step 1: Confirm script paths do not exist**

Run:

~~~powershell
Test-Path scripts/start-local.ps1
Test-Path scripts/check-health.ps1
Test-Path scripts/test-api.ps1
~~~

Expected: False for every path.

- [ ] **Step 2: Implement start-local.ps1**

The script resolves the project root as the parent of its script folder. It refuses a missing .env and a missing Go command, builds bond.exe in root, prints that Ctrl+C stops the process, then invokes bond.exe directly.

~~~powershell
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
if (-not (Test-Path (Join-Path $root '.env'))) { throw 'Missing .env. Copy sample.env to .env first.' }
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) { throw 'Go was not found in PATH.' }
Set-Location $root
& $go.Source build -o bond.exe .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host 'Server runs in this window. Stop it with Ctrl+C.'
& (Join-Path $root 'bond.exe')
exit $LASTEXITCODE
~~~

- [ ] **Step 3: Implement check-health.ps1**

Add an integer Port parameter defaulting to 18080. It calls localhost health and admin, requires both 200, requires the trace_code input, rejects any api_key input, prints a safe PASS line, and otherwise exits 1.

~~~powershell
param([int]$Port = 18080)
$health = Invoke-WebRequest -UseBasicParsing "http://localhost:$Port/health" -TimeoutSec 5
$admin = Invoke-WebRequest -UseBasicParsing "http://localhost:$Port/admin" -TimeoutSec 5
if ($health.StatusCode -ne 200 -or $admin.StatusCode -ne 200) { throw 'Service health or admin check failed.' }
if ($admin.Content -notmatch 'name="trace_code"' -or $admin.Content -match 'name="api_key"') { throw 'Local admin contract failed.' }
Write-Host "PASS: health and local admin are ready on port $Port."
~~~

- [ ] **Step 4: Implement test-api.ps1**

Require TraceCode. Parse API_KEY and PUBLIC_BASE_URL from .env without printing their values. Construct all eight product fields, post JSON to PUBLIC_BASE_URL/api/products with X-API-Key, request the returned trace_url, and print only trace_code, trace_url, and qr_url. Let a duplicate request stop with its 409 response.

~~~powershell
param([Parameter(Mandatory)][string]$TraceCode)
$product = @{
  trace_code = $TraceCode; product_code = 'API-DEMO-001'; name = 'Sản phẩm API demo'
  batch_code = 'LO-API-001'; manufactured_at = '2026-08-26'; expires_at = '2027-08-26'
  origin = 'Đắk Lắk, Việt Nam'; verification_status = 'verified'
}
$response = Invoke-RestMethod -Method Post -Uri "$publicBase/api/products" -Headers @{ 'X-API-Key' = $apiKey } -ContentType 'application/json; charset=utf-8' -Body ($product | ConvertTo-Json)
$trace = Invoke-WebRequest -UseBasicParsing $response.trace_url -TimeoutSec 10
if ($trace.StatusCode -ne 200) { throw 'Public trace page was not reachable.' }
$response | Select-Object trace_code, trace_url, qr_url
~~~

- [ ] **Step 5: Verify the scripts fail safely and parse cleanly**

Run the start script from a temporary folder lacking .env and assert its missing-config message. Then parse all scripts:

~~~powershell
$tokens = $null; $errors = $null
Get-ChildItem scripts/*.ps1 | ForEach-Object {
  [System.Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$tokens, [ref]$errors) | Out-Null
}
if ($errors.Count -gt 0) { $errors | Format-List; exit 1 }
~~~

Expected: missing .env fails before build; no script parser errors.

- [ ] **Step 6: Verify against the running service**

Run:

~~~powershell
.scriptscheck-health.ps1 -Port 18080
.scripts	est-api.ps1 -TraceCode SP-HANDOVER-001
~~~

Expected: health script prints PASS; API script returns a new trace code, trace URL, and QR URL without an API key.

- [ ] **Step 7: Commit scripts**

~~~powershell
git add scripts/start-local.ps1 scripts/check-health.ps1 scripts/test-api.ps1
git commit -m "feat: add QR handover scripts"
~~~

### Task 2: Write handover documentation and README index

**Files:**

- Create: docs/handover/01-quick-start.md
- Create: docs/handover/02-api-integration.md
- Create: docs/handover/03-test-guide.md
- Create: docs/handover/04-handover-checklist.md
- Modify: README.md

**Interfaces:**

- Consumes: script names, existing configuration, routes, and verified ten-case runtime outcomes.
- Produces: a standalone path for operators, API callers, and acceptance testers.

- [ ] **Step 1: Write the quick-start guide**

Include prerequisite software, copying sample.env to .env, LAN IPv4 discovery, PUBLIC_BASE_URL explanation, foreground start script, check-health command, local UI URL, shutdown by Ctrl+C, and runtime-file exclusions.

- [ ] **Step 2: Write complete API integration guide**

Document Product JSON with eight fields; POST product creation; authenticated QR PNG retrieval; public trace; health; response fields; status 201, 400, 401, 404, 409, 500, 503; both PowerShell and curl. Explain callers persist the returned trace_url instead of assembling URLs.

- [ ] **Step 3: Write detailed test guide**

Document UI without an API key, correct date ordering, QR output, six trace fields, same-Wi-Fi phone scan, API tests, and all ten runtime scenarios. State server verification does not replace a real phone scan.

- [ ] **Step 4: Write handover checklist**

Use task checkboxes for tests/build, health, local form, LAN admin 403, API 401/201/409, QR plus public trace, physical phone scan, data backup, secret boundary, and HTTPS/session/CSRF/database requirements for public deployment.

- [ ] **Step 5: Add README handover section**

Link all four handover documents and three scripts. Mark docs/superpowers as engineering history and .env, data, and executables as per-instance runtime artifacts.

- [ ] **Step 6: Verify documentation links and boundaries**

Run:

~~~powershell
rg -n ']([^)]*)' README.md docs/handover
git status --ignored --short
git diff --check
~~~

Expected: all linked paths exist; .env, data, and executables remain ignored; no secret or runtime product becomes staged.

- [ ] **Step 7: Commit documentation**

~~~powershell
git add README.md docs/handover
git commit -m "docs: add QR module handover guide"
~~~

### Task 3: Run final handover acceptance

**Files:**

- Verify: scripts, docs/handover, README, and Go source.

**Interfaces:**

- Consumes: Tasks 1 and 2.
- Produces: verifiable handover evidence.

- [ ] **Step 1: Verify source**

~~~powershell
$go = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
& $go test ./...
& $go build -o bond-verify.exe .
~~~

Expected: both commands exit 0.

- [ ] **Step 2: Verify operations**

Run:

~~~powershell
.scriptscheck-health.ps1 -Port 18080
.scripts	est-api.ps1 -TraceCode SP-HANDOVER-002
~~~

Expected: both pass without printing the API key.

- [ ] **Step 3: Verify LAN administration boundary**

Run:

~~~powershell
try { Invoke-WebRequest -UseBasicParsing http://192.168.1.22:18080/admin }
catch { [int]$_.Exception.Response.StatusCode }
~~~

Expected: 403.

- [ ] **Step 4: Verify repository boundary**

Run:

~~~powershell
git diff --check
git status --short
git status --ignored --short
~~~

Expected: no whitespace errors, no tracked runtime artifacts, and only expected ignored .env/data/executable entries.

- [ ] **Step 5: Commit a needed final documentation correction only**

~~~powershell
git add README.md docs/handover scripts
git commit -m "docs: finalize QR handover package"
~~~

Make this commit only if final verification identifies and corrects a documentation or script issue.

## Plan self-review

| Requirement | Plan coverage |
| --- | --- |
| Preserve Go layout | Global constraints |
| Startup, health, and API scripts | Task 1 |
| Detailed quick-start, API, test, and handover docs | Task 2 |
| Secrets and runtime artifacts excluded | Global constraints, Task 2 Step 6, Task 3 Step 4 |
| Live API/UI/LAN evidence | Task 1 Step 6 and Task 3 |
| Complete handover package | All tasks |

Completeness review: every file, command, route, and secret boundary matches the approved handover spec.
