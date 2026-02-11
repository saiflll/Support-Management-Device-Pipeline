// static/app.js
// Megdev IoT Core - Runtime Logic
console.log("App.js loaded v3");

window.handleBtnClick = function (btn) {
  const action = btn.dataset.action;
  const node = btn.dataset.node || '';
  const name = btn.dataset.name || '';

  console.log('Button clicked:', action, node, name);

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
};

window._lastFilesJson = "";

async function fetchFiles() {
  try {
    const res = await fetch('/api/files');
    const files = await res.json();

    // Prevent "blink" if nothing changed
    const currentJson = JSON.stringify(files);
    if (currentJson === window._lastFilesJson) return;
    window._lastFilesJson = currentJson;

    const fileList = document.getElementById('fileList');
    fileList.innerHTML = '';

    if (!files || files.length === 0) {
      fileList.innerHTML = `
        <div class="col-span-full py-12 flex flex-col items-center justify-center border-2 border-dashed border-gray-800 rounded-lg opacity-40">
            <svg class="w-8 h-8 mb-2 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"></path></svg>
            <div class="text-[10px] uppercase font-bold tracking-widest text-gray-400">Inventory Empty</div>
        </div>`;
      return;
    }

    files.sort((a, b) => new Date(b.upload_time) - new Date(a.upload_time)).forEach(f => {
      const el = document.createElement('div');
      el.className = 'group relative glass-panel border border-gray-800 p-3 rounded hover:border-indigo-500/50 transition-all animate-fade-in flex flex-col justify-between h-[100px] bg-black/40';
      const fileUrl = `${location.origin}/files/${f.name}`;
      const uploadDate = new Date(f.upload_time).toLocaleDateString('en-GB', { day: '2-digit', month: 'short' });

      el.innerHTML = `
        <div class="flex justify-between items-start">
            <div class="flex items-center gap-2">
                <svg class="w-4 h-4 text-indigo-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"></path></svg>
                <div class="truncate pr-2">
                    <div class="text-[10px] font-bold text-gray-200 truncate" title="${escapeHtml(f.name)}">${escapeHtml(f.name)}</div>
                    <div class="text-[9px] text-gray-500 font-mono mt-0.5 uppercase tracking-tighter">${uploadDate} • ${formatBytes(f.size || 0)}</div>
                </div>
            </div>
            <button data-action="delete-file" data-name="${escapeHtml(f.name)}" class="p-1.5 text-gray-600 hover:text-red-500 hover:bg-red-500/10 rounded transition-all">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
            </button>
        </div>
        
        <div class="mt-auto">
            <button onclick="copyToClipboard('${fileUrl}', this)" class="w-full text-[9px] font-bold text-indigo-400/70 hover:text-indigo-300 bg-indigo-500/5 py-1 rounded border border-indigo-500/10 hover:border-indigo-500/30 transition-all font-mono">
                COPY_URL
            </button>
        </div>
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
      <div class="h-full flex flex-col">
        <!-- Header -->
        <div class="flex justify-between items-start mb-3 pb-2 border-b border-gray-800/50">
            <div class="flex items-center gap-2">
                <div class="relative flex h-2 w-2">
                  ${isOnline ? '<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>' : ''}
                  <span class="relative inline-flex rounded-full h-2 w-2 ${isOnline ? 'bg-emerald-500' : 'bg-gray-600'}"></span>
                </div>
                <div>
                    <div class="text-xs font-bold text-gray-200 leading-none tracking-wide">${info.model || 'UNKNOWN'}</div>
                    ${info.version ? `<div class="text-[9px] text-gray-500 font-mono mt-0.5">v${info.version}</div>` : ''}
                </div>
            </div>
            ${!isOnline ? `
            <button onclick="handleBtnClick(this)" data-action="delete-node" data-node="${k}" class="text-gray-600 hover:text-red-500 transition-colors" title="Purge Node">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-3.5 h-3.5">
                <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd" />
              </svg>
            </button>` : ''}
        </div>

        <!-- Info Grid -->
        <div class="space-y-2 mb-4 flex-1">
            <!-- ID Row -->
            <div class="flex items-center gap-2 group cursor-pointer" title="Double click to copy ID" ondblclick="copyToClipboard('${escapeHtml(k)}', this)">
                <svg class="w-3.5 h-3.5 text-gray-600 group-hover:text-indigo-400 transition-colors shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path></svg>
                <span class="text-[10px] font-mono text-indigo-300/90 truncate">${escapeHtml(k)}</span>
            </div>

            <!-- IP + RAM Row -->
            <div class="grid grid-cols-2 gap-2">
                <div class="flex items-center gap-2" title="IP Address">
                    <svg class="w-3.5 h-3.5 text-gray-600 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                    <span class="text-[10px] font-mono text-gray-400">${info.ip || '0.0.0.0'}</span>
                </div>
                 <div class="flex items-center gap-2" title="Free RAM">
                    <svg class="w-3.5 h-3.5 text-gray-600 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"></path></svg>
                    <span class="text-[10px] font-mono text-orange-300">${formatBytes(info.ram_free_bytes || 0)}</span>
                </div>
            </div>
            
            <!-- Config Context Row -->
            <div class="flex items-center gap-2 pt-2 border-t border-gray-800/30 mt-2">
                 <svg class="w-3.5 h-3.5 text-teal-600 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path></svg>
                 <span class="text-[10px] font-mono text-teal-500/80 truncate">
                    ${info.model?.startsWith('MDCW') ? `PREFIX: ${info.prefix || '-'}` : `CK: ${info.ck || '-'}`}
                 </span>
            </div>
        </div>

        <!-- Actions -->
        <div class="grid grid-cols-4 gap-2 mt-auto">
            <button onclick="handleBtnClick(this)" data-action="ota" data-node="${k}" class="h-6 rounded bg-gray-800 hover:bg-gray-700 hover:text-emerald-400 transition-all text-gray-500 border border-gray-700 flex items-center justify-center" title="OTA Update">
                 <span class="text-[9px] font-bold tracking-wider">OTA</span>
            </button>
            <button onclick="handleBtnClick(this)" data-action="configure" data-node="${k}" class="h-6 rounded bg-gray-800 hover:bg-gray-700 hover:text-indigo-400 transition-all text-gray-500 border border-gray-700 flex items-center justify-center" title="Configuration">
                <span class="text-[9px] font-bold tracking-wider">CFG</span>
            </button>
            <button onclick="handleBtnClick(this)" data-action="reboot" data-node="${k}" class="h-6 rounded bg-gray-800 hover:bg-gray-700 hover:text-yellow-400 transition-all text-gray-500 border border-gray-700 flex items-center justify-center" title="Reboot Device">
                <span class="text-[9px] font-bold tracking-wider">RBT</span>
            </button>
            <button onclick="handleBtnClick(this)" data-action="logs" data-node="${k}" class="h-6 rounded bg-gray-800 hover:bg-gray-700 hover:text-blue-400 transition-all text-gray-500 border border-gray-700 flex items-center justify-center" title="View Logs">
                <span class="text-[9px] font-bold tracking-wider">LOG</span>
            </button>
        </div>
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

  // Move upload listener before potentially crashing forwarder tools
  const uploadForm = document.getElementById('uploadForm');
  const fileInput = document.getElementById('fileInput');
  const fileNameLabel = document.getElementById('fileNameLabel');

  if (uploadForm) {
    uploadForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const file = fileInput.files[0];
      if (!file) {
        document.getElementById('uploadMsg').textContent = 'SELECT FILE FIRST.';
        return;
      }
      const fd = new FormData();
      fd.append('file', file);
      document.getElementById('uploadMsg').textContent = 'TRANSMITTING ASSET...';
      try {
        const res = await fetch('/upload', { method: 'POST', body: fd });
        if (res.ok) {
          document.getElementById('uploadMsg').textContent = 'PACKET SAVED.';
          uploadForm.reset();
          if (fileNameLabel) fileNameLabel.textContent = "SELECT_FIRMWARE";
          setTimeout(() => {
            document.getElementById('uploadMsg').textContent = '';
            document.getElementById('uploadFormContainer').classList.add('hidden');
            fetchFiles();
          }, 1500);
        } else {
          let errorMsg = 'SERVER ERROR.';
          try {
            const errData = await res.json();
            errorMsg = errData.error || errorMsg;
          } catch (e) {
            if (res.status === 401 || res.status === 403) errorMsg = 'AUTH REQUIRED.';
            if (res.status === 413) errorMsg = 'FILE TOO LARGE.';
          }
          document.getElementById('uploadMsg').textContent = `FAIL: ${errorMsg}`;
          console.error('Upload failed:', res.status, errorMsg);
        }
      } catch (err) {
        document.getElementById('uploadMsg').textContent = 'NETWORK FAILURE.';
        console.error('Network error during upload:', err);
      }
    });
  }

  // Forwarder Tools - Ensure elements exist before adding listeners
  const fFilter = document.getElementById('forwardFilter');
  const fRefresh = document.getElementById('btn-refresh-fwd');
  const fExport = document.getElementById('btn-export-csv');

  if (fFilter) fFilter.addEventListener('input', renderForwarderBuffer);
  if (fRefresh) fRefresh.addEventListener('click', typeof updateForwarder === 'function' ? updateForwarder : () => console.warn('updateForwarder not found'));
  if (fExport) fExport.addEventListener('click', exportForwarderCSV);

  // Delegation for static and dynamic elements using a safer approach
  document.addEventListener('click', async (e) => {
    // Traverse up to find the button
    const btn = e.target.closest('button');
    if (!btn || !btn.dataset.action) return;

    const action = btn.dataset.action;
    const node = btn.dataset.node || '';
    const name = btn.dataset.name || '';

    console.log('Action clicked:', action, node, name); // Debug

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

      let input;

      if (f.type === 'range') {
        // Dual Slider Logic (Min + Max)
        const wrapper = document.createElement('div');
        wrapper.className = 'w-full flexflex-col gap-1';

        const valuesDiv = document.createElement('div');
        valuesDiv.className = 'flex justify-between text-[9px] text-emerald-500 font-mono mb-1';
        const valMin = document.createElement('span');
        const valMax = document.createElement('span');
        valuesDiv.appendChild(valMin);
        valuesDiv.appendChild(valMax);

        const inputMin = document.createElement('input');
        inputMin.type = 'range';
        inputMin.min = f.min !== undefined ? f.min : -100;
        inputMin.max = f.max !== undefined ? f.max : 100;
        inputMin.step = f.step !== undefined ? f.step : 0.1;
        inputMin.className = 'w-full h-1 bg-gray-700 rounded-lg appearance-none cursor-pointer mb-1';

        const inputMax = document.createElement('input');
        inputMax.type = 'range';
        inputMax.min = f.min !== undefined ? f.min : -100;
        inputMax.max = f.max !== undefined ? f.max : 100;
        inputMax.step = f.step !== undefined ? f.step : 0.1;
        inputMax.className = 'w-full h-1 bg-gray-700 rounded-lg appearance-none cursor-pointer';

        // IDs for retrieval
        // Name format: range0 -> min0, max0
        const baseName = f.name.replace('range', ''); // "0"
        inputMin.id = `input-min${baseName}`;
        inputMax.id = `input-max${baseName}`;
        inputMin.dataset.group = f.name; // Tag for grouping
        inputMax.dataset.group = f.name;

        // Initial Values
        const currentMin = info[`min${baseName}`] !== undefined ? info[`min${baseName}`] : 0;
        const currentMax = info[`max${baseName}`] !== undefined ? info[`max${baseName}`] : 0;
        inputMin.value = currentMin;
        inputMax.value = currentMax;

        const updateLabels = () => {
          // Enforce Min <= Max
          if (parseFloat(inputMin.value) > parseFloat(inputMax.value)) {
            inputMin.value = inputMax.value;
          }
          valMin.textContent = `MIN: ${inputMin.value}`;
          valMax.textContent = `MAX: ${inputMax.value}`;
        };

        inputMin.addEventListener('input', updateLabels);
        inputMax.addEventListener('input', () => {
          if (parseFloat(inputMax.value) < parseFloat(inputMin.value)) {
            inputMax.value = inputMin.value;
          }
          updateLabels();
        });

        updateLabels();

        wrapper.appendChild(valuesDiv);
        wrapper.appendChild(inputMin);
        wrapper.appendChild(inputMax);

        fieldDiv.appendChild(label);
        fieldDiv.appendChild(wrapper);
        fieldsContainer.appendChild(fieldDiv);
        return;
      }

      if (f.type === 'slider') {
        // Wrapper for slider + value display
        const wrapper = document.createElement('div');
        wrapper.className = 'flex items-center gap-2 w-full';

        input = document.createElement('input');
        input.type = 'range';
        input.min = f.min !== undefined ? f.min : 0;
        input.max = f.max !== undefined ? f.max : 100;
        input.step = f.step !== undefined ? f.step : 1;
        input.className = 'flex-1 h-1 bg-gray-700 rounded-lg appearance-none cursor-pointer'; // Tailwind slider style

        const valDisplay = document.createElement('span');
        valDisplay.className = 'text-[10px] text-emerald-500 font-mono w-8 text-right';
        valDisplay.textContent = '0';

        input.addEventListener('input', () => {
          valDisplay.textContent = input.value;
        });

        wrapper.appendChild(input);
        wrapper.appendChild(valDisplay);

        // We attach the input to wrapper so we can find it later easily or just append wrapper
        input.id = `input-${f.name}`; // ID goes to input
        input.name = f.name;

        // Fill initial value
        let initialVal = 0;
        if (info[f.name] !== undefined) {
          initialVal = info[f.name];
        }
        input.value = initialVal;
        valDisplay.textContent = initialVal;

        fieldDiv.appendChild(label);
        fieldDiv.appendChild(wrapper);
        fieldsContainer.appendChild(fieldDiv);
        return; // Skip default input appending
      }

      input = document.createElement('input');
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
    // Handle standard inputs
    const input = document.getElementById(`input-${f.name}`);
    if (input) {
      let val = input.value;
      if (f.type === 'number') val = parseFloat(val);
      payload[f.name] = val;
    }

    // Handle range inputs (split back to min/max)
    if (f.type === 'range') {
      const baseName = f.name.replace('range', '');
      const minInput = document.getElementById(`input-min${baseName}`);
      const maxInput = document.getElementById(`input-max${baseName}`);
      if (minInput && maxInput) {
        payload[`min${baseName}`] = parseFloat(minInput.value);
        payload[`max${baseName}`] = parseFloat(maxInput.value);
      }
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

function renderForwarderBuffer() {
  const filter = document.getElementById('forwardFilter')?.value.toLowerCase() || '';
  const buffer = document.getElementById('data-buffer');
  if (!buffer) return;

  if (!window.currentBufferData || window.currentBufferData.length === 0) {
    buffer.innerHTML = '<div class="col-span-full py-20 text-center text-gray-700 italic opacity-50">[AWAITING_DATA_PACKETS]</div>';
    return;
  }

  const filtered = window.currentBufferData.filter(item => {
    const ck = String(item.ck || '').toLowerCase();
    const area = String(item.area || '').toLowerCase();
    return ck.includes(filter) || area.includes(filter);
  });

  if (filtered.length > 0) {
    buffer.innerHTML = filtered.slice(0, 12).map((item, i) => `
                        <div onclick="openBufferDetail(${i})" class="group relative bg-[#111113] border border-gray-800 p-3 rounded cursor-pointer hover:border-gray-600 hover:bg-gray-900/20 transition-all">
                             <div class="flex justify-between items-start mb-2">
                                <div class="flex items-center gap-1.5">
                                    <svg class="w-3 h-3 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>
                                    <span class="text-[10px] font-bold text-gray-300 group-hover:text-white">PACKET #${i + 1}</span>
                                </div>
                                <span class="text-[9px] text-gray-600 font-mono group-hover:text-emerald-400 transition-colors">READY</span>
                             </div>
                             
                             <div class="space-y-1.5 pt-2 border-t border-gray-800/50">
                                <div class="flex items-center justify-between text-[10px] font-mono">
                                     <div class="flex items-center gap-1 text-indigo-400">
                                        <svg class="w-3 h-3 opacity-70" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"></path></svg>
                                        <span>${item.ck || '?'} / ${item.area || '?'}</span>
                                     </div>
                                     <svg class="w-3 h-3 text-gray-700 group-hover:text-purple-500 transition-colors transform group-hover:translate-x-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3"></path></svg>
                                </div>
                                <div class="flex gap-2 text-[9px] text-gray-500">
                                   <span class="flex items-center gap-1 bg-gray-900 px-1 rounded">
                                     <span class="w-1.5 h-1.5 rounded-full bg-red-500/50"></span>
                                     TEMP: ${item.temp?.length || 0}
                                   </span>
                                   <span class="flex items-center gap-1 bg-gray-900 px-1 rounded">
                                     <span class="w-1.5 h-1.5 rounded-full bg-blue-500/50"></span>
                                     DOOR: ${item.door?.length || 0}
                                   </span>
                                </div>
                             </div>
                        </div>
                    `).join('');
  } else {
    buffer.innerHTML = '<div class="col-span-full py-20 text-center text-gray-700 italic opacity-50">[NO_MATCHING_DATA]</div>';
  }
}

function exportForwarderCSV() {
  if (!window.currentBufferData || window.currentBufferData.length === 0) {
    alert('NO_DATA_TO_EXPORT');
    return;
  }

  const headers = ['CK', 'AREA', 'TEMP_COUNT', 'DOOR_COUNT'];
  const rows = window.currentBufferData.map(item => [
    item.ck || '',
    item.area || '',
    item.temp?.length || 0,
    item.door?.length || 0
  ]);

  const csvContent = [headers, ...rows].map(e => e.join(",")).join("\n");
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");

  const now = new Date();
  const dateStr = now.toISOString().split('T')[0];
  link.setAttribute("href", url);
  link.setAttribute("download", `mdcw_export_${dateStr}.csv`);
  link.style.visibility = 'hidden';
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}
