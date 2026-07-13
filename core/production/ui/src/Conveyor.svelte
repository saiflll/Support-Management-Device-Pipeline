<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Chart, registerables } from "chart.js";
  Chart.register(...registerables);

  let { api = async () => null, token = "", theme = "dark" }: {
    api: (path: string, opts?: RequestInit) => Promise<any>;
    token: string;
    theme: string;
  } = $props();

  let summaryToday = $state<any>({ total: 0, per_shift: {}, last_update: "" });
  let weeklyData = $state<any[]>([]);
  let productDistData = $state<any[]>([]);
  let timelineData = $state<any[]>([]);
  let selectedDate = $state(new Date().toISOString().split("T")[0]);
  let isLoading = $state(true);

  let chartWeekly: Chart | null = null;
  let chartTimeline: Chart | null = null;

  let canvasWeekly = $state<HTMLCanvasElement | null>(null);
  let canvasTimeline = $state<HTMLCanvasElement | null>(null);

  let chartsInited = $state(false);
  let pollInterval: ReturnType<typeof setInterval>;

  const getChartColors = (th: string) => {
    const isDark = th === "dark";
    return {
      text: isDark ? "#a1a1aa" : "#4b5563",
      grid: isDark ? "rgba(255, 255, 255, 0.04)" : "rgba(0, 0, 0, 0.04)",
      tick: isDark ? "#71717a" : "#9ca3af",
    };
  };

  onMount(async () => {
    await loadData();
    chartsInited = true;
    pollInterval = setInterval(loadData, 30000);
  });

  onDestroy(() => {
    clearInterval(pollInterval);
    destroyCharts();
  });

  async function loadData() {
    isLoading = true;
    try {
      const [sum, wk, dist] = await Promise.all([
        api("/api/analytics/summary-today"),
        api("/api/analytics/weekly"),
        api("/api/analytics/product-dist")
      ]);
      summaryToday = sum || { total: 0, per_shift: {}, last_update: "" };
      weeklyData = wk || [];
      productDistData = dist || [];
      await loadTimeline();
    } catch (e) {
      console.error("Conveyor dashboard load error", e);
    } finally {
      isLoading = false;
    }
  }

  async function loadTimeline() {
    try {
      const tl = await api(`/api/analytics/shift-timeline?date=${selectedDate}`);
      timelineData = tl || [];
      if (chartsInited) {
        initCharts();
      }
    } catch (e) {
      console.error("Timeline load error", e);
    }
  }

  // Reactive chart redraw on theme/data changes
  $effect(() => {
    const currentTheme = theme; // registers dependency
    if (chartsInited && weeklyData.length > 0 && canvasWeekly) {
      initCharts();
    }
  });

  function destroyCharts() {
    [chartWeekly, chartTimeline].forEach(c => c?.destroy());
  }

  function initCharts() {
    destroyCharts();
    createWeeklyChart();
    createTimelineChart();
  }

  function createWeeklyChart() {
    if (!canvasWeekly || weeklyData.length === 0) return;
    const labels = weeklyData.map(d => {
      const dt = new Date(d.date);
      return dt.toLocaleDateString('id-ID', { weekday: 'short', day: 'numeric' });
    });
    const counts = weeklyData.map(d => d.count);
    
    chartWeekly = new Chart(canvasWeekly, {
      type: 'bar',
      data: {
        labels,
        datasets: [{
          label: 'Scan Volume',
          data: counts,
          backgroundColor: 'rgba(59, 130, 246, 0.4)',
          borderColor: 'var(--accent-2)',
          borderWidth: 1,
          borderRadius: 2
        }]
      },
      options: chartOptions()
    });
  }

  function createTimelineChart() {
    if (!canvasTimeline) return;
    const shiftMap: Record<string, number[]> = {};
    (timelineData || []).forEach(p => {
      if (!shiftMap[p.shift]) shiftMap[p.shift] = new Array(24).fill(0);
      shiftMap[p.shift][p.hour] = p.count;
    });

    const hours = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`);
    const colors = { '1': '#10b981', '2': '#f59e0b', '3': '#ef4444' };
    const datasets = Object.entries(shiftMap).map(([sh, vals]) => ({
      label: `Shift ${sh}`,
      data: vals,
      borderColor: colors[sh as keyof typeof colors] || 'var(--accent)',
      backgroundColor: 'transparent',
      borderWidth: 1.5,
      tension: 0.3,
      pointRadius: 1
    }));

    chartTimeline = new Chart(canvasTimeline, {
      type: 'line',
      data: { labels: hours, datasets },
      options: chartOptions()
    });
  }

  function chartOptions() {
    const colors = getChartColors(theme);
    return {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          labels: {
            color: colors.text,
            font: { size: 8, family: 'JetBrains Mono' },
            boxWidth: 8,
            padding: 6
          }
        }
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: colors.tick, font: { size: 8 } }
        },
        y: {
          beginAtZero: true,
          grid: { color: colors.grid },
          ticks: { color: colors.tick, font: { size: 8 } }
        }
      }
    };
  }

  // Shift details logic
  const shiftsList = ['1', '2', '3'];
  const shiftLabels = { '1': 'Shift 1 · Pagi', '2': 'Shift 2 · Siang', '3': 'Shift 3 · Malam' };
  const shiftColors = { '1': 'var(--green)', '2': 'var(--yellow)', '3': 'var(--red)' };

  // Product Code to Name Map for readability
  const productCodeToName: Record<string, string> = {
    "100294": "UDANG KEJU",
    "100256": "SIOMAY DIMSUM",
    "100286": "UDANG RAMBUTAN (PEN)",
    "100209": "ADONAN PANGSIT",
    "100211": "AYAM CINCANG",
    "100244": "LUMPIA UDANG",
    "100238": "KERUPUK MIE",
    "100239": "KULIT PANGSIT",
    "100245": "MIE",
    "100339": "CABAI FROZEN",
  };

  // Sparkline SVG Path generator
  function getSparklinePath(points: number[]): string {
    if (!points || points.length === 0) return "";
    const max = Math.max(...points) || 1;
    const min = Math.min(...points);
    const range = max - min || 1;
    const width = 100;
    const height = 28;
    const step = width / (points.length - 1 || 1);
    
    return points.map((p, idx) => {
      const x = idx * step;
      const y = height - ((p - min) / range) * height + 1;
      return `${idx === 0 ? 'M' : 'L'} ${x.toFixed(1)} ${y.toFixed(1)}`;
    }).join(" ");
  }

  // Derived Sparkline Points
  const sparklineWeeklyPoints = $derived(
    weeklyData.length > 0 
      ? weeklyData.map(d => d.count || 0) 
      : [100, 120, 115, 130, 125, 140, 135]
  );

  const sparklineShift3Points = $derived(
    timelineData.filter(t => t.shift === "3" || t.shift === 3).length > 0
      ? timelineData.filter(t => t.shift === "3" || t.shift === 3).map(d => d.count || 0)
      : []
  );

  const sparklineHourlyPoints = $derived(
    timelineData.length > 0
      ? timelineData.map(d => d.count || 0)
      : []
  );

  const avgHourly = $derived(
    timelineData.length > 0
      ? Math.round(timelineData.reduce((sum, item) => sum + (item.count || 0), 0) / timelineData.length)
      : 0
  );

  // Computed Top Products
  const topProducts = $derived(
    [...productDistData]
      .sort((a, b) => b.count - a.count)
      .slice(0, 6)
  );

  const maxDistCount = $derived(
    topProducts.length > 0 
      ? Math.max(...topProducts.map(d => d.count || 1)) 
      : 1
  );
</script>

{#if isLoading}<div class="loading-bar"></div>{/if}

<div class="dash">
  <!-- Data Origin Info Badge -->
  <div style="font-size: 10px; color: var(--text-muted); display: flex; flex-wrap: wrap; gap: 16px; padding: 6px 10px; border-bottom: 1px solid var(--border); margin-bottom: 4px; font-family: var(--font-mono);">
    <span>🔌 <strong>SOURCE:</strong> EXTERNAL DATABASE (MYSQL <code>10.201.40.2:3306</code>) &rarr; LOCAL POSTGRES SCRAPER</span>
    <span>⏱️ <strong>SYNC INTERVAL:</strong> BACKGROUND WORKER EVERY 30S</span>
  </div>

  <!-- Dashboard Grid: 6 KPI Cards + Weekly trend chart -->
  <div class="dashboard-grid">
    <!-- Left: 6 KPI Cards -->
    <div class="kpi-grid">
      <!-- Row 1 -->
      <div class="kpi-card" style="border-top: 3px solid var(--accent-2);">
        <div class="kpi-card-header">
          <span class="kpi-title">Total Produksi</span>
        </div>
        <div class="kpi-value">{summaryToday.total?.toLocaleString('id-ID') || 0}</div>
        <span class="kpi-subtext">scan hari ini</span>
      </div>

      <div class="kpi-card" style="border-top: 3px solid var(--green);">
        <div class="kpi-card-header">
          <span class="kpi-title">Shift 1 (Pagi)</span>
        </div>
        <div class="kpi-value" style="color:var(--green)">{summaryToday.per_shift?.['1']?.toLocaleString('id-ID') || 0}</div>
        <span class="kpi-subtext">06:00 - 14:00</span>
      </div>

      <div class="kpi-card" style="border-top: 3px solid var(--yellow);">
        <div class="kpi-card-header">
          <span class="kpi-title">Shift 2 (Siang)</span>
        </div>
        <div class="kpi-value" style="color:var(--yellow)">{summaryToday.per_shift?.['2']?.toLocaleString('id-ID') || 0}</div>
        <span class="kpi-subtext">14:00 - 22:00</span>
      </div>

      <!-- Row 2: Sparkline cards -->
      <div class="kpi-card" style="border-top: 3px solid var(--red);">
        <div class="kpi-card-header">
          <span class="kpi-title">Shift 3 (Malam)</span>
        </div>
        <div class="kpi-value" style="color:var(--red)">{summaryToday.per_shift?.['3']?.toLocaleString('id-ID') || 0}</div>
        <span class="kpi-subtext">22:00 - 06:00</span>
        {#if sparklineShift3Points.length >= 2}
        <div class="sparkline-container">
          <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
            <defs>
              <linearGradient id="shift3-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--red)" stop-opacity="0.15" />
                <stop offset="100%" stop-color="var(--red)" stop-opacity="0" />
              </linearGradient>
            </defs>
            <path d={getSparklinePath(sparklineShift3Points)} fill="none" stroke="var(--red)" stroke-width="1.2" />
            <path d="{getSparklinePath(sparklineShift3Points)} L 100 30 L 0 30 Z" fill="url(#shift3-grad)" />
          </svg>
        </div>
        {/if}
      </div>

      <div class="kpi-card" style="border-top: 3px solid var(--purple);">
        <div class="kpi-card-header">
          <span class="kpi-title">Rerata per Jam</span>
        </div>
        <div class="kpi-value" style="color:var(--purple)">{avgHourly} <span style="font-size:10px; font-weight:normal; color:var(--text-muted);">scans</span></div>
        <span class="kpi-subtext">average hourly output</span>
        {#if sparklineHourlyPoints.length >= 2}
        <div class="sparkline-container">
          <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
            <defs>
              <linearGradient id="hourly-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--purple)" stop-opacity="0.15" />
                <stop offset="100%" stop-color="var(--purple)" stop-opacity="0" />
              </linearGradient>
            </defs>
            <path d={getSparklinePath(sparklineHourlyPoints)} fill="none" stroke="var(--purple)" stroke-width="1.2" />
            <path d="{getSparklinePath(sparklineHourlyPoints)} L 100 30 L 0 30 Z" fill="url(#hourly-grad)" />
          </svg>
        </div>
        {/if}
      </div>

      <div class="kpi-card" style="border-top: 3px solid var(--orange);">
        <div class="kpi-card-header">
          <span class="kpi-title">Terakhir Sinkron</span>
        </div>
        <div class="kpi-value" style="font-size:16px; line-height: 28px; color:var(--orange);">{summaryToday.last_update || '--:--:--'}</div>
        <span class="kpi-subtext">auto sync (30s)</span>
        <div class="sparkline-container">
          <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
            <defs>
              <linearGradient id="weekly-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--orange)" stop-opacity="0.15" />
                <stop offset="100%" stop-color="var(--orange)" stop-opacity="0" />
              </linearGradient>
            </defs>
            <path d={getSparklinePath(sparklineWeeklyPoints)} fill="none" stroke="var(--orange)" stroke-width="1.2" />
            <path d="{getSparklinePath(sparklineWeeklyPoints)} L 100 30 L 0 30 Z" fill="url(#weekly-grad)" />
          </svg>
        </div>
      </div>
    </div>

    <!-- Right: 7 Days Trend Chart -->
    <div class="card" style="min-height: 250px; display:flex; flex-direction:column; justify-content:space-between;">
      <span class="kpi-title" style="margin-bottom:8px;">Volume Produksi 7 Hari Terakhir</span>
      <div style="position: relative; flex:1; height: 180px;">
        <canvas bind:this={canvasWeekly}></canvas>
      </div>
    </div>
  </div>

  <!-- Timeline hourly production chart -->
  <div class="card">
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom: 10px; border-bottom: 1px solid var(--border); padding-bottom: 8px;">
      <div style="display:flex; align-items:center; gap:6px;">
        <span class="section-dot" style="background:var(--accent); width:6px; height:6px; border-radius:50%; display:inline-block;"></span>
        <span style="font-family:var(--font-mono); font-size:10px; font-weight:700; text-transform:uppercase;">Timeline Produksi per Jam</span>
      </div>
      <div style="display:flex; gap:6px; align-items:center;">
        <input type="date" bind:value={selectedDate} onchange={loadTimeline} class="filter-pill" style="padding:4px 10px;" />
        <button class="btn btn-ghost" onclick={loadTimeline} style="border-radius: 9999px; height: 26px; padding: 0 10px;">CARI</button>
      </div>
    </div>
    <div class="chart-wrap" style="height: 180px;">
      <canvas bind:this={canvasTimeline}></canvas>
    </div>
  </div>

  <!-- Bottom: Shift productivity & Top product distribution side-by-side -->
  <div class="bottom-lists-grid">
    <!-- Left Column: Shift Productivity ratios -->
    <div class="topic-list-card">
      <div class="topic-list-title">Rasio Produktivitas Shift Hari Ini</div>
      <div class="topic-list">
        {#each shiftsList as s}
          {@const cnt = summaryToday.per_shift?.[s] || 0}
          {@const total = summaryToday.total || 1}
          {@const pct = Math.round((cnt / total) * 100)}
          {@const color = shiftColors[s as keyof typeof shiftColors]}
          
          <div class="topic-item">
            <div class="topic-meta">
              <div class="topic-thumb" style="color:{color}; font-weight: 700;">S{s}</div>
              <div class="topic-info">
                <span class="topic-name">{shiftLabels[s as keyof typeof shiftLabels]}</span>
                <span class="topic-subname">Today's scan allocation</span>
              </div>
            </div>
            <div class="topic-bar-container">
              <div class="topic-bar-outer">
                <div class="topic-bar-inner" style="width: {pct}%; background: {color};"></div>
              </div>
              <span class="topic-percentage" style="color: {color};">
                {cnt.toLocaleString('id-ID')} ({pct}%)
              </span>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Right Column: Product distribution progress list -->
    <div class="topic-list-card">
      <div class="topic-list-title">Top Distribusi Produk (30 Hari)</div>
      <div class="topic-list">
        {#each topProducts as d, idx}
          {@const name = productCodeToName[d.kode_produk] || 'UNKNOWN'}
          {@const pct = Math.round((d.count / maxDistCount) * 100)}
          
          <div class="topic-item">
            <div class="topic-meta">
              <div class="topic-thumb">P{idx + 1}</div>
              <div class="topic-info">
                <span class="topic-name">{name}</span>
                <span class="topic-subname">CODE: {d.kode_produk}</span>
              </div>
            </div>
            <div class="topic-bar-container">
              <div class="topic-bar-outer">
                <div class="topic-bar-inner" style="width: {pct}%; background: linear-gradient(90deg, var(--accent) 0%, var(--accent-2) 100%);"></div>
              </div>
              <span class="topic-percentage">{d.count?.toLocaleString('id-ID')} scans</span>
            </div>
          </div>
        {:else}
          <div style="padding: 40px; text-align:center; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">
            Menunggu data distribusi produk...
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>
