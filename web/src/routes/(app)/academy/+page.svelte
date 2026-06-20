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
  <header class="masthead">
    <span class="eyebrow">VulnArena · Academy</span>
    <h1 class="masthead-title">The Academy</h1>
    <p class="masthead-lede">Deep-dive technical breakdowns of how vulnerabilities manifest at the function level.</p>
  </header>

  <div class="section-header">
    <h2>Lessons</h2>
    {#if total > 0}
      <span class="smallcaps tnum">{total} {total !== 1 ? 'entries' : 'entry'}</span>
    {/if}
  </div>

  {#if loading}
    <div class="state-msg">
      <span class="rule-s"></span>
      <span class="state-text">Loading lessons…</span>
    </div>
  {:else if error}
    <div class="state-msg">
      <span class="rule-s"></span>
      <p class="error-text">{error}</p>
    </div>
  {:else if lessons.length === 0}
    <div class="state-msg">
      <span class="rule-s"></span>
      <span class="state-text">No lessons found</span>
      <p class="state-sub">Run <code>make seed</code> to populate the academy.</p>
    </div>
  {:else}
    <div class="index-list">
      {#each lessons as lesson}
        <a href="/academy/{lesson.id}" class="index-row">
          <div class="row-main">
            <h3 class="row-title">{lesson.title}</h3>
            <p class="desc">{lesson.description.slice(0, 200)}{lesson.description.length > 200 ? '…' : ''}</p>
            <p class="dateline">
              <span class="cat">{lesson.category}</span>
              <span class="sep">·</span>
              <span>Difficulty <span class="tnum">{lesson.difficulty}</span></span>
              <span class="sep">·</span>
              <span class="tnum">{lesson.read_time_min} min read</span>
            </p>
          </div>
          <div class="dl">
            <DifficultyBadge level={lesson.difficulty} size="sm" />
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>

<style>
  .academy {
    display: flex;
    flex-direction: column;
  }

  /* Masthead */
  .masthead {
    margin-bottom: var(--space-7);
  }

  .masthead-title {
    font-family: var(--font-serif);
    font-size: var(--fs-h1);
    font-weight: 600;
    letter-spacing: -0.015em;
    line-height: 1.05;
    color: var(--text-primary);
    margin: var(--space-3) 0 var(--space-3);
  }

  .masthead-lede {
    max-width: var(--measure);
    font-family: var(--font-serif);
    font-size: var(--fs-lead);
    line-height: 1.55;
    color: var(--text-secondary);
  }

  /* Index list — hairline-ruled rows */
  .index-list {
    border-top: 1px solid var(--border-primary);
  }

  .index-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: var(--space-4) var(--space-5);
    align-items: start;
    padding: var(--space-4) var(--space-2);
    border-bottom: 1px solid var(--border-primary);
    border-left: 2px solid transparent;
    transition: border-color 0.15s ease, background 0.15s ease;
    text-decoration: none;
    color: inherit;
  }

  .index-row:hover {
    border-left-color: var(--accent-primary);
    background: var(--bg-hover);
  }

  .row-title {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    margin-bottom: var(--space-1);
  }

  .desc {
    color: var(--text-secondary);
    font-size: var(--fs-micro);
    line-height: 1.55;
    display: -webkit-box;
    -webkit-line-clamp: 1;
    line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
    margin-bottom: var(--space-2);
  }

  .dateline .cat {
    text-transform: capitalize;
  }

  .dl {
    align-self: center;
    text-align: right;
    white-space: nowrap;
  }

  /* States */
  .state-msg {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-card);
    padding: var(--space-12) var(--space-6);
    text-align: center;
  }

  .rule-s {
    width: 32px;
    height: 1px;
    background: var(--border-secondary);
  }

  .state-text {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    letter-spacing: 0.03em;
    color: var(--text-secondary);
  }

  .state-sub {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    color: var(--text-tertiary);
  }

  .state-sub code {
    font-family: var(--font-mono);
    color: var(--accent-primary);
  }

  .error-text {
    font-family: var(--font-mono);
    color: var(--accent-red);
    font-size: var(--fs-micro);
    letter-spacing: 0.02em;
  }

  @media (max-width: 600px) {
    .index-row {
      grid-template-columns: 1fr;
    }

    .dl {
      text-align: left;
    }
  }
</style>
