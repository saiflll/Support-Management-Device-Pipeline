@echo off
setlocal enabledelayedexpansion

echo Testing Docker availability...
docker info >nul 2>&1
if errorlevel 1 (
    echo [91mERROR: Docker is NOT running.[0m
    echo Please start Docker Desktop and try again.
    exit /b 1
) else (
    echo [92mDocker is running.[0m
)
exit /b 0
