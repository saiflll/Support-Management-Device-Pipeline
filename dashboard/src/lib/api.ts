// API base URL — sama satu origin (Fiber OTA serve di 9999)
const BASE = '';

async function apiFetch(path: string, opts?: RequestInit) {
    const res = await fetch(BASE + path, opts);
    if (!res.ok) throw new Error(`API Error ${res.status}: ${await res.text()}`);
    return res.json();
}

// ─── Nodes ────────────────────────────────────────────────
export function getNodes() { return apiFetch('/api/nodes'); }
export function getLogs(id: string) { return apiFetch(`/logs/${id}`); }
export function getNodeConfig(id: string) { return apiFetch(`/api/nodes/${id}/config`); }
export function deleteNode(id: string) {
    return apiFetch(`/api/nodes/${id}`, { method: 'DELETE' });
}
export function sendConfig(payload: Record<string, unknown>) {
    return apiFetch('/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });
}
export function sendOTA(node: string, url: string) {
    return apiFetch('/ota', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node, url })
    });
}
export function sendReboot(node: string) {
    return apiFetch('/reboot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node })
    });
}

// ─── Files ────────────────────────────────────────────────
export function getFiles() { return apiFetch('/api/files'); }
export function deleteFile(name: string) {
    return apiFetch(`/api/files/${name}`, { method: 'DELETE' });
}
export function renameFile(name: string, newName: string) {
    return apiFetch(`/api/files/${name}/rename`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_name: newName })
    });
}
export async function uploadFiles(files: File[]) {
    const form = new FormData();
    files.forEach(f => form.append('file', f));
    const res = await fetch('/upload', { method: 'POST', body: form });
    if (!res.ok) throw new Error(`Upload error: ${res.status}`);
    return res.json();
}

// ─── Forwarder ────────────────────────────────────────────
export function getForwarderStatus() { return apiFetch('/forwarder/status'); }
export function getPipelines() { return apiFetch('/api/pipelines'); }
export function createPipeline(data: unknown) {
    return apiFetch('/api/pipelines', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
}
export function deletePipeline(id: number) {
    return apiFetch(`/api/pipelines/${id}`, { method: 'DELETE' });
}

// ─── Monitor ──────────────────────────────────────────────
export function getMonitorStatus() { return apiFetch('/monitor/status'); }

// ─── Utils ────────────────────────────────────────────────
export function formatBytes(bytes: number, decimals = 2): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

export function formatSpeed(bytes: number): string {
    if (bytes < 1024) return bytes + ' B/s';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB/s';
    return (bytes / 1024 / 1024).toFixed(1) + ' MB/s';
}

export function formatUptime(sec: number): string {
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    return `${h}h ${m}m ${s}s`;
}
