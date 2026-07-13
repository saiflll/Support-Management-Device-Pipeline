const BASE = '';

async function apiFetch(pth: string, opt?: RequestInit) {
    const res = await fetch(BASE + pth, opt);
    if (!res.ok) throw new Error(`API Error ${res.status}: ${await res.text()}`);
    return res.json();
}

export function getNodes() { return apiFetch('/api/nodes'); }
export function getLogs(id: string) { return apiFetch(`/logs/${id}`); }
export function getNodeConfig(id: string) { return apiFetch(`/api/nodes/${id}/config`); }
export function deleteNode(id: string) {
    return apiFetch(`/api/nodes/${id}`, { method: 'DELETE' });
}
export function sendConfig(pl: Record<string, unknown>) {
    return apiFetch('/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(pl)
    });
}
export function sendOTA(nd: string, url: string) {
    return apiFetch('/ota', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node: nd, url })
    });
}
export function sendReboot(nd: string) {
    return apiFetch('/reboot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node: nd })
    });
}

export function getFiles() { return apiFetch('/api/files'); }
export function deleteFile(nm: string) {
    return apiFetch(`/api/files/${nm}`, { method: 'DELETE' });
}
export function renameFile(nm: string, nmBru: string) {
    return apiFetch(`/api/files/${nm}/rename`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_name: nmBru })
    });
}
export async function uploadFiles(fls: File[]) {
    const frm = new FormData();
    fls.forEach(f => frm.append('file', f));
    const res = await fetch('/upload', { method: 'POST', body: frm });
    if (!res.ok) throw new Error(`Upload error: ${res.status}`);
    return res.json();
}

export function getForwarderStatus() { return apiFetch('/forwarder/status'); }
export function getMonitorStatus() { return apiFetch('/monitor/status'); }

export function formatBytes(b: number, dec = 2): string {
    if (b === 0) return '0 B';
    const k = 1024;
    const dm = dec < 0 ? 0 : dec;
    const sz = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(b) / Math.log(k));
    return parseFloat((b / Math.pow(k, i)).toFixed(dm)) + ' ' + sz[i];
}

export function formatSpeed(b: number): string {
    if (b < 1024) return b + ' B/s';
    if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB/s';
    return (b / 1024 / 1024).toFixed(1) + ' MB/s';
}

export function formatUptime(sec: number): string {
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    return `${h}h ${m}m ${s}s`;
}
