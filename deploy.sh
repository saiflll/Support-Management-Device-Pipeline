#!/bin/bash
# Deploy Script for IoT OTA Server

echo "================================"
echo "IoT OTA Server - Deploy Script"
echo "================================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    exit 1
fi

# Stop existing containers
echo -e "${YELLOW}Stopping existing containers...${NC}"
docker-compose down

# Remove old images (optional)
read -p "Remove old images? (y/n): " remove_images
if [ "$remove_images" == "y" ]; then
    echo -e "${YELLOW}Removing old images...${NC}"
    docker-compose down --rmi all
fi

# Build and start containers
echo -e "${YELLOW}Building and starting containers...${NC}"
docker-compose up --build -d

# Wait for containers to start
echo -e "${YELLOW}Waiting for containers to start...${NC}"
sleep 5

# Check container status
echo -e "${GREEN}Container Status:${NC}"
docker-compose ps

# Show logs
echo -e "${YELLOW}Recent logs:${NC}"
docker-compose logs --tail=50

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo "OTA Server running on: http://localhost:9999"
echo ""
echo "Useful commands:"
echo "  docker-compose logs -f          # View all logs"
echo "  docker-compose logs -f ota      # View OTA logs"
echo "  docker-compose logs -f forwarder # View Forwarder logs"
echo "  docker-compose restart          # Restart services"
echo "  docker-compose down             # Stop services"
echo ""
