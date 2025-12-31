
echo "=========================================="
echo "Updating Services to Latest Version..."
echo "=========================================="
echo "Step 1: Pulling latest images from Docker Hub..."
docker compose pull
echo "Step 2: Re-creating containers (if updated)..."
docker compose up -d
echo "Step 3: Cleaning up old dangling images..."
docker image prune -f
echo "=========================================="
echo "UPDATE COMPLETE! All data preserved."
echo "=========================================="
