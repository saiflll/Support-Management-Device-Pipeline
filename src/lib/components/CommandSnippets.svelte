<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";

  let { compact = true } = $props();

  type Snippet = {
    id: string;
    title: string;
    command: string;
    language: string;
    created: string;
  };

  const SNIPPETS_FILE = "C:\\Users\\Lenovo\\AppData\\Local\\Temp\\contlib\\snippets.json";

  let snippets = $state<Snippet[]>([]);
  let editingId = $state<string | null>(null);
  let newTitle = $state("");
  let newCommand = $state("");
  let newLanguage = $state("powershell");
  let isEditing = $state(false);
  let selectedTerminalId = $state("term-1");
  let terminalSessions = $state<string[]>(["term-1", "term-2"]);

  const LANGUAGES = [
    { id: "powershell", label: "PowerShell", icon: "\u{1F4BB}" },
    { id: "bash", label: "Bash", icon: "\u{1F4BB}" },
    { id: "python", label: "Python", icon: "\u{1F40D}" },
    { id: "node", label: "Node.js", icon: "\u{1F4DC}" },
    { id: "cmd", label: "CMD", icon: "\u{1F4BB}" },
  ];

  function genId() { return Date.now().toString(36) + Math.random().toString(36).slice(2, 6); }

  async function loadSnippets() {
    try { snippets = JSON.parse(await invoke<string>("fs_read_file", { path: SNIPPETS_FILE })); }
    catch { snippets = []; }
  }

  async function saveSnippets() {
    try { await invoke("fs_write_file", { path: SNIPPETS_FILE, content: JSON.stringify(snippets, null, 2) }); }
    catch (e: any) { console.error("Failed to save snippets:", e); }
  }

  function startAdd() { isEditing = true; editingId = null; newTitle = ""; newCommand = ""; newLanguage = "powershell"; }
  function startEdit(s: Snippet) { isEditing = true; editingId = s.id; newTitle = s.title; newCommand = s.command; newLanguage = s.language; }
  function cancelEdit() { isEditing = false; editingId = null; newTitle = ""; newCommand = ""; }

  async function saveSnippet() {
    if (!newTitle.trim() || !newCommand.trim()) return;
    if (editingId) {
      snippets = snippets.map((s) => s.id === editingId ? { ...s, title: newTitle.trim(), command: newCommand.trim(), language: newLanguage } : s);
    } else {
      snippets = [...snippets, { id: genId(), title: newTitle.trim(), command: newCommand.trim(), language: newLanguage, created: new Date().toISOString() }];
    }
    await saveSnippets();
    cancelEdit();
  }

  async function deleteSnippet(id: string) {
    snippets = snippets.filter((s) => s.id !== id);
    await saveSnippets();
    if (editingId === id) cancelEdit();
  }

  async function runSnippet(snippet: Snippet, terminalId: string) {
    try {
      let cmd = snippet.command;
      if (snippet.language === "python") {
        const f = `C:\\Users\\Lenovo\\AppData\\Local\\Temp\\contlib\\snip_${snippet.id}.py`;
        await invoke("fs_write_file", { path: f, content: cmd });
        cmd = `python "${f}"`;
      } else if (snippet.language === "node") {
        const f = `C:\\Users\\Lenovo\\AppData\\Local\\Temp\\contlib\\snip_${snippet.id}.js`;
        await invoke("fs_write_file", { path: f, content: cmd });
        cmd = `node "${f}"`;
      }
      await invoke("pty_write", { sessionId: terminalId, data: cmd + "\n" });
    } catch (e: any) { console.error("Failed to run snippet:", e); }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") cancelEdit();
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter" && isEditing) { e.preventDefault(); saveSnippet(); }
  }

  $effect(() => { loadSnippets(); });
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="snip" class:snip-compact={compact}>
  <div class="snip-toolbar">
    <button class="snip-btn snip-btn-add" onclick={startAdd} disabled={isEditing}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      New
    </button>
    {#if snippets.length > 0}
      <span class="snip-count">{snippets.length}</span>
    {/if}
  </div>

  {#if isEditing}
    <div class="snip-form">
      <div class="snip-fields">
        <input bind:value={newTitle} placeholder="Snippet title..." class="snip-input" autofocus />
        <select bind:value={newLanguage} class="snip-select">
          {#each LANGUAGES as lang}
            <option value={lang.id}>{lang.icon} {lang.label}</option>
          {/each}
        </select>
      </div>
      <textarea bind:value={newCommand} placeholder="Paste your command here..." class="snip-textarea" rows={compact ? 2 : 4} spellcheck={false}></textarea>
      <div class="snip-form-actions">
        <span class="snip-hint">Ctrl+Enter to save</span>
        <div class="snip-btns">
          <button class="snip-btn snip-btn-cancel" onclick={cancelEdit}>Cancel</button>
          <button class="snip-btn snip-btn-save" onclick={saveSnippet} disabled={!newTitle.trim() || !newCommand.trim()}>
            {editingId ? "Update" : "Save"}
          </button>
        </div>
      </div>
    </div>
  {/if}

  <div class="snip-list">
    {#if snippets.length === 0 && !isEditing}
      <div class="snip-empty">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" opacity="0.25"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="15" x2="15" y2="15"/></svg>
        <p>No snippets yet</p>
        <button class="snip-btn snip-btn-add" onclick={startAdd}>+ Create one</button>
      </div>
    {:else}
      {#each snippets as s (s.id)}
        <div class="snip-card" class:snip-editing={editingId === s.id}>
          <div class="snip-card-header">
            <span class="snip-card-title">{s.title}</span>
            <span class="snip-card-lang">{LANGUAGES.find(l => l.id === s.language)?.icon || ""} {s.language}</span>
          </div>
          <pre class="snip-card-cmd">{s.command}</pre>
          <div class="snip-card-actions">
            <select class="snip-term-select" bind:value={selectedTerminalId}>
              {#each terminalSessions as tid}
                <option value={tid}>{tid}</option>
              {/each}
            </select>
            <button class="snip-action snip-action-run" onclick={() => runSnippet(s, selectedTerminalId)} title="Run in terminal">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
              Run
            </button>
            <button class="snip-action snip-action-edit" onclick={() => startEdit(s)} title="Edit">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            </button>
            <button class="snip-action snip-action-del" onclick={() => deleteSnippet(s.id)} title="Delete">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            </button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .snip { display:flex; flex-direction:column; height:100%; background:transparent; color:var(--text-primary); font-size:var(--font-size); overflow:hidden; }
  .snip-compact { font-size:var(--fs-11); }

  .snip-toolbar { display:flex; align-items:center; gap:8px; padding:6px 10px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .snip-count { background:var(--bg-elevated); color:var(--text-muted); font-size:var(--fs-10); padding:1px 6px; border-radius:8px; }

  .snip-btn { display:inline-flex; align-items:center; gap:4px; border:none; border-radius:4px; padding:4px 10px; font-size:var(--fs-11); font-weight:500; cursor:pointer; transition:all 0.12s ease; }
  .snip-btn-add { background:var(--accent-blue); color:var(--bg-primary); }
  .snip-btn-add:disabled { opacity:0.4; cursor:not-allowed; }
  .snip-btn-add:hover:not(:disabled) { filter:brightness(1.1); }
  .snip-btn-cancel { background:transparent; color:var(--text-muted); border:1px solid var(--border-subtle); }
  .snip-btn-cancel:hover { color:var(--text-primary); background:var(--bg-hover); }
  .snip-btn-save { background:var(--accent-green); color:var(--bg-primary); }
  .snip-btn-save:disabled { opacity:0.4; cursor:not-allowed; }
  .snip-btn-save:hover:not(:disabled) { filter:brightness(1.1); }

  .snip-form { padding:8px 10px; background:var(--bg-surface); border-bottom:1px solid var(--border-subtle); display:flex; flex-direction:column; gap:6px; flex-shrink:0; animation:slideDown 0.15s ease; }
  @keyframes slideDown { from { opacity:0; max-height:0; } to { opacity:1; max-height:200px; } }
  .snip-fields { display:flex; gap:6px; }
  .snip-input { flex:1; background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:4px; padding:6px 8px; font-size:var(--font-size); }
  .snip-input:focus { outline:none; border-color:var(--accent-blue); }
  .snip-select { background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:4px; padding:4px 6px; font-size:var(--fs-11); }
  .snip-textarea { background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:4px; padding:6px 8px; font-size:var(--font-size); font-family:monospace; resize:vertical; min-height:40px; }
  .snip-textarea:focus { outline:none; border-color:var(--accent-blue); }
  .snip-form-actions { display:flex; align-items:center; justify-content:space-between; }
  .snip-hint { color:var(--text-muted); font-size:var(--fs-10); }
  .snip-btns { display:flex; gap:4px; }

  .snip-list { flex:1; overflow-y:auto; padding:6px 8px; }

  .snip-empty { display:flex; flex-direction:column; align-items:center; justify-content:center; gap:8px; height:100%; color:var(--text-muted); font-size:var(--fs-11); }

  .snip-card { background:var(--bg-surface); border:1px solid var(--border-subtle); border-radius:8px; padding:8px 10px; margin-bottom:6px; transition:all 0.12s ease; }
  .snip-card:hover { border-color:var(--border-primary); }
  .snip-editing { border-color:var(--accent-blue) !important; box-shadow:0 0 0 1px var(--accent-blue); }
  .snip-card-header { display:flex; align-items:center; justify-content:space-between; margin-bottom:4px; }
  .snip-card-title { font-weight:600; font-size:var(--font-size); color:var(--text-primary); }
  .snip-card-lang { font-size:var(--fs-10); color:var(--text-muted); background:var(--bg-primary); padding:1px 6px; border-radius:3px; }
  .snip-card-cmd { font-family:monospace; font-size:var(--fs-11); color:var(--text-secondary); background:var(--bg-primary); padding:6px 8px; border-radius:5px; overflow-x:auto; white-space:pre-wrap; word-break:break-all; margin-bottom:6px; max-height:60px; overflow-y:auto; }
  .snip-card-actions { display:flex; align-items:center; gap:4px; }
  .snip-term-select { background:var(--bg-primary); color:var(--text-secondary); border:1px solid var(--border-subtle); border-radius:3px; padding:2px 4px; font-size:var(--fs-10); }
  .snip-action { display:inline-flex; align-items:center; gap:3px; border:none; border-radius:3px; padding:3px 8px; font-size:var(--fs-10); cursor:pointer; transition:all 0.12s ease; }
  .snip-action-run { background:var(--accent-green); color:var(--bg-primary); font-weight:600; }
  .snip-action-run:hover { filter:brightness(1.1); }
  .snip-action-edit { background:transparent; color:var(--accent-blue); border:1px solid var(--border-subtle); }
  .snip-action-edit:hover { border-color:var(--accent-blue); }
  .snip-action-del { background:transparent; color:var(--accent-red); border:1px solid var(--border-subtle); }
  .snip-action-del:hover { border-color:var(--accent-red); }
  .snip-list::-webkit-scrollbar { width:4px; }
  .snip-list::-webkit-scrollbar-thumb { background:var(--bg-hover); border-radius:2px; }
</style>
