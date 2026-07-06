#!/bin/sh
WORKSPACE_FILE="/workspace/.last_access"

# 60 days in seconds = 60 * 24 * 3600 = 5184000
TIMEOUT=5184000

echo "Monitoring service started."

while true; do
  if [ ! -f "$WORKSPACE_FILE" ]; then
    echo "Initializing tracker..."
    touch "$WORKSPACE_FILE"
  fi

  # Get last modification time of the file
  MOD_TIME=$(stat -c %Y "$WORKSPACE_FILE")
  CUR_TIME=$(date +%s)
  AGE=$((CUR_TIME - MOD_TIME))

  # Sleep interval: check every 24 hours (86400 seconds)
  # But for the check itself, compare the age with the timeout
  if [ "$AGE" -gt "$TIMEOUT" ]; then
    echo "Retention period exceeded. Wiping workspace data..."
    
    # Safely clear the workspace directory contents
    find /workspace -mindepth 1 -delete
    
    echo "Tearing down development environment..."
    # Self-destruct compose services and volumes
    docker compose -f /app/docker-compose.yml down -v
    exit 0
  fi

  sleep 86400
done
