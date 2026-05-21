<script lang="ts">
  import { invoke } from "@tauri-apps/api/core";
  import type { Agent, ChatMessage } from "../stores.svelte";

  let agents = $state<Agent[]>([]);
  let messages = $state<ChatMessage[]>([]);
  let input = $state("");
  let selectedMode = $state("auto");
  let selectedAgentId = $state("coder");
  let isStreaming = $state(false);
  let taskType = $state("");
  let routedAgents = $state<Agent[]>([]);
  let agentList = $state<{ id: string; label: string }[]>([]);

  async function loadAgents() {
    agents = await invoke<Agent[]>("ai_list_agents");
    agentList = [
      { id: "auto", label: "Auto (default)" },
      ...agents.map((a) => ({ id: a.id, label: `${a.name} (${a.provider})` })),
    ];
  }

  async function sendMessage() {
    if (!input.trim() || isStreaming) return;
    const userMsg: ChatMessage = { role: "user", content: input };
    messages = [...messages, userMsg];
    input = "";
    isStreaming = true;
    try {
      if (selectedMode === "auto") {
        const matched = await invoke<Agent[]>("ai_auto_route", { taskType: userMsg.content });
        if (matched.length === 0) {
          messages = [...messages, { role: "assistant", content: "No suitable agent found for this task." }];
          isStreaming = false;
          return;
        }
        const agent = matched[0];
        const response = await invoke<{ agent_id: string; content: string; provider: string; model: string }>("ai_chat", {
          agentId: agent.id,
          messages: messages.concat([userMsg]).map((m) => ({ role: m.role, content: m.content })),
        });
        messages = [...messages, { role: "assistant", content: `[Auto → ${response.provider}/${response.model}]\n${response.content}` }];
      } else {
        const response = await invoke<{ agent_id: string; content: string; provider: string; model: string }>("ai_chat", {
          agentId: selectedAgentId,
          messages: [...messages, userMsg].map((m) => ({ role: m.role, content: m.content })),
        });
        messages = [...messages, { role: "assistant", content: `[${response.provider}/${response.model}]\n${response.content}` }];
      }
    } catch (e: any) {
      messages = [...messages, { role: "assistant", content: `Error: ${e}` }];
    }
    isStreaming = false;
  }

  function doAutoRoute() {
    if (!taskType.trim()) return;
    invoke<Agent[]>("ai_auto_route", { taskType }).then((matched) => {
      routedAgents = matched;
      if (matched.length > 0) {
        selectedMode = matched[0].id;
        selectedAgentId = matched[0].id;
        messages = [...messages, { role: "system", content: `Auto-routed "${taskType}" → ${matched.map((a) => a.name).join(", ")}` }];
      }
    });
    taskType = "";
  }

  function handleModeChange(val: string) {
    selectedMode = val;
    if (val !== "auto") {
      selectedAgentId = val;
    }
  }

  function clearChat() { messages = []; }
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendMessage(); }
  }

  $effect(() => { loadAgents(); });
</script>

<div class="ai-chat">
  <div class="ai-toolbar">
    <div class="ai-agent-select">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="8" r="4"/><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/></svg>
      <select bind:value={selectedMode} onchange={(e) => handleModeChange((e.target as HTMLSelectElement).value)}>
        <option value="auto">Auto (default)</option>
        {#each agents as agent}
          <option value={agent.id}>{agent.name} ({agent.provider})</option>
        {/each}
      </select>
    </div>
    <button class="ai-btn ai-btn-ghost" onclick={clearChat} title="Clear chat">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
    </button>
  </div>

  <div class="ai-messages">
    {#each messages as msg}
      <div class="msg" class:msg-user={msg.role === "user"} class:msg-system={msg.role === "system"} class:msg-assistant={msg.role === "assistant"}>
        <div class="msg-badge">{msg.role}</div>
        <div class="msg-text">{msg.content}</div>
      </div>
    {/each}
    {#if isStreaming}
      <div class="msg msg-assistant">
        <div class="msg-badge">assistant</div>
        <div class="msg-text streaming">Thinking<span class="dots"><span>.</span><span>.</span><span>.</span></span></div>
      </div>
    {/if}
  </div>

  <div class="ai-input">
    <textarea bind:value={input} placeholder="Type a message..." onkeydown={handleKeydown} rows={2}></textarea>
    <button class="ai-btn ai-btn-send" onclick={sendMessage} disabled={!input.trim() || isStreaming}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
    </button>
  </div>
</div>

<style>
  .ai-chat { display:flex; flex-direction:column; height:100%; background:transparent; color:var(--text-primary); font-size:var(--font-size); }
  .ai-toolbar { display:flex; align-items:center; gap:6px; padding:6px 10px; border-bottom:1px solid var(--border-subtle); flex-shrink:0; }
  .ai-agent-select { display:flex; align-items:center; gap:6px; flex:1; color:var(--text-muted); }
  .ai-agent-select select { flex:1; background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:4px; padding:3px 6px; font-size:var(--fs-11); }
  .ai-btn { display:inline-flex; align-items:center; justify-content:center; gap:4px; border:none; border-radius:4px; padding:4px 10px; font-size:var(--fs-11); cursor:pointer; transition:all 0.12s ease; font-weight:500; }
  .ai-btn-ghost { background:none; color:var(--text-muted); padding:4px; }
  .ai-btn-ghost:hover { color:var(--accent-red); background:var(--bg-hover); }
  .ai-btn-send { background:var(--accent-blue); color:var(--bg-primary); padding:6px; border-radius:6px; align-self:flex-end; }
  .ai-btn-send:disabled { opacity:0.3; cursor:not-allowed; }
  .ai-btn-send:hover:not(:disabled) { filter:brightness(1.1); }
  .ai-messages { flex:1; overflow-y:auto; padding:8px 10px; display:flex; flex-direction:column; gap:8px; }
  .msg { padding:8px 12px; border-radius:8px; max-width:92%; word-wrap:break-word; white-space:pre-wrap; animation:fadeIn 0.15s ease; }
  @keyframes fadeIn { from { opacity:0; transform:translateY(4px); } to { opacity:1; transform:translateY(0); } }
  .msg-user { background:var(--accent-blue); color:var(--bg-primary); align-self:flex-end; }
  .msg-assistant { background:var(--bg-surface); border:1px solid var(--border-subtle); align-self:flex-start; }
  .msg-system { background:var(--bg-elevated); color:var(--text-muted); align-self:center; font-size:var(--fs-10); font-style:italic; padding:4px 12px; border-radius:12px; }
  .msg-badge { font-size:var(--fs-9); text-transform:uppercase; opacity:0.5; margin-bottom:4px; font-weight:600; letter-spacing:0.5px; }
  .msg-text { font-size:var(--font-size); line-height:1.6; }
  .streaming { opacity:0.7; }
  .dots span { animation:dotAnim 1.4s infinite; opacity:0; }
  .dots span:nth-child(2) { animation-delay:0.2s; }
  .dots span:nth-child(3) { animation-delay:0.4s; }
  @keyframes dotAnim { 0%,60%,100% { opacity:0; } 30% { opacity:1; } }
  .ai-input { display:flex; gap:6px; padding:8px 10px; border-top:1px solid var(--border-subtle); flex-shrink:0; }
  .ai-input textarea { flex:1; background:var(--bg-primary); color:var(--text-primary); border:1px solid var(--border-subtle); border-radius:8px; padding:8px; font-size:var(--font-size); resize:none; }
  .ai-input textarea:focus { outline:none; border-color:var(--accent-blue); }
  .ai-messages::-webkit-scrollbar { width:4px; }
  .ai-messages::-webkit-scrollbar-thumb { background:var(--bg-hover); border-radius:2px; }
</style>
