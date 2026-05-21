<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";
  import { currentDir, activeTerminalSessionId, terminalSessions } from "../stores.svelte";

  let content = $state("");
  let language = $state("typescript");
  let fileName = $state("scratch.ts");
  let isExecuting = $state(false);
  let activePane = $state<"editor" | "saved">("editor");

  // Subscribe to global stores
  let workspaceDir = $state("");
  let activeTermId = $state<string | null>(null);
  let sessionsMap = $state<Map<string, any>>(new Map());
  let savedFiles = $state<string[]>([]);

  const LANGUAGES = [
    { id: "typescript", label: "TypeScript", ext: ".ts" },
    { id: "javascript", label: "JavaScript", ext: ".js" },
    { id: "python", label: "Python", ext: ".py" },
    { id: "rust", label: "Rust", ext: ".rs" },
    { id: "html", label: "HTML", ext: ".html" },
    { id: "css", label: "CSS", ext: ".css" },
    { id: "json", label: "JSON", ext: ".json" },
    { id: "markdown", label: "Markdown", ext: ".md" },
    { id: "shell", label: "Shell", ext: ".sh" },
    { id: "sql", label: "SQL", ext: ".sql" },
  ];

  $effect(() => {
    const unsubDir = currentDir.subscribe(val => {
      workspaceDir = val;
      loadSavedFiles();
    });
    const unsubTerm = activeTerminalSessionId.subscribe(val => {
      activeTermId = val;
    });
    const unsubSessions = terminalSessions.subscribe(val => {
      sessionsMap = val;
    });
    return () => {
      unsubDir();
      unsubTerm();
      unsubSessions();
    };
  });

  async function loadSavedFiles() {
    try {
      const baseDir = workspaceDir || "C:\\Users\\Lenovo\\Documents\\dev\\contlib";
      const notebooksPath = `${baseDir}\\notebooks`;
      
      // Ensure directory exists
      await invoke("fs_create_dir", { path: notebooksPath }).catch(() => {});
      
      const entries = await invoke<any[]>("fs_list_dir", { path: notebooksPath });
      savedFiles = entries
        .filter(e => !e.is_dir)
        .map(e => e.name);
    } catch (e) {
      console.error("Failed to load saved notebooks:", e);
    }
  }

  function updateLanguage() {
    const lang = LANGUAGES.find((l) => l.id === language);
    if (lang) {
      const baseName = fileName.replace(/\.[^.]+$/, "");
      fileName = `${baseName}${lang.ext}`;
    }
  }

  async function runInTerminal() {
    if (!content.trim()) return;
    isExecuting = true;

    try {
      const baseDir = workspaceDir || "C:\\Users\\Lenovo\\Documents\\dev\\contlib";
      const tempFolder = `${baseDir}\\.notepad_temp`;
      const tempFilePath = `${tempFolder}\\${fileName}`;

      // Write content to workspace temp directory
      await invoke("fs_write_file", {
        path: tempFilePath,
        content,
      });

      // Find the active terminal session
      let termId = activeTermId;
      if (!termId) {
        if (sessionsMap.size > 0) {
          termId = Array.from(sessionsMap.keys())[0];
        }
      }
      if (!termId) {
        // Open a new terminal session if none exist
        termId = await invoke<string>("pty_open", {
          shell: null,
          rows: 24,
          cols: 80,
        });
        activeTerminalSessionId.set(termId);
      }

      // Format execution commands
      let cmd = "";
      if (language === "shell") {
        cmd = content;
      } else if (language === "python") {
        cmd = `python "${tempFilePath}"`;
      } else if (language === "rust") {
        const binPath = tempFilePath.replace(".rs", "");
        cmd = `rustc "${tempFilePath}" -o "${binPath}" && "${binPath}"`;
      } else if (language === "javascript" || language === "typescript") {
        cmd = `node "${tempFilePath}"`;
      } else {
        cmd = `echo "Running ${fileName} in terminal..."`;
      }

      // Write the run command into the active PTY session
      await invoke("pty_write", {
        sessionId: termId,
        data: cmd + "\r\n",
      });
    } catch (e: any) {
      console.error("Failed to run notepad script:", e);
    }
    isExecuting = false;
  }

  async function saveNotepad() {
    if (!fileName.trim()) return;
    try {
      const baseDir = workspaceDir || "C:\\Users\\Lenovo\\Documents\\dev\\contlib";
      const targetPath = `${baseDir}\\notebooks\\${fileName}`;
      await invoke("fs_write_file", {
        path: targetPath,
        content,
      });
      await loadSavedFiles();
      alert(`File saved to notebooks/${fileName}`);
    } catch (e: any) {
      console.error("Failed to save notepad file:", e);
      alert("Failed to save notepad: " + e.toString());
    }
  }

  async function deleteFile(name: string) {
    if (!confirm(`Are you sure you want to delete "${name}"?`)) return;
    try {
      const baseDir = workspaceDir || "C:\\Users\\Lenovo\\Documents\\dev\\contlib";
      const targetPath = `${baseDir}\\notebooks\\${name}`;
      await invoke("fs_delete", { path: targetPath });
      await loadSavedFiles();
      // If deleting currently loaded file, reset filename
      if (fileName === name) {
        fileName = "scratch.ts";
      }
    } catch (e: any) {
      console.error("Failed to delete notepad file:", e);
      alert("Failed to delete: " + e.toString());
    }
  }

  async function editFile(name: string) {
    try {
      const baseDir = workspaceDir || "C:\\Users\\Lenovo\\Documents\\dev\\contlib";
      const targetPath = `${baseDir}\\notebooks\\name`;
      const fileContent = await invoke<string>("fs_read_file", { path: `${baseDir}\\notebooks\\${name}` });
      
      content = fileContent;
      fileName = name;
      
      // Auto-detect language
      const ext = "." + name.split(".").pop();
      const foundLang = LANGUAGES.find(l => l.ext === ext);
      if (foundLang) {
        language = foundLang.id;
      }
      
      activePane = "editor";
    } catch (e: any) {
      console.error("Failed to load file for editing:", e);
      alert("Failed to load file: " + e.toString());
    }
  }

  async function runSavedFile(name: string) {
    try {
      const baseDir = workspaceDir || "C:\\Users\\Lenovo\\Documents\\dev\\contlib";
      const fileContent = await invoke<string>("fs_read_file", { path: `${baseDir}\\notebooks\\${name}` });
      
      // Temporary write to run
      const tempFolder = `${baseDir}\\.notepad_temp`;
      const tempFilePath = `${tempFolder}\\${name}`;
      
      await invoke("fs_write_file", {
        path: tempFilePath,
        content: fileContent,
      });

      let termId = activeTermId;
      if (!termId) {
        if (sessionsMap.size > 0) {
          termId = Array.from(sessionsMap.keys())[0];
        }
      }
      if (!termId) {
        termId = await invoke<string>("pty_open", {
          shell: null, rows: 24, cols: 80,
        });
        activeTerminalSessionId.set(termId);
      }

      const ext = "." + name.split(".").pop();
      const foundLang = LANGUAGES.find(l => l.ext === ext);
      const fileLang = foundLang ? foundLang.id : "shell";

      let cmd = "";
      if (fileLang === "shell") {
        cmd = fileContent;
      } else if (fileLang === "python") {
        cmd = `python "${tempFilePath}"`;
      } else if (fileLang === "rust") {
        const binPath = tempFilePath.replace(".rs", "");
        cmd = `rustc "${tempFilePath}" -o "${binPath}" && "${binPath}"`;
      } else if (fileLang === "javascript" || fileLang === "typescript") {
        cmd = `node "${tempFilePath}"`;
      } else {
        cmd = `echo "Running ${name}..."`;
      }

      await invoke("pty_write", {
        sessionId: termId,
        data: cmd + "\r\n",
      });
    } catch (e: any) {
      console.error("Failed to run saved file:", e);
      alert("Failed to run file: " + e.toString());
    }
  }

  function startNewNote() {
    content = "";
    fileName = "scratch.ts";
    language = "typescript";
    activePane = "editor";
  }
</script>

<div class="notepad">
  <!-- ═══ Inner Tab Header ═══ -->
  <div class="notepad-tab-header">
    <div class="notepad-tabs">
      <button 
        class="notepad-tab" 
        class:active={activePane === "editor"} 
        onclick={() => activePane = "editor"}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
        Editor
      </button>
      <button 
        class="notepad-tab" 
        class:active={activePane === "saved"} 
        onclick={() => { activePane = "saved"; loadSavedFiles(); }}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        Saved Notes ({savedFiles.length})
      </button>
    </div>
    
    <button class="new-note-btn" onclick={startNewNote} title="Create New Scratchpad">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      New Note
    </button>
  </div>

  {#if activePane === "editor"}
    <!-- Editor Pane -->
    <div class="notepad-header">
      <div class="file-info">
        <input
          bind:value={fileName}
          class="file-name-input"
          placeholder="filename.ext"
        />
        <select bind:value={language} onchange={updateLanguage}>
          {#each LANGUAGES as lang}
            <option value={lang.id}>{lang.label}</option>
          {/each}
        </select>
      </div>
      <div class="notepad-actions">
        <button onclick={saveNotepad} class="action-btn" title="Save file to notebooks/">
          Save
        </button>
        <button
          onclick={runInTerminal}
          class="action-btn run-btn"
          disabled={isExecuting || !content.trim()}
          title="Run in active terminal"
        >
          {isExecuting ? "Running..." : "Run"}
        </button>
      </div>
    </div>
    <div class="notepad-editor">
      <textarea
        bind:value={content}
        class="notepad-textarea"
        placeholder="Write code or notes here... Save files inside notebooks folder, or run scripts directly in terminal!"
        spellcheck={false}
      ></textarea>
    </div>
  {:else}
    <!-- Saved Notes Pane -->
    <div class="saved-notes-list">
      {#each savedFiles as file}
        <div class="saved-note-item">
          <div class="note-item-info">
            <svg class="note-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            <span class="note-name" title={file}>{file}</span>
          </div>
          <div class="note-item-actions">
            <button onclick={() => editFile(file)} class="note-btn note-btn-edit" title="Edit File">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              Edit
            </button>
            <button onclick={() => runSavedFile(file)} class="note-btn note-btn-run" title="Run in Terminal">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"/></svg>
              Run
            </button>
            <button onclick={() => deleteFile(file)} class="note-btn note-btn-del" title="Delete File">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            </button>
          </div>
        </div>
      {:else}
        <div class="no-notes-fallback">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          <p>No saved files found in notebooks/</p>
          <button class="create-first-btn" onclick={startNewNote}>Create a Note</button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .notepad {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  /* Inner tab headers style */
  .notepad-tab-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-subtle);
    padding: 0 8px;
    flex-shrink: 0;
    height: 32px;
  }
  .notepad-tabs {
    display: flex;
    height: 100%;
    align-items: flex-end;
  }
  .notepad-tab {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    height: 100%;
    padding: 0 10px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--text-muted);
    font-size: var(--fs-11);
    font-weight: 500;
    cursor: pointer;
    transition: all 0.12s ease;
  }
  .notepad-tab:hover {
    color: var(--text-secondary);
  }
  .notepad-tab.active {
    color: var(--accent-blue);
    border-bottom-color: var(--accent-blue);
  }
  
  .new-note-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: var(--bg-elevated);
    border: 1px solid var(--border-subtle);
    border-radius: 4px;
    padding: 3px 8px;
    font-size: var(--fs-10);
    font-weight: 600;
    color: var(--accent-blue);
    cursor: pointer;
    transition: all 0.12s ease;
  }
  .new-note-btn:hover {
    background: var(--bg-hover);
    border-color: var(--accent-blue);
  }

  /* Editor Header Style */
  .notepad-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 8px;
    background: var(--bg-surface);
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
    gap: 8px;
  }
  .file-info {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
  }
  .file-name-input {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: 4px;
    color: var(--text-primary);
    padding: 3px 8px;
    font-size: 11.5px;
    font-family: monospace;
    width: 140px;
    transition: border-color 0.15s ease;
  }
  .file-name-input:focus {
    border-color: var(--accent-blue);
    outline: none;
  }
  .file-info select {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-subtle);
    border-radius: 4px;
    padding: 3px 6px;
    font-size: var(--fs-11);
    outline: none;
    transition: border-color 0.15s ease;
  }
  .file-info select:focus {
    border-color: var(--accent-blue);
  }
  
  .notepad-actions {
    display: flex;
    gap: 6px;
  }
  .action-btn {
    background: var(--bg-elevated);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
    border-radius: 4px;
    padding: 4px 10px;
    font-size: var(--fs-11);
    font-weight: 500;
    cursor: pointer;
    transition: all 0.12s ease;
  }
  .action-btn:hover {
    border-color: var(--accent-blue);
    color: var(--text-primary);
    background: var(--bg-hover);
  }
  .run-btn {
    background: color-mix(in srgb, var(--accent-green) 15%, var(--bg-elevated));
    color: var(--accent-green);
    border-color: color-mix(in srgb, var(--accent-green) 35%, var(--border-subtle));
    font-weight: 600;
  }
  .run-btn:hover:not(:disabled) {
    background: color-mix(in srgb, var(--accent-green) 25%, var(--bg-elevated));
    border-color: var(--accent-green);
  }
  .run-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .notepad-editor {
    flex: 1;
    overflow: hidden;
  }
  .notepad-textarea {
    width: 100%;
    height: 100%;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: none;
    resize: none;
    padding: 10px;
    font-family: inherit;
    font-size: 12.5px;
    line-height: 1.6;
    tab-size: 2;
    outline: none;
  }
  .notepad-textarea::placeholder {
    color: var(--text-muted);
  }

  /* Saved notes list style */
  .saved-notes-list {
    flex: 1;
    overflow-y: auto;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .saved-notes-list::-webkit-scrollbar {
    width: 4px;
  }
  .saved-notes-list::-webkit-scrollbar-thumb {
    background: var(--bg-hover);
    border-radius: 2px;
  }
  .saved-note-item {
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: 8px;
    padding: 8px 10px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    transition: all 0.12s ease;
  }
  .saved-note-item:hover {
    border-color: var(--accent-blue);
    background: color-mix(in srgb, var(--accent-blue) 3%, var(--bg-surface));
  }
  .note-item-info {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }
  .note-icon {
    color: var(--text-muted);
    flex-shrink: 0;
  }
  .saved-note-item:hover .note-icon {
    color: var(--accent-blue);
  }
  .note-name {
    font-size: 11.5px;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .note-item-actions {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }
  .note-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 3px;
    background: var(--bg-elevated);
    color: var(--text-secondary);
    border: 1px solid var(--border-subtle);
    border-radius: 4px;
    padding: 3px 8px;
    font-size: var(--fs-10);
    font-weight: 500;
    cursor: pointer;
    transition: all 0.12s ease;
  }
  .note-btn:hover {
    background: var(--bg-hover);
  }
  .note-btn-edit:hover {
    border-color: var(--accent-blue);
    color: var(--accent-blue);
  }
  .note-btn-run {
    background: color-mix(in srgb, var(--accent-green) 10%, var(--bg-elevated));
    color: var(--accent-green);
    border-color: color-mix(in srgb, var(--accent-green) 30%, var(--border-subtle));
    font-weight: 600;
  }
  .note-btn-run:hover {
    background: color-mix(in srgb, var(--accent-green) 20%, var(--bg-elevated));
    border-color: var(--accent-green);
  }
  .note-btn-del:hover {
    border-color: var(--accent-red);
    color: var(--accent-red);
  }

  .no-notes-fallback {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 48px 16px;
    color: var(--text-muted);
    text-align: center;
  }
  .no-notes-fallback p {
    margin: 0;
    font-size: var(--fs-11);
  }
  .create-first-btn {
    margin-top: 6px;
    background: var(--accent-blue);
    color: var(--bg-primary);
    border: none;
    border-radius: 4px;
    padding: 5px 12px;
    font-size: var(--fs-11);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.12s ease;
  }
  .create-first-btn:hover {
    filter: brightness(1.1);
  }
</style>
