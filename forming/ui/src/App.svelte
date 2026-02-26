<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  const FORMING_BASE = ''; // Same origin (port 3000)

  // ── State ─────────────────────────────────────────────────
  let loggedIn = $state(false);
  let username = $state('');
  let password = $state('');
  let token = $state('');
  let loginError = $state('');

  let records: any[] = $state([]);
  let summaries: any[] = $state([]);
  let prefixes: string[] = $state([]);
  let activeTab = $state('all');
  let statusFilter = $state('all');
  let sortBy = $state('newest');
  let startDate = $state('');
  let endDate = $state('');

  let toasts: { id: number; type: string; msg: string }[] = $state([]);
  let toastId = 0;

  let pollInterval: ReturnType<typeof setInterval>;

  // ── API Helper ────────────────────────────────────────────
  async function api(path: string, opts: RequestInit = {}) {
    const res = await fetch(FORMING_BASE + path, {
      ...opts,
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: 'Bearer ' + token } : {}),
        ...(opts.headers || {}),
      }
    });
    if (res.status === 401) { loggedIn = false; return null; }
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  // ── Auth ──────────────────────────────────────────────────
  onMount(() => {
    const saved = localStorage.getItem('forming_token');
    if (saved) { token = saved; loggedIn = true; fetchData(); }
  });
  onDestroy(() => pollInterval && clearInterval(pollInterval));

  async function handleLogin(e: Event) {
    e.preventDefault();
    loginError = '';
    try {
      const data = await api('/api/login', {
        method: 'POST',
        body: JSON.stringify({ username, password })
      });
      token = data.token;
      localStorage.setItem('forming_token', token);
      loggedIn = true;
      fetchData();
    } catch {
      loginError = 'Username atau password salah.';
    }
  }

  async function handleLogout() {
    await api('/api/logout', { method: 'POST' });
    token = '';
    localStorage.removeItem('forming_token');
    loggedIn = false;
  }

  // ── Data ──────────────────────────────────────────────────
  async function fetchData() {
    await fetchRecords();
    await fetchSummary();
    await fetchPrefixes();
    clearInterval(pollInterval);
    pollInterval = setInterval(fetchRecords, 15000);
  }

  async function fetchRecords() {
    try {
      const params = new URLSearchParams();
      if (activeTab !== 'all') params.set('prefix', activeTab);
      if (statusFilter !== 'all') params.set('status', statusFilter);
      params.set('sort', sortBy);
      if (startDate) params.set('start_date', startDate);
      if (endDate) params.set('end_date', endDate);
      const data = await api('/api/data?' + params);
      records = data || [];
    } catch {}
  }

  async function fetchSummary() {
    try {
      summaries = await api('/api/summary') || [];
    } catch {}
  }

  async function fetchPrefixes() {
    try {
      prefixes = await api('/api/prefixes') || [];
    } catch {}
  }

  function exportCSV() {
    const params = new URLSearchParams();
    if (activeTab !== 'all') params.set('prefix', activeTab);
    if (statusFilter !== 'all') params.set('status', statusFilter);
    if (startDate) params.set('start_date', startDate);
    if (endDate) params.set('end_date', endDate);
    const url = `/api/export-csv?${params}`;
    const a = document.createElement('a');
    a.href = url;
    a.click();
  }

  function toast(msg: string, type = 'info') {
    const id = ++toastId;
    toasts = [...toasts, { id, type, msg }];
    setTimeout(() => { toasts = toasts.filter(t => t.id !== id); }, 3500);
  }

  function statusLabel(reg5: number): { label: string; cls: string } {
    const map: Record<number, [string, string]> = {
      41: ['OK', 'badge-green'], 521: ['OK', 'badge-green'], 553: ['OK', 'badge-green'],
      8: ['MATI', 'badge-red'], 9: ['IDLE', 'badge-yellow'], 90: ['IDLE', 'badge-yellow'],
      8201: ['METAL', 'badge-purple'], 25: ['UNDER', 'badge-blue'], 73: ['OVER', 'badge-yellow']
    };
    const entry = map[reg5];
    if (entry) return { label: entry[0], cls: entry[1] };
    return { label: `? (${reg5})`, cls: 'badge-red' };
  }
</script>

<svelte:head><title>MDCW Production Monitor</title></svelte:head>

<!-- Toast -->
<div class="toast-container">
  {#each toasts as t (t.id)}<div class="toast {t.type}">{t.msg}</div>{/each}
</div>

{#if !loggedIn}
  <!-- Login Screen -->
  <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;background:var(--bg);">
    <div style="width:360px;">
      <div style="text-align:center;margin-bottom:32px;">
        <div style="font-family:var(--font-mono);font-size:22px;font-weight:700;color:var(--purple);">MDCW_MONITOR</div>
        <div style="font-size:11px;color:var(--text-muted);margin-top:6px;">PRODUCTION LINE INTELLIGENCE SYSTEM</div>
      </div>
      <div class="card">
        <form onsubmit={handleLogin}>
          <div class="form-group">
            <label class="form-label">OPERATOR_USERNAME</label>
            <input type="text" bind:value={username} required autocomplete="username" class="form-input" />
          </div>
          <div class="form-group">
            <label class="form-label">ACCESS_CODE</label>
            <input type="password" bind:value={password} required autocomplete="current-password" class="form-input" />
          </div>
          {#if loginError}
            <div class="mono text-xs text-red" style="margin-bottom:10px;">[ERR] {loginError}</div>
          {/if}
          <button type="submit" class="btn btn-primary" style="width:100%;">LOGIN_SYSTEM</button>
        </form>
      </div>
    </div>
  </div>
{:else}
  <!-- Main App -->
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar-brand">MDCW_PRODUKSI</div>
      <div style="display:flex;align-items:center;gap:16px;">
        <span class="live-dot">LIVE</span>
        <button onclick={handleLogout} style="font-family:var(--font-mono);font-size:10px;color:var(--text-muted);background:none;border:none;cursor:pointer;">[LOGOUT]</button>
      </div>
    </header>

    <main class="content">
      <!-- Summary Cards -->
      <div class="grid-4" style="margin-bottom:24px;">
        {#each summaries as s}
          <div class="card" style="border-left:3px solid var(--purple);">
            <div class="card-label">{s.prefix}</div>
            <div class="mono" style="font-size:20px;font-weight:700;color:var(--purple);margin:8px 0;">{s.total_count}</div>
            <div class="flex justify-between text-xs text-muted">
              <span>OK: {s.ok_count}</span>
              <span>Avg: {s.avg_weight !== undefined ? (s.avg_weight / 10).toFixed(1) + 'g' : '-'}</span>
            </div>
          </div>
        {:else}
          <div class="card" style="grid-column:span 4;text-align:center;padding:20px;color:var(--text-muted);font-family:var(--font-mono);font-size:11px;">[AWAITING PRODUCTION DATA]</div>
        {/each}
      </div>

      <!-- Filters & Controls -->
      <div class="card" style="margin-bottom:20px;">
        <div class="flex" style="gap:10px;flex-wrap:wrap;align-items:flex-end;">
          <!-- Prefix Tabs -->
          <div style="display:flex;gap:4px;flex-wrap:wrap;">
            <button class="btn" class:btn-primary={activeTab==='all'} class:btn-ghost={activeTab!=='all'} style="font-size:9px;padding:5px 10px;" onclick={()=>{activeTab='all';fetchRecords();}}>ALL</button>
            {#each prefixes as p}
              <button class="btn" class:btn-primary={activeTab===p} class:btn-ghost={activeTab!==p} style="font-size:9px;padding:5px 10px;" onclick={()=>{activeTab=p;fetchRecords();}}>{p}</button>
            {/each}
          </div>
          <select bind:value={statusFilter} onchange={fetchRecords} style="font-size:11px;">
            <option value="all">Status: Semua</option>
            <option value="ok">OK</option><option value="metal">Metal</option>
            <option value="under">Under</option><option value="over">Over</option>
            <option value="unknown">Unknown</option><option value="mati">Mati</option>
            <option value="idle">Idle</option>
          </select>
          <select bind:value={sortBy} onchange={fetchRecords} style="font-size:11px;">
            <option value="newest">Terbaru</option>
            <option value="weight_desc">Berat Max</option>
            <option value="weight_asc">Berat Min</option>
          </select>
          <input type="date" bind:value={startDate} onchange={fetchRecords} style="font-size:11px;" />
          <input type="date" bind:value={endDate} onchange={fetchRecords} style="font-size:11px;" />
          <button class="btn btn-success" onclick={exportCSV} style="font-size:10px;">EXPORT CSV</button>
          <button class="btn btn-ghost" onclick={fetchRecords} style="font-size:10px;">REFRESH</button>
        </div>
      </div>

      <!-- Data Table -->
      <div class="card">
        <div style="overflow-x:auto;">
          <table style="width:100%;border-collapse:collapse;">
            <thead>
              <tr style="border-bottom:1px solid var(--border);">
                {#each ['ID','TIMESTAMP','PREFIX','BERAT','PACK_CNT','STATUS'] as h}
                  <th style="text-align:left;padding:8px 12px;font-family:var(--font-mono);font-size:9px;font-weight:700;color:var(--text-muted);text-transform:uppercase;">{h}</th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each records as r}
                {@const st = statusLabel(r.reg5)}
                <tr style="border-bottom:1px solid var(--border);transition:background 0.1s;" onmouseenter={(e:any)=>e.currentTarget.style.background='var(--surface-2)'} onmouseleave={(e:any)=>e.currentTarget.style.background=''}>
                  <td style="padding:8px 12px;font-family:var(--font-mono);font-size:10px;color:var(--text-muted);">{r.id}</td>
                  <td style="padding:8px 12px;font-family:var(--font-mono);font-size:10px;">{r.ts}</td>
                  <td style="padding:8px 12px;"><span class="badge badge-blue">{r.prefix}</span></td>
                  <td style="padding:8px 12px;font-family:var(--font-mono);font-size:11px;font-weight:700;color:var(--purple);">{r.weight_formatted}</td>
                  <td style="padding:8px 12px;font-family:var(--font-mono);font-size:11px;">{r.reg2}</td>
                  <td style="padding:8px 12px;"><span class="badge {st.cls}">{st.label}</span></td>
                </tr>
              {:else}
                <tr><td colspan="6" style="padding:40px;text-align:center;color:var(--text-muted);font-family:var(--font-mono);font-size:11px;">[NO_DATA_IN_CURRENT_FILTER]</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </main>
  </div>
{/if}

<style>
  :global(body) { margin: 0; }
</style>
