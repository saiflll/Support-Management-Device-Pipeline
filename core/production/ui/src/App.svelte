<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import Analytics from "./Analytics.svelte";
  import Conveyor from "./Conveyor.svelte";
  import { Chart, registerables } from "chart.js";
  import {
    LayoutDashboard, ScanBarcode, BarChart3,
    Sun, Moon, LogOut, RefreshCw, X, Download, FileSpreadsheet,
    AlertTriangle, ShieldCheck, Activity, Boxes,
    Clock, TrendingUp, Wifi, CircleDot, ChevronRight,
    Settings, Gauge, Layers, Filter, CalendarDays,
    ArrowUpDown, ArrowUp, ArrowDown, FileDown
  } from "@lucide/svelte";

  Chart.register(...registerables);

  const FORMING_BASE = "";
  type Module = "mdcw" | "sp" | "analytics" | "conveyor";
  type SortDir = "asc" | "desc";

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

  // Sort state for detail log tables
  let mdcwSortCol = $state("ts");
  let mdcwSortDir = $state<SortDir>("desc");
  let convSortCol = $state("tanggal_record");
  let convSortDir = $state<SortDir>("desc");

  let toasts: { id: number; type: string; msg: string }[] = $state([]);
  let toastId = 0;
  let pollInterval: ReturnType<typeof setInterval>;

  $effect(() => { document.documentElement.setAttribute("data-theme", theme); });

  function toggleTheme() {
    theme = theme === "dark" ? "light" : "dark";
    localStorage.setItem("forming_theme", theme);
  }

  function moduleApi(path: string): string { return `/api/${activeModule}${path}`; }

  async function api(path: string, opts: RequestInit = {}) {
    const res = await fetch(FORMING_BASE + path, {
      ...opts,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: "Bearer " + token } : {}),
        ...(opts.headers || {}),
      },
    });
    if (res.status === 401) { loggedIn = false; return null; }
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  function addToast(msg: string, type: string = "info") {
    const id = ++toastId;
    toasts = [...toasts, { id, type, msg }];
    setTimeout(() => { toasts = toasts.filter(t => t.id !== id); }, 4000);
  }

  onMount(() => {
    const saved = localStorage.getItem("forming_token");
    if (saved) { token = saved; loggedIn = true; fetchData(); }
  });
  onDestroy(() => pollInterval && clearInterval(pollInterval));

  async function handleLogin(e: Event) {
    e.preventDefault();
    loginError = "";
    try {
      const data = await api("/api/login", { method: "POST", body: JSON.stringify({ username, password }) });
      token = data.token;
      localStorage.setItem("forming_token", token);
      loggedIn = true;
      fetchData();
    } catch { loginError = "Username atau password salah."; }
  }

  async function handleLogout() {
    await api("/api/logout", { method: "POST" });
    token = "";
    localStorage.removeItem("forming_token");
    loggedIn = false;
  }

  async function switchModule(mod: Module) {
    activeModule = mod;
    if (mod === "analytics" || mod === "conveyor") { clearInterval(pollInterval); return; }
    activeFilter = "all"; statusFilter = "all"; sortBy = "newest"; startDate = ""; endDate = "";
    records = []; summaries = []; filters = []; conveyorRecords = [];
    detailConveyorSearch = ""; detailBatchSearch = ""; detailShiftFilter = "all";
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
    await Promise.all([fetchRecords(), fetchSummary(), fetchFilters()]);
    if (activeModule === "sp") {
      await Promise.all([fetchSpDailyStats(), fetchSpComparison()]);
    }
    clearInterval(pollInterval);
    pollInterval = setInterval(async () => {
      if (activeModule === "sp") {
        await Promise.all([fetchRecords(), fetchSpComparison()]);
      } else {
        await fetchRecords();
      }
    }, 15000);
  }

  async function fetchRecords() {
    isLoading = true;
    try {
      const params = new URLSearchParams();
      const isMdcw = activeModule === "mdcw";
      if (activeFilter !== "all") params.set(isMdcw ? "prefix" : "session_id", activeFilter);
      if (isMdcw) { if (statusFilter !== "all") params.set("status", statusFilter); params.set("sort", sortBy); }
      if (startDate) params.set("start_date", startDate);
      if (endDate) params.set("end_date", endDate);
      const data = await api(moduleApi("/data?") + params);
      records = data || [];
    } finally { isLoading = false; }
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
    } catch (e) { console.error("conveyor error", e); }
    finally { isLoading = false; }
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
    try { mdcwDailyStats = (await api("/api/mdcw/daily-stats")) || []; } catch {}
  }

  async function fetchSpDailyStats() {
    try {
      const data = await api("/api/sp/daily-stats?days=7");
      spDailyStats = data || [];
    } catch {}
  }

  async function fetchSpComparison() {
    try {
      spComparisons = (await api("/api/sp/comparison")) || [];
    } catch {}
  }

  // ── EXPORT CSV (backend) ──────────────────────────────────
  function exportCSV() {
    const params = new URLSearchParams();
    if (activeFilter !== "all") params.set(activeModule === "mdcw" ? "prefix" : "session_id", activeFilter);
    if (activeModule === "mdcw" && statusFilter !== "all") params.set("status", statusFilter);
    if (startDate) params.set("start_date", startDate);
    if (endDate) params.set("end_date", endDate);
    const a = document.createElement("a");
    a.href = moduleApi("/export-csv?") + params;
    a.click();
    addToast("Export CSV dimulai…", "success");
  }

  // ── EXPORT EXCEL (client-side dari data yang sudah di-fetch) ──
  function exportExcelFromRecords() {
    if (!records.length) { addToast("Tidak ada data untuk diekspor.", "error"); return; }
    const rows: string[][] = [];
    if (activeModule === "mdcw") {
      rows.push(["ID","Timestamp","Prefix","Berat","Pack Count","Status","Data Type","Confidence"]);
      for (const r of records) {
        const st = statusLabel(r.reg5);
        rows.push([r.id, r.ts, r.prefix, r.weight_formatted, r.reg2, st.label, r.data_type, (r.confidence||0).toFixed(2)]);
      }
    } else {
      rows.push(["ID","Session ID","Data","Timestamp"]);
      for (const r of records) rows.push([r.id, r.session_id, r.data, r.ts || r.created_at]);
    }
    downloadAsCSVWithTabs(rows, `${activeModule}_export_${new Date().toISOString().slice(0,10)}.xls`);
    addToast("Export Excel selesai.", "success");
  }

  function exportConveyorExcel() {
    if (!conveyorRecords.length) { addToast("Tidak ada data batch untuk diekspor.", "error"); return; }
    const rows: string[][] = [["ID","Timestamp","Kode Produk","Nama Produk","Qty/Pack","Shift","Kode Batch","Best Before","Factory"]];
    for (const c of conveyorRecords) {
      rows.push([
        c.id_record,
        c.tanggal_record ? c.tanggal_record.slice(0,19).replace('T',' ') : '-',
        c.kode_produk,
        productCodeToName[c.kode_produk] || '?',
        c.qty_per_pack,
        `Shift ${c.shift}`,
        c.kode_batch,
        c.tanggal_best_before,
        c.factory
      ]);
    }
    downloadAsCSVWithTabs(rows, `batch_log_${new Date().toISOString().slice(0,10)}.xls`);
    addToast("Export Excel batch selesai.", "success");
  }

  function downloadAsCSVWithTabs(rows: any[][], filename: string) {
    // Tab-separated = opens cleanly in Excel
    const content = rows.map(r => r.map(c => `"${String(c??'').replace(/"/g,'""')}"`).join("\t")).join("\r\n");
    const blob = new Blob(["\uFEFF" + content], { type: "text/tab-separated-values;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = filename; a.click();
    URL.revokeObjectURL(url);
  }

  function getDaysRemaining(tglStr: string): number | null {
    if (!tglStr || tglStr.length !== 6) return null;
    const dd = parseInt(tglStr.slice(0, 2));
    const mm = parseInt(tglStr.slice(2, 4)) - 1;
    const yy = 2000 + parseInt(tglStr.slice(4, 6));
    const expiry = new Date(yy, mm, dd);
    const today = new Date(); today.setHours(0, 0, 0, 0);
    return Math.ceil((expiry.getTime() - today.getTime()) / 86400000);
  }

  function clearFilter() {
    activeFilter = "all"; statusFilter = "all"; sortBy = "newest"; startDate = ""; endDate = "";
    detailConveyorSearch = ""; detailBatchSearch = ""; detailShiftFilter = "all";
    if (activeModule === "mdcw") { fetchRecords(); fetchConveyorRecords(); } else fetchRecords();
  }

  function statusLabel(reg5: number): { label: string; cls: string } {
    const map: Record<number, [string, string]> = {
      41:["OK","badge-green"],521:["OK","badge-green"],553:["OK","badge-green"],
      8:["MATI","badge-red"],9:["IDLE","badge-yellow"],90:["IDLE","badge-yellow"],
      8201:["METAL","badge-purple"],25:["UNDER","badge-blue"],73:["OVER","badge-yellow"],
    };
    const e = map[reg5];
    return e ? { label: e[0], cls: e[1] } : { label: `?(${reg5})`, cls: "badge-red" };
  }

  // Sort helpers
  function toggleMdcwSort(col: string) {
    if (mdcwSortCol === col) mdcwSortDir = mdcwSortDir === "desc" ? "asc" : "desc";
    else { mdcwSortCol = col; mdcwSortDir = "desc"; }
  }
  function toggleConvSort(col: string) {
    if (convSortCol === col) convSortDir = convSortDir === "desc" ? "asc" : "desc";
    else { convSortCol = col; convSortDir = "desc"; }
  }

  function sortedRecords() {
    return [...records].sort((a, b) => {
      const av = a[mdcwSortCol] ?? "";
      const bv = b[mdcwSortCol] ?? "";
      const cmp = String(av).localeCompare(String(bv), undefined, { numeric: true });
      return mdcwSortDir === "desc" ? -cmp : cmp;
    });
  }

  function sortedConveyorRecords() {
    return [...conveyorRecords].sort((a, b) => {
      const av = a[convSortCol] ?? "";
      const bv = b[convSortCol] ?? "";
      const cmp = String(av).localeCompare(String(bv), undefined, { numeric: true });
      return convSortDir === "desc" ? -cmp : cmp;
    });
  }

  // ── MDCW STATE ────────────────────────────────────────────
  let activeMdcwSubTab = $state("ringkasan");
  let activeSpSubTab = $state("scans");
  let mdcwDailyStats = $state<any[]>([]);
  let spDailyStats = $state<any[]>([]);
  let spComparisons = $state<any[]>([]);
  let selectedShift = $state("all");

  const prefixToProductCode: Record<string, string> = {
    "MDCW1 (UK)":"100294","MDCW2 (Siomay)":"100256","MDCW3 (Pentol)":"100286",
    "MDCW4 (AP)":"100209","MDCW5 (ACIN)":"100211","MDCW6 (Lumpia)":"100244",
    "MDCW7 (Kulit/Kerupuk)":"100239",
    "MDCW8 (Mie)":"100245",
    "MDCW9 (Mie)":"100245",
  };
  const productCodeToName: Record<string, string> = {
    "100294":"UDANG KEJU","100256":"SIOMAY DIMSUM","100286":"UDANG RAMBUTAN (PEN)",
    "100209":"ADONAN PANGSIT","100211":"AYAM CINCANG","100244":"LUMPIA UDANG",
    "100238":"KERUPUK MIE","100239":"KULIT PANGSIT","100245":"MIE","100339":"CABAI FROZEN",
  };

  let chartTrend: Chart | null = null;
  let chartReject: Chart | null = null;
  let chartSpDaily: Chart | null = null;
  let canvasTrend = $state<HTMLCanvasElement | null>(null);
  let canvasReject = $state<HTMLCanvasElement | null>(null);
  let canvasSpDaily = $state<HTMLCanvasElement | null>(null);

  const getChartColors = (th: string) => ({
    text: th === "dark" ? "#a1a1aa" : "#4b5563",
    grid: th === "dark" ? "rgba(255,255,255,0.04)" : "rgba(0,0,0,0.04)",
    tick: th === "dark" ? "#71717a" : "#9ca3af",
  });

  $effect(() => {
    const _t = theme;
    if (activeModule === "mdcw" && activeMdcwSubTab === "ringkasan" && canvasTrend && mdcwDailyStats.length > 0)
      setTimeout(initTrendChart, 50);
    if (activeModule === "mdcw" && activeMdcwSubTab === "qc_mesin" && canvasReject && summaries.length > 0)
      setTimeout(initRejectChart, 50);
    if (activeModule === "sp" && canvasSpDaily && spDailyStats.length > 0)
      setTimeout(initSpDailyChart, 50);
  });

  onDestroy(() => { chartTrend?.destroy(); chartReject?.destroy(); chartSpDaily?.destroy(); });

  function initTrendChart() {
    chartTrend?.destroy();
    if (!canvasTrend || !mdcwDailyStats.length) return;
    const c = getChartColors(theme);
    chartTrend = new Chart(canvasTrend, {
      type: "line",
      data: {
        labels: mdcwDailyStats.map(d => d.date?.slice(5) || ""),
        datasets: [
          { label:"OK",    data:mdcwDailyStats.map(d=>d.ok_count||0),    borderColor:"#10b981",backgroundColor:"rgba(16,185,129,0.05)",fill:true,tension:0.3,pointRadius:2,borderWidth:1.5 },
          { label:"UNDER", data:mdcwDailyStats.map(d=>d.under_count||0), borderColor:"#3b82f6",backgroundColor:"rgba(59,130,246,0.05)",fill:true,tension:0.3,pointRadius:2,borderWidth:1.5 },
          { label:"OVER",  data:mdcwDailyStats.map(d=>d.over_count||0),  borderColor:"#f59e0b",backgroundColor:"rgba(245,158,11,0.05)",fill:true,tension:0.3,pointRadius:2,borderWidth:1.5 },
          { label:"METAL", data:mdcwDailyStats.map(d=>d.metal_count||0), borderColor:"#8b5cf6",backgroundColor:"rgba(139,92,246,0.05)",fill:true,tension:0.3,pointRadius:2,borderWidth:1.5 },
        ]
      },
      options: {
        responsive:true, maintainAspectRatio:false,
        plugins:{legend:{position:"top",labels:{color:c.text,boxWidth:8,padding:8,font:{size:8,family:"JetBrains Mono"}}}},
        scales:{x:{grid:{display:false},ticks:{color:c.tick,font:{size:8}}},y:{beginAtZero:true,grid:{color:c.grid},ticks:{color:c.tick,font:{size:8}}}}
      }
    });
  }

  function initRejectChart() {
    chartReject?.destroy();
    if (!canvasReject || !summaries.length) return;
    const c = getChartColors(theme);
    chartReject = new Chart(canvasReject, {
      type: "bar",
      data: {
        labels: summaries.map(s=>s.prefix?.replace("MDCW","") || ""),
        datasets: [
          { label:"UNDER",data:summaries.map(s=>s.under_count||0),backgroundColor:"#3b82f6",borderRadius:2 },
          { label:"OVER", data:summaries.map(s=>s.over_count||0), backgroundColor:"#f59e0b",borderRadius:2 },
          { label:"METAL",data:summaries.map(s=>s.metal_count||0),backgroundColor:"#8b5cf6",borderRadius:2 },
        ]
      },
      options: {
        responsive:true, maintainAspectRatio:false,
        plugins:{legend:{position:"top",labels:{color:c.text,boxWidth:8,padding:8,font:{size:8,family:"JetBrains Mono"}}}},
        scales:{x:{grid:{display:false},ticks:{color:c.tick,font:{size:8}}},y:{beginAtZero:true,grid:{color:c.grid},ticks:{color:c.tick,font:{size:8}}}}
      }
    });
  }

  function initSpDailyChart() {
    chartSpDaily?.destroy();
    if (!canvasSpDaily || !spDailyStats.length) return;
    const c = getChartColors(theme);
    chartSpDaily = new Chart(canvasSpDaily, {
      type: "bar",
      data: {
        labels: spDailyStats.map((d: any) => d.date?.slice(5) || ""),
        datasets: [{
          label: "Scans",
          data: spDailyStats.map((d: any) => d.total_count || 0),
          backgroundColor: "rgba(99,102,241,0.5)",
          borderColor: "#6366f1",
          borderWidth: 1,
          borderRadius: 2,
        }]
      },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { legend: { display: false } },
        scales: {
          x: { grid: { display: false }, ticks: { color: c.tick, font: { size: 8 } } },
          y: { beginAtZero: true, grid: { color: c.grid }, ticks: { color: c.tick, font: { size: 8 } } }
        }
      }
    });
  }

  function getOeeForPrefix(pfx: string, shiftFilter: string = "all") {
    const pfxRecs = records.filter(r => {
      if (r.prefix !== pfx) return false;
      if (shiftFilter === "all") return true;
      const hour = parseInt(r.ts?.slice(11,13) || "0");
      const shift = hour>=7&&hour<15?1:hour>=15&&hour<23?2:3;
      return shift.toString() === shiftFilter;
    });
    if (!pfxRecs.length) return { availability:0,performance:0,quality:0,oee:0,uptime:0,idle:0,downtime:0 };

    const sorted = [...pfxRecs].sort((a,b)=>new Date(a.ts).getTime()-new Date(b.ts).getTime());
    let uptime=0, idle=0, downtime=0;
    for (let i=0;i<sorted.length-1;i++) {
      const diff = Math.min((new Date(sorted[i+1].ts).getTime()-new Date(sorted[i].ts).getTime())/1000, 3600);
      if (sorted[i].reg5===8) downtime+=diff;
      else if (sorted[i].reg5===9||sorted[i].reg5===90) idle+=diff;
      else uptime+=diff;
    }
    const total = uptime+downtime+idle || 1;
    const availability = ((uptime+idle)/total)*100;
    const ok = pfxRecs.filter(r=>[41,521,553].includes(r.reg5)).length;
    const totalQC = pfxRecs.filter(r=>[41,521,553,25,73,8201].includes(r.reg5)).length;
    const quality = totalQC>0?(ok/totalQC)*100:100;
    const spanH = sorted.length>1?(new Date(sorted[sorted.length-1].ts).getTime()-new Date(sorted[0].ts).getTime())/3600000:0.05;
    const performance = Math.min(100,(pfxRecs.length/Math.max(1,Math.round(spanH*500)))*100);
    return { availability, performance, quality, oee:(availability*performance*quality)/10000, uptime, idle, downtime };
  }

  const overallStats = $derived.by(()=>{
    let output=0,ok=0,reject=0,metal=0;
    for (const s of summaries) {
      output+=s.total_count||0; ok+=s.ok_count||0;
      reject+=(s.under_count||0)+(s.over_count||0)+(s.metal_count||0);
      metal+=s.metal_count||0;
    }
    const passRate = output>0?(ok/output)*100:0;
    const bbKritis = conveyorRecords.filter(c=>{const d=getDaysRemaining(c.tanggal_best_before);return d!==null&&d<=30;}).length;
    const PREFIXES = ["MDCW1 (UK)","MDCW2 (Siomay)","MDCW3 (Pentol)","MDCW4 (AP)","MDCW5 (ACIN)","MDCW6 (Lumpia)","MDCW7 (Kulit/Kerupuk)","MDCW8 (Mie)","MDCW9 (Mie)"];
    let oeeSum=0,activeCount=0;
    for (const p of PREFIXES) {
      if (records.some(r=>r.prefix===p)) { oeeSum+=getOeeForPrefix(p).oee; activeCount++; }
    }
    return { output, ok, passRate, reject, metal, bbKritis, avgOee:activeCount>0?oeeSum/activeCount:0 };
  });

  const priorityAlerts = $derived.by(()=>{
    const alerts: { type:"critical"|"warning"; msg:string }[] = [];
    if (overallStats.metal>0)
      alerts.push({ type:"critical", msg:`KONTAMINASI LOGAM — Sensor mendeteksi ${overallStats.metal} reject logam hari ini` });
    let minDays:number|null=null, minBatch="";
    for (const c of conveyorRecords) {
      const d=getDaysRemaining(c.tanggal_best_before);
      if (d!==null&&(minDays===null||d<minDays)){minDays=d;minBatch=c.kode_batch;}
    }
    if (minDays!==null&&minDays<=30)
      alerts.push({ type:minDays<=7?"critical":"warning", msg:`BB HAMPIR KADALUWARSA — Batch ${minBatch} sisa ${minDays} hari` });
    for (const s of summaries) {
      const tot=s.total_count||0;
      const rej=(s.under_count||0)+(s.over_count||0)+(s.metal_count||0);
      if (tot>15&&(rej/tot)*100>15)
        alerts.push({ type:"warning", msg:`REJECT TINGGI — ${s.prefix}: ${((rej/tot)*100).toFixed(1)}% reject rate` });
    }
    if (!alerts.length)
      alerts.push({ type:"warning", msg:"KONDISI NORMAL — Seluruh checkweigher & conveyor berjalan dalam rentang aman" });
    return alerts.slice(0,3);
  });

  const gramasiDriftStats = $derived.by(()=>{
    const list: { kode:string;nama:string;std:number;avg:number;drift:number;status:string;statusCls:string }[] = [];
    for (const code of [...new Set(conveyorRecords.map(c=>c.kode_produk))]) {
      const cRec = conveyorRecords.find(c=>c.kode_produk===code);
      if (!cRec) continue;
      const std = cRec.gramasi_pack||0;
      const name = productCodeToName[code]||"Produk "+code;
      let prefix="";
      for (const [pfx,cde] of Object.entries(prefixToProductCode)){if(cde===code){prefix=pfx;break;}}
      if (!prefix) continue;
      const pfxRecs = records.filter(r=>{
        if (r.prefix!==prefix) return false;
        if (selectedShift==="all") return true;
        const hour=parseInt(r.ts?.slice(11,13)||"0");
        const shift=hour>=7&&hour<15?1:hour>=15&&hour<23?2:3;
        return shift.toString()===selectedShift;
      });
      const validRecs = pfxRecs.filter(r=>r.reg114>0);
      if (!validRecs.length) continue;
      const avg = validRecs.reduce((a,r)=>a+r.reg114,0)/(validRecs.length*10);
      const drift = avg-std;
      const limit = std*0.02;
      const status    = Math.abs(drift)>limit?"Drift (Kalibrasi!)":Math.abs(drift)>limit*0.5?"Warning":"Stabil";
      const statusCls = Math.abs(drift)>limit?"badge-red":Math.abs(drift)>limit*0.5?"badge-yellow":"badge-green";
      list.push({kode:code,nama:name,std,avg,drift,status,statusCls});
    }
    return list;
  });

  let shiftRecs = $derived.by(()=>records.filter(r=>{
    if (selectedShift==="all") return true;
    const hour=parseInt(r.ts?.slice(11,13)||"0");
    const shift=hour>=7&&hour<15?1:hour>=15&&hour<23?2:3;
    return shift.toString()===selectedShift;
  }));
  let okShift       = $derived(shiftRecs.filter(r=>[41,521,553].includes(r.reg5)).length);
  let rejectShift   = $derived(shiftRecs.length-okShift);
  let passRateShift = $derived(shiftRecs.length>0?(okShift/shiftRecs.length)*100:100);

  // Sparkline — NO DUMMY. Hanya tampil kalau ada data nyata.
  function getSparklinePath(points: number[]): string {
    if (!points || points.length < 2) return "";
    const max=Math.max(...points)||1, min=Math.min(...points);
    const range=max-min||1, W=100, H=28;
    const step=W/(points.length-1);
    return points.map((p,i)=>{
      const x=i*step, y=H-((p-min)/range)*H+1;
      return `${i===0?"M":"L"} ${x.toFixed(1)} ${y.toFixed(1)}`;
    }).join(" ");
  }

  // Only real data, no fallback arrays
  const sparklineOutputPoints = $derived(mdcwDailyStats.map(d=>d.ok_count||0));
  const sparklineMetalPoints  = $derived(mdcwDailyStats.map(d=>d.metal_count||0));
  const sparklineOeePoints    = $derived(mdcwDailyStats.map(d=>d.ok_count?Math.round((d.ok_count/(d.total_count||1))*100):0));

  // ── SP DERIVED KPIs ───────────────────────────────────
  const spTotalScans = $derived(summaries.reduce((a: number, s: any) => a + (s.total_count || 0), 0));
  const spActiveSessions = $derived(summaries.length);
  const spLatestScan = $derived.by(() => {
    if (!records.length) return "-";
    // records typically sorted newest-first by API
    for (const r of records) {
      if (r.ts || r.created_at) return r.ts || r.created_at;
    }
    return "-";
  });
  const spSparklinePoints = $derived(spDailyStats.map((d: any) => d.total_count || 0));

  const MDCW_PREFIXES = ["MDCW1 (UK)","MDCW2 (Siomay)","MDCW3 (Pentol)","MDCW4 (AP)","MDCW5 (ACIN)","MDCW6 (Lumpia)","MDCW7 (Kulit/Kerupuk)","MDCW8 (Mie)","MDCW9 (Mie)"];
</script>

<svelte:head><title>Production Monitor</title></svelte:head>

<!-- Toasts -->
<div class="toast-container">
  {#each toasts as t (t.id)}<div class="toast {t.type}">{t.msg}</div>{/each}
</div>

{#if !loggedIn}
<!-- ── LOGIN ──────────────────────────────────────── -->
<div class="login-screen">
  <div class="login-box">
    <div class="login-brand">
      <div class="login-brand-name">PRODUCTION_MONITOR</div>
      <div class="login-brand-sub">Line Intelligence System · v2</div>
    </div>
    <div class="card">
      <form onsubmit={handleLogin}>
        <div class="form-group">
          <label class="form-label" for="username_input">OPERATOR ID</label>
          <input id="username_input" type="text" bind:value={username} required autocomplete="username" class="form-input" placeholder="username" />
        </div>
        <div class="form-group">
          <label class="form-label" for="password_input">ACCESS CODE</label>
          <input id="password_input" type="password" bind:value={password} required autocomplete="current-password" class="form-input" placeholder="••••••••" />
        </div>
        {#if loginError}
          <div class="mono text-xs mb-2" style="color:var(--red)">{loginError}</div>
        {/if}
        <button type="submit" class="btn btn-primary" style="width:100%;justify-content:center;gap:6px">
          <ChevronRight size={12} /> MASUK
        </button>
      </form>
    </div>
  </div>
</div>

{:else}
{#if isLoading}<div class="loading-bar"></div>{/if}
<!-- ── APP SHELL ─────────────────────────────────── -->
<div class="app-shell">

  <!-- SIDEBAR -->
  <aside class="sidebar">
    <div class="sidebar-top">
      <div class="sidebar-logo"><Activity size={13} /> PRODUKSI</div>
      <nav class="sidebar-menu">
        <button class="sidebar-btn" class:active={activeModule==="mdcw"} onclick={()=>switchModule("mdcw")}>
          <span class="btn-icon"><Gauge size={13}/></span> MDCW
        </button>
        <button class="sidebar-btn" class:active={activeModule==="sp"} onclick={()=>switchModule("sp")}>
          <span class="btn-icon"><ScanBarcode size={13}/></span> SP
        </button>
        <button class="sidebar-btn" class:active={activeModule==="analytics"} onclick={()=>switchModule("analytics")}>
          <span class="btn-icon"><BarChart3 size={13}/></span> ANALYTICS
        </button>
        <button class="sidebar-btn" class:active={activeModule==="conveyor"} onclick={()=>switchModule("conveyor")}>
          <span class="btn-icon"><Layers size={13}/></span> CONVEYOR
        </button>
      </nav>
    </div>
    <div class="sidebar-bottom">
      <div class="live-dot">LIVE · AUTO 15s</div>
      <div class="row-flex gap-2 justify-between">
        <button class="icon-btn" onclick={toggleTheme} title="Toggle theme">
          {#if theme === "dark"}<Sun size={14}/>{:else}<Moon size={14}/>{/if}
        </button>
        <button class="icon-btn" onclick={handleLogout} title="Logout"><LogOut size={14}/></button>
      </div>
    </div>
  </aside>

  <!-- MAIN -->
  <main class="main-content">

    {#if activeModule === "analytics"}
      <div class="module-wrapper"><Analytics {api} {token} {theme} /></div>

    {:else if activeModule === "conveyor"}
      <div class="module-wrapper"><Conveyor {api} {token} {theme} /></div>

    {:else if activeModule === "sp"}
    <!-- ── SP MODULE ──────────────────────────────── -->
      <div class="data-info">
        <span><Wifi size={10}/> <strong>SOURCE</strong> BARCODE SCANNERS → MQTT <code>emqx:1883</code> → <code>SP_data</code> → POSTGRES</span>
        <span><Clock size={10}/> Real-time events</span>
      </div>

      <!-- Sub-tab Nav SP -->
      <div class="card subtab-nav">
        <div class="subtab-group">
          <button class="subtab-btn" class:active={activeSpSubTab==="scans"} onclick={()=>activeSpSubTab="scans"}>
            <ScanBarcode size={11}/> LOGS SCAN
          </button>
          <button class="subtab-btn" class:active={activeSpSubTab==="discrepancy"} onclick={()=>activeSpSubTab="discrepancy"}>
            <AlertTriangle size={11}/> DISCREPANCY (MDCW vs SP)
          </button>
        </div>
        <div class="row-flex gap-2">
          {#if activeSpSubTab==="scans" && records.length}<span class="subtab-tag">{records.length} records loaded</span>{/if}
          {#if activeSpSubTab==="discrepancy" && spComparisons.length}<span class="subtab-tag">{spComparisons.length} lines loaded</span>{/if}
          <span class="subtab-tag"><Settings size={9}/> SP ENGINE</span>
        </div>
      </div>

      <!-- SP KPI Cards -->
      <div class="kpi-grid" style="margin-bottom:0">
        <div class="kpi-card" style="border-top:2px solid var(--accent-2)">
          <div class="kpi-card-header"><span class="kpi-title">Total Scans</span><ScanBarcode size={12} style="color:var(--text-dim)"/></div>
          <div class="kpi-value">{summaries.length ? spTotalScans : '—'}</div>
          <span class="kpi-subtext">scan hari ini</span>
          {#if spSparklinePoints.length >= 2}
            <div class="sparkline-container">
              <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                <defs><linearGradient id="sp-scan-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="var(--accent-2)" stop-opacity="0.15"/><stop offset="100%" stop-color="var(--accent-2)" stop-opacity="0"/>
                </linearGradient></defs>
                <path d={getSparklinePath(spSparklinePoints)} fill="none" stroke="var(--accent-2)" stroke-width="1.2"/>
                <path d="{getSparklinePath(spSparklinePoints)} L 100 30 L 0 30 Z" fill="url(#sp-scan-grad)"/>
              </svg>
            </div>
          {/if}
        </div>
        <div class="kpi-card" style="border-top:2px solid var(--green)">
          <div class="kpi-card-header"><span class="kpi-title">Active Sessions</span><BarChart3 size={12} style="color:var(--text-dim)"/></div>
          <div class="kpi-value" style="color:var(--green)">{summaries.length ? spActiveSessions : '—'}</div>
          <span class="kpi-subtext">sesi aktif hari ini</span>
        </div>
        <div class="kpi-card" style="border-top:2px solid var(--purple)">
          <div class="kpi-card-header"><span class="kpi-title">Latest Scan</span><Clock size={12} style="color:var(--text-dim)"/></div>
          <div class="kpi-value" style="font-size:13px;line-height:1.3;color:var(--purple)">{records.length ? spLatestScan : '—'}</div>
          <span class="kpi-subtext">waktu scan terakhir</span>
        </div>
      </div>

      <!-- SP Daily Trend Chart -->
      <div class="trend-card" style="margin-bottom:0">
        <div class="card-label row-flex gap-1"><TrendingUp size={10}/> Scan Volume — 7 Hari Terakhir</div>
        {#if spDailyStats.length}
          <div class="trend-canvas-wrap" style="min-height:120px"><canvas bind:this={canvasSpDaily}></canvas></div>
        {:else}
          <div class="chart-empty" style="padding:24px"><BarChart3 size={24}/><span>Belum ada data tren</span></div>
        {/if}
      </div>

      {#if activeSpSubTab === "scans"}
      <div class="sp-grid">
        <!-- Filters -->
        <div class="sp-sidebar">
          <div class="card">
            <div class="card-label row-flex gap-1 mb-2"><Filter size={10}/> Session</div>
            <div class="session-btn-group">
              <button class="btn" class:btn-primary={activeFilter==="all"} class:btn-ghost={activeFilter!=="all"}
                onclick={()=>{activeFilter="all";fetchRecords();}}>ALL</button>
              {#each filters as f}
                <button class="btn" class:btn-primary={activeFilter===f} class:btn-ghost={activeFilter!==f}
                  onclick={()=>{activeFilter=f;fetchRecords();}}>{f}</button>
              {/each}
            </div>
          </div>

          <div class="card">
            <div class="card-label row-flex gap-1 mb-2"><CalendarDays size={10}/> Rentang Tanggal</div>
            <div class="col-flex" style="gap:6px">
              <input type="date" bind:value={startDate} onchange={fetchRecords} />
              <span class="muted text-xs" style="text-align:center">s/d</span>
              <input type="date" bind:value={endDate} onchange={fetchRecords} />
            </div>
          </div>

          <!-- Export actions -->
          <div class="card">
            <div class="card-label row-flex gap-1 mb-2"><FileDown size={10}/> Export</div>
            <div class="col-flex" style="gap:5px">
              <button class="btn btn-ghost" onclick={exportCSV} style="justify-content:center;gap:5px">
                <Download size={10}/> CSV (server)
              </button>
              <button class="btn btn-ghost" onclick={exportExcelFromRecords} style="justify-content:center;gap:5px">
                <FileSpreadsheet size={10}/> Excel (client)
              </button>
            </div>
          </div>

          <div class="card">
            <div class="row-flex gap-2">
              <button class="btn btn-ghost"  onclick={fetchRecords} style="flex:1;justify-content:center" title="Refresh"><RefreshCw size={10}/></button>
              <button class="btn btn-danger" onclick={clearFilter} style="flex:1;justify-content:center" title="Reset filter"><X size={10}/> Reset</button>
            </div>
          </div>
        </div>

        <!-- Table -->
        <div class="card" style="overflow:hidden">
          <div class="table-scroll">
            <table class="data-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>SESSION</th>
                  <th>DATA</th>
                  <th>TIMESTAMP</th>
                </tr>
              </thead>
              <tbody>
                {#each records as r}
                  <tr>
                    <td class="td-mono td-muted">{r.id}</td>
                    <td><span class="badge badge-blue">{r.session_id}</span></td>
                    <td class="td-mono td-accent td-lg">{r.data}</td>
                    <td class="td-mono td-nowrap td-muted">{r.ts||r.created_at}</td>
                  </tr>
                {/each}
                {#if !isLoading && records.length === 0}
                  <tr><td colspan="4" class="empty-cell">
                    <span class="empty-icon"><ScanBarcode size={20}/></span>
                    <span>Tidak ada data scan</span>
                    {#if !startDate && !endDate}<span class="empty-hint">Pilih rentang tanggal atau tunggu data masuk</span>{/if}
                  </td></tr>
                {/if}
              </tbody>
            </table>
          </div>
          {#if records.length}
            <div class="table-footer">
              <span>{records.length} records</span>
              <div class="row-flex gap-2">
                <button class="btn btn-ghost" onclick={exportCSV} style="gap:4px"><Download size={9}/> CSV</button>
                <button class="btn btn-ghost" onclick={exportExcelFromRecords} style="gap:4px"><FileSpreadsheet size={9}/> Excel</button>
              </div>
            </div>
          {/if}
        </div>

        <!-- Session Summaries -->
        <div class="summary-grid">
          {#each summaries as s}
            <div class="summary-card">
              <div class="summary-card-id">{s.session_id}</div>
              <div class="summary-card-count">{s.total_count||0}<span class="muted text-xs" style="font-weight:400;margin-left:2px">scans</span></div>
              <div class="summary-card-meta">
                <span>From: {s.first_scan?s.first_scan.slice(0,10):"-"}</span>
                <span>To: {s.last_scan?s.last_scan.slice(0,10):"-"}</span>
              </div>
            </div>
          {/each}
          {#if !isLoading && summaries.length === 0}
            <div class="card" style="grid-column:1/-1">
              <div class="empty-cell">
                <span class="empty-icon"><BarChart3 size={20}/></span>
                <span>Belum ada data hari ini</span>
              </div>
            </div>
          {/if}
        </div>
      </div>
      {/if}

      {#if activeSpSubTab === "discrepancy"}
        <div class="card" style="overflow:hidden">
          <div class="card-label mb-2">Perbandingan Output MDCW (Checkweigher) vs SP (Secondary Packing)</div>
          <div class="table-scroll">
            <table class="data-table">
              <thead>
                <tr>
                  <th>TANGGAL</th>
                  <th>LINE</th>
                  <th>PRODUK</th>
                  <th>MDCW (PACK OK)</th>
                  <th>SP (CARTONS)</th>
                  <th>SP (EST. PACKS)</th>
                  <th>QTY/CARTON</th>
                  <th>DISCREPANCY (PACKS)</th>
                  <th>STATUS</th>
                </tr>
              </thead>
              <tbody>
                {#each spComparisons as c}
                  {@const discVal = c.discrepancy}
                  {@const discPct = c.discrepancy_pct}
                  {@const statusText = Math.abs(discVal) === 0 ? "SEIMBANG" : Math.abs(discVal) <= c.qty_pack ? "ATTENTION" : "DISCREPANCY"}
                  {@const statusCls = Math.abs(discVal) === 0 ? "badge-green" : Math.abs(discVal) <= c.qty_pack ? "badge-yellow" : "badge-red"}
                  <tr>
                    <td class="td-mono">{c.date}</td>
                    <td><span class="badge badge-blue">{c.line}</span></td>
                    <td>
                      <div class="col-flex">
                        <span class="fw-700">{c.product_name || '?'}</span>
                        <span class="td-muted text-xs">{c.product_code}</span>
                      </div>
                    </td>
                    <td class="td-mono td-lg fw-700">{c.mdcw_packs}</td>
                    <td class="td-mono td-lg">{c.sp_cartons}</td>
                    <td class="td-mono td-lg td-purple fw-700">{c.sp_packs}</td>
                    <td class="td-mono">{c.qty_pack}</td>
                    <td class="td-mono td-lg fw-700" style="color:{discVal > 0 ? 'var(--red)' : discVal < 0 ? 'var(--yellow)' : 'var(--green)'}">
                      {discVal > 0 ? '+' : ''}{discVal} ({discPct.toFixed(1)}%)
                    </td>
                    <td><span class="badge {statusCls}">{statusText}</span></td>
                  </tr>
                {:else}
                  <tr><td colspan="9" class="empty-cell">
                    <span class="empty-icon"><AlertTriangle size={20}/></span>
                    <span>Tidak ada data perbandingan</span>
                  </td></tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}

    {:else if activeModule === "mdcw"}
    <!-- ── MDCW MODULE ─────────────────────────────── -->

      <!-- Global Filter Bar -->
      <div class="filter-row card">
        <span class="filter-label"><Filter size={10}/> Filter</span>
        <div class="sep"></div>

        <select class="filter-pill" bind:value={detailShiftFilter} onchange={fetchConveyorRecords}>
          <option value="all">Shift: Semua</option>
          <option value="1">Shift 1 — Pagi</option>
          <option value="2">Shift 2 — Siang</option>
          <option value="3">Shift 3 — Malam</option>
        </select>

        <select class="filter-pill" bind:value={activeFilter} onchange={fetchRecords}>
          <option value="all">Mesin: Semua</option>
          {#each filters as f}<option value={f}>{f}</option>{/each}
        </select>

        <select class="filter-pill" bind:value={statusFilter} onchange={fetchRecords}>
          <option value="all">Status: Semua</option>
          <option value="ok">OK</option>
          <option value="metal">Metal</option>
          <option value="under">Under</option>
          <option value="over">Over</option>
          <option value="mati">Mati</option>
          <option value="idle">Idle</option>
        </select>

        <select class="filter-pill" bind:value={sortBy} onchange={fetchRecords}>
          <option value="newest">Sort: Terbaru</option>
          <option value="oldest">Sort: Terlama</option>
        </select>

        <div class="sep"></div>

        <div class="date-range-group">
          <CalendarDays size={10} style="color:var(--text-muted);flex-shrink:0" />
          <input type="date" class="filter-pill" bind:value={startDate} onchange={fetchData} title="Dari tanggal" />
          <span class="muted" style="font-size:9px;padding:0 2px">–</span>
          <input type="date" class="filter-pill" bind:value={endDate} onchange={fetchData} title="Sampai tanggal" />
        </div>

        <div class="sep"></div>

        <input type="text" placeholder="Kode Produk" class="filter-pill filter-input-sm" bind:value={detailConveyorSearch} oninput={fetchConveyorRecords} />
        <input type="text" placeholder="Kode Batch"  class="filter-pill filter-input-sm" bind:value={detailBatchSearch} oninput={fetchConveyorRecords} />

        <div class="sep"></div>

        <!-- Export buttons -->
        <button class="btn btn-ghost" onclick={exportCSV} title="Export CSV (server)" style="gap:4px"><Download size={10}/> CSV</button>
        <button class="btn btn-ghost" onclick={exportExcelFromRecords} title="Export Excel sensor log" style="gap:4px"><FileSpreadsheet size={10}/> Excel</button>

        <button class="icon-btn" onclick={fetchData}   title="Refresh"><RefreshCw size={12}/></button>
        <button class="icon-btn" onclick={clearFilter} title="Reset filter" style="color:var(--red)"><X size={12}/></button>
      </div>

      <!-- Sub-tab Nav -->
      <div class="card subtab-nav">
        <div class="subtab-group">
          <button class="subtab-btn" class:active={activeMdcwSubTab==="ringkasan"} onclick={()=>activeMdcwSubTab="ringkasan"}>
            <LayoutDashboard size={11}/> RINGKASAN
          </button>
          <button class="subtab-btn" class:active={activeMdcwSubTab==="qc_mesin"}  onclick={()=>activeMdcwSubTab="qc_mesin"}>
            <ShieldCheck size={11}/> QC &amp; MESIN
          </button>
          <button class="subtab-btn" class:active={activeMdcwSubTab==="per_shift"} onclick={()=>activeMdcwSubTab="per_shift"}>
            <Clock size={11}/> PER SHIFT
          </button>
          <button class="subtab-btn" class:active={activeMdcwSubTab==="detail"}    onclick={()=>activeMdcwSubTab="detail"}>
            <TrendingUp size={11}/> DETAIL LOGS
          </button>
        </div>
        <div class="row-flex gap-2">
          {#if records.length}<span class="subtab-tag">{records.length} records loaded</span>{/if}
          <span class="subtab-tag"><Settings size={9}/> MDCW ENGINE</span>
        </div>
      </div>

      <!-- Data origin -->
      <div class="data-info">
        <span><Wifi size={10}/> <strong>SOURCE</strong> WEIGHING SENSORS → MQTT <code>emqx:1883</code> → <code>production/mdcw</code> → POSTGRES</span>
        <span><Clock size={10}/> Real-time · auto-refresh 15s</span>
        {#if startDate && endDate}<span style="color:var(--accent-2)"><CalendarDays size={10}/> Range: {startDate} → {endDate}</span>{/if}
      </div>

      <!-- ── TAB: RINGKASAN ── -->
      {#if activeMdcwSubTab === "ringkasan"}

        <div class="dashboard-grid">
          <!-- KPI Grid -->
          <div class="kpi-grid">
            <div class="kpi-card" style="border-top:2px solid var(--accent)">
              <div class="kpi-card-header"><span class="kpi-title">Total Output</span><Boxes size={12} style="color:var(--text-dim)"/></div>
              <div class="kpi-value">{summaries.length?overallStats.output:'—'}</div>
              <span class="kpi-subtext">packs hari ini</span>
            </div>
            <div class="kpi-card" style="border-top:2px solid var(--green)">
              <div class="kpi-card-header"><span class="kpi-title">Pass Rate</span><ShieldCheck size={12} style="color:var(--text-dim)"/></div>
              <div class="kpi-value" style="color:var(--green)">{summaries.length?overallStats.passRate.toFixed(1)+'%':'—'}</div>
              <span class="kpi-subtext">target &gt;95%</span>
            </div>
            <div class="kpi-card" style="border-top:2px solid var(--yellow)">
              <div class="kpi-card-header"><span class="kpi-title">Total Reject</span><AlertTriangle size={12} style="color:var(--text-dim)"/></div>
              <div class="kpi-value" style="color:var(--yellow)">{summaries.length?overallStats.reject:'—'}</div>
              <span class="kpi-subtext">under / over / metal</span>
            </div>

            <!-- Metal -->
            <div class="kpi-card" style="border-top:2px solid var(--red)">
              <div class="kpi-card-header">
                <span class="kpi-title">Metal Alert</span>
                {#if overallStats.metal>0}<span class="badge badge-red" style="animation:blink 1s infinite">CRITICAL</span>{/if}
              </div>
              <div class="kpi-value" style="color:{overallStats.metal>0?'var(--red)':'var(--text)'}">
                {summaries.length?overallStats.metal:'—'}
              </div>
              <span class="kpi-subtext">kontaminasi logam</span>
              {#if sparklineMetalPoints.length >= 2}
                <div class="sparkline-container">
                  <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                    <defs><linearGradient id="g-metal" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stop-color="var(--red)" stop-opacity="0.15"/><stop offset="100%" stop-color="var(--red)" stop-opacity="0"/>
                    </linearGradient></defs>
                    <path d={getSparklinePath(sparklineMetalPoints)} fill="none" stroke="var(--red)" stroke-width="1.2"/>
                    <path d="{getSparklinePath(sparklineMetalPoints)} L 100 30 L 0 30 Z" fill="url(#g-metal)"/>
                  </svg>
                </div>
              {/if}
            </div>

            <!-- OEE -->
            <div class="kpi-card" style="border-top:2px solid var(--purple)">
              <div class="kpi-card-header"><span class="kpi-title">OEE Rata-rata</span><Gauge size={12} style="color:var(--text-dim)"/></div>
              <div class="kpi-value" style="color:var(--purple)">{records.length?overallStats.avgOee.toFixed(1)+'%':'—'}</div>
              <span class="kpi-subtext">avail · perf · quality</span>
              {#if sparklineOeePoints.length >= 2}
                <div class="sparkline-container">
                  <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                    <defs><linearGradient id="g-oee" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stop-color="var(--purple)" stop-opacity="0.15"/><stop offset="100%" stop-color="var(--purple)" stop-opacity="0"/>
                    </linearGradient></defs>
                    <path d={getSparklinePath(sparklineOeePoints)} fill="none" stroke="var(--purple)" stroke-width="1.2"/>
                    <path d="{getSparklinePath(sparklineOeePoints)} L 100 30 L 0 30 Z" fill="url(#g-oee)"/>
                  </svg>
                </div>
              {/if}
            </div>

            <!-- BB Kritis -->
            <div class="kpi-card" style="border-top:2px solid var(--orange)">
              <div class="kpi-card-header">
                <span class="kpi-title">BB Kritis</span>
                {#if overallStats.bbKritis>0}<span class="badge badge-yellow">ATTN</span>{/if}
              </div>
              <div class="kpi-value" style="color:var(--orange)">{conveyorRecords.length?overallStats.bbKritis:'—'}</div>
              <span class="kpi-subtext">umur simpan &lt;30d</span>
              {#if sparklineOutputPoints.length >= 2}
                <div class="sparkline-container">
                  <svg viewBox="0 0 100 30" width="100%" height="30" preserveAspectRatio="none">
                    <defs><linearGradient id="g-bb" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stop-color="var(--orange)" stop-opacity="0.15"/><stop offset="100%" stop-color="var(--orange)" stop-opacity="0"/>
                    </linearGradient></defs>
                    <path d={getSparklinePath(sparklineOutputPoints)} fill="none" stroke="var(--orange)" stroke-width="1.2"/>
                    <path d="{getSparklinePath(sparklineOutputPoints)} L 100 30 L 0 30 Z" fill="url(#g-bb)"/>
                  </svg>
                </div>
              {/if}
            </div>
          </div>

          <!-- Trend Chart -->
          <div class="trend-card">
            <div class="card-label row-flex gap-1"><TrendingUp size={10}/> Tren Produksi 7 Hari</div>
            {#if mdcwDailyStats.length}
              <div class="trend-canvas-wrap"><canvas bind:this={canvasTrend}></canvas></div>
            {:else}
              <div class="chart-empty"><BarChart3 size={28}/><span>Belum ada data tren</span></div>
            {/if}
          </div>
        </div>

        <!-- Machine OEE + Drift -->
        <div class="bottom-lists-grid">
          <div class="topic-list-card">
            <div class="topic-list-title"><CircleDot size={10}/> Live Machine OEE &amp; Status</div>
            {#if records.length}
              <div class="topic-list">
                {#each MDCW_PREFIXES as pfx, idx}
                  {@const latestRec = records.find(r=>r.prefix===pfx)}
                  {@const oeeStats  = getOeeForPrefix(pfx)}
                  {@const isMati    = !latestRec||latestRec.reg5===8}
                  {@const isIdle    = latestRec&&(latestRec.reg5===9||latestRec.reg5===90)}
                  {@const stateText = isMati?"MATI":isIdle?"IDLE":"RUNNING"}
                  {@const stateColor = isMati?"var(--red)":isIdle?"var(--yellow)":"var(--green)"}
                  {@const oeeValue  = oeeStats.oee*100}
                  <div class="topic-item">
                    <div class="topic-meta">
                      <div class="topic-thumb">M{idx+1}</div>
                      <div class="topic-info">
                        <span class="topic-name">{pfx}</span>
                        <span class="topic-subname" style="color:{stateColor};font-weight:700">{stateText} · {productCodeToName[prefixToProductCode[pfx]]||'-'}</span>
                      </div>
                    </div>
                    <div class="topic-bar-container">
                      <div class="topic-bar-outer"><div class="topic-bar-inner" style="width:{oeeValue}%;background:linear-gradient(90deg,var(--accent),var(--accent-2))"></div></div>
                      <span class="topic-percentage">{oeeValue.toFixed(0)}%</span>
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="list-empty"><Gauge size={22}/><span>Belum ada data mesin</span></div>
            {/if}
          </div>

          <div class="topic-list-card">
            <div class="topic-list-title"><Activity size={10}/> Audit Penyimpangan Gramasi</div>
            {#if gramasiDriftStats.length}
              <div class="topic-list">
                {#each gramasiDriftStats.slice(0,9) as d}
                  {@const driftPct = Math.min(100,Math.max(0,100-Math.round((Math.abs(d.drift)/d.std)*100*20)))}
                  {@const barColor = d.status.includes("Drift")?"var(--red)":d.status.includes("Warning")?"var(--yellow)":"var(--green)"}
                  <div class="topic-item">
                    <div class="topic-meta">
                      <div class="topic-thumb" style="color:{barColor}">{d.kode.slice(-2)}</div>
                      <div class="topic-info">
                        <span class="topic-name">{d.nama}</span>
                        <span class="topic-subname">std:{d.std.toFixed(1)}g · avg:{d.avg.toFixed(1)}g</span>
                      </div>
                    </div>
                    <div class="topic-bar-container">
                      <div class="topic-bar-outer"><div class="topic-bar-inner" style="width:{driftPct}%;background:{barColor}"></div></div>
                      <span class="topic-percentage" style="color:{barColor}">{d.drift>=0?'+':''}{d.drift.toFixed(1)}g</span>
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="list-empty"><Activity size={22}/><span>Tidak ada data gramasi yang cocok</span></div>
            {/if}
          </div>
        </div>

        <!-- Priority Alerts -->
        <div class="card">
          <div class="card-label row-flex gap-1 mb-2" style="color:var(--red)"><AlertTriangle size={10}/> Alert Otomatis Prioritas</div>
          <div class="col-flex" style="gap:5px">
            {#each priorityAlerts as alert}
              <div class="alert-strip {alert.type}">
                {#if alert.type==="critical"}<AlertTriangle size={11}/>{:else}<ShieldCheck size={11}/>{/if}
                {alert.msg}
              </div>
            {/each}
          </div>
        </div>

      <!-- ── TAB: QC & MESIN ── -->
      {:else if activeMdcwSubTab === "qc_mesin"}
        <div class="qc-grid">
          <div class="col-flex">
            <div class="card">
              <div class="card-label mb-2">Reject Breakdown Per Mesin</div>
              {#if summaries.length}
                <div class="chart-canvas-wrap"><canvas bind:this={canvasReject}></canvas></div>
              {:else}
                <div class="chart-empty"><BarChart3 size={24}/><span>Belum ada data summary</span></div>
              {/if}
            </div>
            <div class="card">
              <div class="card-label mb-2">Durasi Uptime vs Idle vs Downtime</div>
              {#if records.length}
                <div style="overflow-x:auto">
                  <table class="data-table">
                    <thead><tr>{#each ["MESIN","UPTIME","IDLE","DOWNTIME","RATIO"] as h}<th>{h}</th>{/each}</tr></thead>
                    <tbody>
                      {#each MDCW_PREFIXES as pfx}
                        {@const d=getOeeForPrefix(pfx)}
                        {@const total=d.uptime+d.idle+d.downtime||1}
                        <tr>
                          <td class="td-mono fw-700">{pfx}</td>
                          <td class="td-mono td-green">{Math.round(d.uptime)}s ({((d.uptime/total)*100).toFixed(0)}%)</td>
                          <td class="td-mono td-yellow">{Math.round(d.idle)}s ({((d.idle/total)*100).toFixed(0)}%)</td>
                          <td class="td-mono td-red">{Math.round(d.downtime)}s ({((d.downtime/total)*100).toFixed(0)}%)</td>
                          <td><div class="stacked-bar">
                            <div style="width:{(d.uptime/total)*100}%;background:var(--green)"></div>
                            <div style="width:{(d.idle/total)*100}%;background:var(--yellow)"></div>
                            <div style="width:{(d.downtime/total)*100}%;background:var(--red)"></div>
                          </div></td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </div>
              {:else}
                <div class="empty-cell"><span>Belum ada data sensor</span></div>
              {/if}
            </div>
          </div>
          <div class="card">
            <div class="card-label mb-2">OEE Tiga Komponen (Sensor)</div>
            {#if records.length}
              <table class="data-table">
                <thead><tr>{#each ["MESIN","AVAIL%","PERF%","QUAL%","OEE%"] as h}<th>{h}</th>{/each}</tr></thead>
                <tbody>
                  {#each MDCW_PREFIXES as pfx}
                    {@const o=getOeeForPrefix(pfx)}
                    {@const ov=o.oee*100}
                    {@const oc=ov>=85?"var(--green)":ov>=60?"var(--yellow)":"var(--red)"}
                    <tr>
                      <td class="td-mono fw-700">{pfx}</td>
                      <td class="td-mono">{o.availability.toFixed(0)}</td>
                      <td class="td-mono">{o.performance.toFixed(0)}</td>
                      <td class="td-mono">{o.quality.toFixed(0)}</td>
                      <td class="td-mono td-lg fw-700" style="color:{oc}">{ov.toFixed(1)}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {:else}
              <div class="empty-cell"><span>Belum ada data</span></div>
            {/if}
          </div>
        </div>

      <!-- ── TAB: PER SHIFT ── -->
      {:else if activeMdcwSubTab === "per_shift"}
        <div class="col-flex">
          <div class="card">
            <div class="shift-header">
              <div>
                <div class="card-label row-flex gap-1 mb-2"><Clock size={10}/> Filter Analisis Shift</div>
                <div class="row-flex gap-2">
                  {#each ["all","1","2","3"] as sft}
                    <button class="btn" class:btn-primary={selectedShift===sft} class:btn-ghost={selectedShift!==sft}
                      onclick={()=>selectedShift=sft}>{sft==="all"?"SEMUA":"SHIFT "+sft}</button>
                  {/each}
                </div>
              </div>
              <div class="shift-kpi-row">
                <div class="shift-kpi">
                  <div class="shift-kpi-lbl">Output</div>
                  <div class="shift-kpi-val">{shiftRecs.length||'—'}</div>
                </div>
                <div class="shift-kpi">
                  <div class="shift-kpi-lbl">Pass Rate</div>
                  <div class="shift-kpi-val" style="color:var(--green)">{shiftRecs.length?passRateShift.toFixed(1)+'%':'—'}</div>
                </div>
                <div class="shift-kpi">
                  <div class="shift-kpi-lbl">Reject</div>
                  <div class="shift-kpi-val" style="color:var(--yellow)">{shiftRecs.length?rejectShift:'—'}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="card">
            <div class="row-flex justify-between mb-2">
              <div class="card-label row-flex gap-1"><Activity size={10}/> Audit Gramasi — Aktual vs Standar</div>
              <span class="badge badge-purple">Toleransi: 2%</span>
            </div>
            {#if gramasiDriftStats.length}
              <div style="overflow-x:auto">
                <table class="data-table">
                  <thead><tr>{#each ["KODE","NAMA PRODUK","GRAMASI STD","AVG SENSOR","DRIFT","STATUS"] as h}<th>{h}</th>{/each}</tr></thead>
                  <tbody>
                    {#each gramasiDriftStats as d}
                      <tr>
                        <td class="td-mono fw-700">{d.kode}</td>
                        <td>{d.nama}</td>
                        <td class="td-mono">{d.std.toFixed(1)} g</td>
                        <td class="td-mono td-purple">{d.avg.toFixed(1)} g</td>
                        <td class="td-mono fw-700" style="color:{d.drift>=0?'var(--green)':'var(--red)'}">
                          {d.drift>=0?'+':''}{d.drift.toFixed(1)}g ({((d.drift/d.std)*100).toFixed(2)}%)
                        </td>
                        <td><span class="badge {d.statusCls}">{d.status}</span></td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {:else}
              <div class="empty-cell">
                <span class="empty-icon"><Activity size={20}/></span>
                <span>Tidak ada kecocokan data shift untuk audit gramasi</span>
                <span class="empty-hint">Pastikan filter tanggal sudah dipilih dan data batch tersedia</span>
              </div>
            {/if}
          </div>
        </div>

      <!-- ── TAB: DETAIL LOGS ── -->
      {:else if activeMdcwSubTab === "detail"}
        <div class="split-tables-wrapper">

          <!-- MDCW Sensor Log -->
          <div class="card" style="overflow:hidden">
            <div class="table-header">
              <div class="table-title"><span class="section-dot dot-purple"></span> MDCW Sensor Log</div>
              <div class="row-flex gap-2">
                {#if records.length}<span class="subtab-tag">{records.length} rows</span>{/if}
                <span class="live-dot" style="font-size:8px">LIVE</span>
              </div>
            </div>
            <!-- Sort controls -->
            <div class="sort-bar">
              <span class="sort-label">Sort by:</span>
              {#each [["ts","Waktu"],["prefix","Mesin"],["weight_formatted","Berat"],["reg5","Status"]] as [col,lbl]}
                <button class="sort-btn" class:active={mdcwSortCol===col} onclick={()=>toggleMdcwSort(col)}>
                  {lbl}
                  {#if mdcwSortCol===col}
                    {#if mdcwSortDir==="desc"}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}
                  {:else}<ArrowUpDown size={9}/>{/if}
                </button>
              {/each}
              <div style="flex:1"></div>
              <button class="btn btn-ghost" onclick={exportCSV} style="gap:4px"><Download size={9}/> CSV</button>
              <button class="btn btn-ghost" onclick={exportExcelFromRecords} style="gap:4px"><FileSpreadsheet size={9}/> Excel</button>
            </div>
            <div class="table-scroll">
              <table class="data-table">
                <thead>
                  <tr>
                    <th onclick={()=>toggleMdcwSort('id')}><span class="sort-btn" class:active={mdcwSortCol==='id'}>ID{#if mdcwSortCol==='id'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleMdcwSort('ts')}><span class="sort-btn" class:active={mdcwSortCol==='ts'}>TIMESTAMP{#if mdcwSortCol==='ts'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleMdcwSort('prefix')}><span class="sort-btn" class:active={mdcwSortCol==='prefix'}>PREFIX{#if mdcwSortCol==='prefix'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleMdcwSort('weight_formatted')}><span class="sort-btn" class:active={mdcwSortCol==='weight_formatted'}>BERAT{#if mdcwSortCol==='weight_formatted'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleMdcwSort('reg2')}><span class="sort-btn" class:active={mdcwSortCol==='reg2'}>PACK{#if mdcwSortCol==='reg2'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleMdcwSort('reg5')}><span class="sort-btn" class:active={mdcwSortCol==='reg5'}>STATUS{#if mdcwSortCol==='reg5'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleMdcwSort('data_type')}><span class="sort-btn" class:active={mdcwSortCol==='data_type'}>DATA TYPE{#if mdcwSortCol==='data_type'}{#if mdcwSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                  </tr>
                </thead>
                <tbody>
                  {#each sortedRecords() as r}
                    {@const st=statusLabel(r.reg5)}
                    {@const dtCls=r.data_type==="VALID"?"badge-green":r.data_type==="TEST"?"badge-blue":r.data_type==="SPAM"?"badge-red":"badge-yellow"}
                    <tr>
                      <td class="td-mono td-muted">{r.id}</td>
                      <td class="td-mono td-nowrap td-muted">{r.ts}</td>
                      <td><span class="badge badge-blue">{r.prefix}</span></td>
                      <td class="td-mono td-purple td-lg td-nowrap">{r.weight_formatted}</td>
                      <td class="td-mono td-lg">{r.reg2}</td>
                      <td><span class="badge {st.cls}">{st.label}</span></td>
                      <td><span class="badge {dtCls}">{r.data_type}</span></td>
                    </tr>
                  {:else}
                    <tr><td colspan="7" class="empty-cell">
                      <span class="empty-icon"><Gauge size={20}/></span>
                      <span>Tidak ada data sensor</span>
                      <span class="empty-hint">Pilih filter atau tunggu data masuk real-time</span>
                    </td></tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

          <!-- Batch Log -->
          <div class="card" style="overflow:hidden">
            <div class="table-header">
              <div class="table-title"><span class="section-dot dot-green"></span> Production Batch Log</div>
              <div class="row-flex gap-2">
                {#if conveyorRecords.length}<span class="subtab-tag">{conveyorRecords.length} rows</span>{/if}
              </div>
            </div>
            <!-- Sort controls -->
            <div class="sort-bar">
              <span class="sort-label">Sort by:</span>
              {#each [["tanggal_record","Waktu"],["kode_produk","Produk"],["shift","Shift"],["kode_batch","Batch"]] as [col,lbl]}
                <button class="sort-btn" class:active={convSortCol===col} onclick={()=>toggleConvSort(col)}>
                  {lbl}
                  {#if convSortCol===col}
                    {#if convSortDir==="desc"}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}
                  {:else}<ArrowUpDown size={9}/>{/if}
                </button>
              {/each}
              <div style="flex:1"></div>
              <button class="btn btn-ghost" onclick={exportConveyorExcel} style="gap:4px"><FileSpreadsheet size={9}/> Excel</button>
            </div>
            <div class="table-scroll">
              <table class="data-table">
                <thead>
                  <tr>
                    <th onclick={()=>toggleConvSort('id_record')}><span class="sort-btn" class:active={convSortCol==='id_record'}>ID{#if convSortCol==='id_record'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('tanggal_record')}><span class="sort-btn" class:active={convSortCol==='tanggal_record'}>TIMESTAMP{#if convSortCol==='tanggal_record'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('kode_produk')}><span class="sort-btn" class:active={convSortCol==='kode_produk'}>PRODUK{#if convSortCol==='kode_produk'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('qty_per_pack')}><span class="sort-btn" class:active={convSortCol==='qty_per_pack'}>QTY{#if convSortCol==='qty_per_pack'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('shift')}><span class="sort-btn" class:active={convSortCol==='shift'}>SHF{#if convSortCol==='shift'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('kode_batch')}><span class="sort-btn" class:active={convSortCol==='kode_batch'}>BATCH{#if convSortCol==='kode_batch'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('tanggal_best_before')}><span class="sort-btn" class:active={convSortCol==='tanggal_best_before'}>BEST BEFORE{#if convSortCol==='tanggal_best_before'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                    <th onclick={()=>toggleConvSort('factory')}><span class="sort-btn" class:active={convSortCol==='factory'}>FC{#if convSortCol==='factory'}{#if convSortDir==='desc'}<ArrowDown size={9}/>{:else}<ArrowUp size={9}/>{/if}{/if}</span></th>
                  </tr>
                </thead>
                <tbody>
                  {#each sortedConveyorRecords() as c}
                    {@const daysLeft=getDaysRemaining(c.tanggal_best_before)}
                    <tr>
                      <td class="td-mono td-muted">{c.id_record}</td>
                      <td class="td-mono td-nowrap td-muted">{c.tanggal_record?c.tanggal_record.slice(0,19).replace('T',' '):'-'}</td>
                      <td class="td-mono td-accent">
                        {c.kode_produk} <span class="td-muted" style="font-size:8px;font-weight:400">({productCodeToName[c.kode_produk]||'?'})</span>
                      </td>
                      <td class="td-mono td-lg">{c.qty_per_pack}</td>
                      <td class="td-mono">S{c.shift}</td>
                      <td><span class="badge badge-green">{c.kode_batch}</span></td>
                      <td class="td-mono td-nowrap">
                        {c.tanggal_best_before}
                        {#if daysLeft!==null}
                          {#if daysLeft<=7}<span class="badge badge-red" style="margin-left:3px;animation:blink 1s infinite">{daysLeft}d</span>
                          {:else if daysLeft<=30}<span class="badge badge-yellow" style="margin-left:3px">{daysLeft}d</span>
                          {:else}<span class="badge badge-green" style="margin-left:3px;opacity:.6">{daysLeft}d</span>{/if}
                        {/if}
                      </td>
                      <td class="td-mono td-muted">{c.factory}</td>
                    </tr>
                  {:else}
                    <tr><td colspan="8" class="empty-cell">
                      <span class="empty-icon"><Layers size={20}/></span>
                      <span>Tidak ada data batch</span>
                      <span class="empty-hint">Pilih filter shift/tanggal atau tunggu data batch masuk</span>
                    </td></tr>
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
