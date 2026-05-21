<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";
  import { currentDir } from "../stores.svelte";

  let workspacePath = $state("");
  let branchName = $state("detached");
  let untracked = $state<string[]>([]);
  let modified = $state<string[]>([]);
  let staged = $state<string[]>([]);
  let commitMessage = $state("");
  let isLoading = $state(false);
  let commitStatus = $state<string | null>(null);

  // Subscribe to workspace path changes
  $effect(() => {
    const unsub = currentDir.subscribe((val) => {
      workspacePath = val;
      if (val) {
        refreshStatus();
      }
    });
    return unsub;
  });

  async function refreshStatus() {
    if (!workspacePath) return;
    isLoading = true;
    commitStatus = null;
    try {
      const res = await invoke<{
        branch: string;
        untracked: string[];
        modified: string[];
        staged: string[];
      }>("git_get_status", { repoPath: workspacePath });
      
      branchName = res.branch;
      untracked = res.untracked;
      modified = res.modified;
      staged = res.staged;
    } catch (e) {
      console.error("Failed to check Git status:", e);
      branchName = "no git repo";
      untracked = [];
      modified = [];
      staged = [];
    }
    isLoading = false;
  }

  async function handleCommit() {
    if (!workspacePath || !commitMessage.trim()) return;
    isLoading = true;
    commitStatus = "Committing...";
    try {
      await invoke<string>("git_commit", {
        repoPath: workspacePath,
        message: commitMessage.trim(),
      });
      commitStatus = "Committed successfully!";
      commitMessage = "";
      await refreshStatus();
    } catch (e: any) {
      console.error("Failed to commit changes:", e);
      commitStatus = `Error: ${e.toString()}`;
    }
    isLoading = false;
  }

  async function stageFile(file: string) {
    if (!workspacePath) return;
    isLoading = true;
    try {
      await invoke("git_stage_file", { repoPath: workspacePath, filePath: file });
      await refreshStatus();
    } catch (e) {
      console.error(e);
    }
    isLoading = false;
  }

  async function unstageFile(file: string) {
    if (!workspacePath) return;
    isLoading = true;
    try {
      await invoke("git_unstage_file", { repoPath: workspacePath, filePath: file });
      await refreshStatus();
    } catch (e) {
      console.error(e);
    }
    isLoading = false;
  }

  async function discardFile(file: string) {
    if (!workspacePath) return;
    if (!confirm(`Are you sure you want to discard changes for "${file}"? This action cannot be undone.`)) return;
    isLoading = true;
    try {
      await invoke("git_discard_file", { repoPath: workspacePath, filePath: file });
      await refreshStatus();
    } catch (e) {
      console.error(e);
    }
    isLoading = false;
  }

  async function stageAll() {
    if (!workspacePath) return;
    isLoading = true;
    try {
      await invoke("git_stage_all", { repoPath: workspacePath });
      await refreshStatus();
    } catch (e) {
      console.error(e);
    }
    isLoading = false;
  }

  async function unstageAll() {
    if (!workspacePath) return;
    isLoading = true;
    try {
      await invoke("git_unstage_all", { repoPath: workspacePath });
      await refreshStatus();
    } catch (e) {
      console.error(e);
    }
    isLoading = false;
  }
</script>

<div class="git-status">
  <div class="git-header">
    <div class="git-branch-info">
      <svg class="branch-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="6" y1="3" x2="6" y2="15"></line>
        <circle cx="18" cy="6" r="3"></circle>
        <circle cx="6" cy="18" r="3"></circle>
        <path d="M18 9a9 9 0 0 1-9 9"></path>
      </svg>
      <span class="branch-name" title="Active Branch">{branchName}</span>
    </div>
    <button class="refresh-btn" onclick={refreshStatus} disabled={isLoading} title="Refresh Git Status">
      <svg class="refresh-icon" class:spinning={isLoading} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
      </svg>
    </button>
  </div>

  <div class="git-body">
    <!-- Commit Section -->
    <div class="commit-section">
      <textarea
        bind:value={commitMessage}
        class="commit-textarea"
        placeholder="Commit message (Ctrl+Enter)..."
        rows="2"
        disabled={isLoading || (untracked.length === 0 && modified.length === 0 && staged.length === 0)}
        onkeydown={(e) => {
          if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            handleCommit();
          }
        }}
      ></textarea>
      <button
        class="commit-btn"
        onclick={handleCommit}
        disabled={isLoading || !commitMessage.trim() || (untracked.length === 0 && modified.length === 0 && staged.length === 0)}
      >
        Commit to {branchName}
      </button>
      {#if commitStatus}
        <div class="commit-message-log" class:error={commitStatus.startsWith('Error')}>
          {commitStatus}
        </div>
      {/if}
    </div>

    <!-- Changes Lists -->
    <div class="changes-section">
      {#if untracked.length === 0 && modified.length === 0 && staged.length === 0}
        <div class="empty-state">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          <p>No changes detected</p>
          <span>Working tree is clean</span>
        </div>
      {:else}
        <!-- Staged Changes -->
        {#if staged.length > 0}
          <div class="changes-group">
            <div class="group-header">
              <span class="group-title">Staged Changes ({staged.length})</span>
              <button class="bulk-action-btn" onclick={unstageAll} disabled={isLoading} title="Unstage All">Unstage All</button>
            </div>
            <div class="changes-list">
              {#each staged as file}
                <div class="change-item change-staged">
                  <span class="change-badge badge-staged">A</span>
                  <span class="change-path" title={file}>{file.split('/').pop()}</span>
                  <span class="change-fullpath">{file}</span>
                  <div class="change-actions">
                    <button class="action-btn" onclick={() => unstageFile(file)} disabled={isLoading} title="Unstage File">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="5" y1="12" x2="19" y2="12"/></svg>
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Unstaged Changes (Modified) -->
        {#if modified.length > 0}
          <div class="changes-group">
            <div class="group-header">
              <span class="group-title">Modified Changes ({modified.length})</span>
              <button class="bulk-action-btn" onclick={stageAll} disabled={isLoading} title="Stage All">Stage All</button>
            </div>
            <div class="changes-list">
              {#each modified as file}
                <div class="change-item change-modified">
                  <span class="change-badge badge-modified">M</span>
                  <span class="change-path" title={file}>{file.split('/').pop()}</span>
                  <span class="change-fullpath">{file}</span>
                  <div class="change-actions">
                    <button class="action-btn" onclick={() => discardFile(file)} disabled={isLoading} title="Discard Changes">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><polyline points="3 3 3 8 8 8"/></svg>
                    </button>
                    <button class="action-btn" onclick={() => stageFile(file)} disabled={isLoading} title="Stage File">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Untracked Changes -->
        {#if untracked.length > 0}
          <div class="changes-group">
            <div class="group-header">
              <span class="group-title">Untracked Files ({untracked.length})</span>
              <button class="bulk-action-btn" onclick={stageAll} disabled={isLoading} title="Stage All">Stage All</button>
            </div>
            <div class="changes-list">
              {#each untracked as file}
                <div class="change-item change-untracked">
                  <span class="change-badge badge-untracked">U</span>
                  <span class="change-path" title={file}>{file.split('/').pop()}</span>
                  <span class="change-fullpath">{file}</span>
                  <div class="change-actions">
                    <button class="action-btn" onclick={() => discardFile(file)} disabled={isLoading} title="Delete File">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                    </button>
                    <button class="action-btn" onclick={() => stageFile(file)} disabled={isLoading} title="Stage File">
                      <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>

<style>
  .git-status {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: transparent;
    color: var(--text-primary);
    font-size: var(--font-size);
    overflow: hidden;
  }
  .git-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--bg-secondary);
    flex-shrink: 0;
  }
  .git-branch-info {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--accent-blue);
    font-weight: 500;
  }
  .branch-icon {
    flex-shrink: 0;
  }
  .branch-name {
    font-size: var(--fs-11);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 160px;
  }
  .refresh-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.12s;
  }
  .refresh-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }
  .spinning {
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .git-body {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    padding: 10px 12px;
    gap: 12px;
  }
  .git-body::-webkit-scrollbar {
    width: 4px;
  }
  .git-body::-webkit-scrollbar-thumb {
    background: var(--bg-hover);
    border-radius: 2px;
  }

  .commit-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: 8px;
    padding: 8px;
    flex-shrink: 0;
  }
  .commit-textarea {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-subtle);
    border-radius: 5px;
    padding: 6px;
    font-family: inherit;
    font-size: var(--fs-11);
    resize: none;
    outline: none;
    transition: border-color 0.12s;
  }
  .commit-textarea:focus {
    border-color: var(--accent-blue);
  }
  .commit-btn {
    background: var(--accent-blue);
    color: var(--bg-primary);
    border: none;
    border-radius: 5px;
    padding: 5px 10px;
    font-size: var(--fs-11);
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.12s;
  }
  .commit-btn:hover:not(:disabled) {
    filter: brightness(1.1);
  }
  .commit-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  .commit-message-log {
    font-size: 10.5px;
    color: var(--accent-green);
    margin-top: 2px;
    padding: 3px 6px;
    background: color-mix(in srgb, var(--accent-green) 10%, transparent);
    border-radius: 4px;
  }
  .commit-message-log.error {
    color: var(--accent-red);
    background: color-mix(in srgb, var(--accent-red) 10%, transparent);
  }

  .changes-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px 10px;
    color: var(--text-muted);
    text-align: center;
  }
  .empty-state svg {
    color: var(--accent-green);
    margin-bottom: 8px;
  }
  .empty-state p {
    font-size: var(--font-size);
    font-weight: 500;
    margin: 0;
    color: var(--text-secondary);
  }
  .empty-state span {
    font-size: var(--fs-10);
    opacity: 0.8;
  }

  .changes-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .group-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-right: 2px;
  }
  .group-title {
    font-size: var(--fs-10);
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding-left: 2px;
  }
  .bulk-action-btn {
    background: none;
    border: none;
    color: var(--accent-blue);
    font-size: var(--fs-10);
    font-weight: 600;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 3px;
    transition: all 0.12s;
  }
  .bulk-action-btn:hover:not(:disabled) {
    background: var(--bg-hover);
  }
  .bulk-action-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .changes-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .change-item {
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 8px;
    background: var(--bg-surface);
    border: 1px solid var(--border-subtle);
    border-radius: 5px;
    transition: all 0.1s;
  }
  .change-item:hover {
    border-color: var(--border-primary);
    background: var(--bg-hover);
  }
  .change-badge {
    font-size: var(--fs-9);
    font-weight: 700;
    width: 15px;
    height: 15px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
  }
  .badge-staged {
    background: color-mix(in srgb, var(--accent-green) 15%, transparent);
    color: var(--accent-green);
  }
  .badge-modified {
    background: color-mix(in srgb, var(--accent-blue) 15%, transparent);
    color: var(--accent-blue);
  }
  .badge-untracked {
    background: color-mix(in srgb, var(--text-muted) 25%, transparent);
    color: var(--text-muted);
  }
  .change-path {
    font-weight: 500;
    color: var(--text-primary);
  }
  .change-fullpath {
    font-size: var(--fs-9-5);
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    text-align: right;
    margin-left: 6px;
    direction: rtl;
    transition: opacity 0.12s;
  }
  .change-item:hover .change-fullpath {
    opacity: 0.1;
  }
  .change-actions {
    display: none;
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    gap: 2px;
    background: var(--bg-surface);
    padding: 1px;
    border-radius: 4px;
    border: 1px solid var(--border-subtle);
    box-shadow: 0 2px 6px rgba(0,0,0,0.2);
  }
  .change-item:hover .change-actions {
    display: flex;
  }
  .action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: var(--text-muted);
    width: 18px;
    height: 18px;
    cursor: pointer;
    border-radius: 3px;
    transition: all 0.1s;
  }
  .action-btn:hover:not(:disabled) {
    color: var(--text-primary);
    background: var(--bg-hover);
  }
  .action-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
