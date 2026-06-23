<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Chart, registerables } from "chart.js";
  Chart.register(...registerables);

  let { api = async () => null, token = "" }: {
    api: (path: string, opts?: RequestInit) => Promise<any>;
    token: string;
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

  let canvasMdcwTrend: HTMLCanvasElement;
  let canvasMdcwStatus: HTMLCanvasElement;
  let canvasMdcwMachine: HTMLCanvasElement;
  let canvasSpTrend: HTMLCanvasElement;
  let canvasSpSession: HTMLCanvasElement;

  let pollInterval: ReturnType<typeof setInterval>;
  let chartsInited = $state(false);

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
    const labels = mdcwDaily.map((d: any) => d.date?.slice(5) || "");
    chartMdcwTrend = new Chart(canvasMdcwTrend, { type: "line",
      data: { labels,
        datasets: [
          { label: "OK", data: mdcwDaily.map((d: any) => d.ok_count || 0), borderColor: "#16a34a", backgroundColor: "rgba(22,163,74,0.06)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
          { label: "UNDER", data: mdcwDaily.map((d: any) => d.under_count || 0), borderColor: "#3b82f6", backgroundColor: "rgba(59,130,246,0.06)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
          { label: "OVER", data: mdcwDaily.map((d: any) => d.over_count || 0), borderColor: "#ca8a04", backgroundColor: "rgba(202,138,4,0.06)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
          { label: "METAL", data: mdcwDaily.map((d: any) => d.metal_count || 0), borderColor: "#7c3aed", backgroundColor: "rgba(124,58,237,0.06)", fill: true, tension: 0.3, pointRadius: 1, borderWidth: 1.5 },
        ] },
      options: { responsive: true, maintainAspectRatio: false,
        plugins: { legend: { position: "top", labels: { boxWidth: 8, padding: 6, font: { size: 8 } } } },
        scales: { x: { grid: { display: false }, ticks: { font: { size: 8 } } }, y: { beginAtZero: true, grid: { color: "rgba(255,255,255,0.03)" }, ticks: { font: { size: 8 } } } },
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
    chartMdcwStatus = new Chart(canvasMdcwStatus, { type: "doughnut",
      data: { labels: ["OK","UNDER","OVER","METAL","Other"], datasets: [{ data: [ok,under,over,metal,other], backgroundColor: ["#16a34a","#3b82f6","#ca8a04","#7c3aed","#6b7280"], borderWidth: 0 }] },
      options: { responsive: true, maintainAspectRatio: false, cutout: "68%",
        plugins: { legend: { position: "right", labels: { boxWidth: 8, padding: 4, font: { size: 8 } } } },
      },
    });
  }

  function createMdcwMachine() {
    if (!canvasMdcwMachine || mdcwSummary.length === 0) return;
    chartMdcwMachine = new Chart(canvasMdcwMachine, { type: "bar",
      data: { labels: mdcwSummary.map((s: any) => s.prefix?.replace("MDCW","") || ""),
        datasets: [
          { label: "OK", data: mdcwSummary.map((s: any) => s.ok_count||0), backgroundColor: "#16a34a", borderRadius: 2 },
          { label: "UNDER", data: mdcwSummary.map((s: any) => s.under_count||0), backgroundColor: "#3b82f6", borderRadius: 2 },
          { label: "OVER", data: mdcwSummary.map((s: any) => s.over_count||0), backgroundColor: "#ca8a04", borderRadius: 2 },
          { label: "METAL", data: mdcwSummary.map((s: any) => s.metal_count||0), backgroundColor: "#7c3aed", borderRadius: 2 },
        ] },
      options: { responsive: true, maintainAspectRatio: false,
        plugins: { legend: { position: "top", labels: { boxWidth: 8, padding: 6, font: { size: 8 } } } },
        scales: { x: { grid: { display: false }, ticks: { font: { size: 7 } } }, y: { beginAtZero: true, grid: { color: "rgba(255,255,255,0.03)" }, ticks: { font: { size: 8 } } } },
      },
    });
  }

  function createSpTrend() {
    if (!canvasSpTrend || spDaily.length === 0) return;
    chartSpTrend = new Chart(canvasSpTrend, { type: "bar",
      data: { labels: spDaily.map((d: any) => d.date?.slice(5) || ""),
        datasets: [{ label: "Scans", data: spDaily.map((d: any) => d.total_count||0), backgroundColor: "rgba(99,102,241,0.5)", borderColor: "#6366f1", borderWidth: 1, borderRadius: 2 }] },
      options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } },
        scales: { x: { grid: { display: false }, ticks: { font: { size: 8 } } }, y: { beginAtZero: true, grid: { color: "rgba(255,255,255,0.03)" }, ticks: { font: { size: 8 } } } },
      },
    });
  }

  function createSpSession() {
    if (!canvasSpSession || spSummary.length === 0) return;
    const top = spSummary.slice(0, 10);
    chartSpSession = new Chart(canvasSpSession, { type: "bar",
      data: { labels: top.map((s: any) => s.session_id || ""),
        datasets: [{ label: "Scans", data: top.map((s: any) => s.total_count||0), backgroundColor: "rgba(124,58,237,0.5)", borderColor: "#7c3aed", borderWidth: 1, borderRadius: 2 }] },
      options: { indexAxis: "y", responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } },
        scales: { x: { beginAtZero: true, grid: { color: "rgba(255,255,255,0.03)" }, ticks: { font: { size: 8 } } }, y: { grid: { display: false }, ticks: { font: { size: 7 } } } },
      },
    });
  }

  $effect(() => { if (chartsInited && mdcwDaily.length > 0) { initCharts(); } });

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
  <!-- Overview Row -->
  <div class="stat-row">
    <div class="stat-box"><div class="stat-lbl">Output MDCW</div><div class="stat-val">{totalMdcwToday}</div><div class="stat-sub">produksi hari ini</div></div>
    <div class="stat-box"><div class="stat-lbl">OK Produced</div><div class="stat-val" style="color:var(--green)">{totalOkToday}</div><div class="stat-sub">{okRate.toFixed(1)}% ok rate</div></div>
    <div class="stat-box"><div class="stat-lbl">Carton Filled</div><div class="stat-val" style="color:var(--accent)">{totalCartons}</div><div class="stat-sub">{cartonSize} pcs/carton</div></div>
    <div class="stat-box"><div class="stat-lbl">Loose Items</div><div class="stat-val" style="color:var(--yellow)">{totalLoose}</div><div class="stat-sub">belum di-carton</div></div>
    <div class="stat-box"><div class="stat-lbl">SP Scans</div><div class="stat-val" style="color:var(--accent-2)">{spToday}</div><div class="stat-sub">scan hari ini</div></div>
    <div class="stat-box">
      <div class="stat-lbl">Periode</div>
      <div class="stat-val" style="font-size:14px">{days}d</div>
      <div class="stat-sub">
        <select bind:value={days} onchange={loadData} style="font-size:8px;background:transparent;border:1px solid var(--border);color:var(--text-muted);padding:1px 4px;border-radius:4px;">
          <option value={7}>7d</option><option value={14}>14d</option><option value={30}>30d</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Carton Pipeline -->
  <div class="section-h"><span class="section-dot" style="background:var(--accent)"></span> MDCW → Carton Pipeline</div>
  <div class="pipeline-row">
    {#each machineCartons as m}
      <div class="pipe-card">
        <div class="pipe-header">{m.prefix?.replace("MDCW ","") || m.prefix}</div>
        <div class="pipe-flow">
          <div class="pipe-node">
            <div class="pipe-num" style="color:var(--green)">{m.ok}</div>
            <div class="pipe-lbl">OK</div>
          </div>
          <div class="pipe-arrow">→</div>
          <div class="pipe-node">
            <div class="pipe-num" style="color:var(--accent)">{m.cartons}</div>
            <div class="pipe-lbl">Carton</div>
          </div>
          <div class="pipe-arrow">→</div>
          <div class="pipe-node">
            <div class="pipe-num" style="color:var(--yellow)">{m.loose}</div>
            <div class="pipe-lbl">Loose</div>
          </div>
        </div>
        {#if m.total > 0}
          <div class="pipe-bar">
            <div class="pipe-bar-fill" style="width:{(m.ok/m.total)*100}%;background:var(--green);"></div>
            <div class="pipe-bar-fill" style="width:{((m.total-m.ok)/m.total)*100}%;background:var(--red-dim);"></div>
          </div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Charts: MDCW -->
  <div class="section-h"><span class="section-dot" style="background:var(--green)"></span> MDCW Analytics</div>
  <div class="chart-row">
    <div class="chart-card"><div class="chart-title">Trend Produksi</div><div class="chart-wrap"><canvas bind:this={canvasMdcwTrend}></canvas></div></div>
    <div class="chart-card"><div class="chart-title">Distribusi Status</div><div class="chart-wrap"><canvas bind:this={canvasMdcwStatus}></canvas></div></div>
    <div class="chart-card chart-wide"><div class="chart-title">Performa Mesin</div><div class="chart-wrap"><canvas bind:this={canvasMdcwMachine}></canvas></div></div>
  </div>

  <!-- Charts: SP -->
  <div class="section-h"><span class="section-dot" style="background:var(--accent-2)"></span> SP Analytics</div>
  <div class="chart-row">
    <div class="chart-card"><div class="chart-title">Scan Harian</div><div class="chart-wrap"><canvas bind:this={canvasSpTrend}></canvas></div></div>
    <div class="chart-card"><div class="chart-title">Top Session</div><div class="chart-wrap"><canvas bind:this={canvasSpSession}></canvas></div></div>
  </div>

  <!-- Carton Config -->
  <div class="section-h"><span class="section-dot" style="background:var(--text-muted)"></span> Konfigurasi</div>
  <div class="config-row">
    <div class="config-item">
      <span class="config-lbl">Ukuran Carton (pcs)</span>
      <input type="number" bind:value={cartonSize} min="1" max="100" style="width:60px;font-size:10px;text-align:center;" />
    </div>
    <div class="config-item">
      <span class="config-lbl">Total Carton Terisi</span>
      <span class="config-val">{totalCartons}</span>
    </div>
    <div class="config-item">
      <span class="config-lbl">Total Loose</span>
      <span class="config-val" style="color:var(--yellow)">{totalLoose}</span>
    </div>
    <div class="config-item">
      <span class="config-lbl">OK Rate Rata-rata</span>
      <span class="config-val" style="color:var(--green)">{okRate.toFixed(1)}%</span>
    </div>
  </div>
</div>

<style>
  .dash { display:flex; flex-direction:column; gap:10px; }

  .stat-row { display:grid; grid-template-columns:repeat(6,1fr); gap:6px; }
  .stat-box { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:10px; text-align:center; }
  .stat-lbl { font-size:8px; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.4px; font-weight:600; }
  .stat-val { font-size:20px; font-weight:800; color:var(--text); margin:2px 0; font-family:var(--font-mono); }
  .stat-sub { font-size:8px; color:var(--text-muted); }

  .section-h { display:flex; align-items:center; gap:6px; padding:2px 0; font-size:10px; font-weight:600; color:var(--text); text-transform:uppercase; letter-spacing:0.3px; }
  .section-dot { width:6px; height:6px; border-radius:50%; display:inline-block; }

  .pipeline-row { display:grid; grid-template-columns:repeat(auto-fill, minmax(200px,1fr)); gap:6px; }
  .pipe-card { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:10px; }
  .pipe-header { font-size:9px; font-weight:700; color:var(--text); margin-bottom:8px; font-family:var(--font-mono); }
  .pipe-flow { display:flex; align-items:center; justify-content:center; gap:6px; margin-bottom:6px; }
  .pipe-node { text-align:center; }
  .pipe-num { font-size:16px; font-weight:800; font-family:var(--font-mono); }
  .pipe-lbl { font-size:7px; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.3px; }
  .pipe-arrow { color:var(--text-dim); font-size:12px; }
  .pipe-bar { display:flex; height:3px; border-radius:2px; overflow:hidden; }

  .chart-row { display:grid; grid-template-columns:1fr 1fr; gap:6px; }
  .chart-wide { grid-column:1/-1; }
  .chart-card { background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:10px; }
  .chart-title { font-size:8px; font-weight:600; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.3px; margin-bottom:6px; }
  .chart-wrap { position:relative; height:160px; }
  .chart-wide .chart-wrap { height:180px; }

  .config-row { display:flex; gap:12px; flex-wrap:wrap; align-items:center; }
  .config-item { display:flex; align-items:center; gap:6px; background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:8px 12px; }
  .config-lbl { font-size:9px; color:var(--text-muted); }
  .config-val { font-size:14px; font-weight:700; font-family:var(--font-mono); color:var(--text); }

  @media (max-width:800px) { .stat-row { grid-template-columns:repeat(3,1fr); } .chart-row { grid-template-columns:1fr; } }
  @media (max-width:500px) { .stat-row { grid-template-columns:repeat(2,1fr); } }
</style>
