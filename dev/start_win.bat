@echo off
setlocal EnableDelayedExpansion
cls

echo ========================================================
echo   IOT SERVICE LAUNCHER (MINIMALIST)
echo ========================================================
echo.

:: 1. Stop Containers (Quiet)
echo [1/3] Stopping existing containers...
docker-compose down >nul 2>&1

:: 2. Build & Start (With Built-in Progress UI)
echo [2/3] Building and Starting...
echo       (Docker will show progress below)
echo.
:: Set BuildKit to tty mode for single-line animation (cleaner than plain text)
set BUILDKIT_PROGRESS=tty
docker-compose up -d --build --remove-orphans

if !errorlevel! neq 0 (
    echo.
    echo [ERROR] Failed to start services.
    pause
    exit /b 1
)

:: 3. Initializing & Checking
echo.
echo [3/3] Initializing (5s)...
timeout /t 5 /nobreak >nul
cls

echo ========================================================
echo   READY - SERVICE STATUS
echo ========================================================
docker-compose ps --format "table {{.Name}}\t{{.State}}\t{{.Status}}\t{{.Ports}}"

echo.
echo ========================================================
echo View logs: docker-compose logs -f [service_name]
echo.
pause
