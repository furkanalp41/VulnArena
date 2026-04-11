<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { getTeam, joinTeam, leaveTeam, type TeamWithMembers } from '$lib/api/teams';
  import { currentUser } from '$lib/stores/auth';
  import { ApiError } from '$lib/api/client';
  import Card from '$lib/components/ui/Card.svelte';

  let team = $state<TeamWithMembers | null>(null);
  let loading = $state(true);
  let error = $state('');
  let notFound = $state(false);
  let actionMsg = $state('');

  const tag = $derived(($page.params as Record<string, string>).tag);

  const isMember = $derived(
    (() => {
      if (!team || !$currentUser) return false;
      return team.members.some((m) => m.username === $currentUser?.username);
    })(),
  );

  onMount(async () => {
    try {
      team = await getTeam(tag);
      if (!team) notFound = true;
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        notFound = true;
      } else if (e instanceof Error) {
        error = e.message;
      }
    } finally {
      loading = false;
    }
  });

  async function handleJoin() {
    actionMsg = '';
    try {
      await joinTeam(tag);
      team = await getTeam(tag);
      actionMsg = 'Joined squad!';
    } catch (e: any) {
      actionMsg = e?.message || 'Failed to join';
    }
  }

  async function handleLeave() {
    actionMsg = '';
    try {
      await leaveTeam();
      team = await getTeam(tag);
      actionMsg = 'Left squad.';
    } catch (e: any) {
      actionMsg = e?.message || 'Failed to leave';
    }
  }
</script>

<svelte:head>
  <title>{team ? `[${team.team.tag}] ${team.team.name}` : 'Squad'} | VulnArena</title>
</svelte:head>

{#if loading}
  <div class="loading-state">
    <div class="loading-pulse"></div>
    <span class="font-mono loading-text">LOADING SQUAD...</span>
  </div>
{:else if notFound}
  <div class="not-found">
    <div class="nf-icon font-mono">[404]</div>
    <h2 class="font-mono">SQUAD NOT FOUND</h2>
    <p>This squad tag doesn't exist or has been dissolved.</p>
    <a href="/teams" class="nf-link font-mono">&lt; BACK TO SQUADS</a>
  </div>
{:else if error}
  <div class="error-state font-mono">[ERROR] {error}</div>
{:else if team}
  <div class="squad-profile">
    <header class="squad-header">
      <div class="squad-identity">
        <div class="squad-tag-badge font-mono">[{team.team.tag}]</div>
        <h1 class="squad-name font-mono">{team.team.name}</h1>
        {#if team.team.description}
          <p class="squad-desc">{team.team.description}</p>
        {/if}
      </div>
      <div class="squad-stats">
        <div class="stat-item">
          <span class="stat-value font-mono accent">{team.total_xp.toLocaleString()}</span>
          <span class="stat-label">TOTAL XP</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-value font-mono" style="color: var(--accent-cyan)">{team.total_solved}</span>
          <span class="stat-label">PWNED</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-value font-mono">{team.members.length}</span>
          <span class="stat-label">MEMBERS</span>
        </div>
      </div>
    </header>

    <div class="squad-actions">
      {#if isMember}
        <button class="action-btn leave-btn font-mono" onclick={handleLeave}>LEAVE SQUAD</button>
      {:else}
        <button class="action-btn join-btn font-mono" onclick={handleJoin}>JOIN SQUAD</button>
      {/if}
      {#if actionMsg}
        <span class="action-msg font-mono">{actionMsg}</span>
      {/if}
    </div>

    <Card variant="elevated" padding="lg">
      <div class="section-header font-mono">
        <span class="bracket">[</span>MEMBERS<span class="bracket">]</span>
      </div>
      <div class="member-list">
        {#each team.members as member}
          <a href="/profile/{member.username}" class="member-row">
            <div class="member-avatar font-mono">
              {member.username.charAt(0).toUpperCase()}
            </div>
            <div class="member-info">
              <span class="member-name font-mono">{member.username}</span>
              {#if member.display_name}
                <span class="member-display">{member.display_name}</span>
              {/if}
            </div>
            <span class="member-role font-mono" class:leader={member.role === 'leader'}>
              {member.role.toUpperCase()}
            </span>
          </a>
        {/each}
      </div>
    </Card>

    <a href="/teams" class="back-link font-mono">&lt; BACK TO SQUADS</a>
  </div>
{/if}

<style>
  .squad-profile {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .loading-state, .not-found, .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    min-height: 50vh;
    text-align: center;
  }

  .loading-pulse {
    width: 40px;
    height: 40px;
    border: 2px solid var(--accent-green);
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

  .nf-icon {
    font-size: 2.5rem;
    font-weight: 800;
    color: var(--accent-red, #ff4444);
  }

  .not-found h2 {
    font-size: 1.1rem;
    letter-spacing: 0.1em;
    color: var(--text-primary);
  }

  .not-found p {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .nf-link {
    font-size: 0.75rem;
    color: var(--accent-green);
    text-decoration: none;
    letter-spacing: 0.06em;
  }

  .error-state {
    color: var(--accent-red, #ff4444);
    font-size: 0.85rem;
  }

  /* Header */
  .squad-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-5);
    padding-bottom: var(--space-4);
    border-bottom: 1px solid var(--border-primary);
  }

  .squad-identity {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .squad-tag-badge {
    font-size: 0.8rem;
    letter-spacing: 0.12em;
    color: var(--accent-green);
    font-weight: 700;
  }

  .squad-name {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .squad-desc {
    font-size: 0.85rem;
    color: var(--text-secondary);
    line-height: 1.4;
    max-width: 500px;
  }

  .squad-stats {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-shrink: 0;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .stat-value {
    font-size: 1.5rem;
    font-weight: 700;
  }

  .stat-label {
    font-size: 0.6rem;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .stat-divider {
    width: 1px;
    height: 32px;
    background: var(--border-primary);
  }

  .accent { color: var(--accent-green); }
  .bracket { color: var(--accent-green); }

  /* Actions */
  .squad-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .action-btn {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.06em;
    padding: var(--space-2) var(--space-4);
    border: 1px solid;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .join-btn {
    background: var(--accent-green);
    color: var(--text-inverse);
    border-color: var(--accent-green);
  }

  .join-btn:hover {
    box-shadow: var(--shadow-glow-green);
  }

  .leave-btn {
    background: transparent;
    color: var(--accent-red, #ff4444);
    border-color: var(--accent-red, #ff4444);
  }

  .leave-btn:hover {
    background: rgba(255, 68, 68, 0.08);
  }

  .action-msg {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  /* Section header */
  .section-header {
    font-size: 0.75rem;
    letter-spacing: 0.1em;
    color: var(--text-tertiary);
    margin-bottom: var(--space-4);
  }

  /* Member list */
  .member-list {
    display: flex;
    flex-direction: column;
  }

  .member-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-2);
    border-bottom: 1px solid var(--border-primary);
    text-decoration: none;
    color: inherit;
    transition: background var(--transition-fast);
  }

  .member-row:last-child {
    border-bottom: none;
  }

  .member-row:hover {
    background: rgba(0, 255, 136, 0.03);
  }

  .member-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    border: 2px solid var(--border-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--text-primary);
    background: var(--bg-primary);
    flex-shrink: 0;
  }

  .member-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .member-name {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .member-display {
    font-size: 0.7rem;
    color: var(--text-tertiary);
  }

  .member-role {
    font-size: 0.6rem;
    letter-spacing: 0.1em;
    color: var(--text-tertiary);
    padding: 2px 8px;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
  }

  .member-role.leader {
    color: var(--accent-green);
    border-color: var(--accent-green);
  }

  .back-link {
    font-size: 0.75rem;
    color: var(--text-tertiary);
    text-decoration: none;
    letter-spacing: 0.06em;
    transition: color var(--transition-fast);
  }

  .back-link:hover {
    color: var(--accent-green);
  }

  @media (max-width: 768px) {
    .squad-header {
      flex-direction: column;
    }

    .squad-stats {
      justify-content: center;
    }
  }
</style>
