<script lang="ts">
  import type { Achievement, UserAchievement } from '$lib/api/achievements';

  interface Props {
    unlocked: UserAchievement[];
    all: Achievement[];
  }

  let { unlocked, all }: Props = $props();

  const safeUnlocked = $derived(unlocked ?? []);
  const safeAll = $derived(all ?? []);
  const unlockedSlugs = $derived(new Set(safeUnlocked.map((ua) => ua.achievement.slug)));

  function isUnlocked(slug: string): boolean {
    return unlockedSlugs.has(slug);
  }

  function getUnlockDate(slug: string): string | null {
    const ua = safeUnlocked.find((u) => u.achievement.slug === slug);
    if (!ua) return null;
    return new Date(ua.unlocked_at).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  }

  function glowColor(category: string): string {
    switch (category) {
      case 'special':
        return '#d4a574';
      case 'mastery':
        return '#8b9dc3';
      case 'combat':
        return '#7c9f6b';
      case 'dedication':
        return '#a78bba';
      default:
        return '#d4a574';
    }
  }
</script>

<div class="showcase-grid">
  {#each safeAll as achievement}
    {@const active = isUnlocked(achievement.slug)}
    {@const date = getUnlockDate(achievement.slug)}
    {@const glow = glowColor(achievement.category)}
    <div
      class="badge-card"
      class:unlocked={active}
      class:locked={!active}
      style:--glow-color={glow}
    >
      <div class="badge-icon" title={achievement.description}>
        {@html achievement.icon_svg}
      </div>
      <div class="badge-info">
        <span class="badge-name">{achievement.name}</span>
        {#if active && date}
          <span class="badge-date">{date}</span>
        {:else if !active}
          <span class="badge-locked">Locked</span>
        {/if}
      </div>
      <span class="badge-xp">+{achievement.xp_reward} XP</span>

      <!-- Tooltip -->
      <div class="badge-tooltip">
        <strong>{achievement.name}</strong>
        <p>{achievement.description}</p>
        <span class="tooltip-cat">{achievement.category.toUpperCase()}</span>
      </div>
    </div>
  {/each}
</div>

<style>
  .showcase-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: var(--space-4);
  }

  .badge-card {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-3);
    border-radius: var(--radius-md);
    background: var(--bg-secondary);
    border: 1px solid transparent;
    transition: all 0.25s ease;
    cursor: default;
  }

  .badge-card.unlocked {
    border-color: color-mix(in srgb, var(--glow-color) 30%, transparent);
  }

  .badge-card.unlocked:hover {
    border-color: color-mix(in srgb, var(--glow-color) 60%, transparent);
    transform: scale(1.04);
    box-shadow: var(--shadow-md);
  }

  .badge-card.locked {
    border-color: rgba(255, 255, 255, 0.05);
    opacity: 0.4;
  }

  .badge-card.locked:hover {
    opacity: 0.6;
  }

  .badge-icon {
    width: 72px;
    height: 72px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
  }

  .badge-card.unlocked .badge-icon {
    filter: drop-shadow(0 1px 3px rgba(0, 0, 0, 0.2));
  }

  .badge-card.locked .badge-icon {
    filter: grayscale(1) brightness(0.5);
  }

  .badge-icon :global(svg) {
    width: 100%;
    height: 100%;
  }

  .badge-info {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    text-align: center;
  }

  .badge-name {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--text-primary);
    letter-spacing: 0.03em;
  }

  .badge-date {
    font-size: 0.6rem;
    color: var(--text-tertiary);
  }

  .badge-locked {
    font-size: 0.6rem;
    color: var(--text-tertiary);
    letter-spacing: 0.1em;
  }

  .badge-xp {
    font-size: 0.6rem;
    font-weight: 600;
    letter-spacing: 0.06em;
  }

  .badge-card.unlocked .badge-xp {
    color: var(--glow-color);
  }

  .badge-card.locked .badge-xp {
    color: var(--text-tertiary);
  }

  /* Tooltip */
  .badge-tooltip {
    position: absolute;
    bottom: calc(100% + 8px);
    left: 50%;
    transform: translateX(-50%);
    background: var(--bg-primary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    min-width: 180px;
    max-width: 240px;
    opacity: 0;
    visibility: hidden;
    transition: all 0.15s ease;
    z-index: 10;
    pointer-events: none;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  }

  .badge-card:hover .badge-tooltip {
    opacity: 1;
    visibility: visible;
  }

  .badge-tooltip strong {
    font-size: 0.75rem;
    color: var(--text-primary);
    display: block;
    margin-bottom: 4px;
  }

  .badge-tooltip p {
    font-size: 0.7rem;
    color: var(--text-secondary);
    line-height: 1.4;
    margin: 0 0 6px;
  }

  .tooltip-cat {
    font-size: 0.6rem;
    letter-spacing: 0.1em;
    color: var(--text-tertiary);
  }

  @media (max-width: 600px) {
    .showcase-grid {
      grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
      gap: var(--space-3);
    }

    .badge-icon {
      width: 56px;
      height: 56px;
    }
  }
</style>
