@echo off
REM Docker Hub Build & Push Script (Batch)
REM Usage: build-and-push.bat [version]

setlocal enabledelayedexpansion

REM Configuration
set DOCKER_USERNAME=rennnagge
set VERSION=%1
if "%VERSION%"=="" set VERSION=latest

REM Colors (using ANSI escape codes - works on Windows 10+)
set "INFO=[94m"
set "SUCCESS=[92m"
set "WARNING=[93m"
set "ERROR=[91m"
set "RESET=[0m"

echo.
echo %INFO%==========================================%RESET%
echo %INFO%Docker Hub Build ^& Push Script%RESET%
echo %INFO%Version: %VERSION%%RESET%
echo %INFO%Username: %DOCKER_USERNAME%%RESET%
echo %INFO%==========================================%RESET%
echo.

REM Note: Make sure you're logged in
echo %WARNING%Make sure you're logged in to Docker Hub!%RESET%
echo %INFO%If not logged in, run: docker login%RESET%
echo.
pause
echo.

REM Build and push each service
echo %INFO%Building and pushing services...%RESET%
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
echo %INFO%==========================================%RESET%
echo %SUCCESS%All services built and pushed successfully!%RESET%
echo %INFO%==========================================%RESET%
echo.
echo %INFO%Images pushed:%RESET%
echo   - %DOCKER_USERNAME%/ota-app:%VERSION%
echo   - %DOCKER_USERNAME%/forwarder-app:%VERSION%
echo   - %DOCKER_USERNAME%/forming-app:%VERSION%
echo.
echo %INFO%To use these images, update docker-compose.yml:%RESET%
echo   image: %DOCKER_USERNAME%/ota-app:%VERSION%
echo.
echo %SUCCESS%Done!%RESET%
exit /b 0

REM Function to build and push image
:BuildAndPush
set SERVICE_NAME=%~1
set CONTEXT=%~2
set IMAGE_NAME=%~3

echo %INFO%Building %SERVICE_NAME%...%RESET%

REM Build image
docker build --platform linux/amd64 -t "%DOCKER_USERNAME%/%IMAGE_NAME%:%VERSION%" -t "%DOCKER_USERNAME%/%IMAGE_NAME%:latest" %CONTEXT%
if errorlevel 1 (
    echo %ERROR%Failed to build %SERVICE_NAME%%RESET%
    exit /b 1
)

echo %SUCCESS%%SERVICE_NAME% built successfully%RESET%

REM Push to Docker Hub
echo %INFO%Pushing %SERVICE_NAME% to Docker Hub...%RESET%

docker push "%DOCKER_USERNAME%/%IMAGE_NAME%:%VERSION%"
if errorlevel 1 (
    echo %ERROR%Failed to push %SERVICE_NAME%%RESET%
    exit /b 1
)

docker push "%DOCKER_USERNAME%/%IMAGE_NAME%:latest"
if errorlevel 1 (
    echo %ERROR%Failed to push %SERVICE_NAME%%RESET%
    exit /b 1
)

echo %SUCCESS%%SERVICE_NAME% pushed successfully%RESET%
exit /b 0
