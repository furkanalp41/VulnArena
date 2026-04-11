<script lang="ts">
  import { onMount } from 'svelte';
  import { listLessons, type LessonListItem } from '$lib/api/academy';
  import Card from '$lib/components/ui/Card.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';

  let lessons = $state<LessonListItem[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state('');

  onMount(async () => {
    try {
      const res = await listLessons({ limit: 50 });
      lessons = res.lessons || [];
      total = res.total;
    } catch (e) {
      error = 'Failed to load lessons. Is the API server running?';
    } finally {
      loading = false;
    }
  });
</script>

<div class="academy">
  <header class="academy-header">
    <div>
      <h1 class="academy-title">The Academy</h1>
      <p class="academy-subtitle">Deep-dive technical breakdowns of how vulnerabilities manifest at the function level.</p>
    </div>
    {#if total > 0}
      <span class="lesson-count">{total} lesson{total !== 1 ? 's' : ''} available</span>
    {/if}
  </header>

  {#if loading}
    <Card variant="bordered" padding="lg">
      <div class="state-msg">
        <span class="state-text">Loading lessons...</span>
      </div>
    </Card>
  {:else if error}
    <Card variant="bordered" padding="lg">
      <div class="state-msg">
        <span class="error-icon">!</span>
        <p class="error-text">{error}</p>
      </div>
    </Card>
  {:else if lessons.length === 0}
    <Card variant="bordered" padding="lg">
      <div class="state-msg">
        <span class="state-text">No lessons found</span>
        <p class="state-sub">Run <code>make seed</code> to populate the academy.</p>
      </div>
    </Card>
  {:else}
    <div class="lesson-grid">
      {#each lessons as lesson}
        <a href="/academy/{lesson.id}" class="lesson-link">
          <article class="lesson-card">
            <div class="card-stripe"></div>
            <div class="card-body">
              <div class="card-top">
                <span class="classification">{lesson.category}</span>
                <DifficultyBadge level={lesson.difficulty} size="sm" />
              </div>

              <h2 class="card-title">{lesson.title}</h2>

              <p class="card-desc">{lesson.description.slice(0, 200)}{lesson.description.length > 200 ? '...' : ''}</p>

              <div class="card-footer">
                <div class="tags">
                  {#each lesson.tags.slice(0, 4) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
                <span class="read-time">{lesson.read_time_min} min read</span>
              </div>
            </div>
          </article>
        </a>
      {/each}
    </div>
  {/if}
</div>

<style>
  .academy {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .academy-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
  }

  .academy-title {
    font-family: var(--font-serif);
    font-size: 1.5rem;
  }

  .academy-subtitle {
    color: var(--text-secondary);
    font-size: 0.9375rem;
    margin-top: var(--space-1);
  }

  .lesson-count {
    font-family: var(--font-sans);
    font-size: 0.75rem;
    color: var(--text-tertiary);
  }

  /* Lesson grid */
  .lesson-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
    gap: var(--space-5);
  }

  .lesson-link {
    text-decoration: none;
    color: inherit;
  }

  .lesson-card {
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-lg);
    overflow: hidden;
    display: flex;
    transition: all var(--transition-base);
    height: 100%;
  }

  .lesson-card:hover {
    border-color: var(--accent-blue);
    box-shadow: var(--shadow-glow-blue);
    transform: translateY(-2px);
  }

  .card-stripe {
    width: 4px;
    background: linear-gradient(180deg, var(--accent-red) 0%, var(--accent-orange) 50%, var(--accent-yellow) 100%);
    flex-shrink: 0;
  }

  .card-body {
    padding: var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    flex: 1;
  }

  .card-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .classification {
    font-family: var(--font-sans);
    font-size: 0.5625rem;
    color: var(--text-tertiary);
    font-weight: 600;
    text-transform: capitalize;
  }

  .card-title {
    font-family: var(--font-sans);
    font-size: 1.0625rem;
    font-weight: 600;
    color: var(--text-primary);
    line-height: 1.4;
  }

  .card-desc {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.6;
    flex: 1;
  }

  .card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-primary);
    gap: var(--space-2);
  }

  .tags {
    display: flex;
    gap: var(--space-1);
    flex-wrap: wrap;
  }

  .tag {
    font-family: var(--font-sans);
    font-size: 0.5625rem;
    color: var(--text-tertiary);
    padding: 1px 6px;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
  }

  .read-time {
    font-family: var(--font-sans);
    font-size: 0.625rem;
    color: var(--text-tertiary);
    white-space: nowrap;
  }

  /* States */
  .state-msg {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-12);
  }

  .state-text {
    font-family: var(--font-sans);
    font-size: 0.875rem;
    color: var(--text-secondary);
    animation: pulse 1.5s ease-in-out infinite;
  }

  .state-sub {
    font-size: 0.8125rem;
    color: var(--text-tertiary);
  }

  .state-sub code {
    color: var(--text-primary);
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: var(--radius-sm);
  }

  .error-icon {
    font-size: 2rem;
    color: var(--accent-red);
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px solid var(--accent-red);
    border-radius: 50%;
  }

  .error-text {
    color: var(--accent-red);
    font-size: 0.875rem;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
</style>
