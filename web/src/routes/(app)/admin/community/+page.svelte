<script lang="ts">
  import { onMount } from 'svelte';
  import { listCommunityQueue } from '$lib/api/admin';
  import type { CommunityChallenge } from '$lib/api/community';
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';

  let challenges = $state<CommunityChallenge[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state('');
  let activeStatus = $state<string>('pending');
  let currentPage = $state(1);

  const statuses = ['pending', 'approved', 'rejected', 'published'];

  async function loadQueue() {
    loading = true;
    error = '';
    try {
      const res = await listCommunityQueue(activeStatus, currentPage);
      challenges = res.challenges ?? [];
      total = res.total;
    } catch (e: any) {
      error = e.message || 'Failed to load queue';
    } finally {
      loading = false;
    }
  }

  onMount(() => loadQueue());

  function switchStatus(status: string) {
    activeStatus = status;
    currentPage = 1;
    loadQueue();
  }

  const statusColors: Record<string, string> = {
    pending: 'var(--accent-yellow)',
    approved: 'var(--accent-green)',
    rejected: 'var(--accent-red)',
    published: 'var(--accent-cyan)',
  };
</script>

<div class="queue-page">
  <h2 class="queue-title">Community forge queue</h2>

  <div class="status-tabs">
    {#each statuses as status}
      <button
        class="status-tab"
        class:active={activeStatus === status}
        style={activeStatus === status ? `color: ${statusColors[status]}; border-color: ${statusColors[status]}` : ''}
        onclick={() => switchStatus(status)}
      >
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </button>
    {/each}
  </div>

  {#if loading}
    <p class="loading">Loading...</p>
  {:else if error}
    <p class="error">{error}</p>
  {:else if challenges.length === 0}
    <p class="empty">No {activeStatus} submissions.</p>
  {:else}
    <div class="queue-list">
      {#each challenges as ch}
        <Card>
          <a href="/admin/community/{ch.id}" class="queue-item">
            <div class="queue-item-left">
              <h3 class="item-title">{ch.title}</h3>
              <div class="item-meta">
                <DifficultyBadge level={ch.difficulty} size="sm" />
                <span class="meta-tag">{ch.language_slug}</span>
                <span class="meta-tag">{ch.vuln_category_slug}</span>
                <span class="meta-tag">{ch.points} PTS</span>
              </div>
              <span class="item-author">by {ch.author_username}</span>
            </div>
            <div class="queue-item-right">
              <span class="item-date">{new Date(ch.created_at).toLocaleDateString()}</span>
              <Button variant="ghost" size="sm">Review &rarr;</Button>
            </div>
          </a>
        </Card>
      {/each}
    </div>
    <div class="pagination">
      <span>Showing {challenges.length} of {total}</span>
    </div>
  {/if}
</div>

<style>
  .queue-page {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .queue-title {
    font-family: var(--font-serif);
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .status-tabs {
    display: flex;
    gap: var(--space-1);
    border-bottom: 1px solid var(--border-primary);
    padding-bottom: var(--space-2);
  }

  .status-tab {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    padding: var(--space-1) var(--space-3);
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    color: var(--text-tertiary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .status-tab:hover {
    color: var(--text-secondary);
    background: var(--bg-tertiary);
  }

  .status-tab.active {
    border-bottom: 2px solid;
  }

  .loading, .error, .empty {
    font-size: 0.875rem;
    color: var(--text-tertiary);
    padding: var(--space-4);
    text-align: center;
  }

  .error {
    color: var(--accent-red);
  }

  .queue-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .queue-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    text-decoration: none;
    color: inherit;
  }

  .item-title {
    font-family: var(--font-sans);
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .item-meta {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    margin-top: var(--space-1);
  }

  .meta-tag {
    font-size: 0.5625rem;
    color: var(--text-tertiary);
    padding: 1px 6px;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
  }

  .item-author {
    font-size: 0.625rem;
    color: var(--text-tertiary);
    margin-top: var(--space-1);
  }

  .queue-item-right {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--space-2);
  }

  .item-date {
    font-size: 0.625rem;
    color: var(--text-tertiary);
  }

  .pagination {
    font-size: 0.75rem;
    color: var(--text-tertiary);
    text-align: center;
    padding: var(--space-2);
  }
</style>
