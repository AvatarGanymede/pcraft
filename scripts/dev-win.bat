@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0.."

set "ROOT_DIR=%CD%"
set "BACKEND_DIR=%ROOT_DIR%\apps\backend"
set "WEB_DIR=%ROOT_DIR%\apps\web"
set "APPS_DIR=%ROOT_DIR%\apps"
set "BACKEND_PORT=38429"
set "WEB_PORT=37429"

:: ============================================================
::  pcraft build-and-run (Windows)
::  Usage:
::    scripts\dev-win.bat              Build backend + run (with embedded web)
::    scripts\dev-win.bat --dev        Dev mode: backend + Next.js dev server
::    scripts\dev-win.bat --backend    Build and run backend only
::    scripts\dev-win.bat --web        Run Next.js dev server only
::    scripts\dev-win.bat --build      Build only (no run)
:: ============================================================

set "MODE=full"
if /i "%~1"=="--dev"   set "MODE=dev"
if /i "%~1"=="--backend" set "MODE=backend"
if /i "%~1"=="--web"   set "MODE=web"
if /i "%~1"=="--build" set "MODE=build"

:: ---- Step 0: Ensure pnpm is available ----
where pnpm >nul 2>&1
if errorlevel 1 (
    echo   Installing pnpm via npm...
    call npm install -g pnpm
    if errorlevel 1 (
        echo ERROR: Failed to install pnpm
        exit /b 1
    )
)

:: ---- Step 1: Install frontend dependencies ----
if "%MODE%"=="full" goto :install
if "%MODE%"=="dev"   goto :install
if "%MODE%"=="web"   goto :install
if "%MODE%"=="build" goto :install
goto :dispatch

:install
echo.
echo ====================================
echo  [1/4] Installing npm dependencies...
echo ====================================
cd /d "%APPS_DIR%"
call pnpm install
if errorlevel 1 (
    echo ERROR: pnpm install failed
    exit /b 1
)
echo   ^> Dependencies installed.

:dispatch
if "%MODE%"=="web" goto :start_web

:: ---- Step 2: Build frontend (for embedded mode) ----
if "%MODE%"=="dev" goto :build_backend
if "%MODE%"=="backend" goto :build_backend

echo.
echo ====================================
echo  [2/4] Building web frontend...
echo ====================================
cd /d "%APPS_DIR%"
call pnpm --filter @pcraft/web build
if errorlevel 1 (
    echo ERROR: web build failed
    exit /b 1
)
echo   ^> Frontend built.

:: ---- Step 3: Sync embedded web into Go binary ----
echo.
echo ====================================
echo  [3/4] Syncing embedded web assets...
echo ====================================
set "EMBED_DIR=%BACKEND_DIR%\internal\webapp\embedded\generated"
if not exist "%EMBED_DIR%" mkdir "%EMBED_DIR%"
del /q "%EMBED_DIR%\*" 2>nul
for /d %%d in ("%EMBED_DIR%\*") do rmdir /s /q "%%d" 2>nul
xcopy /e /y "%WEB_DIR%\dist\*" "%EMBED_DIR%\" >nul
echo   ^> Embedded web synced.

:: ---- Step 4: Build backend ----
:build_backend
echo.
echo ====================================
echo  Building Go backend...
echo ====================================
cd /d "%BACKEND_DIR%"

:: Clean old binary
if exist "bin\pcraft.exe" del "bin\pcraft.exe" 2>nul

:: Build with embedded web (production mode) or without (dev mode)
if "%MODE%"=="dev" (
    set "TAGS="
) else (
    set "TAGS="
)
go build -o bin\pcraft.exe .\cmd\pcraft\ || exit /b 1
echo   ^> pcraft.exe built

:: Build agentctl (sidecar process for Claude Code communication)
go build -o bin\agentctl.exe .\cmd\agentctl\ || exit /b 1
echo   ^> agentctl.exe built

:: Build mock-agent (for dev/e2e testing without real Claude token usage)
go build -o bin\mock-agent.exe .\cmd\mock-agent\ || exit /b 1
echo   ^> mock-agent.exe built

if "%MODE%"=="build" (
    echo.
    echo ====================================
    echo  Build complete! Binary at:
    echo  %BACKEND_DIR%\bin\pcraft.exe
    echo ====================================
    exit /b 0
)

:: ---- Run ----
if "%MODE%"=="dev" goto :run_dev
if "%MODE%"=="backend" goto :run_backend

:: Full mode: backend serves embedded web
echo.
echo ====================================
echo  Starting pcraft server...
echo  Backend : http://localhost:%BACKEND_PORT%
echo ====================================
cd /d "%BACKEND_DIR%"
bin\pcraft.exe __backend -port %BACKEND_PORT%
goto :end

:run_backend
echo.
echo ====================================
echo  Starting backend only...
echo  Backend API: http://localhost:%BACKEND_PORT%
echo ====================================
cd /d "%BACKEND_DIR%"
bin\pcraft.exe __backend -port %BACKEND_PORT%
goto :end

:run_dev
echo.
echo ====================================
echo  Starting in DEV mode...
echo  Backend : http://localhost:%BACKEND_PORT%
echo  Web UI  : http://localhost:%WEB_PORT%
echo  (backend proxies to Next.js dev server)
echo ====================================

:: Start Next.js dev server in a new window
start "pcraft-web" cmd /c "cd /d "%APPS_DIR%" && set PORT=%WEB_PORT% && pnpm --filter @pcraft/web dev && pause"

:: Give Next.js a moment to start
echo Waiting for Next.js dev server...
timeout /t 5 /nobreak >nul

:: Start backend with web proxy pointing to Next.js
cd /d "%BACKEND_DIR%"
set "PCRAFT_WEB_INTERNAL_URL=http://localhost:%WEB_PORT%"
bin\pcraft.exe __backend -port %BACKEND_PORT%
goto :end

:start_web
echo.
echo ====================================
echo  Starting Next.js dev server...
echo  Web UI : http://localhost:%WEB_PORT%
echo ====================================
cd /d "%APPS_DIR%"
set "PORT=%WEB_PORT%"
pnpm --filter @pcraft/web dev
goto :end

:end
endlocal
