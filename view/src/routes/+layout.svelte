<script lang="ts">
  import "../app.css";
  import { page } from "$app/stores";

  const tabs = [
    { label: "[ /OTA ]", href: "/" },
  ];

  let currentPath = $derived($page.url.pathname);

  function isActive(href: string): boolean {
    if (href === "/") return currentPath === "/";
    return currentPath.startsWith(href);
  }

  interface Props {
    children: import("svelte").Snippet;
  }
  let { children }: Props = $props();
</script>

<div class="app-shell">
  <header class="topbar">
    <div class="topbar-brand">[MENT_IoT]</div>
    <nav class="topbar-nav">
      {#each tabs as tab}
        <a href={tab.href} class="nav-tab" class:active={isActive(tab.href)}>
          {tab.label}
        </a>
      {/each}
    </nav>
    <div class="topbar-right">
      <span class="live-dot">LIVE</span>
    </div>
  </header>

  <main class="content">
    {@render children()}
  </main>
</div>
