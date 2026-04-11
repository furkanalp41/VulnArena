<script lang="ts">
  import { onMount } from 'svelte';
  import { listMyCommunitySubmissions, deleteCommunityChallenge, type CommunityChallenge } from '$lib/api/community';
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';

  let challenges = $state<CommunityChallenge[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state('');

  onMount(async () => {
    try {
      const res = await listMyCommunitySubmissions();
      challenges = res.challenges ?? [];
      total = res.total;
    } catch (e: any) {
      error = e.message || 'Failed to load submissions';
    } finally {
      loading = false;
    }
  });

  async function handleDelete(id: string) {
    if (!confirm('Delete this submission? This cannot be undone.')) return;
    try {
      await deleteCommunityChallenge(id);
      challenges = challenges.filter(c => c.id !== id);
      total--;
    } catch {
      error = 'Failed to delete submission';
    }
  }

  const statusColors: Record<string, string> = {
    pending: 'var(--accent-yellow)',
    approved: 'var(--accent-green)',
    rejected: 'var(--accent-red)',
    published: 'var(--accent-cyan, #06b6d4)',
  };
</script>

<div class="forge-page">
  <div class="forge-header">
    <div>
      <h1 class="forge-title">Community Forge</h1>
      <p class="forge-subtitle">Submit your own vulnerable code challenges for the community.</p>
      <p class="forge-note">Requires Pro Hacker rank (600+ XP)</p>
    </div>
    <a href="/forge/new">
      <Button variant="primary">+ New submission</Button>
    </a>
  </div>

  {#if loading}
    <p class="loading">Loading submissions...</p>
  {:else if error}
    <p class="error">{error}</p>
  {:else if challenges.length === 0}
    <Card>
      <div class="empty-state">
        <p>No submissions yet.</p>
        <p>Create your first vulnerable code challenge and share it with the community.</p>
        <a href="/forge/new"><Button variant="primary">Submit a Challenge</Button></a>
      </div>
    </Card>
  {:else}
    <div class="submissions-list">
      {#each challenges as ch}
        <Card>
          <div class="submission-card">
            <div class="submission-top">
              <div class="submission-info">
                <h3 class="submission-title">{ch.title}</h3>
                <div class="submission-meta">
                  <DifficultyBadge level={ch.difficulty} size="sm" />
                  <span class="meta-tag">{ch.language_slug}</span>
                  <span class="meta-tag">{ch.vuln_category_slug}</span>
                  <span class="meta-tag">{ch.points} PTS</span>
                </div>
              </div>
              <span
                class="status-badge"
                style="color: {statusColors[ch.status]}; border-color: {statusColors[ch.status]}"
              >
                {ch.status.charAt(0).toUpperCase() + ch.status.slice(1)}
              </span>
            </div>

            {#if ch.reviewer_notes}
              <div class="reviewer-notes">
                <span class="notes-label">Reviewer notes</span>
                <p>{ch.reviewer_notes}</p>
              </div>
            {/if}

            <div class="submission-actions">
              <span class="submission-date">
                {new Date(ch.created_at).toLocaleDateString()}
              </span>
              {#if ch.status === 'pending'}
                <a href="/forge/new?edit={ch.id}">
                  <Button variant="ghost" size="sm">Edit</Button>
                </a>
                <Button variant="ghost" size="sm" onclick={() => handleDelete(ch.id)}>Delete</Button>
              {/if}
              {#if ch.status === 'published' && ch.challenge_id}
                <a href="/arena/{ch.challenge_id}">
                  <Button variant="ghost" size="sm">View in arena</Button>
                </a>
              {/if}
            </div>
          </div>
        </Card>
      {/each}
    </div>
  {/if}
</div>

<style>
  .forge-page {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .forge-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .forge-title {
    font-family: var(--font-serif);
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .forge-subtitle {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-top: var(--space-1);
  }

  .forge-note {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    margin-top: var(--space-1);
  }

  .loading, .error {
    font-size: 0.875rem;
    color: var(--text-tertiary);
  }

  .error {
    color: var(--accent-red);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-6);
    text-align: center;
    color: var(--text-secondary);
  }

  .submissions-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .submission-card {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .submission-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .submission-title {
    font-family: var(--font-sans);
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .submission-meta {
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

  .status-badge {
    font-family: var(--font-sans);
    font-size: 0.625rem;
    font-weight: 600;
    padding: 2px 8px;
    border: 1px solid;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
  }

  .reviewer-notes {
    padding: var(--space-2) var(--space-3);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    border-left: 2px solid var(--accent-yellow);
  }

  .notes-label {
    font-family: var(--font-sans);
    font-size: 0.5625rem;
    color: var(--accent-yellow);
    display: block;
    margin-bottom: var(--space-1);
  }

  .reviewer-notes p {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .submission-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding-top: var(--space-2);
    border-top: 1px solid var(--border-primary);
  }

  .submission-date {
    font-size: 0.625rem;
    color: var(--text-tertiary);
    margin-right: auto;
  }
</style>
