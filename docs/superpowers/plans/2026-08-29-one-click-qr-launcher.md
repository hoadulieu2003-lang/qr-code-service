# One-Click QR Test Launcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a Windows operator double-click one batch file to start the local QR service when needed and open the local product form.

**Architecture:** `MO_TEST_QR.bat` stays at the repository root so `%~dp0` resolves the project directory regardless of the current shell folder. It gets a validated local HTTP port from a PowerShell helper, probes `/health`, and uses a temporary per-port startup lock so only one simultaneous click can open the existing foreground PowerShell start script. The lock records an owner PID and timestamp; a later launcher reclaims it only when the owner is gone after a short grace period or when it is older than the 30-second startup deadline. The launcher then opens the browser at `/admin` when health is ready.

**Tech Stack:** Windows CMD batch, PowerShell, existing `scripts/start-local.ps1` and Go QR service.

## Global Constraints

- Work only in `C:\Users\game\Documents\app\qr code\.worktrees\product-traceability-mvp` on `codex/product-traceability-mvp`.
- Do not change Go source, routes, `.env`, product data, or API authentication behavior.
- Do not print or persist `API_KEY`.
- Use a visible foreground PowerShell server window; never start a hidden/detached service.
- Accept `PORT` from `.env` with or without quotes; it must be an integer from `1` to `65535`, otherwise stop before composing a URL or command.
- This launcher supports local HTTP test mode only: `SSL` in `.env` must be false/off/0; HTTPS uses its separate configured workflow.
- Prefer `go` on PATH; when absent, use the existing Codex portable Go only if that executable is present.

---

### Task 1: Add and verify the one-click launcher

**Files:**
- Create: `MO_TEST_QR.bat`
- Create: `scripts/read-launch-config.ps1`
- Modify: `README.md`

**Interfaces:**
- Consumes: `.env` through `scripts/read-launch-config.ps1`, `scripts/start-local.ps1`, `GET /health`, and `GET /admin`.
- Produces: browser navigation to `http://localhost:<PORT>/admin`; visible server terminal only when health is not already `200`.

- [ ] **Step 1: Prove the launcher does not yet exist**

Run:

```powershell
if (Test-Path -LiteralPath '.\MO_TEST_QR.bat') {
  throw 'RED failed: launcher already exists.'
}
throw 'RED: one-click QR launcher is missing.'
```

Expected: the command fails with `RED: one-click QR launcher is missing.`

- [ ] **Step 2: Add the minimal batch launcher**

Create `scripts/read-launch-config.ps1` to parse `.env` entirely inside PowerShell and emit only a validated decimal port plus a boolean SSL flag. `MO_TEST_QR.bat` resolves its own folder, fails visibly if required files are missing, consumes only that validated output, rejects SSL mode, probes health, atomically acquires a temporary startup lock, and records the batch owner PID/timestamp. Other simultaneous clicks wait for the same health result; a subsequent click reclaims only a lock with a dead owner after five seconds or any lock older than 45 seconds. Use a 30-second deadline, then use `start` to open `/admin`. On failure it must pause with an operator-readable message.

- [ ] **Step 3: Check batch file contents and run the no-server flow**

Run:

```powershell
cmd /d /c .\MO_TEST_QR.bat
Invoke-WebRequest -UseBasicParsing http://localhost:18080/health
Invoke-WebRequest -UseBasicParsing http://localhost:18080/admin
```

Expected: batch opens one visible server window, `/health` is `200`, and `/admin` is `200`.

- [ ] **Step 4: Verify no duplicate server starts when already healthy or starting**

Run the batch a second time after health is ready, then repeat with two launchers started while health is unavailable. Confirm that the configured port has only one listener PID and `/admin` returns `200` in both cases.

- [ ] **Step 5: Add a README quick-launch reference**

Add one concise bullet under the existing handover entrypoint: double-click `MO_TEST_QR.bat` to launch/open the local form; close the separate PowerShell server window or press `Ctrl+C` there to stop it.

- [ ] **Step 6: Verify repository and source integrity**

Run:

```powershell
$go = 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
& $go test ./...
git diff --check
git status --short
git status --ignored --short
```

Expected: Go tests pass, no whitespace error, `MO_TEST_QR.bat`, `scripts/read-launch-config.ps1`, README and this plan are the only tracked changes, and `.env`, `data/`, `.exe` remain ignored.

- [ ] **Step 7: Commit the launcher**

```powershell
git add MO_TEST_QR.bat scripts/read-launch-config.ps1 README.md docs/superpowers/plans/2026-08-29-one-click-qr-launcher.md
git commit -m "feat: add one-click QR test launcher"
```
