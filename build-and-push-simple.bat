@echo off
REM Docker Hub Build & Push Script (Batch - Simple Version)
REM Usage: build-and-push-simple.bat [version]

setlocal enabledelayedexpansion

REM Configuration
set DOCKER_USERNAME=rennnagge
set VERSION=%1
if "%VERSION%"=="" set VERSION=latest

echo.
echo ==========================================
echo Docker Hub Build ^& Push Script
echo Version: %VERSION%
echo Username: %DOCKER_USERNAME%
echo ==========================================
echo.

REM Note: Make sure you're logged in
echo [WARNING] Make sure you're logged in to Docker Hub!
echo [INFO] If not logged in, run: docker login
echo.
pause
echo.

REM Build and push each service
echo [INFO] Building and pushing services...
echo.

REM 1. OTA Service
call :BuildAndPush "OTA Service" ".\ota" "ota-app"
if errorlevel 1 exit /b 1
echo.

REM 2. Forwarder Service
call :BuildAndPush "Forwarder Service" ".\forward" "forwarder-app"
if errorlevel 1 exit /b 1
echo.

REM 3. Forming Service
call :BuildAndPush "Forming Service" ".\forming" "forming-app"
if errorlevel 1 exit /b 1
echo.

REM Summary
echo ==========================================
echo [SUCCESS] All services built and pushed successfully!
echo ==========================================
echo.
echo Images pushed:
echo   - %DOCKER_USERNAME%/ota-app:%VERSION%
echo   - %DOCKER_USERNAME%/forwarder-app:%VERSION%
echo   - %DOCKER_USERNAME%/forming-app:%VERSION%
echo.
echo To use these images, update docker-compose.yml:
echo   image: %DOCKER_USERNAME%/ota-app:%VERSION%
echo.
echo [SUCCESS] Done!
exit /b 0

REM Function to build and push image
:BuildAndPush
set SERVICE_NAME=%~1
set CONTEXT=%~2
set IMAGE_NAME=%~3

echo [INFO] Building %SERVICE_NAME%...

REM Build image
docker build --platform linux/amd64 -t "%DOCKER_USERNAME%/%IMAGE_NAME%:%VERSION%" -t "%DOCKER_USERNAME%/%IMAGE_NAME%:latest" %CONTEXT%
if errorlevel 1 (
    echo [ERROR] Failed to build %SERVICE_NAME%
    exit /b 1
)

echo [SUCCESS] %SERVICE_NAME% built successfully

REM Push to Docker Hub
echo [INFO] Pushing %SERVICE_NAME% to Docker Hub...

docker push "%DOCKER_USERNAME%/%IMAGE_NAME%:%VERSION%"
if errorlevel 1 (
    echo [ERROR] Failed to push %SERVICE_NAME%
    exit /b 1
)

docker push "%DOCKER_USERNAME%/%IMAGE_NAME%:latest"
if errorlevel 1 (
    echo [ERROR] Failed to push %SERVICE_NAME%
    exit /b 1
)

echo [SUCCESS] %SERVICE_NAME% pushed successfully
exit /b 0
