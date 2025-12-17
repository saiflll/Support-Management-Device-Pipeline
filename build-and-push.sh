#!/bin/bash

# Docker Hub Build & Push Script
# Usage: ./build-and-push.sh [version]

set -e

# Configuration
DOCKER_USERNAME="rennn"
VERSION=${1:-"latest"}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to build and push image
build_and_push() {
    local service=$1
    local context=$2
    local image_name=$3
    
    print_info "Building $service..."
    
    # Build image
    docker build \
        --platform linux/amd64 \
        -t ${DOCKER_USERNAME}/${image_name}:${VERSION} \
        -t ${DOCKER_USERNAME}/${image_name}:latest \
        ${context}
    
    if [ $? -eq 0 ]; then
        print_success "$service built successfully"
    else
        print_error "Failed to build $service"
        exit 1
    fi
    
    # Push to Docker Hub
    print_info "Pushing $service to Docker Hub..."
    
    docker push ${DOCKER_USERNAME}/${image_name}:${VERSION}
    docker push ${DOCKER_USERNAME}/${image_name}:latest
    
    if [ $? -eq 0 ]; then
        print_success "$service pushed successfully"
    else
        print_error "Failed to push $service"
        exit 1
    fi
}

# Main script
echo ""
print_info "=========================================="
print_info "Docker Hub Build & Push Script"
print_info "Version: $VERSION"
print_info "Username: $DOCKER_USERNAME"
print_info "=========================================="
echo ""

# Check if logged in to Docker Hub
print_info "Checking Docker Hub login..."
if ! docker info | grep -q "Username: ${DOCKER_USERNAME}"; then
    print_warning "Not logged in to Docker Hub"
    print_info "Please login first: docker login"
    exit 1
fi
print_success "Logged in to Docker Hub"
echo ""

# Build and push each service
print_info "Building and pushing services..."
echo ""

# 1. OTA Service
build_and_push "OTA Service" "./ota" "ota-app"
echo ""

# 2. Forwarder Service
build_and_push "Forwarder Service" "./forward" "forwarder-app"
echo ""

# 3. Forming Service
build_and_push "Forming Service" "./forming" "forming-app"
echo ""

# 4. pgweb Proxy
build_and_push "pgweb Proxy" "./pgweb" "pgweb-proxy"
echo ""

# Summary
print_info "=========================================="
print_success "All services built and pushed successfully!"
print_info "=========================================="
echo ""
print_info "Images pushed:"
echo "  - ${DOCKER_USERNAME}/ota-app:${VERSION}"
echo "  - ${DOCKER_USERNAME}/forwarder-app:${VERSION}"
echo "  - ${DOCKER_USERNAME}/forming-app:${VERSION}"
echo "  - ${DOCKER_USERNAME}/pgweb-proxy:${VERSION}"
echo ""
print_info "To use these images, update docker-compose.yml:"
echo "  image: ${DOCKER_USERNAME}/ota-app:${VERSION}"
echo ""
print_success "Done! 🚀"
