<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import Analytics from "./Analytics.svelte";

  const FORMING_BASE = "";
  type Module = "mdcw" | "sp" | "analytics";

  let loggedIn = $state(false);
  let username = $state("");
  let password = $state("");
  let token = $state("");
  let loginError = $state("");

  let activeModule = $state<Module>("mdcw");
  let theme = $state(localStorage.getItem("forming_theme") || "dark");

  let records: any[] = $state([]);
  let summaries: any[] = $state([]);
  let filters: string[] = $state([]);
  let activeFilter = $state("all");
  let statusFilter = $state("all");
  let sortBy = $state("newest");
  let startDate = $state("");
  let endDate = $state("");
  let isLoading = $state(false);

  let toasts: { id: number; type: string; msg: string }[] = $state([]);
  let toastId = 0;
  let pollInterval: ReturnType<typeof setInterval>;

  $effect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  });

  function toggleTheme() {
    theme = theme === "dark" ? "light" : "dark";
    localStorage.setItem("forming_theme", theme);
  }

  function moduleApi(path: string): string {
    return `/api/${activeModule}${path}`;
  }

  async function api(path: string, opts: RequestInit = {}) {
    const res = await fetch(FORMING_BASE + path, {
      ...opts,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: "Bearer " + token } : {}),
        ...(opts.headers || {}),
      },
    });
    if (res.status === 401) {
      loggedIn = false;
      return null;
    }
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  onMount(() => {
    const saved = localStorage.getItem("forming_token");
    if (saved) {
      token = saved;
      loggedIn = true;
      fetchData();
    }
  });
  onDestroy(() => pollInterval && clearInterval(pollInterval));

  async function handleLogin(e: Event) {
    e.preventDefault();
    loginError = "";
    try {
      const data = await api("/api/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      token = data.token;
      localStorage.setItem("forming_token", token);
      loggedIn = true;
      fetchData();
    } catch {
      loginError = "Username atau password salah.";
    }
  }

  async function handleLogout() {
    await api("/api/logout", { method: "POST" });
    token = "";
    localStorage.removeItem("forming_token");
    loggedIn = false;
  }

  async function switchModule(mod: Module) {
    activeModule = mod;
    if (mod === "analytics") {
      clearInterval(pollInterval);
      return;
    }
    activeFilter = "all";
    statusFilter = "all";
    sortBy = "newest";
    startDate = "";
    endDate = "";
    records = [];
    summaries = [];
    filters = [];
    await fetchData();
  }

  async function fetchData() {
    await fetchRecords();
    await fetchSummary();
    await fetchFilters();
    clearInterval(pollInterval);
    pollInterval = setInterval(fetchRecords, 15000);
  }

  async function fetchRecords() {
    isLoading = true;
    try {
      const params = new URLSearchParams();
      if (activeFilter !== "all") {
        params.set(activeModule === "mdcw" ? "prefix" : "session_id", activeFilter);
      }
      if (activeModule === "mdcw") {
        if (statusFilter !== "all") params.set("status", statusFilter);
        params.set("sort", sortBy);
      }
      if (startDate) params.set("start_date", startDate);
      if (endDate) params.set("end_date", endDate);
      const data = await api(moduleApi("/data?") + params);
      records = data || [];
    } finally {
      isLoading = false;
    }
  }

  async function fetchSummary() {
    try { summaries = (await api(moduleApi("/summary"))) || []; } catch {}
  }

  async function fetchFilters() {
    try {
      filters = (await api(moduleApi(activeModule === "mdcw" ? "/prefixes" : "/sessions"))) || [];
    } catch {}
  }

  function exportCSV() {
    if (!startDate || !endDate) {
      alert("Silakan pilih rentang tanggal (Start Date & End Date) pada menu filter terlebih dahulu sebelum mengekspor data CSV!");
      return;
    }
    const params = new URLSearchParams();
    if (activeFilter !== "all") {
      params.set(activeModule === "mdcw" ? "prefix" : "session_id", activeFilter);
    }
    if (activeModule === "mdcw" && statusFilter !== "all") params.set("status", statusFilter);
    if (startDate) params.set("start_date", startDate);
    if (endDate) params.set("end_date", endDate);
    const a = document.createElement("a");
    a.href = moduleApi("/export-csv?") + params;
    a.click();
  }

  function clearFilter() {
    activeFilter = "all"; statusFilter = "all"; sortBy = "newest"; startDate = ""; endDate = "";
    fetchRecords();
  }

  function statusLabel(reg5: number): { label: string; cls: string } {
    const map: Record<number, [string, string]> = {
      41: ["OK", "badge-green"], 521: ["OK", "badge-green"], 553: ["OK", "badge-green"],
      8: ["MATI", "badge-red"], 9: ["IDLE", "badge-yellow"], 90: ["IDLE", "badge-yellow"],
      8201: ["METAL", "badge-purple"], 25: ["UNDER", "badge-blue"], 73: ["OVER", "badge-yellow"],
    };
    const entry = map[reg5];
    return entry ? { label: entry[0], cls: entry[1] } : { label: `? (${reg5})`, cls: "badge-red" };
  }

  let mlStats = $derived.by(() => {
    if (activeModule !== "mdcw" || records.length === 0) return { validity: 0, testCount: 0, isenCount: 0, spamCount: 0, accuracy: 0 };
    const total = records.length;
    return {
      validity: (records.filter((r: any) => r.data_type === "VALID").length / total) * 100,
      testCount: records.filter((r: any) => r.data_type === "TEST").length,
      isenCount: records.filter((r: any) => r.data_type === "ISEN").length,
      spamCount: records.filter((r: any) => r.data_type === "SPAM").length,
      accuracy: (records.reduce((a: number, r: any) => a + (r.confidence || 0), 0) / total) * 100,
    };
  });
</script>

<svelte:head><title>Production Monitor</title></svelte:head>

<div class="toast-container">
  {#each toasts as t (t.id)}<div class="toast {t.type}">{t.msg}</div>{/each}
</div>

{#if !loggedIn}
  <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;background:var(--bg);">
    <div style="width:320px;">
      <div style="text-align:center;margin-bottom:28px;">
        <div style="font-family:var(--font-mono);font-size:20px;font-weight:700;color:var(--accent);">PRODUCTION_MONITOR</div>
        <div style="font-size:10px;color:var(--text-muted);margin-top:4px;">LINE INTELLIGENCE SYSTEM</div>
      </div>
      <div class="card">
        <form onsubmit={handleLogin}>
          <div class="form-group">
            <label class="form-label">OPERATOR</label>
            <input type="text" bind:value={username} required autocomplete="username" class="form-input" />
          </div>
          <div class="form-group">
            <label class="form-label">ACCESS CODE</label>
            <input type="password" bind:value={password} required autocomplete="current-password" class="form-input" />
          </div>
          {#if loginError}
            <div class="mono text-xs" style="color:var(--red);margin-bottom:8px;">{loginError}</div>
          {/if}
          <button type="submit" class="btn btn-primary" style="width:100%;justify-content:center;">LOGIN</button>
        </form>
      </div>
    </div>
  </div>
{:else}
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar-brand">PRODUKSI</div>
      <div class="nav-group">
        <button class="nav-tab" class:active={activeModule === "mdcw"} onclick={() => switchModule("mdcw")}>MDCW</button>
        <button class="nav-tab" class:active={activeModule === "sp"} onclick={() => switchModule("sp")}>SP</button>
        <button class="nav-tab" class:active={activeModule === "analytics"} onclick={() => switchModule("analytics")}>ANALYTICS</button>
      </div>
      <div class="topbar-right">
        <span class="live-dot">LIVE</span>
        <button class="theme-btn" onclick={toggleTheme} title="Toggle theme">{theme === "dark" ? "☀" : "☾"}</button>
        <button onclick={handleLogout} style="font-family:var(--font-mono);font-size:9px;color:var(--text-muted);background:none;border:none;cursor:pointer;">LOGOUT</button>
      </div>
    </header>

    {#if activeModule === "analytics"}
      <div style="padding:16px;width:100%;max-width:1400px;margin:0 auto;">
        <Analytics {api} {token} />
      </div>
    {:else}
      <main class="content main-grid-layout">
        <div class="side-col left-col">
          <div style="display:flex;flex-direction:column;gap:12px;margin-bottom:16px;">
            <div class="card">
              <div class="card-label" style="margin-bottom:8px;">
                {activeModule === "mdcw" ? "Mesin" : "Session"}
              </div>
              <div style="display:flex;gap:3px;flex-wrap:wrap;">
                <button class="btn" class:btn-primary={activeFilter === "all"} class:btn-ghost={activeFilter !== "all"}
                  style="font-size:9px;padding:4px 8px;" onclick={() => { activeFilter = "all"; fetchRecords(); }}>ALL</button>
                {#each filters as f}
                  <button class="btn" class:btn-primary={activeFilter === f} class:btn-ghost={activeFilter !== f}
                    style="font-size:9px;padding:4px 8px;" onclick={() => { activeFilter = f; fetchRecords(); }}>{f}</button>
                {/each}
              </div>
            </div>

            {#if activeModule === "mdcw"}
              <div class="card">
                <div class="card-label" style="margin-bottom:8px;">Filter</div>
                <div style="display:flex;flex-direction:column;gap:8px;">
                  <select bind:value={statusFilter} onchange={fetchRecords} style="font-size:10px;width:100%;">
                    <option value="all">Status: Semua</option>
                    <option value="ok">OK</option><option value="metal">Metal</option>
                    <option value="under">Under</option><option value="over">Over</option>
                    <option value="unknown">Unknown</option><option value="mati">Mati</option>
                    <option value="idle">Idle</option>
                  </select>
                  <select bind:value={sortBy} onchange={fetchRecords} style="font-size:10px;width:100%;">
                    <option value="newest">Terbaru</option>
                    <option value="weight_desc">Berat Max</option>
                    <option value="weight_asc">Berat Min</option>
                  </select>
                </div>
              </div>
            {/if}

            <div class="card">
              <div class="card-label" style="margin-bottom:8px;">Tanggal</div>
              <div style="display:flex;flex-direction:column;gap:8px;">
                <input type="date" bind:value={startDate} onchange={fetchRecords} style="font-size:10px;width:100%;" />
                <span style="color:var(--text-muted);text-align:center;font-size:9px;">s/d</span>
                <input type="date" bind:value={endDate} onchange={fetchRecords} style="font-size:10px;width:100%;" />
              </div>
            </div>

            <div class="card">
              <div style="display:flex;gap:6px;flex-wrap:wrap;">
                <button class="btn btn-success" onclick={exportCSV} style="font-size:9px;flex:1;justify-content:center;">CSV</button>
                <button class="btn btn-ghost" onclick={fetchRecords} style="font-size:9px;flex:1;justify-content:center;">↻</button>
                <button class="btn btn-ghost" onclick={clearFilter} style="font-size:9px;color:var(--red);flex:1;justify-content:center;">×</button>
              </div>
            </div>

            {#if activeModule === "mdcw"}
              <div class="card">
                <div class="card-label" style="margin-bottom:10px;">ML Insights</div>
                <div style="position:relative;height:80px;display:flex;flex-direction:column;align-items:center;justify-content:flex-end;">
                  <svg viewBox="0 0 100 50" style="width:120px;height:60px;">
                    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="var(--border)" stroke-width="6" stroke-linecap="round" />
                    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="var(--accent)" stroke-width="6" stroke-linecap="round"
                      stroke-dasharray="125.66" stroke-dashoffset={125.66 - 125.66 * (mlStats.validity / 100)}
                      style="transition:stroke-dashoffset 0.6s ease-out;" />
                  </svg>
                  <div style="position:absolute;bottom:0;text-align:center;">
                    <div style="font-size:16px;font-weight:800;color:var(--text);">{mlStats.validity.toFixed(0)}%</div>
                    <div style="font-size:8px;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.5px;">Valid</div>
                  </div>
                </div>
                <div style="display:flex;flex-direction:column;gap:3px;margin-top:8px;border-top:1px solid var(--border);padding-top:8px;font-size:9px;font-family:var(--font-mono);">
                  <div style="display:flex;justify-content:space-between;"><span style="color:var(--text-muted);">Test</span><span style="color:var(--accent-2);">{mlStats.testCount}</span></div>
                  <div style="display:flex;justify-content:space-between;"><span style="color:var(--text-muted);">Akurasi</span><span style="color:var(--green);">{mlStats.accuracy.toFixed(1)}%</span></div>
                  <div style="display:flex;justify-content:space-between;"><span style="color:var(--text-muted);">Abu"</span><span style="color:var(--yellow);">{mlStats.isenCount}</span></div>
                  <div style="display:flex;justify-content:space-between;"><span style="color:var(--text-muted);">Spam</span><span style="color:var(--red);">{mlStats.spamCount}</span></div>
                </div>
              </div>
            {/if}
          </div>
        </div>

        <div class="center-col">
          <div class="card">
            <div style="overflow-x:auto;">
              <table style="width:100%;border-collapse:collapse;">
                <thead>
                  <tr style="border-bottom:1px solid var(--border);">
                    {#if activeModule === "mdcw"}
                      {#each ["ID", "TS", "PREFIX", "BERAT", "PCK", "ST", "DT", "CONF"] as h}
                        <th style="text-align:left;padding:6px 10px;font-family:var(--font-mono);font-size:8px;font-weight:600;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.04em;">{h}</th>
                      {/each}
                    {:else}
                      {#each ["ID", "SESSION", "DATA", "TS"] as h}
                        <th style="text-align:left;padding:6px 10px;font-family:var(--font-mono);font-size:8px;font-weight:600;color:var(--text-muted);text-transform:uppercase;letter-spacing:0.04em;">{h}</th>
                      {/each}
                    {/if}
                  </tr>
                </thead>
                <tbody>
                  {#each records as r}
                    {#if activeModule === "mdcw"}
                      {@const st = statusLabel(r.reg5)}
                      {@const dtCls = r.data_type === "VALID" ? "badge-green" : r.data_type === "TEST" ? "badge-blue" : r.data_type === "SPAM" ? "badge-red" : "badge-yellow"}
                      <tr style="border-bottom:1px solid var(--border);transition:background 0.1s;"
                        onmouseenter={(e: any) => (e.currentTarget.style.background = "var(--surface-2)")}
                        onmouseleave={(e: any) => (e.currentTarget.style.background = "")}>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:9px;color:var(--text-muted);">{r.id}</td>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:9px;white-space:nowrap;">{r.ts}</td>
                        <td style="padding:6px 10px;"><span class="badge badge-blue" style="font-size:8px;">{r.prefix}</span></td>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:10px;font-weight:600;color:var(--purple);white-space:nowrap;">{r.weight_formatted}</td>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:10px;">{r.reg2}</td>
                        <td style="padding:6px 10px;"><span class="badge {st.cls}" style="font-size:8px;">{st.label}</span></td>
                        <td style="padding:6px 10px;"><span class="badge {dtCls}" style="font-size:8px;">{r.data_type}</span></td>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:8px;color:var(--text-muted);">{(r.confidence * 100).toFixed(0)}%</td>
                      </tr>
                    {:else}
                      <tr style="border-bottom:1px solid var(--border);transition:background 0.1s;"
                        onmouseenter={(e: any) => (e.currentTarget.style.background = "var(--surface-2)")}
                        onmouseleave={(e: any) => (e.currentTarget.style.background = "")}>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:9px;color:var(--text-muted);">{r.id}</td>
                        <td style="padding:6px 10px;"><span class="badge badge-blue" style="font-size:8px;">{r.session_id}</span></td>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:10px;font-weight:600;color:var(--accent-2);">{r.data}</td>
                        <td style="padding:6px 10px;font-family:var(--font-mono);font-size:9px;white-space:nowrap;">{r.ts || r.created_at}</td>
                      </tr>
                    {/if}
                  {:else}
                    <tr><td colspan="8" style="padding:32px;text-align:center;color:var(--text-muted);font-family:var(--font-mono);font-size:10px;">Tidak ada data</td></tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div class="right-col">
          <div class="summary-grid">
            {#each summaries as s}
              {#if activeModule === "mdcw"}
                <div class="card" style="border-left:2px solid var(--purple);padding:12px;">
                  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px;">
                    <span class="card-label" style="font-size:10px;font-weight:700;color:var(--text);">{s.prefix}</span>
                    <span style="font-size:9px;color:var(--text-muted);font-family:var(--font-mono);">{s.total_count || 0}</span>
                  </div>
                  <div style="font-size:18px;font-weight:700;color:var(--purple);font-family:var(--font-mono);margin-bottom:6px;">
                    {s.sum_weight !== undefined ? (s.sum_weight / 10).toFixed(1) + "g" : "-"}
                  </div>
                  {#if (s.total_count || 0) > 0}
                    {@const weigher = (s.under_count || 0) + (s.over_count || 0)}
                    {@const validTotal = (s.ok_count || 0) + weigher + (s.metal_count || 0)}
                    {#if validTotal > 0}
                      <div style="display:flex;height:4px;border-radius:2px;overflow:hidden;margin-bottom:6px;">
                        <div style="width:{((s.ok_count || 0) / validTotal) * 100}%;background:var(--green);"></div>
                        <div style="width:{(weigher / validTotal) * 100}%;background:var(--yellow);"></div>
                        <div style="width:{((s.metal_count || 0) / validTotal) * 100}%;background:var(--purple);"></div>
                      </div>
                    {/if}
                  {/if}
                  <div style="display:flex;justify-content:space-between;font-size:8px;color:var(--text-muted);border-top:1px solid var(--border);padding-top:6px;margin-top:4px;">
                    <span>OK: <span style="color:var(--green);">{s.ok_count || 0}</span></span>
                    <span>Weigher: <span style="color:var(--yellow);">{(s.under_count || 0) + (s.over_count || 0)}</span></span>
                    <span>Metal: <span style="color:var(--purple);">{s.metal_count || 0}</span></span>
                  </div>
                  <div style="display:flex;justify-content:space-between;font-size:8px;color:var(--text-muted);padding-top:3px;">
                    <span>Min: {s.min_weight ? (s.min_weight / 10).toFixed(1) + "g" : "-"}</span>
                    <span>Rata: {s.avg_weight ? (s.avg_weight / 10).toFixed(1) + "g" : "-"}</span>
                    <span>Max: {s.max_weight ? (s.max_weight / 10).toFixed(1) + "g" : "-"}</span>
                  </div>
                </div>
              {:else}
                <div class="card" style="border-left:2px solid var(--accent-2);padding:12px;">
                  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px;">
                    <span class="card-label" style="font-size:10px;font-weight:700;color:var(--text);">{s.session_id}</span>
                    <span style="font-size:9px;color:var(--text-muted);font-family:var(--font-mono);">{s.total_count || 0}</span>
                  </div>
                  <div style="font-size:18px;font-weight:700;color:var(--accent-2);font-family:var(--font-mono);margin-bottom:4px;">
                    {s.total_count || 0} scans
                  </div>
                  <div style="display:flex;justify-content:space-between;font-size:8px;color:var(--text-muted);border-top:1px solid var(--border);padding-top:6px;">
                    <span>First: {s.first_scan ? s.first_scan.slice(0,10) : "-"}</span>
                    <span>Last: {s.last_scan ? s.last_scan.slice(0,10) : "-"}</span>
                  </div>
                </div>
              {/if}
            {:else}
              <div class="card" style="text-align:center;padding:20px;color:var(--text-muted);font-family:var(--font-mono);font-size:10px;">
                Menunggu data produksi...
              </div>
            {/each}
          </div>
        </div>
      </main>
    {/if}
  </div>
{/if}

{#if isLoading}
  <div class="loading-bar"></div>
{/if}
