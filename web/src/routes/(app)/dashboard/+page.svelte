<script lang="ts">
  import { onMount } from 'svelte';
  import { currentUser } from '$lib/stores/auth';
  import { getDashboardProfile, type DashboardProfile } from '$lib/api/dashboard';
  import { getAllAchievements, type Achievement } from '$lib/api/achievements';
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
  <div class="shell page">
    <!-- Page head -->
    <div class="page-head">
      <span class="eyebrow">Operator · {$currentUser?.display_name || $currentUser?.username}</span>
      <h1>Dashboard</h1>
    </div>

    <!-- Career ledger -->
    <div class="ledger" role="group" aria-label="Career figures">
      <div class="ledger-cell">
        <div class="lab">Solved</div>
        <div class="fig tnum">{profile.stats.total_solved}<small>/ {profile.stats.total_available}</small></div>
      </div>
      <div class="ledger-cell">
        <div class="lab">Streak</div>
        <div class="fig tnum">{profile.stats.current_streak}<small>days</small></div>
      </div>
      <div class="ledger-cell">
        <div class="lab">Lessons</div>
        <div class="fig tnum">{profile.stats.lessons_read}</div>
      </div>
      <div class="ledger-cell">
        <div class="lab">Avg Score</div>
        <div class="fig tnum">{profile.stats.average_score}<small>%</small></div>
      </div>
    </div>

    <!-- Two-column dashboard grid -->
    <div class="dash-grid">
      <div class="dash-col">
        <!-- Rank -->
        <div class="card">
          <div class="section-header"><h3>Rank</h3><span class="smallcaps">tier gauge</span></div>
          <RankCard rank={profile.rank} />
        </div>

        <!-- Recent Activity -->
        <div class="card">
          <div class="section-header"><h3>Recent activity</h3><span class="smallcaps">$ log --tail</span></div>
          <ActivityFeed activities={profile.recent_activity ?? []} />
        </div>
      </div>

      <div class="dash-col">
        <!-- Skill Radar -->
        <div class="card">
          <div class="section-header"><h3>Skill profile</h3><span class="smallcaps">six axes</span></div>
          <SkillRadar skills={profile.skill_radar ?? []} />
        </div>

        <!-- Achievement Showcase -->
        {#if allAchievements.length > 0}
          <div class="card">
            <div class="section-header"><h3>Achievements</h3><span class="smallcaps">earned · {(profile.achievements ?? []).length}</span></div>
            <AchievementShowcase unlocked={profile.achievements ?? []} all={allAchievements} />
          </div>
        {/if}

        <!-- Jump Back In -->
        {#if profile.next_challenge}
          <div class="card">
            <div class="section-header"><h3>Jump back in</h3><span class="smallcaps">next audit</span></div>
            <a href="/arena/{profile.next_challenge.id}" class="jump-card">
              <div class="jump-content">
                <div class="jump-info">
                  <h3 class="jump-title">{profile.next_challenge.title}</h3>
                  <div class="jump-meta">
                    <DifficultyBadge level={profile.next_challenge.difficulty} />
                    <span class="jump-lang">{profile.next_challenge.language?.name ?? ''}</span>
                    <span class="jump-cat">{profile.next_challenge.vuln_category?.name ?? ''}</span>
                    <span class="jump-pts tnum">+{profile.next_challenge.points} XP</span>
                  </div>
                </div>
                <Button variant="primary" size="sm">Start Challenge</Button>
              </div>
            </a>
          </div>
        {:else}
          <div class="card">
            <div class="all-done">All challenges completed. More incoming.</div>
          </div>
        {/if}

        <!-- Quick Links -->
        <div class="card">
          <div class="section-header"><h3>Jump to</h3><span class="smallcaps">sections</span></div>
          <div class="quick-links">
            <a href="/arena" class="action-card">
              <h3>Arena</h3>
              <p>Browse all vulnerability challenges</p>
            </a>
            <a href="/academy" class="action-card">
              <h3>Academy</h3>
              <p>Deep-dive into secure coding lessons</p>
            </a>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Page shell */
  .page {
    padding: var(--space-7) 0 var(--space-8);
  }

  .page-head {
    margin-bottom: var(--space-6);
  }

  .page-head h1 {
    font-family: var(--font-serif);
    font-size: var(--fs-h1);
    font-weight: 600;
    letter-spacing: -0.015em;
    margin-top: var(--space-1);
    color: var(--text-primary);
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
    border: 2px solid var(--accent-primary);
    border-radius: 50%;
    animation: pulse 1.2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { transform: scale(0.8); opacity: 0.3; }
    50% { transform: scale(1.1); opacity: 1; }
  }

  .loading-text {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    text-transform: uppercase;
    color: var(--text-tertiary);
    letter-spacing: 0.15em;
  }

  .dash-error {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 30vh;
    color: var(--accent-red);
    font-size: var(--fs-micro);
  }

  /* Career ledger */
  .ledger {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-card);
    overflow: hidden;
    margin-bottom: var(--space-6);
  }

  .ledger-cell {
    padding: var(--space-4) var(--space-5);
    border-right: 1px solid var(--border-primary);
  }

  .ledger-cell:last-child {
    border-right: 0;
  }

  .ledger-cell .lab {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    text-transform: uppercase;
    letter-spacing: 0.13em;
    color: var(--text-tertiary);
  }

  .ledger-cell .fig {
    font-family: var(--font-serif);
    font-size: 1.9rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums oldstyle-nums;
    margin-top: var(--space-2);
    letter-spacing: -0.01em;
    color: var(--text-primary);
  }

  .ledger-cell .fig small {
    font-family: var(--font-mono);
    font-size: 0.7rem;
    color: var(--text-tertiary);
    font-weight: 400;
    margin-left: 0.2em;
  }

  /* Dashboard grid */
  .dash-grid {
    display: grid;
    grid-template-columns: 1.3fr 1fr;
    gap: var(--space-6);
    align-items: start;
  }

  .dash-col {
    display: grid;
    gap: var(--space-6);
  }

  .card {
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-card);
    background: var(--bg-surface);
    padding: var(--space-5);
  }

  /* Jump back in */
  .jump-card {
    display: block;
    text-decoration: none;
    color: inherit;
  }

  .jump-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
  }

  .jump-info {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
  }

  .jump-title {
    font-family: var(--font-serif);
    font-size: var(--fs-lead);
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .jump-meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
  }

  .jump-lang {
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  .jump-pts {
    color: var(--accent-primary);
    font-variant-numeric: tabular-nums;
  }

  /* All done */
  .all-done {
    padding: var(--space-2);
    font-size: var(--fs-micro);
    color: var(--text-secondary);
  }

  /* Quick links */
  .quick-links {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-5);
  }

  .action-card {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    text-decoration: none;
    color: inherit;
    padding: var(--space-4);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    background: var(--bg-surface);
    transition: border-color 0.15s ease;
  }

  .action-card:hover {
    border-color: var(--accent-primary);
  }

  .action-card h3 {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    color: var(--text-primary);
  }

  .action-card p {
    font-size: var(--fs-micro);
    color: var(--text-secondary);
  }

  /* Responsive */
  @media (max-width: 980px) {
    .dash-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 600px) {
    .ledger {
      grid-template-columns: 1fr 1fr;
    }

    .ledger-cell:nth-child(2) {
      border-right: 0;
    }

    .ledger-cell:nth-child(1),
    .ledger-cell:nth-child(2) {
      border-bottom: 1px solid var(--border-primary);
    }

    .jump-content {
      flex-direction: column;
      align-items: flex-start;
    }

    .quick-links {
      grid-template-columns: 1fr;
    }
  }
</style>
