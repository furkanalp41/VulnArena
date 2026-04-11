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

<div class="arena">
  <header class="arena-header">
    <div>
      <h1 class="arena-title">The Arena</h1>
      <p class="arena-subtitle">Find the vulnerability. Explain the fix. Prove your skills.</p>
    </div>
    {#if total > 0}
      <span class="challenge-count">{total} challenge{total !== 1 ? 's' : ''}</span>
    {/if}
  </header>

  <!-- Filter Bar -->
  <div class="filter-bar">
    <select bind:value={filterLanguage} onchange={applyFilter}>
      <option value="">All Languages</option>
      {#each languages as lang}
        <option value={lang.slug}>{lang.name}</option>
      {/each}
    </select>

    <select bind:value={filterCategory} onchange={applyFilter}>
      <option value="">All Categories</option>
      {#each categories as cat}
        <option value={cat.slug}>{cat.name}</option>
      {/each}
    </select>

    <select bind:value={filterDiffMin} onchange={applyFilter}>
      <option value={0}>Min Difficulty</option>
      {#each Array(10) as _, i}
        <option value={i + 1}>{i + 1}</option>
      {/each}
    </select>

    <select bind:value={filterDiffMax} onchange={applyFilter}>
      <option value={0}>Max Difficulty</option>
      {#each Array(10) as _, i}
        <option value={i + 1}>{i + 1}</option>
      {/each}
    </select>

    {#if hasActiveFilters}
      <button class="reset-btn" onclick={resetFilters}>Reset</button>
    {/if}
  </div>

  {#if loading}
    <Card variant="bordered" padding="lg">
      <div class="loading-state">
        <span class="loading-text">Loading challenges...</span>
      </div>
    </Card>
  {:else if error}
    <Card variant="bordered" padding="lg">
      <div class="error-state">
        <span class="error-icon">!</span>
        <p class="error-text">{error}</p>
      </div>
    </Card>
  {:else if challenges.length === 0}
    <Card variant="bordered" padding="lg">
      <div class="empty-state">
        <p class="empty-text">No challenges found</p>
        <p class="empty-sub">
          {#if hasActiveFilters}
            Try adjusting your filters.
          {:else}
            Run <code>make seed</code> to populate the arena.
          {/if}
        </p>
      </div>
    </Card>
  {:else}
    <div class="challenge-grid">
      {#each challenges as challenge}
        <a href="/arena/{challenge.id}" class="challenge-link">
          <div class="challenge-card">
            <div class="card-top">
              <DifficultyBadge level={challenge.difficulty} size="sm" />
              <span class="points">{challenge.points} pts</span>
            </div>

            <h3 class="card-title">{challenge.title}</h3>

            <p class="card-desc">{challenge.description.slice(0, 140)}{challenge.description.length > 140 ? '...' : ''}</p>

            <div class="card-meta">
              <span class="meta-tag">{challenge.language.name}</span>
              <span class="meta-divider">&middot;</span>
              <span class="meta-tag">{challenge.vuln_category.name}</span>
              <span class="meta-divider">&middot;</span>
              <span class="meta-tag">{challenge.line_count} lines</span>
            </div>
          </div>
        </a>
      {/each}
    </div>

    {#if totalPages > 1}
      <div class="pagination">
        <button class="page-btn" disabled={currentPage <= 1} onclick={prevPage}>Previous</button>
        <span class="page-info">Page {currentPage} of {totalPages}</span>
        <button class="page-btn" disabled={currentPage >= totalPages} onclick={nextPage}>Next</button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .arena {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .arena-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
  }

  .arena-title {
    font-family: var(--font-serif);
    font-size: 1.75rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .arena-subtitle {
    color: var(--text-secondary);
    font-size: 0.9375rem;
    margin-top: var(--space-1);
  }

  .challenge-count {
    font-size: 0.875rem;
    color: var(--text-tertiary);
  }

  /* Filter bar */
  .filter-bar {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3);
    align-items: center;
    padding: var(--space-4);
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-lg);
  }

  .filter-bar select {
    font-family: var(--font-sans);
    font-size: 0.875rem;
    padding: var(--space-2) var(--space-3);
    background: var(--bg-input);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    outline: none;
    cursor: pointer;
    transition: border-color var(--transition-fast);
  }

  .filter-bar select:focus {
    border-color: var(--accent-green);
  }

  .reset-btn {
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    padding: var(--space-2) var(--space-3);
    background: transparent;
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-md);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .reset-btn:hover {
    border-color: var(--accent-red);
    color: var(--accent-red);
  }

  /* Challenge grid */
  .challenge-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
    gap: var(--space-4);
  }

  .challenge-link {
    text-decoration: none;
    color: inherit;
  }

  .challenge-card {
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-lg);
    padding: var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    transition: all var(--transition-base);
    height: 100%;
  }

  .challenge-card:hover {
    border-color: var(--border-secondary);
    box-shadow: var(--shadow-md);
    transform: translateY(-1px);
  }

  .card-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .points {
    font-size: 0.8125rem;
    color: var(--accent-green);
    font-weight: 500;
  }

  .card-title {
    font-family: var(--font-serif);
    font-size: 1.0625rem;
    font-weight: 600;
    color: var(--text-primary);
    line-height: 1.4;
  }

  .card-desc {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.5;
    flex: 1;
  }

  .card-meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-primary);
  }

  .meta-tag {
    font-size: 0.8125rem;
    color: var(--text-tertiary);
  }

  .meta-divider {
    color: var(--border-secondary);
    font-size: 0.8125rem;
  }

  /* Pagination */
  .pagination {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: var(--space-4);
  }

  .page-btn {
    font-family: var(--font-sans);
    font-size: 0.875rem;
    padding: var(--space-2) var(--space-4);
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .page-btn:hover:not(:disabled) {
    border-color: var(--accent-green);
    color: var(--text-primary);
  }

  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-info {
    font-size: 0.875rem;
    color: var(--text-tertiary);
  }

  /* States */
  .loading-state, .error-state, .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-12);
  }

  .loading-text {
    font-size: 0.875rem;
    color: var(--text-secondary);
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

  .error-text {
    color: var(--accent-red);
    font-size: 0.875rem;
  }

  .empty-text {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .empty-sub {
    font-size: 0.8125rem;
    color: var(--text-tertiary);
  }

  .empty-sub code {
    color: var(--accent-green);
    background: var(--accent-green-glow);
    padding: 2px 6px;
    border-radius: var(--radius-sm);
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  @media (max-width: 600px) {
    .filter-bar {
      flex-direction: column;
    }

    .filter-bar select {
      width: 100%;
    }

    .challenge-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
