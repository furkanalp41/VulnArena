<script lang="ts">
  import { auth, isAuthenticated } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();
  let ready = $state(false);

  onMount(() => {
    const unsub = auth.subscribe((state) => {
      if (!state.loading && !state.user) {
        goto('/login');
      } else if (!state.loading) {
        ready = true;
      }
    });
    return unsub;
  });
</script>

{#if ready}
  {@render children()}
{:else}
  <div class="loading-screen">
    <span class="loading-text font-mono">Initializing...</span>
  </div>
{/if}

<style>
  .loading-screen {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 50vh;
  }

  .loading-text {
    color: var(--text-tertiary);
    font-size: 0.875rem;
    letter-spacing: 0.05em;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
</style>
