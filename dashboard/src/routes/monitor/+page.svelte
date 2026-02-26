<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getMonitorStatus, formatSpeed, formatUptime } from '$lib/api';

  let stats: any = $state(null);
  let pollInterval: ReturnType<typeof setInterval>;
  let error = $state('');

  onMount(() => {
    fetchStats();
    pollInterval = setInterval(fetchStats, 2000);
  });
  onDestroy(() => clearInterval(pollInterval));

  async function fetchStats() {
    try {
      stats = await getMonitorStatus();
      error = '';
    } catch (e: any) {
      error = e.message;
    }
  }

  function cpuColor(v: number) {
    if (v > 85) return 'var(--red)';
    if (v > 60) return 'var(--yellow)';
    return 'var(--accent-2)';
  }

  function ramColor(v: number) {
    if (v > 85) return 'var(--red)';
    if (v > 60) return 'var(--yellow)';
    return 'var(--purple)';
  }

  function diskColor(v: number) {
    if (v > 85) return 'var(--red)';
    if (v > 70) return 'var(--yellow)';
    return 'var(--green)';
  }

  function gb(bytes: number) {
    return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB';
  }
</script>

{#if error}
  <div class="card">
    <div class="mono text-xs text-red" style="padding:20px;">[MONITOR_SERVICE_UNAVAILABLE] — {error}</div>
  </div>
{:else if !stats}
  <div class="loading-state">
    <div class="spinner"></div>
    <span>CONNECTING_TO_MONITOR...</span>
  </div>
{:else}
  <!-- Metric Cards -->
  <div class="grid-4" style="margin-bottom:20px;">
    <!-- CPU -->
    <div class="card" style="border-left:3px solid {cpuColor(stats.cpu_usage)};">
      <div class="flex justify-between items-center">
        <span class="card-label">CPU_USAGE</span>
        <span class="mono" style="font-size:18px;font-weight:700;color:{cpuColor(stats.cpu_usage)};">{stats.cpu_usage.toFixed(1)}%</span>
      </div>
      <div class="progress-bar" style="margin-top:12px;">
        <div class="progress-fill" style="width:{stats.cpu_usage}%;background:{cpuColor(stats.cpu_usage)};"></div>
      </div>
      <div class="flex justify-between" style="margin-top:8px;">
        <span class="mono text-xs text-muted">LOAD_AVG</span>
        <span class="mono text-xs">{stats.load1.toFixed(2)} {stats.load5.toFixed(2)} {stats.load15.toFixed(2)}</span>
      </div>
    </div>

    <!-- RAM -->
    <div class="card" style="border-left:3px solid {ramColor(stats.ram_percent)};">
      <div class="flex justify-between items-center">
        <span class="card-label">RAM_USAGE</span>
        <span class="mono" style="font-size:18px;font-weight:700;color:{ramColor(stats.ram_percent)};">{stats.ram_percent.toFixed(1)}%</span>
      </div>
      <div class="progress-bar" style="margin-top:12px;">
        <div class="progress-fill" style="width:{stats.ram_percent}%;background:{ramColor(stats.ram_percent)};"></div>
      </div>
      <div class="flex justify-between" style="margin-top:8px;">
        <span class="mono text-xs" style="color:{ramColor(stats.ram_percent)};">{gb(stats.ram_used)}</span>
        <span class="mono text-xs text-muted">/ {gb(stats.ram_total)}</span>
      </div>
    </div>

    <!-- Disk -->
    <div class="card" style="border-left:3px solid {diskColor(stats.disk_percent)};">
      <div class="flex justify-between items-center">
        <span class="card-label">DISK_STORAGE</span>
        <span class="mono" style="font-size:18px;font-weight:700;color:{diskColor(stats.disk_percent)};">{stats.disk_percent.toFixed(1)}%</span>
      </div>
      <div class="progress-bar" style="margin-top:12px;">
        <div class="progress-fill" style="width:{stats.disk_percent}%;background:{diskColor(stats.disk_percent)};"></div>
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
          <span class="mono text-xs" style="color:var(--orange);">{formatSpeed(stats.net_sent)}</span>
        </div>
        <div class="flex justify-between">
          <span class="mono text-xs text-muted">↓ DOWNLOAD:</span>
          <span class="mono text-xs" style="color:var(--orange);">{formatSpeed(stats.net_recv)}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- Server Info -->
  <div class="card">
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
        <div class="mono text-xs">{stats.os || '-'}</div>
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
