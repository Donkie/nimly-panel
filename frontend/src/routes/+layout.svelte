<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import '../app.css';
  import { app, bootstrap } from '$lib/store.svelte.js';

  let { children } = $props();

  onMount(() => {
    bootstrap();
  });

  const TABS = [
    { href: '/',         label: 'Lock',     icon: 'lock'     },
    { href: '/pins',     label: 'PINs',     icon: 'keypad'   },
    { href: '/settings', label: 'Settings', icon: 'settings' },
    { href: '/activity', label: 'Activity', icon: 'activity' },
  ];
</script>

{#if !app.ready}
  <div class="loading-screen">
    <div class="loading-lockup">
      <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
        <rect width="32" height="32" rx="8" fill="#3d7ef6" opacity="0.15"/>
        <path d="M10 15V11a6 6 0 0 1 12 0v4" stroke="#3d7ef6" stroke-width="2.5" stroke-linecap="round"/>
        <rect x="7" y="15" width="18" height="12" rx="3" fill="#3d7ef6" opacity="0.9"/>
      </svg>
      <div class="loading-wordmark">nimly<strong>panel</strong></div>
      <div class="loading-sub">Smart lock admin</div>
    </div>
    <div class="spinner"></div>
  </div>
{:else}
  <div class="app-shell">
    <header class="top-bar">
      <span class="app-wordmark">nimly<strong>panel</strong></span>
      <span class="conn-badge" class:live={app.connected} title={app.connected ? 'Live' : 'Offline'}>
        <span class="conn-dot"></span>
        {app.connected ? 'Live' : 'Offline'}
      </span>
    </header>

    <main class="main-scroll">
      {@render children()}
    </main>

    <nav class="bottom-nav" aria-label="Main navigation">
      {#each TABS as tab}
        <a
          href={tab.href}
          class="nav-tab"
          class:active={$page.url.pathname === tab.href}
          aria-label={tab.label}
          aria-current={$page.url.pathname === tab.href ? 'page' : undefined}
        >
          <span class="nav-tab-icon" aria-hidden="true">
            {#if tab.icon === 'lock'}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {:else if tab.icon === 'keypad'}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="3" width="7" height="7" rx="1.5"/>
                <rect x="14" y="3" width="7" height="7" rx="1.5"/>
                <rect x="3" y="14" width="7" height="7" rx="1.5"/>
                <rect x="14" y="14" width="7" height="7" rx="1.5"/>
              </svg>
            {:else if tab.icon === 'settings'}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
              </svg>
            {:else if tab.icon === 'activity'}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
            {/if}
          </span>
          <span class="nav-tab-label">{tab.label}</span>
        </a>
      {/each}
    </nav>

    {#if app.toast}
      <div
        class="toast"
        class:toast-error={app.toast.isError}
        role="status"
        aria-live="polite"
      >
        {app.toast.message}
      </div>
    {/if}
  </div>
{/if}
