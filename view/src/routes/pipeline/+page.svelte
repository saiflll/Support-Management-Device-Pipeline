<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    getForwarderStatus,
    getPipelines,
    createPipeline,
    deletePipeline,
    formatBytes,
    getMonitorStatus,
    formatSpeed,
    formatUptime,
  } from "$lib/api";

  // Pipeline states
  let pipelines: any[] = $state([]);
  let buffer: any[] = $state([]);
  let showModal = $state(false);
  let showPacketModal = $state(false);
  let selectedPacket: any = $state(null);
  let toasts: { id: number; type: string; msg: string }[] = $state([]);
  let toastId = 0;

  // Monitor states
  let stats: any = $state(null);
  let monError = $state("");

  let form = $state({
    name: "",
    source_topic: "",
    broker_url: "",
    dest_topic: "",
    username: "",
    password: "",
    interval_minutes: 8,
    is_active: true,
  });

  let pollInterval: ReturnType<typeof setInterval>;

  onMount(() => {
    fetchStatus();
    pollInterval = setInterval(fetchStatus, 3000);
  });
  onDestroy(() => clearInterval(pollInterval));

  async function fetchStatus() {
    try {
      const data = await getForwarderStatus();
      pipelines = data.pipelines || [];
      buffer = [...(data.ReceivedDataBuffer || [])].reverse().slice(0, 12);
    } catch {}

    // Fetch Monitor Status
    try {
      stats = await getMonitorStatus();
      monError = "";
    } catch (e: any) {
      monError = e.message;
    }
  }

  // --- Monitor Formatters ---
  function cpuColor(v: number) {
    if (v > 85) return "var(--red)";
    if (v > 60) return "var(--yellow)";
    return "var(--accent-2)";
  }

  function ramColor(v: number) {
    if (v > 85) return "var(--red)";
    if (v > 60) return "var(--yellow)";
    return "var(--purple)";
  }

  function diskColor(v: number) {
    if (v > 85) return "var(--red)";
    if (v > 70) return "var(--yellow)";
    return "var(--green)";
  }

  function gb(bytes: number) {
    return (bytes / 1024 / 1024 / 1024).toFixed(2) + " GB";
  }

  function toast(msg: string, type = "info") {
    const id = ++toastId;
    toasts = [...toasts, { id, type, msg }];
    setTimeout(() => {
      toasts = toasts.filter((t) => t.id !== id);
    }, 4000);
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    try {
      await createPipeline({
        ...form,
        interval_minutes: Number(form.interval_minutes),
      });
      toast("Pipeline baru ditambahkan", "success");
      showModal = false;
      resetForm();
      fetchStatus();
    } catch (err: any) {
      toast("Error: " + err.message, "error");
    }
  }

  async function handleDelete(id: number) {
    if (!confirm("Hapus pipeline ini?")) return;
    try {
      await deletePipeline(id);
      toast("Pipeline dihapus", "success");
      fetchStatus();
    } catch (err: any) {
      toast("Error: " + err.message, "error");
    }
  }

  function resetForm() {
    form = {
      name: "",
      source_topic: "",
      broker_url: "",
      dest_topic: "",
      username: "",
      password: "",
      interval_minutes: 8,
      is_active: true,
    };
  }
</script>

<!-- Toast -->
<div class="toast-container">
  {#each toasts as t (t.id)}
    <div class="toast {t.type}">{t.msg}</div>
  {/each}
</div>

<!-- ======================= -->
<!-- MONITORING SECTION      -->
<!-- ======================= -->
<div class="flex items-center justify-between mb-4">
  <div>
    <div class="card-label">SYSTEM_MONITOR</div>
    <div class="mono text-xs text-green" style="margin-top:4px;">
      [ HOST_RESOURCE_USAGE ]
    </div>
  </div>
</div>

{#if monError}
  <div class="card" style="margin-bottom:24px;">
    <div class="mono text-xs text-red" style="padding:20px;">
      [MONITOR_SERVICE_UNAVAILABLE] — {monError}
    </div>
  </div>
{:else if !stats}
  <div class="loading-state" style="margin-bottom:24px;">
    <div class="spinner"></div>
    <span>CONNECTING_TO_MONITOR...</span>
  </div>
{:else}
  <!-- Metric Cards -->
  <div class="grid-4" style="margin-bottom:24px;">
    <!-- CPU -->
    <div
      class="card"
      style="border-left:3px solid {cpuColor(stats.cpu_usage)};"
    >
      <div class="flex justify-between items-center">
        <span class="card-label">CPU_USAGE</span>
        <span
          class="mono"
          style="font-size:18px;font-weight:700;color:{cpuColor(
            stats.cpu_usage,
          )};">{stats.cpu_usage.toFixed(1)}%</span
        >
      </div>
      <div class="progress-bar" style="margin-top:12px;">
        <div
          class="progress-fill"
          style="width:{stats.cpu_usage}%;background:{cpuColor(
            stats.cpu_usage,
          )};"
        ></div>
      </div>
      <div class="flex justify-between" style="margin-top:8px;">
        <span class="mono text-xs text-muted">LOAD_AVG</span>
        <span class="mono text-xs"
          >{stats.load1.toFixed(2)}
          {stats.load5.toFixed(2)}
          {stats.load15.toFixed(2)}</span
        >
      </div>
    </div>

    <!-- RAM -->
    <div
      class="card"
      style="border-left:3px solid {ramColor(stats.ram_percent)};"
    >
      <div class="flex justify-between items-center">
        <span class="card-label">RAM_USAGE</span>
        <span
          class="mono"
          style="font-size:18px;font-weight:700;color:{ramColor(
            stats.ram_percent,
          )};">{stats.ram_percent.toFixed(1)}%</span
        >
      </div>
      <div class="progress-bar" style="margin-top:12px;">
        <div
          class="progress-fill"
          style="width:{stats.ram_percent}%;background:{ramColor(
            stats.ram_percent,
          )};"
        ></div>
      </div>
      <div class="flex justify-between" style="margin-top:8px;">
        <span class="mono text-xs" style="color:{ramColor(stats.ram_percent)};"
          >{gb(stats.ram_used)}</span
        >
        <span class="mono text-xs text-muted">/ {gb(stats.ram_total)}</span>
      </div>
    </div>

    <!-- Disk -->
    <div
      class="card"
      style="border-left:3px solid {diskColor(stats.disk_percent)};"
    >
      <div class="flex justify-between items-center">
        <span class="card-label">DISK_STORAGE</span>
        <span
          class="mono"
          style="font-size:18px;font-weight:700;color:{diskColor(
            stats.disk_percent,
          )};">{stats.disk_percent.toFixed(1)}%</span
        >
      </div>
      <div class="progress-bar" style="margin-top:12px;">
        <div
          class="progress-fill"
          style="width:{stats.disk_percent}%;background:{diskColor(
            stats.disk_percent,
          )};"
        ></div>
      </div>
      <div class="flex justify-between" style="margin-top:8px;">
        <span class="mono text-xs text-green">{gb(stats.disk_used)}</span>
        <span class="mono text-xs text-muted">/ {gb(stats.disk_total)}</span>
      </div>
    </div>

    <!-- Network -->
    <div class="card" style="border-left:3px solid var(--orange);">
      <div class="flex justify-between items-center">
        <span class="card-label">NETWORK_IO</span>
        <span class="mono text-xs" style="color:var(--orange);">BPS</span>
      </div>
      <div style="margin-top:14px;display:flex;flex-direction:column;gap:8px;">
        <div class="flex justify-between">
          <span class="mono text-xs text-muted">↑ UPLOAD:</span>
          <span class="mono text-xs" style="color:var(--orange);"
            >{formatSpeed(stats.net_sent)}</span
          >
        </div>
        <div class="flex justify-between">
          <span class="mono text-xs text-muted">↓ DOWNLOAD:</span>
          <span class="mono text-xs" style="color:var(--orange);"
            >{formatSpeed(stats.net_recv)}</span
          >
        </div>
      </div>
    </div>
  </div>

  <!-- Server Info -->
  <div class="card" style="margin-bottom:32px;">
    <div class="card-header">
      <span class="card-label">SERVER_IDENTITY</span>
      <span class="mono text-xs text-green">● CONNECTED</span>
    </div>
    <div class="grid-4">
      <div>
        <div class="card-label" style="margin-bottom:4px;">HOSTNAME</div>
        <div class="mono text-xs">{stats.hostname}</div>
      </div>
      <div>
        <div class="card-label" style="margin-bottom:4px;">OS_RUNTIME</div>
        <div class="mono text-xs">{stats.os || "-"}</div>
      </div>
      <div>
        <div class="card-label" style="margin-bottom:4px;">UPTIME</div>
        <div class="mono text-xs text-accent">{formatUptime(stats.uptime)}</div>
      </div>
      <div>
        <div class="card-label" style="margin-bottom:4px;">LAST_POLL</div>
        <div class="mono text-xs text-green">{stats.timestamp}</div>
      </div>
    </div>
  </div>
{/if}

<!-- ======================= -->
<!-- PIPELINE SECTION      -->
<!-- ======================= -->
<div class="flex items-center justify-between mb-4">
  <div>
    <div class="card-label">AGGREGATION_ENGINE</div>
    <div class="mono text-xs text-green" style="margin-top:4px;">
      [ DYN_PIPELINES_ACTIVE ]
    </div>
  </div>
  <button class="btn btn-primary" onclick={() => (showModal = true)}
    >+ NEW_FORWARD_RULE</button
  >
</div>

<!-- Pipeline Cards -->
{#if pipelines.length === 0}
  <div class="card" style="margin-bottom:24px;">
    <div
      style="padding:40px;text-align:center;color:var(--text-muted);font-family:var(--font-mono);font-size:11px;"
    >
      [NO_ACTIVE_PIPELINES_IN_REGISTRY]
    </div>
  </div>
{:else}
  <div class="grid-3" style="margin-bottom:24px;">
    {#each pipelines as p}
      <div
        class="pipeline-card"
        class:ok={p.last_forward_status === "Sukses"}
        class:err={p.last_forward_status !== "Sukses"}
      >
        <button
          onclick={() => handleDelete(p.id)}
          style="position:absolute;top:8px;right:8px;background:none;border:none;color:var(--text-dim);cursor:pointer;font-size:14px;opacity:0;"
          class="del-btn">✕</button
        >
        <div
          class="flex items-center justify-between"
          style="margin-bottom:10px;"
        >
          <div>
            <div
              class="mono text-sm"
              style="color:var(--text);font-weight:700;"
            >
              {p.source_topic}
            </div>
            <div class="mono text-xs text-accent" style="margin-top:2px;">
              ▶ {p.dest_topic}
            </div>
          </div>
          <span
            class="badge"
            class:badge-green={p.last_forward_status === "Sukses"}
            class:badge-red={p.last_forward_status !== "Sukses"}
          >
            {p.last_forward_status || "UNKNOWN"}
          </span>
        </div>
        <div
          style="border-top:1px solid var(--border);padding-top:10px;display:flex;flex-direction:column;gap:4px;"
        >
          <div class="flex justify-between text-xs">
            <span class="text-muted">BROKER:</span>
            <span
              class="mono text-accent"
              style="max-width:140px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;"
              title={p.broker_url}>{p.broker_url}</span
            >
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-muted">QUEUE:</span>
            <span class="mono text-green"
              >{p.buffer_item_count} pkts / {formatBytes(
                p.buffer_size || 0,
              )}</span
            >
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-muted">NEXT_SYNC:</span>
            <span class="mono text-yellow"
              >{p.next_forward_time
                ? new Date(p.next_forward_time).toLocaleTimeString()
                : "-"}</span
            >
          </div>
        </div>
        <button
          onclick={() => handleDelete(p.id)}
          class="btn btn-danger"
          style="width:100%;margin-top:10px;font-size:9px;"
          >DELETE PIPELINE</button
        >
      </div>
    {/each}
  </div>
{/if}

<!-- Stream Buffer -->
<div>
  <div class="card-label" style="margin-bottom:12px;">
    TELEMETRY_PIPELINE_STREAM:
  </div>
  {#if buffer.length === 0}
    <div class="loading-state">[AWAITING_DATA_PACKETS]</div>
  {:else}
    <div class="grid-3">
      {#each buffer as item, i}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="card"
          style="cursor:pointer;"
          onmouseenter={(e: any) =>
            (e.currentTarget.style.borderColor = "var(--accent)")}
          onmouseleave={(e: any) =>
            (e.currentTarget.style.borderColor = "var(--border)")}
          onclick={() => {
            selectedPacket = item;
            showPacketModal = true;
          }}
        >
          <div
            class="flex items-center justify-between"
            style="margin-bottom:8px;"
          >
            <span class="text-xs text-green mono">✓ PACKET</span>
            <span class="text-xs text-muted mono"
              >{new Date().toLocaleTimeString()}</span
            >
          </div>
          <div
            class="flex items-center justify-between mono text-xs"
            style="border-top:1px solid var(--border);padding-top:8px;"
          >
            <span class="text-accent">CK:{item.ck} / AREA:{item.area}</span>
            <span class="text-muted"
              >T:{item.temp?.length || 0} D:{item.door?.length || 0}</span
            >
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- New Pipeline Modal -->
{#if showModal}
  <div
    class="modal-overlay"
    onclick={() => (showModal = false)}
    role="button"
    tabindex="-1"
  >
    <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog">
      <div class="modal-header">
        <span class="modal-title">[ PIPELINE_STRUCT_EDITOR ]</span>
        <button
          onclick={() => (showModal = false)}
          style="color:var(--text-muted);font-size:16px;">✕</button
        >
      </div>
      <form onsubmit={handleSubmit}>
        <div class="modal-body">
          <div class="grid-2">
            <div class="form-group" style="grid-column:span 2;">
              <label class="form-label">PIPELINE_LABEL (OPT)</label>
              <input
                type="text"
                bind:value={form.name}
                placeholder="e.g. Jakarta Site A"
                class="form-input"
              />
            </div>
            <div class="form-group" style="grid-column:span 2;">
              <label class="form-label">SOURCE_MQTT_TOPIC</label>
              <input
                type="text"
                bind:value={form.source_topic}
                required
                placeholder="sensors/v2/data/+"
                class="form-input"
              />
            </div>
            <div class="form-group" style="grid-column:span 2;">
              <label class="form-label">DEST_BROKER_URL</label>
              <input
                type="text"
                bind:value={form.broker_url}
                required
                placeholder="tcp://broker.emqx.io:1883"
                class="form-input"
              />
            </div>
            <div class="form-group" style="grid-column:span 2;">
              <label class="form-label">TARGET_MQTT_TOPIC</label>
              <input
                type="text"
                bind:value={form.dest_topic}
                required
                placeholder="cloud/forward/data"
                class="form-input"
              />
            </div>
            <div class="form-group">
              <label class="form-label">USERNAME (OPT)</label>
              <input
                type="text"
                bind:value={form.username}
                class="form-input"
              />
            </div>
            <div class="form-group">
              <label class="form-label">PASSWORD (OPT)</label>
              <input
                type="password"
                bind:value={form.password}
                class="form-input"
              />
            </div>
            <div class="form-group" style="grid-column:span 2;">
              <label class="form-label">SYNC_INTERVAL (MENIT)</label>
              <input
                type="number"
                bind:value={form.interval_minutes}
                min="1"
                required
                class="form-input"
              />
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button
            type="button"
            class="btn btn-ghost"
            onclick={() => (showModal = false)}>CANCEL</button
          >
          <button type="submit" class="btn btn-success"
            >SAVE_PIPELINE_POLICY</button
          >
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Packet Data Modal -->
{#if showPacketModal && selectedPacket}
  <div
    class="modal-overlay"
    onclick={() => (showPacketModal = false)}
    role="button"
    tabindex="-1"
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="modal"
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      style="max-width: 600px;"
    >
      <div class="modal-header">
        <span class="modal-title">[ PACKET_DATA_MODEL ]</span>
        <button
          onclick={() => (showPacketModal = false)}
          style="color:var(--text-muted);font-size:16px;cursor:pointer;background:none;border:none;"
          >✕</button
        >
      </div>
      <div
        class="modal-body"
        style="background:#0a0a0a; border: 1px solid var(--border); border-radius: 4px; max-height: 400px; overflow-y: auto; padding: 16px;"
      >
        <pre
          class="mono text-xs"
          style="color:var(--green); margin:0;">{JSON.stringify(
            selectedPacket,
            null,
            2,
          )}</pre>
      </div>
    </div>
  </div>
{/if}
