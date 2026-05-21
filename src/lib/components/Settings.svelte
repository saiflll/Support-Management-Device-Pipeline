<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";
  import type { Agent } from "../stores.svelte";
  // ─── Agent state (JSON-based editor) ──────────
  let agents = $state<Agent[]>([]);
  let agentConfigJson = $state("");
  let saveStatus = $state("");
  let validating = $state(false);
  let validationResults = $state<Record<string, { valid: boolean; error?: string }>>({});

  function getDefaultAgentsJson(): string {
    return JSON.stringify([
      {
        id: "coder",
        name: "Coder",
        provider: "openai",
        model: "gpt-4o",
        api_key: null,
        base_url: null,
        temperature: 0.2,
        system_prompt: "You are an expert software engineer. Write clean, efficient code.",
        capabilities: ["code", "debug", "refactor"]
      },
      {
        id: "shell",
        name: "Shell",
        provider: "openai",
        model: "gpt-4o",
        api_key: null,
        base_url: null,
        temperature: 0.1,
        system_prompt: "You are a shell expert. Provide precise terminal commands.",
        capabilities: ["shell", "terminal", "automation"]
      },
      {
        id: "architect",
        name: "Architect",
        provider: "anthropic",
        model: "claude-sonnet-4-20250514",
        api_key: null,
        base_url: null,
        temperature: 0.4,
        system_prompt: "You are a software architect. Design robust, scalable systems.",
        capabilities: ["design", "planning", "architecture"]
      }
    ], null, 2);
  }

  async function loadAgents() {
    try {
      agents = await invoke<Agent[]>("ai_list_agents");
      agentConfigJson = JSON.stringify(agents, null, 2);
    } catch (e) {
      console.error("Failed to load agents:", e);
    }
  }

  function parseAgentsFromJson(json: string): Agent[] {
    const parsed = JSON.parse(json);
    if (!Array.isArray(parsed)) throw new Error("Root must be an array of agents");
    for (const a of parsed) {
      if (!a.id || !a.name || !a.provider || !a.model) {
        throw new Error(`Agent missing required field (id, name, provider, model): ${JSON.stringify(a)}`);
      }
    }
    return parsed;
  }

  async function saveAgents() {
    saveStatus = "";
    validationResults = {};
    try {
      const parsedAgents = parseAgentsFromJson(agentConfigJson);
      for (const agent of parsedAgents) {
        await invoke("ai_update_agent", { config: agent });
      }
      await loadAgents();
      saveStatus = `${parsedAgents.length} agents saved.`;
    } catch (e) {
      saveStatus = `Save failed: ${e}`;
    }
  }

  async function validateAgents() {
    validating = true;
    validationResults = {};
    try {
      const parsedAgents = parseAgentsFromJson(agentConfigJson);
      for (const agent of parsedAgents) {
        try {
          const res = await invoke<{ agent_id: string; valid: boolean; error: string | null }>(
            "ai_validate_agent", { agentId: agent.id }
          );
          validationResults[agent.id] = { valid: res.valid, error: res.error || undefined };
        } catch (e) {
          validationResults[agent.id] = { valid: false, error: String(e) };
        }
      }
    } catch (e) {
      saveStatus = `Validation error: ${e}`;
    }
    validating = false;
  }

  function agentsWithKeys(): number { return agents.filter(a => a.api_key).length; }
  function validatedCount(): number { return Object.values(validationResults).filter(v => v.valid).length; }
  // ─── Shared Theme Layer ──────────────────────
  import { THEMES, FONTS, getStoredTheme, getStoredFont, getStoredFontSize, applyTheme as sharedApplyTheme, applyFont as sharedApplyFont, applyFontSize as sharedApplyFontSize } from "$lib/themes";

  let settingsTab = $state<"agent" | "appearance" | "setup" | "shortcuts" | "about">("appearance");

  let currentTheme = $state(getStoredTheme());
  let currentFont = $state(getStoredFont());
  let currentFontSize = $state(getStoredFontSize());

  function applyTheme(themeId: string) {
    currentTheme = themeId;
    sharedApplyTheme(themeId);
  }

  function applyFont(fontValue: string) {
    currentFont = fontValue;
    sharedApplyFont(fontValue);
  }

  function applyFontSize(val: number) {
    currentFontSize = val;
    sharedApplyFontSize(val);
  }

  import { activeTerminalSessionId, editorLanguages, saveEditorLangs, SUPPORTED_LANGS } from "../stores.svelte";

  let activeTermId = $state<string | null>(null);
  $effect(() => {
    const unsub = activeTerminalSessionId.subscribe(val => {
      activeTermId = val;
    });
    return unsub;
  });

  let enabledLangs = $state<Record<string, boolean>>({});
  $effect(() => {
    const unsub = editorLanguages.subscribe(val => {
      enabledLangs = val;
    });
    return unsub;
  });

  function enableLang(langId: string) {
    const updated = { ...enabledLangs, [langId]: true };
    saveEditorLangs(updated);
  }

  function disableLang(langId: string) {
    const updated = { ...enabledLangs, [langId]: false };
    saveEditorLangs(updated);
  }

  // ─── Auto Suspend ───────────────────────────
  let autoSuspend = $state(
    (() => { try { return localStorage.getItem("contlib-auto-suspend") === "true"; } catch { return false; } })()
  );
  function toggleAuto() {
    autoSuspend = !autoSuspend;
    try { localStorage.setItem("contlib-auto-suspend", String(autoSuspend)); } catch {}
  }

  // ─── Language Autocomplete ────────────────
  let langSearch = $state("");
  let showSuggestions = $state(false);

  const searchResults = $derived(
    langSearch.trim()
      ? SUPPORTED_LANGS.filter(l =>
          l.name.toLowerCase().includes(langSearch.toLowerCase()) &&
          enabledLangs[l.id] !== true
        ).slice(0, 6)
      : []
  );

  function selectLang(langId: string) {
    enableLang(langId);
    langSearch = "";
    showSuggestions = false;
  }

  const activeLangList = $derived(
    SUPPORTED_LANGS.filter(l => enabledLangs[l.id] === true)
  );

  // ─── Install Command Notes ─────────────────
  type InstallNote = { name: string; cmd: string };
  const INSTALL_NOTES: InstallNote[] = [
    { name: "Python", cmd: "winget install Python.Python.3" },
    { name: "Node.js", cmd: "winget install OpenJS.NodeJS" },
    { name: "Rust", cmd: "winget install Rustlang.Rustup" },
    { name: "Go", cmd: "winget install GoLang.Go" },
    { name: "Java", cmd: "winget install Microsoft.OpenJDK.17" },
    { name: "PlatformIO", cmd: "pip install platformio" },
    { name: "Flutter", cmd: "winget install Flutter.Flutter" },
  ];
  let copiedNote = $state<string | null>(null);
  async function copyCmd(cmd: string) {
    try {
      await navigator.clipboard.writeText(cmd);
      copiedNote = cmd;
      setTimeout(() => { if (copiedNote === cmd) copiedNote = null; }, 1500);
    } catch {}
  }

  // ─── Keyboard Shortcuts State ─────────────────
  type ShortcutKey = {
    ctrl: boolean;
    alt: boolean;
    shift: boolean;
    key: string;
  };

  type ShortcutBinding = {
    id: string;
    name: string;
    description: string;
    binding: ShortcutKey;
  };

  const DEFAULT_SHORTCUTS: ShortcutBinding[] = [
    { id: "sidebar", name: "Toggle Sidebar Explorer", description: "Toggle left explorer sidebar panel visibility", binding: { ctrl: true, alt: false, shift: false, key: "b" } },
    { id: "terminal", name: "Toggle Active Terminal", description: "Activate or focus the main terminal panel", binding: { ctrl: true, alt: false, shift: false, key: "j" } },
    { id: "ai", name: "Toggle Floating AI Chat", description: "Toggle the bottom-right AI Assistant chat box", binding: { ctrl: true, alt: true, shift: false, key: "a" } },
    { id: "notepad", name: "Toggle Floating Notepad", description: "Toggle the bottom-right notebook panel", binding: { ctrl: true, alt: true, shift: false, key: "n" } },
  ];

  let shortcuts = $state<ShortcutBinding[]>(loadShortcuts());
  let shortcutSearch = $state("");
  let editingShortcutId = $state<string | null>(null);
  let editCtrl = $state(false);
  let editAlt = $state(false);
  let editShift = $state(false);
  let editKey = $state("");

  function loadShortcuts(): ShortcutBinding[] {
    try {
      const stored = localStorage.getItem("contlib-shortcuts");
      if (stored) {
        const parsed = JSON.parse(stored);
        return DEFAULT_SHORTCUTS.map(item => {
          if (parsed[item.id]) {
            return { ...item, binding: parsed[item.id] };
          }
          return item;
        });
      }
    } catch {}
    return DEFAULT_SHORTCUTS;
  }

  function saveShortcuts() {
    const map: Record<string, ShortcutKey> = {};
    for (const s of shortcuts) {
      map[s.id] = s.binding;
    }
    try {
      localStorage.setItem("contlib-shortcuts", JSON.stringify(map));
    } catch {}
  }

  function startEditShortcut(item: ShortcutBinding) {
    editingShortcutId = item.id;
    editCtrl = item.binding.ctrl;
    editAlt = item.binding.alt;
    editShift = item.binding.shift;
    editKey = item.binding.key;
  }

  function cancelEditShortcut() {
    editingShortcutId = null;
  }

  function saveEditedShortcut() {
    if (!editKey.trim()) return;
    const item = shortcuts.find(s => s.id === editingShortcutId);
    if (item) {
      item.binding = {
        ctrl: editCtrl,
        alt: editAlt,
        shift: editShift,
        key: editKey.trim().toLowerCase(),
      };
      saveShortcuts();
    }
    editingShortcutId = null;
  }

  function formatBinding(binding: ShortcutKey): string {
    const parts = [];
    if (binding.ctrl) parts.push("Ctrl");
    if (binding.alt) parts.push("Alt");
    if (binding.shift) parts.push("Shift");
    parts.push(binding.key.toUpperCase());
    return parts.join(" + ");
  }

  const filteredShortcuts = $derived(
    shortcuts.filter(s =>
      s.name.toLowerCase().includes(shortcutSearch.toLowerCase()) ||
      s.description.toLowerCase().includes(shortcutSearch.toLowerCase())
    )
  );

  $effect(() => {
    loadAgents();
    applyTheme(currentTheme);
    applyFont(currentFont);
  });

</script>

<div class="settings">
  <!-- ═══ Sub-tabs ═══ -->
  <div class="settings-tabs">
    <button class="settings-tab" class:active={settingsTab === "appearance"} onclick={() => (settingsTab = "appearance")}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
      Appearance
    </button>
    <button class="settings-tab" class:active={settingsTab === "agent"} onclick={() => (settingsTab = "agent")}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="8" r="4"/><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/></svg>
      Agents
    </button>
    <button class="settings-tab" class:active={settingsTab === "setup"} onclick={() => (settingsTab = "setup")}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
      Setup
    </button>
    <button class="settings-tab" class:active={settingsTab === "shortcuts"} onclick={() => (settingsTab = "shortcuts")}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="4" width="20" height="16" rx="2" ry="2"/><path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01M6 12h.01M10 12h.01M14 12h.01M18 12h.01M7 16h10"/></svg>
      Shortcuts
    </button>
    <button class="settings-tab" class:active={settingsTab === "about"} onclick={() => (settingsTab = "about")}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
      About
    </button>
  </div>

  <!-- ═══ Appearance Tab ═══ -->
  {#if settingsTab === "appearance"}
    <div class="settings-header">
      <span class="settings-title">Appearance</span>
    </div>
    <div class="appearance">
      <div class="section-title">Theme</div>
      <div class="theme-grid">
        {#each Object.entries(THEMES) as [id, theme]}
          <button class="theme-card" class:theme-active={currentTheme === id} onclick={() => applyTheme(id)}>
            <div class="theme-preview">
              <div class="tp-bg" style="background:{theme.vars['--bg-primary']}">
                <div class="tp-side" style="background:{theme.vars['--bg-secondary']};border-right-color:{theme.vars['--border-primary']}"></div>
                <div class="tp-main">
                  <div class="tp-line" style="background:{theme.vars['--text-primary']};width:55%"></div>
                  <div class="tp-line" style="background:{theme.vars['--text-secondary']};width:35%"></div>
                  <div class="tp-line" style="background:{theme.vars['--accent-blue']};width:45%"></div>
                  <div class="tp-line" style="background:{theme.vars['--text-muted']};width:25%"></div>
                </div>
              </div>
            </div>
            <span class="theme-name">{theme.name}</span>
            {#if currentTheme === id}<span class="theme-check">&#x2713;</span>{/if}
          </button>
        {/each}
      </div>

      <div class="section-title">Font</div>
      <div class="font-section">
        <select class="font-select" onchange={(e) => applyFont((e.target as HTMLSelectElement).value)}>
          {#each FONTS as f}
            <option value={f.value} selected={currentFont === f.value}>{f.label}</option>
          {/each}
        </select>
        <div class="font-preview" style="font-family:{currentFont}">
          → const hello = "contlib"; ←
        </div>
      </div>

      <div class="section-title">Font Size</div>
      <div class="fontsize-section">
        <input type="range" min="9" max="24" value={currentFontSize}
          oninput={(e) => applyFontSize(parseInt((e.target as HTMLInputElement).value))}
          class="fontsize-slider"
        />
        <span class="fontsize-label">{currentFontSize}px</span>
      </div>
    </div>

  <!-- ═══ Agent Tab (JSON-based) ═══ -->
  {:else if settingsTab === "agent"}
    <div class="agent-section">
      <div class="settings-header">
        <span class="settings-title">AI Agents</span>
        <div class="agent-header-actions">
          <button class="settings-btn settings-btn-cancel" onclick={() => { agentConfigJson = getDefaultAgentsJson(); }}>Reset to Default</button>
          <button class="settings-btn settings-btn-add" onclick={saveAgents} disabled={!agentConfigJson.trim()}>Save</button>
        </div>
      </div>
      <div class="agent-editor-wrap">
        <textarea class="agent-json-editor" spellcheck="false" bind:value={agentConfigJson}></textarea>
      </div>
      <div class="agent-toolbar">
        {#if saveStatus}
          <span class="agent-status-msg">{saveStatus}</span>
        {/if}
        <span class="agent-count">
          {agents.length} agent{agents.length !== 1 ? 's' : ''} configured
          ({agentsWithKeys()} with API key)
        </span>
        <button class="settings-btn settings-btn-validate" onclick={validateAgents} disabled={validating || agents.length === 0}>
          {validating ? "Validating..." : "Validate Connections"}
        </button>
      </div>

      {#if agents.length > 0}
        <div class="settings-list">
          {#each agents as agent (agent.id)}
            {@const vr = validationResults[agent.id]}
            <div class="agent-card" class:agent-card-valid={vr?.valid} class:agent-card-invalid={vr && !vr.valid}>
              <div class="agent-header">
                <div class="agent-info">
                  <span class="agent-name">{agent.name}</span>
                  <span class="agent-meta">{agent.provider} / {agent.model}</span>
                </div>
                {#if vr}
                  <span class="agent-badge" class:agent-badge-ok={vr.valid} class:agent-badge-fail={!vr.valid}>
                    {vr.valid ? "OK" : "FAIL"}
                  </span>
                {:else if agent.api_key}
                  <span class="agent-badge agent-badge-key" title="Has API key">&#x1F512;</span>
                {/if}
                {#if ["coder", "shell", "architect"].includes(agent.id)}
                  <span class="agent-badge agent-badge-default">default</span>
                {/if}
              </div>
              {#if agent.base_url}
                <div class="agent-url">{agent.base_url}</div>
              {/if}
              {#if vr && vr.error}
                <div class="agent-error">{vr.error}</div>
              {/if}
            </div>
          {/each}
        </div>
      {:else}
        <div class="agent-empty">
          <p>No agents configured. Add agents to the JSON above and click Save.</p>
        </div>
      {/if}
    </div>
  {:else if settingsTab === "setup"}
    <div class="setup-section">
      <div class="setup-header">
        <span class="settings-title">Languages</span>
        <label class="switch-toggle" title="Suspend all, activate on file open">
          <span class="switch-label">Auto</span>
          <input type="checkbox" checked={autoSuspend} onchange={toggleAuto} />
          <span class="switch-slider"></span>
        </label>
      </div>

      <div class="setup-add-row">
        <div class="setup-autocomplete">
          <input type="text" class="setup-add-input" placeholder="Search language..." bind:value={langSearch}
            onfocus={() => { showSuggestions = true; }}
            onblur={() => setTimeout(() => { showSuggestions = false; }, 200)}
          />
          {#if showSuggestions && searchResults.length > 0}
            <div class="setup-suggestions">
              {#each searchResults as lang}
                <button class="setup-suggestion-item" onmousedown={() => selectLang(lang.id)}>
                  <span class="setup-suggestion-name">{lang.name}</span>
                  <span class="setup-suggestion-ext">.{lang.extensions.slice(0, 3).join(", .")}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <button class="setup-add-btn" onclick={() => { if (searchResults.length > 0) selectLang(searchResults[0].id); }} disabled={searchResults.length === 0}>Add</button>
      </div>

      <div class="setup-lang-tags">
        {#each activeLangList as lang}
          <button class="setup-lang-tag" style="border-color:{lang.color};color:{lang.color}" onclick={() => disableLang(lang.id)}>
            {lang.iconText}
            <span class="setup-lang-tag-close">&times;</span>
          </button>
        {/each}
        {#if activeLangList.length === 0}
          <span class="setup-empty-hint">No languages added. Search and add above.</span>
        {/if}
      </div>

      <div class="setup-notes-section">
        <span class="settings-title">Install Commands</span>
        <span class="setup-notes-hint">click to copy</span>
      </div>
      <div class="setup-notes">
        {#each INSTALL_NOTES as note}
          <button class="setup-note" onclick={() => copyCmd(note.cmd)}>
            <span class="setup-note-name">{note.name}</span>
            <code class="setup-note-cmd">{note.cmd}</code>
            <span class="setup-note-copy">{copiedNote === note.cmd ? "Copied!" : "Copy"}</span>
          </button>
        {/each}
      </div>
    </div>
  {:else if settingsTab === "shortcuts"}
    <div class="shortcuts-section">
      <div class="settings-header">
        <span class="settings-title">Keyboard Shortcuts</span>
        <div class="shortcut-search-wrap">
          <svg class="search-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <input 
            type="text" 
            class="shortcut-search" 
            placeholder="Search shortcuts..." 
            bind:value={shortcutSearch} 
          />
        </div>
      </div>

      {#if editingShortcutId !== null}
        {@const editingItem = shortcuts.find(s => s.id === editingShortcutId)}
        {#if editingItem}
          <div class="settings-form">
            <div class="form-title">
              Configure Hotkey: <span class="accent-text">{editingItem.name}</span>
            </div>
            
            <div class="shortcut-editor-grid">
              <div class="form-field">
                <span>Modifiers</span>
                <div class="checkbox-row">
                  <label class="checkbox-container">
                    <input type="checkbox" bind:checked={editCtrl} />
                    <span class="checkmark"></span>
                    Ctrl
                  </label>
                  <label class="checkbox-container">
                    <input type="checkbox" bind:checked={editAlt} />
                    <span class="checkmark"></span>
                    Alt
                  </label>
                  <label class="checkbox-container">
                    <input type="checkbox" bind:checked={editShift} />
                    <span class="checkmark"></span>
                    Shift
                  </label>
                </div>
              </div>

              <div class="form-field">
                <span>Key Trigger</span>
                <div class="key-input-wrapper">
                  <input 
                    type="text" 
                    class="key-input"
                    bind:value={editKey} 
                    placeholder="Press keys directly here..." 
                    maxlength="12" 
                    onkeydown={(e) => {
                      if (e.key === "Tab" || e.key === "Escape") return;
                      e.preventDefault();
                      
                      editCtrl = e.ctrlKey;
                      editAlt = e.altKey;
                      editShift = e.shiftKey;
                      
                      const lowerKey = e.key.toLowerCase();
                      if (["control", "alt", "shift", "meta"].includes(lowerKey)) {
                        return;
                      }
                      
                      editKey = lowerKey;
                    }}
                  />
                  <span class="key-hint">Press keys directly in the box above to capture modifiers and the key.</span>
                </div>
              </div>
            </div>

            <div class="form-actions">
              <span class="form-hint">Shortcuts update globally and persist automatically across reloads.</span>
              <div class="form-btns">
                <button class="settings-btn settings-btn-cancel" onclick={cancelEditShortcut}>Cancel</button>
                <button 
                  class="settings-btn settings-btn-save" 
                  onclick={saveEditedShortcut} 
                  disabled={!editKey.trim()}
                >
                  Save Shortcut
                </button>
              </div>
            </div>
          </div>
        {/if}
      {/if}

      <div class="shortcuts-list">
        {#each filteredShortcuts as item}
          <div class="shortcut-card" class:shortcut-editing={editingShortcutId === item.id}>
            <div class="shortcut-card-info">
              <span class="shortcut-name">{item.name}</span>
              <p class="shortcut-desc">{item.description}</p>
            </div>
            
            <div class="shortcut-card-right">
              <div class="shortcut-badges">
                {#if item.binding.ctrl}
                  <kbd class="kbd-key">Ctrl</kbd>
                {/if}
                {#if item.binding.alt}
                  <kbd class="kbd-key">Alt</kbd>
                {/if}
                {#if item.binding.shift}
                  <kbd class="kbd-key">Shift</kbd>
                {/if}
                <kbd class="kbd-key trigger-key">{item.binding.key.toUpperCase()}</kbd>
              </div>

              <button class="shortcut-edit-btn" onclick={() => startEditShortcut(item)} title="Modify Keybinding">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                Change
              </button>
            </div>
          </div>
        {/each}

        {#if filteredShortcuts.length === 0}
          <div class="no-shortcuts">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
            <p>No matching shortcuts found</p>
          </div>
        {/if}
      </div>
    </div>
  {:else if settingsTab === "about"}
    <div class="about-section">
      <div class="about-hero">
        <div class="about-logo">
          <svg width="48" height="48" viewBox="0 0 48 48" fill="none" stroke="var(--accent-blue)" stroke-width="1.5">
            <rect x="4" y="4" width="40" height="40" rx="8" stroke="var(--bg-primary)" stroke-width="4" fill="var(--accent-blue)" fill-opacity="0.15"/>
            <path d="M16 28l8-12 8 12" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
            <line x1="24" y1="16" x2="24" y2="36" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/>
          </svg>
        </div>
        <h1 class="about-name">Contlib</h1>
        <p class="about-version">v0.1.0</p>
        <p class="about-desc">AI-native workshop with split terminal, multi-agent chat, notepad, file manager, and code editor</p>
      </div>

      <div class="about-info">
        <div class="about-section-title">Tech Stack</div>
        <div class="about-tag-row">
          <span class="about-tag">Tauri v2</span>
          <span class="about-tag">Svelte 5</span>
          <span class="about-tag">Rust</span>
          <span class="about-tag">TypeScript</span>
          <span class="about-tag">xterm.js</span>
          <span class="about-tag">CodeMirror 6</span>
          <span class="about-tag">portable-pty</span>
        </div>
      </div>

      <div class="about-info">
        <div class="about-section-title">Features</div>
        <div class="about-features">
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>Multi-pane split terminal with CWD tracking</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>Multi-agent AI chat (All Api support)</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>Command snippet notepad (running script & portabel note)</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>File explorer</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>Code editor with syntax highlighting & theme-aware colors (can adjust)</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>9 themes (Dracula, Nord, Tokyo Night, Catppuccin, etc.)</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>Platform IO integration (auto-detect, install, build, upload)</span>
          </div>
          <div class="about-feature">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent-green)" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
            <span>AI cost tracking with per-agent token & pricing breakdown</span>
          </div>
        </div>
      </div>

      <div class="about-info">
        <div class="about-section-title">Repository</div>
        <p class="about-repo">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
          <a href="https://github.com/saiflll/contlib" target="_blank" rel="noopener noreferrer">github.com/saiflll/contlib</a>
        </p>
      </div>
      <div class="about-info">
        <div class="about-section-title">Inspired</div>
        <p class="about-repo">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
          <a href="https://github.com/crynta/terax-ai" target="_blank" rel="noopener noreferrer">github.com/crynta/terax-ai</a>
        </p>
      </div>
      <div class="about-footer">
        <p>Built with &lt;3 by the Renagge39 aka Saiflll &middot; MIT License</p>
      </div>
    </div>
  {/if}
</div>

<style>
  .settings { display:flex; flex-direction:column; height:100%; background:transparent; color:var(--text-primary); font-size:var(--font-size); overflow:hidden; }

  /* ═══ Sub-tabs ═══ */
  .settings-tabs { display:flex; gap:0; border-bottom:1px solid var(--border-subtle); flex-shrink:0; padding:0 8px; }
  .settings-tab { display:inline-flex; align-items:center; gap:5px; padding:8px 16px; background:none; border:none; border-bottom:2px solid transparent; color:var(--text-muted); font-size:var(--fs-11); font-weight:500; cursor:pointer; transition:all 0.12s ease; }
  .settings-tab:hover { color:var(--text-secondary); }
  .settings-tab.active { color:var(--accent-blue); border-bottom-color:var(--accent-blue); }

  /* ═══ Scrollable Tab Content Container Spacing ═══ */
  .appearance,
  .shortcuts-list {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .settings-list {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    align-content: start;
    gap: 12px;
  }

  /* ═══ Consolidated Premium Card Styles ═══ */
  .theme-card,
  .agent-card,
  .shortcut-card {
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: 10px;
    padding: 12px 16px;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    box-sizing: border-box;
  }

  .theme-card:hover,
  .agent-card:hover,
  .shortcut-card:hover {
    border-color: var(--accent-blue);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
    transform: translateY(-1.5px);
  }

  /* ═══ Scrollbar Overhauls ═══ */
  .appearance::-webkit-scrollbar,
  .settings-list::-webkit-scrollbar,
  .shortcuts-list::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  .appearance::-webkit-scrollbar-thumb,
  .settings-list::-webkit-scrollbar-thumb,
  .shortcuts-list::-webkit-scrollbar-thumb {
    background: var(--border-primary);
    border-radius: 3px;
  }
  .appearance::-webkit-scrollbar-thumb:hover,
  .settings-list::-webkit-scrollbar-thumb:hover,
  .shortcuts-list::-webkit-scrollbar-thumb:hover {
    background: var(--bg-hover);
  }

  /* ═══ Appearance Specifics ═══ */
  .section-title { font-size:var(--fs-10); font-weight:600; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px; margin-bottom:4px; margin-top:4px; }
  .section-title:not(:first-child) { margin-top:12px; }

  .theme-grid { display:grid; grid-template-columns:repeat(auto-fill, minmax(130px, 1fr)); gap:10px; }
  .theme-card {
    position:relative; display:flex; flex-direction:column; align-items:center; gap:6px;
    padding:10px 8px 8px; cursor:pointer;
  }
  .theme-active { border-color:var(--accent-blue) !important; background:color-mix(in srgb, var(--accent-blue) 8%, var(--bg-surface)) !important; }
  .theme-preview { width:100%; height:46px; border-radius:6px; overflow:hidden; }
  .tp-bg { width:100%; height:100%; display:flex; border-radius:4px; overflow:hidden; }
  .tp-side { width:14px; border-right:1px solid; flex-shrink:0; }
  .tp-main { flex:1; display:flex; flex-direction:column; justify-content:center; gap:4px; padding:0 6px; }
  .tp-line { height:4px; border-radius:2px; opacity:0.7; }
  .theme-name { font-size:var(--fs-10); font-weight:500; color:var(--text-primary); }
  .theme-check { position:absolute; top:4px; right:6px; font-size:var(--font-size); color:var(--accent-blue); }

  .font-section { display:flex; flex-direction:column; gap:8px; }
  .font-select { background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:6px; padding:6px 10px; font-size:var(--font-size); }
  .font-select:focus { outline:none; border-color:var(--accent-blue); }
  .font-preview { background:var(--bg-primary); border:1px solid var(--border-subtle); border-radius:6px; padding:10px 12px; font-size:var(--fs-14); color:var(--accent-blue); text-align:center; }

  /* ═══ Agent Specifics ═══ */
  .agent-section { display:flex; flex-direction:column; flex:1; overflow:hidden; }
  .settings-header { display:flex; align-items:center; justify-content:space-between; padding:10px 16px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .settings-title { font-size:var(--fs-10); font-weight:600; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px; }
  
  /* Consolidated Interactive Button Design */
  .settings-btn, .shortcut-edit-btn {
    display:inline-flex; align-items:center; justify-content:center; gap:6px;
    border:1px solid transparent; border-radius:6px; padding:6px 12px; font-size:var(--fs-11);
    font-weight:500; cursor:pointer; transition:all 0.15s ease;
  }
  .settings-btn-add { background:var(--accent-blue); color:var(--bg-primary); }
  .settings-btn-add:disabled { opacity:0.4; cursor:not-allowed; }
  .settings-btn-add:hover:not(:disabled) { filter:brightness(1.15); box-shadow:0 0 8px rgba(129, 140, 248, 0.25); }
  
  .settings-btn-cancel { background:transparent; color:var(--text-muted); border-color:var(--border-subtle); }
  .settings-btn-cancel:hover { color:var(--text-primary); background:var(--bg-hover); }
  
  .settings-btn-save { background:var(--accent-green); color:var(--bg-primary); }
  .settings-btn-save:disabled { opacity:0.4; cursor:not-allowed; }
  .settings-btn-save:hover:not(:disabled) { filter:brightness(1.15); }

  .settings-form { padding:14px 16px; background:var(--bg-surface); border-bottom:1px solid var(--border-subtle); flex-shrink:0; animation:slideDown 0.2s cubic-bezier(0.4, 0, 0.2, 1); }
  @keyframes slideDown { from { opacity:0; transform: translateY(-4px); } to { opacity:1; transform: translateY(0); } }
  .form-field { display:flex; flex-direction:column; gap:4px; }
  .form-field span { font-size:var(--fs-10); color:var(--text-muted); text-transform:uppercase; letter-spacing:0.5px; }
  .form-field input, .form-field select { background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:6px; padding:6px 10px; font-size:var(--font-size); }
  .form-field input:focus, .form-field select:focus { outline:none; border-color:var(--accent-blue); box-shadow:0 0 0 2px color-mix(in srgb, var(--accent-blue) 15%, transparent); }
  .form-actions { display:flex; align-items:center; justify-content:space-between; }
  .form-hint { color:var(--text-muted); font-size:var(--fs-10); }
  .form-btns { display:flex; gap:6px; }

  .agent-card { display:flex; flex-direction:column; gap:8px; }

  .agent-header { display:flex; align-items:center; gap:8px; margin-bottom:2px; }
  .agent-info { flex:1; min-width:0; }
  .agent-name { font-weight:600; font-size:var(--font-size); color:var(--text-primary); }
  .agent-meta { font-size:var(--fs-10); color:var(--text-muted); margin-left:6px; }
  .agent-badge { font-size:var(--fs-9); padding:2px 6px; border-radius:3px; }
  .agent-badge-key { font-size:var(--font-size); padding:0; background:none; }
  .agent-badge-default { background:var(--accent-blue); color:var(--bg-primary); text-transform:uppercase; letter-spacing:0.3px; font-weight: 600; }
  .agent-url { font-size:var(--fs-10); color:var(--text-muted); font-family:monospace; margin-bottom:2px; padding:4px 8px; background:var(--bg-primary); border-radius:4px; border:1px solid var(--border-subtle); }
  .agent-card-valid { border-color:var(--accent-green) !important; }
  .agent-card-invalid { border-color:var(--accent-red) !important; }
  .agent-error { font-size:var(--fs-10); color:var(--accent-red); padding:4px 8px; background:color-mix(in srgb, var(--accent-red) 6%, transparent); border-radius:4px; margin-top:4px; word-break:break-word; }
  .agent-header-actions { display:flex; gap:6px; }
  .agent-editor-wrap { flex-shrink:0; border-bottom:1px solid var(--border-subtle); }
  .agent-json-editor {
    width:100%; height:220px; padding:10px 12px; box-sizing:border-box;
    background:var(--bg-primary); color:var(--text-primary);
    border:none; font-family:monospace; font-size:var(--fs-11); line-height:1.5;
    resize:vertical; outline:none; tab-size:2;
  }
  .agent-json-editor:focus { background:var(--bg-elevated); }
  .agent-toolbar {
    display:flex; align-items:center; gap:8px; padding:6px 16px;
    border-bottom:1px solid var(--border-subtle); flex-shrink:0;
  }
  .agent-count { font-size:var(--fs-10); color:var(--text-muted); flex:1; }
  .agent-status-msg { font-size:var(--fs-10); color:var(--accent-blue); }
  .agent-empty { display:flex; align-items:center; justify-content:center; flex:1; padding:24px; }
  .agent-empty p { font-size:var(--fs-11); color:var(--text-muted); margin:0; }
  .agent-badge-ok { background:var(--accent-green); color:var(--bg-primary); padding:2px 6px; border-radius:3px; font-size:var(--fs-9); font-weight:600; }
  .agent-badge-fail { background:var(--accent-red); color:var(--bg-primary); padding:2px 6px; border-radius:3px; font-size:var(--fs-9); font-weight:600; }
  .settings-btn-validate {
    background:transparent; color:var(--text-secondary);
    border:1px solid var(--border-subtle); border-radius:6px;
    padding:4px 10px; font-size:var(--fs-10); cursor:pointer; transition:all 0.12s ease;
  }
  .settings-btn-validate:hover:not(:disabled) { border-color:var(--accent-blue); color:var(--accent-blue); }
  .settings-btn-validate:disabled { opacity:0.4; cursor:not-allowed; }


  /* ═══ Setup (minimal language management) ═══ */
  .setup-section { display:flex; flex-direction:column; height:100%; overflow-y:auto; gap:10px; padding:14px; }
  .setup-header { display:flex; align-items:center; justify-content:space-between; gap:8px; margin-bottom:2px; }
  .setup-header .settings-title { margin:0; }
  .switch-toggle { position:relative; display:inline-flex; align-items:center; gap:6px; width:auto; height:20px; flex-shrink:0; cursor:pointer; }
  .switch-toggle .switch-label { font-size:var(--fs-10); color:var(--text-muted); user-select:none; }
  .switch-toggle input { opacity:0; width:0; height:0; position:absolute; }
  .switch-slider { position:relative; width:30px; height:16px; background-color:var(--bg-primary); border:1px solid var(--border-subtle); border-radius:10px; transition:.2s ease; display:inline-block; }
  .switch-slider:before { content:""; position:absolute; height:10px; width:10px; left:2px; top:2px; background-color:var(--text-muted); border-radius:50%; transition:.2s cubic-bezier(0.4,0,0.2,1); }
  .switch-toggle input:checked + .switch-slider { background-color:var(--accent-blue); border-color:var(--accent-blue); }
  .switch-toggle input:checked + .switch-slider:before { transform:translateX(14px); background-color:var(--bg-primary); }

  .setup-add-row { display:flex; align-items:center; gap:6px; position:relative; }
  .setup-autocomplete { flex:1; position:relative; }
  .setup-add-input { width:100%; background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:6px; padding:5px 8px; font-size:var(--font-size); box-sizing:border-box; }
  .setup-add-input:focus { outline:none; border-color:var(--accent-blue); }
  .setup-suggestions { position:absolute; top:100%; left:0; right:0; background:var(--bg-elevated); border:1px solid var(--border-subtle); border-radius:6px; margin-top:2px; z-index:10; overflow:hidden; box-shadow:0 4px 12px rgba(0,0,0,0.15); }
  .setup-suggestion-item { display:flex; align-items:center; justify-content:space-between; gap:8px; width:100%; padding:6px 10px; border:none; background:none; color:var(--text-primary); font-size:var(--font-size); cursor:pointer; text-align:left; transition:background 0.1s; }
  .setup-suggestion-item:hover { background:var(--bg-hover); }
  .setup-suggestion-name { font-weight:500; }
  .setup-suggestion-ext { font-size:var(--fs-10); color:var(--text-muted); font-family:monospace; }
  .setup-add-btn { background:var(--accent-blue); color:var(--bg-primary); border:none; border-radius:6px; padding:5px 12px; font-size:var(--font-size); font-weight:500; cursor:pointer; white-space:nowrap; transition:filter 0.15s; }
  .setup-add-btn:hover:not(:disabled) { filter:brightness(1.15); }
  .setup-add-btn:disabled { opacity:0.4; cursor:not-allowed; }

  .setup-lang-tags { display:flex; flex-wrap:wrap; gap:4px; min-height:24px; align-items:center; }
  .setup-lang-tag { display:inline-flex; align-items:center; gap:3px; padding:2px 8px; border:1px solid; border-radius:4px; background:color-mix(in srgb, currentColor 8%, transparent); font-size:var(--fs-10); font-weight:600; cursor:pointer; transition:opacity 0.15s; }
  .setup-lang-tag:hover { opacity:0.7; }
  .setup-lang-tag-close { font-size:var(--fs-11); line-height:1; margin-left:2px; }
  .setup-empty-hint { font-size:var(--fs-10); color:var(--text-muted); font-style:italic; }

  .setup-notes-section { display:flex; align-items:baseline; gap:8px; margin-top:4px; }
  .setup-notes-hint { font-size:var(--fs-9); color:var(--text-muted); }
  .setup-notes { display:flex; flex-direction:column; gap:3px; }
  .setup-note { display:flex; align-items:center; gap:8px; width:100%; padding:4px 8px; border:1px solid var(--border-subtle); border-radius:5px; background:var(--bg-primary); color:var(--text-primary); cursor:pointer; transition:all 0.12s; text-align:left; }
  .setup-note:hover { border-color:var(--accent-blue); background:color-mix(in srgb, var(--accent-blue) 4%, transparent); }
  .setup-note-name { font-size:var(--font-size); font-weight:600; min-width:60px; color:var(--text-secondary); }
  .setup-note-cmd { flex:1; font-family:monospace; font-size:var(--fs-10); color:var(--text-primary); padding:2px 6px; background:var(--bg-surface); border-radius:3px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .setup-note-copy { font-size:var(--fs-9); color:var(--accent-blue); white-space:nowrap; flex-shrink:0; }

  /* ═══ Keyboard Shortcuts Specifics ═══ */
  .shortcuts-section { display:flex; flex-direction:column; flex:1; overflow:hidden; }
  
  .shortcut-search-wrap { position:relative; display:flex; align-items:center; }
  .shortcut-search {
    background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle);
    border-radius:6px; padding:6px 8px 6px 26px; font-size:var(--fs-11); width:160px; transition:all 0.15s ease;
  }
  .shortcut-search:focus { outline:none; border-color:var(--accent-blue); width:200px; box-shadow:0 0 0 2px color-mix(in srgb, var(--accent-blue) 15%, transparent); }
  .search-icon { position:absolute; left:8px; color:var(--text-muted); pointer-events:none; }
  
  .shortcut-card {
    display:flex; justify-content:space-between; align-items:center; gap:16px;
  }
  .shortcut-editing { border-color:var(--accent-blue) !important; background:color-mix(in srgb, var(--accent-blue) 4%, var(--bg-surface)) !important; }
  
  .shortcut-card-info { display:flex; flex-direction:column; gap:2px; }
  .shortcut-name { font-size:var(--font-size); font-weight:600; color:var(--text-primary); }
  .shortcut-desc { font-size:var(--fs-10); color:var(--text-muted); margin:0; }
  
  .shortcut-card-right { display:flex; align-items:center; gap:12px; }
  .shortcut-badges { display:flex; align-items:center; gap:4px; }
  
  .kbd-key {
    display:inline-flex; align-items:center; justify-content:center;
    background:var(--bg-primary); color:var(--text-secondary);
    border:1px solid var(--border-primary); border-bottom-width:2px;
    border-radius:4px; padding:2px 6px; font-family:monospace; font-size:var(--fs-9-5); font-weight:600;
    box-shadow:0 1px 2px rgba(0,0,0,0.15); text-transform:uppercase; min-width:22px; text-align:center;
  }
  .trigger-key {
    background:color-mix(in srgb, var(--accent-blue) 12%, transparent);
    color:var(--accent-blue); border-color:color-mix(in srgb, var(--accent-blue) 35%, transparent);
  }
  
  .shortcut-edit-btn { background:transparent; color:var(--text-secondary); border:1px solid var(--border-subtle); }
  .shortcut-edit-btn:hover { border-color:var(--accent-blue); color:var(--accent-blue); background:color-mix(in srgb, var(--accent-blue) 6%, transparent); }
  
  .no-shortcuts { display:flex; flex-direction:column; align-items:center; justify-content:center; gap:8px; padding:32px 0; color:var(--text-muted); }
  .no-shortcuts p { margin:0; font-size:var(--fs-11); }

  /* Modifier checkboxes / Editor layout */
  .form-title { font-size:var(--fs-11); font-weight:600; color:var(--text-primary); margin-bottom:8px; }
  .accent-text { color:var(--accent-blue); }
  .shortcut-editor-grid { display:grid; grid-template-columns:1.2fr 1fr; gap:16px; margin-bottom:10px; }
  
  .checkbox-row { display:flex; gap:12px; margin-top:4px; }
  .checkbox-container {
    position:relative; display:inline-flex; align-items:center; gap:6px;
    font-size:var(--fs-11); color:var(--text-secondary); cursor:pointer; user-select:none;
  }
  .checkbox-container input { display:none; }
  .checkmark {
    width:14px; height:14px; border:1px solid var(--border-subtle); border-radius:3px;
    background:var(--bg-primary); display:inline-block; position:relative; transition:all 0.12s ease;
  }
  .checkbox-container:hover input ~ .checkmark { border-color:var(--accent-blue); }
  .checkbox-container input:checked ~ .checkmark { background:var(--accent-blue); border-color:var(--accent-blue); }
  .checkmark:after {
    content:""; position:absolute; display:none;
    left:4px; top:1px; width:4px; height:7px;
    border:solid var(--bg-primary); border-width:0 2px 2px 0; transform:rotate(45deg);
  }
  .checkbox-container input:checked ~ .checkmark:after { display:block; }
  
  .key-input-wrapper { display:flex; flex-direction:column; gap:4px; }
  .key-input {
    background:var(--bg-primary); color:var(--accent-blue); font-family:monospace; font-weight:600;
    border:1px solid var(--border-subtle); border-radius:6px; padding:6px 10px; font-size:var(--fs-11); text-transform:uppercase; text-align:center;
    letter-spacing:1px; box-shadow:inset 0 1px 3px rgba(0,0,0,0.1); width:100%; box-sizing:border-box;
  }
  .key-input:focus { outline:none; border-color:var(--accent-blue); box-shadow:0 0 0 2px color-mix(in srgb, var(--accent-blue) 15%, transparent); }
  .key-hint { font-size:var(--fs-9-5); color:var(--text-muted); line-height:1.3; }

  /* Font Size */
  .fontsize-section { display:flex; align-items:center; gap:10px; padding:6px 0; }
  .fontsize-slider { flex:1; height:4px; -webkit-appearance:none; background:var(--bg-elevated); border-radius:2px; outline:none; cursor:pointer; }
  .fontsize-slider::-webkit-slider-thumb { -webkit-appearance:none; width:14px; height:14px; border-radius:50%; background:var(--accent-blue); border:2px solid var(--bg-primary); cursor:pointer; }
  .fontsize-label { font-size:var(--font-size); color:var(--text-primary); font-family:monospace; min-width:32px; text-align:right; }

  /* About */
  .about-section { display:flex; flex-direction:column; height:100%; overflow-y:auto; padding:24px 20px; gap:20px; }
  .about-hero { display:flex; flex-direction:column; align-items:center; gap:8px; text-align:center; }
  .about-logo { opacity:0.9; }
  .about-name { font-size:var(--fs-22); font-weight:700; color:var(--text-primary); margin:0; }
  .about-version { font-size:var(--font-size); color:var(--text-muted); font-family:monospace; }
  .about-desc { font-size:var(--font-size); color:var(--text-secondary); max-width:360px; line-height:1.5; }
  .about-section-title { font-size:var(--fs-10); font-weight:600; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px; margin-bottom:8px; }
  .about-tag-row { display:flex; flex-wrap:wrap; gap:6px; }
  .about-tag { font-size:var(--fs-10); padding:3px 10px; background:color-mix(in srgb, var(--accent-blue) 10%, transparent); color:var(--accent-blue); border:1px solid color-mix(in srgb, var(--accent-blue) 20%, transparent); border-radius:10px; font-family:monospace; }
  .about-features { display:flex; flex-direction:column; gap:5px; }
  .about-feature { display:flex; align-items:center; gap:8px; font-size:var(--fs-11); color:var(--text-secondary); }
  .about-repo { display:flex; align-items:center; gap:6px; font-size:var(--fs-11); color:var(--text-secondary); }
  .about-repo a { color:var(--accent-blue); text-decoration:none; }
  .about-repo a:hover { text-decoration:underline; }
  .about-footer { margin-top:auto; text-align:center; font-size:var(--fs-10); color:var(--text-muted); padding-top:16px; }
</style>
