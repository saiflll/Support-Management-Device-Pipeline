// static/app.js
// Megdev IoT Core - Runtime Logic

async function fetchFiles() {
  try {
    const res = await fetch('/api/files');
    const files = await res.json();
    const fileList = document.getElementById('fileList');
    fileList.innerHTML = '';
    if (!files || files.length === 0) {
      fileList.innerHTML = '<div class="text-[10px] text-slate-300 text-center py-12 italic uppercase tracking-[.25em] font-black opacity-60">Inventory Empty</div>';
      return;
    }
    files.forEach(f => {
      const el = document.createElement('div');
      el.className = 'flex items-center gap-1 animate-fade-in group';
      const fileUrl = `${location.origin}/files/${f.name}`;
      el.innerHTML = `
        <div class="btn-minimal text-[10px] flex items-center gap-2 cursor-pointer select-none" 
             title="Double click to copy link"
             ondblclick="copyToClipboard('${fileUrl}', this)">
          <span class="text-gray-500">FILE:</span>
          <span>${escapeHtml(f.name)}</span>
        </div>
        <button data-action="delete-file" data-name="${escapeHtml(f.name)}" 
                class="btn-minimal text-red-500 hover:bg-red-500/20 font-bold transition-all">
          [X]
        </button>
      `;
      fileList.appendChild(el);
    });
  } catch (err) { console.error(err); }
}
window.fetchFiles = fetchFiles;

async function fetchNodes() {
  try {
    const res = await fetch('/api/nodes');
    const nodes = await res.json();
    renderNodes(nodes);
  } catch (err) { console.error(err); }
}
window.fetchNodes = fetchNodes;

function renderNodes(nodes) {
  const runningArea = document.getElementById('runningNodes');
  const offlineArea = document.getElementById('offlineNodes');

  const processedIds = new Set();
  const sortedKeys = Object.keys(nodes).sort();

  let runningCount = 0;
  let offlineCount = 0;

  sortedKeys.forEach(k => {
    const info = nodes[k] || {};
    if (!info.model && !info.status) return;

    processedIds.add(k);
    const isOnline = info.status && info.status !== 'offline';
    if (isOnline) runningCount++; else offlineCount++;

    const cardId = `node-card-${encodeURIComponent(k)}`;
    let card = document.getElementById(cardId);
    const targetArea = isOnline ? runningArea : offlineArea;

    const metricsStr = info.metrics ? JSON.stringify(info.metrics) : '{}';
    const ram = info.metrics?.ram_free || 'N/A';
    const formattedTime = info.last_seen ? new Date(info.last_seen).toLocaleString() : 'Never';

    const cardHtml = `
      <div class="flex justify-between items-start mb-2">
        <span class="${isOnline ? 'text-emerald-500' : 'text-gray-600'} font-bold">${isOnline ? '●' : '○'} ${info.model || 'UNKNOWN'} ${info.version ? `<span class="text-[8px] opacity-70">v${info.version}</span>` : ''}</span>
        ${!isOnline ? `
        <button data-action="delete-node" data-node="${k}" class="text-red-500 hover:bg-red-900/30 px-1 rounded transition-colors" title="Purge Node">
          [DELETE]
        </button>` : ''}
      </div>
      <div class="space-y-0.5 font-mono text-[10px] leading-tight text-gray-400">
        <div><span class="json-key">"id"</span>: <span class="json-val-str">"${escapeHtml(k)}"</span>,</div>
        <div><span class="json-key">"ip"</span>: <span class="json-val-str">"${info.ip || '0.0.0.0'}"</span>,</div>
        <div><span class="json-key">"status"</span>: <span class="${isOnline ? 'json-val-str' : 'text-gray-600'}">"${info.status || 'unknown'}"</span>,</div>
        <div><span class="json-key">"metrics"</span>: { <span class="json-key">"ram"</span>: <span class="json-val-num">${formatBytes(info.ram_free_bytes || 0)}</span> },</div>
        <div><span class="json-key">"config"</span>: { ${info.model?.startsWith('MDCW') ? `"${info.prefix || '-'}"` : `"${info.ck || '-'}"`} }</div>
      </div>
      <div class="flex justify-center gap-3 mt-4 border-t border-gray-800 pt-3">
        <button data-action="ota" data-node="${k}" class="btn-minimal">[ ota ]</button>
        <button data-action="configure" data-node="${k}" class="btn-minimal">[ config ]</button>
        <button data-action="reboot" data-node="${k}" class="btn-minimal">[ reboot ]</button>
        <button data-action="logs" data-node="${k}" class="btn-minimal">[ log ]</button>
      </div>
    `;

    if (!card) {
      card = document.createElement('div');
      card.id = cardId;
      card.className = `node-card animate-fade-in`;
      card.innerHTML = cardHtml;
      targetArea.appendChild(card);
    } else {
      if (card.parentElement !== targetArea) {
        card.remove();
        targetArea.appendChild(card);
      }
      card.innerHTML = cardHtml;
    }
  });

  const allCards = document.querySelectorAll('.node-card');
  allCards.forEach(c => {
    const id = c.id.replace('node-card-', '');
    const found = processedIds.has(decodeURIComponent(id));
    if (!found) c.remove();
  });

  const prevRunningCount = parseInt(document.getElementById('count-running').textContent || '0');
  document.getElementById('count-running').textContent = runningCount;
  document.getElementById('count-offline').textContent = offlineCount;

  if (prevRunningCount > 0 && runningCount === 0 && document.getElementById('tab-running').classList.contains('active')) {
    document.getElementById('tab-offline').click();
  }

  if (runningCount === 0 && !runningArea.querySelector('.empty-msg')) {
    runningArea.innerHTML = '<div class="empty-msg col-span-full py-10 text-gray-700 italic text-[10px]">[NO_NODES_ACTIVE]</div>';
  } else if (runningCount > 0) {
    const msg = runningArea.querySelector('.empty-msg');
    if (msg) msg.remove();
  }

  if (offlineCount === 0 && !offlineArea.querySelector('.empty-msg')) {
    offlineArea.innerHTML = '<div class="empty-msg col-span-full py-10 text-gray-700 italic text-[10px]">[OFFLINE_REGISTRY_EMPTY]</div>';
  } else if (offlineCount > 0) {
    const msg = offlineArea.querySelector('.empty-msg');
    if (msg) msg.remove();
  }
}

function formatNodeId(id) {
  return escapeHtml(id);
}

function formatBytes(b) {
  if (b === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(b) / Math.log(k));
  return parseFloat((b / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// === Interaction Logic ===
document.addEventListener('DOMContentLoaded', () => {
  fetchFiles();
  fetchNodes();
  setInterval(fetchFiles, 4000);
  setInterval(fetchNodes, 4000);

  const uploadForm = document.getElementById('uploadForm');
  const fileInput = document.getElementById('fileInput');
  const fileNameLabel = document.getElementById('fileNameLabel');

  // Forwarder Tools
  document.getElementById('forwardFilter').addEventListener('input', renderForwarderBuffer);
  document.getElementById('btn-refresh-fwd').addEventListener('click', updateForwarder);
  document.getElementById('btn-export-csv').addEventListener('click', exportForwarderCSV);

  uploadForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const file = fileInput.files[0];
    if (!file) return;
    const fd = new FormData();
    fd.append('file', file);
    document.getElementById('uploadMsg').textContent = 'TRANSMITTING ASSET...';
    try {
      const res = await fetch('/upload', { method: 'POST', body: fd });
      if (res.ok) {
        document.getElementById('uploadMsg').textContent = 'PACKET SAVED.';
        uploadForm.reset();
        fileNameLabel.textContent = "NONE";
        setTimeout(() => {
          document.getElementById('uploadMsg').textContent = '';
          document.getElementById('uploadFormContainer').classList.add('hidden');
          fetchFiles();
        }, 1500);
      } else {
        document.getElementById('uploadMsg').textContent = 'ERROR IN TRANSMISSION.';
      }
    } catch (err) { document.getElementById('uploadMsg').textContent = 'NETWORK FAILURE.'; }
  });

  document.body.addEventListener('click', async (e) => {
    const btn = e.target.closest('[data-action]');
    if (!btn) return;

    const action = btn.dataset.action;
    const node = btn.dataset.node || '';
    const name = btn.dataset.name || '';

    if (action === 'ota') openOta(node);
    if (action === 'configure') openConfig(node);
    if (action === 'reboot') requestReboot(node);
    if (action === 'logs') openLogs(node);
    if (action === 'send-ota') sendOta();
    if (action === 'send-config') sendConfig();
    if (action === 'delete-file') confirmDeleteFile(name);
    if (action === 'delete-node') confirmDeleteNode(node);
    if (action === 'close-modal') document.getElementById('logModal').classList.add('hidden');
    if (action === 'close-config-modal') document.getElementById('configModal').classList.add('hidden');
    if (action === 'close-ota-modal') document.getElementById('otaModal').classList.add('hidden');
  });
});

async function openLogs(node) {
  const modal = document.getElementById('logModal');
  const body = document.getElementById('modalBody');
  body.innerHTML = 'AWAITING_STREAM...';
  modal.classList.remove('hidden');
  modal.classList.add('flex');
  try {
    const res = await fetch(`/logs/${encodeURIComponent(node)}`);
    const data = await res.json();
    body.innerHTML = (data.logs || []).map(l => {
      let color = 'text-emerald-500';
      if (l.startsWith('[MON]')) color = 'text-blue-400';
      if (l.startsWith('[LOG]')) color = 'text-emerald-500';
      return `<div class="${color} break-all">> ${escapeHtml(l)}</div>`;
    }).join('') || '[NO_LOGS_AVAILABLE]';
    // Auto scroll to bottom
    body.scrollTop = body.scrollHeight;
  } catch (err) { body.innerHTML = 'LOG_FETCH_ERROR'; }
}

function openOta(node) {
  window.currentOtaNode = node;
  document.getElementById('otaModal').classList.remove('hidden');
  document.getElementById('otaModal').classList.add('flex');
}

async function sendOta() {
  let url = document.getElementById('otaUrlInput').value;
  if (!url) return;

  // Deteksi localhost/127.0.0.1 - Masalah umum yang Anda alami
  if (url.includes('localhost') || url.includes('127.0.0.1')) {
    const serverIp = location.hostname === 'localhost' || location.hostname === '127.0.0.1' ?
      'MASUKKAN_IP_KOMPUTER_ANDA' : location.hostname;

    if (!confirm(`PERINGATAN: URL mengandung 'localhost'. ESP32 TIDAK BISA mendownload dari localhost.\n\nGanti dengan IP: ${serverIp}?\n\nKlik OK untuk melanjutkan apa adanya (mungkin gagal), atau Cancel untuk memperbaiki.`)) {
      return;
    }
  }

  const res = await fetch('/ota', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ node: window.currentOtaNode, url })
  });
  if (res.ok) {
    alert('OTA_COMMAND_DISPATCHED');
    document.getElementById('otaModal').classList.add('hidden');
  }
}

async function openConfig(node) {
  window.currentConfigNode = node;
  const modal = document.getElementById('configModal');
  const fieldsContainer = document.getElementById('dynamicConfigFields');
  fieldsContainer.innerHTML = '<div class="text-center py-4">[LOADING_MODEL_DEF]</div>';
  modal.classList.remove('hidden');
  modal.classList.add('flex');

  try {
    const nodesRes = await fetch('/api/nodes');
    const nodes = await nodesRes.json();
    const info = nodes[node] || {};

    // Extract variant from version (e.g., "1.0.0-M4" -> "M4")
    let specificModel = info.model || 'TEMP';
    if (info.version && info.version.includes('-')) {
      const parts = info.version.split('-');
      const variant = parts[parts.length - 1];
      if (variant) specificModel = variant;
    }

    let modelDef;
    try {
      const modelRes = await fetch(`/api/models/${specificModel}`);
      if (!modelRes.ok) throw new Error('Specific model not found');
      modelDef = await modelRes.json();
    } catch (e) {
      // Fallback to base model
      const baseModel = info.model || 'TEMP';
      const modelRes = await fetch(`/api/models/${baseModel}`);
      if (!modelRes.ok) throw new Error('Model definition not found');
      modelDef = await modelRes.json();
    }

    window.currentModelDef = modelDef;

    fieldsContainer.innerHTML = '';
    modelDef.fields.forEach(f => {
      const fieldDiv = document.createElement('div');
      fieldDiv.className = 'grid grid-cols-2 items-center gap-2 py-1 border-b border-gray-800/50';

      const label = document.createElement('label');
      label.className = 'text-[10px] text-gray-400 font-mono uppercase truncate';
      label.textContent = f.label;

      const input = document.createElement('input');
      input.id = `input-${f.name}`;
      input.name = f.name;
      input.type = f.type || 'text';
      input.placeholder = f.placeholder || f.label;
      input.className = 'w-full text-[11px] h-7 px-2';
      if (f.step) input.step = f.step;

      // Fill current value
      if (info[f.name] !== undefined) {
        input.value = info[f.name];
      } else if (f.name === 'prefix' && info.prefix) {
        input.value = info.prefix;
      }

      fieldDiv.appendChild(label);
      fieldDiv.appendChild(input);
      fieldsContainer.appendChild(fieldDiv);
    });
  } catch (err) {
    fieldsContainer.innerHTML = `<div class="text-red-500 text-center py-4">[ERROR: ${err.message}]</div>`;
  }
}

async function sendConfig() {
  if (!window.currentModelDef) return;

  const payload = { node: window.currentConfigNode };
  window.currentModelDef.fields.forEach(f => {
    const input = document.getElementById(`input-${f.name}`);
    if (input) {
      let val = input.value;
      if (f.type === 'number') val = parseFloat(val);
      payload[f.name] = val;
    }
  });

  const res = await fetch('/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (res.ok) {
    alert('CONFIG_DISPATCHED');
    document.getElementById('configModal').classList.add('hidden');
    fetchNodes();
  } else {
    const err = await res.json();
    alert('CONFIG_FAILED: ' + (err.error || 'Unknown'));
  }
}

async function confirmDeleteFile(name) {
  if (!name) return;
  if (confirm(`DELETE ${name}?`)) {
    try {
      const res = await fetch(`/api/files/${encodeURIComponent(name)}`, { method: 'DELETE' });
      if (res.ok) {
        fetchFiles();
      } else {
        const err = await res.json();
        alert('DELETE_FAILED: ' + (err.error || 'Unknown error'));
      }
    } catch (e) {
      alert('NETWORK_ERROR: ' + e.message);
    }
  }
}

async function confirmDeleteNode(node) {
  if (confirm('DISCARD DEVICE REGISTRY?')) {
    await fetch(`/api/nodes/${encodeURIComponent(node)}`, { method: 'DELETE' });
    fetchNodes();
  }
}

async function requestReboot(node) {
  if (!confirm(`REBOOT ${node}?`)) return;
  try {
    const res = await fetch('/reboot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node })
    });
    if (res.ok) {
      alert('REBOOT_COMMAND_SENT');
    } else {
      alert('REBOOT_FAILED');
    }
  } catch (e) { alert('NETWORK_ERROR'); }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/[&<>"']/g, m => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m]));
}

function copyToClipboard(text, el) {
  const original = el.innerHTML;

  const onSuccess = () => {
    el.innerHTML = '<span class="text-emerald-500">[COPIED_LINK]</span>';
    setTimeout(() => { el.innerHTML = original; }, 1000);
  };

  const onError = (err) => {
    console.error('Copy failed:', err);
    el.innerHTML = '<span class="text-red-500">[ERROR_COPY]</span>';
    setTimeout(() => { el.innerHTML = original; }, 1000);
  };

  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(onSuccess).catch(onError);
  } else {
    try {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.left = "-9999px";
      textArea.style.top = "0";
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      const successful = document.execCommand('copy');
      document.body.removeChild(textArea);
      if (successful) onSuccess(); else onError('Fallback failed');
    } catch (err) {
      onError(err);
    }
  }
}

