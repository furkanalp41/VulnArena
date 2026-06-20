<script lang="ts">
  import { onMount } from 'svelte';
  import { listChallenges, type ChallengeListItem } from '$lib/api/arena';
  import Card from '$lib/components/ui/Card.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';

  let challenges = $state<ChallengeListItem[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state('');

  // Filter state
  let filterLanguage = $state('');
  let filterCategory = $state('');
  let filterDiffMin = $state(0);
  let filterDiffMax = $state(0);
  let currentPage = $state(1);
  const PAGE_SIZE = 20;

  // Derived from initial data load
  let languages = $state<{ slug: string; name: string }[]>([]);
  let categories = $state<{ slug: string; name: string }[]>([]);

  onMount(async () => {
    // Initial load to populate filter options
    try {
      const res = await listChallenges({ limit: 200 });
      const all = res.challenges || [];
      const langMap = new Map(all.map((c) => [c.language.slug, c.language.name]));
      languages = [...langMap.entries()].map(([slug, name]) => ({ slug, name })).sort((a, b) => a.name.localeCompare(b.name));
      const catMap = new Map(all.map((c) => [c.vuln_category.slug, c.vuln_category.name]));
      categories = [...catMap.entries()].map(([slug, name]) => ({ slug, name })).sort((a, b) => a.name.localeCompare(b.name));
    } catch {
      // Non-critical — filters just won't have options
    }
    await fetchChallenges();
  });

  async function fetchChallenges() {
    loading = true;
    error = '';
    try {
      const params: Record<string, any> = { limit: PAGE_SIZE, page: currentPage };
      if (filterLanguage) params.language = filterLanguage;
      if (filterCategory) params.category = filterCategory;
      if (filterDiffMin > 0) params.difficulty_min = filterDiffMin;
      if (filterDiffMax > 0) params.difficulty_max = filterDiffMax;
      const res = await listChallenges(params);
      challenges = res.challenges || [];
      total = res.total;
    } catch {
      error = 'Failed to load challenges. Is the API server running?';
    } finally {
      loading = false;
    }
  }

  function applyFilter() {
    currentPage = 1;
    fetchChallenges();
  }

  function resetFilters() {
    filterLanguage = '';
    filterCategory = '';
    filterDiffMin = 0;
    filterDiffMax = 0;
    currentPage = 1;
    fetchChallenges();
  }

  function prevPage() {
    if (currentPage > 1) {
      currentPage--;
      fetchChallenges();
    }
  }

  function nextPage() {
    if (currentPage < Math.ceil(total / PAGE_SIZE)) {
      currentPage++;
      fetchChallenges();
    }
  }

  $effect(() => {
    // Keep diffMax >= diffMin when both are set
    if (filterDiffMin > 0 && filterDiffMax > 0 && filterDiffMax < filterDiffMin) {
      filterDiffMax = filterDiffMin;
    }
  });

  const hasActiveFilters = $derived(
    !!filterLanguage || !!filterCategory || filterDiffMin > 0 || filterDiffMax > 0
  );

  const totalPages = $derived(Math.ceil(total / PAGE_SIZE));
</script>

<div class="arena page">
  <header class="page-head">
    <span class="eyebrow">The Arena</span>
    <h1 class="arena-title">Open audits</h1>
    <p class="arena-subtitle">
      Find the vulnerability. Explain the fix. Prove your skills.
      {#if total > 0}
        <span class="sep">·</span>
        <span class="challenge-count tnum">{total} challenge{total !== 1 ? 's' : ''}</span>
      {/if}
    </p>
  </header>

  <!-- Filter toolbar -->
  <div class="toolbar">
    <div class="sel">
      <label for="filter-language">Language</label>
      <select id="filter-language" bind:value={filterLanguage} onchange={applyFilter}>
        <option value="">All languages</option>
        {#each languages as lang}
          <option value={lang.slug}>{lang.name}</option>
        {/each}
      </select>
    </div>

    <div class="sel">
      <label for="filter-category">Category</label>
      <select id="filter-category" bind:value={filterCategory} onchange={applyFilter}>
        <option value="">Every category</option>
        {#each categories as cat}
          <option value={cat.slug}>{cat.name}</option>
        {/each}
      </select>
    </div>

    <div class="sel">
      <label for="filter-diff-min">Min difficulty</label>
      <select id="filter-diff-min" bind:value={filterDiffMin} onchange={applyFilter}>
        <option value={0}>Any</option>
        {#each Array(10) as _, i}
          <option value={i + 1}>{i + 1}</option>
        {/each}
      </select>
    </div>

    <div class="sel">
      <label for="filter-diff-max">Max difficulty</label>
      <select id="filter-diff-max" bind:value={filterDiffMax} onchange={applyFilter}>
        <option value={0}>Any</option>
        {#each Array(10) as _, i}
          <option value={i + 1}>{i + 1}</option>
        {/each}
      </select>
    </div>

    {#if hasActiveFilters}
      <div class="sel sel-action">
        <button class="reset-btn" onclick={resetFilters}>Reset</button>
      </div>
    {/if}
  </div>

  {#if loading}
    <div class="state">
      <span class="loading-text">Loading challenges…</span>
    </div>
  {:else if error}
    <div class="state">
      <span class="error-icon">!</span>
      <p class="error-text">{error}</p>
    </div>
  {:else if challenges.length === 0}
    <div class="state">
      <p class="empty-text">No challenges found</p>
      <p class="empty-sub">
        {#if hasActiveFilters}
          Try adjusting your filters.
        {:else}
          Run <code>make seed</code> to populate the arena.
        {/if}
      </p>
    </div>
  {:else}
    <div class="index-list">
      {#each challenges as challenge}
        <a href="/arena/{challenge.id}" class="index-row">
          <div class="row-main">
            <h3>{challenge.title}</h3>
            <p class="desc">{challenge.description}</p>
          </div>
          <div class="dl dateline">
            <span class="diff-mark tnum">Diff {challenge.difficulty}</span>
            <span class="sep">·</span>
            <span class="tnum">{challenge.points} pts</span>
            <span class="sep">·</span>
            {challenge.language.name}
            <span class="sep">·</span>
            {challenge.vuln_category.name}
          </div>
        </a>
      {/each}
    </div>

    {#if totalPages > 1}
      <div class="pagination">
        <button class="page-btn" disabled={currentPage <= 1} onclick={prevPage}>Previous</button>
        <span class="page-info tnum">Page {currentPage} of {totalPages}</span>
        <button class="page-btn" disabled={currentPage >= totalPages} onclick={nextPage}>Next</button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .page {
    padding: var(--space-7) 0 var(--space-8);
  }

  /* Page head */
  .page-head {
    margin-bottom: var(--space-6);
  }

  .arena-title {
    font-family: var(--font-serif);
    font-size: var(--fs-h1);
    font-weight: 600;
    letter-spacing: -0.015em;
    color: var(--text-primary);
    margin-top: var(--space-1);
  }

  .arena-subtitle {
    color: var(--text-secondary);
    font-size: var(--fs-body);
    margin-top: var(--space-2);
  }

  .arena-subtitle .sep {
    color: var(--text-tertiary);
    margin: 0 0.5em;
  }

  .challenge-count {
    color: var(--text-tertiary);
  }

  /* Filter toolbar — hairline-divided, floating small-caps labels */
  .toolbar {
    display: flex;
    align-items: stretch;
    gap: 0;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    margin-bottom: var(--space-6);
    overflow: hidden;
  }

  .sel {
    position: relative;
    flex: 1;
    border-right: 1px solid var(--border-primary);
  }

  .sel:last-child {
    border-right: 0;
  }

  .sel label {
    position: absolute;
    top: 7px;
    left: 0.85rem;
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.13em;
    text-transform: uppercase;
    color: var(--text-tertiary);
    pointer-events: none;
  }

  .sel select {
    appearance: none;
    width: 100%;
    background: transparent;
    border: 0;
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: var(--fs-body);
    padding: 1.5rem 0.85rem 0.6rem;
    border-bottom: 2px solid transparent;
    cursor: pointer;
  }

  .sel select:focus {
    outline: none;
    border-bottom-color: var(--accent-primary);
  }

  /* custom chevron */
  .sel:not(.sel-action)::after {
    content: '';
    position: absolute;
    right: 0.95rem;
    bottom: 1.05rem;
    width: 7px;
    height: 7px;
    border-right: 1.5px solid var(--text-tertiary);
    border-bottom: 1.5px solid var(--text-tertiary);
    transform: rotate(45deg);
    pointer-events: none;
  }

  .sel-action {
    flex: 0 0 auto;
    display: flex;
    align-items: stretch;
  }

  .reset-btn {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.13em;
    text-transform: uppercase;
    padding: 0 1.1rem;
    background: transparent;
    border: 0;
    color: var(--text-secondary);
    cursor: pointer;
    transition: color 0.15s ease;
  }

  .reset-btn:hover {
    color: var(--accent-red);
  }

  /* Index list — hairline-ruled rows */
  .index-list {
    border-top: 1px solid var(--border-primary);
  }

  .index-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: var(--space-4) var(--space-5);
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

  .index-row h3 {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    margin-bottom: var(--space-1);
  }

  .index-row .desc {
    color: var(--text-secondary);
    font-size: var(--fs-micro);
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 1;
    line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .index-row .dl {
    align-self: center;
    text-align: right;
    white-space: nowrap;
  }

  .diff-mark {
    color: var(--accent-primary);
  }

  /* Pagination */
  .pagination {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: var(--space-4);
    margin-top: var(--space-6);
  }

  .page-btn {
    font-family: var(--font-sans);
    font-size: var(--fs-micro);
    padding: 0.4rem 0.9rem;
    background: transparent;
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-input);
    color: var(--text-primary);
    cursor: pointer;
    transition: border-color 0.15s ease, color 0.15s ease;
  }

  .page-btn:hover:not(:disabled) {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
  }

  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-info {
    font-size: var(--fs-micro);
    color: var(--text-tertiary);
  }

  /* States */
  .state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-12) var(--space-6);
    border-top: 1px solid var(--border-primary);
  }

  .loading-text {
    font-size: var(--fs-micro);
    color: var(--text-secondary);
    animation: pulse 1.5s ease-in-out infinite;
  }

  .error-icon {
    font-family: var(--font-serif);
    font-size: 1.5rem;
    color: var(--accent-red);
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--accent-red);
    border-radius: 50%;
  }

  .error-text {
    color: var(--accent-red);
    font-size: var(--fs-micro);
  }

  .empty-text {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    color: var(--text-primary);
  }

  .empty-sub {
    font-size: var(--fs-micro);
    color: var(--text-tertiary);
  }

  .empty-sub code {
    font-family: var(--font-mono);
    color: var(--accent-primary);
    padding: 2px 6px;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  @media (max-width: 640px) {
    .toolbar {
      flex-wrap: wrap;
    }

    .sel {
      flex: 1 1 50%;
      border-bottom: 1px solid var(--border-primary);
    }

    .index-row {
      grid-template-columns: 1fr;
      gap: var(--space-2);
    }

    .index-row .dl {
      align-self: start;
      text-align: left;
    }
  }
</style>
