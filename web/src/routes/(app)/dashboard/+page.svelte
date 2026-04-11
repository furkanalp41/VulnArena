<script lang="ts">
  import { onMount } from 'svelte';
  import { currentUser } from '$lib/stores/auth';
  import { getDashboardProfile, type DashboardProfile } from '$lib/api/dashboard';
  import { getAllAchievements, type Achievement } from '$lib/api/achievements';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';
  import RankCard from '$lib/components/dashboard/RankCard.svelte';
  import SkillRadar from '$lib/components/dashboard/SkillRadar.svelte';
  import ActivityFeed from '$lib/components/dashboard/ActivityFeed.svelte';
  import AchievementShowcase from '$lib/components/dashboard/AchievementShowcase.svelte';

  let profile: DashboardProfile | null = $state(null);
  let allAchievements: Achievement[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  onMount(async () => {
    try {
      profile = await getDashboardProfile();
    } catch (e: any) {
      error = e?.message || 'Failed to load dashboard';
    } finally {
      loading = false;
    }

    // Fetch achievements separately — non-critical, never blocks dashboard
    try {
      allAchievements = (await getAllAchievements()) ?? [];
    } catch {
      allAchievements = [];
    }
  });
</script>

{#if loading}
  <div class="dash-loading">
    <div class="loading-pulse"></div>
    <span class="loading-text">Loading...</span>
  </div>
{:else if error}
  <div class="dash-error">
    <span>{error}</span>
  </div>
{:else if profile}
  <div class="dashboard">
    <!-- Header -->
    <header class="dash-header">
      <div>
        <h1 class="dash-title">
          Welcome back, <span class="accent">{$currentUser?.display_name || $currentUser?.username}</span>
        </h1>
        <p class="dash-subtitle">Dashboard</p>
      </div>
    </header>

    <!-- Top row: Rank + Stats -->
    <div class="top-row">
      <Card variant="elevated" padding="lg">
        <div class="section-header">Rank</div>
        <RankCard rank={profile.rank} />
      </Card>

      <div class="stats-grid">
        <Card variant="default">
          <div class="stat">
            <span class="stat-value accent">{profile.stats.total_solved}</span>
            <span class="stat-label">Solved</span>
            <span class="stat-sub">/ {profile.stats.total_available}</span>
          </div>
        </Card>
        <Card variant="default">
          <div class="stat">
            <span class="stat-value" style="color: var(--accent-blue)">{profile.stats.current_streak}</span>
            <span class="stat-label">Day Streak</span>
          </div>
        </Card>
        <Card variant="default">
          <div class="stat">
            <span class="stat-value" style="color: var(--accent-purple)">{profile.stats.lessons_read}</span>
            <span class="stat-label">Lessons</span>
          </div>
        </Card>
        <Card variant="default">
          <div class="stat">
            <span class="stat-value" style="color: var(--accent-yellow)">{profile.stats.average_score}<span class="stat-unit">%</span></span>
            <span class="stat-label">Avg Score</span>
          </div>
        </Card>
      </div>
    </div>

    <!-- Achievement Showcase -->
    {#if allAchievements.length > 0}
      <div class="achievements-section">
        <h2 class="section-header">Achievement Showcase</h2>
        <AchievementShowcase unlocked={profile.achievements ?? []} all={allAchievements} />
      </div>
    {/if}

    <!-- Middle row: Radar + Activity -->
    <div class="mid-row">
      <Card variant="elevated" padding="lg">
        <div class="section-header">Skill Radar</div>
        <SkillRadar skills={profile.skill_radar ?? []} />
      </Card>

      <Card variant="elevated" padding="sm">
        <div class="section-header" style="padding: 0.5rem 0.75rem 0;">Recent Activity</div>
        <ActivityFeed activities={profile.recent_activity ?? []} />
      </Card>
    </div>

    <!-- Jump Back In -->
    {#if profile.next_challenge}
      <div class="jump-section">
        <h2 class="section-header">Jump Back In</h2>
        <a href="/arena/{profile.next_challenge.id}" class="jump-card">
          <Card variant="bordered">
            <div class="jump-content">
              <div class="jump-left">
                <div class="jump-info">
                  <h3>{profile.next_challenge.title}</h3>
                  <div class="jump-meta">
                    <DifficultyBadge level={profile.next_challenge.difficulty} />
                    <span class="jump-lang">{profile.next_challenge.language?.name ?? ''}</span>
                    <span class="jump-cat">{profile.next_challenge.vuln_category?.name ?? ''}</span>
                    <span class="jump-pts">+{profile.next_challenge.points} XP</span>
                  </div>
                </div>
              </div>
              <Button variant="primary" size="sm">Start Challenge</Button>
            </div>
          </Card>
        </a>
      </div>
    {:else}
      <Card variant="bordered">
        <div class="all-done">
          <span class="accent">All challenges completed. More incoming.</span>
        </div>
      </Card>
    {/if}

    <!-- Quick Links -->
    <div class="quick-links">
      <a href="/arena" class="action-card">
        <Card variant="bordered">
          <div class="action-content">
            <h3>Arena</h3>
            <p>Browse all vulnerability challenges</p>
          </div>
        </Card>
      </a>
      <a href="/academy" class="action-card">
        <Card variant="bordered">
          <div class="action-content">
            <h3>Academy</h3>
            <p>Deep-dive into secure coding lessons</p>
          </div>
        </Card>
      </a>
    </div>
  </div>
{/if}

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  /* Loading & Error */
  .dash-loading {
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

  .dash-error {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 30vh;
    color: var(--accent-red, #ff4444);
    font-size: 0.85rem;
  }

  /* Header */
  .dash-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
  }

  .dash-title {
    font-size: 1.5rem;
    font-weight: 600;
  }

  .accent {
    color: var(--accent-green);
  }

  .dash-subtitle {
    font-size: 0.875rem;
    color: var(--text-tertiary);
    margin-top: var(--space-1);
  }

  /* Section headers */
  .section-header {
    font-family: var(--font-serif);
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: var(--space-4);
  }

  /* Top row */
  .top-row {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--space-4);
    align-items: stretch;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--space-4);
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    padding: var(--space-2);
  }

  .stat-value {
    font-size: 1.75rem;
    font-weight: 700;
  }

  .stat-unit {
    font-size: 1rem;
    opacity: 0.6;
  }

  .stat-label {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .stat-sub {
    font-size: 0.7rem;
    color: var(--text-tertiary);
  }

  /* Middle row */
  .mid-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
    align-items: start;
  }

  /* Jump back in */
  .jump-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .jump-card {
    text-decoration: none;
    color: inherit;
  }

  .jump-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
  }

  .jump-left {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    min-width: 0;
  }

  .jump-info {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
  }

  .jump-info h3 {
    font-family: var(--font-serif);
    font-size: 0.95rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .jump-meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    font-size: 0.7rem;
    color: var(--text-tertiary);
  }

  .jump-lang {
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .jump-pts {
    color: var(--accent-green);
  }

  /* All done */
  .all-done {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2);
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  /* Quick links */
  .quick-links {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: var(--space-4);
  }

  .action-card {
    text-decoration: none;
    color: inherit;
  }

  .action-content {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .action-content h3 {
    font-family: var(--font-serif);
    font-size: 0.9rem;
    color: var(--text-primary);
  }

  .action-content p {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  /* Responsive */
  @media (max-width: 900px) {
    .top-row {
      grid-template-columns: 1fr;
    }

    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .mid-row {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 600px) {
    .stats-grid {
      grid-template-columns: 1fr 1fr;
    }

    .jump-content {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
