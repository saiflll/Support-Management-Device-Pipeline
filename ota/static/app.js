// static/app.js
// Frontend for IoT OTA & Monitor
// - Sends JSON to /set-threshold in this format:
//   { node, min, max, ck, area, no }
// - Uses prompt flow for inputs (min -> max -> ck -> area -> no)
// - Node IDs are shown as-is (server provides them)

async function fetchFiles() {
  try {
    const res = await fetch('/api/files');
    const files = await res.json();
    const fileList = document.getElementById('fileList');
    fileList.innerHTML = '';
    if (!files || files.length === 0) {
      fileList.innerHTML = '<div class="text-sm text-gray-400 text-center py-4">No files</div>';
      return;
    }
    files.forEach(f => {
      const el = document.createElement('div');
      el.className = 'file-card cyber-card card-red flex justify-between items-center p-3 rounded-xl transition-all';
      const uploadTime = f.upload_time ? new Date(f.upload_time).toLocaleString('en-US', { 
        year: 'numeric', 
        month: '2-digit', 
        day: '2-digit', 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit',
        hour12: true 
      }) : '';
      el.innerHTML = `
        <div class="flex-1 truncate pr-3">
          <div class="text-sm font-medium text-gray-200">${escapeHtml(f.name)}</div>
          <div class="text-xs text-gray-500 mt-0.5">${uploadTime}</div>
        </div>
        <div class="flex gap-1.5">
          <button data-action="rename-file" data-name="${encodeURIComponent(f.name)}" 
            class="p-2 rounded-lg border border-cyan-400/50 text-cyan-400 hover:bg-cyan-400 hover:text-black hover:scale-110 transition-all duration-200"
            title="Rename">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path>
            </svg>
          </button>
          <a href="${f.url}" target="_blank" 
            class="p-2 rounded-lg border border-purple-400/50 text-purple-400 hover:bg-purple-400 hover:text-black hover:scale-110 transition-all duration-200 inline-flex items-center justify-center"
            title="Download">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path>
            </svg>
          </a>
          <button data-action="delete-file" data-name="${encodeURIComponent(f.name)}" 
            class="p-2 rounded-lg border border-red-400/50 text-red-400 hover:bg-red-400 hover:text-black hover:scale-110 transition-all duration-200"
            title="Delete">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
            </svg>
          </button>
        </div>`;
      fileList.appendChild(el);
    });
  } catch (err) {
    console.error(err);
  }
}

function copyLink(url) {
  navigator.clipboard?.writeText(url).then(() => {
    showToast('Link copied!', 'success');
  }).catch(() => {
    // fallback
    const textarea = document.createElement('textarea');
    textarea.value = url;
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand('copy');
      showToast('Link copied!', 'success');
    } catch (err) {
      showToast('Failed to copy link.', 'error');
    }
    document.body.removeChild(textarea);
  });
}

async function renameFile(nameEnc) {
  const name = decodeURIComponent(nameEnc);
  const newName = prompt('Enter new name for ' + name);
  if (!newName || newName === name) return;

  const res = await fetch('/api/files/' + encodeURIComponent(name) + '/rename', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({new_name: newName})
  });
  const j = await res.json();
  if (res.ok) {
    showToast(`Renamed to: ${newName}`, 'success');
    fetchFiles();
  } else {
    showToast(`Rename failed: ${j.error || 'unknown'}`, 'error');
  }
}

async function deleteNode(node) {
  if (!confirm(`Delete node ${decodeURIComponent(node)}?`)) return;
  const res = await fetch('/api/nodes/' + node, { method: 'DELETE' });
  const j = await res.json();
  if (res.ok) {
    showToast(`Deleted node: ${decodeURIComponent(node)}`, 'success');
    fetchNodes();
  } else {
    showToast(`Delete failed: ${j.error || 'unknown'}`, 'error');
  }
}

async function deleteFile(nameEnc) {
  if (!confirm('Delete file?')) return;
  const name = decodeURIComponent(nameEnc);
  const res = await fetch('/api/files/' + encodeURIComponent(name), { method: 'DELETE' });
  const j = await res.json();
  if (res.ok) {
    showToast(`Deleted file: ${name}`, 'success');
    fetchFiles();
  } else {
    showToast(`Delete failed: ${j.error || 'unknown'}`, 'error');
  }
}

async function fetchNodes() {
  try {
    const res = await fetch('/api/nodes');
    const nodes = await res.json();
    renderNodes(nodes);
  } catch (err) {
    console.error(err);
  }
}

function renderNodes(nodes) {
  const runningArea = document.getElementById('runningNodes');
  const offlineArea = document.getElementById('offlineNodes');
  runningArea.innerHTML = '';
  offlineArea.innerHTML = '';

  const keys = Object.keys(nodes).sort();
  let runningCount = 0;
  let offlineCount = 0;

  if (keys.length === 0) {
    runningArea.innerHTML = '<div class="text-sm text-gray-400 md:col-span-2 text-center py-8">No nodes yet (waiting for MQTT messages)</div>';
    document.getElementById('count-running').textContent = '0';
    document.getElementById('count-offline').textContent = '0';
    return;
  }

  keys.forEach(k => {
    const formattedNodeId = formatNodeId(k);
    const info = nodes[k] || {};
    const status = info.status || '';
    const isOnline = String(status).toLowerCase() !== 'offline';
    const dot = isOnline ? 'bg-green-400' : 'bg-red-500';
    const ram = info.ram_free_bytes !== undefined ? formatBytes(info.ram_free_bytes) : '-';
    const sd_ok = info.sd_ok;
    const updated = info.updated || '';
    
    // Format waktu ke timezone lokal
    const formattedTime = updated ? new Date(updated).toLocaleString('en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: true
    }) : '-';

    const card = document.createElement('div');
    card.className = `node-card ${isOnline ? 'node-card-running' : 'node-card-offline'} cyber-card p-4 rounded-xl transition-all hover:scale-[1.02]`;
    
    card.innerHTML = `
      <div class="flex justify-between items-start mb-3">
        <div class="font-semibold text-lg truncate flex-1" title="${escapeHtml(k)}">${formattedNodeId}</div>
        <div class="flex items-center gap-2">
          ${sd_ok !== undefined && sd_ok !== null ? `
            <div class="flex items-center gap-1.5" title="SD Card Status">
              <div class="w-3 h-3 rounded-full ${sd_ok ? 'bg-green-400' : 'bg-red-500'}"></div>
            </div>
          ` : ''}
          <div class="w-3 h-3 rounded-full ${dot}"></div>
          <div class="text-sm ${isOnline ? 'text-orange-400' : 'text-gray-500'}">${escapeHtml(String(status))}</div>
        </div>
      </div>

      <div class="text-sm text-gray-300 space-y-1 mb-3">
        <div>RAM Free: <span class="text-gray-100 font-medium">${ram}</span></div>
        <div class="text-xs text-gray-500">Last: ${formattedTime}</div>
      </div>

      <div class="flex gap-2">
        <button data-action="ota" data-node="${encodeURIComponent(k)}" 
          class="flex-1 px-3 py-2 rounded-lg border border-cyan-400/50 text-cyan-400 hover:bg-cyan-400 hover:text-black text-sm font-medium transition-all hover:scale-105">
          OTA
        </button>
        <button data-action="configure" data-node="${encodeURIComponent(k)}" 
          class="flex-1 px-3 py-2 rounded-lg border border-purple-400/50 text-purple-400 hover:bg-purple-400 hover:text-black text-sm font-medium transition-all hover:scale-105">
          Config
        </button>
        <button data-action="logs" data-node="${encodeURIComponent(k)}" 
          class="flex-1 px-3 py-2 rounded-lg border border-orange-400/50 text-orange-400 hover:bg-orange-400 hover:text-black text-sm font-medium transition-all hover:scale-105">
          Logs
        </button>
        <button data-action="delete-node" data-node="${encodeURIComponent(k)}" 
          class="px-3 py-2 rounded-lg border border-red-400/50 text-red-400 hover:bg-red-400 hover:text-black text-sm font-medium transition-all hover:scale-105">
          ✕
        </button>
      </div>
    `;

    if (isOnline) {
      runningArea.appendChild(card);
      runningCount++;
    } else {
      offlineArea.appendChild(card);
      offlineCount++;
    }
  });

  document.getElementById('count-running').textContent = runningCount;
  document.getElementById('count-offline').textContent = offlineCount;
  if (runningCount === 0) runningArea.innerHTML = '<div class="text-sm text-gray-400 md:col-span-2 text-center py-8">No running nodes.</div>';
  if (offlineCount === 0) offlineArea.innerHTML = '<div class="text-sm text-gray-400 md:col-span-2 text-center py-8">No offline nodes.</div>';
}

function formatBytes(bytes) {
  if (!bytes || bytes == 0) return '0 B';
  const kb = 1024;
  if (bytes < kb) return bytes + ' B';
  if (bytes < kb * kb) return Math.round(bytes / kb) + ' KB';
  return Math.round(bytes / (kb * kb)) + ' MB';
}

function formatNodeId(nodeId) {
  if (!nodeId) return '';

  // The MAC address is always the last 12 characters.
  if (nodeId.length < 12) {
    return escapeHtml(nodeId);
  }

  const mac = nodeId.slice(-12);
  let prefix = nodeId.slice(0, -12);

  // Format the prefix: replace hyphens and remove any trailing slash
  prefix = prefix.replace(/-/g, '/').replace(/\/$/, '');

  // Format MAC with colons
  const formattedMac = mac.match(/.{1,2}/g)?.join(':') || mac;

  return `${escapeHtml(prefix)} - <span class="text-indigo-300">${escapeHtml(formattedMac)}</span>`;
}

// === Config flow with Modal ===
let currentConfigNode = '';

async function openConfigModal(nodeEnc, action = 'Config') {
  currentConfigNode = nodeEnc;
  const node = decodeURIComponent(nodeEnc);
  
  // Get current node data to pre-fill
  const nodesRes = await fetch('/api/nodes');
  const allNodes = await nodesRes.json();
  const nodeInfo = allNodes[node];
  
  document.getElementById('configModalNode').textContent = node;
  document.getElementById('configMinInput').value = nodeInfo?.min || '16';
  document.getElementById('configMaxInput').value = nodeInfo?.max || '20';
  document.getElementById('configCkInput').value = nodeInfo?.ck || '';
  document.getElementById('configAreaInput').value = nodeInfo?.area || '';
  document.getElementById('configNoInput').value = nodeInfo?.no || '';
  
  const modal = document.getElementById('configModal');
  modal.classList.remove('hidden');
  modal.classList.add('flex');
}

function closeConfigModal() {
  const modal = document.getElementById('configModal');
  modal.classList.add('hidden');
  modal.classList.remove('flex');
  currentConfigNode = '';
}

async function sendConfig() {
  const node = decodeURIComponent(currentConfigNode);
  const payload = {
    node: node,
    min: parseFloat(document.getElementById('configMinInput').value),
    max: parseFloat(document.getElementById('configMaxInput').value),
    ck: document.getElementById('configCkInput').value,
    area: document.getElementById('configAreaInput').value,
    no: document.getElementById('configNoInput').value
  };
  
  try {
    showToast('Sending Config...');
    const res = await fetch('/config', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify(payload)
    });
    const j = await res.json();
    if (res.ok) {
      showToast('Config sent successfully!', 'success');
      closeConfigModal();
      setTimeout(fetchNodes, 800);
    } else {
      showToast(`Failed to send Config: ${j.error || 'Unknown error'}`, 'error');
    }
  } catch (err) {
    showToast(`Network error: ${err.message}`, 'error');
  }
}

/* The old openThresholdModal and promptConfigFlow functions have been removed. */


// OTA modal
let currentOtaNode = '';

async function openOTAModal(nodeEnc) {
  currentOtaNode = nodeEnc;
  const node = decodeURIComponent(nodeEnc);
  const urlDefault = location.origin + '/files/';
  
  document.getElementById('otaModalNode').textContent = node;
  document.getElementById('otaUrlInput').value = '';
  document.getElementById('otaUrlInput').placeholder = urlDefault + 'firmware.bin';
  
  const modal = document.getElementById('otaModal');
  modal.classList.remove('hidden');
  modal.classList.add('flex');
}

function closeOtaModal() {
  const modal = document.getElementById('otaModal');
  modal.classList.add('hidden');
  modal.classList.remove('flex');
  currentOtaNode = '';
}

async function sendOta() {
  const node = decodeURIComponent(currentOtaNode);
  const url = document.getElementById('otaUrlInput').value.trim();
  
  if (!url) {
    return showToast('Please enter a valid URL', 'error');
  }
  
  try {
    showToast('Sending OTA command...');
    const res = await fetch('/ota', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({node, url})
    });
    const j = await res.json();
    if (res.ok) {
      showToast('OTA command sent!', 'success');
      closeOtaModal();
    } else {
      showToast(`OTA failed: ${j.error || 'Unknown error'}`, 'error');
    }
  } catch (err) {
    showToast(`Network error: ${err.message}`, 'error');
  }
}

// LOG modal
function openLogModal(nodeEnc) {
  const node = decodeURIComponent(nodeEnc);
  document.getElementById('modalNode').textContent = node;
  const modal = document.getElementById('logModal');
  const body = document.getElementById('modalBody');
  body.textContent = 'Loading...';
  modal.classList.remove('hidden');
  modal.classList.add('flex');

  fetch('/logs/' + encodeURIComponent(node))
    .then(r => {
      if (!r.ok) throw new Error('No logs');
      return r.json();
    })
    .then(j => {
      const logs = j.logs || [];
      if (logs.length === 0) {
        body.innerHTML = '<div class="text-sm text-slate-400">No logs</div>';
      } else {
        body.innerHTML = logs.map(l => `<div class="mb-1 text-xs text-slate-200">▶ ${escapeHtml(l)}</div>`).join('');
      }
    }).catch(err => {
      body.innerHTML = '<div class="text-sm text-slate-400">No logs / node not found</div>';
    });
}

function closeModal() {
  const modal = document.getElementById('logModal');
  modal.classList.add('hidden');
  modal.classList.remove('flex');
}

// simple escape to avoid HTML injection
function escapeHtml(unsafe) {
  if (!unsafe) return '';
  return String(unsafe)
       .replaceAll('&', '&amp;')
       .replaceAll('<', '&lt;')
       .replaceAll('>', '&gt;')
       .replaceAll('"', '&quot;')
       .replaceAll("'", '&#039;');
}

// Toast notification function
function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  const colors = {
    info: 'bg-blue-500',
    success: 'bg-emerald-500',
    error: 'bg-red-500',
  };
  toast.className = `fixed bottom-5 right-5 px-4 py-2 rounded-md text-white shadow-lg transition-opacity duration-300 ${colors[type] || colors.info}`;
  toast.textContent = message;
  document.body.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

// upload form
document.addEventListener('DOMContentLoaded', function() {
  const form = document.getElementById('uploadForm');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const file = document.getElementById('fileInput').files[0];
    if (!file) return alert('Pilih file terlebih dahulu');
    const fd = new FormData();
    fd.append('file', file);
    document.getElementById('uploadMsg').textContent = 'Uploading...';
    try {
      const res = await fetch('/upload', { method: 'POST', body: fd });
      if (res.redirected) {
        document.getElementById('uploadMsg').textContent = 'Upload OK';
      } else {
        document.getElementById('uploadMsg').textContent = 'Upload finished';
      }
      setTimeout(()=> fetchFiles(), 800);
    } catch (err) {
      document.getElementById('uploadMsg').textContent = 'Upload error';
      console.error(err);
    }
  });

  // Centralized event listener for all actions
  document.body.addEventListener('click', (e) => {
    const button = e.target.closest('[data-action]');
    if (!button) return;

    const action = button.dataset.action;
    const node = button.dataset.node;
    const name = button.dataset.name;
    const url = button.dataset.url;

    switch (action) {
      case 'ota':
        openOTAModal(node);
        break;
      case 'configure':
        openConfigModal(node, 'Configure');
        break;
      case 'logs':
        openLogModal(node);
        break;
      case 'delete-node':
        deleteNode(node);
        break;
      case 'copy-link':
        copyLink(url);
        break;
      case 'rename-file':
        renameFile(name);
        break;
      case 'delete-file':
        deleteFile(name);
        break;
      case 'close-modal':
        closeModal();
        break;
      case 'close-ota-modal':
        closeOtaModal();
        break;
      case 'send-ota':
        sendOta();
        break;
      case 'close-config-modal':
        closeConfigModal();
        break;
      case 'send-config':
        sendConfig();
        break;
    }
  });

  // Tab switching logic
  const tabs = document.querySelectorAll('.tab-button');
  const tabContents = document.querySelectorAll('.tab-content');
  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      // Deactivate all tabs
      tabs.forEach(t => {
        t.classList.remove('border-indigo-500', 'text-indigo-400');
        t.classList.add('border-transparent', 'text-slate-400', 'hover:text-slate-200', 'hover:border-slate-400');
      });
      // Deactivate all content
      tabContents.forEach(c => c.classList.add('hidden'));

      // Activate clicked tab
      tab.classList.add('border-indigo-500', 'text-indigo-400');
      tab.classList.remove('border-transparent', 'text-slate-400', 'hover:text-slate-200', 'hover:border-slate-400');
      
      // Activate corresponding content
      const targetContentId = tab.id.replace('tab-', '') + 'Nodes';
      document.getElementById(targetContentId).classList.remove('hidden');
    });
  });

  // initial load + interval
  fetchFiles();
  fetchNodes();
  setInterval(fetchFiles, 5000);
  setInterval(fetchNodes, 5000);
});
