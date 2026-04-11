<script lang="ts">
  import { onMount } from 'svelte';
  import { getPlatformStats, type PlatformStats } from '$lib/api/admin';
  import Card from '$lib/components/ui/Card.svelte';

  let stats: PlatformStats | null = $state(null);
  let loading = $state(true);
  let error = $state('');
  let apiHealthy = $state(true);

  onMount(async () => {
    try {
      stats = await getPlatformStats();
    } catch (e: any) {
      error = e?.message || 'Failed to load platform stats';
      apiHealthy = false;
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>Admin C2 | VulnArena</title>
</svelte:head>

{#if loading}
  <div class="c2-loading">
    <div class="loading-pulse c2-pulse"></div>
    <span class="font-mono loading-text">LOADING C2 TELEMETRY...</span>
  </div>
{:else if error}
  <div class="c2-error">
    <span class="font-mono">[C2 ERROR] {error}</span>
  </div>
{:else if stats}
  <div class="c2-dashboard">
    <header class="c2-header">
      <h1 class="c2-title font-mono">
        <span class="c2-bracket">[</span>PLATFORM TELEMETRY<span class="c2-bracket">]</span>
      </h1>
      <div class="c2-status font-mono">
        <span class="status-dot" class:healthy={apiHealthy} class:unhealthy={!apiHealthy}></span>
        {apiHealthy ? 'API OPERATIONAL' : 'API DEGRADED'}
      </div>
    </header>

    <div class="stats-grid">
      <div class="stat-card c2-card">
        <span class="stat-icon font-mono">&#9632;</span>
        <div class="stat-content">
          <span class="stat-value font-mono">{stats.total_users}</span>
          <span class="stat-label">Total Operators</span>
        </div>
      </div>
      <div class="stat-card c2-card">
        <span class="stat-icon font-mono" style="color: #ff8c00">&#9632;</span>
        <div class="stat-content">
          <span class="stat-value font-mono">{stats.total_challenges}</span>
          <span class="stat-label">Active Challenges</span>
        </div>
      </div>
      <div class="stat-card c2-card">
        <span class="stat-icon font-mono" style="color: #ff4444">&#9632;</span>
        <div class="stat-content">
          <span class="stat-value font-mono">{stats.total_submissions}</span>
          <span class="stat-label">Total Submissions</span>
        </div>
      </div>
      <div class="stat-card c2-card">
        <span class="stat-icon font-mono" style="color: #aa55ff">&#9632;</span>
        <div class="stat-content">
          <span class="stat-value font-mono">{stats.total_lessons}</span>
          <span class="stat-label">Published Reports</span>
        </div>
      </div>
      <div class="stat-card c2-card">
        <span class="stat-icon font-mono" style="color: var(--accent-blue)">&#9632;</span>
        <div class="stat-content">
          <span class="stat-value font-mono">{stats.total_solves}</span>
          <span class="stat-label">Total Solves</span>
        </div>
      </div>
      <div class="stat-card c2-card">
        <span class="stat-icon font-mono" style="color: var(--accent-green)">&#9632;</span>
        <div class="stat-content">
          <span class="stat-value font-mono">{stats.active_today}</span>
          <span class="stat-label">Active Today</span>
        </div>
      </div>
    </div>

    <div class="c2-actions">
      <h2 class="c2-section font-mono">
        <span class="c2-bracket">[</span>QUICK ACTIONS<span class="c2-bracket">]</span>
      </h2>
      <div class="actions-grid">
        <a href="/admin/challenges" class="action-card c2-card">
          <span class="action-icon font-mono">&gt;_</span>
          <div>
            <h3 class="font-mono">Challenge Forge</h3>
            <p>Deploy new vulnerable code labs into the Arena</p>
          </div>
        </a>
        <a href="/admin/lessons" class="action-card c2-card">
          <span class="action-icon font-mono">[C]</span>
          <div>
            <h3 class="font-mono">Academy Publisher</h3>
            <p>Publish classified threat reports to the Academy</p>
          </div>
        </a>
      </div>
    </div>
  </div>
{/if}

<style>
  .c2-dashboard {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .c2-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    min-height: 40vh;
  }

  .c2-pulse {
    width: 40px;
    height: 40px;
    border: 2px solid #ff6432;
    border-radius: 50%;
    animation: pulse 1.2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { transform: scale(0.8); opacity: 0.3; }
    50% { transform: scale(1.1); opacity: 1; }
  }

  .loading-text {
    font-size: 0.7rem;
    color: var(--text-tertiary);
    letter-spacing: 0.15em;
  }

  .c2-error {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 30vh;
    color: #ff4444;
    font-size: 0.85rem;
  }

  /* Header */
  .c2-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .c2-title {
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    color: var(--text-primary);
  }

  .c2-bracket {
    color: #ff6432;
  }

  .c2-section {
    font-size: 0.75rem;
    letter-spacing: 0.1em;
    color: var(--text-tertiary);
    margin-bottom: var(--space-3);
  }

  .c2-status {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.65rem;
    letter-spacing: 0.12em;
    color: var(--text-tertiary);
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .status-dot.healthy {
    background: var(--accent-green);
    box-shadow: 0 2px 6px rgba(212, 165, 116, 0.3);
  }

  .status-dot.unhealthy {
    background: var(--accent-red);
    box-shadow: 0 2px 6px rgba(201, 114, 107, 0.3);
  }

  /* Stats Grid */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-4);
  }

  .c2-card {
    background: var(--bg-secondary);
    border: 1px solid rgba(255, 100, 50, 0.12);
    border-radius: var(--radius-md);
    padding: var(--space-4);
    transition: all var(--transition-fast);
  }

  .c2-card:hover {
    border-color: rgba(255, 100, 50, 0.3);
  }

  .stat-card {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .stat-icon {
    font-size: 1.25rem;
    color: #ff6432;
    flex-shrink: 0;
  }

  .stat-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .stat-value {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .stat-label {
    font-size: 0.7rem;
    color: var(--text-secondary);
  }

  /* Actions */
  .actions-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-4);
  }

  .action-card {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
    text-decoration: none;
    color: inherit;
  }

  .action-icon {
    font-size: 1.1rem;
    color: #ff6432;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 100, 50, 0.06);
    border-radius: var(--radius-md);
    flex-shrink: 0;
  }

  .action-card h3 {
    font-size: 0.85rem;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .action-card p {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  /* Responsive */
  @media (max-width: 768px) {
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .actions-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
