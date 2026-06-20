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

  function getPodiumClass(rank: number): string {
    if (rank === 1) return 'p1';
    if (rank === 2) return 'p2';
    if (rank === 3) return 'p3';
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
  <div class="shell page leaderboard">
    <div class="page-head">
      <span class="eyebrow">Standings</span>
      <h1 class="lb-title">Leaderboard</h1>
    </div>

    <div class="tabs" role="tablist">
      <button
        class="tab"
        class:is-on={activeTab === 'hackers'}
        role="tab"
        aria-selected={activeTab === 'hackers'}
        onclick={() => switchTab('hackers')}
      >Global</button>
      <button
        class="tab"
        class:is-on={activeTab === 'squads'}
        role="tab"
        aria-selected={activeTab === 'squads'}
        onclick={() => switchTab('squads')}
      >Squads</button>
    </div>

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
        <div class="podium">
          {#each squadEntries.slice(0, 3) as entry (entry.rank)}
            {@const podium = getPodiumClass(entry.rank)}
            <a
              href="/teams/{entry.tag}"
              class="podium-card {podium}"
            >
              <div class="podium-rank osf">{entry.rank}</div>
              <div class="podium-name">{entry.team_name}</div>
              <div class="podium-meta">
                <span class="tier">{entry.tag}</span> · {entry.total_xp.toLocaleString()} XP · {entry.total_solved} solved
              </div>
            </a>
          {/each}
        </div>

        <table class="standings">
          <thead>
            <tr>
              <th class="num">Rank</th>
              <th>Squad</th>
              <th class="hide-sm">Members</th>
              <th class="num">XP</th>
              <th class="num">Solved</th>
            </tr>
          </thead>
          <tbody>
            {#each squadEntries as entry (entry.rank)}
              {@const podium = getPodiumClass(entry.rank)}
              <tr
                class="lb-row {podium ? 'podium ' + podium : ''}"
                onclick={() => (window.location.href = `/teams/${entry.tag}`)}
              >
                <td class="num rk">{entry.rank}</td>
                <td class="op">
                  <a class="row-link" href="/teams/{entry.tag}">
                    {entry.team_name} <span class="squad-tag">{entry.tag}</span>
                  </a>
                </td>
                <td class="hide-sm tier tnum">{entry.member_count}</td>
                <td class="num xp">{entry.total_xp.toLocaleString()}</td>
                <td class="num solved">{entry.total_solved}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {:else if entries.length === 0}
      <Card variant="bordered">
        <div class="empty-state">
          <span>No rankings yet. Be the first to solve a challenge.</span>
        </div>
      </Card>
    {:else}
      <div class="podium">
        {#each entries.slice(0, 3) as entry (entry.rank)}
          {@const podium = getPodiumClass(entry.rank)}
          {@const tierColor = getTierColor(entry.rank_title)}
          <a
            href="/profile/{entry.username}"
            class="podium-card {podium}"
          >
            <div class="podium-rank osf">{entry.rank}</div>
            <div class="podium-name">{entry.username}</div>
            <div class="podium-meta">
              <span class="tier" style:color={tierColor}>T{entry.tier}</span> · {entry.total_xp.toLocaleString()} XP · {entry.total_solved} solved
            </div>
          </a>
        {/each}
      </div>

      <table class="standings">
        <thead>
          <tr>
            <th class="num">Rank</th>
            <th>Operator</th>
            <th class="hide-sm">Tier</th>
            <th class="num">XP</th>
            <th class="num">Solved</th>
          </tr>
        </thead>
        <tbody>
          {#each entries as entry (entry.rank)}
            {@const podium = getPodiumClass(entry.rank)}
            {@const tierColor = getTierColor(entry.rank_title)}
            <tr
              class="lb-row {podium ? 'podium ' + podium : ''}"
              onclick={() => (window.location.href = `/profile/${entry.username}`)}
            >
              <td class="num rk">{entry.rank}</td>
              <td class="op">
                <a class="row-link" href="/profile/{entry.username}">
                  {entry.username}
                  {#if entry.display_name}
                    <span class="op-display">{entry.display_name}</span>
                  {/if}
                </a>
              </td>
              <td class="hide-sm tier">
                <span style:color={tierColor}>T{entry.tier}</span>
                <span class="tier-title">{entry.rank_title}</span>
              </td>
              <td class="num xp">{entry.total_xp.toLocaleString()}</td>
              <td class="num solved">{entry.total_solved}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
{/if}

<style>
  .leaderboard {
    padding: var(--space-7) 0 var(--space-8);
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
    border: 1px solid var(--border-secondary);
    border-radius: 50%;
    animation: pulse 1.2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { transform: scale(0.8); opacity: 0.3; }
    50% { transform: scale(1.1); opacity: 1; }
  }

  .loading-text {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    letter-spacing: 0.04em;
    color: var(--text-tertiary);
  }

  .lb-error {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 30vh;
    color: var(--accent-red);
    font-size: var(--fs-body);
  }

  /* Header */
  .page-head {
    margin-bottom: var(--space-6);
  }

  .lb-title {
    font-family: var(--font-serif);
    font-size: var(--fs-h1);
    font-weight: 700;
    letter-spacing: -0.015em;
    color: var(--text-primary);
    margin-top: var(--space-1);
  }

  /* Tabs — mono small-caps underline */
  .tabs {
    display: flex;
    gap: var(--space-4);
    border-bottom: 1px solid var(--border-primary);
    margin-bottom: var(--space-6);
  }

  .tab {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--text-tertiary);
    padding: 0 0 var(--space-3);
    background: none;
    border: 0;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    cursor: pointer;
    transition: color 0.15s ease;
  }

  .tab.is-on {
    color: var(--text-primary);
    border-color: var(--accent-primary);
  }

  .tab:not(.is-on):hover {
    color: var(--text-secondary);
  }

  /* Empty state */
  .empty-state {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4);
    font-size: var(--fs-body);
    color: var(--text-secondary);
  }

  /* Podium — cards with 3px earthy left-rule accent */
  .podium {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-4);
    margin-bottom: var(--space-7);
  }

  .podium-card {
    border: 1px solid var(--border-primary);
    border-left-width: 3px;
    border-radius: var(--radius-card);
    padding: var(--space-4) var(--space-5);
    background: var(--bg-surface);
    text-decoration: none;
    color: inherit;
    transition: border-color 0.15s ease;
  }

  .podium-card:hover {
    border-color: var(--border-secondary);
  }

  .podium-card.p1 { border-left-color: var(--accent-primary); }
  .podium-card.p2 { border-left-color: var(--accent-blue); }
  .podium-card.p3 { border-left-color: var(--accent-purple); }

  .podium-rank {
    font-family: var(--font-serif);
    font-size: 2.25rem;
    line-height: 1;
    color: var(--text-tertiary);
    font-variant-numeric: oldstyle-nums;
    float: right;
    margin-left: var(--space-3);
  }

  .podium-card.p1 .podium-rank { color: var(--accent-primary); }
  .podium-card.p2 .podium-rank { color: var(--accent-blue); }
  .podium-card.p3 .podium-rank { color: var(--accent-purple); }

  .podium-name {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .podium-meta {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
    margin-top: var(--space-1);
  }

  .podium-meta .tier {
    color: var(--text-tertiary);
  }

  /* Standings table */
  .standings {
    width: 100%;
    border-collapse: collapse;
  }

  .standings thead th {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    text-transform: uppercase;
    letter-spacing: 0.13em;
    color: var(--text-tertiary);
    font-weight: 500;
    text-align: left;
    padding: 0 var(--space-4) var(--space-3);
    border-bottom: 1px solid var(--border-secondary);
  }

  .standings th.num,
  .standings td.num {
    text-align: right;
    font-variant-numeric: tabular-nums;
    font-family: var(--font-mono);
  }

  .standings tbody td {
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--border-primary);
    vertical-align: baseline;
  }

  .lb-row {
    cursor: pointer;
    transition: background 0.15s ease;
  }

  .standings tbody tr:hover td {
    background: var(--bg-hover);
  }

  /* Podium-rank rows carry the same earthy left accent */
  .lb-row.podium td:first-child {
    box-shadow: inset 3px 0 0 var(--accent-primary);
  }
  .lb-row.p2 td:first-child {
    box-shadow: inset 3px 0 0 var(--accent-blue);
  }
  .lb-row.p3 td:first-child {
    box-shadow: inset 3px 0 0 var(--accent-purple);
  }

  .standings .rk {
    font-family: var(--font-mono);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
    width: 3rem;
  }

  .standings .op {
    font-family: var(--font-serif);
    font-size: 1.05rem;
    font-weight: 500;
  }

  .row-link {
    color: var(--text-primary);
    text-decoration: none;
    display: inline-flex;
    align-items: baseline;
    gap: 0.5em;
  }

  .row-link:hover {
    color: var(--accent-primary);
  }

  .squad-tag {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
  }

  .op-display {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.04em;
    color: var(--text-tertiary);
  }

  .standings .tier {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    color: var(--text-secondary);
  }

  .tier-title {
    color: var(--text-tertiary);
    margin-left: 0.5em;
  }

  .standings .xp { color: var(--text-primary); }
  .standings .solved { color: var(--text-secondary); }

  /* Responsive */
  @media (max-width: 768px) {
    .podium {
      grid-template-columns: 1fr;
    }

    .standings .hide-sm,
    .tier-title {
      display: none;
    }
  }
</style>
