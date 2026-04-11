<script lang="ts">
  import { auth, currentUser } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();
  let authorized = $state(false);

  onMount(() => {
    const unsub = auth.subscribe((state) => {
      if (!state.loading) {
        if (!state.user || state.user.role !== 'admin') {
          goto('/dashboard');
        } else {
          authorized = true;
        }
      }
    });
    return unsub;
  });
</script>

{#if authorized}
  <div class="admin-shell">
    <div class="admin-banner">
      <span class="banner-icon">&#9888;</span>
      <span>Admin access</span>
      <span class="banner-icon">&#9888;</span>
    </div>
    <nav class="admin-nav">
      <a href="/admin" class="admin-nav-link">Overview</a>
      <a href="/admin/challenges" class="admin-nav-link">Challenge Forge</a>
      <a href="/admin/lessons" class="admin-nav-link">Academy Publisher</a>
      <a href="/admin/community" class="admin-nav-link">Community Queue</a>
    </nav>
    {@render children()}
  </div>
{:else}
  <div class="admin-denied">
    <span>Access denied. Insufficient privileges.</span>
  </div>
{/if}

<style>
  .admin-shell {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .admin-banner {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    background: var(--bg-tertiary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    font-family: var(--font-sans);
    color: var(--text-secondary);
    font-size: 0.7rem;
  }

  .banner-icon {
    font-size: 0.85rem;
  }

  .admin-nav {
    display: flex;
    gap: var(--space-1);
    border-bottom: 1px solid var(--border-primary);
    padding-bottom: var(--space-2);
  }

  .admin-nav-link {
    font-family: var(--font-sans);
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    text-decoration: none;
    transition: all var(--transition-fast);
  }

  .admin-nav-link:hover {
    color: var(--text-primary);
    background: var(--bg-tertiary);
  }

  .admin-denied {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 40vh;
    color: var(--accent-red);
    font-size: 0.9rem;
  }
</style>
