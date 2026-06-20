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

  function monogram(name: string): string {
    const words = name.trim().split(/\s+/).filter(Boolean);
    if (words.length === 0) return '··';
    if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
    return (words[0][0] + words[1][0]).toUpperCase();
  }
</script>

<div class="section-header"><h3>Achievements</h3><span class="smallcaps">earned · {safeUnlocked.length}</span></div>

<div class="ach-row">
  {#each safeAll as achievement}
    {@const active = isUnlocked(achievement.slug)}
    {@const date = getUnlockDate(achievement.slug)}
    {@const glow = glowColor(achievement.category)}
    <div class="ach" class:unlocked={active} class:locked={!active} style:--glow-color={glow}>
      <div class="mark">{monogram(achievement.name)}</div>
      <span class="alab">{achievement.name}</span>

      <!-- Tooltip -->
      <div class="tip">
        <b>{achievement.name}</b>
        {achievement.description}
        <span class="tip-meta">
          <span class="smallcaps">{achievement.category}</span>
          <span class="sep">·</span>
          {#if active && date}
            <span class="tnum">{date}</span>
          {:else}
            <span class="smallcaps">locked</span>
          {/if}
          <span class="sep">·</span>
          <span class="tnum">+{achievement.xp_reward} XP</span>
        </span>
      </div>
    </div>
  {/each}
</div>

<style>
  .ach-row {
    display: flex;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .ach {
    position: relative;
    flex: 0 0 auto;
  }

  .ach .mark {
    display: grid;
    place-items: center;
    width: 52px;
    height: 52px;
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-input);
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    letter-spacing: 0.05em;
    color: var(--accent-primary);
    background: var(--bg-surface);
    transition:
      border-color 0.15s ease,
      transform 0.15s ease;
  }

  .ach:hover .mark {
    border-color: var(--accent-primary);
    transform: translateY(1px);
  }

  .ach.locked .mark {
    color: var(--text-tertiary);
    border-style: dashed;
  }

  .ach.locked:hover .mark {
    border-color: var(--border-secondary);
    transform: none;
  }

  .ach .alab {
    display: block;
    text-align: center;
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
    letter-spacing: 0.04em;
    margin-top: var(--space-2);
    max-width: 64px;
  }

  /* Tooltip — opens BELOW the mark so the top row isn't clipped */
  .ach .tip {
    position: absolute;
    top: 62px;
    left: 50%;
    transform: translateX(-50%) translateY(-4px);
    width: 180px;
    padding: var(--space-2) var(--space-3);
    background: var(--bg-elevated);
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-input);
    box-shadow: var(--shadow-lg);
    font-size: var(--fs-micro);
    line-height: 1.45;
    color: var(--text-secondary);
    opacity: 0;
    visibility: hidden;
    pointer-events: none;
    z-index: 20;
    transition:
      opacity 0.15s ease,
      transform 0.15s ease;
  }

  .ach .tip b {
    display: block;
    margin-bottom: 2px;
    font-family: var(--font-serif);
    font-weight: 700;
    color: var(--text-primary);
  }

  .ach .tip-meta {
    display: block;
    margin-top: var(--space-2);
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
    font-variant-numeric: tabular-nums;
  }

  .ach .tip-meta .sep {
    color: var(--border-secondary);
    margin: 0 0.35em;
  }

  .ach:hover .tip {
    opacity: 1;
    visibility: visible;
    transform: translateX(-50%) translateY(0);
  }
</style>
