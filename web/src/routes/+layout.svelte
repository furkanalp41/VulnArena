<script lang="ts">
  import '../app.css';
  import { auth } from '$lib/stores/auth';
  import { theme } from '$lib/stores/theme';
  import { connect, disconnect } from '$lib/stores/websocket';
  import { wsEvents } from '$lib/stores/websocket';
  import { addToast } from '$lib/stores/toast';
  import Navbar from '$lib/components/layout/Navbar.svelte';
  import GlobalToast from '$lib/components/layout/GlobalToast.svelte';
  import type { Snippet } from 'svelte';
  import { onMount, onDestroy } from 'svelte';

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();

  // Bridge WS events to toast store
  const unsubWs = wsEvents.subscribe((event) => {
    if (event) {
      addToast({
        type: event.type,
        user: event.user || '',
        challenge: event.challenge,
        achievement: event.achievement,
      });
    }
  });

  onMount(() => {
    theme.initialize();
    auth.initialize();
    connect();
  });

  onDestroy(() => {
    disconnect();
    unsubWs();
  });
</script>

<svelte:head>
  <title>VulnArena - Cybersecurity Training Platform</title>
  <meta name="description" content="Master vulnerability detection and secure coding through interactive challenges" />
</svelte:head>

<GlobalToast />

<div class="app-shell">
  <Navbar />
  <main class="main-content">
    {@render children()}
  </main>
</div>

<style>
  .app-shell {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .main-content {
    flex: 1;
    width: 100%;
    max-width: 1400px;
    margin: 0 auto;
    padding: var(--space-6);
  }
</style>
