@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0.."

:: ============================================================
::  Publish @beilin/runtime-* packages from a GitHub release
::
::  Usage:
::    scripts\publish-runtime-from-release.bat
::    scripts\publish-runtime-from-release.bat 0.3.0
::    scripts\publish-runtime-from-release.bat 0.3.0 v0.3.0
::
::  Steps:
::    1. Download pcraft-*.tar.gz from the GitHub release
::    2. Repackage into @beilin/runtime-* npm folders
::    3. npm publish --access public each runtime package
::
::  Requires: gh, npm, and bash (Git Bash) on PATH
:: ============================================================

set "VERSION=%~1"
set "TAG=%~2"
if "%VERSION%"=="" set "VERSION=0.3.0"
if "%TAG%"=="" set "TAG=v%VERSION%"

set "ROOT_DIR=%CD%"
set "ASSETS_DIR=%ROOT_DIR%\dist\release-assets"
set "OUT_DIR=%ROOT_DIR%\dist\npm-runtime"
set "PKG_SCRIPT=%ROOT_DIR%\scripts\release\package-npm-runtime.sh"

echo.
echo ====================================
echo  Publish @beilin/runtime-* @%VERSION%
echo  Tag: %TAG%
echo ====================================
echo.

:: ---- Prerequisites ----
where gh >nul 2>&1
if errorlevel 1 (
  echo ERROR: gh not found on PATH. Install GitHub CLI first.
  exit /b 1
)

where npm >nul 2>&1
if errorlevel 1 (
  echo ERROR: npm not found on PATH.
  exit /b 1
)

where bash >nul 2>&1
if errorlevel 1 (
  echo ERROR: bash not found on PATH.
  echo Install Git for Windows and ensure "Git Bash" is on PATH.
  exit /b 1
)

if not exist "%PKG_SCRIPT%" (
  echo ERROR: Missing packaging script:
  echo   %PKG_SCRIPT%
  exit /b 1
)

if not exist "%ASSETS_DIR%" mkdir "%ASSETS_DIR%"
if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

:: Convert Windows paths to Git-Bash posix paths (/c/Users/...)
for /f "usebackq delims=" %%I in (`bash -lc "cygpath -u '%PKG_SCRIPT%'"`) do set "PKG_SCRIPT_UNIX=%%I"
for /f "usebackq delims=" %%I in (`bash -lc "cygpath -u '%ASSETS_DIR%'"`) do set "ASSETS_DIR_UNIX=%%I"
for /f "usebackq delims=" %%I in (`bash -lc "cygpath -u '%OUT_DIR%'"`) do set "OUT_DIR_UNIX=%%I"

if not defined PKG_SCRIPT_UNIX (
  echo ERROR: Failed to convert paths with cygpath. Is Git Bash installed correctly?
  exit /b 1
)

:: ---- Step 1: Download release tarballs ----
echo [1/3] Downloading release assets from %TAG%...
for %%P in (linux-x64 linux-arm64 macos-x64 macos-arm64 windows-x64) do (
  echo   downloading pcraft-%%P.tar.gz
  gh release download "%TAG%" --pattern "pcraft-%%P.tar.gz" --dir "%ASSETS_DIR%" --clobber
  if errorlevel 1 (
    echo ERROR: Failed to download pcraft-%%P.tar.gz from release %TAG%
    exit /b 1
  )
)
echo   done.
echo.

:: ---- Step 2: Package npm runtime folders ----
echo [2/3] Packaging @beilin/runtime-* folders...
echo   bash "%PKG_SCRIPT_UNIX%" "%VERSION%" "%ASSETS_DIR_UNIX%" "%OUT_DIR_UNIX%"
bash "%PKG_SCRIPT_UNIX%" "%VERSION%" "%ASSETS_DIR_UNIX%" "%OUT_DIR_UNIX%"
if errorlevel 1 (
  echo ERROR: package-npm-runtime.sh failed
  exit /b 1
)
echo.

:: ---- Step 3: Publish each runtime package ----
echo [3/3] Publishing packages to npm...
set "FAILED=0"
for %%N in (
  runtime-linux-x64
  runtime-linux-arm64
  runtime-darwin-x64
  runtime-darwin-arm64
  runtime-win32-x64
) do (
  set "PKG_DIR=%OUT_DIR%\@beilin\%%N"
  if not exist "!PKG_DIR!\package.json" (
    echo   FAIL missing package dir: !PKG_DIR!
    set "FAILED=1"
  ) else (
    echo   publishing @beilin/%%N@%VERSION%
    pushd "!PKG_DIR!" >nul
    call npm publish --access public
    if errorlevel 1 (
      echo   FAIL @beilin/%%N
      set "FAILED=1"
    ) else (
      echo   ok   @beilin/%%N
    )
    popd >nul
  )
  echo.
)

if "%FAILED%"=="1" (
  echo.
  echo ====================================
  echo  Finished with errors.
  echo  If you saw EOTP, complete the npm
  echo  browser auth link and re-run this
  echo  bat ^(already-published packages can
  echo  be ignored / will fail on re-publish^).
  echo ====================================
  exit /b 1
)

echo.
echo ====================================
echo  All 5 @beilin/runtime-* packages published at %VERSION%
echo  Next: configure Trusted Publisher on each package page.
echo ====================================
exit /b 0
