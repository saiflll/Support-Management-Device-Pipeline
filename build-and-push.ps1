# Docker Hub Build & Push Script (PowerShell)
# Usage: .\build-and-push.ps1 [version]

param(
    [string]$Version = "latest"
)

# Configuration
$DockerUsername = "rennn"

# Function to print colored output
function Print-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Blue
}

function Print-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Print-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Print-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

# Function to build and push image
function Build-And-Push {
    param(
        [string]$Service,
        [string]$Context,
        [string]$ImageName
    )
    
    Print-Info "Building $Service..."
    
    # Build image
    docker build `
        --platform linux/amd64 `
        -t "${DockerUsername}/${ImageName}:${Version}" `
        -t "${DockerUsername}/${ImageName}:latest" `
        $Context
    
    if ($LASTEXITCODE -ne 0) {
        Print-Error "Failed to build $Service"
        exit 1
    }
    
    Print-Success "$Service built successfully"
    
    # Push to Docker Hub
    Print-Info "Pushing $Service to Docker Hub..."
    
    docker push "${DockerUsername}/${ImageName}:${Version}"
    docker push "${DockerUsername}/${ImageName}:latest"
    
    if ($LASTEXITCODE -ne 0) {
        Print-Error "Failed to push $Service"
        exit 1
    }
    
    Print-Success "$Service pushed successfully"
}

# Main script
Write-Host ""
Print-Info "=========================================="
Print-Info "Docker Hub Build & Push Script"
Print-Info "Version: $Version"
Print-Info "Username: $DockerUsername"
Print-Info "=========================================="
Write-Host ""

# Check if logged in to Docker Hub
Print-Info "Checking Docker Hub login..."
$dockerInfo = docker info 2>&1 | Out-String
if ($dockerInfo -notmatch "Username: $DockerUsername") {
    Print-Warning "Not logged in to Docker Hub"
    Print-Info "Please login first: docker login"
    exit 1
}
Print-Success "Logged in to Docker Hub"
Write-Host ""

# Build and push each service
Print-Info "Building and pushing services..."
Write-Host ""

# 1. OTA Service
Build-And-Push -Service "OTA Service" -Context ".\ota" -ImageName "ota-app"
Write-Host ""

# 2. Forwarder Service
Build-And-Push -Service "Forwarder Service" -Context ".\forward" -ImageName "forwarder-app"
Write-Host ""

# 3. Forming Service
Build-And-Push -Service "Forming Service" -Context ".\forming" -ImageName "forming-app"
Write-Host ""

# 4. pgweb Proxy
Build-And-Push -Service "pgweb Proxy" -Context ".\pgweb" -ImageName "pgweb-proxy"
Write-Host ""

# Summary
Print-Info "=========================================="
Print-Success "All services built and pushed successfully!"
Print-Info "=========================================="
Write-Host ""
Print-Info "Images pushed:"
Write-Host "  - ${DockerUsername}/ota-app:${Version}"
Write-Host "  - ${DockerUsername}/forwarder-app:${Version}"
Write-Host "  - ${DockerUsername}/forming-app:${Version}"
Write-Host "  - ${DockerUsername}/pgweb-proxy:${Version}"
Write-Host ""
Print-Info "To use these images, update docker-compose.yml:"
Write-Host "  image: ${DockerUsername}/ota-app:${Version}"
Write-Host ""
Print-Success "Done! 🚀"
