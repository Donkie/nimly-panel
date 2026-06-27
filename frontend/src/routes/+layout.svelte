<script>
  import '../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { app, init } from '$lib/store.svelte.js';

  let { children } = $props();

  onMount(init);

  const nav = [
    { href: '/', label: 'Lock', ico: '🔒' },
    { href: '/pins', label: 'PIN codes', ico: '🔢' },
    { href: '/settings', label: 'Settings', ico: '⚙️' },
    { href: '/activity', label: 'Activity', ico: '📜' }
  ];

  let path = $derived($page.url.pathname);
</script>

{#if !app.ready}
  <div class="loading-screen">Loading…</div>
{:else}
  <div class="app">
    <header class="topbar">
      <h1>Nimly Lock</h1>
      <div class="status">
        <span class="dot {app.connected ? 'on' : 'off'}"></span>
        {app.connected ? 'live' : 'offline'}
      </div>
    </header>

    <main class="content">
      {@render children()}
    </main>

    <nav class="bottom-nav">
      {#each nav as item}
        <a href={item.href} class:active={path === item.href} aria-current={path === item.href ? 'page' : undefined}>
          <span class="ico">{item.ico}</span>
          {item.label}
        </a>
      {/each}
    </nav>
  </div>
{/if}

{#if app.toast}
  <div class="toast" class:error={app.toast.isError}>{app.toast.message}</div>
{/if}
