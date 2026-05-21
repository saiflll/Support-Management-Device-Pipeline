<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { invoke } from "@tauri-apps/api/core";
  import { editorLanguages } from "../stores.svelte";

  let enabledLangs = $state<Record<string, boolean>>({});
  $effect(() => {
    const unsub = editorLanguages.subscribe(val => {
      enabledLangs = val;
    });
    return unsub;
  });


  let {
    filePath = $bindable(""),
    initialContent = "",
    onSave = (_path: string, _content: string) => {},
  } = $props();

  let editorEl: HTMLDivElement;
  let editor: any = null;
  let currentContent = $state("");
  let isDirty = $state(false);
  let fileType = $state("");
  let langCompartment: any = null;

  const EXT_MAP: Record<string, string> = {
    ts: "typescript",
    tsx: "typescript",
    js: "javascript",
    jsx: "javascript",
    rs: "rust",
    py: "python",
    html: "html",
    htm: "html",
    css: "css",
    json: "json",
    md: "markdown",
    svelte: "html",
    vue: "html",
    go: "go",
    java: "java",
    kt: "kotlin",
    swift: "swift",
    c: "c",
    cpp: "cpp",
    h: "c",
    hpp: "cpp",
    sh: "shell",
    bash: "shell",
    yml: "yaml",
    yaml: "yaml",
    toml: "toml",
    sql: "sql",
    xml: "xml",
    rb: "ruby",
    php: "php",
  };

  function getLangFromExt(path: string): string {
    const ext = path.split(".").pop()?.toLowerCase() || "";
    return EXT_MAP[ext] || "";
  }

  async function saveFile() {
    if (!filePath) return;
    try {
      await invoke("fs_write_file", {
        path: filePath,
        content: currentContent,
      });
      isDirty = false;
      onSave(filePath, currentContent);
    } catch (e: any) {
      console.error("Save failed:", e);
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === "s") {
      e.preventDefault();
      saveFile();
    }
  }

  function getFileIconAndColor(path: string) {
    const name = path.split("\\").pop() || path.split("/").pop() || "";
    const ext = name.split(".").pop()?.toLowerCase() || "";
    
    const FILE_COLORS: Record<string, string> = {
      ts: "#3178c6", js: "#f7df1e", rs: "#ff8243", py: "#3572A5",
      svelte: "#ff3e00", html: "#e34f26", css: "#38bdf8", json: "#10b981",
      md: "#a855f7", yml: "#fb7185", yaml: "#fb7185", toml: "#d946ef",
      sh: "#4ade80", bash: "#4ade80", ps1: "#38bdf8",
      png: "#fbbf24", jpg: "#fbbf24", jpeg: "#fbbf24",
      svg: "#fbbf24", gif: "#fbbf24",
    };
    
    const color = FILE_COLORS[ext] || "var(--text-muted)";
    
    let svg = "";
    if (ext === "svelte") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z" fill="${color}" fill-opacity="0.1" stroke="${color}" stroke-width="1.5"/><path d="M12 6c-3.31 0-6 2.69-6 6s2.69 6 6 6 6-2.69 6-6-2.69-6-6-6zm0 10c-2.21 0-4-1.79-4-4s1.79-4 4-4 4 1.79 4 4-1.79 4-4 4z" fill="${color}" fill-opacity="0.6"/></svg>`;
    } else if (ext === "rs") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="7" stroke="${color}" stroke-width="1.5" stroke-dasharray="3 1.5"/><circle cx="12" cy="12" r="4.5" fill="${color}" fill-opacity="0.4" stroke="${color}" stroke-width="1"/></svg>`;
    } else if (ext === "json" || ext === "lock") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.5"><path d="M8 4H6a2 2 0 0 0-2 2v3a2 2 0 0 1-2 2 2 2 0 0 1 2 2v3a2 2 0 0 0 2 2h2M16 4h2a2 2 0 0 1 2 2v3a2 2 0 0 0 2 2 2 2 0 0 0-2 2v3a2 2 0 0 1-2 2h-2" stroke-linecap="round"/></svg>`;
    } else if (ext === "js") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><rect x="3" y="3" width="18" height="18" rx="3" stroke="${color}" stroke-width="1.5" fill="${color}" fill-opacity="0.1"/><text x="7" y="16" fill="${color}" font-family="monospace" font-weight="900" font-size="10px">JS</text></svg>`;
    } else if (ext === "ts") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><rect x="3" y="3" width="18" height="18" rx="3" stroke="${color}" stroke-width="1.5" fill="${color}" fill-opacity="0.1"/><text x="7" y="16" fill="${color}" font-family="monospace" font-weight="900" font-size="10px">TS</text></svg>`;
    } else if (ext === "md") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><rect x="3" y="5" width="18" height="14" rx="2" stroke="${color}" stroke-width="1.5" fill="${color}" fill-opacity="0.1"/><path d="M7 15V9l3 3 3-3v6" stroke="${color}" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M16 11l2-2m0 0l2 2m-2-2v5" stroke="${color}" stroke-width="1.5" stroke-linecap="round"/></svg>`;
    } else if (ext === "css") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><rect x="3" y="3" width="18" height="18" rx="3" stroke="${color}" stroke-width="1.5" fill="${color}" fill-opacity="0.1"/><path d="M8 8h8M8 12h5" stroke="${color}" stroke-width="1.5" stroke-linecap="round"/></svg>`;
    } else if (ext === "html") {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.5"><path d="M8 9l-3 3 3 3M16 9l3 3-3 3M11.5 7l1 10" stroke-linecap="round"/></svg>`;
    } else {
      svg = `<svg width="15" height="15" viewBox="0 0 24 24" fill="none"><rect x="2" y="4" width="20" height="16" rx="2" stroke="${color}" stroke-width="1.5" fill="${color}" fill-opacity="0.05"/><path d="M9 12l3 2 3-2" stroke="${color}" stroke-width="1.5" stroke-linecap="round"/></svg>`;
    }
    
    return { name, ext, color, svg };
  }

  onMount(async () => {
    const codemirror = await import("codemirror");
    const { EditorView, keymap, lineNumbers, highlightActiveLine,
            highlightSpecialChars, drawSelection, rectangularSelection } = await import("@codemirror/view");
    const { EditorState, Compartment } = await import("@codemirror/state");
    const { defaultKeymap, history, historyKeymap, indentWithTab } = await import("@codemirror/commands");
    const { HighlightStyle, syntaxHighlighting, bracketMatching, indentOnInput, foldGutter, foldKeymap } = await import("@codemirror/language");
    const { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } = await import("@codemirror/autocomplete");
    const { searchKeymap, highlightSelectionMatches } = await import("@codemirror/search");
    const { lintKeymap } = await import("@codemirror/lint");
    const { tags } = await import("@lezer/highlight");

    const s = getComputedStyle(document.documentElement);
    function c(name: string, fallback: string) { return s.getPropertyValue(name).trim() || fallback; }
    const bg = c("--bg-primary", "#0d0d1a");
    const bg2 = c("--bg-secondary", "#13132b");
    const borderC = c("--border-primary", "#1e1f42");
    const gutterBg = c("--bg-surface", "#1a1b3e");
    const text = c("--text-primary", "#e2e8f0");
    const text2 = c("--text-secondary", "#94a3b8");
    const font = s.getPropertyValue("font-family").trim() || "'Cascadia Code', 'Fira Code', monospace";
    const accent = c("--accent-blue", "#818cf8");

    langCompartment = new Compartment();

    // Premium theme-aware syntax highlight style
    const myHighlightStyle = HighlightStyle.define([
      { tag: tags.keyword, color: "var(--accent-blue)", fontWeight: "bold" },
      { tag: tags.controlKeyword, color: "var(--accent-blue)", fontWeight: "bold" },
      { tag: tags.string, color: "var(--accent-green)" },
      { tag: tags.character, color: "var(--accent-green)" },
      { tag: tags.number, color: "var(--accent-yellow)" },
      { tag: tags.integer, color: "var(--accent-yellow)" },
      { tag: tags.float, color: "var(--accent-yellow)" },
      { tag: tags.bool, color: "var(--accent-yellow)" },
      { tag: tags.comment, color: "var(--text-muted)", fontStyle: "italic" },
      { tag: tags.variableName, color: text },
      { tag: tags.propertyName, color: "var(--accent-blue)" },
      { tag: tags.function(tags.variableName), color: "var(--accent-blue)" },
      { tag: tags.definition(tags.variableName), color: text, fontWeight: "600" },
      { tag: tags.operator, color: "var(--text-secondary)" },
      { tag: tags.className, color: "var(--accent-yellow)" },
      { tag: tags.tagName, color: "var(--accent-blue)" },
      { tag: tags.attributeName, color: "var(--accent-blue)" },
    ]);

    const state = EditorState.create({
      doc: initialContent,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        highlightSpecialChars(),
        drawSelection(),
        rectangularSelection(),
        bracketMatching(),
        indentOnInput(),
        foldGutter(),
        history(),
        autocompletion(),
        closeBrackets(),
        highlightSelectionMatches(),
        syntaxHighlighting(myHighlightStyle, { fallback: true }),
        keymap.of([
          ...defaultKeymap,
          ...historyKeymap,
          ...foldKeymap,
          ...completionKeymap,
          ...closeBracketsKeymap,
          ...searchKeymap,
          ...lintKeymap,
          indentWithTab,
        ]),
        EditorView.updateListener.of((update: any) => {
          if (update.docChanged) {
            currentContent = update.state.doc.toString();
            isDirty = true;
          }
        }),
        langCompartment.of([]),
        EditorView.theme({
          "&": { fontSize: "13px", backgroundColor: bg, color: text },
          ".cm-scroller": { fontFamily: font },
          ".cm-gutters": { backgroundColor: gutterBg, borderRight: `1px solid ${borderC}`, color: text2 },
          ".cm-activeLineGutter": { backgroundColor: bg2 },
          ".cm-cursor": { borderLeftColor: accent },
          ".cm-selectionBackground": { backgroundColor: accent + "30" },
          "&.cm-focused .cm-selectionBackground": { backgroundColor: accent + "40" },
          ".cm-matchingBracket": { backgroundColor: accent + "20", borderBottom: `1px solid ${accent}` },
        }),
      ],
    });

    editor = new EditorView({
      state,
      parent: editorEl,
    });

    // Load initial content and language support
    if (initialContent) {
      editor.dispatch({
        changes: { from: 0, to: editor.state.doc.length, insert: initialContent }
      });
      currentContent = initialContent;
      isDirty = false;
    }
    if (filePath) {
      updateLanguage(filePath);
    }
  });

  $effect(() => {
    // Whenever initialContent changes, sync with editor if needed
    if (editor && initialContent !== undefined) {
      const state = editor.state;
      const currentDoc = state.doc.toString();
      if (currentDoc !== initialContent) {
        editor.dispatch({
          changes: { from: 0, to: state.doc.length, insert: initialContent }
        });
        currentContent = initialContent;
        isDirty = false;
      }
    }
  });

  $effect(() => {
    // Whenever filePath or enabledLangs changes, reconfigure language
    if (editor && filePath && langCompartment && enabledLangs) {
      updateLanguage(filePath);
    }
  });

  async function updateLanguage(path: string) {
    fileType = getLangFromExt(path);
    if (!fileType) return;

    let storeKey = fileType;
    if (fileType === "javascript" || fileType === "typescript") {
      storeKey = "typescript";
    }

    const isEnabled = enabledLangs[storeKey] !== false;
    if (!isEnabled) {
      if (editor && langCompartment) {
        editor.dispatch({
          effects: langCompartment.reconfigure([]),
        });
      }
      return;
    }

    try {
      let langModule: any;
      switch (fileType) {
        case "typescript":
        case "javascript":
          langModule = await import("@codemirror/lang-javascript");
          break;
        case "rust":
          langModule = await import("@codemirror/lang-rust");
          break;
        case "python":
          langModule = await import("@codemirror/lang-python");
          break;
        case "html":
          langModule = await import("@codemirror/lang-html");
          break;
        case "css":
          langModule = await import("@codemirror/lang-css");
          break;
        case "json":
          langModule = await import("@codemirror/lang-json");
          break;
        case "markdown":
          langModule = await import("@codemirror/lang-markdown");
          break;
      }

      let extension: any = null;
      if (langModule?.javascript) extension = langModule.javascript();
      else if (langModule?.rustLanguage) extension = langModule.rustLanguage();
      else if (langModule?.pythonLanguage) extension = langModule.pythonLanguage();
      else if (langModule?.html) extension = langModule.html();
      else if (langModule?.cssLanguage) extension = langModule.cssLanguage();
      else if (langModule?.jsonLanguage) extension = langModule.jsonLanguage();
      else if (langModule?.markdownLanguage) extension = langModule.markdownLanguage();

      if (extension) {
        editor.dispatch({
          effects: langCompartment.reconfigure(extension),
        });
      }
    } catch (e) {
      console.warn("Language support not available for:", fileType);
    }
  }

  onDestroy(() => {
    if (editor) editor.destroy();
  });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="code-editor" onkeydown={handleKeydown}>
  <div class="editor-header">
    <div class="editor-title-area">
      {#if filePath}
        {@const fileInfo = getFileIconAndColor(filePath)}
        <span class="editor-file-icon">{@html fileInfo.svg}</span>
        <div class="editor-file-info">
          <span class="editor-file-name">{fileInfo.name}</span>
          <span class="editor-file-dir">{filePath.substring(0, filePath.length - fileInfo.name.length)}</span>
        </div>
      {:else}
        <span class="editor-file-icon">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" stroke-width="1.5">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
          </svg>
        </span>
        <span class="editor-file-name" style="color: var(--text-muted)">Untitled File</span>
      {/if}
    </div>
    
    <div class="editor-actions">
      {#if isDirty}
        <span class="dirty-badge">● Unsaved Changes</span>
      {/if}
      <button onclick={saveFile} class="save-btn" class:dirty={isDirty} disabled={!isDirty}>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
          <polyline points="17 21 17 13 7 13 7 21"/>
          <polyline points="7 3 7 8 15 8"/>
        </svg>
        <span>Save (Ctrl+S)</span>
      </button>
    </div>
  </div>
  <div bind:this={editorEl} class="editor-instance"></div>
</div>

<style>
  .code-editor {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
  }
  .editor-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 42px;
    padding: 0 14px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    flex-shrink: 0;
    backdrop-filter: blur(8px);
  }
  .editor-title-area {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }
  .editor-file-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .editor-file-info {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .editor-file-name {
    font-size: var(--font-size);
    font-weight: 600;
    color: var(--text-primary);
    line-height: 1.2;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .editor-file-dir {
    font-size: var(--fs-10);
    color: var(--text-muted);
    font-family: monospace;
    line-height: 1.1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin-top: 1px;
    opacity: 0.7;
    direction: rtl;
    text-align: left;
  }
  .editor-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }
  .dirty-badge {
    font-size: var(--fs-9);
    background: rgba(251, 191, 36, 0.08);
    color: var(--accent-yellow);
    border: 1px solid rgba(251, 191, 36, 0.25);
    border-radius: 4px;
    padding: 1.5px 6px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    animation: pulse 2s infinite ease-in-out;
  }
  .save-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    background: transparent;
    border: 1.5px solid var(--accent-blue);
    border-radius: 6px;
    padding: 4px 10px;
    font-size: var(--fs-11);
    color: var(--accent-blue);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease-in-out;
  }
  .save-btn svg {
    flex-shrink: 0;
  }
  .save-btn:hover:not(:disabled) {
    background: var(--accent-blue);
    color: #fff;
    box-shadow: 0 0 8px rgba(129, 140, 248, 0.35);
  }
  .save-btn.dirty {
    background: var(--accent-blue);
    color: var(--bg-primary);
    border-color: var(--accent-blue);
  }
  .save-btn.dirty:hover {
    filter: brightness(1.15);
    box-shadow: 0 0 10px rgba(129, 140, 248, 0.5);
  }
  .save-btn:disabled {
    opacity: 0.35;
    border-color: var(--border-primary);
    color: var(--text-muted);
    cursor: not-allowed;
  }
  .editor-instance {
    flex: 1;
    overflow: hidden;
  }
  .editor-instance :global(.cm-editor) {
    height: 100%;
  }
  .editor-instance :global(.cm-scroller) {
    overflow: auto;
  }
  
  /* Scrollbar styles for the editor scroller */
  .editor-instance :global(.cm-scroller::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }
  .editor-instance :global(.cm-scroller::-webkit-scrollbar-track) {
    background: transparent;
  }
  .editor-instance :global(.cm-scroller::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 3px;
  }
  .editor-instance :global(.cm-scroller::-webkit-scrollbar-thumb:hover) {
    background: var(--bg-hover);
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.85; }
    50% { opacity: 0.5; }
  }
</style>
