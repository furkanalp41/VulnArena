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
  <article class="lesson-view">
    <!-- Document Header -->
    <header class="doc-header">
      <a href="/academy" class="back-link">&larr; Academy</a>

      <div class="doc-classification">
        <div class="classification-bar"></div>
        <div class="classification-content">
          <span class="classification-label">Deep dive</span>
          <span class="classification-cat">{lesson.category}</span>
        </div>
        <div class="classification-bar"></div>
      </div>

      <h1 class="doc-title">{lesson.title}</h1>

      <div class="doc-meta">
        <DifficultyBadge level={lesson.difficulty} size="sm" />
        <span class="meta-sep">|</span>
        <span class="meta-item">{lesson.read_time_min} min read</span>
        <span class="meta-sep">|</span>
        <span class="meta-item">{formatDate(lesson.created_at)}</span>
      </div>

      {#if lesson.tags.length > 0}
        <div class="doc-tags">
          {#each lesson.tags as tag}
            <span class="doc-tag">{tag}</span>
          {/each}
        </div>
      {/if}

      {#if lesson.description}
        <p class="doc-abstract">{lesson.description}</p>
      {/if}
    </header>

    <!-- Document Body -->
    <div class="doc-body">
      <MarkdownRenderer content={lesson.content} />
    </div>

    <!-- Document Footer -->
    <footer class="doc-footer">
      <div class="footer-classification">
        <div class="classification-bar"></div>
        <span class="classification-label">End of lesson — {lesson.category}</span>
        <div class="classification-bar"></div>
      </div>

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
    font-family: var(--font-sans);
    font-size: 0.875rem;
    animation: pulse 1.5s ease-in-out infinite;
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

  /* Document layout */
  .lesson-view {
    max-width: 820px;
    margin: 0 auto;
  }

  .back-link {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    text-decoration: none;
    transition: color var(--transition-fast);
    display: inline-block;
    margin-bottom: var(--space-6);
  }

  .back-link:hover {
    color: var(--accent-green);
  }

  /* Header */
  .doc-header {
    margin-bottom: var(--space-8);
  }

  .doc-classification {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-5);
  }

  .classification-bar {
    flex: 1;
    height: 1px;
    background: var(--border-secondary);
    opacity: 0.6;
  }

  .classification-content {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-shrink: 0;
  }

  .classification-label {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    font-weight: 500;
  }

  .classification-cat {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    padding: 2px 8px;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
  }

  .doc-title {
    font-family: var(--font-serif);
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.3;
    margin-bottom: var(--space-4);
  }

  .doc-meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
    flex-wrap: wrap;
  }

  .meta-sep {
    color: var(--border-secondary);
    font-size: 0.75rem;
  }

  .meta-item {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
  }

  .doc-tags {
    display: flex;
    gap: var(--space-1);
    flex-wrap: wrap;
    margin-bottom: var(--space-4);
  }

  .doc-tag {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--accent-blue);
    padding: 2px 8px;
    border: 1px solid var(--accent-blue-glow);
    border-radius: var(--radius-sm);
    background: var(--accent-blue-glow);
  }

  .doc-abstract {
    font-size: 0.9375rem;
    color: var(--text-secondary);
    line-height: 1.7;
    padding: var(--space-4);
    border-left: 2px solid var(--accent-blue);
    background: var(--bg-tertiary);
    border-radius: 0 var(--radius-md) var(--radius-md) 0;
  }

  /* Body */
  .doc-body {
    padding-top: var(--space-4);
  }

  /* Footer */
  .doc-footer {
    margin-top: var(--space-12);
    padding-top: var(--space-6);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-6);
  }

  .footer-classification {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
  }

  .doc-footer a {
    text-decoration: none;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
</style>
