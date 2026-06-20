<script lang="ts">
  import type { RankInfo } from '$lib/api/dashboard';
  import { getTierColor } from '$lib/utils/tiers';

  interface Props {
    rank: RankInfo;
  }

  let { rank }: Props = $props();

  // Gauge geometry: a 270° arc (open at the bottom), driven by real props.
  const radius = 70;
  const stroke = 6;
  const circumference = 2 * Math.PI * radius; // ≈ 439.82
  const arcSweep = circumference * 0.75; // 270° visible track ≈ 329.87
  const trackGap = circumference - arcSweep; // bottom gap ≈ 109.96

  const clampedProgress = $derived(Math.max(0, Math.min(100, rank.progress)));
  const progressLen = $derived((clampedProgress / 100) * arcSweep);

  const glowColor = $derived(getTierColor(rank.title));
  const xpToNext = $derived(Math.max(0, rank.next_tier_xp - rank.total_xp));
</script>

<div class="gauge-wrap">
  <div class="gauge">
    <svg viewBox="0 0 180 180" width="160" height="160" aria-hidden="true">
      <!-- Track -->
      <circle
        cx="90"
        cy="90"
        r={radius}
        fill="none"
        stroke="var(--border-secondary)"
        stroke-width={stroke}
        stroke-dasharray="{arcSweep} {trackGap}"
        stroke-linecap="butt"
        transform="rotate(135 90 90)"
      />
      <!-- Progress -->
      <circle
        cx="90"
        cy="90"
        r={radius}
        fill="none"
        stroke="var(--accent-primary)"
        stroke-width={stroke}
        stroke-dasharray="{progressLen} {circumference}"
        stroke-linecap="butt"
        transform="rotate(135 90 90)"
        class="progress-arc"
      />
      <!-- Engraved start / end ticks -->
      <line x1="47.6" y1="132.4" x2="40.5" y2="139.5" stroke="var(--text-tertiary)" stroke-width="1.5" />
      <line x1="132.4" y1="132.4" x2="139.5" y2="139.5" stroke="var(--text-tertiary)" stroke-width="1.5" />
    </svg>
    <div class="gauge-center">
      <div>
        <div class="rk">{rank.title}</div>
        <div class="tier">TIER {rank.tier}</div>
      </div>
    </div>
  </div>
  <div class="gauge-side">
    <div class="lab">Next tier</div>
    <div class="big tnum">{clampedProgress}%</div>
    {#if rank.next_tier_xp > 0}
      <div class="sub">{xpToNext.toLocaleString()} XP to T{rank.tier + 1}</div>
    {:else}
      <div class="sub">{rank.total_xp.toLocaleString()} XP · max tier</div>
    {/if}
  </div>
</div>

<style>
  .gauge-wrap {
    display: flex;
    align-items: center;
    gap: var(--space-5);
  }

  .gauge {
    position: relative;
    width: 160px;
    height: 160px;
    flex: 0 0 160px;
  }

  .gauge svg {
    display: block;
  }

  .progress-arc {
    transition: stroke-dasharray 0.8s ease;
  }

  .gauge-center {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    text-align: center;
  }

  .gauge-center .rk {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    line-height: 1.05;
    color: var(--text-primary);
    letter-spacing: -0.01em;
    max-width: 6.5rem;
  }

  .gauge-center .tier {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.14em;
    color: var(--accent-primary);
    margin-top: 4px;
  }

  .gauge-side .lab {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    text-transform: uppercase;
    letter-spacing: 0.13em;
    color: var(--text-tertiary);
  }

  .gauge-side .big {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    color: var(--text-primary);
    margin-top: var(--space-1);
  }

  .gauge-side .sub {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    color: var(--text-secondary);
    margin-top: var(--space-1);
  }
</style>
