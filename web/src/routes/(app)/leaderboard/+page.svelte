<script lang="ts">
  import { onMount } from 'svelte';
  import { getLeaderboard, type LeaderboardEntry } from '$lib/api/leaderboard';
  import { getTeamLeaderboard, type TeamLeaderboardEntry } from '$lib/api/teams';
  import { getTierColor } from '$lib/utils/tiers';
  import Card from '$lib/components/ui/Card.svelte';

  let activeTab: 'hackers' | 'squads' = $state('hackers');
  let entries: LeaderboardEntry[] = $state([]);
  let squadEntries: TeamLeaderboardEntry[] = $state([]);
  let loading = $state(true);
  let squadLoading = $state(false);
  let error = $state('');

  onMount(async () => {
    try {
      entries = await getLeaderboard();
    } catch (e: any) {
      error = e?.message || 'Failed to load leaderboard';
    } finally {
      loading = false;
    }
  });

  async function switchTab(tab: 'hackers' | 'squads') {
    activeTab = tab;
    if (tab === 'squads' && squadEntries.length === 0) {
      squadLoading = true;
      try {
        squadEntries = await getTeamLeaderboard();
      } catch (e: any) {
        error = e?.message || 'Failed to load squad rankings';
      } finally {
        squadLoading = false;
      }
    }
  }

  const podiumColors = ['#ffd700', '#c0c0c0', '#cd7f32'];

  function getPodiumClass(rank: number): string {
    if (rank === 1) return 'gold';
    if (rank === 2) return 'silver';
    if (rank === 3) return 'bronze';
    return '';
  }
</script>

<svelte:head>
  <title>Leaderboard | VulnArena</title>
</svelte:head>

{#if loading}
  <div class="lb-loading">
    <div class="loading-pulse"></div>
    <span class="loading-text">Loading rankings...</span>
  </div>
{:else if error}
  <div class="lb-error">
    <span>{error}</span>
  </div>
{:else}
  <div class="leaderboard">
    <header class="lb-header">
      <h1 class="lb-title">Leaderboard</h1>
      <div class="tab-bar">
        <button
          class="tab-btn"
          class:active={activeTab === 'hackers'}
          onclick={() => switchTab('hackers')}
        >Global hackers</button>
        <button
          class="tab-btn"
          class:active={activeTab === 'squads'}
          onclick={() => switchTab('squads')}
        >Top squads</button>
      </div>
    </header>

    {#if activeTab === 'squads'}
      {#if squadLoading}
        <div class="lb-loading" style="min-height: 20vh">
          <div class="loading-pulse"></div>
          <span class="loading-text">Loading squads...</span>
        </div>
      {:else if squadEntries.length === 0}
        <Card variant="bordered">
          <div class="empty-state">
            <span>No squads ranked yet. Form a team and start competing.</span>
          </div>
        </Card>
      {:else}
        <div class="lb-table">
          <div class="lb-row squad-header-row">
            <span class="col-rank">#</span>
            <span class="col-user">Squad</span>
            <span class="col-tier">Members</span>
            <span class="col-xp">XP</span>
            <span class="col-solved">Solved</span>
          </div>
          {#each squadEntries as entry (entry.rank)}
            {@const podium = getPodiumClass(entry.rank)}
            <a href="/teams/{entry.tag}" class="lb-row lb-data-row {podium ? 'podium ' + podium : ''}">
              <span class="col-rank">
                {#if entry.rank <= 3}
                  <span class="rank-medal" style:color={podiumColors[entry.rank - 1]}>{entry.rank}</span>
                {:else}
                  <span class="rank-num">{entry.rank}</span>
                {/if}
              </span>
              <span class="col-user">
                <span class="squad-tag">{entry.tag}</span>
                <span class="user-name">{entry.team_name}</span>
              </span>
              <span class="col-tier">{entry.member_count}</span>
              <span class="col-xp">
                <span class="xp-value">{entry.total_xp.toLocaleString()}</span>
                <span class="xp-label">XP</span>
              </span>
              <span class="col-solved">
                <span class="solved-value">{entry.total_solved}</span>
              </span>
            </a>
          {/each}
        </div>
      {/if}
    {:else if entries.length === 0}
      <Card variant="bordered">
        <div class="empty-state">
          <span>No rankings yet. Be the first to solve a challenge.</span>
        </div>
      </Card>
    {:else}
      <div class="lb-table">
        <div class="lb-row lb-header-row">
          <span class="col-rank">#</span>
          <span class="col-user">Player</span>
          <span class="col-tier">Tier</span>
          <span class="col-xp">XP</span>
          <span class="col-solved">Solved</span>
        </div>

        {#each entries as entry (entry.rank)}
          {@const podium = getPodiumClass(entry.rank)}
          {@const tierColor = getTierColor(entry.rank_title)}
          <a href="/profile/{entry.username}" class="lb-row lb-data-row {podium ? 'podium ' + podium : ''}">
            <span class="col-rank">
              {#if entry.rank <= 3}
                <span class="rank-medal" style:color={podiumColors[entry.rank - 1]}>
                  {entry.rank}
                </span>
              {:else}
                <span class="rank-num">{entry.rank}</span>
              {/if}
            </span>

            <span class="col-user">
              <span class="user-avatar" style:border-color={tierColor}>
                {entry.username.charAt(0).toUpperCase()}
              </span>
              <span class="user-info">
                <span class="user-name">{entry.username}</span>
                {#if entry.display_name}
                  <span class="user-display">{entry.display_name}</span>
                {/if}
              </span>
            </span>

            <span class="col-tier">
              <span class="tier-badge" style:color={tierColor} style:border-color={tierColor}>
                T{entry.tier}
              </span>
              <span class="tier-title" style:color={tierColor}>{entry.rank_title}</span>
            </span>

            <span class="col-xp">
              <span class="xp-value">{entry.total_xp.toLocaleString()}</span>
              <span class="xp-label">XP</span>
            </span>

            <span class="col-solved">
              <span class="solved-value">{entry.total_solved}</span>
            </span>
          </a>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .leaderboard {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  /* Loading & Error */
  .lb-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    min-height: 50vh;
  }

  .loading-pulse {
    width: 40px;
    height: 40px;
    border: 2px solid var(--border-secondary);
    border-radius: 50%;
    animation: pulse 1.2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { transform: scale(0.8); opacity: 0.3; }
    50% { transform: scale(1.1); opacity: 1; }
  }

  .loading-text {
    font-family: var(--font-sans);
    font-size: 0.7rem;
    color: var(--text-tertiary);
  }

  .lb-error {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 30vh;
    color: var(--accent-red);
    font-size: 0.85rem;
  }

  /* Header */
  .lb-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .lb-title {
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .tab-bar {
    display: flex;
    gap: 0;
    margin-top: var(--space-3);
  }

  .tab-btn {
    font-family: var(--font-sans);
    font-size: 0.7rem;
    padding: var(--space-2) var(--space-4);
    background: transparent;
    border: 1px solid var(--border-primary);
    color: var(--text-tertiary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .tab-btn:first-child {
    border-radius: var(--radius-sm) 0 0 var(--radius-sm);
    border-right: none;
  }

  .tab-btn:last-child {
    border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  }

  .tab-btn.active {
    background: var(--accent-primary);
    border-color: var(--accent-primary);
    color: var(--text-inverse);
  }

  .tab-btn:not(.active):hover {
    border-color: var(--text-secondary);
    color: var(--text-secondary);
  }

  .squad-tag {
    font-family: var(--font-sans);
    font-size: 0.7rem;
    color: var(--text-secondary);
    font-weight: 700;
    margin-right: var(--space-2);
  }

  .squad-header-row {
    height: 44px;
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    border-bottom: 1px solid var(--border-primary);
    background: var(--bg-tertiary, var(--bg-secondary));
  }

  /* Empty state */
  .empty-state {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4);
    font-size: 0.85rem;
    color: var(--text-secondary);
  }


  /* Table */
  .lb-table {
    display: flex;
    flex-direction: column;
    background: var(--bg-secondary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .lb-row {
    display: grid;
    grid-template-columns: 60px 1fr 160px 100px 80px;
    align-items: center;
    padding: 0 var(--space-4);
    text-decoration: none;
    color: inherit;
  }

  .lb-header-row {
    height: 44px;
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    border-bottom: 1px solid var(--border-primary);
    background: var(--bg-tertiary, var(--bg-secondary));
  }

  .lb-data-row {
    height: 64px;
    border-bottom: 1px solid var(--border-primary);
    transition: background var(--transition-fast);
  }

  .lb-data-row:last-child {
    border-bottom: none;
  }

  .lb-data-row:hover {
    background: var(--bg-hover);
  }

  /* Podium rows */
  .lb-data-row.podium {
    position: relative;
  }

  .lb-data-row.gold {
    border-left: 3px solid #ffd700;
    background: rgba(255, 215, 0, 0.03);
  }

  .lb-data-row.gold:hover {
    background: rgba(255, 215, 0, 0.06);
  }

  .lb-data-row.silver {
    border-left: 3px solid #c0c0c0;
    background: rgba(192, 192, 192, 0.02);
  }

  .lb-data-row.silver:hover {
    background: rgba(192, 192, 192, 0.05);
  }

  .lb-data-row.bronze {
    border-left: 3px solid #cd7f32;
    background: rgba(205, 127, 50, 0.02);
  }

  .lb-data-row.bronze:hover {
    background: rgba(205, 127, 50, 0.05);
  }

  /* Columns */
  .col-rank {
    text-align: center;
  }

  .rank-medal {
    font-size: 1.1rem;
    font-weight: 800;
  }

  .rank-num {
    font-size: 0.85rem;
    color: var(--text-tertiary);
  }

  .col-user {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .user-avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    border: 2px solid;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8rem;
    font-weight: 700;
    color: var(--text-primary);
    background: var(--bg-primary);
    flex-shrink: 0;
  }

  .user-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .user-name {
    font-family: var(--font-sans);
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-display {
    font-size: 0.7rem;
    color: var(--text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .col-tier {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .tier-badge {
    font-family: var(--font-sans);
    font-size: 0.65rem;
    font-weight: 700;
    padding: 2px 6px;
    border: 1px solid;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
  }

  .tier-title {
    font-family: var(--font-sans);
    font-size: 0.7rem;
    white-space: nowrap;
  }

  .col-xp {
    display: flex;
    align-items: baseline;
    gap: 4px;
  }

  .xp-value {
    font-size: 0.9rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .xp-label {
    font-family: var(--font-sans);
    font-size: 0.6rem;
    color: var(--text-tertiary);
  }

  .col-solved {
    text-align: center;
  }

  .solved-value {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  /* Responsive */
  @media (max-width: 768px) {
    .lb-row {
      grid-template-columns: 40px 1fr 80px 70px;
      padding: 0 var(--space-2);
    }

    .col-solved {
      display: none;
    }

    .tier-title {
      display: none;
    }

    .lb-data-row {
      height: 56px;
    }
  }

  @media (max-width: 480px) {
    .lb-row {
      grid-template-columns: 32px 1fr 60px;
    }

    .col-xp {
      display: none;
    }

    .user-avatar {
      width: 28px;
      height: 28px;
      font-size: 0.65rem;
    }
  }
</style>
