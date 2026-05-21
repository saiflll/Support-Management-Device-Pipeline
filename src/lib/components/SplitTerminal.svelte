<script lang="ts">
  import Terminal from "./Terminal.svelte";
  import { activeTerminalSessionId } from "../stores.svelte";

  let {
    onCwdChange = (_cwd: string) => {},
  } = $props();

  let terminalIds = $state(["term-1"]);
  let sessions = $state<Record<string, string>>({});
  let cols = $state(1);
  let activeSessionId = $state<string | null>(null);

  $effect(() => {
    const unsub = activeTerminalSessionId.subscribe(val => {
      activeSessionId = val;
    });
    return unsub;
  });

  // CWD tracking per terminal
  let cwds = $state<Record<string, string>>({
    "term-1": "C:\\Users\\Lenovo\\Documents\\dev\\contlib",
  });
  let inputBuffers = $state<Record<string, string>>({});

  $effect(() => {
    cols = Math.ceil(Math.sqrt(terminalIds.length));
  });

  function primaryTerminalId(): string | null {
    if (terminalIds.length === 0) return null;
    return terminalIds.toSorted((a, b) => {
      const na = parseInt(a.replace("term-", "")) || 0;
      const nb = parseInt(b.replace("term-", "")) || 0;
      return na - nb;
    })[0];
  }

  function resolvePath(cwd: string, target: string): string {
    const trimmed = target.trim().replace(/^["']|["']$/g, "");
    if (trimmed.includes(":\\") || trimmed.startsWith("/")) return trimmed;
    const parts = cwd.split("\\");
    for (const p of trimmed.replace(/\//g, "\\").split("\\")) {
      if (!p || p === ".") continue;
      if (p === "..") { if (parts.length > 1) parts.pop(); }
      else parts.push(p);
    }
    return parts.join("\\");
  }

  function handleCommand(termId: string, line: string) {
    const trimmed = line.trim();
    if (trimmed.startsWith("cd ") || trimmed.startsWith("cd\t")) {
      const target = trimmed.slice(trimmed.startsWith("cd ") ? 3 : 2).trim();
      if (target) {
        const currentCwd = cwds[termId] || "C:\\";
        const resolved = resolvePath(currentCwd, target);
        cwds[termId] = resolved;
        if (termId === primaryTerminalId()) {
          onCwdChange(resolved);
        }
      }
    }
  }

  function addTerminal() {
    terminalIds = [...terminalIds, `term-${terminalIds.length + 1}`];
  }
  function removeTerminal(id: string) {
    if (terminalIds.length <= 1) return;
    terminalIds = terminalIds.filter((t) => t !== id);
  }
  function onSessionCreated(id: string, sessionId: string) {
    sessions[id] = sessionId;
  }
</script>

<div class="tiling">
  <div class="tiling-header">
    <span class="tiling-title">Terminal ({terminalIds.length})</span>
    <button class="tiling-btn" onclick={addTerminal} title="New terminal">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
    </button>
  </div>
  <div class="tiling-grid" style="grid-template-columns: repeat({cols}, 1fr)">
    {#each terminalIds as id (id)}
      {@const isActive = sessions[id] === activeSessionId}
      <div class="tiling-cell" class:cell-active={isActive}>
        <div class="tiling-cell-header" class:header-active={isActive}>
          <span class="tiling-cell-label" class:active={isActive}>{id}</span>
          <div class="tiling-cell-actions">
            <button class="tiling-cell-btn" onclick={() => addTerminal()} title="Split">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
            </button>
            {#if terminalIds.length > 1}
              <button class="tiling-cell-btn" onclick={() => removeTerminal(id)} title="Close">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            {/if}
          </div>
        </div>
        <div class="tiling-cell-body">
          <Terminal sessionId={sessions[id]} onReady={(sid: string) => onSessionCreated(id, sid)} onCommand={(sid: string, line: string) => handleCommand(id, line)} />
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .tiling { display:flex; flex-direction:column; height:100%; background:var(--bg-primary); }

  .tiling-header { display:flex; align-items:center; justify-content:space-between; padding:4px 10px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .tiling-title { font-size:var(--fs-11); font-weight:600; color:var(--text-muted); }
  .tiling-btn { display:flex; align-items:center; justify-content:center; background:none; border:none; color:var(--text-muted); padding:3px; cursor:pointer; border-radius:4px; }
  .tiling-btn:hover { color:var(--text-primary); background:var(--bg-hover); }

  .tiling-grid { flex:1; display:grid; gap:2px; padding:2px; overflow:hidden; grid-auto-rows:1fr; }

  .tiling-cell { display:flex; flex-direction:column; border:1px solid var(--border-subtle); border-radius:6px; overflow:hidden; min-height:60px; background:var(--bg-primary); transition: border-color 0.15s ease; }
  .tiling-cell.cell-active { border-color: var(--text-primary); }
  .tiling-cell-header { display:flex; align-items:center; justify-content:space-between; padding:1px 8px; background:var(--bg-secondary); border-bottom:1px solid var(--border-subtle); font-size:var(--fs-10); color:var(--text-muted); user-select:none; flex-shrink:0; }
  .tiling-cell-header.header-active { background: color-mix(in srgb, var(--text-primary) 5%, var(--bg-secondary)); }
  .tiling-cell-label { font-family:monospace; display:inline-flex; align-items:center; }
  .tiling-cell-label.active { color: var(--text-primary); font-weight: 600; }
  .tiling-cell-label.active::before {
    content: "";
    display: inline-block;
    width: 6px;
    height: 6px;
    background: var(--text-primary);
    border-radius: 50%;
    margin-right: 6px;
    box-shadow: 0 0 8px var(--text-primary), 0 0 2px var(--text-primary);
    animation: active-pulse 2s infinite ease-in-out;
  }
  @keyframes active-pulse {
    0% { transform: scale(1); opacity: 0.8; }
    50% { transform: scale(1.2); opacity: 1; box-shadow: 0 0 10px var(--text-primary), 0 0 4px var(--text-primary); }
    100% { transform: scale(1); opacity: 0.8; }
  }
  .tiling-cell-actions { display:flex; gap:1px; }
  .tiling-cell-btn { display:flex; align-items:center; justify-content:center; background:none; border:none; color:var(--text-muted); padding:1px 3px; cursor:pointer; border-radius:3px; }
  .tiling-cell-btn:hover { color:var(--text-primary); background:var(--bg-hover); }
  .tiling-cell-body { flex:1; overflow:hidden; }
</style>
