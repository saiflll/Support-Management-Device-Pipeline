@echo off
REM Deploy Script for IoT OTA Server (Windows)

echo ================================
echo IoT OTA Server - Deploy Script
echo ================================
echo.

REM Check if docker is running
docker ps >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not running!
    echo Please start Docker Desktop first.
    pause
    exit /b 1
)

echo [INFO] Docker is running...
echo.

REM Stop existing containers
echo [INFO] Stopping existing containers...
docker-compose down
echo.

REM Ask to remove old images
set /p remove_images="Remove old images? (y/n): "
if /i "%remove_images%"=="y" (
    echo [INFO] Removing old images...
    docker-compose down --rmi all
    echo.
)

REM Build and start containers
echo [INFO] Building and starting containers...
docker-compose up --build -d
echo.

REM Wait for containers
echo [INFO] Waiting for containers to start...
timeout /t 5 >nul
echo.

REM Check status
echo [INFO] Container Status:
docker-compose ps
echo.

REM Show logs
echo [INFO] Recent logs:
docker-compose logs --tail=50
echo.

echo ================================
echo Deployment Complete!
echo ================================
echo.
echo OTA Server running on: http://localhost:9999
echo.
echo Useful commands:
echo   docker-compose logs -f               # View all logs
echo   docker-compose logs -f ota           # View OTA logs
echo   docker-compose logs -f forwarder     # View Forwarder logs
echo   docker-compose restart               # Restart services
echo   docker-compose down                  # Stop services
echo.

pause
