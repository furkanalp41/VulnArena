<script lang="ts">
  import type { RankInfo } from '$lib/api/dashboard';
  import { getTierColor } from '$lib/utils/tiers';

  interface Props {
    rank: RankInfo;
  }

  let { rank }: Props = $props();

  const radius = 52;
  const stroke = 6;
  const circumference = 2 * Math.PI * radius;
  const dashOffset = $derived(circumference - (rank.progress / 100) * circumference);

  const glowColor = $derived(getTierColor(rank.title));
</script>

<div class="rank-card">
  <div class="ring-container">
    <svg width="130" height="130" viewBox="0 0 130 130" class="progress-ring">
      <!-- Background ring -->
      <circle
        cx="65"
        cy="65"
        r={radius}
        fill="none"
        stroke="var(--border-primary)"
        stroke-width={stroke}
        opacity="0.3"
      />
      <!-- Progress ring -->
      <circle
        cx="65"
        cy="65"
        r={radius}
        fill="none"
        stroke={glowColor}
        stroke-width={stroke}
        stroke-linecap="round"
        stroke-dasharray={circumference}
        stroke-dashoffset={dashOffset}
        class="progress-arc"
        style:--glow-color={glowColor}
        transform="rotate(-90 65 65)"
      />
    </svg>
    <div class="ring-label">
      <span class="tier-number">T{rank.tier}</span>
    </div>
  </div>

  <div class="rank-info">
    <span class="rank-title" style:color={glowColor}>{rank.title}</span>
    <div class="xp-row">
      <span class="xp-current">{rank.total_xp.toLocaleString()} XP</span>
      {#if rank.next_tier_xp > 0}
        <span class="xp-divider">/</span>
        <span class="xp-next">{rank.next_tier_xp.toLocaleString()}</span>
      {/if}
    </div>
    <div class="progress-bar">
      <div class="progress-fill" style:width="{rank.progress}%" style:background={glowColor}></div>
    </div>
  </div>
</div>

<style>
  .rank-card {
    display: flex;
    align-items: center;
    gap: 1.5rem;
  }

  .ring-container {
    position: relative;
    flex-shrink: 0;
  }

  .progress-ring {
    display: block;
    filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.2));
  }

  .progress-arc {
    transition: stroke-dashoffset 0.8s ease;
    filter: none;
  }

  .ring-label {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .tier-number {
    font-family: var(--font-sans);
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: 0.05em;
  }

  .rank-info {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .rank-title {
    font-family: var(--font-serif);
    font-size: 0.9rem;
    font-weight: 600;
    letter-spacing: 0.02em;
  }

  .xp-row {
    display: flex;
    align-items: baseline;
    gap: 0.25rem;
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }

  .xp-current {
    color: var(--text-secondary);
  }

  .xp-divider {
    color: var(--text-tertiary);
  }

  .xp-next {
    color: var(--text-tertiary);
    font-size: 0.7rem;
  }

  .progress-bar {
    width: 100%;
    height: 3px;
    background: var(--border-primary);
    border-radius: 2px;
    overflow: hidden;
    margin-top: 0.15rem;
  }

  .progress-fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.8s ease;
  }
</style>
