<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    getNodes, getLogs, getFiles, uploadFiles, deleteFile,
    sendOTA, sendReboot, sendConfig, deleteNode, getNodeConfig,
    formatBytes
  } from '$lib/api';

  // ── State ─────────────────────────────────────────────────
  let nodes: Record<string, any> = $state({});
  let files: any[] = $state([]);
  let activeTab: 'running' | 'offline' = $state('running');
  let toasts: { id: number; type: string; msg: string }[] = $state([]);
  let toastId = 0;

  // Modal states
  let showLogModal = $state(false);
  let showConfigModal = $state(false);
  let showOtaModal = $state(false);
  let modalLogs: string[] = $state([]);
  let modalNode = $state('');
  let configJson = $state('');
  let otaUrl = $state('');
  let otaNode = $state('');

  // OTA Upload
  let selectedFiles: File[] = $state([]);
  let uploadProgress = $state('');

  // Intervals
  let pollInterval: ReturnType<typeof setInterval>;

  // ── Computed ──────────────────────────────────────────────
  let runningNodes = $derived(
    Object.entries(nodes).filter(([, v]: any) => v.status === 'online')
  );
  let offlineNodes = $derived(
    Object.entries(nodes).filter(([, v]: any) => v.status !== 'online')
  );
  let displayNodes = $derived(activeTab === 'running' ? runningNodes : offlineNodes);

  // ── Lifecycle ─────────────────────────────────────────────
  onMount(() => {
    fetchAll();
    pollInterval = setInterval(fetchAll, 3000);
  });
  onDestroy(() => clearInterval(pollInterval));

  // ── Data Fetch ────────────────────────────────────────────
  async function fetchAll() {
    try {
      const data = await getNodes();
      nodes = data || {};
    } catch {}
    try {
      const data = await getFiles();
      files = data || [];
    } catch {}
  }

  // ── Toast ─────────────────────────────────────────────────
  function toast(msg: string, type = 'info') {
    const id = ++toastId;
    toasts = [...toasts, { id, type, msg }];
    setTimeout(() => { toasts = toasts.filter(t => t.id !== id); }, 4000);
  }

  // ── Log Modal ─────────────────────────────────────────────
  async function openLogs(nodeId: string) {
    modalNode = nodeId;
    showLogModal = true;
    try {
      const data = await getLogs(nodeId);
      modalLogs = data.logs || [];
    } catch {
      modalLogs = ['[ERROR] Failed to fetch logs'];
    }
  }

  // ── Config Modal ──────────────────────────────────────────
  async function openConfig(nodeId: string) {
    modalNode = nodeId;
    showConfigModal = true;
    try {
      const data = await getNodeConfig(nodeId);
      configJson = JSON.stringify(data, null, 2);
    } catch {
      configJson = '{}';
    }
  }

  async function applyConfig() {
    try {
      const parsed = JSON.parse(configJson);
      parsed.node = modalNode;
      await sendConfig(parsed);
      toast('Config applied — node akan reboot', 'success');
      showConfigModal = false;
    } catch (e: any) {
      toast('Error: ' + e.message, 'error');
    }
  }

  // ── OTA Modal ─────────────────────────────────────────────
  function openOta(nodeId: string) {
    otaNode = nodeId;
    otaUrl = '';
    showOtaModal = true;
  }

  function selectFileForOTA(url: string) {
    otaUrl = `${window.location.origin}${url}`;
  }

  async function handleUpload() {
    if (!selectedFiles.length) return;
    try {
      uploadProgress = 'Uploading...';
      await uploadFiles(selectedFiles);
      uploadProgress = 'Upload berhasil!';
      selectedFiles = [];
      const data = await getFiles();
      files = data || [];
      setTimeout(() => uploadProgress = '', 3000);
    } catch (e: any) {
      uploadProgress = 'Error: ' + e.message;
    }
  }

  async function handleDeleteFile(name: string) {
    if (!confirm(`Hapus file ${name}?`)) return;
    try {
      await deleteFile(name);
      files = files.filter(f => f.name !== name);
      toast(`File ${name} dihapus`, 'success');
    } catch (e: any) {
      toast('Error: ' + e.message, 'error');
    }
  }

  async function triggerOTA() {
    if (!otaUrl || !otaNode) return;
    try {
      await sendOTA(otaNode, otaUrl);
      toast(`OTA flash initiated → ${otaNode}`, 'success');
      showOtaModal = false;
    } catch (e: any) {
      toast('OTA error: ' + e.message, 'error');
    }
  }

  async function handleReboot(nodeId: string) {
    if (!confirm(`Reboot node ${nodeId}?`)) return;
    try {
      await sendReboot(nodeId);
      toast(`Reboot command sent → ${nodeId}`, 'success');
    } catch (e: any) {
      toast('Error: ' + e.message, 'error');
    }
  }

  async function handleDeleteNode(nodeId: string) {
    if (!confirm(`Hapus node ${nodeId} dari registry?`)) return;
    try {
      await deleteNode(nodeId);
      toast(`Node ${nodeId} dihapus`, 'success');
    } catch (e: any) {
      toast('Error: ' + e.message, 'error');
    }
  }

  function ramLabel(bytes: number): string {
    return bytes > 0 ? formatBytes(bytes) + ' free' : '-';
  }
</script>

<!-- Toast Container -->
<div class="toast-container">
  {#each toasts as t (t.id)}
    <div class="toast {t.type}">{t.msg}</div>
  {/each}
</div>

<!-- Tab Bar -->
<div class="tab-bar">
  <button class="tab-item" class:active={activeTab === 'running'} onclick={() => activeTab = 'running'}>
    [ RUNNING: {runningNodes.length} ]
  </button>
  <button class="tab-item" class:active={activeTab === 'offline'} onclick={() => activeTab = 'offline'}>
    [ OFFLINE: {offlineNodes.length} ]
  </button>
</div>

<!-- Node Grid -->
{#if displayNodes.length === 0}
  <div class="loading-state">
    <div class="spinner"></div>
    <span>AWAITING_NODES...</span>
  </div>
{:else}
  <div class="grid-4">
    {#each displayNodes as [id, info]}
      <div class="node-card" class:offline={info.status !== 'online'}>
        <div style="display:flex;align-items:flex-start;justify-content:space-between;gap:6px;">
          <div class="node-id">{id}</div>
          <span class="badge" class:badge-green={info.status === 'online'} class:badge-red={info.status !== 'online'}>
            {info.status === 'online' ? 'ONLINE' : 'OFFLINE'}
          </span>
        </div>

        <div class="node-meta">
          <div class="node-meta-item">
            <div class="node-meta-label">MODEL</div>
            <div class="mono text-xs text-accent">{info.model || '-'}</div>
          </div>
          <div class="node-meta-item">
            <div class="node-meta-label">IP</div>
            <div class="mono text-xs">{info.ip || '-'}</div>
          </div>
          <div class="node-meta-item">
            <div class="node-meta-label">RAM</div>
            <div class="mono text-xs text-yellow">{ramLabel(info.ram_free_bytes || 0)}</div>
          </div>
          <div class="node-meta-item">
            <div class="node-meta-label">VER</div>
            <div class="mono text-xs">{info.version || '-'}</div>
          </div>
          <div class="node-meta-item" style="grid-column: span 2;">
            <div class="node-meta-label">UPDATED</div>
            <div class="mono text-xs text-muted">{info.updated || '-'}</div>
          </div>
        </div>

        <div class="node-action-bar">
          <button class="btn btn-ghost" style="font-size:9px;padding:4px 8px;" onclick={() => openLogs(id)}>LOGS</button>
          <button class="btn btn-primary" style="font-size:9px;padding:4px 8px;" onclick={() => openConfig(id)}>CONFIG</button>
          <button class="btn btn-success" style="font-size:9px;padding:4px 8px;" onclick={() => openOta(id)}>OTA</button>
          <button class="btn btn-danger" style="font-size:9px;padding:4px 8px;" onclick={() => handleReboot(id)}>RBT</button>
          <button style="font-size:9px;padding:4px 6px;background:none;border:none;color:var(--text-dim);cursor:pointer;" onclick={() => handleDeleteNode(id)}>✕</button>
        </div>
      </div>
    {/each}
  </div>
{/if}

<!-- Log Modal -->
{#if showLogModal}
  <div class="modal-overlay" onclick={() => showLogModal = false} role="button" tabindex="-1">
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
      <div class="modal-header">
        <span class="modal-title">SYSTEM_LOGS — {modalNode}</span>
        <button onclick={() => showLogModal = false} style="color:var(--text-muted);font-size:16px;">✕</button>
      </div>
      <div class="modal-body" style="background:black;">
        {#each modalLogs as line}
          <div class="mono text-xs" style="padding:2px 0;color:var(--green);line-height:1.7;">{line}</div>
        {:else}
          <div class="text-muted mono text-xs">No logs available.</div>
        {/each}
      </div>
    </div>
  </div>
{/if}

<!-- Config Modal -->
{#if showConfigModal}
  <div class="modal-overlay" onclick={() => showConfigModal = false} role="button" tabindex="-1">
    <div class="modal" style="max-width:780px;" onclick={(e) => e.stopPropagation()} role="dialog">
      <div class="modal-header">
        <span class="modal-title">CONFIG_JSON_EDITOR — {modalNode}</span>
        <button onclick={() => showConfigModal = false} style="color:var(--text-muted);font-size:16px;">✕</button>
      </div>
      <div class="modal-body">
        <div style="display:flex;gap:16px;height:350px;">
          <div style="flex:1;display:flex;flex-direction:column;">
            <div class="form-label">JSON PAYLOAD</div>
            <textarea
              bind:value={configJson}
              style="flex:1;background:black;color:var(--green);font-family:var(--font-mono);font-size:11px;border:1px solid var(--border);border-radius:4px;padding:10px;resize:none;outline:none;"
              spellcheck="false"
            ></textarea>
          </div>
          <div style="width:220px;font-size:10px;color:var(--text-muted);line-height:1.8;">
            <div class="form-label" style="color:var(--accent-2);">INSTRUCTION SET</div>
            <p style="margin-top:8px;">01. Edit values langsung di JSON.</p>
            <p>02. Pastikan JSON valid (no trailing comma).</p>
            <p>03. Field system (IP, Model) akan diabaikan.</p>
            <div style="margin-top:16px;padding:10px;border:1px solid rgba(251,191,36,0.2);border-radius:4px;background:rgba(251,191,36,0.04);">
              <span style="color:var(--yellow);font-weight:700;">⚠ CRITICAL</span><br>
              Apply akan trigger reboot otomatis. Koneksi mungkin terputus 10-15 detik.
            </div>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-ghost" onclick={() => showConfigModal = false}>CANCEL</button>
        <button class="btn btn-primary" onclick={applyConfig}>APPLY_DEPLOYMENT</button>
      </div>
    </div>
  </div>
{/if}

<!-- OTA Modal -->
{#if showOtaModal}
  <div class="modal-overlay" onclick={() => showOtaModal = false} role="button" tabindex="-1">
    <div class="modal" style="max-width:800px;" onclick={(e) => e.stopPropagation()} role="dialog">
      <div class="modal-header">
        <span class="modal-title">OTA_FLASH_MANAGER — {otaNode}</span>
        <button onclick={() => showOtaModal = false} style="color:var(--text-muted);font-size:16px;">✕</button>
      </div>
      <div class="modal-body" style="display:flex;gap:20px;">
        <!-- Asset Library -->
        <div style="flex:1;">
          <div class="form-label" style="margin-bottom:10px;">ASSET_LIBRARY</div>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;max-height:240px;overflow-y:auto;">
            {#each files as f}
              <div
                style="border:1px solid {otaUrl.includes(f.url) ? 'var(--green)' : 'var(--border)'};border-radius:4px;padding:8px;cursor:pointer;background:var(--surface-2);transition:border-color 0.2s;"
                onclick={() => selectFileForOTA(f.url)}
                role="button"
                tabindex="0"
              >
                <div class="mono text-xs" style="color:var(--text);word-break:break-all;">{f.name}</div>
                <div class="text-muted" style="font-size:9px;margin-top:4px;">{formatBytes(f.size || 0)}</div>
                <button onclick={(e)=>{e.stopPropagation();handleDeleteFile(f.name);}} style="font-size:9px;color:var(--red);background:none;border:none;cursor:pointer;margin-top:4px;">DELETE</button>
              </div>
            {:else}
              <div class="text-muted mono text-xs" style="grid-column:span 2;padding:20px;text-align:center;">[NO_ASSETS_IN_VAULT]</div>
            {/each}
          </div>
          <!-- Upload -->
          <div style="margin-top:14px;padding-top:12px;border-top:1px solid var(--border);">
            <div class="form-label">UPLOAD_NEW_ASSET</div>
            <input type="file" multiple onchange={(e:any) => selectedFiles = Array.from(e.target.files)} style="margin:6px 0;width:100%;font-size:10px;" />
            <button class="btn btn-primary" style="font-size:10px;" onclick={handleUpload}>UPLOAD_TO_VAULT</button>
            {#if uploadProgress}
              <span class="mono text-xs" style="margin-left:10px;color:var(--green);">{uploadProgress}</span>
            {/if}
          </div>
        </div>

        <!-- Control Panel -->
        <div style="width:220px;display:flex;flex-direction:column;gap:14px;">
          <div>
            <div class="form-label">MANUAL TARGET URL</div>
            <input type="text" bind:value={otaUrl} placeholder="Pilih dari library atau masukkan URL" class="form-input" style="width:100%;font-size:10px;" />
          </div>
          <button class="btn btn-success" style="width:100%;margin-top:auto;" onclick={triggerOTA}>INITIATE_OTA_FLASH</button>
        </div>
      </div>
    </div>
  </div>
{/if}
