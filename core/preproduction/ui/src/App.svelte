<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import Analytics from "./Analytics.svelte";
  import Conveyor from "./Conveyor.svelte";
  import { Chart, registerables } from "chart.js";
  Chart.register(...registerables);

  const FORMING_BASE = "";
  type Module = "mdcw" | "sp" | "analytics" | "conveyor";

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

  let conveyorRecords: any[] = $state([]);
  let detailConveyorSearch = $state("");
  let detailBatchSearch = $state("");
  let detailShiftFilter = $state("all");

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
    if (mod === "analytics" || mod === "conveyor") {
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
    conveyorRecords = [];
    detailConveyorSearch = "";
    detailBatchSearch = "";
    detailShiftFilter = "all";
    await fetchData();
  }

  async function fetchData() {
    if (activeModule === "mdcw") {
      await fetchFilters();
      await Promise.all([fetchRecords(), fetchConveyorRecords(), fetchDailyStats()]);
      clearInterval(pollInterval);
      pollInterval = setInterval(async () => {
        await Promise.all([fetchRecords(), fetchConveyorRecords(), fetchDailyStats()]);
      }, 15000);
      return;
    }
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
      const isMdcw = activeModule === "mdcw";
      if (activeFilter !== "all") {
        params.set(isMdcw ? "prefix" : "session_id", activeFilter);
      }
      if (isMdcw) {
        if (statusFilter !== "all") params.set("status", statusFilter);
        params.set("sort", sortBy);
      }
      if (startDate) params.set("start_date", startDate);
      if (endDate) params.set("end_date", endDate);
      const path = moduleApi("/data?");
      const data = await api(path + params);
      records = data || [];
    } finally {
      isLoading = false;
    }
  }

  async function fetchConveyorRecords() {
    isLoading = true;
    try {
      const params = new URLSearchParams();
      if (detailConveyorSearch) params.set("kode_produk", detailConveyorSearch);
      if (detailBatchSearch) params.set("kode_batch", detailBatchSearch);
      if (detailShiftFilter !== "all") params.set("shift", detailShiftFilter);
      if (startDate) params.set("start_date", startDate);
      if (endDate) params.set("end_date", endDate);
      const data = await api("/api/mdcw/conveyor-data?" + params);
      conveyorRecords = data || [];
    } catch (e) {
      console.error("Error fetching conveyor records", e);
    } finally {
      isLoading = false;
    }
  }

  async function fetchSummary() {
    try { summaries = (await api(moduleApi("/summary"))) || []; } catch {}
  }

  async function fetchFilters() {
    try {
      const path = moduleApi(activeModule === "mdcw" ? "/prefixes" : "/sessions");
      filters = (await api(path)) || [];
    } catch {}
  }

  async function fetchDailyStats() {
    try {
      mdcwDailyStats = (await api("/api/mdcw/daily-stats")) || [];
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

  function getDaysRemaining(tglStr: string): number | null {
    if (!tglStr || tglStr.length !== 6) return null;
    const dd = parseInt(tglStr.slice(0, 2));
    const mm = parseInt(tglStr.slice(2, 4)) - 1;
    const yy = 2000 + parseInt(tglStr.slice(4, 6));
    const expiry = new Date(yy, mm, dd);
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const diffTime = expiry.getTime() - today.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    return diffDays;
  }

  function clearFilter() {
    activeFilter = "all"; statusFilter = "all"; sortBy = "newest"; startDate = ""; endDate = "";
    detailConveyorSearch = ""; detailBatchSearch = ""; detailShiftFilter = "all";
    if (activeModule === "mdcw") {
      fetchRecords();
      fetchConveyorRecords();
    } else {
      fetchRecords();
    }
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

  // =========================================================================
  // LOGIKA TAMBAHAN UNTUK DASHBOARD MDCW LENGKAP (TAB 1, 2, 3)
  // =========================================================================
  let activeMdcwSubTab = $state("ringkasan"); // "ringkasan" | "qc_mesin" | "per_shift" | "detail"
  let mdcwDailyStats = $state<any[]>([]);
  let selectedShift = $state("all"); // Shift filter untuk Tab 3

  // Reference Maps
  const prefixToProductCode: Record<string, string> = {
    "MDCW1 (UK)": "100294",
    "MDCW2 (Siomay)": "100256",
    "MDCW3 (Pentol)": "100286",
    "MDCW4 (AP)": "100209",
    "MDCW5 (ACIN)": "100211",
    "MDCW6 (Lumpia)": "100244",
  };

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

  const prefixToStandardWeight: Record<string, number> = {
    "MDCW1 (UK)": 911.5,
    "MDCW2 (Siomay)": 729.0,
    "MDCW3 (Pentol)": 609.0,
    "MDCW4 (AP)": 1530.0,
    "MDCW5 (ACIN)": 1018.5,
    "MDCW6 (Lumpia)": 321.0,
  };

  // Chart References
  let chartTrend: Chart | null = null;
  let chartReject: Chart | null = null;

  let canvasTrend = $state<HTMLCanvasElement | null>(null);
  let canvasReject = $state<HTMLCanvasElement | null>(null);

  // Helper to resolve chart colors based on current theme
  const getChartColors = (th: string) => {
    const isDark = th === "dark";
    return {
      text: isDark ? "#a1a1aa" : "#4b5563",
      grid: isDark ? "rgba(255, 255, 255, 0.04)" : "rgba(0, 0, 0, 0.04)",
      tick: isDark ? "#71717a" : "#9ca3af",
    };
  };

  // Redraw charts when tab changes, theme changes, or data updates
  $effect(() => {
    const currentTheme = theme; // registers dependency
    if (activeModule === "mdcw" && activeMdcwSubTab === "ringkasan" && canvasTrend && mdcwDailyStats.length > 0) {
      setTimeout(initTrendChart, 50);
    }
    if (activeModule === "mdcw" && activeMdcwSubTab === "qc_mesin" && canvasReject && summaries.length > 0) {
      setTimeout(initRejectChart, 50);
    }
  });

  onDestroy(() => {
    if (chartTrend) chartTrend.destroy();
    if (chartReject) chartReject.destroy();
  });

  function initTrendChart() {
    if (chartTrend) chartTrend.destroy();
    if (!canvasTrend) return;
    const colors = getChartColors(theme);
    const labels = mdcwDailyStats.map(d => d.date?.slice(5) || "");
    chartTrend = new Chart(canvasTrend, {
      type: "line",
      data: {
        labels,
        datasets: [
          { label: "OK", data: mdcwDailyStats.map(d => d.ok_count || 0), borderColor: "#10b981", backgroundColor: "rgba(16,185,129,0.04)", fill: true, tension: 0.3, pointRadius: 2, borderWidth: 1.5 },
          { label: "UNDER", data: mdcwDailyStats.map(d => d.under_count || 0), borderColor: "#3b82f6", backgroundColor: "rgba(59,130,246,0.04)", fill: true, tension: 0.3, pointRadius: 2, borderWidth: 1.5 },
          { label: "OVER", data: mdcwDailyStats.map(d => d.over_count || 0), borderColor: "#f59e0b", backgroundColor: "rgba(245,158,11,0.04)", fill: true, tension: 0.3, pointRadius: 2, borderWidth: 1.5 },
          { label: "METAL", data: mdcwDailyStats.map(d => d.metal_count || 0), borderColor: "#8b5cf6", backgroundColor: "rgba(139,92,246,0.04)", fill: true, tension: 0.3, pointRadius: 2, borderWidth: 1.5 },
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { position: "top", labels: { color: colors.text, boxWidth: 8, padding: 8, font: { size: 8, family: 'JetBrains Mono' } } } },
        scales: {
          x: { grid: { display: false }, ticks: { color: colors.tick, font: { size: 8 } } },
          y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.tick, font: { size: 8 } } }
        }
      }
    });
  }

  function initRejectChart() {
    if (chartReject) chartReject.destroy();
    if (!canvasReject) return;
    const colors = getChartColors(theme);
    const labels = summaries.map(s => s.prefix?.replace("MDCW", "") || "");
    chartReject = new Chart(canvasReject, {
      type: "bar",
      data: {
        labels,
        datasets: [
          { label: "UNDER", data: summaries.map(s => s.under_count || 0), backgroundColor: "#3b82f6", borderRadius: 2 },
          { label: "OVER", data: summaries.map(s => s.over_count || 0), backgroundColor: "#f59e0b", borderRadius: 2 },
          { label: "METAL", data: summaries.map(s => s.metal_count || 0), backgroundColor: "#8b5cf6", borderRadius: 2 }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { position: "top", labels: { color: colors.text, boxWidth: 8, padding: 8, font: { size: 8, family: 'JetBrains Mono' } } } },
        scales: {
          x: { grid: { display: false }, ticks: { color: colors.tick, font: { size: 8 } } },
          y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.tick, font: { size: 8 } } }
        }
      }
    });
  }

  // OEE Logic
  function getOeeForPrefix(pfx: string, shiftFilter: string = "all") {
    const pfxRecs = records.filter(r => {
      const matchPfx = r.prefix === pfx;
      if (shiftFilter === "all") return matchPfx;
      const hour = parseInt(r.ts?.slice(11, 13) || "0");
      let shift = 3;
      if (hour >= 7 && hour < 15) shift = 1;
      else if (hour >= 15 && hour < 23) shift = 2;
      return matchPfx && shift.toString() === shiftFilter;
    });

    if (pfxRecs.length === 0) {
      return { availability: 0, performance: 0, quality: 0, oee: 0, uptime: 0, idle: 0, downtime: 0 };
    }

    const sorted = [...pfxRecs].sort((a, b) => new Date(a.ts).getTime() - new Date(b.ts).getTime());
    let uptime = 0;
    let idle = 0;
    let downtime = 0;
    
    for (let i = 0; i < sorted.length - 1; i++) {
      const cur = sorted[i];
      const next = sorted[i+1];
      const diff = (new Date(next.ts).getTime() - new Date(cur.ts).getTime()) / 1000;
      const clampedDiff = Math.min(diff, 3600); // 1 hour max gap
      
      if (cur.reg5 === 8) {
        downtime += clampedDiff;
      } else if (cur.reg5 === 9 || cur.reg5 === 90) {
        idle += clampedDiff;
      } else {
        uptime += clampedDiff;
      }
    }

    const totalTime = uptime + downtime + idle;
    const availability = (uptime + idle) > 0 ? ((uptime + idle) / (totalTime || 1)) * 100 : 100;

    const ok = pfxRecs.filter(r => [41, 521, 553].includes(r.reg5)).length;
    const totalQC = pfxRecs.filter(r => [41, 521, 553, 25, 73, 8201].includes(r.reg5)).length;
    const quality = totalQC > 0 ? (ok / totalQC) * 100 : 100;

    const spanHours = sorted.length > 1 ? (new Date(sorted[sorted.length-1].ts).getTime() - new Date(sorted[0].ts).getTime()) / (1000 * 3600) : 0.05;
    const expectedThroughput = Math.max(1, Math.round(spanHours * 500));
    const performance = Math.min(100, (pfxRecs.length / expectedThroughput) * 100);

    const oee = (availability * performance * quality) / 10000;

    return { availability, performance, quality, oee, uptime, idle, downtime };
  }

  // Combined Stats & OEE average
  const overallStats = $derived.by(() => {
    let output = 0;
    let ok = 0;
    let reject = 0;
    let metal = 0;
    for (const s of summaries) {
      output += s.total_count || 0;
      ok += s.ok_count || 0;
      reject += (s.under_count || 0) + (s.over_count || 0) + (s.metal_count || 0);
      metal += s.metal_count || 0;
    }
    const passRate = output > 0 ? (ok / output) * 100 : 0;
    const bbKritis = conveyorRecords.filter(c => {
      const days = getDaysRemaining(c.tanggal_best_before);
      return days !== null && days <= 30;
    }).length;

    const prefixes = ["MDCW1 (UK)", "MDCW2 (Siomay)", "MDCW3 (Pentol)", "MDCW4 (AP)", "MDCW5 (ACIN)", "MDCW6 (Lumpia)"];
    let oeeSum = 0;
    let activeCount = 0;
    for (const p of prefixes) {
      const stats = getOeeForPrefix(p);
      const hasData = records.some(r => r.prefix === p);
      if (hasData) {
        oeeSum += stats.oee;
        activeCount++;
      }
    }
    const avgOee = activeCount > 0 ? oeeSum / activeCount : 0;

    return { output, ok, passRate, reject, metal, bbKritis, avgOee };
  });

  // Priority Alerts
  const priorityAlerts = $derived.by(() => {
    const alerts: { type: "critical" | "warning"; msg: string }[] = [];
    
    if (overallStats.metal > 0) {
      alerts.push({
        type: "critical",
        msg: `🚨 KONTAMINASI LOGAM: Sensor mendeteksi ${overallStats.metal} reject logam (METAL) hari ini!`
      });
    }

    let minDays: number | null = null;
    let minBatch = "";
    for (const c of conveyorRecords) {
      const days = getDaysRemaining(c.tanggal_best_before);
      if (days !== null) {
        if (minDays === null || days < minDays) {
          minDays = days;
          minBatch = c.kode_batch;
        }
      }
    }
    if (minDays !== null && minDays <= 30) {
      alerts.push({
        type: minDays <= 7 ? "critical" : "warning",
        msg: `⚠️ BB HAMPIR KADALUWARSA: Batch ${minBatch} habis sisa umur ${minDays} hari!`
      });
    }

    for (const s of summaries) {
      const tot = s.total_count || 0;
      const rej = (s.under_count || 0) + (s.over_count || 0) + (s.metal_count || 0);
      if (tot > 15) {
        const rate = (rej / tot) * 100;
        if (rate > 15) {
          alerts.push({
            type: "warning",
            msg: `📉 REJECT TINGGI: Lini ${s.prefix} terdeteksi reject rate tinggi (${rate.toFixed(1)}%)!`
          });
        }
      }
    }

    if (alerts.length === 0) {
      alerts.push({
        type: "warning",
        msg: "✅ KONDISI NORMAL: Seluruh checkweigher & conveyor berjalan dalam rentang aman."
      });
    }

    return alerts.slice(0, 3);
  });

  // Grammage Drift Calibration Analysis
  const gramasiDriftStats = $derived.by(() => {
    const list: { kode: string; nama: string; std: number; avg: number; drift: number; status: string; statusCls: string }[] = [];
    const uniqueCodes = [...new Set(conveyorRecords.map(c => c.kode_produk))];
    
    for (const code of uniqueCodes) {
      const cRec = conveyorRecords.find(c => c.kode_produk === code);
      if (!cRec) continue;
      
      const std = cRec.gramasi_pack || 0;
      const name = productCodeToName[code] || "Produk " + code;
      
      let prefix = "";
      for (const [pfx, cde] of Object.entries(prefixToProductCode)) {
        if (cde === code) { prefix = pfx; break; }
      }
      if (!prefix) continue;
      
      const pfxRecs = records.filter(r => {
        const matchPfx = r.prefix === prefix;
        if (selectedShift === "all") return matchPfx;
        const hour = parseInt(r.ts?.slice(11, 13) || "0");
        let shift = 3;
        if (hour >= 7 && hour < 15) shift = 1;
        else if (hour >= 15 && hour < 23) shift = 2;
        return matchPfx && shift.toString() === selectedShift;
      });
      
      const validRecs = pfxRecs.filter(r => r.reg114 > 0);
      if (validRecs.length === 0) continue;
      
      const sum = validRecs.reduce((a, r) => a + r.reg114, 0);
      const avg = sum / (validRecs.length * 10);
      
      const drift = avg - std;
      const limit = std * 0.02;
      
      let status = "Stabil";
      let statusCls = "badge-green";
      if (Math.abs(drift) > limit) {
        status = "Drift (Kalibrasi!)";
        statusCls = "badge-red";
      } else if (Math.abs(drift) > limit * 0.5) {
        status = "Warning";
        statusCls = "badge-yellow";
      }
      
      list.push({ kode: code, nama: name, std, avg, drift, status, statusCls });
    }
    return list;
  });

  let shiftRecs = $derived.by(() => {
    return records.filter(r => {
      if (selectedShift === "all") return true;
      const hour = parseInt(r.ts?.slice(11, 13) || "0");
      let shift = 3;
      if (hour >= 7 && hour < 15) shift = 1;
      else if (hour >= 15 && hour < 23) shift = 2;
      return shift.toString() === selectedShift;
    });
  });

  let okShift = $derived(shiftRecs.filter(r => [41, 521, 553].includes(r.reg5)).length);
  let rejectShift = $derived(shiftRecs.length - okShift);
  let passRateShift = $derived(shiftRecs.length > 0 ? (okShift / shiftRecs.length) * 100 : 100);

  // Sparkline data generator for bottom row KPI cards
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

  const sparklineOutputPoints = $derived(mdcwDailyStats.length > 0 ? mdcwDailyStats.map(d => d.ok_count || 0) : [10, 15, 12, 18, 14, 22, 20]);
  const sparklineMetalPoints = $derived(mdcwDailyStats.length > 0 ? mdcwDailyStats.map(d => d.metal_count || 0) : [0, 0, 1, 0, 0, 2, 0]);
  const sparklineOeePoints = $derived(mdcwDailyStats.length > 0 ? mdcwDailyStats.map(d => d.ok_count ? Math.round((d.ok_count / (d.total_count || 1)) * 100) : 80) : [85, 88, 84, 89, 87, 91, 90]);
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
            <label class="form-label" for="username_input">OPERATOR</label>
            <input id="username_input" type="text" bind:value={username} required autocomplete="username" class="form-input" />
          </div>
          <div class="form-group">
            <label class="form-label" for="password_input">ACCESS CODE</label>
            <input id="password_input" type="password" bind:value={password} required autocomplete="current-password" class="form-input" />
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
    <!-- Left Sidebar Shell -->
    <aside class="sidebar">
      <div class="sidebar-top">
        <div class="sidebar-logo">PRODUKSI</div>
        <nav class="sidebar-menu">
          <button class="sidebar-btn" class:active={activeModule === "mdcw"} onclick={() => switchModule("mdcw")}>
            <span>📊</span> MDCW
          </button>
          <button class="sidebar-btn" class:active={activeModule === "sp"} onclick={() => switchModule("sp")}>
            <span>🔍</span> SP
          </button>
          <button class="sidebar-btn" class:active={activeModule === "analytics"} onclick={() => switchModule("analytics")}>
            <span>📈</span> ANALYTICS
          </button>
          <button class="sidebar-btn" class:active={activeModule === "conveyor"} onclick={() => switchModule("conveyor")}>
            <span>⚙️</span> CONVEYOR
          </button>
        </nav>
      </div>
      <div class="sidebar-bottom">
        <span class="live-dot" style="margin-left:6px;">LIVE MONITOR</span>
        <div style="display:flex; justify-content:space-between; align-items:center; padding: 0 4px;">
          <button class="theme-btn" onclick={toggleTheme} title="Toggle theme">{theme === "dark" ? "☀" : "☾"}</button>
          <button onclick={handleLogout} style="font-family:var(--font-mono); font-size:9px; color:var(--text-muted); cursor:pointer;">LOGOUT</button>
        </div>
      </div>
    </aside>

    <!-- Main Right Panel -->
    <main class="main-content">
      {#if activeModule === "analytics"}
        <div style="width:100%; max-width:1400px; margin:0 auto;">
          <Analytics {api} {token} {theme} />
        </div>
      {:else if activeModule === "conveyor"}
        <div style="width:100%; max-width:1400px; margin:0 auto;">
          <Conveyor {api} {token} {theme} />
        </div>
      {:else if activeModule === "sp"}
        <!-- SP BARCODE MONITOR VIEW -->
        <div style="font-size: 10px; color: var(--text-muted); display: flex; gap: 16px; font-family: var(--font-mono); border-bottom: 1px solid var(--border); padding-bottom: 8px; margin-bottom: 4px;">
          <span>🔌 <strong>SOURCE:</strong> BARCODE SCANNERS (MQTT Broker <code>emqx:1883</code> &rarr; Topic <code>SP_data</code>) &rarr; LOCAL POSTGRES</span>
          <span>⏱️ <strong>UPDATE INTERVAL:</strong> REAL-TIME EVENTS</span>
        </div>
        
        <div class="main-grid-layout" style="width:100%;">
          <div class="side-col left-col">
            <div style="display:flex; flex-direction:column; gap:12px; margin-bottom:16px;">
              <div class="card">
                <div class="card-label" style="margin-bottom:8px;">Session</div>
                <div style="display:flex; gap:3px; flex-wrap:wrap;">
                  <button class="btn" class:btn-primary={activeFilter === "all"} class:btn-ghost={activeFilter !== "all"}
                    style="font-size:9px; padding:4px 8px;" onclick={() => { activeFilter = "all"; fetchRecords(); }}>ALL</button>
                  {#each filters as f}
                    <button class="btn" class:btn-primary={activeFilter === f} class:btn-ghost={activeFilter !== f}
                      style="font-size:9px; padding:4px 8px;" onclick={() => { activeFilter = f; fetchRecords(); }}>{f}</button>
                  {/each}
                </div>
              </div>

              <div class="card">
                <div class="card-label" style="margin-bottom:8px;">Tanggal</div>
                <div style="display:flex; flex-direction:column; gap:8px;">
                  <input type="date" bind:value={startDate} onchange={fetchRecords} style="font-size:10px; width:100%;" />
                  <span style="color:var(--text-muted); text-align:center; font-size:9px;">s/d</span>
                  <input type="date" bind:value={endDate} onchange={fetchRecords} style="font-size:10px; width:100%;" />
                </div>
              </div>

              <div class="card">
                <div style="display:flex; gap:6px; flex-wrap:wrap;">
                  <button class="btn btn-success" onclick={exportCSV} style="font-size:9px; flex:1; justify-content:center;">CSV</button>
                  <button class="btn btn-ghost" onclick={fetchRecords} style="font-size:9px; flex:1; justify-content:center;">↻</button>
                  <button class="btn btn-ghost" onclick={clearFilter} style="font-size:9px; color:var(--red); flex:1; justify-content:center;">×</button>
                </div>
              </div>
            </div>
          </div>

          <div class="center-col">
            <div class="card">
              <div style="overflow-x:auto;">
                <table style="width:100%; border-collapse:collapse;">
                  <thead>
                    <tr style="border-bottom:1px solid var(--border);">
                      {#each ["ID", "SESSION", "DATA", "TS"] as h}
                        <th style="text-align:left; padding:6px 10px; font-family:var(--font-mono); font-size:8px; font-weight:600; color:var(--text-muted); text-transform:uppercase;">{h}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody>
                    {#each records as r}
                      <tr style="border-bottom:1px solid var(--border); transition:background 0.1s;"
                        onmouseenter={(e: any) => (e.currentTarget.style.background = "var(--surface-2)")}
                        onmouseleave={(e: any) => (e.currentTarget.style.background = "")}>
                        <td style="padding:6px 10px; font-family:var(--font-mono); font-size:9px; color:var(--text-muted);">{r.id}</td>
                        <td style="padding:6px 10px;"><span class="badge badge-blue" style="font-size:8px;">{r.session_id}</span></td>
                        <td style="padding:6px 10px; font-family:var(--font-mono); font-size:10px; font-weight:600; color:var(--accent-2);">{r.data}</td>
                        <td style="padding:6px 10px; font-family:var(--font-mono); font-size:9px; white-space:nowrap;">{r.ts || r.created_at}</td>
                      </tr>
                    {:else}
                      <tr><td colspan="4" style="padding:32px; text-align:center; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">Tidak ada data</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          <div class="right-col">
            <div class="summary-grid">
              {#each summaries as s}
                <div class="card" style="border-left:2px solid var(--accent-2); padding:12px;">
                  <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:6px;">
                    <span class="card-label" style="font-size:10px; font-weight:700; color:var(--text);">{s.session_id}</span>
                    <span style="font-size:9px; color:var(--text-muted); font-family:var(--font-mono);">{s.total_count || 0}</span>
                  </div>
                  <div style="font-size:18px; font-weight:700; color:var(--accent-2); font-family:var(--font-mono); margin-bottom:4px;">
                    {s.total_count || 0} scans
                  </div>
                  <div style="display:flex; justify-content:space-between; font-size:8px; color:var(--text-muted); border-top:1px solid var(--border); padding-top:6px;">
                    <span>First: {s.first_scan ? s.first_scan.slice(0,10) : "-"}</span>
                    <span>Last: {s.last_scan ? s.last_scan.slice(0,10) : "-"}</span>
                  </div>
                </div>
              {:else}
                <div class="card" style="text-align:center; padding:20px; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">
                  Menunggu data produksi...
                </div>
              {/each}
            </div>
          </div>
        </div>
      {:else if activeModule === "mdcw"}
        <!-- GLOBAL FILTER PILLS ROW -->
        <div class="filter-row card" style="padding: 10px 14px; align-items: center;">
          <span style="font-family: var(--font-mono); font-size: 9px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Filters &rarr;</span>
          
          <select class="filter-pill" bind:value={detailShiftFilter} onchange={fetchConveyorRecords}>
            <option value="all">Shift: All</option>
            <option value="1">Shift 1 (Pagi)</option>
            <option value="2">Shift 2 (Siang)</option>
            <option value="3">Shift 3 (Malam)</option>
          </select>
          
          <select class="filter-pill" bind:value={activeFilter} onchange={fetchRecords}>
            <option value="all">Machine: All</option>
            {#each filters as f}
              <option value={f}>{f}</option>
            {/each}
          </select>
          
          <select class="filter-pill" bind:value={statusFilter} onchange={fetchRecords}>
            <option value="all">Status: All</option>
            <option value="ok">OK</option>
            <option value="metal">Metal</option>
            <option value="under">Under</option>
            <option value="over">Over</option>
            <option value="mati">Mati</option>
            <option value="idle">Idle</option>
          </select>
          
          <input type="date" class="filter-pill" bind:value={startDate} onchange={fetchData} title="Start Date" />
          <span style="font-size: 10px; color: var(--text-muted);">to</span>
          <input type="date" class="filter-pill" bind:value={endDate} onchange={fetchData} title="End Date" />
          
          <input type="text" placeholder="Prod Code..." class="filter-pill" style="width: 100px; padding: 4px 10px;" bind:value={detailConveyorSearch} oninput={fetchConveyorRecords} />
          <input type="text" placeholder="Batch Code..." class="filter-pill" style="width: 100px; padding: 4px 10px;" bind:value={detailBatchSearch} oninput={fetchConveyorRecords} />
          
          <button class="btn btn-ghost" onclick={fetchData} title="Reload Data" style="border-radius: 9999px; height: 26px; padding: 0 8px;">↻</button>
          <button class="btn btn-ghost" onclick={clearFilter} title="Reset Filter" style="border-radius: 9999px; height: 26px; padding: 0 8px; color: var(--red); border-color: var(--border);">×</button>
        </div>

        <!-- SUB NAVIGATION TAB BAR -->
        <div class="card" style="padding: 0 14px; display: flex; justify-content: space-between; align-items: center;">
          <div class="subtab-group" style="border-bottom: none; padding: 0;">
            <button class="subtab-btn" class:active={activeMdcwSubTab === "ringkasan"} onclick={() => activeMdcwSubTab = "ringkasan"}>RINGKASAN</button>
            <button class="subtab-btn" class:active={activeMdcwSubTab === "qc_mesin"} onclick={() => activeMdcwSubTab = "qc_mesin"}>QC & MESIN</button>
            <button class="subtab-btn" class:active={activeMdcwSubTab === "per_shift"} onclick={() => activeMdcwSubTab = "per_shift"}>PER SHIFT</button>
            <button class="subtab-btn" class:active={activeMdcwSubTab === "detail"} onclick={() => activeMdcwSubTab = "detail"}>DETAIL LOGS</button>
          </div>
          <span style="font-family: var(--font-mono); font-size: 8px; color: var(--text-muted); font-weight: 700; text-transform: uppercase;">MDCW ENGINE</span>
        </div>

        <!-- Data Origin Info Badge -->
        <div style="font-size: 9px; color: var(--text-muted); display: flex; gap: 16px; padding: 0 10px; margin-top: -8px; font-family: var(--font-mono);">
          <span>🔌 <strong>SOURCE:</strong> WEIGHING SENSORS (MQTT <code>emqx:1883</code> &rarr; <code>production/mdcw</code>) &rarr; LOCAL POSTGRES</span>
          <span>⏱️ <strong>UPDATE:</strong> REAL-TIME PACKS INGESTION</span>
        </div>

        {#if activeMdcwSubTab === "ringkasan"}
          <!-- TAB 1: RINGKASAN VIEW -->
          <div class="dashboard-grid">
            <!-- Left: 6 KPI Cards Grid -->
            <div class="kpi-grid">
              <div class="kpi-card" style="border-top: 3px solid var(--accent);">
                <div class="kpi-card-header">
                  <span class="kpi-title">Total Output</span>
                </div>
                <div class="kpi-value">{overallStats.output}</div>
                <span class="kpi-subtext">packs today</span>
              </div>
              <div class="kpi-card" style="border-top: 3px solid var(--green);">
                <div class="kpi-card-header">
                  <span class="kpi-title">Pass Rate</span>
                </div>
                <div class="kpi-value" style="color:var(--green)">{overallStats.passRate.toFixed(1)}%</div>
                <span class="kpi-subtext">target: &gt;95%</span>
              </div>
              <div class="kpi-card" style="border-top: 3px solid var(--yellow);">
                <div class="kpi-card-header">
                  <span class="kpi-title">Total Reject</span>
                </div>
                <div class="kpi-value" style="color:var(--yellow)">{overallStats.reject}</div>
                <span class="kpi-subtext">under/over/metal</span>
              </div>

              <!-- Bottom Row: KPI Cards with Sparkline lines -->
              <div class="kpi-card" style="border-top: 3px solid var(--red);">
                <div class="kpi-card-header">
                  <span class="kpi-title">Metal Alert</span>
                  {#if overallStats.metal > 0}<span class="badge badge-red" style="font-size:7px; animation: blink 1s infinite;">CRITICAL</span>{/if}
                </div>
                <div class="kpi-value" style="color:{overallStats.metal > 0 ? 'var(--red)' : 'var(--text)'}">{overallStats.metal}</div>
                <span class="kpi-subtext">kontaminasi logam</span>
                <div class="sparkline-container">
                  <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                    <defs>
                      <linearGradient id="metal-grad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stop-color="var(--red)" stop-opacity="0.15" />
                        <stop offset="100%" stop-color="var(--red)" stop-opacity="0" />
                      </linearGradient>
                    </defs>
                    <path d={getSparklinePath(sparklineMetalPoints)} fill="none" stroke="var(--red)" stroke-width="1.2" />
                    <path d="{getSparklinePath(sparklineMetalPoints)} L 100 30 L 0 30 Z" fill="url(#metal-grad)" />
                  </svg>
                </div>
              </div>
              <div class="kpi-card" style="border-top: 3px solid var(--purple);">
                <div class="kpi-card-header">
                  <span class="kpi-title">OEE Rata-rata</span>
                </div>
                <div class="kpi-value" style="color:var(--purple)">{overallStats.avgOee.toFixed(1)}%</div>
                <span class="kpi-subtext">availability+perf+quality</span>
                <div class="sparkline-container">
                  <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                    <defs>
                      <linearGradient id="oee-grad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stop-color="var(--purple)" stop-opacity="0.15" />
                        <stop offset="100%" stop-color="var(--purple)" stop-opacity="0" />
                      </linearGradient>
                    </defs>
                    <path d={getSparklinePath(sparklineOeePoints)} fill="none" stroke="var(--purple)" stroke-width="1.2" />
                    <path d="{getSparklinePath(sparklineOeePoints)} L 100 30 L 0 30 Z" fill="url(#oee-grad)" />
                  </svg>
                </div>
              </div>
              <div class="kpi-card" style="border-top: 3px solid var(--orange);">
                <div class="kpi-card-header">
                  <span class="kpi-title">BB Kritis</span>
                  {#if overallStats.bbKritis > 0}<span class="badge badge-yellow" style="font-size:7px;">ATTN</span>{/if}
                </div>
                <div class="kpi-value" style="color:var(--orange)">{overallStats.bbKritis}</div>
                <span class="kpi-subtext">umur simpan &lt; 30d</span>
                <div class="sparkline-container">
                  <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                    <defs>
                      <linearGradient id="bb-grad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stop-color="var(--orange)" stop-opacity="0.15" />
                        <stop offset="100%" stop-color="var(--orange)" stop-opacity="0" />
                      </linearGradient>
                    </defs>
                    <path d={getSparklinePath(sparklineOutputPoints)} fill="none" stroke="var(--orange)" stroke-width="1.2" />
                    <path d="{getSparklinePath(sparklineOutputPoints)} L 100 30 L 0 30 Z" fill="url(#bb-grad)" />
                  </svg>
                </div>
              </div>
            </div>

            <!-- Right: Line Trend Chart -->
            <div class="card" style="min-height: 250px; display:flex; flex-direction:column; justify-content:space-between;">
              <span class="kpi-title" style="margin-bottom:8px;">Tren Produksi 7 Hari</span>
              <div style="position: relative; flex:1; height: 180px;">
                <canvas bind:this={canvasTrend}></canvas>
              </div>
            </div>
          </div>

          <!-- Bottom: List items progress grid -->
          <div class="bottom-lists-grid">
            <!-- Left Column: Live Machine Uptime & OEE Status -->
            <div class="topic-list-card">
              <div class="topic-list-title">Live Machine OEE & Status</div>
              <div class="topic-list">
                {#each ["MDCW1 (UK)", "MDCW2 (Siomay)", "MDCW3 (Pentol)", "MDCW4 (AP)", "MDCW5 (ACIN)", "MDCW6 (Lumpia)"] as pfx, idx}
                  {@const pfxSummary = summaries.find(s => s.prefix === pfx)}
                  {@const pfxRecs = records.filter(r => r.prefix === pfx)}
                  {@const latestRec = pfxRecs[0]}
                  {@const oeeStats = getOeeForPrefix(pfx)}
                  {@const isMati = !latestRec || latestRec.reg5 === 8}
                  {@const isIdle = latestRec && (latestRec.reg5 === 9 || latestRec.reg5 === 90)}
                  {@const stateText = isMati ? "MATI" : isIdle ? "IDLE" : "RUNNING"}
                  {@const stateColor = isMati ? "var(--red)" : isIdle ? "var(--yellow)" : "var(--green)"}
                  {@const oeeValue = oeeStats.oee * 100}
                  
                  <div class="topic-item">
                    <div class="topic-meta">
                      <div class="topic-thumb">M{idx+1}</div>
                      <div class="topic-info">
                        <span class="topic-name">{pfx}</span>
                        <span class="topic-subname" style="color: {stateColor}; font-weight:700;">{stateText} · {productCodeToName[prefixToProductCode[pfx]] || '-'}</span>
                      </div>
                    </div>
                    <div class="topic-bar-container">
                      <div class="topic-bar-outer">
                        <div class="topic-bar-inner" style="width: {oeeValue}%; background: linear-gradient(90deg, var(--accent) 0%, var(--accent-2) 100%);"></div>
                      </div>
                      <span class="topic-percentage">{oeeValue.toFixed(0)}% OEE</span>
                    </div>
                  </div>
                {/each}
              </div>
            </div>

            <!-- Right Column: Product Drift Deviation Progress list -->
            <div class="topic-list-card">
              <div class="topic-list-title">Audit Penyimpangan Gramasi (Drift)</div>
              <div class="topic-list">
                {#each gramasiDriftStats.slice(0, 6) as d}
                  {@const driftPct = Math.min(100, Math.max(0, 100 - Math.round((Math.abs(d.drift) / d.std) * 100 * 20)))}
                  {@const isDriftLimit = d.status.includes("Drift")}
                  {@const isWarning = d.status.includes("Warning")}
                  {@const barColor = isDriftLimit ? 'var(--red)' : isWarning ? 'var(--yellow)' : 'var(--green)'}
                  
                  <div class="topic-item">
                    <div class="topic-meta">
                      <div class="topic-thumb" style="color:{barColor}">{d.kode.slice(-2)}</div>
                      <div class="topic-info">
                        <span class="topic-name">{d.nama}</span>
                        <span class="topic-subname">std: {d.std.toFixed(1)}g · avg: {d.avg.toFixed(1)}g</span>
                      </div>
                    </div>
                    <div class="topic-bar-container">
                      <div class="topic-bar-outer">
                        <div class="topic-bar-inner" style="width: {driftPct}%; background: {barColor};"></div>
                      </div>
                      <span class="topic-percentage" style="color:{barColor};">
                        {d.drift >= 0 ? '+' : ''}{d.drift.toFixed(1)}g
                      </span>
                    </div>
                  </div>
                {:else}
                  <div style="padding: 40px; text-align:center; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">
                    Tidak ada produk aktif dalam audit drift
                  </div>
                {/each}
              </div>
            </div>
          </div>

          <!-- Alert Otomatis Prioritas Row -->
          <div class="card" style="margin-top:4px;">
            <div class="kpi-title" style="margin-bottom:10px; color:var(--red);">Alert Otomatis Prioritas</div>
            <div style="display: flex; flex-direction: column; gap: 8px;">
              {#each priorityAlerts as alert}
                <div style="padding: 10px; border-radius: var(--radius); background: {alert.type === 'critical' ? 'var(--red-dim)' : 'var(--yellow-dim)'}; border-left: 3px solid {alert.type === 'critical' ? 'var(--red)' : 'var(--yellow)'}; font-family: var(--font-mono); font-size: 10px; color: var(--text);">
                  {alert.msg}
                </div>
              {/each}
            </div>
          </div>

        {:else if activeMdcwSubTab === "qc_mesin"}
          <!-- TAB 2: QC & MESIN VIEW -->
          <div class="grid-split-1-2">
            <div style="display: flex; flex-direction: column; gap: 16px;">
              <!-- Reject Chart -->
              <div class="card" style="min-height: 250px; background: var(--surface);">
                <div class="card-label" style="margin-bottom:12px;">Reject Breakdown Per Mesin</div>
                <div style="position: relative; height: 210px;">
                  <canvas bind:this={canvasReject}></canvas>
                </div>
              </div>

              <!-- Uptime vs Idle vs Downtime per machine -->
              <div class="card" style="background: var(--surface);">
                <div class="card-label" style="margin-bottom:12px;">Durasi Uptime vs Idle vs Downtime (Timestamp-derived)</div>
                <div style="overflow-x:auto;">
                  <table style="width:100%; border-collapse:collapse;">
                    <thead>
                      <tr style="border-bottom:1px solid var(--border);">
                        {#each ["MESIN", "UPTIME (RUN)", "IDLE", "DOWNTIME (MATI)", "UPTIME RATIO"] as h}
                          <th style="text-align:left; padding:8px; font-family:var(--font-mono); font-size:8px; color:var(--text-muted); text-transform:uppercase;">{h}</th>
                        {/each}
                      </tr>
                    </thead>
                    <tbody>
                      {#each ["MDCW1 (UK)", "MDCW2 (Siomay)", "MDCW3 (Pentol)", "MDCW4 (AP)", "MDCW5 (ACIN)", "MDCW6 (Lumpia)"] as pfx}
                        {@const durations = getOeeForPrefix(pfx)}
                        {@const total = durations.uptime + durations.idle + durations.downtime || 1}
                        <tr style="border-bottom:1px solid var(--border);">
                          <td style="padding:8px; font-family:var(--font-mono); font-size:9px; font-weight:700;">{pfx}</td>
                          <td style="padding:8px; font-family:var(--font-mono); font-size:9px; color:var(--green);">{Math.round(durations.uptime)}s ({((durations.uptime/total)*100).toFixed(0)}%)</td>
                          <td style="padding:8px; font-family:var(--font-mono); font-size:9px; color:var(--yellow);">{Math.round(durations.idle)}s ({((durations.idle/total)*100).toFixed(0)}%)</td>
                          <td style="padding:8px; font-family:var(--font-mono); font-size:9px; color:var(--red);">{Math.round(durations.downtime)}s ({((durations.downtime/total)*100).toFixed(0)}%)</td>
                          <td style="padding:8px;">
                            <div style="display:flex; height:5px; border-radius:2px; overflow:hidden; background:var(--surface-2); max-width:100px;">
                              <div style="width:{((durations.uptime)/total)*100}%; background:var(--green);"></div>
                              <div style="width:{(durations.idle)/total}%; background:var(--yellow);"></div>
                              <div style="width:{(durations.downtime/total)*100}%; background:var(--red);"></div>
                            </div>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>

            <!-- OEE per Mesin Table -->
            <div class="card" style="background: var(--surface);">
              <div class="card-label" style="margin-bottom:12px;">OEE Tiga Komponen Otomatis Sensor</div>
              <div style="overflow-x:auto;">
                <table style="width:100%; border-collapse:collapse;">
                  <thead>
                    <tr style="border-bottom:1px solid var(--border);">
                      {#each ["MESIN", "AVAIL %", "PERF %", "QUAL %", "OEE %"] as h}
                        <th style="text-align:left; padding:8px; font-family:var(--font-mono); font-size:8px; color:var(--text-muted); text-transform:uppercase;">{h}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody>
                    {#each ["MDCW1 (UK)", "MDCW2 (Siomay)", "MDCW3 (Pentol)", "MDCW4 (AP)", "MDCW5 (ACIN)", "MDCW6 (Lumpia)"] as pfx}
                      {@const oeeStats = getOeeForPrefix(pfx)}
                      {@const oeeValue = oeeStats.oee * 100}
                      {@const oeeColor = oeeValue >= 85 ? "var(--green)" : oeeValue >= 60 ? "var(--yellow)" : "var(--red)"}
                      <tr style="border-bottom:1px solid var(--border);">
                        <td style="padding:8px; font-family:var(--font-mono); font-size:9px; font-weight:700;">{pfx}</td>
                        <td style="padding:8px; font-family:var(--font-mono); font-size:9px;">{oeeStats.availability.toFixed(0)}%</td>
                        <td style="padding:8px; font-family:var(--font-mono); font-size:9px;">{oeeStats.performance.toFixed(0)}%</td>
                        <td style="padding:8px; font-family:var(--font-mono); font-size:9px;">{oeeStats.quality.toFixed(0)}%</td>
                        <td style="padding:8px; font-family:var(--font-mono); font-size:10px; font-weight:700; color:{oeeColor};">{oeeValue.toFixed(1)}%</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

        {:else if activeMdcwSubTab === "per_shift"}
          <!-- TAB 3: PER SHIFT VIEW -->
          <div style="display:flex; flex-direction:column; gap:16px;">
            <!-- Shift Header Selector & Shift KPI Cards -->
            <div class="card" style="display:flex; flex-wrap:wrap; justify-content:space-between; align-items:center; gap:16px; background: var(--surface);">
              <div>
                <span class="card-label">Filter Analisis Shift</span>
                <div style="display:flex; gap:6px; margin-top:6px;">
                  {#each ["all", "1", "2", "3"] as sft}
                    <button class="btn" class:btn-primary={selectedShift === sft} class:btn-ghost={selectedShift !== sft} onclick={() => selectedShift = sft}>
                      {sft === "all" ? "SEMUA SHIFT" : "SHIFT " + sft}
                    </button>
                  {/each}
                </div>
              </div>
              
              <!-- Shift KPI Summary Cards -->
              <div style="display:flex; gap:12px;">
                <div class="card" style="padding:8px 16px; text-align:center; min-width:100px; background:var(--bg-2);">
                  <div class="card-label" style="font-size:8px;">Shift Output</div>
                  <div class="mono" style="font-size:16px; font-weight:700; color:var(--text);">{shiftRecs.length}</div>
                </div>
                <div class="card" style="padding:8px 16px; text-align:center; min-width:100px; background:var(--bg-2);">
                  <div class="card-label" style="font-size:8px;">Shift Pass Rate</div>
                  <div class="mono" style="font-size:16px; font-weight:700; color:var(--green);">{passRateShift.toFixed(1)}%</div>
                </div>
                <div class="card" style="padding:8px 16px; text-align:center; min-width:100px; background:var(--bg-2);">
                  <div class="card-label" style="font-size:8px;">Shift Reject</div>
                  <div class="mono" style="font-size:16px; font-weight:700; color:var(--yellow);">{rejectShift}</div>
                </div>
              </div>
            </div>

            <!-- Gramasi Aktual vs Standar (Audit & Drift Calibration) -->
            <div class="card" style="background: var(--surface);">
              <div class="card-label" style="margin-bottom:12px; display:flex; justify-content:space-between; align-items:center;">
                <span>Audit Penyimpangan Gramasi (Aktual vs Standar - Join Barcode)</span>
                <span class="badge badge-purple" style="font-size:8px;">Batas Toleransi Drift: 2%</span>
              </div>
              <div style="overflow-x:auto;">
                <table style="width:100%; border-collapse:collapse;">
                  <thead>
                    <tr style="border-bottom:1px solid var(--border);">
                      {#each ["KODE PRODUK", "NAMA PRODUK", "GRAMASI STANDAR BATCH", "RATA-RATA BERAT SENSOR", "SELISIH (DRIFT)", "STATUS KALIBRASI"] as h}
                        <th style="text-align:left; padding:8px 10px; font-family:var(--font-mono); font-size:8px; color:var(--text-muted); text-transform:uppercase;">{h}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody>
                    {#each gramasiDriftStats as d}
                      <tr style="border-bottom:1px solid var(--border);">
                        <td style="padding:8px 10px; font-family:var(--font-mono); font-size:9px; font-weight:700;">{d.kode}</td>
                        <td style="padding:8px 10px; font-family:var(--font-mono); font-size:9px; color:var(--text);">{d.nama}</td>
                        <td style="padding:8px 10px; font-family:var(--font-mono); font-size:9px;">{d.std.toFixed(1)}g</td>
                        <td style="padding:8px 10px; font-family:var(--font-mono); font-size:9px; font-weight:600; color:var(--purple);">{d.avg.toFixed(1)}g</td>
                        <td style="padding:8px 10px; font-family:var(--font-mono); font-size:9px; font-weight:700; color:{d.drift >= 0 ? 'var(--green)' : 'var(--red)'}">
                          {d.drift >= 0 ? '+' : ''}{d.drift.toFixed(1)}g ({((d.drift/d.std)*100).toFixed(2)}%)
                        </td>
                        <td style="padding:8px 10px;"><span class="badge {d.statusCls}">{d.status}</span></td>
                      </tr>
                    {:else}
                      <tr><td colspan="6" style="padding:32px; text-align:center; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">Tidak ada kecocokan data shift untuk audit gramasi</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

        {:else if activeMdcwSubTab === "detail"}
          <!-- TAB 4: DETAIL SPLIT SCREEN TABLES -->
          <div class="split-tables-wrapper grid-split-1-1" style="align-items: start;">
            <!-- Left Table: MDCW Sensor Logs -->
            <div class="card" style="padding: 14px; overflow:hidden; background: var(--surface);">
              <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:10px; border-bottom:1px solid var(--border); padding-bottom:6px;">
                <div style="display:flex; align-items:center; gap:6px;">
                  <span class="section-dot" style="background:var(--purple); width:6px; height:6px; border-radius:50%; display:inline-block;"></span>
                  <span style="font-family:var(--font-mono); font-size:10px; font-weight:700; text-transform:uppercase;">MDCW Sensor Real-Time Log</span>
                </div>
                <span class="live-dot" style="font-size:8px;">LIVE</span>
              </div>
              <div style="overflow-x:auto; max-height: 62vh;">
                <table style="width:100%; border-collapse:collapse;">
                  <thead>
                    <tr style="border-bottom:1px solid var(--border); position: sticky; top: 0; background: var(--surface); z-index: 10;">
                      {#each ["ID", "TS", "PREFIX", "BERAT", "PCK", "ST", "DT"] as h}
                        <th style="text-align:left; padding:6px 8px; font-family:var(--font-mono); font-size:8px; font-weight:600; color:var(--text-muted); text-transform:uppercase;">{h}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody>
                    {#each records as r}
                      {@const st = statusLabel(r.reg5)}
                      {@const dtCls = r.data_type === "VALID" ? "badge-green" : r.data_type === "TEST" ? "badge-blue" : r.data_type === "SPAM" ? "badge-red" : "badge-yellow"}
                      <tr style="border-bottom:1px solid var(--border); transition:background var(--transition);"
                          onmouseenter={(e: any) => (e.currentTarget.style.background = "var(--surface-2)")}
                          onmouseleave={(e: any) => (e.currentTarget.style.background = "")}>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; color:var(--text-muted);">{r.id}</td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; white-space:nowrap;">{r.ts}</td>
                        <td style="padding:6px 8px;"><span class="badge badge-blue" style="font-size:8px;">{r.prefix}</span></td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:10px; font-weight:600; color:var(--purple); white-space:nowrap;">{r.weight_formatted}</td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:10px;">{r.reg2}</td>
                        <td style="padding:6px 8px;"><span class="badge {st.cls}" style="font-size:8px;">{st.label}</span></td>
                        <td style="padding:6px 8px;"><span class="badge {dtCls}" style="font-size:8px;">{r.data_type}</span></td>
                      </tr>
                    {:else}
                      <tr><td colspan="7" style="padding:24px; text-align:center; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">Tidak ada data sensor</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>

            <!-- Right Table: Production Conveyor Logs -->
            <div class="card" style="padding: 14px; overflow:hidden; background: var(--surface);">
              <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:10px; border-bottom:1px solid var(--border); padding-bottom:6px;">
                <div style="display:flex; align-items:center; gap:6px;">
                  <span class="section-dot" style="background:var(--green); width:6px; height:6px; border-radius:50%; display:inline-block;"></span>
                  <span style="font-family:var(--font-mono); font-size:10px; font-weight:700; text-transform:uppercase;">Production Batch Log</span>
                </div>
              </div>
              <div style="overflow-x:auto; max-height: 62vh;">
                <table style="width:100%; border-collapse:collapse;">
                  <thead>
                    <tr style="border-bottom:1px solid var(--border); position: sticky; top: 0; background: var(--surface); z-index: 10;">
                      {#each ["ID", "TS", "PRODUK", "QTY", "SHIFT", "BATCH", "BEST BEFORE", "FC"] as h}
                        <th style="text-align:left; padding:6px 8px; font-family:var(--font-mono); font-size:8px; font-weight:600; color:var(--text-muted); text-transform:uppercase;">{h}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody>
                    {#each conveyorRecords as c}
                      {@const daysLeft = getDaysRemaining(c.tanggal_best_before)}
                      <tr style="border-bottom:1px solid var(--border); transition:background var(--transition);"
                          onmouseenter={(e: any) => (e.currentTarget.style.background = "var(--surface-2)")}
                          onmouseleave={(e: any) => (e.currentTarget.style.background = "")}>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; color:var(--text-muted);">{c.id_record}</td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; white-space:nowrap;">{c.tanggal_record ? c.tanggal_record.slice(0, 19).replace('T', ' ') : '-'}</td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; font-weight:600; color:var(--accent-2); white-space:nowrap;">
                          {c.kode_produk} <span style="font-size:8px; font-weight:400; color:var(--text-muted);">({productCodeToName[c.kode_produk] || 'Unknown'})</span>
                        </td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:10px;">{c.qty_per_pack}</td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:10px;">Shift {c.shift}</td>
                        <td style="padding:6px 8px;"><span class="badge badge-green" style="font-size:8px;">{c.kode_batch}</span></td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; white-space:nowrap;">
                          {c.tanggal_best_before}
                          {#if daysLeft !== null}
                            {#if daysLeft <= 7}
                              <span class="badge badge-red" style="font-size:7px; margin-left:4px; animation: blink 1s infinite;">{daysLeft}d ⚠️</span>
                            {:else if daysLeft <= 30}
                              <span class="badge badge-yellow" style="font-size:7px; margin-left:4px;">{daysLeft}d ⚠️</span>
                            {:else}
                              <span class="badge badge-green" style="font-size:7px; margin-left:4px; opacity: 0.7;">{daysLeft}d</span>
                            {/if}
                          {/if}
                        </td>
                        <td style="padding:6px 8px; font-family:var(--font-mono); font-size:9px; color:var(--text-muted);">{c.factory}</td>
                      </tr>
                    {:else}
                      <tr><td colspan="8" style="padding:24px; text-align:center; color:var(--text-muted); font-family:var(--font-mono); font-size:10px;">Tidak ada data batch</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        {/if}
      {/if}
    </main>
  </div>
{/if}

{#if isLoading}
  <div class="loading-bar"></div>
{/if}

<style>
  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
  @media (max-width: 900px) {
    .split-tables-wrapper {
      grid-template-columns: 1fr !important;
    }
  }
</style>
