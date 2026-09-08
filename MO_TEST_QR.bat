@echo off
setlocal EnableExtensions DisableDelayedExpansion

set "ROOT=%~dp0"
set "ENV_FILE=%ROOT%.env"
set "START_SCRIPT=%ROOT%scripts\start-local.ps1"
set "CONFIG_SCRIPT=%ROOT%scripts\read-launch-config.ps1"
set "SERVER_LAUNCH_SCRIPT=%ROOT%scripts\launch-server-window.ps1"
set "PORT="
set "SSL_ENABLED="
set "PORTABLE_GO=C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe"

if not exist "%ENV_FILE%" (
    echo [LOI] Khong tim thay .env trong thu muc module.
    echo Hay copy sample.env thanh .env va cau hinh truoc khi test.
    pause
    exit /b 1
)

if not exist "%START_SCRIPT%" (
    echo [LOI] Khong tim thay scripts\start-local.ps1.
    pause
    exit /b 1
)

if not exist "%CONFIG_SCRIPT%" (
    echo [LOI] Khong tim thay scripts\read-launch-config.ps1.
    pause
    exit /b 1
)

if not exist "%SERVER_LAUNCH_SCRIPT%" (
    echo [LOI] Khong tim thay scripts\launch-server-window.ps1.
    pause
    exit /b 1
)

for /f "tokens=1,2,3 delims=|" %%A in ('powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%CONFIG_SCRIPT%" -EnvPath "%ENV_FILE%" 2^>nul') do (
    if "%%A"=="VALID" (
        set "PORT=%%B"
        set "SSL_ENABLED=%%C"
    )
)
if not defined PORT goto invalid_config
if not defined SSL_ENABLED goto invalid_config

if /I "%SSL_ENABLED%"=="TRUE" goto ssl_not_supported

call :service_is_ready
if not errorlevel 1 goto open_admin

call :set_deadline
set "STARTER=0"
set "LOCK_DIR=%TEMP%\bond-qr-start-%PORT%.lock"
call :try_acquire_start_lock
if errorlevel 1 goto wait_for_service

:start_server
echo Dang khoi dong QR server trong cua so PowerShell rieng...
where.exe go >nul 2>&1
if not errorlevel 1 goto launch_server_from_path
if exist "%PORTABLE_GO%" goto launch_server_with_portable_go

:launch_server_from_path
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SERVER_LAUNCH_SCRIPT%" -StartScript "%START_SCRIPT%" -LockDir "%LOCK_DIR%" >nul 2>&1
if errorlevel 1 goto start_failed
set "LAUNCHED_SERVER=1"
goto wait_for_service

:launch_server_with_portable_go
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SERVER_LAUNCH_SCRIPT%" -StartScript "%START_SCRIPT%" -LockDir "%LOCK_DIR%" -GoPath "%PORTABLE_GO%" >nul 2>&1
if errorlevel 1 goto start_failed
set "LAUNCHED_SERVER=1"

:wait_for_service
call :service_is_ready
if not errorlevel 1 goto open_admin
if "%STARTER%"=="0" call :try_acquire_start_lock
if "%STARTER%"=="1" if not defined LAUNCHED_SERVER goto start_server
call :deadline_reached
if not errorlevel 1 goto start_failed
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "Start-Sleep -Seconds 1"
goto wait_for_service

:open_admin
call :release_ready_service_lock
start "" "http://localhost:%PORT%/admin"
echo Da mo trang test QR: http://localhost:%PORT%/admin
echo Muon dung server: quay lai cua so "QR Server" va nhan Ctrl+C.
exit /b 0

:start_failed
call :retain_or_release_start_lock_after_failure
echo [LOI] Server chua san sang sau khoang 30 giay.
echo Hay xem cua so "QR Server" de biet loi cau hinh, Go hoac cong %PORT%.
pause
exit /b 1

:invalid_config
echo [LOI] PORT trong .env phai la so tu 1 den 65535.
echo Hay dat SSL="FALSE" trong .env de dung launcher test local.
pause
exit /b 1

:ssl_not_supported
echo [LOI] MO_TEST_QR.bat chi dung de test local voi SSL=FALSE.
echo Hay dat SSL="FALSE" trong .env, hoac khoi dong theo cau hinh HTTPS rieng.
pause
exit /b 1

:set_deadline
set "DEADLINE="
for /f %%T in ('powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "[DateTimeOffset]::UtcNow.ToUnixTimeSeconds() + 30"') do set "DEADLINE=%%T"
if not defined DEADLINE goto start_failed
exit /b 0

:deadline_reached
set "NOW="
for /f %%T in ('powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "[DateTimeOffset]::UtcNow.ToUnixTimeSeconds()"') do set "NOW=%%T"
if not defined NOW exit /b 0
if %NOW% GEQ %DEADLINE% exit /b 0
exit /b 1

:release_ready_service_lock
if defined LOCK_DIR rmdir /s /q "%LOCK_DIR%" >nul 2>&1
exit /b 0

:retain_or_release_start_lock_after_failure
if not "%STARTER%"=="1" exit /b 0
call :server_window_is_alive
if not errorlevel 1 exit /b 0
call :release_ready_service_lock
exit /b 0

:try_acquire_start_lock
mkdir "%LOCK_DIR%" >nul 2>&1
if not errorlevel 1 goto lock_acquired
call :reclaim_stale_lock
mkdir "%LOCK_DIR%" >nul 2>&1
if not errorlevel 1 goto lock_acquired
exit /b 1

:lock_acquired
set "STARTER=1"
call :write_start_lock
if not errorlevel 1 exit /b 0
set "STARTER=0"
rmdir /s /q "%LOCK_DIR%" >nul 2>&1
exit /b 1

:write_start_lock
set "QR_TEST_LOCK_DIR=%LOCK_DIR%"
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference = 'Stop'; try { Set-Content -LiteralPath (Join-Path $env:QR_TEST_LOCK_DIR 'started-at.txt') -Value ([DateTimeOffset]::UtcNow.ToUnixTimeSeconds()) -NoNewline; exit 0 } catch { exit 1 }" >nul 2>&1
exit /b %ERRORLEVEL%

:reclaim_stale_lock
set "QR_TEST_LOCK_DIR=%LOCK_DIR%"
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference = 'Stop'; try { $lock = $env:QR_TEST_LOCK_DIR; if (-not (Test-Path -LiteralPath $lock)) { exit 0 }; $item = Get-Item -LiteralPath $lock; $started = [DateTimeOffset]$item.CreationTimeUtc; $startedAt = Join-Path $lock 'started-at.txt'; if (Test-Path -LiteralPath $startedAt) { [long]$parsed = 0; if ([long]::TryParse((Get-Content -LiteralPath $startedAt -Raw).Trim(), [ref]$parsed)) { $started = [DateTimeOffset]::FromUnixTimeSeconds($parsed) } }; $age = ([DateTimeOffset]::UtcNow - $started).TotalSeconds; $serverAlive = $false; $serverPath = Join-Path $lock 'server.pid'; if (Test-Path -LiteralPath $serverPath) { [int]$server = 0; if ([int]::TryParse((Get-Content -LiteralPath $serverPath -Raw).Trim(), [ref]$server)) { $serverAlive = $null -ne (Get-Process -Id $server -ErrorAction SilentlyContinue) } }; if (($age -ge 5) -and -not $serverAlive) { Remove-Item -LiteralPath $lock -Recurse -Force }; exit 0 } catch { exit 0 }" >nul 2>&1
exit /b 0

:server_window_is_alive
set "QR_TEST_LOCK_DIR=%LOCK_DIR%"
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$path = Join-Path $env:QR_TEST_LOCK_DIR 'server.pid'; if (Test-Path -LiteralPath $path) { [int]$server = 0; if ([int]::TryParse((Get-Content -LiteralPath $path -Raw).Trim(), [ref]$server) -and $null -ne (Get-Process -Id $server -ErrorAction SilentlyContinue)) { exit 0 } }; exit 1" >nul 2>&1
exit /b %ERRORLEVEL%

:service_is_ready
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference = 'Stop'; try { $response = Invoke-WebRequest -UseBasicParsing 'http://localhost:%PORT%/health' -TimeoutSec 1; if ($response.StatusCode -eq 200) { exit 0 } } catch {}; exit 1" >nul 2>&1
exit /b %ERRORLEVEL%
