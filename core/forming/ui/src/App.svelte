<script lang="ts">
  import { onMount, onDestroy } from "svelte";

  const FORMING_BASE = ""; // Same origin (port 3000)

  // ── State ─────────────────────────────────────────────────
  let loggedIn = $state(false);
  let username = $state("");
  let password = $state("");
  let token = $state("");
  let loginError = $state("");

  let records: any[] = $state([]);
  let summaries: any[] = $state([]);
  let prefixes: string[] = $state([]);
  let activeTab = $state("all");
  let statusFilter = $state("all");
  let sortBy = $state("newest");
  let startDate = $state("");
  let endDate = $state("");
  let isLoading = $state(false);

  let toasts: { id: number; type: string; msg: string }[] = $state([]);
  let toastId = 0;

  let pollInterval: ReturnType<typeof setInterval>;

  // ── API Helper ────────────────────────────────────────────
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

  // ── Auth ──────────────────────────────────────────────────
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

  // ── Data ──────────────────────────────────────────────────
  async function fetchData() {
    await fetchRecords();
    await fetchSummary();
    await fetchPrefixes();
    clearInterval(pollInterval);
    pollInterval = setInterval(fetchRecords, 15000);
  }

  async function fetchRecords() {
    isLoading = true;
    try {
      const params = new URLSearchParams();
      if (activeTab !== "all") params.set("prefix", activeTab);
      if (statusFilter !== "all") params.set("status", statusFilter);
      params.set("sort", sortBy);
      if (startDate) params.set("start_date", startDate);
      if (endDate) params.set("end_date", endDate);
      const data = await api("/api/data?" + params);
      records = data || [];
    } finally {
      isLoading = false;
    }
  }

  async function fetchSummary() {
    try {
      summaries = (await api("/api/summary")) || [];
    } catch {}
  }

  async function fetchPrefixes() {
    try {
      prefixes = (await api("/api/prefixes")) || [];
    } catch {}
  }

  function exportCSV() {
    if (!startDate || !endDate) {
      alert(
        "Silakan pilih rentang tanggal (Start Date & End Date) pada menu filter terlebih dahulu sebelum mengekspor data CSV!",
      );
      return;
    }
    const params = new URLSearchParams();
    if (activeTab !== "all") params.set("prefix", activeTab);
    if (statusFilter !== "all") params.set("status", statusFilter);
    if (startDate) params.set("start_date", startDate);
    if (endDate) params.set("end_date", endDate);
    const url = `/api/export-csv?${params}`;
    const a = document.createElement("a");
    a.href = url;
    a.click();
  }

  function clearFilter() {
    activeTab = "all";
    statusFilter = "all";
    sortBy = "newest";
    startDate = "";
    endDate = "";
    fetchRecords();
  }

  function toast(msg: string, type = "info") {
    const id = ++toastId;
    toasts = [...toasts, { id, type, msg }];
    setTimeout(() => {
      toasts = toasts.filter((t) => t.id !== id);
    }, 3500);
  }

  function statusLabel(reg5: number): { label: string; cls: string } {
    const map: Record<number, [string, string]> = {
      41: ["OK", "badge-green"],
      521: ["OK", "badge-green"],
      553: ["OK", "badge-green"],
      8: ["MATI", "badge-red"],
      9: ["IDLE", "badge-yellow"],
      90: ["IDLE", "badge-yellow"],
      8201: ["METAL", "badge-purple"],
      25: ["UNDER", "badge-blue"],
      73: ["OVER", "badge-yellow"],
    };
    const entry = map[reg5];
    if (entry) return { label: entry[0], cls: entry[1] };
    return { label: `? (${reg5})`, cls: "badge-red" };
  }

  // ── ML Insights Metrics ──────────────────────────────────
  let mlStats = $derived.by(() => {
    if (records.length === 0) {
      return {
        validity: 0,
        testCount: 0,
        isenCount: 0,
        spamCount: 0,
        accuracy: 0,
      };
    }
    const total = records.length;
    const valid = records.filter((r) => r.data_type === "VALID").length;
    const test = records.filter((r) => r.data_type === "TEST").length;
    const isen = records.filter((r) => r.data_type === "ISEN").length;
    const spam = records.filter((r) => r.data_type === "SPAM").length;
    const avgConf =
      records.reduce((acc, r) => acc + (r.confidence || 0), 0) / total;

    return {
      validity: (valid / total) * 100,
      testCount: test,
      isenCount: isen,
      spamCount: spam,
      accuracy: avgConf * 100,
    };
  });
</script>

<svelte:head><title>MDCW Production Monitor</title></svelte:head>

<!-- Toast -->
<div class="toast-container">
  {#each toasts as t (t.id)}<div class="toast {t.type}">{t.msg}</div>{/each}
</div>

{#if !loggedIn}
  <!-- Login Screen -->
  <div
    style="min-height:100vh;display:flex;align-items:center;justify-content:center;background:var(--bg);"
  >
    <div style="width:360px;">
      <div style="text-align:center;margin-bottom:32px;">
        <div
          style="font-family:var(--font-mono);font-size:22px;font-weight:700;color:var(--purple);"
        >
          MDCW_MONITOR
        </div>
        <div style="font-size:11px;color:var(--text-muted);margin-top:6px;">
          PRODUCTION LINE INTELLIGENCE SYSTEM
        </div>
      </div>
      <div class="card">
        <form onsubmit={handleLogin}>
          <div class="form-group">
            <label class="form-label">OPERATOR_USERNAME</label>
            <input
              type="text"
              bind:value={username}
              required
              autocomplete="username"
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label class="form-label">ACCESS_CODE</label>
            <input
              type="password"
              bind:value={password}
              required
              autocomplete="current-password"
              class="form-input"
            />
          </div>
          {#if loginError}
            <div class="mono text-xs text-red" style="margin-bottom:10px;">
              [ERR] {loginError}
            </div>
          {/if}
          <button type="submit" class="btn btn-primary" style="width:100%;"
            >LOGIN_SYSTEM</button
          >
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
        <button
          onclick={handleLogout}
          style="font-family:var(--font-mono);font-size:10px;color:var(--text-muted);background:none;border:none;cursor:pointer;"
          >[LOGOUT]</button
        >
      </div>
    </header>

    <main class="content main-grid-layout">
      <!-- ================= SISI KIRI (3): FILTER BLOK ================= -->
      <div class="side-col left-col">
        <div
          style="display: flex; flex-direction: column; gap: 16px; margin-bottom: 24px;"
        >
          <!-- Filter Block: Mesin -->
          <div
            style="border: 1px solid var(--border); border-radius: 8px; padding: 12px; background: var(--surface); display: flex; flex-direction: column; gap: 10px;"
          >
            <div
              style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); font-weight: 700; text-transform: uppercase; display: flex; align-items: center; gap: 8px;"
            >
              <svg
                width="14"
                height="14"
                fill="none"
                stroke="var(--accent-2)"
                stroke-width="2"
                viewBox="0 0 24 24"
                ><path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                ></path></svg
              >
              Mesin / MDCW
            </div>
            <div style="display:flex;gap:4px;flex-wrap:wrap;">
              <button
                class="btn"
                class:btn-primary={activeTab === "all"}
                class:btn-ghost={activeTab !== "all"}
                style="font-size:10px;padding:5px 10px;"
                onclick={() => {
                  activeTab = "all";
                  fetchRecords();
                }}>ALL</button
              >
              {#each prefixes as p}
                <button
                  class="btn"
                  class:btn-primary={activeTab === p}
                  class:btn-ghost={activeTab !== p}
                  style="font-size:10px;padding:5px 10px;"
                  onclick={() => {
                    activeTab = p;
                    fetchRecords();
                  }}>{p}</button
                >
              {/each}
            </div>
          </div>

          <!-- Filter Block: Status & Sort -->
          <div
            style="border: 1px solid var(--border); border-radius: 8px; padding: 12px; background: var(--surface); display: flex; flex-direction: column; gap: 10px;"
          >
            <div
              style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); font-weight: 700; text-transform: uppercase; display: flex; align-items: center; gap: 8px;"
            >
              <svg
                width="14"
                height="14"
                fill="none"
                stroke="var(--yellow)"
                stroke-width="2"
                viewBox="0 0 24 24"
                ><path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
                ></path></svg
              >
              Filter & Sortir
            </div>
            <div style="display:flex;flex-direction:column;gap:10px;">
              <select
                bind:value={statusFilter}
                onchange={fetchRecords}
                style="font-size:11px; width:100%;"
              >
                <option value="all">Status: Semua</option>
                <option value="ok">OK</option><option value="metal"
                  >Metal</option
                >
                <option value="under">Under</option><option value="over"
                  >Over</option
                >
                <option value="unknown">Unknown</option><option value="mati"
                  >Mati</option
                >
                <option value="idle">Idle</option>
              </select>
              <select
                bind:value={sortBy}
                onchange={fetchRecords}
                style="font-size:11px; width:100%;"
              >
                <option value="newest">Terbaru</option>
                <option value="weight_desc">Berat Max</option>
                <option value="weight_asc">Berat Min</option>
              </select>
            </div>
          </div>

          <!-- Filter Block: Tanggal -->
          <div
            style="border: 1px solid var(--border); border-radius: 8px; padding: 12px; background: var(--surface); display: flex; flex-direction: column; gap: 10px;"
          >
            <div
              style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); font-weight: 700; text-transform: uppercase; display: flex; align-items: center; gap: 8px;"
            >
              <svg
                width="14"
                height="14"
                fill="none"
                stroke="var(--purple)"
                stroke-width="2"
                viewBox="0 0 24 24"
                ><path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
                ></path></svg
              >
              Rentang Waktu
            </div>
            <div style="display:flex;flex-direction:column;gap:10px;">
              <input
                type="date"
                bind:value={startDate}
                onchange={fetchRecords}
                style="font-size:11px; width: 100%;"
              />
              <span style="color:var(--text-muted);text-align:center;"
                >- to -</span
              >
              <input
                type="date"
                bind:value={endDate}
                onchange={fetchRecords}
                style="font-size:11px; width: 100%;"
              />
            </div>
          </div>

          <!-- Filter Block: Aksi -->
          <div
            style="border: 1px solid var(--border); border-radius: 8px; padding: 12px; background: var(--surface); display: flex; flex-direction: column; gap: 10px;"
          >
            <div
              style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); font-weight: 700; text-transform: uppercase; display: flex; align-items: center; gap: 8px;"
            >
              <svg
                width="14"
                height="14"
                fill="none"
                stroke="var(--green)"
                stroke-width="2"
                viewBox="0 0 24 24"
                ><path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                ></path></svg
              >
              Aksi Data
            </div>
            <div style="display:flex;gap:8px; flex-wrap:wrap;">
              <button
                class="btn btn-success"
                onclick={exportCSV}
                style="font-size:10px;flex:1;justify-content:center;"
                >EXPORT</button
              >
              <button
                class="btn btn-ghost"
                onclick={fetchRecords}
                style="font-size:10px;flex:1;justify-content:center;"
                >REFRESH</button
              >
              <button
                class="btn btn-ghost"
                onclick={clearFilter}
                style="font-size:10px;color:var(--red);flex:1;justify-content:center;"
                >CLEAR</button
              >
            </div>
          </div>

          <!-- ML Wisdom Block: Analytics -->
          <div
            style="border: 1px solid var(--border); border-radius: 8px; padding: 16px; background: var(--surface); display: flex; flex-direction: column; gap: 16px; border-top: 3px solid var(--accent-3);"
          >
            <div
              style="font-family: var(--font-mono); font-size: 11px; color: var(--accent-3); font-weight: 700; text-transform: uppercase; display: flex; align-items: center; gap: 8px;"
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                ><path
                  d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41-1.41"
                /></svg
              >
              --|Suport Prediksi data |--
            </div>

            <!-- Gauge UI -->
            <div
              style="position: relative; width: 100%; height: 100px; display: flex; flex-direction: column; align-items: center; justify-content: flex-end;"
            >
              <svg viewBox="0 0 100 50" style="width: 150px; height: 75px;">
                <path
                  d="M 10 50 A 40 40 0 0 1 90 50"
                  fill="none"
                  stroke="var(--border)"
                  stroke-width="8"
                  stroke-linecap="round"
                />
                <path
                  d="M 10 50 A 40 40 0 0 1 90 50"
                  fill="none"
                  stroke="var(--accent-3)"
                  stroke-width="8"
                  stroke-linecap="round"
                  stroke-dasharray="125.66"
                  stroke-dashoffset={125.66 - 125.66 * (mlStats.validity / 100)}
                  style="transition: stroke-dashoffset 0.8s ease-out;"
                />
              </svg>
              <div style="position: absolute; bottom: 0; text-align: center;">
                <div
                  style="font-size: 20px; font-weight: 800; color: var(--text);"
                >
                  {mlStats.validity.toFixed(0)}%
                </div>
                <div
                  style="font-size: 9px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 1px;"
                >
                  Kebenaran Data
                </div>
              </div>
            </div>

            <!-- Stats Table -->
            <table
              style="width: 100%; border-top: 1px solid var(--border); padding-top: 10px; font-family: var(--font-mono); font-size: 10px;"
            >
              <tbody>
                <tr>
                  <td style="padding: 4px 0; color: var(--text-muted);"
                    >Nilai Test</td
                  >
                  <td
                    style="padding: 4px 0; text-align: right; color: var(--accent-1);"
                    >{mlStats.testCount} pts</td
                  >
                </tr>
                <tr>
                  <td style="padding: 4px 0; color: var(--text-muted);"
                    >Akurasi Filter</td
                  >
                  <td
                    style="padding: 4px 0; text-align: right; color: var(--green);"
                    >{mlStats.accuracy.toFixed(1)}%</td
                  >
                </tr>
                <tr>
                  <td style="padding: 4px 0; color: var(--text-muted);"
                    >Data Abu"</td
                  >
                  <td
                    style="padding: 4px 0; text-align: right; color: var(--yellow);"
                    >{mlStats.isenCount}</td
                  >
                </tr>
                <tr>
                  <td style="padding: 4px 0; color: var(--text-muted);"
                    >Spam Block</td
                  >
                  <td
                    style="padding: 4px 0; text-align: right; color: var(--red);"
                    >{mlStats.spamCount}</td
                  >
                </tr>
              </tbody>
            </table>

            <div
              style="font-size: 9px; color: var(--text-muted); font-style: italic; text-align: center;"
            >
              *Analisis berbasis pelatihan dari ({records.length} data , ini test
              model dulu :v)
            </div>
          </div>
        </div>
      </div>

      <!-- ================= SISI TENGAH (6): DATA TABLE ================= -->
      <div class="center-col">
        <div class="card">
          <div style="overflow-x:auto;">
            <table style="width:100%;border-collapse:collapse;">
              <thead>
                <tr style="border-bottom:1px solid var(--border);">
                  {#each ["ID", "TS", "PREFIX", "BERAT", "PCK_CNT", "ST", "DT", "CONF"] as h}
                    <th
                      style="text-align:left;padding:8px 12px;font-family:var(--font-mono);font-size:9px;font-weight:700;color:var(--text-muted);text-transform:uppercase;"
                      >{h}</th
                    >
                  {/each}
                </tr>
              </thead>
              <tbody>
                {#each records as r}
                  {@const st = statusLabel(r.reg5)}
                  {@const dtCls =
                    r.data_type === "VALID"
                      ? "badge-green"
                      : r.data_type === "TEST"
                        ? "badge-blue"
                        : r.data_type === "SPAM"
                          ? "badge-red"
                          : "badge-yellow"}
                  <tr
                    style="border-bottom:1px solid var(--border);transition:background 0.1s;"
                    onmouseenter={(e: any) =>
                      (e.currentTarget.style.background = "var(--surface-2)")}
                    onmouseleave={(e: any) =>
                      (e.currentTarget.style.background = "")}
                  >
                    <td
                      style="padding:8px 12px;font-family:var(--font-mono);font-size:10px;color:var(--text-muted);"
                      >{r.id}</td
                    >
                    <td
                      style="padding:8px 12px;font-family:var(--font-mono);font-size:10px;white-space:nowrap;"
                      >{r.ts}</td
                    >
                    <td style="padding:8px 12px;white-space:nowrap;"
                      ><span class="badge badge-blue">{r.prefix}</span></td
                    >
                    <td
                      style="padding:8px 12px;font-family:var(--font-mono);font-size:11px;font-weight:700;color:var(--purple);white-space:nowrap;"
                      >{r.weight_formatted}</td
                    >
                    <td
                      style="padding:8px 12px;font-family:var(--font-mono);font-size:11px;"
                      >{r.reg2}</td
                    >
                    <td style="padding:8px 12px;"
                      ><span class="badge {st.cls}">{st.label}</span></td
                    >
                    <td style="padding:8px 12px;"
                      ><span class="badge {dtCls}">{r.data_type}</span></td
                    >
                    <td
                      style="padding:8px 12px;font-family:var(--font-mono);font-size:9px;color:var(--text-muted);"
                      >{(r.confidence * 100).toFixed(0)}%</td
                    >
                  </tr>
                {:else}
                  <tr
                    ><td
                      colspan="6"
                      style="padding:40px;text-align:center;color:var(--text-muted);font-family:var(--font-mono);font-size:11px;"
                      >[NO_DATA_IN_CURRENT_FILTER]</td
                    ></tr
                  >
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- ================= SISI KANAN (3): SUMMARY CARDS ================= -->
      <div class="right-col">
        <div class="summary-grid">
          {#each summaries as s}
            <div class="card" style="border-left:3px solid var(--purple);">
              <div
                class="card-label"
                style="font-weight: 800; font-size: 13px; letter-spacing: 1px; color: var(--text); display: flex; justify-content: space-between;"
              >
                <span>{s.prefix}</span>
                <span
                  style="font-size:10px; color:var(--text-muted); font-weight:400; text-transform:none;"
                  >Total: {s.total_count || 0}</span
                >
              </div>
              <div
                class="mono"
                style="font-size:22px;font-weight:700;color:var(--purple);margin:8px 0;"
              >
                {s.sum_weight !== undefined
                  ? (s.sum_weight / 10).toFixed(1) + "g"
                  : "-"}
              </div>

              <!-- Gauge 3 Color -->
              {#if (s.total_count || 0) > 0}
                {@const weigher = (s.under_count || 0) + (s.over_count || 0)}
                {@const validTotal =
                  (s.ok_count || 0) + weigher + (s.metal_count || 0)}
                {#if validTotal > 0}
                  <div
                    style="display: flex; height: 6px; border-radius: 3px; overflow: hidden; margin-bottom: 8px;"
                  >
                    <div
                      style="width: {((s.ok_count || 0) / validTotal) *
                        100}%; background-color: var(--green);"
                    ></div>
                    <div
                      style="width: {(weigher / validTotal) *
                        100}%; background-color: var(--yellow);"
                    ></div>
                    <div
                      style="width: {((s.metal_count || 0) / validTotal) *
                        100}%; background-color: var(--purple);"
                    ></div>
                  </div>
                {/if}
              {/if}

              <div
                class="flex justify-between text-xs text-muted"
                style="margin-bottom: 6px;"
              >
                <span
                  >OK: <span style="color:var(--green);">{s.ok_count || 0}</span
                  ></span
                >
                <span
                  >Weigher: <span style="color:var(--yellow);"
                    >{(s.under_count || 0) + (s.over_count || 0)}</span
                  ></span
                >
                <span
                  >Metal: <span style="color:var(--purple);"
                    >{s.metal_count || 0}</span
                  ></span
                >
              </div>

              <div
                class="flex justify-between text-xs text-muted"
                style="border-top: 1px solid var(--border); padding-top: 8px; margin-top: 4px;"
              >
                <span
                  >Min: {s.min_weight
                    ? (s.min_weight / 10).toFixed(1) + "g"
                    : "-"}</span
                >
                <span
                  >Avg: {s.avg_weight
                    ? (s.avg_weight / 10).toFixed(1) + "g"
                    : "-"}</span
                >
                <span
                  >Max: {s.max_weight
                    ? (s.max_weight / 10).toFixed(1) + "g"
                    : "-"}</span
                >
              </div>
            </div>
          {:else}
            <div
              class="card"
              style="text-align:center;padding:20px;color:var(--text-muted);font-family:var(--font-mono);font-size:11px;"
            >
              [AWAITING PRODUCTION DATA]
            </div>
          {/each}
        </div>
      </div>
    </main>
  </div>
{/if}

{#if isLoading}
  <div class="loading-bar"></div>
{/if}

<style>
  :global(body) {
    margin: 0;
  }
  .loading-bar {
    position: fixed;
    bottom: 0;
    left: 0;
    width: 100%;
    height: 4px;
    background-color: var(--surface-2, #333);
    z-index: 9999;
    overflow: hidden;
  }
  .loading-bar::after {
    content: "";
    display: block;
    position: absolute;
    top: 0;
    left: -30%;
    height: 100%;
    width: 30%;
    background-color: var(--purple, #a855f7);
    animation: loading-slide 1.5s infinite ease-in-out;
  }
  @keyframes loading-slide {
    0% {
      left: -30%;
    }
    100% {
      left: 100%;
    }
  }
</style>
