# Stealthy Development Environment (sup-developt & developt)

This folder contains the Docker Compose configuration for an isolated development environment running a sandboxed VS Code editor and Docker-in-Docker daemon.

## Container Naming Schema
To blend in on the production server, the containers are named unobtrusively:
- `sup-developt` (Lazytainer Proxy): Listens on port `18080` and manages automatic sleep/wake cycles.
- `developt` (VS Code Server): The actual editor. Only runs when you are actively using it.
- `developt-daemon` (Docker-in-Docker): Wakes up to run your nested dev containers.
- `developt-mon` (Self-Destroyer): Telemetry daemon that wipes files and stops the stack if unused for 60 days.

---

## Features
- **Auto-Suspend (Sleep)**: If you don't open the browser for 10 minutes, `developt` and `developt-daemon` automatically stop to free up memory on the server.
- **Resource Constraints**:
  - `developt`: Limited to **1 CPU** and **3 GB RAM**.
  - `developt-daemon`: Limited to **2 CPUs** and **5 GB RAM**.
  - Total combined resource footprint when actively developing will never exceed **8 GB RAM**. When sleeping, the footprint is virtually zero (~10MB RAM for the proxy and monitor).
- **Self-Destruction (Security)**: If the workspace is not accessed (woken up) for **60 consecutive days**, the `developt-mon` container will wipe all files in the `./workspace` folder, and tear down the container stack and volumes.

---

## Getting Started

### 1. Manual Startup (on Server)
To start the services manually on the server:
```bash
# Navigate to the vscode folder
cd openvscode-server

# Create the workspace folder if it doesn't exist
mkdir -p workspace

# Start the stack
docker compose up -d
```

### 2. Accessing VS Code
Open your web browser and navigate to:
```
http://<YOUR_SERVER_IP>:18080/?tkn=170845Hutri
```
- **Port**: `18080` (mapped through the proxy)
- **Token**: `170845Hutri`

### 3. Verification
- Open the terminal inside VS Code and run `docker ps`. You are inside the isolated docker daemon (`developt-daemon`).
- On the host server, running `docker ps` will only show:
  - `sup-developt` (always up)
  - `developt-mon` (always up)
  - `developt` (running only when browser is open)
  - `developt-daemon` (running only when browser is open)
