<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { getCommunitySubmission, reviewCommunityChallenge, publishCommunityChallenge } from '$lib/api/admin';
  import type { CommunityChallenge } from '$lib/api/community';
  import CodeEditor from '$lib/components/editor/CodeEditor.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';

  const id = $derived($page.params.id ?? '');

  let challenge = $state<CommunityChallenge | null>(null);
  let loading = $state(true);
  let error = $state('');
  let actionLoading = $state(false);
  let reviewNotes = $state('');

  onMount(async () => {
    try {
      challenge = await getCommunitySubmission(id);
    } catch (e: any) {
      error = e.message || 'Failed to load submission';
    } finally {
      loading = false;
    }
  });

  async function handleReview(action: 'approve' | 'reject') {
    if (action === 'reject' && !reviewNotes.trim()) {
      error = 'Please provide notes when rejecting.';
      return;
    }
    actionLoading = true;
    error = '';
    try {
      await reviewCommunityChallenge(id, action, reviewNotes);
      goto('/admin/community');
    } catch (e: any) {
      error = e.message || `Failed to ${action}`;
    } finally {
      actionLoading = false;
    }
  }

  async function handlePublish() {
    actionLoading = true;
    error = '';
    try {
      await publishCommunityChallenge(id);
      goto('/admin/community');
    } catch (e: any) {
      error = e.message || 'Failed to publish';
    } finally {
      actionLoading = false;
    }
  }

  const statusColors: Record<string, string> = {
    pending: 'var(--accent-yellow)',
    approved: 'var(--accent-green)',
    rejected: 'var(--accent-red)',
    published: 'var(--accent-cyan)',
  };
</script>

<div class="review-page">
  <a href="/admin/community" class="back-link">&larr; Community queue</a>

  {#if loading}
    <p class="loading">Loading...</p>
  {:else if error && !challenge}
    <p class="error">{error}</p>
  {:else if challenge}
    <div class="review-header">
      <div>
        <h2 class="review-title">{challenge.title}</h2>
        <div class="review-meta">
          <DifficultyBadge level={challenge.difficulty} size="sm" />
          <span class="meta-tag">{challenge.language_slug}</span>
          <span class="meta-tag">{challenge.vuln_category_slug}</span>
          <span class="meta-tag">{challenge.points} PTS</span>
          <span class="author-tag">by {challenge.author_username}</span>
        </div>
      </div>
      <span
        class="status-badge"
        style="color: {statusColors[challenge.status]}; border-color: {statusColors[challenge.status]}"
      >
        {challenge.status.charAt(0).toUpperCase() + challenge.status.slice(1)}
      </span>
    </div>

    <div class="review-layout">
      <!-- Left: Code -->
      <div class="code-section">
        <Card>
          <div class="card-inner">
            <span class="section-label">Vulnerable code</span>
            <div class="code-preview">
              <CodeEditor
                code={challenge.vulnerable_code}
                language={challenge.language_slug}
                readonly={true}
                height="400px"
              />
            </div>
          </div>
        </Card>

        {#if challenge.description}
          <Card>
            <div class="card-inner">
              <span class="section-label">Description</span>
              <p class="desc-text">{challenge.description}</p>
            </div>
          </Card>
        {/if}
      </div>

      <!-- Right: Solution + Actions -->
      <div class="solution-section">
        <Card>
          <div class="card-inner">
            <span class="section-label">Target vulnerability</span>
            <p class="solution-text">{challenge.target_vulnerability}</p>
          </div>
        </Card>

        {#if challenge.conceptual_fix}
          <Card>
            <div class="card-inner">
              <span class="section-label">Conceptual fix</span>
              <p class="solution-text">{challenge.conceptual_fix}</p>
            </div>
          </Card>
        {/if}

        {#if challenge.vulnerable_lines}
          <Card>
            <div class="card-inner">
              <span class="section-label">Vulnerable lines</span>
              <span class="lines-display font-mono">{challenge.vulnerable_lines}</span>
            </div>
          </Card>
        {/if}

        {#if challenge.hints && challenge.hints.length > 0}
          <Card>
            <div class="card-inner">
              <span class="section-label">Hints ({challenge.hints.length})</span>
              {#each challenge.hints as hint, i}
                <div class="hint-row">
                  <span class="hint-num">#{i + 1}</span>
                  <span>{hint}</span>
                </div>
              {/each}
            </div>
          </Card>
        {/if}

        <!-- Actions -->
        {#if challenge.status === 'pending'}
          <Card>
            <div class="card-inner">
              <span class="section-label">Review actions</span>
              <textarea
                class="notes-input"
                placeholder="Reviewer notes (required for rejection)..."
                bind:value={reviewNotes}
                rows="3"
              ></textarea>
              {#if error}
                <p class="action-error">{error}</p>
              {/if}
              <div class="action-buttons">
                <Button variant="primary" onclick={() => handleReview('approve')} loading={actionLoading} disabled={actionLoading}>
                  Approve
                </Button>
                <Button variant="ghost" onclick={() => handleReview('reject')} loading={actionLoading} disabled={actionLoading}>
                  Reject
                </Button>
              </div>
            </div>
          </Card>
        {:else if challenge.status === 'approved'}
          <Card>
            <div class="card-inner">
              <span class="section-label">Publish to arena</span>
              <p class="publish-note">This will create an official challenge in the Arena from this submission.</p>
              {#if error}
                <p class="action-error">{error}</p>
              {/if}
              <Button variant="primary" onclick={handlePublish} loading={actionLoading} disabled={actionLoading}>
                Publish challenge
              </Button>
            </div>
          </Card>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .review-page {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .back-link {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    text-decoration: none;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .loading, .error {
    font-size: 0.875rem;
    color: var(--text-tertiary);
  }

  .error {
    color: var(--accent-red);
  }

  .review-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .review-title {
    font-family: var(--font-serif);
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .review-meta {
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

  .author-tag {
    font-size: 0.625rem;
    color: var(--text-secondary);
  }

  .status-badge {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    font-weight: 600;
    padding: 2px 10px;
    border: 1px solid;
    border-radius: var(--radius-sm);
  }

  .review-layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
  }

  .code-section, .solution-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .card-inner {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .section-label {
    font-family: var(--font-sans);
    font-size: 0.625rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .code-preview {
    border-radius: var(--radius-sm);
    overflow: hidden;
  }

  .desc-text, .solution-text {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.6;
    white-space: pre-wrap;
  }

  .lines-display {
    font-size: 0.875rem;
    color: var(--accent-red);
  }

  .hint-row {
    display: flex;
    gap: var(--space-2);
    font-size: 0.8125rem;
    color: var(--text-secondary);
    padding: var(--space-1) 0;
  }

  .hint-num {
    color: var(--accent-yellow);
    font-weight: 600;
    font-size: 0.625rem;
    flex-shrink: 0;
  }

  .notes-input {
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    padding: var(--space-2) var(--space-3);
    background: var(--bg-input);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    outline: none;
    width: 100%;
    resize: vertical;
    line-height: 1.5;
  }

  .notes-input:focus {
    border-color: var(--accent-primary);
  }

  .action-error {
    font-size: 0.75rem;
    color: var(--accent-red);
  }

  .action-buttons {
    display: flex;
    gap: var(--space-3);
  }

  .publish-note {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  @media (max-width: 1024px) {
    .review-layout {
      grid-template-columns: 1fr;
    }
  }
</style>
