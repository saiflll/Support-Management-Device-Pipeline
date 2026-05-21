<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";
  import type { FileEntry } from "../stores.svelte";

  let {
    onFileOpen = (_path: string) => {},
    onDirChange = (_path: string) => {},
    revealPath = "",
  } = $props();

  let currentPath = $state("C:\\Users\\Lenovo\\Documents\\dev\\contlib");
  let rootEntries = $state<FileEntry[]>([]);
  let expandedDirs = $state<Set<string>>(new Set());
  let childCache = $state<Map<string, FileEntry[]>>(new Map());
  let loadingDirs = $state<Set<string>>(new Set());
  let isLoading = $state(false);
  let selectedPath = $state<string | null>(null);
  let showNewFolder = $state(false);
  let showNewFile = $state(false);
  let newFolderName = $state("");
  let newFileName = $state("");
  let clipboardSource = $state<string | null>(null);
  let clipboardCut = $state(false);
  let renameTarget = $state<string | null>(null);
  let renameValue = $state("");

  function getTargetDir(): string {
    if (selectedPath) {
      // Find the entry in flatList — if dir, use it; if file, use parent
      for (const item of flatList) {
        if (item.entry.path === selectedPath && item.entry.is_dir) {
          return item.entry.path;
        }
      }
      // selectedPath is a file — use its parent
      const idx = selectedPath.lastIndexOf("\\");
      if (idx > 0) return selectedPath.slice(0, idx);
    }
    return currentPath;
  }

  type FlatEntry = { entry: FileEntry; depth: number; isLast: boolean; guides: boolean[] };

  let flatList = $state<FlatEntry[]>([]);

  $effect(() => {
    buildFlatList();
  });

  function buildFlatList() {
    const result: FlatEntry[] = [];
    function walk(list: FileEntry[], depth: number) {
      for (let i = 0; i < list.length; i++) {
        const e = list[i];
        const isExpanded = e.is_dir && expandedDirs.has(e.path);
        result.push({ entry: e, depth, isLast: i === list.length - 1, guides: [] });
        if (isExpanded) {
          const children = childCache.get(e.path);
          if (children) {
            walk(children, depth + 1);
          }
        }
      }
    }
    walk(rootEntries, 0);
    // Compute guides for each item
    for (let i = 0; i < result.length; i++) {
      const item = result[i];
      const guides = new Array(item.depth).fill(false);
      for (let d = 0; d < item.depth; d++) {
        // Look ahead for any item with depth <= d
        for (let j = i + 1; j < result.length; j++) {
          if (result[j].depth <= d) break;
          if (result[j].depth === d) { guides[d] = true; break; }
        }
      }
      result[i].guides = guides;
    }
    flatList = result;
  }

  const LINE_COLORS = ["var(--accent-blue)", "var(--accent-green)", "var(--accent-yellow)", "var(--accent-red)", "var(--accent-purple)", "var(--accent-cyan)", "var(--accent-orange)", "var(--accent-pink)"];

  async function loadDir(path: string): Promise<FileEntry[]> {
    try { return await invoke<FileEntry[]>("fs_list_dir", { path }); }
    catch { return []; }
  }

  async function loadRoot() {
    isLoading = true;
    rootEntries = await loadDir(currentPath);
    isLoading = false;
  }

  async function toggleDir(dirPath: string) {
    if (expandedDirs.has(dirPath)) {
      const s = new Set(expandedDirs);
      s.delete(dirPath);
      expandedDirs = s;
      return;
    }
    loadingDirs = new Set(loadingDirs).add(dirPath);
    const children = await loadDir(dirPath);
    childCache.set(dirPath, children);
    loadingDirs.delete(dirPath);
    loadingDirs = new Set(loadingDirs);
    const s = new Set(expandedDirs);
    s.add(dirPath);
    expandedDirs = s;
  }

  function openEntry(entry: FileEntry, depth: number) {
    selectedPath = entry.path;
    if (entry.is_dir) {
      toggleDir(entry.path);
    } else {
      onFileOpen(entry.path);
    }
  }

  async function createFolder() {
    const name = newFolderName.trim();
    if (!name) return;
    const target = getTargetDir();
    const dir = target + "\\" + name;
    try {
      await invoke("fs_create_dir", { path: dir });
      newFolderName = "";
      showNewFolder = false;
      if (expandedDirs.has(target)) {
        const children = await loadDir(target);
        childCache.set(target, children);
        childCache = new Map(childCache);
      }
      await loadRoot();
    } catch (e) {
      console.error("Failed to create folder:", e);
    }
  }

  async function createFile() {
    const name = newFileName.trim();
    if (!name) return;
    const target = getTargetDir();
    const filePath = target + "\\" + name;
    try {
      await invoke("fs_write_file", { path: filePath, content: "" });
      newFileName = "";
      showNewFile = false;
      if (expandedDirs.has(target)) {
        const children = await loadDir(target);
        childCache.set(target, children);
        childCache = new Map(childCache);
      }
      await loadRoot();
    } catch (e) {
      console.error("Failed to create file:", e);
    }
  }

  function handleFmKeydown(e: KeyboardEvent) {
    const isCtrl = e.ctrlKey || e.metaKey;
    if (isCtrl && e.key === "c" && selectedPath) {
      e.preventDefault();
      clipboardSource = selectedPath;
      clipboardCut = false;
      return;
    }
    if (isCtrl && e.key === "x" && selectedPath) {
      e.preventDefault();
      clipboardSource = selectedPath;
      clipboardCut = true;
      return;
    }
    if (isCtrl && e.key === "v" && clipboardSource) {
      e.preventDefault();
      pasteItem();
      return;
    }
    if (isCtrl && e.key === "a") {
      e.preventDefault();
      // Focus first item
      if (flatList.length > 0) {
        selectedPath = flatList[0].entry.path;
      }
      return;
    }
    if (e.key === "Delete" && selectedPath) {
      e.preventDefault();
      deleteItem();
      return;
    }
    if (e.key === "F2" && selectedPath) {
      e.preventDefault();
      startRename();
      return;
    }
  }

  async function deleteItem() {
    if (!selectedPath) return;
    const entry = flatList.find(x => x.entry.path === selectedPath);
    const label = entry ? entry.entry.name : selectedPath.split("\\").pop();
    if (!confirm(`Delete "${label}"?`)) return;
    try {
      await invoke("fs_delete", { path: selectedPath });
      selectedPath = null;
      childCache.clear();
      await loadRoot();
    } catch (e) {
      console.error("Delete failed:", e);
    }
  }

  function startRename() {
    if (!selectedPath) return;
    const name = selectedPath.split("\\").pop() || "";
    renameTarget = selectedPath;
    renameValue = name;
  }

  async function commitRename() {
    if (!renameTarget || !renameValue.trim()) { renameTarget = null; return; }
    const newName = renameValue.trim();
    const parent = renameTarget.slice(0, renameTarget.lastIndexOf("\\"));
    const newPath = parent + "\\" + newName;
    if (newPath === renameTarget) { renameTarget = null; return; }
    try {
      await invoke("fs_rename", { from: renameTarget, to: newPath });
      renameTarget = null;
      childCache.clear();
      await loadRoot();
    } catch (e) {
      console.error("Rename failed:", e);
    }
  }

  async function pasteItem() {
    if (!clipboardSource) return;
    const target = getTargetDir();
    const name = clipboardSource.split("\\").pop() || "item";
    const dest = target + "\\" + name;

    if (clipboardSource === dest) {
      clipboardSource = null;
      return;
    }

    try {
      if (clipboardCut) {
        // Move (rename)
        await invoke("fs_rename", { from: clipboardSource, to: dest });
      } else {
        // Copy: read then write
        const content = await invoke<string>("fs_read_file", { path: clipboardSource });
        await invoke("fs_write_file", { path: dest, content });
      }
      clipboardSource = null;
      clipboardCut = false;
      if (expandedDirs.has(target)) {
        const children = await loadDir(target);
        childCache.set(target, children);
        childCache = new Map(childCache);
      }
      await loadRoot();
    } catch (e) {
      console.error("Paste failed:", e);
    }
  }

  const FILE_COLORS: Record<string, string> = {
    ts: "#3178c6", js: "#f7df1e", rs: "#ff8243", py: "#3572A5",
    svelte: "#ff3e00", html: "#e34f26", css: "#38bdf8", json: "#10b981",
    md: "#a855f7", yml: "#fb7185", yaml: "#fb7185", toml: "#d946ef",
    sh: "#4ade80", bash: "#4ade80", ps1: "#38bdf8",
    png: "#fbbf24", jpg: "#fbbf24", jpeg: "#fbbf24",
    svg: "#fbbf24", gif: "#fbbf24",
  };

  function fileColor(name: string): string {
    const ext = name.split(".").pop()?.toLowerCase() || "";
    return FILE_COLORS[ext] || "var(--text-muted)";
  }

  function folderColor(name: string): string {
    const n = name.toLowerCase();
    if (n === ".git") return "#f05032";
    if (n === ".github") return "#a855f7";
    if (n === "node_modules") return "#e11d48";
    if (n === "src" || n === "lib" || n === "components") return "#38bdf8";
    if (n === "static" || n === "public" || n === "assets" || n === "icons") return "#fbbf24";
    if (n === "build" || n === "dist" || n === ".svelte-kit" || n === ".notepad_temp") return "#c084fc";
    return "var(--accent-blue)";
  }

  function iconFor(name: string, isDir: boolean, expanded: boolean): string {
    if (isDir) {
      const c = folderColor(name);
      const n = name.toLowerCase();
      
      if (n === ".git") {
        return expanded
          ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v1" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M3 15l2-6h16l-2 6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.25"/><circle cx="12" cy="12" r="2.5" fill="#f05032"/><line x1="12" y1="9.5" x2="12" y2="12" stroke="#fff" stroke-width="1"/></svg>`
          : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.15"/><circle cx="13" cy="13" r="3" stroke="${c}" stroke-width="1.2" fill="${c}" fill-opacity="0.2"/><line x1="13" y1="10" x2="13" y2="13" stroke="${c}" stroke-width="1.2"/><circle cx="10" cy="13" r="1.5" fill="${c}"/></svg>`;
      }
      if (n === "node_modules") {
        return expanded
          ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v1" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M3 15l2-6h16l-2 6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.25"/><rect x="10" y="10" width="4" height="4" fill="#e11d48" rx="0.5"/></svg>`
          : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.15"/><rect x="11" y="11" width="5" height="5" stroke="${c}" stroke-width="1.2" fill="${c}" fill-opacity="0.3" rx="1"/></svg>`;
      }
      if (n === "src" || n === "lib") {
        return expanded
          ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v1" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M3 15l2-6h16l-2 6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.25"/><path d="M10 11l-2 2 2 2M14 11l2-2-2 2" stroke="${c}" stroke-width="1.2" stroke-linecap="round"/></svg>`
          : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.15"/><path d="M11 11.5l-2 1.5 2 1.5M14 11.5l2 1.5-2 1.5" stroke="${c}" stroke-width="1" stroke-linecap="round"/></svg>`;
      }
      if (n === "components") {
        return expanded
          ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v1" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M3 15l2-6h16l-2 6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.25"/><polygon points="12 10 15 12 12 14 9 12" fill="${c}" fill-opacity="0.6"/></svg>`
          : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.15"/><polygon points="13 11 16 13 13 15 10 13" stroke="${c}" stroke-width="1" fill="${c}" fill-opacity="0.3"/></svg>`;
      }
      if (n === ".github") {
        return expanded
          ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v1" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M3 15l2-6h16l-2 6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.25"/><circle cx="12" cy="12" r="2.5" fill="${c}" fill-opacity="0.8"/></svg>`
          : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.15"/><circle cx="13" cy="13" r="2.5" stroke="${c}" stroke-width="1" fill="${c}" fill-opacity="0.4"/></svg>`;
      }

      return expanded
        ? `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M5 19a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v1" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M3 15l2-6h16l-2 6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.3"/></svg>`
        : `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.2"/></svg>`;
    }

    const c = fileColor(name);
    const ext = name.split(".").pop()?.toLowerCase() || "";
    
    if (ext === "svelte") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z" fill="${c}" fill-opacity="0.1" stroke="${c}" stroke-width="1.5"/><path d="M12 6c-3.31 0-6 2.69-6 6s2.69 6 6 6 6-2.69 6-6-2.69-6-6-6zm0 10c-2.21 0-4-1.79-4-4s1.79-4 4-4 4 1.79 4 4-1.79 4-4 4z" fill="${c}" fill-opacity="0.6"/></svg>`;
    }
    if (ext === "rs") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="7" stroke="${c}" stroke-width="1.5" stroke-dasharray="3 1.5"/><circle cx="12" cy="12" r="4.5" fill="${c}" fill-opacity="0.4" stroke="${c}" stroke-width="1"/></svg>`;
    }
    if (ext === "json" || ext === "lock") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="${c}" stroke-width="1.5"><path d="M8 4H6a2 2 0 0 0-2 2v3a2 2 0 0 1-2 2 2 2 0 0 1 2 2v3a2 2 0 0 0 2 2h2M16 4h2a2 2 0 0 1 2 2v3a2 2 0 0 0 2 2 2 2 0 0 0-2 2v3a2 2 0 0 1-2 2h-2" stroke-linecap="round"/></svg>`;
    }
    if (ext === "js") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><rect x="3" y="3" width="18" height="18" rx="3" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><text x="7" y="16" fill="${c}" font-family="monospace" font-weight="900" font-size="10px">JS</text></svg>`;
    }
    if (ext === "ts") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><rect x="3" y="3" width="18" height="18" rx="3" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><text x="7" y="16" fill="${c}" font-family="monospace" font-weight="900" font-size="10px">TS</text></svg>`;
    }
    if (ext === "md") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><rect x="3" y="5" width="18" height="14" rx="2" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M7 15V9l3 3 3-3v6" stroke="${c}" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M16 11l2-2m0 0l2 2m-2-2v5" stroke="${c}" stroke-width="1.5" stroke-linecap="round"/></svg>`;
    }
    if (ext === "css") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><rect x="3" y="3" width="18" height="18" rx="3" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.1"/><path d="M8 8h8M8 12h5" stroke="${c}" stroke-width="1.5" stroke-linecap="round"/></svg>`;
    }
    if (ext === "html") {
      return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="${c}" stroke-width="1.5"><path d="M8 9l-3 3 3 3M16 9l3 3-3 3M11.5 7l1 10" stroke-linecap="round"/></svg>`;
    }

    return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none"><rect x="2" y="4" width="20" height="16" rx="2" stroke="${c}" stroke-width="1.5" fill="${c}" fill-opacity="0.05"/><path d="M9 12l3 2 3-2" stroke="${c}" stroke-width="1.5" stroke-linecap="round"/></svg>`;
  }

  $effect(() => { loadRoot(); });

  let lastRevealed = $state("");
  $effect(() => {
    if (!revealPath || revealPath === lastRevealed) return;
    if (!revealPath.startsWith(currentPath)) return;
    lastRevealed = revealPath;
    const rel = revealPath.slice(currentPath.length).replace(/^\\+/, "");
    if (!rel) return;
    const parts = rel.split("\\");
    const s = new Set(expandedDirs);
    let acc = currentPath;
    for (const p of parts) {
      if (!p) continue;
      acc = acc + "\\" + p;
      s.add(acc);
      if (!childCache.has(acc)) {
        loadDir(acc).then((children) => {
          childCache.set(acc, children);
          childCache = new Map(childCache);
        });
      }
    }
    expandedDirs = s;
  });
</script>

<div class="fm" onkeydown={handleFmKeydown} tabindex="-1">
  <div class="fm-header">
    <span class="fm-title">EXPLORER</span>
    <div class="fm-header-actions">
      <span class="fm-root">{currentPath.split("\\").pop()}</span>
      <button class="fm-btn" onclick={() => { showNewFolder = !showNewFolder; showNewFile = false; }} title="New Folder">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
      </button>
      <button class="fm-btn" onclick={() => { showNewFile = !showNewFile; showNewFolder = false; }} title="New File">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><line x1="12" y1="9" x2="12" y2="15"/><line x1="9" y1="12" x2="15" y2="12"/></svg>
      </button>
      <button class="fm-btn" onclick={() => { childCache.clear(); loadRoot(); }} title="Refresh">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
      </button>
    </div>
  </div>
  {#if showNewFolder}
    <div class="fm-new-row">
      <input class="fm-new-input" bind:value={newFolderName} placeholder="folder name..." autofocus
        onkeydown={(e) => { if (e.key === "Enter") createFolder(); if (e.key === "Escape") { showNewFolder = false; newFolderName = ""; }}}
        onblur={() => { if (!newFolderName.trim()) { showNewFolder = false; }}}
      />
      <button class="fm-new-btn" onclick={createFolder}>+</button>
    </div>
  {/if}
  {#if showNewFile}
    <div class="fm-new-row">
      <input class="fm-new-input" bind:value={newFileName} placeholder="file.txt..." autofocus
        onkeydown={(e) => { if (e.key === "Enter") createFile(); if (e.key === "Escape") { showNewFile = false; newFileName = ""; }}}
        onblur={() => { if (!newFileName.trim()) { showNewFile = false; }}}
      />
      <button class="fm-new-btn" onclick={createFile}>+</button>
    </div>
  {/if}

  <div class="fm-body">
    {#if isLoading}
      <div class="fm-loader"><div class="spinner"></div></div>
    {:else if rootEntries.length === 0}
      <div class="fm-empty">Empty directory</div>
    {:else}
      {#each flatList as item (item.entry.path)}
        {@const expanded = item.entry.is_dir && expandedDirs.has(item.entry.path)}
        {@const loading = item.entry.is_dir && loadingDirs.has(item.entry.path)}
        <div
          class="fm-row"
          class:selected={selectedPath === item.entry.path}
          class:dir={item.entry.is_dir}
          style="padding-left: {8 + item.depth * 10}px"
          tabindex="0"
          role="treeitem"
          aria-expanded={item.entry.is_dir ? expanded : undefined}
          onclick={() => openEntry(item.entry, item.depth)}
        >
          {#each item.guides as hasGuide, di}
            <span class="guide" style="opacity:{hasGuide ? 1 : 0.25}">
              <svg width="10" height="100%" viewBox="0 0 10 28" preserveAspectRatio="none" style="color:{LINE_COLORS[di % LINE_COLORS.length]}">
                {#if hasGuide}
                  <line x1="5" y1="0" x2="5" y2="28" stroke="currentColor" stroke-width="1.5"/>
                {/if}
                <circle cx="5" cy="50%" r="2" fill="currentColor"/>
              </svg>
            </span>
          {/each}
          {#if item.entry.is_dir}
            <span class="row-chevron" class:open={expanded}>
              {#if loading}
                <div class="row-spinner"></div>
              {:else}
                <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor"><polygon points="6 3 20 12 6 21 6 3"/></svg>
              {/if}
            </span>
          {:else}
            <span class="row-chevron row-chevron-dummy"></span>
          {/if}
          <span class="row-icon">{@html iconFor(item.entry.name, item.entry.is_dir, expanded)}</span>
          {#if renameTarget === item.entry.path}
            <input class="row-rename-input" bind:value={renameValue}
              autofocus
              onblur={commitRename}
              onkeydown={(e) => { if (e.key === "Enter") commitRename(); if (e.key === "Escape") { renameTarget = null; }}}
            />
          {:else}
            <span class="row-name">{item.entry.name}</span>
          {/if}
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .fm { display:flex; flex-direction:column; height:100%; background:transparent; font-size:var(--font-size); user-select:none; }
  .fm-header { display:flex; align-items:center; justify-content:space-between; padding:6px 10px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .fm-title { font-size:var(--fs-10); font-weight:600; color:var(--text-muted); letter-spacing:0.8px; text-transform:uppercase; }
  .fm-header-actions { display:flex; align-items:center; gap:6px; }
  .fm-root { font-size:var(--fs-10); color:var(--text-muted); }
  .fm-btn { background:none; border:none; color:var(--text-muted); padding:2px; cursor:pointer; border-radius:3px; display:flex; }
  .fm-btn:hover { color:var(--text-primary); background:var(--bg-hover); }

  .fm-new-row { display:flex; align-items:center; gap:4px; padding:4px 10px; border-bottom:1px solid var(--border-subtle); }
  .fm-new-input { flex:1; background:var(--bg-surface); border:1px solid var(--accent-blue); border-radius:4px; padding:3px 6px; font-size:var(--fs-11); color:var(--text-primary); font-family:monospace; min-width:0; }
  .fm-new-input:focus { outline:none; }
  .fm-new-btn { background:var(--accent-blue); border:none; color:#fff; border-radius:3px; width:20px; height:20px; font-size:var(--fs-14); line-height:1; cursor:pointer; display:flex; align-items:center; justify-content:center; font-weight:700; }
  .fm-new-btn:hover { filter:brightness(1.2); }

  .fm-body { flex:1; overflow-y:auto; padding:2px 0; }
  .fm-loader { display:flex; align-items:center; justify-content:center; height:100%; }
  .fm-empty { color:var(--text-muted); font-size:var(--fs-11); padding:20px; text-align:center; }

  .fm-row {
    display:flex; align-items:center; gap:2px; padding:2px 6px; cursor:pointer;
    border-radius:3px; transition:all 0.1s ease; border:1px solid transparent;
    margin:0 4px; position:relative; font-size:var(--font-size);
  }
  .fm-row:hover { background:var(--bg-hover); }
  .fm-row.selected { background:var(--bg-surface); border-color:var(--border-primary); }
  .fm-row:focus { outline:none; border-color:var(--accent-blue); }

  .guide { display:flex; align-items:center; justify-content:center; width:10px; height:16px; flex-shrink:0; pointer-events:none; }
  .guide svg { width:10px; height:100%; min-height:16px; }
  .row-chevron { display:flex; align-items:center; justify-content:center; width:14px; height:14px; color:var(--text-muted); flex-shrink:0; transition:transform 0.12s ease; }
  .row-chevron.open { transform:rotate(90deg); }
  .row-chevron-dummy { visibility:hidden; }
  .row-spinner { width:10px; height:10px; border:1.5px solid var(--bg-hover); border-top-color:var(--accent-blue); border-radius:50%; animation:spin 0.5s linear infinite; }

  .row-icon { font-size:var(--fs-13); width:18px; text-align:center; flex-shrink:0; line-height:1; }
  .row-name { font-size:var(--font-size); color:var(--text-primary); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .row-rename-input { flex:1; min-width:0; background:var(--bg-surface); border:1px solid var(--accent-blue); border-radius:3px; padding:1px 4px; font-size:var(--font-size); color:var(--text-primary); font-family:monospace; }
  .row-rename-input:focus { outline:none; }

  @keyframes spin { to { transform:rotate(360deg); } }

  .fm-body::-webkit-scrollbar { width:4px; }
  .fm-body::-webkit-scrollbar-thumb { background:var(--bg-hover); border-radius:2px; }
</style>
