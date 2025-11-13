@echo off
REM prepare-deploy.bat - Generate vendor directories for deployment (Windows)

echo ===================================
echo Generating vendor directories...
echo ===================================

REM Forward service
echo.
echo 📦 Processing forward service...
cd forward
if exist go.mod (
    echo   Running go mod vendor...
    go mod vendor
    if exist vendor (
        echo   ✅ forward/vendor generated successfully
    ) else (
        echo   ❌ Failed to generate forward/vendor
        exit /b 1
    )
) else (
    echo   ❌ go.mod not found in forward/
    exit /b 1
)
cd ..

REM OTA service
echo.
echo 📦 Processing ota service...
cd ota
if exist go.mod (
    echo   Running go mod vendor...
    go mod vendor
    if exist vendor (
        echo   ✅ ota/vendor generated successfully
    ) else (
        echo   ❌ Failed to generate ota/vendor
        exit /b 1
    )
) else (
    echo   ❌ go.mod not found in ota/
    exit /b 1
)
cd ..

echo.
echo ===================================
echo ✅ All vendor directories generated
echo ===================================
echo.
echo Next steps:
echo 1. Review the changes: git status
echo 2. Add to git: git add forward\vendor ota\vendor
echo 3. Commit: git commit -m "Add vendor directories for deployment"
echo 4. Push: git push
echo 5. Deploy on server: docker-compose up -d --build
echo.
pause
