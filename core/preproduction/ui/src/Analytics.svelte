<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Chart, registerables } from "chart.js";
  Chart.register(...registerables);

  let { api = async () => null, token = "", theme = "dark" }: {
    api: (path: string, opts?: RequestInit) => Promise<any>;
    token: string;
    theme: string;
  } = $props();

  let mdcwSummary: any[] = $state([]);
  let spSummary: any[] = $state([]);
  let mdcwDaily: any[] = $state([]);
  let spDaily: any[] = $state([]);
  let mdcwRecords: any[] = $state([]);
  let spRecords: any[] = $state([]);
  let cartonSize = $state(10);
  let days = $state(14);
  let isLoading = $state(true);

  let chartMdcwTrend: Chart | null = null;
  let chartMdcwStatus: Chart | null = null;
  let chartMdcwMachine: Chart | null = null;
  let chartSpTrend: Chart | null = null;
  let chartSpSession: Chart | null = null;

  let canvasMdcwTrend = $state<HTMLCanvasElement | null>(null);
  let canvasMdcwStatus = $state<HTMLCanvasElement | null>(null);
  let canvasMdcwMachine = $state<HTMLCanvasElement | null>(null);
  let canvasSpTrend = $state<HTMLCanvasElement | null>(null);
  let canvasSpSession = $state<HTMLCanvasElement | null>(null);

  let pollInterval: ReturnType<typeof setInterval>;
  let chartsInited = $state(false);

  const getChartColors = (th: string) => {
    const isDark = th === "dark";
    return {
      text: isDark ? "#a1a1aa" : "#4b5563",
      grid: isDark ? "rgba(255, 255, 255, 0.04)" : "rgba(0, 0, 0, 0.04)",
      tick: isDark ? "#71717a" : "#9ca3af",
    };
  };

  function mdcwApi(p: string) { return `/api/mdcw${p}`; }
  function spApi(p: string) { return `/api/sp${p}`; }

  onMount(async () => {
    await loadData();
    chartsInited = true;
    pollInterval = setInterval(loadData, 30000);
  });
  onDestroy(() => { clearInterval(pollInterval); destroyCharts(); });

  async function loadData() {
    isLoading = true;
    try {
      const [mSum, sSum, mDaily, sDaily] = await Promise.all([
        api(mdcwApi("/summary")), api(spApi("/summary")),
        api(mdcwApi("/daily-stats?days=" + days)), api(spApi("/daily-stats?days=" + days)),
      ]);
      mdcwSummary = mSum || []; spSummary = sSum || []; mdcwDaily = mDaily || []; spDaily = sDaily || [];

      const [mRec, sRec] = await Promise.all([
        api(mdcwApi("/data?sort=newest")), api(spApi("/data")),
      ]);
      mdcwRecords = (mRec || []).slice(0, 50);
      spRecords = (sRec || []).slice(0, 50);

      if (chartsInited) { initCharts(); }
    } catch (e) { console.error("Analytics error", e); }
    finally { isLoading = false; }
  }

  function destroyCharts() {
    [chartMdcwTrend, chartMdcwStatus, chartMdcwMachine, chartSpTrend, chartSpSession].forEach(c => c?.destroy());
  }

  function initCharts() {
    destroyCharts();
    createMdcwTrend(); createMdcwStatus(); createMdcwMachine(); createSpTrend(); createSpSession();
  }

  function createMdcwTrend() {
    if (!canvasMdcwTrend || mdcwDaily.length === 0) return;
    const colors = getChartColors(theme);
    const labels = mdcwDaily.map((d: any) => d.date?.slice(5) || "");
    chartMdcwTrend = new Chart(canvasMdcwTrend, { type: "line",
      data: { labels,
        datasets: [
          { label: "OK", data: mdcwDaily.map((d: any) => d.ok_count || 0), borderColor: "#10b981", backgroundColor: "rgba(16,185,129,0.04)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
          { label: "UNDER", data: mdcwDaily.map((d: any) => d.under_count || 0), borderColor: "#3b82f6", backgroundColor: "rgba(59,130,246,0.04)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
          { label: "OVER", data: mdcwDaily.map((d: any) => d.over_count || 0), borderColor: "#f59e0b", backgroundColor: "rgba(245,158,11,0.04)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
          { label: "METAL", data: mdcwDaily.map((d: any) => d.metal_count || 0), borderColor: "#8b5cf6", backgroundColor: "rgba(139,92,246,0.04)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
        ] },
      options: { responsive: true, maintainAspectRatio: false,
        plugins: { legend: { position: "top", labels: { color: colors.text, boxWidth: 8, padding: 6, font: { size: 8, family: 'JetBrains Mono' } } } },
        scales: { x: { grid: { display: false }, ticks: { color: colors.tick, font: { size: 8 } } }, y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.tick, font: { size: 8 } } } },
      },
    });
  }

  function createMdcwStatus() {
    if (!canvasMdcwStatus) return;
    const recs = mdcwRecords;
    const ok = recs.filter((r: any) => [41,521,553].includes(r.reg5)).length;
    const under = recs.filter((r: any) => r.reg5 === 25).length;
    const over = recs.filter((r: any) => r.reg5 === 73).length;
    const metal = recs.filter((r: any) => r.reg5 === 8201).length;
    const other = recs.length - ok - under - over - metal;
    const colors = getChartColors(theme);
    chartMdcwStatus = new Chart(canvasMdcwStatus, { type: "doughnut",
      data: { labels: ["OK","UNDER","OVER","METAL","Other"], datasets: [{ data: [ok,under,over,metal,other], backgroundColor: ["#10b981","#3b82f6","#f59e0b","#8b5cf6","#4b5563"], borderWidth: 0 }] },
      options: { responsive: true, maintainAspectRatio: false, cutout: "68%",
        plugins: { legend: { position: "right", labels: { color: colors.text, boxWidth: 8, padding: 4, font: { size: 8, family: 'JetBrains Mono' } } } },
      },
    });
  }

  // Reactive chart redraw on theme/data changes
  $effect(() => {
    const currentTheme = theme; // registers dependency
    if (chartsInited && mdcwDaily.length > 0) {
      initCharts();
    }
  });

  function createMdcwMachine() {
    if (!canvasMdcwMachine || mdcwSummary.length === 0) return;
    const colors = getChartColors(theme);
    chartMdcwMachine = new Chart(canvasMdcwMachine, { type: "bar",
      data: { labels: mdcwSummary.map((s: any) => s.prefix?.replace("MDCW","") || ""),
        datasets: [
          { label: "OK", data: mdcwSummary.map((s: any) => s.ok_count||0), backgroundColor: "#10b981", borderRadius: 2 },
          { label: "UNDER", data: mdcwSummary.map((s: any) => s.under_count||0), backgroundColor: "#3b82f6", borderRadius: 2 },
          { label: "OVER", data: mdcwSummary.map((s: any) => s.over_count||0), backgroundColor: "#f59e0b", borderRadius: 2 },
          { label: "METAL", data: mdcwSummary.map((s: any) => s.metal_count||0), backgroundColor: "#8b5cf6", borderRadius: 2 },
        ] },
      options: { responsive: true, maintainAspectRatio: false,
        plugins: { legend: { position: "top", labels: { color: colors.text, boxWidth: 8, padding: 6, font: { size: 8, family: 'JetBrains Mono' } } } },
        scales: { x: { grid: { display: false }, ticks: { color: colors.tick, font: { size: 7 } } }, y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.tick, font: { size: 8 } } } },
      },
    });
  }

  function createSpTrend() {
    if (!canvasSpTrend || spDaily.length === 0) return;
    const colors = getChartColors(theme);
    chartSpTrend = new Chart(canvasSpTrend, { type: "bar",
      data: { labels: spDaily.map((d: any) => d.date?.slice(5) || ""),
        datasets: [{ label: "Scans", data: spDaily.map((d: any) => d.total_count||0), backgroundColor: "rgba(59,130,246,0.5)", borderColor: "#3b82f6", borderWidth: 1, borderRadius: 2 }] },
      options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } },
        scales: { x: { grid: { display: false }, ticks: { color: colors.tick, font: { size: 8 } } }, y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.tick, font: { size: 8 } } } },
      },
    });
  }

  function createSpSession() {
    if (!canvasSpSession || spSummary.length === 0) return;
    const top = spSummary.slice(0, 10);
    const colors = getChartColors(theme);
    chartSpSession = new Chart(canvasSpSession, { type: "bar",
      data: { labels: top.map((s: any) => s.session_id || ""),
        datasets: [{ label: "Scans", data: top.map((s: any) => s.total_count||0), backgroundColor: "rgba(139,92,246,0.5)", borderColor: "#8b5cf6", borderWidth: 1, borderRadius: 2 }] },
      options: { indexAxis: "y", responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } },
        scales: { x: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.tick, font: { size: 8 } } }, y: { grid: { display: false }, ticks: { color: colors.tick, font: { size: 7 } } } },
      },
    });
  }

  // === COMPUTED STATS ===
  const totalMdcwToday = $derived(mdcwSummary.reduce((a: number, s: any) => a + (s.total_count||0), 0));
  const totalSpToday = $derived(spSummary.reduce((a: number, s: any) => a + (s.total_count||0), 0));
  const totalOkToday = $derived(mdcwSummary.reduce((a: number, s: any) => a + (s.ok_count||0), 0));
  const okRate = $derived(totalMdcwToday > 0 ? ((totalOkToday / totalMdcwToday) * 100) : 0);

  // Per-machine carton logic
  const machineCartons = $derived(mdcwSummary.map((s: any) => {
    const ok = s.ok_count || 0;
    const cartons = Math.floor(ok / cartonSize);
    const loose = ok % cartonSize;
    return { prefix: s.prefix, ok, cartons, loose, total: s.total_count || 0 };
  }));

  const totalCartons = $derived(machineCartons.reduce((a: number, m: any) => a + m.cartons, 0));
  const totalLoose = $derived(machineCartons.reduce((a: number, m: any) => a + m.loose, 0));

  // SP stats
  const spToday = $derived(spSummary.reduce((a: number, s: any) => a + (s.total_count||0), 0));
</script>

<div class="dash">
  <!-- 6 KPI Grid Overview -->
  <div class="kpi-grid">
    <div class="kpi-card" style="border-top: 3px solid var(--green);">
      <div class="kpi-card-header">
        <span class="kpi-title">Output MDCW</span>
      </div>
      <div class="kpi-value">{totalMdcwToday}</div>
      <span class="kpi-subtext">produksi hari ini</span>
    </div>

    <div class="kpi-card" style="border-top: 3px solid var(--green);">
      <div class="kpi-card-header">
        <span class="kpi-title">OK Produced</span>
      </div>
      <div class="kpi-value" style="color:var(--green)">{totalOkToday}</div>
      <span class="kpi-subtext">{okRate.toFixed(1)}% ok rate</span>
    </div>

    <div class="kpi-card" style="border-top: 3px solid var(--accent);">
      <div class="kpi-card-header">
        <span class="kpi-title">Carton Filled</span>
      </div>
      <div class="kpi-value" style="color:var(--accent)">{totalCartons}</div>
      <span class="kpi-subtext">{cartonSize} pcs/carton</span>
    </div>

    <div class="kpi-card" style="border-top: 3px solid var(--yellow);">
      <div class="kpi-card-header">
        <span class="kpi-title">Loose Items</span>
      </div>
      <div class="kpi-value" style="color:var(--yellow)">{totalLoose}</div>
      <span class="kpi-subtext">belum di-carton</span>
    </div>

    <div class="kpi-card" style="border-top: 3px solid var(--accent-2);">
      <div class="kpi-card-header">
        <span class="kpi-title">SP Scans</span>
      </div>
      <div class="kpi-value" style="color:var(--accent-2)">{spToday}</div>
      <span class="kpi-subtext">scan hari ini</span>
    </div>

    <div class="kpi-card" style="border-top: 3px solid var(--purple);">
      <div class="kpi-card-header">
        <span class="kpi-title">Periode Analisis</span>
      </div>
      <div style="display: flex; align-items: center; justify-content: space-between; margin-top: 8px;">
        <span class="kpi-value" style="margin-top: 0; color: var(--purple);">{days} Hari</span>
        <select bind:value={days} onchange={loadData} class="filter-pill" style="padding: 2px 10px; font-family: var(--font-mono); font-size: 9px; cursor: pointer; border-radius: 9999px;">
          <option value={7}>7 Hari</option>
          <option value={14}>14 Hari</option>
          <option value={30}>30 Hari</option>
        </select>
      </div>
      <span class="kpi-subtext">rentang data tren</span>
    </div>
  </div>

  <!-- Carton Pipeline -->
  <div class="section-h" style="margin-top: 10px;">
    <span class="section-dot" style="background:var(--accent)"></span>
    <span>MDCW &rarr; Carton Pipeline</span>
  </div>
  <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 12px;">
    {#each machineCartons as m}
      <div class="card" style="padding: 12px; display: flex; flex-direction: column; gap: 8px;">
        <div style="font-family: var(--font-mono); font-size: 10px; font-weight: 700; color: var(--text); border-bottom: 1px solid var(--border); padding-bottom: 4px;">
          {m.prefix?.replace("MDCW ","") || m.prefix}
        </div>
        <div style="display: flex; justify-content: space-around; align-items: center; padding: 4px 0;">
          <div style="text-align: center;">
            <div style="font-family: var(--font-mono); font-size: 14px; font-weight: 700; color: var(--green);">{m.ok}</div>
            <div style="font-size: 8px; color: var(--text-muted); text-transform: uppercase;">OK</div>
          </div>
          <div style="color: var(--text-dim); font-size: 12px;">&rarr;</div>
          <div style="text-align: center;">
            <div style="font-family: var(--font-mono); font-size: 14px; font-weight: 700; color: var(--accent);">{m.cartons}</div>
            <div style="font-size: 8px; color: var(--text-muted); text-transform: uppercase;">Carton</div>
          </div>
          <div style="color: var(--text-dim); font-size: 12px;">&rarr;</div>
          <div style="text-align: center;">
            <div style="font-family: var(--font-mono); font-size: 14px; font-weight: 700; color: var(--yellow);">{m.loose}</div>
            <div style="font-size: 8px; color: var(--text-muted); text-transform: uppercase;">Loose</div>
          </div>
        </div>
        {#if m.total > 0}
          <div style="height: 4px; border-radius: 9999px; overflow: hidden; background: var(--surface-2); display: flex;">
            <div style="width: {(m.ok / m.total) * 100}%; background: var(--green); height: 100%;"></div>
            <div style="width: {((m.total - m.ok) / m.total) * 100}%; background: var(--red-dim); height: 100%;"></div>
          </div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Charts: MDCW -->
  <div class="section-h" style="margin-top: 10px;">
    <span class="section-dot" style="background:var(--green)"></span>
    <span>MDCW Analytics</span>
  </div>
  <div class="grid-2col">
    <div class="card" style="display: flex; flex-direction: column; justify-content: space-between; min-height: 250px;">
      <span class="kpi-title" style="margin-bottom: 8px;">Trend Produksi</span>
      <div style="position: relative; flex: 1; height: 180px;">
        <canvas bind:this={canvasMdcwTrend}></canvas>
      </div>
    </div>
    <div class="card" style="display: flex; flex-direction: column; justify-content: space-between; min-height: 250px;">
      <span class="kpi-title" style="margin-bottom: 8px;">Distribusi Status</span>
      <div style="position: relative; flex: 1; height: 180px;">
        <canvas bind:this={canvasMdcwStatus}></canvas>
      </div>
    </div>
  </div>
  <div class="card" style="margin-top: 10px; display: flex; flex-direction: column; justify-content: space-between; min-height: 250px;">
    <span class="kpi-title" style="margin-bottom: 8px;">Performa Mesin</span>
    <div style="position: relative; flex: 1; height: 180px;">
      <canvas bind:this={canvasMdcwMachine}></canvas>
    </div>
  </div>

  <!-- Charts: SP -->
  <div class="section-h" style="margin-top: 10px;">
    <span class="section-dot" style="background:var(--accent-2)"></span>
    <span>SP Analytics</span>
  </div>
  <div class="grid-2col">
    <div class="card" style="display: flex; flex-direction: column; justify-content: space-between; min-height: 250px;">
      <span class="kpi-title" style="margin-bottom: 8px;">Scan Harian</span>
      <div style="position: relative; flex: 1; height: 180px;">
        <canvas bind:this={canvasSpTrend}></canvas>
      </div>
    </div>
    <div class="card" style="display: flex; flex-direction: column; justify-content: space-between; min-height: 250px;">
      <span class="kpi-title" style="margin-bottom: 8px;">Top Session</span>
      <div style="position: relative; flex: 1; height: 180px;">
        <canvas bind:this={canvasSpSession}></canvas>
      </div>
    </div>
  </div>

  <!-- Carton Config -->
  <div class="section-h" style="margin-top: 10px;">
    <span class="section-dot" style="background:var(--text-muted)"></span>
    <span>Konfigurasi Pipeline</span>
  </div>
  <div class="card" style="display: flex; gap: 20px; flex-wrap: wrap; align-items: center; padding: 16px;">
    <div style="display: flex; align-items: center; gap: 8px;">
      <span style="font-family: var(--font-mono); font-size: 9px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Ukuran Carton (pcs):</span>
      <input type="number" bind:value={cartonSize} min="1" max="100" class="filter-pill" style="width: 70px; text-align: center; border-radius: 9999px;" />
    </div>
    <div style="display: flex; align-items: center; gap: 8px;">
      <span style="font-family: var(--font-mono); font-size: 9px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Total Carton Terisi:</span>
      <span style="font-family: var(--font-mono); font-size: 12px; font-weight: 700; color: var(--accent-2);">{totalCartons}</span>
    </div>
    <div style="display: flex; align-items: center; gap: 8px;">
      <span style="font-family: var(--font-mono); font-size: 9px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Total Loose Items:</span>
      <span style="font-family: var(--font-mono); font-size: 12px; font-weight: 700; color: var(--yellow);">{totalLoose}</span>
    </div>
    <div style="display: flex; align-items: center; gap: 8px;">
      <span style="font-family: var(--font-mono); font-size: 9px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Rerata OK Rate:</span>
      <span style="font-family: var(--font-mono); font-size: 12px; font-weight: 700; color: var(--green);">{okRate.toFixed(1)}%</span>
    </div>
  </div>
</div>
