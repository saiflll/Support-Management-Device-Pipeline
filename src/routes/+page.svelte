<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";
  import { getCurrentWindow } from "@tauri-apps/api/window";
  import SplitTerminal from "$lib/components/SplitTerminal.svelte";
  import AIChat from "$lib/components/AIChat.svelte";
  import GitStatus from "$lib/components/GitStatus.svelte";
  import FileManager from "$lib/components/FileManager.svelte";
  import CodeEditor from "$lib/components/CodeEditor.svelte";
  import Settings from "$lib/components/Settings.svelte";
  import Notepad from "$lib/components/Notepad.svelte";
  import { currentDir, type Agent } from "$lib/stores.svelte";
  import { onMount } from "svelte";
  import { getStoredTheme, getStoredFont, applyTheme, applyFont } from "$lib/themes";

  type TabType = "file" | "settings" | "setup" | "terminal" | "preview";
  type SidebarView = "files" | "git" | "platformio" | "aicost" | null;

  let tabs = $state<Tab[]>([
    { id: "tab-term", type: "terminal", label: "Terminal" },
  ]);
  let activeTabId = $state("tab-term");

  type Tab = {
    id: string; type: TabType; label: string;
    filePath?: string; fileContent?: string; previewUrl?: string;
    isNew?: boolean;
  };

  // ─── Sidebar ───────────────────────────────────
  let sidebarView = $state<SidebarView>("files");
  let sidebarWidth = $state(260);

  // ─── Floating Panels ──────────────────────────
  let showFloatingAi = $state(false);
  let showFloatingNotepad = $state(false);
  let showLogs = $state(false);

  // ─── Primary CWD (from terminal) ──────────────
  let primaryCwd = $state("C:\\Users\\Lenovo\\Documents\\dev\\contlib");
  let activeFilePath = $state("");

  // ─── App Logs ─────────────────────────────────
  let logs = $state<{ time: string; msg: string; type: string }[]>([]);
  function addLog(msg: string, type = "info") {
    const t = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
    logs = [...logs, { time: t, msg, type }];
  }
  // Capture console.log/error
  const origLog = console.log;
  const origError = console.error;
  console.log = (...args: any[]) => { origLog(...args); addLog(args.map(a => typeof a === "string" ? a : JSON.stringify(a)).join(" "), "info"); };
  console.error = (...args: any[]) => { origError(...args); addLog(args.map(a => typeof a === "string" ? a : JSON.stringify(a)).join(" "), "error"); };
  addLog("App started");

  // ─── Clock ────────────────────────────────────
  let now = $state(new Date());
  $effect(() => {
    const id = setInterval(() => { now = new Date(); }, 1000);
    return () => clearInterval(id);
  });
  function fmtTime(d: Date) { return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }); }

  // ─── Platform IO ──────────────────────────────
  type PioResult = { success: boolean; output: string; error: string | null };
  type PioStatus = { installed: boolean; version: string | null; python: string | null; error?: string };
  let pioStatus = $state<PioStatus>({ installed: false, version: null, python: null });
  let pioBoards = $state<string[]>([]);
  let pioBusy = $state(false);
  let pioInstalling = $state(false);
  let pioInitPath = $state("");
  let pioStatusMsg = $state("");
  let pioOutput = $state("");
  let pioOutputExpanded = $state(false);
  let pioActionBusy = $state<Record<string, boolean>>({});
  let pioBoardSearch = $state("");
  let pioBoardsError = $state("");

  async function checkPio() {
    pioBoardsError = "";
    try {
      const s = await invoke<PioStatus>("pio_detect");
      pioStatus = s;
      if (s.installed) {
        addLog(`PlatformIO detected: ${s.version}`);
        try {
          const boards = await invoke<string[]>("pio_list_boards", { search: null });
          pioBoards = boards;
        } catch (e: any) {
          pioBoardsError = `Failed to load boards: ${e}`;
          pioBoards = [];
        }
      }
    } catch (e: any) {
      pioStatus = { installed: false, version: null, python: null, error: String(e) };
    }
  }

  function refreshPio() {
    pioBoards = [];
    pioBoardsError = "";
    pioOutput = "";
    pioStatus = { installed: false, version: null, python: null };
    checkPio();
  }

  async function installPio() {
    pioInstalling = true;
    pioStatusMsg = "Installing PlatformIO...";
    try {
      const res = await invoke<PioResult>("pio_install");
      if (res.success) {
        pioStatusMsg = "PlatformIO installed successfully!";
        addLog("PlatformIO installed");
        await checkPio();
      } else {
        pioStatusMsg = "Installation failed";
        console.error(res.error);
      }
    } catch (e: any) {
      pioStatusMsg = "Installation error";
      console.error(e);
    }
    pioInstalling = false;
  }

  async function initPioProject(board?: string) {
    const path = pioInitPath || primaryCwd;
    if (!path) return;
    pioBusy = true;
    pioOutput = "";
    pioOutputExpanded = true;
    pioStatusMsg = "Initializing project...";
    try {
      const res = await invoke<PioResult>("pio_init", { path, board: board || null });
      pioOutput = res.output + (res.error ? `\n[STDERR]\n${res.error}` : "");
      if (res.success) {
        pioStatusMsg = board ? `Project initialized with ${board}!` : "Project initialized!";
        addLog(`PIO project initialized at ${path}${board ? ` (${board})` : ""}`);
        pioInitPath = "";
      } else {
        pioStatusMsg = "Init failed";
        console.error(res.error);
      }
    } catch (e: any) {
      pioOutput = String(e);
      pioStatusMsg = "Init error";
      console.error(e);
    }
    pioBusy = false;
    setTimeout(() => { pioStatusMsg = ""; }, 5000);
  }

  function initPioProjectWithBoard(board: string) {
    pioInitPath = primaryCwd;
    initPioProject(board);
  }

  async function runPioTarget(target: string) {
    pioActionBusy = { ...pioActionBusy, [target]: true };
    pioOutput = "";
    pioOutputExpanded = true;
    pioStatusMsg = `Running pio ${target}...`;
    try {
      const res = await invoke<PioResult>("pio_run", { target, directory: primaryCwd });
      pioOutput = res.output + (res.error ? `\n[STDERR]\n${res.error}` : "");
      pioStatusMsg = `pio ${target}: ${res.success ? "[done]" : "[failed]"}`;
      addLog(`PIO ${target}: ${res.success ? "OK" : "FAIL"}`);
      if (!res.success) console.error(res.error);
    } catch (e: any) {
      pioOutput = String(e);
      pioStatusMsg = `pio ${target} error`;
      console.error(e);
    }
    pioActionBusy = { ...pioActionBusy, [target]: false };
    setTimeout(() => { pioStatusMsg = ""; }, 6000);
  }

  function openSerialMonitor() {
    let termTab = tabs.find(t => t.type === "terminal");
    if (!termTab) {
      const id = `tab-term-${Date.now()}`;
      tabs = [...tabs, { id, type: "terminal" as TabType, label: "Terminal" }];
      termTab = tabs[tabs.length - 1];
    }
    activeTabId = termTab!.id;
    setTimeout(() => {
      invoke("pty_write", { sessionId: "term-1", data: "pio device monitor\n" }).catch(e => console.error(e));
    }, 300);
  }

  const filteredPioBoards = $derived(
    pioBoardSearch.trim()
      ? pioBoards.filter(b => b.toLowerCase().includes(pioBoardSearch.toLowerCase()))
      : pioBoards
  );

  $effect(() => {
    if (sidebarView === "platformio") checkPio();
  });

  // ─── AI Cost (Real Usage Tracking) ────────────
  type AiUsage = {
    agent_id: string; agent_name: string; provider: string; model: string;
    total_requests: number; total_input_tokens: number; total_output_tokens: number; total_cost: number;
  };
  let aiCostData = $state<AiUsage[]>([]);
  let aiCostTotal = $state({ requests: 0, cost: 0, tokens: 0 });
  let aiCostLoading = $state(false);

  async function loadAiCost() {
    aiCostLoading = true;
    try {
      const usage = await invoke<AiUsage[]>("ai_get_usage");
      aiCostData = usage;
      let requests = 0, cost = 0, tokens = 0;
      for (const u of usage) {
        requests += u.total_requests;
        cost += u.total_cost;
        tokens += u.total_input_tokens + u.total_output_tokens;
      }
      aiCostTotal = { requests, cost, tokens };
    } catch (e) {
      console.error("Failed to load AI cost data:", e);
    }
    aiCostLoading = false;
  }

  async function resetAiCost() {
    try {
      await invoke("ai_reset_usage");
      await loadAiCost();
    } catch (e) {
      console.error("Failed to reset AI cost data:", e);
    }
  }

  function fmtCost(n: number): string {
    if (n < 0.001) return "$0.0000";
    if (n < 0.01) return `$${n.toFixed(5)}`;
    if (n < 1) return `$${n.toFixed(4)}`;
    return `$${n.toFixed(3)}`;
  }

  function fmtTokens(n: number): string {
    if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
    if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
    return n.toString();
  }

  $effect(() => {
    if (sidebarView === "aicost") loadAiCost();
  });

  async function runScriptSetup(type: string) {
    const configs: Record<string, { checkCmd: string; runCmd: string; installScript: string; label: string }> = {
      antigravity: {
        checkCmd: "agy",
        runCmd: "agy\n",
        installScript: "powershell -c \"Invoke-Expression (Invoke-RestMethod https://raw.githubusercontent.com/antigravity/cli/main/install.ps1)\"\n",
        label: "Antigravity CLI",
      },
      opencode: {
        checkCmd: "opencode",
        runCmd: "opencode\n",
        installScript: "powershell -c \"Invoke-Expression (Invoke-RestMethod https://raw.githubusercontent.com/opencode-ai/cli/main/install.ps1)\"\n",
        label: "OpenCode CLI",
      },
    };

    const cfg = configs[type];
    if (!cfg) return;

    // Pastikan ada terminal tab aktif
    let termTab = tabs.find(t => t.type === "terminal");
    if (!termTab) {
      const id = `tab-term-${Date.now()}`;
      tabs = [...tabs, { id, type: "terminal" as TabType, label: "Terminal" }];
      termTab = tabs[tabs.length - 1];
    }
    activeTabId = termTab!.id;

    // Cek apakah CLI sudah terinstall
    try {
      const installed = await invoke<boolean>("sys_check_installed", { cmd: cfg.checkCmd });
      if (installed) {
        addLog(`${cfg.label} found, launching...`);
        setTimeout(() => {
          invoke("pty_write", { sessionId: "term-1", data: cfg.runCmd }).catch(e => console.error(e));
        }, 250);
      } else {
        addLog(`${cfg.label} not found, running install script...`);
        setTimeout(() => {
          invoke("pty_write", { sessionId: "term-1", data: cfg.installScript }).catch(e => console.error(e));
        }, 250);
      }
    } catch (e) {
      // Fallback: langsung install jika cek gagal
      console.error("CLI check failed, attempting install:", e);
      addLog(`${cfg.label} check failed, attempting install...`);
      setTimeout(() => {
        invoke("pty_write", { sessionId: "term-1", data: cfg.installScript }).catch(e => console.error(e));
      }, 250);
    }
  }

  // ─── Tab labels ───────────────────────────────
  const TAB_LABELS: Record<TabType, string> = {
    file: "Untitled", settings: "Settings", setup: "Setup",
    terminal: "Terminal", preview: "Preview",
  };
  const TAB_ICONS: Record<TabType, string> = {
    file: `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>`,
    settings: `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>`,
    setup: `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>`,
    terminal: `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>`,
    preview: `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>`,
  };

  // ─── Tab management ───────────────────────────
  let addMenuOpen = $state(false);

  function addTab(type: TabType, extra?: Partial<Tab>) {
    const base = TAB_LABELS[type];
    const count = tabs.filter((t) => t.type === type).length;
    const label = type === "terminal" || type === "file" ? `${base} ${count + 1}` : base;
    const id = "tab-" + Date.now().toString(36) + Math.random().toString(36).slice(2, 4);
    tabs = [...tabs, { id, type, label, ...extra }];
    activeTabId = id;
    addMenuOpen = false;
  }

  function closeTab(id: string) {
    if (tabs.length <= 1) return;
    const idx = tabs.findIndex((t) => t.id === id);
    tabs = tabs.filter((t) => t.id !== id);
    if (activeTabId === id) {
      activeTabId = tabs[Math.min(idx, tabs.length - 1)]?.id || tabs[0]?.id || "";
    }
  }

  function setActiveTab(id: string) {
    activeTabId = id;
  }

  const activeTab = $derived(tabs.find((t) => t.id === activeTabId));

  // ─── File open → create tab ──────────────────
  function onFileOpen(path: string) {
    const existing = tabs.find((t) => t.filePath === path);
    if (existing) { activeTabId = existing.id; return; }
    invoke<string>("fs_read_file", { path }).then((content) => {
      const name = path.split("\\").pop() || "Untitled";
      addTab("file", { label: name, filePath: path, fileContent: content });
    });
  }

  onMount(() => {
    currentDir.set(primaryCwd);
    applyTheme(getStoredTheme());
    applyFont(getStoredFont());
  });

  // ─── Custom Window Controls ───────────────────
  const appWindow = getCurrentWindow();
  function minimizeWindow() {
    console.log("Minimizing window...");
    appWindow.minimize().catch(err => {
      console.error("Minimize error:", err);
      alert("Minimize error: " + err);
    });
  }
  async function toggleMaximizeWindow() {
    console.log("Toggling maximize window...");
    try {
      if (await appWindow.isMaximized()) {
        await appWindow.unmaximize();
      } else {
        await appWindow.maximize();
      }
    } catch (err) {
      console.error("Maximize error:", err);
      alert("Maximize error: " + err);
    }
  }
  function closeWindow() {
    console.log("Closing window...");
    appWindow.close().catch(err => {
      console.error("Close error:", err);
      alert("Close error: " + err);
    });
  }

  function handleHeaderMousedown(e: MouseEvent) {
    if (e.button !== 0) return; // Only left-click
    const target = e.target as HTMLElement;
    if (target.closest(".mac-controls, .tab-add-wrap, button, input, select, textarea, .add-menu")) {
      return; // Skip drag when clicking active controls
    }
    appWindow.startDragging().catch(err => {
      console.log("Manual drag failed:", err);
    });
  }

  // ─── CWD from terminal ────────────────────────
  function onTerminalCwdChange(cwd: string) {
    primaryCwd = cwd;
    currentDir.set(cwd);
  }

  // ─── Sidebar ──────────────────────────────────
  const SIDEBAR_LABELS: Record<string, string> = {
    files: "Explorer", git: "Source Control", platformio: "Platform IO", aicost: "AI Cost",
  };

  function toggleSidebar(view: SidebarView) {
    sidebarView = sidebarView === view ? null : view;
  }

  // ─── Keyboard Shortcuts ────────────────────────
  function handleGlobalKeydown(e: KeyboardEvent) {
    let shortcuts = {
      sidebar: { ctrl: true, alt: false, shift: false, key: "b" },
      terminal: { ctrl: true, alt: false, shift: false, key: "j" },
      ai: { ctrl: true, alt: true, shift: false, key: "a" },
      notepad: { ctrl: true, alt: true, shift: false, key: "n" },
    };
    try {
      const stored = localStorage.getItem("contlib-shortcuts");
      if (stored) {
        Object.assign(shortcuts, JSON.parse(stored));
      }
    } catch {}

    const match = (binding: any) => {
      return (
        e.ctrlKey === binding.ctrl &&
        e.altKey === binding.alt &&
        e.shiftKey === binding.shift &&
        e.key.toLowerCase() === binding.key.toLowerCase()
      );
    };

    if (match(shortcuts.sidebar)) {
      e.preventDefault();
      toggleSidebar(sidebarView === "files" ? null : "files");
    } else if (match(shortcuts.terminal)) {
      e.preventDefault();
      const termTab = tabs.find(t => t.type === "terminal");
      if (termTab) {
        activeTabId = termTab.id;
      } else {
        addTab("terminal");
      }
    } else if (match(shortcuts.ai)) {
      e.preventDefault();
      showFloatingAi = !showFloatingAi;
      if (showFloatingAi) showFloatingNotepad = false;
    } else if (match(shortcuts.notepad)) {
      e.preventDefault();
      showFloatingNotepad = !showFloatingNotepad;
      if (showFloatingNotepad) showFloatingAi = false;
    }
  }

  // ─── Resize ───────────────────────────────────
  let isResizing = $state(false);
  let resizeStartX = $state(0);
  let resizeStartWidth = $state(260);

  function onResizeStart(e: MouseEvent) {
    isResizing = true;
    resizeStartX = e.clientX;
    resizeStartWidth = sidebarWidth;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  }
  function onResizeMove(e: MouseEvent) {
    if (!isResizing) return;
    sidebarWidth = Math.max(180, Math.min(480, resizeStartWidth + (e.clientX - resizeStartX)));
  }
  function onResizeEnd() {
    isResizing = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "none";
  }
</script>

<svelte:window onkeydown={handleGlobalKeydown} onmousemove={onResizeMove} onmouseup={onResizeEnd} onclick={() => (addMenuOpen = false)} />

<div class="workspace">
  <!-- ═══ TAB BAR ═══ -->
  <header class="tab-bar" data-tauri-drag-region onmousedown={handleHeaderMousedown}>

    <!-- macOS-style window controls (left) -->
    <div class="mac-controls">
      <button class="mac-dot mac-close" onclick={closeWindow} aria-label="Close"></button>
      <button class="mac-dot mac-minimize" onclick={minimizeWindow} aria-label="Minimize"></button>
      <button class="mac-dot mac-maximize" onclick={toggleMaximizeWindow} aria-label="Maximize"></button>
    </div>

    <!-- "+" button with rounded border -->
    <div class="tab-add-wrap">
      <button class="tab-add" onclick={(e) => { e.stopPropagation(); addMenuOpen = !addMenuOpen; }} aria-label="Add tab">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      </button>
    </div>

    <div class="tab-scroll">
      {#each tabs as tab (tab.id)}
        <button
          class="tab"
          class:tab-active={activeTabId === tab.id}
          onclick={() => setActiveTab(tab.id)}
          onkeydown={(e) => e.key === "Delete" && closeTab(tab.id)}
          role="tab"
          aria-selected={activeTabId === tab.id}
        >
          <span class="tab-icon">{@html TAB_ICONS[tab.type]}</span>
          <span class="tab-label">{tab.label}</span>
          <span class="tab-close" onclick={(e) => { e.stopPropagation(); closeTab(tab.id); }} onkeydown={(e) => { if (e.key === "Enter" || e.key === " ") { e.stopPropagation(); closeTab(tab.id); } }} role="button" tabindex="-1" aria-label="Close">&times;</span>
        </button>
      {/each}
    </div>

    <!-- Add menu (positioned absolute relative to tab-add-wrap) -->
    {#if addMenuOpen}
      <div class="add-menu" onclick={(e) => e.stopPropagation()} role="presentation">
        {#each [{ type: "terminal", icon: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>`, label: "Terminal" }, { type: "preview", icon: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>`, label: "Preview" }, { type: "file", icon: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>`, label: "Editor" }] as item}
          <button class="add-menu-item" onclick={() => addTab(item.type as TabType)}>
            <span class="add-menu-icon">{@html item.icon}</span>
            <span>{item.label}</span>
          </button>
        {/each}
      </div>
    {/if}
  </header>

  <!-- ═══ BODY ═══ -->
  <div class="body">
    <!-- Activity Bar -->
    <nav class="activity-bar">
      <div class="activity-top">
        {#each ["files", "git", "platformio", "aicost"] as view}
          <button class="activity-btn" class:active={sidebarView === view} onclick={() => toggleSidebar(view as SidebarView)} title={SIDEBAR_LABELS[view]}>
            {#if view === "files"}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
            {:else if view === "git"}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="6" y1="3" x2="6" y2="15"></line><circle cx="18" cy="6" r="3"></circle><circle cx="6" cy="18" r="3"></circle><path d="M18 9a9 9 0 0 1-9 9"></path></svg>
            {:else if view === "platformio"}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M9 12h6M12 9v6"/><path d="M7.5 7.5l9 9M7.5 16.5l9-9"/></svg>
            {:else if view === "aicost"}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/><circle cx="12" cy="12" r="4"/></svg>
            {/if}
          </button>
        {/each}
      </div>
      <div class="activity-bottom">
        <button class="activity-btn" onclick={() => { if (tabs.findIndex(t => t.type === "settings") === -1) addTab("settings"); else activeTabId = tabs.find(t => t.type === "settings")!.id; }} title="Settings" class:active={activeTab?.type === "settings"}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
        </button>
      </div>
    </nav>

    <!-- Sidebar -->
    {#if sidebarView !== null}
      <aside class="sidebar" style="width: {sidebarWidth}px">
        <div class="sidebar-header">
          <span class="sidebar-title">{SIDEBAR_LABELS[sidebarView]}</span>
          <button class="sidebar-close" onclick={() => (sidebarView = null)} aria-label="Close">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
        <div class="sidebar-body">
          {#if sidebarView === "files"}
            <FileManager onFileOpen={onFileOpen} revealPath={primaryCwd} />
          {:else if sidebarView === "git"}
            <GitStatus />
          {:else if sidebarView === "platformio"}
            <div class="pio-sidebar">
              <!-- Status header with refresh -->
              <div class="pio-status-bar">
                {#if pioBusy || pioInstalling || Object.values(pioActionBusy).some(v => v)}
                  <span class="pio-spinner"></span>
                {/if}
                <span class="pio-status-text" class:pio-installed={pioStatus.installed} class:pio-not-installed={!pioStatus.installed}>
                  {#if pioStatusMsg}
                    {pioStatusMsg}
                  {:else if pioStatus.installed}
                    {pioStatus.version || "PlatformIO ready"}
                  {:else if pioStatus.error}
                    Error checking PlatformIO
                  {:else}
                    Checking...
                  {/if}
                </span>
                <button class="pio-refresh-btn" onclick={refreshPio} title="Refresh PlatformIO" disabled={pioBusy || pioInstalling}>
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
                </button>
              </div>

              {#if !pioStatus.installed && !pioBusy}
                <div class="pio-section pio-install-section">
                  <div class="pio-install-info">
                    <svg width="32" height="32" viewBox="0 0 48 48" fill="none">
                      <circle cx="24" cy="24" r="22" stroke="currentColor" stroke-width="2" fill="none"/>
                      <path d="M16 28l8-12 8 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
                      <line x1="24" y1="16" x2="24" y2="36" stroke="currentColor" stroke-width="2"/>
                      <circle cx="24" cy="24" r="6" stroke="currentColor" stroke-width="1.5" fill="none" stroke-dasharray="3 3"/>
                    </svg>
                    <span>PlatformIO is not installed</span>
                    <span class="pio-detail">Python required. Will install via pip.</span>
                    {#if pioStatus.python}
                      <span class="pio-detail pio-py-ok">&#10003; {pioStatus.python}</span>
                    {:else}
                      <span class="pio-detail pio-py-missing">&#10007; Python not found</span>
                    {/if}
                  </div>
                  <button class="pio-install-btn" onclick={installPio} disabled={pioInstalling || !pioStatus.python}>
                    {pioInstalling ? "Installing..." : "Install PlatformIO"}
                  </button>
                </div>
              {/if}

              {#if pioStatus.installed}
                <div class="pio-section">
                  <div class="pio-section-title">QUICK ACCESS</div>
                  <button class="pio-item" onclick={() => runPioTarget("build")} disabled={!!pioActionBusy["build"]}>
                    {#if pioActionBusy["build"]}
                      <span class="pio-spinner"></span>
                    {:else}
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                    {/if}
                    <span>Build</span>
                  </button>
                  <button class="pio-item" onclick={() => runPioTarget("upload")} disabled={!!pioActionBusy["upload"]}>
                    {#if pioActionBusy["upload"]}
                      <span class="pio-spinner"></span>
                    {:else}
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                    {/if}
                    <span>Upload</span>
                  </button>
                  <button class="pio-item" onclick={() => runPioTarget("clean")} disabled={!!pioActionBusy["clean"]}>
                    {#if pioActionBusy["clean"]}
                      <span class="pio-spinner"></span>
                    {:else}
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="9" x2="15" y2="15"/><line x1="15" y1="9" x2="9" y2="15"/></svg>
                    {/if}
                    <span>Clean</span>
                  </button>
                  <button class="pio-item" onclick={openSerialMonitor}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
                    <span>Serial Monitor</span>
                  </button>
                </div>

                <div class="pio-section">
                  <div class="pio-section-title">INIT PROJECT</div>
                  <div class="pio-init-row">
                    <input class="pio-init-input" bind:value={pioInitPath} placeholder={primaryCwd || "Project path..."} />
                    <button class="pio-init-btn" onclick={() => initPioProject()} disabled={pioBusy}>Init</button>
                  </div>
                </div>

                {#if pioOutput}
                  <div class="pio-section">
                    <div class="pio-section-title pio-output-title-bar">
                      OUTPUT
                      <button class="pio-output-toggle" onclick={() => (pioOutputExpanded = !pioOutputExpanded)}>
                        {pioOutputExpanded ? "▲" : "▼"}
                      </button>
                    </div>
                    {#if pioOutputExpanded}
                      <div class="pio-output-panel">{pioOutput}</div>
                    {/if}
                  </div>
                {/if}

                <div class="pio-section">
                  <div class="pio-section-title">
                    BOARDS ({filteredPioBoards.length}{pioBoardSearch ? `/${pioBoards.length}` : ""})
                  </div>
                  <div class="pio-board-search-wrap">
                    <input class="pio-board-search" bind:value={pioBoardSearch} placeholder="Filter boards..." />
                  </div>
                  {#if pioBoardsError}
                    <div class="pio-boards-error">{pioBoardsError}</div>
                  {:else}
                    <div class="pio-boards-list">
                      {#each filteredPioBoards.slice(0, 50) as board}
                        <button class="pio-item pio-board" onclick={() => initPioProjectWithBoard(board)}>
                          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="2" width="16" height="20" rx="2"/><line x1="9" y1="6" x2="15" y2="6"/><line x1="12" y1="2" x2="12" y2="6"/><circle cx="9" cy="12" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="9" cy="17" r="1"/><circle cx="15" cy="17" r="1"/></svg>
                          <span>{board}</span>
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/if}
            </div>
          {:else if sidebarView === "aicost"}
            <div class="aicost-sidebar">
              <!-- 2 thin script buttons at top -->
              <div class="aicost-script-bar">
                <button class="aicost-script-btn" onclick={() => runScriptSetup("antigravity")}>
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
                  <span>Antigravity CLI</span>
                </button>
                <button class="aicost-script-btn" onclick={() => runScriptSetup("opencode")}>
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                  <span>OpenCode CLI</span>
                </button>
              </div>

              <!-- Summary -->
              <div class="aicost-summary">
                <div class="aicost-summary-item">
                  <span class="aicost-summary-value">{aiCostTotal.requests}</span>
                  <span class="aicost-summary-label">Requests</span>
                </div>
                <div class="aicost-summary-item">
                  <span class="aicost-summary-value">{fmtTokens(aiCostTotal.tokens)}</span>
                  <span class="aicost-summary-label">Tokens</span>
                </div>
                <div class="aicost-summary-item">
                  <span class="aicost-summary-value">{fmtCost(aiCostTotal.cost)}</span>
                  <span class="aicost-summary-label">Total Cost</span>
                </div>
              </div>

              <!-- Reset button -->
              <div class="aicost-actions">
                <button class="aicost-reset-btn" onclick={resetAiCost}>Reset Stats</button>
                <button class="aicost-refresh-btn" onclick={loadAiCost}>Refresh</button>
              </div>

              <!-- Per-agent list -->
              <div class="aicost-list">
                {#if aiCostLoading}
                  <div class="aicost-loader"><div class="spinner"></div></div>
                {:else if aiCostData.length === 0}
                  <div class="aicost-empty">No usage data yet.<br/>Send messages in <strong>AI Chat</strong> first.</div>
                {:else}
                  {#each aiCostData as agent}
                    <div class="aicost-card">
                      <div class="aicost-card-top">
                        <span class="aicost-agent-name">{agent.agent_name}</span>
                        <span class="aicost-agent-model">{agent.provider}/{agent.model}</span>
                      </div>
                      <div class="aicost-stat-row">
                        <span>Requests</span>
                        <span>{agent.total_requests}</span>
                      </div>
                      <div class="aicost-stat-row">
                        <span>Input tokens</span>
                        <span>{fmtTokens(agent.total_input_tokens)}</span>
                      </div>
                      <div class="aicost-stat-row">
                        <span>Output tokens</span>
                        <span>{fmtTokens(agent.total_output_tokens)}</span>
                      </div>
                      <div class="aicost-stat-row aicost-stat-total">
                        <span>Cost</span>
                        <span>{fmtCost(agent.total_cost)}</span>
                      </div>
                    </div>
                  {/each}
                {/if}
              </div>
            </div>
          {/if}
        </div>
        <div class="sidebar-resize" onmousedown={onResizeStart} role="presentation"></div>
      </aside>
    {/if}

    <!-- Workspace -->
    <main class="workspace-area">
      {#each tabs as tab (tab.id)}
        <div class="tab-panel" class:hidden={activeTabId !== tab.id}>
          {#if tab.type === "terminal"}
            <SplitTerminal onCwdChange={onTerminalCwdChange} />
          {:else if tab.type === "file"}
            {#if tab.filePath && tab.fileContent !== undefined}
              <CodeEditor filePath={tab.filePath} initialContent={tab.fileContent} />
            {:else}
              <div class="placeholder"><p>Open a file from <strong>Explorer</strong></p></div>
            {/if}
          {:else if tab.type === "preview"}
            <div class="preview-wrap">
              {#if tab.previewUrl}
                <div class="preview-toolbar">
                  <input class="preview-input" bind:value={tab.previewUrl} placeholder="http://localhost:PORT" />
                </div>
                <iframe class="preview-frame" src={tab.previewUrl} sandbox="allow-scripts allow-same-origin" title="Preview"></iframe>
              {:else}
                <div class="placeholder">
                  <p>Enter a URL to preview</p>
                  <input class="preview-start-input" placeholder="http://localhost:5173" onkeydown={(e) => { if (e.key === "Enter") { tab.previewUrl = (e.target as HTMLInputElement).value; } }} />
                </div>
              {/if}
            </div>
          {:else if tab.type === "settings"}
            <Settings />
          {/if}
        </div>
      {/each}
      {#if tabs.length === 0}
        <div class="placeholder"><p>Select a tab</p></div>
      {/if}
    </main>
  </div>

  <!-- ═══ STATUS BAR ═══ -->
  <footer class="status-bar">
    <div class="sb-left">
      {#if primaryCwd}
        <div class="path-breadcrumb">
          {#each primaryCwd.split("\\") as seg, i}
            {#if i > 0}<span class="path-sep">&gt;</span>{/if}
            <span class="path-seg" class:path-seg-last={i === primaryCwd.split("\\").length - 1}>{seg}</span>
          {/each}
        </div>
      {:else}
        <span class="sb-muted">No directory</span>
      {/if}
    </div>

    <div class="sb-center" onclick={() => (showLogs = !showLogs)} role="button" tabindex="0" onkeydown={(e) => e.key === "Enter" && (showLogs = !showLogs)} title="Toggle logs">
      <span class="sb-clock">{fmtTime(now)}</span>
      <span class="sb-log-badge" class:sb-log-has-error={logs.some(l => l.type === "error")}>{logs.length}</span>
    </div>

    <div class="sb-right">
      <button class="sb-btn" class:active={showFloatingAi} onclick={() => { showFloatingAi = !showFloatingAi; if (showFloatingAi) showFloatingNotepad = false; }} title="AI Chat">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2a4 4 0 0 1 4 4c0 2-2 4-4 4s-4-2-4-4a4 4 0 0 1 4-4z"/><path d="M16 14h.2a4 4 0 0 1 3.8 2.8l.8 2.2H3.2l.8-2.2A4 4 0 0 1 7.8 14H8"/></svg>
      </button>
      <button class="sb-btn" class:active={showFloatingNotepad} onclick={() => { showFloatingNotepad = !showFloatingNotepad; if (showFloatingNotepad) showFloatingAi = false; }} title="Notepad">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
      </button>
    </div>
  </footer>

  <!-- Floating AI Chat -->
  {#if showFloatingAi}
    <div class="float-panel float-ai">
      <div class="float-header">
        <span>AI Chat</span>
        <button class="float-close" onclick={() => (showFloatingAi = false)} aria-label="Close">&times;</button>
      </div>
      <div class="float-body"><AIChat /></div>
    </div>
  {/if}

  <!-- Floating Notepad -->
  {#if showFloatingNotepad}
    <div class="float-panel float-notepad">
      <div class="float-header">
        <span>Notepad</span>
        <button class="float-close" onclick={() => (showFloatingNotepad = false)} aria-label="Close">&times;</button>
      </div>
      <div class="float-body"><Notepad /></div>
    </div>
  {/if}

  <!-- Floating Logs -->
  {#if showLogs}
    <div class="float-panel float-logs">
      <div class="float-header">
        <span>App Logs</span>
        <div class="float-header-actions">
          <button class="float-btn" onclick={() => (logs = [])} title="Clear">Clear</button>
          <button class="float-close" onclick={() => (showLogs = false)} aria-label="Close">&times;</button>
        </div>
      </div>
      <div class="float-body">
        <div class="logs-list">
          {#each [...logs].reverse() as log}
            <div class="log-entry" class:log-error={log.type === "error"}>
              <span class="log-time">{log.time}</span>
              <span class="log-msg">{log.msg}</span>
            </div>
          {:else}
            <div class="log-empty">No logs yet</div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .workspace { display:flex; flex-direction:column; height:100vh; width:100vw; background:var(--bg-primary); color:var(--text-primary); overflow:hidden; }

  /* ═══ TAB BAR ═══ */
  .tab-bar {
    display:flex; align-items:flex-end; height:34px; background:var(--bg-secondary);
    border-bottom:1px solid var(--border-primary); flex-shrink:0; user-select:none; padding:0 0 0 4px; gap:0; position:relative;
    -webkit-app-region:drag;
  }
  /* macOS traffic light dots */
  .mac-controls {
    display:flex; align-items:center; gap:7px; height:100%; padding:0 6px 0 10px; flex-shrink:0;
    -webkit-app-region:no-drag;
  }
  .mac-dot {
    display:inline-flex; align-items:center; justify-content:center;
    width:12px; height:12px; border-radius:50%; border:none; cursor:pointer; padding:0;
    transition:filter 0.15s ease, background-color 0.15s ease; position:relative;
    -webkit-app-region:no-drag;
  }
  .mac-dot:hover { filter:brightness(1.2); }
  .mac-close { background:#ff5f57; border:1px solid #e0453e; }
  .mac-minimize { background:#febc2e; border:1px solid #dda01d; }
  .mac-maximize { background:#28c840; border:1px solid #1faa33; }
  /* Custom macOS-style traffic light hover symbols */
  .mac-close::before { content: "×"; position:absolute; font-family:-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; font-weight:700; font-size:var(--fs-9); color:#4c0002; opacity:0; transition:opacity 0.15s; pointer-events:none; line-height:1; }
  .mac-minimize::before { content: "–"; position:absolute; font-family:-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; font-weight:700; font-size:var(--fs-9); color:#5c3e00; opacity:0; transition:opacity 0.15s; pointer-events:none; line-height:1; top:4px; }
  .mac-maximize::before { content: "+"; position:absolute; font-family:-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; font-weight:700; font-size:var(--fs-8); color:#024c0e; opacity:0; transition:opacity 0.15s; pointer-events:none; line-height:1; }
  .mac-controls:hover .mac-dot::before { opacity:1; }
  .tab-add-wrap {
    display:flex; align-items:center; flex-shrink:0; height:100%; padding:0 6px 0 4px;
    -webkit-app-region:no-drag;
  }
  .tab-add {
    display:flex; align-items:center; justify-content:center; width:22px; height:22px;
    border:1.5px solid var(--accent-blue); border-radius:6px; background:transparent;
    color:var(--accent-blue); cursor:pointer; transition:all 0.15s ease;
    padding:0;
  }
  .tab-add:hover { background:var(--accent-blue); color:#fff; }
  .tab-scroll {
    display:flex; align-items:flex-end; gap:0; overflow-x:auto; flex:1; min-width:0; height:100%;
    scrollbar-width:none;
    -webkit-app-region:no-drag;
  }
  .tab {
    position:relative; display:inline-flex; align-items:center; gap:5px;
    padding:4px 12px 4px 10px; height:28px; margin-top:4px;
    font-size:var(--fs-11); font-weight:500; cursor:pointer; white-space:nowrap; flex-shrink:0;
    background:var(--bg-surface); color:var(--text-muted);
    border:1px solid var(--border-subtle);
    border-radius:5px 5px 0 0;
    margin-right:-1px; z-index:0; transition:all 0.12s ease;
    user-select:none; -webkit-app-region:no-drag;
    /* sticky note fold */
    background: linear-gradient(135deg, var(--bg-surface) 92%, var(--bg-secondary) 92%);
  }
  .tab::after {
    content:''; position:absolute; bottom:-1px; left:0; right:0; height:2px;
    background:var(--bg-surface); z-index:2;
  }
  .tab:hover { color:var(--text-primary); background:var(--bg-hover); }
  .tab:hover::after { background:var(--bg-hover); }
  .tab-active {
    color:var(--text-primary); background:var(--bg-primary); z-index:3; border-bottom-color:var(--bg-primary);
  }
  .tab-active::after { background:var(--bg-primary); }
  .tab-icon { font-size:var(--font-size); line-height:1; }
  .tab-label { font-size:var(--fs-11); max-width:120px; overflow:hidden; text-overflow:ellipsis; }
  .tab-close {
    display:inline-flex; align-items:center; justify-content:center;
    width:16px; height:16px; font-size:var(--fs-14); line-height:1;
    opacity:0; border-radius:3px; transition:all 0.1s ease;
    color:var(--text-muted);
  }
  .tab:hover .tab-close, .tab-active .tab-close { opacity:0.7; }
  .tab-close:hover { opacity:1 !important; color:var(--accent-red); background:var(--bg-hover); }

  /* Add button + dropdown */
  .tab-add-wrap { position:relative; display:flex; align-items:flex-end; height:100%; margin-left:4px; -webkit-app-region:no-drag; }
  .tab-add {
    display:flex; align-items:center; justify-content:center;
    width:26px; height:26px; margin-bottom:2px;
    border:none; background:transparent; color:var(--text-muted);
    cursor:pointer; border-radius:5px; transition:all 0.12s ease;
  }
  .tab-add:hover { color:var(--text-primary); background:var(--bg-hover); }
  .add-menu {
    position:absolute; top:100%; left:100px; margin-top:2px;
    background:var(--bg-elevated); border:1px solid var(--border-primary);
    border-radius:8px; padding:4px; z-index:200;
    box-shadow:0 8px 24px rgba(0,0,0,0.4); min-width:160px;
    animation:floatUp 0.15s ease;
  }
  .add-menu-item {
    display:flex; align-items:center; gap:8px; width:100%;
    padding:6px 10px; border:none; background:transparent;
    color:var(--text-primary); font-size:var(--font-size); cursor:pointer;
    border-radius:5px; transition:all 0.1s ease;
  }
  .add-menu-item:hover { background:var(--bg-hover); }
  .add-menu-icon { display:flex; }

  /* ═══ BODY ═══ */
  .body { flex:1; display:flex; overflow:hidden; }

  .activity-bar { width:var(--activity-bar-width); display:flex; flex-direction:column; align-items:center; background:var(--bg-secondary); border-right:1px solid var(--border-primary); padding:4px 0; flex-shrink:0; gap:2px; }
  .activity-top { display:flex; flex-direction:column; align-items:center; gap:2px; flex:1; }
  .activity-bottom { display:flex; flex-direction:column; align-items:center; gap:2px; }
  .activity-btn { display:flex; align-items:center; justify-content:center; width:40px; height:40px; background:none; border:none; color:var(--text-muted); cursor:pointer; border-radius:8px; transition:all 0.15s ease; position:relative; }
  .activity-btn:hover { color:var(--text-primary); background:var(--bg-hover); }
  .activity-btn.active { color:var(--accent-blue); }
  .activity-btn.active::before { content:''; position:absolute; left:-5px; top:50%; transform:translateY(-50%); width:3px; height:20px; background:var(--accent-blue); border-radius:0 3px 3px 0; }

  .sidebar { display:flex; flex-direction:column; background:var(--bg-secondary); border-right:1px solid var(--border-primary); overflow:hidden; flex-shrink:0; position:relative; animation:slideIn 0.15s ease; }
  @keyframes slideIn { from { opacity:0; transform:translateX(-8px); } to { opacity:1; transform:translateX(0); } }
  .sidebar-header { display:flex; align-items:center; justify-content:space-between; padding:6px 12px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .sidebar-title { font-size:var(--fs-10); font-weight:600; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px; }
  .sidebar-close { background:none; border:none; color:var(--text-muted); padding:2px; cursor:pointer; border-radius:3px; display:flex; }
  .sidebar-close:hover { color:var(--text-primary); background:var(--bg-hover); }
  .sidebar-body { flex:1; display:flex; flex-direction:column; overflow:hidden; min-width:0; }
  .sidebar-resize { position:absolute; top:0; right:-3px; width:6px; height:100%; cursor:col-resize; z-index:10; }
  .sidebar-resize:hover, .sidebar-resize:active { background:var(--accent-blue); opacity:0.4; }

  /* Platform IO */
  .pio-sidebar { display:flex; flex-direction:column; height:100%; overflow-y:auto; font-size:var(--font-size); }
  .pio-status-bar { display:flex; align-items:center; gap:8px; padding:8px 12px; border-bottom:1px solid var(--border-subtle); }
  .pio-status-text { font-size:var(--fs-11); font-weight:500; }
  .pio-installed { color:var(--accent-green); }
  .pio-not-installed { color:var(--accent-yellow); }
  .pio-spinner { width:12px; height:12px; border:2px solid var(--border-subtle); border-top-color:var(--accent-blue); border-radius:50%; animation:pioSpin 0.6s linear infinite; }
  @keyframes pioSpin { to{transform:rotate(360deg)} }
  .pio-section { padding:8px 0; border-bottom:1px solid var(--border-subtle); }
  .pio-section:last-child { border-bottom:none; }
  .pio-section-title { font-size:var(--fs-10); font-weight:600; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px; padding:0 12px 6px; }
  .pio-item { display:flex; align-items:center; gap:8px; width:100%; padding:5px 12px; border:none; background:none; color:var(--text-primary); font-size:var(--font-size); cursor:pointer; transition:all 0.1s ease; }
  .pio-item:hover { background:var(--bg-hover); color:var(--accent-blue); }
  .pio-board { font-size:var(--fs-11); color:var(--text-secondary); }
  /* Install section */
  .pio-install-section { display:flex; flex-direction:column; gap:10px; padding:16px 12px; align-items:center; }
  .pio-install-info { display:flex; flex-direction:column; align-items:center; gap:6px; text-align:center; color:var(--text-secondary); font-size:var(--font-size); }
  .pio-detail { font-size:var(--fs-10); color:var(--text-muted); }
  .pio-py-ok { color:var(--accent-green); }
  .pio-py-missing { color:var(--accent-red); }
  .pio-install-btn { padding:7px 20px; border:1.5px solid var(--accent-blue); border-radius:6px; background:transparent; color:var(--accent-blue); font-size:var(--font-size); cursor:pointer; transition:all 0.15s ease; }
  .pio-install-btn:hover:not(:disabled) { background:var(--accent-blue); color:#fff; }
  .pio-install-btn:disabled { opacity:0.5; cursor:default; }
  /* Init project */
  .pio-init-row { display:flex; gap:4px; padding:0 12px; }
  .pio-init-input { flex:1; background:var(--bg-surface); border:1px solid var(--border-subtle); border-radius:4px; padding:4px 8px; font-size:var(--fs-11); color:var(--text-primary); font-family:monospace; min-width:0; }
  .pio-init-input:focus { outline:none; border-color:var(--accent-blue); }
  .pio-init-btn { padding:4px 12px; border:1px solid var(--accent-blue); border-radius:4px; background:transparent; color:var(--accent-blue); font-size:var(--fs-11); cursor:pointer; transition:all 0.15s ease; flex-shrink:0; }
  .pio-init-btn:hover:not(:disabled) { background:var(--accent-blue); color:#fff; }
  .pio-init-btn:disabled { opacity:0.5; cursor:default; }
  .pio-boards-list { max-height:280px; overflow-y:auto; }
  .pio-refresh-btn { background:none; border:none; color:var(--text-muted); padding:3px 5px; cursor:pointer; border-radius:4px; display:flex; align-items:center; margin-left:auto; flex-shrink:0; transition:all 0.12s ease; }
  .pio-refresh-btn:hover:not(:disabled) { color:var(--accent-blue); background:var(--bg-hover); }
  .pio-refresh-btn:disabled { opacity:0.4; cursor:default; }
  .pio-status-text { flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .pio-item:disabled { opacity:0.5; cursor:default; }
  .pio-item:disabled:hover { background:none; color:var(--text-primary); }
  .pio-section-title { display:flex; align-items:center; }
  .pio-output-title-bar { justify-content:space-between; }
  .pio-board-search-wrap { padding:0 12px 6px; }
  .pio-board-search { width:100%; background:var(--bg-surface); border:1px solid var(--border-subtle); border-radius:4px; padding:4px 8px; font-size:var(--fs-11); color:var(--text-primary); box-sizing:border-box; }
  .pio-board-search:focus { outline:none; border-color:var(--accent-blue); }
  .pio-boards-error { padding:6px 12px; font-size:var(--fs-10); color:var(--accent-red); }
  .pio-output-toggle { background:none; border:none; color:var(--text-muted); cursor:pointer; padding:0 4px; font-size:var(--fs-10); line-height:1; }
  .pio-output-toggle:hover { color:var(--text-primary); }
  .pio-output-panel { font-family:monospace; font-size:10px; color:var(--text-secondary); padding:8px 12px; background:var(--bg-primary); white-space:pre-wrap; word-break:break-word; max-height:180px; overflow-y:auto; border-top:1px solid var(--border-subtle); line-height:1.45; }

  /* AI Cost */
  .aicost-sidebar { display:flex; flex-direction:column; height:100%; overflow-y:auto; font-size:var(--font-size); }
  .aicost-script-bar { display:flex; gap:4px; padding:6px 8px; border-bottom:1px solid var(--border-subtle); }
  .aicost-script-btn { display:flex; align-items:center; gap:5px; flex:1; padding:4px 8px; border:1px solid var(--border-subtle); border-radius:5px; background:var(--bg-surface); color:var(--text-secondary); font-size:var(--fs-10); cursor:pointer; transition:all 0.15s ease; }
  .aicost-script-btn:hover { border-color:var(--accent-blue); color:var(--accent-blue); background:color-mix(in srgb, var(--accent-blue) 8%, transparent); }
  .aicost-summary { display:flex; gap:0; border-bottom:1px solid var(--border-subtle); }
  .aicost-summary-item { flex:1; display:flex; flex-direction:column; align-items:center; padding:8px 4px; gap:2px; border-right:1px solid var(--border-subtle); }
  .aicost-summary-item:last-child { border-right:none; }
  .aicost-summary-value { font-size:var(--fs-14); font-weight:700; color:var(--text-primary); font-family:monospace; }
  .aicost-summary-label { font-size:var(--fs-9); color:var(--text-muted); text-transform:uppercase; letter-spacing:0.5px; }
  .aicost-actions { display:flex; gap:4px; padding:6px 8px; border-bottom:1px solid var(--border-subtle); }
  .aicost-reset-btn, .aicost-refresh-btn { flex:1; padding:4px 8px; border:1px solid var(--border-subtle); border-radius:4px; background:transparent; color:var(--text-muted); font-size:var(--fs-10); cursor:pointer; transition:all 0.15s ease; }
  .aicost-reset-btn:hover { border-color:var(--accent-red); color:var(--accent-red); }
  .aicost-refresh-btn:hover { border-color:var(--accent-blue); color:var(--accent-blue); }
  .aicost-list { flex:1; overflow-y:auto; padding:4px 8px; }
  .aicost-loader { display:flex; align-items:center; justify-content:center; height:100px; }
  .aicost-card { padding:8px 10px; margin-bottom:6px; background:var(--bg-surface); border:1px solid var(--border-subtle); border-radius:8px; }
  .aicost-card-top { display:flex; justify-content:space-between; align-items:center; margin-bottom:6px; }
  .aicost-agent-name { font-weight:600; font-size:var(--font-size); color:var(--text-primary); }
  .aicost-agent-model { font-size:var(--fs-10); color:var(--text-muted); }
  .aicost-stat-row { display:flex; justify-content:space-between; align-items:center; padding:2px 0; font-size:var(--fs-11); color:var(--text-secondary); }
  .aicost-stat-total { border-top:1px solid var(--border-subtle); margin-top:4px; padding-top:5px; font-weight:600; color:var(--accent-blue); }
  .aicost-empty { display:flex; flex-direction:column; align-items:center; justify-content:center; height:100%; color:var(--text-muted); font-size:var(--font-size); text-align:center; gap:4px; }

  .workspace-area { flex:1; display:flex; flex-direction:column; overflow:hidden; min-width:200px; }
  .tab-panel { display: flex; flex-direction: column; width: 100%; height: 100%; }
  .tab-panel.hidden { display: none !important; }
  .placeholder { display:flex; flex-direction:column; align-items:center; justify-content:center; gap:12px; height:100%; color:var(--text-muted); font-size:var(--fs-13); }
  .placeholder strong { color:var(--accent-blue); }

  /* Preview */
  .preview-wrap { display:flex; flex-direction:column; height:100%; }
  .preview-toolbar { padding:6px 10px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .preview-input { width:100%; background:var(--bg-surface); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:5px; padding:5px 8px; font-size:var(--font-size); font-family:monospace; }
  .preview-input:focus { outline:none; border-color:var(--accent-blue); }
  .preview-frame { flex:1; border:none; background:#fff; }
  .preview-start-input { background:var(--bg-surface); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:5px; padding:6px 10px; font-size:var(--fs-13); font-family:monospace; width:280px; }
  .preview-start-input:focus { outline:none; border-color:var(--accent-blue); }

  /* ═══ STATUS BAR ═══ */
  .status-bar { display:flex; align-items:center; justify-content:space-between; height:var(--status-bar-height); padding:0 8px; background:var(--bg-secondary); border-top:1px solid var(--border-primary); font-size:var(--fs-11); color:var(--text-muted); flex-shrink:0; user-select:none; }
  .sb-left { display:flex; align-items:center; gap:4px; flex:1; min-width:0; overflow-x:auto; scrollbar-width:none; }
  .path-breadcrumb { display:flex; align-items:center; gap:0; }
  .path-seg { font-size:var(--fs-11); color:var(--text-muted); padding:1px 5px; border:1px solid var(--border-subtle); border-radius:3px; white-space:nowrap; background:var(--bg-surface); margin:0 1px; }
  .path-seg:hover { border-color:var(--accent-blue); color:var(--text-secondary); }
  .path-seg-last { color:var(--accent-blue); border-color:var(--accent-blue); background:color-mix(in srgb, var(--accent-blue) 8%, transparent); font-weight:500; }
  .path-sep { color:var(--text-muted); font-size:var(--fs-10); font-weight:700; margin:0 2px; }
  .sb-muted { color:var(--text-muted); font-style:italic; }
  .sb-center { display:flex; align-items:center; justify-content:center; gap:6px; cursor:pointer; padding:0 6px; border-radius:4px; transition:all 0.12s ease; }
  .sb-center:hover { background:var(--bg-hover); }
  .sb-clock { font-family:monospace; font-size:var(--fs-11); letter-spacing:0.5px; }
  .sb-log-badge { font-size:var(--fs-9); background:var(--bg-elevated); color:var(--text-muted); padding:0 5px; border-radius:8px; line-height:14px; font-weight:600; }
  .sb-log-badge.sb-log-has-error { background:var(--accent-red); color:#fff; }
  .sb-right { display:flex; align-items:center; gap:2px; }
  .sb-btn { display:flex; align-items:center; justify-content:center; background:none; border:none; color:var(--text-muted); padding:3px 6px; cursor:pointer; border-radius:4px; transition:all 0.12s ease; }
  .sb-btn:hover { color:var(--text-primary); background:var(--bg-hover); }
  .sb-btn.active { color:var(--accent-blue); }

  /* Floating panels */
  .float-panel { position:fixed; bottom:calc(var(--status-bar-height) + 8px); width:360px; height:440px; background:var(--bg-secondary); border:1px solid var(--border-primary); border-radius:12px; box-shadow:0 8px 32px rgba(0,0,0,0.5); display:flex; flex-direction:column; overflow:hidden; z-index:100; animation:floatUp 0.2s ease; }
  .float-ai { right:12px; }
  .float-notepad { right:12px; }
  .float-logs { right:calc(360px + 20px); width:380px; height:320px; }
  @keyframes floatUp { from { opacity:0; transform:translateY(16px) scale(0.96); } to { opacity:1; transform:translateY(0) scale(1); } }
  .float-header { display:flex; align-items:center; justify-content:space-between; padding:8px 12px; background:var(--bg-surface); border-bottom:1px solid var(--border-subtle); font-size:var(--font-size); font-weight:600; color:var(--text-primary); }
  .float-close { background:none; border:none; color:var(--text-muted); padding:2px 5px; cursor:pointer; border-radius:3px; font-size:var(--fs-16); line-height:1; }
  .float-close:hover { color:var(--accent-red); background:var(--bg-hover); }
  .float-btn { background:none; border:1px solid var(--border-subtle); color:var(--text-muted); padding:1px 8px; cursor:pointer; border-radius:4px; font-size:var(--fs-10); transition:all 0.12s ease; }
  .float-btn:hover { color:var(--text-primary); border-color:var(--text-muted); }
  .float-header-actions { display:flex; align-items:center; gap:4px; }
  .float-body { flex:1; overflow:hidden; }

  /* Logs */
  .logs-list { height:100%; overflow-y:auto; padding:4px 0; font-size:var(--fs-11); font-family:monospace; }
  .log-entry { display:flex; gap:8px; padding:3px 12px; border-bottom:1px solid var(--border-subtle); transition:background 0.1s ease; }
  .log-entry:hover { background:var(--bg-hover); }
  .log-entry.log-error { color:var(--accent-red); }
  .log-time { color:var(--text-muted); flex-shrink:0; }
  .log-msg { color:var(--text-secondary); word-break:break-word; }
  .log-empty { display:flex; align-items:center; justify-content:center; height:100%; color:var(--text-muted); font-size:var(--font-size); font-family:initial; }
</style>
