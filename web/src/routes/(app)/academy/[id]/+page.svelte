<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { getLesson, type Lesson } from '$lib/api/academy';
  import { ApiError } from '$lib/api/client';
  import MarkdownRenderer from '$lib/components/ui/MarkdownRenderer.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  let lesson = $state<Lesson | null>(null);
  let loading = $state(true);
  let error = $state('');

  const lessonId = $derived($page.params.id ?? '');

  onMount(async () => {
    try {
      lesson = await getLesson(lessonId);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        error = 'Lesson not found';
      } else {
        error = 'Failed to load lesson';
      }
    } finally {
      loading = false;
    }
  });

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }
</script>

{#if loading}
  <div class="loading-screen">
    <span class="loading-text">Loading lesson...</span>
  </div>
{:else if error}
  <div class="error-screen">
    <span class="error-icon">X</span>
    <p>{error}</p>
    <a href="/academy"><Button variant="ghost">Back to Academy</Button></a>
  </div>
{:else if lesson}
  <article class="lesson">
    <a href="/academy" class="back-link">&larr; Academy</a>

    <span class="eyebrow">{lesson.category} &middot; Deep dive</span>
    <h1>{lesson.title}</h1>

    <div class="byline dateline">
      <DifficultyBadge level={lesson.difficulty} size="sm" />
      <span class="sep">&middot;</span>
      <span class="tnum">{lesson.read_time_min} min read</span>
      <span class="sep">&middot;</span>
      <span class="tnum">{formatDate(lesson.created_at)}</span>
    </div>

    {#if lesson.tags.length > 0}
      <div class="tag-row">
        {#each lesson.tags as tag}
          <span class="smallcaps tag">{tag}</span>
        {/each}
      </div>
    {/if}

    {#if lesson.description}
      <p class="lede">{lesson.description}</p>
    {/if}

    <!-- Document Body -->
    <div class="doc-body">
      <MarkdownRenderer content={lesson.content} />
    </div>

    <div class="end">&middot; End of lesson &middot;</div>

    <footer class="doc-footer">
      <a href="/academy">
        <Button variant="ghost" size="md">Return to Academy</Button>
      </a>
    </footer>
  </article>
{/if}

<style>
  .loading-screen, .error-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    min-height: 50vh;
  }

  .loading-text {
    color: var(--text-tertiary);
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    letter-spacing: 0.04em;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .error-screen p {
    font-family: var(--font-serif);
    font-size: var(--fs-lead);
    color: var(--text-secondary);
  }

  .error-icon {
    font-family: var(--font-mono);
    font-size: 1.25rem;
    color: var(--accent-red);
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--accent-red);
    border-radius: 50%;
  }

  /* ============================================================
     ACADEMY — long-form
     ============================================================ */
  .lesson {
    max-width: var(--measure);
    margin: 0 auto;
  }

  .back-link {
    display: inline-block;
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    letter-spacing: 0.04em;
    color: var(--text-tertiary);
    text-decoration: none;
    margin-bottom: var(--space-6);
    transition: color var(--transition-fast);
  }

  .back-link:hover {
    color: var(--accent-primary);
  }

  .lesson h1 {
    font-family: var(--font-serif);
    font-size: var(--fs-h1);
    font-weight: 600;
    letter-spacing: -0.015em;
    line-height: 1.1;
    color: var(--text-primary);
    margin: var(--space-3) 0 var(--space-3);
  }

  .byline {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0 0.35em;
    padding-bottom: var(--space-5);
    margin-bottom: var(--space-6);
    border-bottom: 1px solid var(--border-primary);
  }

  /* Tag row — quiet tracked small-caps over the body */
  .tag-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-4);
    margin: 0 0 var(--space-6);
  }

  .tag-row .tag {
    color: var(--text-tertiary);
  }

  /* Abstract / intro — serif lede with a sand drop-cap */
  .lede {
    font-family: var(--font-serif);
    font-size: var(--fs-lead);
    line-height: 1.6;
    color: var(--text-primary);
    margin: 0 0 var(--space-6);
  }

  .lede::first-letter {
    font-family: var(--font-serif);
    font-weight: 600;
    float: left;
    font-size: 3.6rem;
    line-height: 0.8;
    padding: 0.06em 0.12em 0.02em 0;
    color: var(--accent-primary);
  }

  /* Body — MarkdownRenderer output styled for the reading measure */
  .doc-body {
    color: var(--text-secondary);
    line-height: 1.7;
  }

  .doc-body :global(p) {
    margin: var(--space-4) 0;
    color: var(--text-secondary);
    line-height: 1.7;
  }

  .doc-body :global(h2) {
    font-family: var(--font-mono);
    font-size: var(--fs-label);
    text-transform: uppercase;
    letter-spacing: 0.13em;
    font-weight: 500;
    color: var(--text-tertiary);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-primary);
    margin: var(--space-6) 0 var(--space-4);
  }

  .doc-body :global(h3) {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    text-transform: uppercase;
    letter-spacing: 0.13em;
    font-weight: 500;
    color: var(--text-tertiary);
    margin: var(--space-5) 0 var(--space-3);
  }

  .doc-body :global(a) {
    color: var(--accent-primary);
    text-decoration: none;
    border-bottom: 1px solid color-mix(in srgb, var(--accent-primary) 40%, transparent);
  }

  .doc-body :global(a:hover) {
    border-bottom-color: var(--accent-primary);
  }

  .doc-body :global(pre) {
    background: var(--editor-bg);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    padding: var(--space-4);
    overflow-x: auto;
    margin: var(--space-4) 0;
  }

  /* Colophon — quiet centered end-of-lesson mark */
  .end {
    text-align: center;
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.2em;
    color: var(--text-tertiary);
    margin: var(--space-8) 0 0;
  }

  .doc-footer {
    display: flex;
    justify-content: center;
    margin-top: var(--space-6);
  }

  .doc-footer a {
    text-decoration: none;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
</style>
