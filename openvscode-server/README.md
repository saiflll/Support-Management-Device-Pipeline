# OpenVSCode Server with Docker-in-Docker (Isolated Dev Environment)

This folder contains the Docker Compose setup for a sandboxed OpenVSCode Server environment running alongside a Docker-in-Docker (DinD) service.

## Features
- **Total Isolation**: Development containers you launch inside the VS Code terminal run in a nested Docker daemon (`openvscode_dind`). They cannot see, access, or modify production containers on the host.
- **Resource Constraints**:
  - VS Code Server is limited to **1 CPU** and **2 GB RAM**.
  - Docker-in-Docker (where your dev containers run) is limited to **2 CPUs** and **4 GB RAM**.
  - Total combined resource footprint will never exceed **6 GB RAM**.
- **User Sandboxing**: VS Code runs as a non-root user (UID 1000) with dropped Linux capabilities.

---

## Getting Started

### 1. Manual Startup (on Server)
To start the services manually on the server:
```bash
# Navigate to the vscode folder
cd openvscode-server

# Create the workspace folder if it doesn't exist
mkdir -p workspace

# Start the containers
docker compose up -d
```

### 2. Accessing VS Code
Open your web browser and navigate to:
```
http://<YOUR_SERVER_IP>:18080/?tkn=170845Hutri
```
- **Port**: `18080` (hardcoded)
- **Token**: `170845Hutri` (hardcoded)

### 3. Usage & Persistence
- Place all your dev project folders and files inside the `/home/workspace` directory in VS Code (which maps to `./workspace` on the host).
- Your VS Code settings and installed extensions are automatically persisted in a named volume (`vscode_data`).
- Your downloaded docker images and dev containers inside DinD are persisted in a named volume (`dind_data`).

### 4. Running Docker in VS Code
Open the integrated terminal in VS Code and verify you can run docker:
```bash
docker ps
docker run -d --name test-nginx -p 8080:80 nginx
```
This test container will run inside the DinD sandbox, completely isolated from your host's production containers.
To check it inside the VS Code terminal:
```bash
docker ps
```
On the host terminal (your SSH session), if you run `docker ps`, you will **only** see `openvscode_server` and `openvscode_dind`, but **not** `test-nginx`! This confirms the absolute isolation.
