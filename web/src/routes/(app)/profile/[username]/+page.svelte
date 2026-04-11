<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { getPublicProfile, type PublicProfile } from '$lib/api/leaderboard';
  import { getAllAchievements, type Achievement } from '$lib/api/achievements';
  import { getTierColor } from '$lib/utils/tiers';
  import { ApiError } from '$lib/api/client';
  import Card from '$lib/components/ui/Card.svelte';
  import RankCard from '$lib/components/dashboard/RankCard.svelte';
  import SkillRadar from '$lib/components/dashboard/SkillRadar.svelte';
  import ActivityFeed from '$lib/components/dashboard/ActivityFeed.svelte';
  import AchievementShowcase from '$lib/components/dashboard/AchievementShowcase.svelte';

  let profile: PublicProfile | null = $state(null);
  let allAchievements: Achievement[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let notFound = $state(false);

  const username = $derived(($page.params as Record<string, string>).username);

  onMount(async () => {
    try {
      profile = await getPublicProfile(username);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        notFound = true;
      } else if (e instanceof Error) {
        error = e.message;
      } else {
        error = 'Failed to load profile';
      }
    } finally {
      loading = false;
    }

    // Fetch achievements separately — non-critical, never blocks profile
    try {
      allAchievements = (await getAllAchievements()) ?? [];
    } catch {
      allAchievements = [];
    }
  });

  function profileTierColor(): string {
    if (profile) return getTierColor(profile.rank.title);
    return '#d4a574';
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
    });
  }
</script>

<svelte:head>
  <title>{profile ? `${profile.username} | VulnArena` : 'Hacker Profile | VulnArena'}</title>
</svelte:head>

{#if loading}
  <div class="prof-loading">
    <div class="loading-pulse"></div>
    <span class="font-mono loading-text">LOADING PROFILE...</span>
  </div>
{:else if notFound}
  <div class="prof-not-found">
    <div class="nf-icon font-mono">[404]</div>
    <h2 class="font-mono">OPERATOR NOT FOUND</h2>
    <p>The hacker you're looking for doesn't exist or has gone dark.</p>
    <a href="/leaderboard" class="nf-link font-mono">&lt; BACK TO LEADERBOARD</a>
  </div>
{:else if error}
  <div class="prof-error">
    <span class="font-mono">[ERROR] {error}</span>
  </div>
{:else if profile}
  <div class="profile">
    <!-- Profile Header -->
    <header class="prof-header">
      <div class="prof-avatar font-mono" style:border-color={profileTierColor()}>
        {profile.username.charAt(0).toUpperCase()}
      </div>
      <div class="prof-identity">
        <h1 class="prof-username font-mono">{profile.username}</h1>
        {#if profile.display_name}
          <span class="prof-display">{profile.display_name}</span>
        {/if}
        <span class="prof-joined font-mono">MEMBER SINCE {formatDate(profile.joined_at).toUpperCase()}</span>
      </div>
      <div class="prof-quick-stats">
        <div class="qs-item">
          <span class="qs-value font-mono accent">{profile.stats.total_points.toLocaleString()}</span>
          <span class="qs-label">XP</span>
        </div>
        <div class="qs-divider"></div>
        <div class="qs-item">
          <span class="qs-value font-mono" style="color: var(--accent-cyan)">{profile.stats.total_solved}</span>
          <span class="qs-label">Pwned</span>
        </div>
      </div>
    </header>

    <!-- Rank Section -->
    <div class="prof-grid">
      <Card variant="elevated" padding="lg">
        <div class="section-header font-mono">
          <span class="bracket">[</span>RANK<span class="bracket">]</span>
        </div>
        <RankCard rank={profile.rank} />
      </Card>

      <!-- Skill Radar -->
      <Card variant="elevated" padding="lg">
        <div class="section-header font-mono">
          <span class="bracket">[</span>SKILL RADAR<span class="bracket">]</span>
        </div>
        {#if profile.skill_radar && profile.skill_radar.length > 0}
          <SkillRadar skills={profile.skill_radar} />
        {:else}
          <div class="empty-section font-mono">
            <span class="accent">$_</span> No skill data yet.
          </div>
        {/if}
      </Card>
    </div>

    <!-- Achievement Showcase -->
    {#if allAchievements.length > 0}
      <Card variant="elevated" padding="lg">
        <div class="section-header font-mono">
          <span class="bracket">[</span>ACHIEVEMENTS<span class="bracket">]</span>
        </div>
        <AchievementShowcase unlocked={profile.achievements ?? []} all={allAchievements} />
      </Card>
    {/if}

    <!-- Pwned Labs -->
    <Card variant="elevated" padding="sm">
      <div class="section-header font-mono" style="padding: 0.5rem 0.75rem 0;">
        <span class="bracket">[</span>PWNED LABS<span class="bracket">]</span>
      </div>
      {#if profile.recent_activity && profile.recent_activity.length > 0}
        <ActivityFeed activities={profile.recent_activity} />
      {:else}
        <div class="empty-section font-mono" style="padding: 1rem 0.75rem;">
          <span class="accent">$_</span> No challenges pwned yet.
        </div>
      {/if}
    </Card>

    <!-- Back to leaderboard -->
    <a href="/leaderboard" class="back-link font-mono">&lt; BACK TO LEADERBOARD</a>
  </div>
{/if}

<style>
  .profile {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  /* Loading */
  .prof-loading {
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

  /* Not Found */
  .prof-not-found {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    min-height: 50vh;
    text-align: center;
  }

  .nf-icon {
    font-size: 2.5rem;
    font-weight: 800;
    color: var(--accent-red, #ff4444);
    text-shadow: 0 0 20px rgba(255, 68, 68, 0.3);
  }

  .prof-not-found h2 {
    font-size: 1.1rem;
    letter-spacing: 0.1em;
    color: var(--text-primary);
  }

  .prof-not-found p {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .nf-link {
    font-size: 0.75rem;
    color: var(--accent-green);
    text-decoration: none;
    letter-spacing: 0.06em;
    transition: opacity var(--transition-fast);
  }

  .nf-link:hover {
    opacity: 0.8;
  }

  /* Error */
  .prof-error {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 30vh;
    color: var(--accent-red, #ff4444);
    font-size: 0.85rem;
  }

  /* Header */
  .prof-header {
    display: flex;
    align-items: center;
    gap: var(--space-5);
    padding-bottom: var(--space-4);
    border-bottom: 1px solid var(--border-primary);
  }

  .prof-avatar {
    width: 72px;
    height: 72px;
    border-radius: 50%;
    border: 3px solid;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.5rem;
    font-weight: 800;
    color: var(--text-primary);
    background: var(--bg-secondary);
    flex-shrink: 0;
  }

  .prof-identity {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
    min-width: 0;
  }

  .prof-username {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: 0.03em;
  }

  .prof-display {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .prof-joined {
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    letter-spacing: 0.1em;
    margin-top: 2px;
  }

  .prof-quick-stats {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-shrink: 0;
  }

  .qs-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .qs-value {
    font-size: 1.5rem;
    font-weight: 700;
  }

  .qs-label {
    font-size: 0.65rem;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .qs-divider {
    width: 1px;
    height: 32px;
    background: var(--border-primary);
  }

  .accent {
    color: var(--accent-green);
  }

  /* Section headers */
  .section-header {
    font-size: 0.75rem;
    letter-spacing: 0.1em;
    color: var(--text-tertiary);
    margin-bottom: var(--space-4);
  }

  .bracket {
    color: var(--accent-green);
  }

  /* Grid */
  .prof-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--space-4);
    align-items: start;
  }

  /* Empty sections */
  .empty-section {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.8rem;
    color: var(--text-tertiary);
    padding: var(--space-4) 0;
  }

  /* Back link */
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

  /* Responsive */
  @media (max-width: 768px) {
    .prof-header {
      flex-direction: column;
      text-align: center;
    }

    .prof-quick-stats {
      justify-content: center;
    }

    .prof-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
