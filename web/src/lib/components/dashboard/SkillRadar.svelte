<script lang="ts">
  import type { SkillRadarPoint } from '$lib/api/dashboard';

  interface Props {
    skills: SkillRadarPoint[];
    size?: number;
  }

  let { skills, size = 280 }: Props = $props();

  const center = $derived(size / 2);
  const maxRadius = $derived(center - 40);
  const levels = 5;
  const angleStep = $derived(skills.length > 0 ? (2 * Math.PI) / skills.length : 0);

  // Convert polar to cartesian (SVG: 0 is top, clockwise)
  function polarToXY(angle: number, radius: number): [number, number] {
    const x = center + radius * Math.sin(angle);
    const y = center - radius * Math.cos(angle);
    return [x, y];
  }

  // Build the polygon path for grid levels
  function gridPath(level: number): string {
    const r = (maxRadius / levels) * level;
    const points = skills.map((_, i) => polarToXY(i * angleStep, r));
    return points.map(([x, y]) => `${x},${y}`).join(' ');
  }

  // Build the data polygon
  const dataPath = $derived(() => {
    if (skills.length === 0) return '';
    const points = skills.map((s, i) => {
      const r = (s.score / 100) * maxRadius;
      return polarToXY(i * angleStep, r);
    });
    return points.map(([x, y]) => `${x},${y}`).join(' ');
  });

  // Axis endpoint
  function axisEnd(i: number): [number, number] {
    return polarToXY(i * angleStep, maxRadius);
  }

  // Label position (slightly outside the chart)
  function labelPos(i: number): { x: number; y: number; anchor: string } {
    const [x, y] = polarToXY(i * angleStep, maxRadius + 18);
    let anchor = 'middle';
    if (x < center - 10) anchor = 'end';
    if (x > center + 10) anchor = 'start';
    return { x, y: y + 4, anchor };
  }
</script>

{#if skills.length > 0}
  <div class="radar-wrap">
    <svg
      width={size}
      height={size}
      viewBox="0 0 {size} {size}"
      class="radar"
      aria-hidden="true"
    >
      <!-- Grid levels -->
      {#each Array(levels) as _, level}
        <polygon points={gridPath(level + 1)} class="grid-ring" />
      {/each}

      <!-- Axis lines -->
      {#each skills as _, i}
        {@const [ex, ey] = axisEnd(i)}
        <line x1={center} y1={center} x2={ex} y2={ey} class="axis-line" />
      {/each}

      <!-- Data polygon -->
      <polygon points={dataPath()} class="data-poly" />

      <!-- Data points -->
      {#each skills as skill, i}
        {@const r = (skill.score / 100) * maxRadius}
        {@const [px, py] = polarToXY(i * angleStep, r)}
        <circle cx={px} cy={py} r="2.5" class="data-point" />
      {/each}

      <!-- Labels -->
      {#each skills as skill, i}
        {@const lp = labelPos(i)}
        <text
          x={lp.x}
          y={lp.y}
          text-anchor={lp.anchor}
          class="axis-label"
        >
          {skill.category.length > 14 ? skill.category.slice(0, 12) + '…' : skill.category}
        </text>
      {/each}
    </svg>
  </div>
{:else}
  <div class="radar-empty">
    <p class="eyebrow">No data yet</p>
    <p class="radar-empty-sub">Solve challenges to see your skill profile</p>
  </div>
{/if}

<style>
  .radar-wrap {
    display: flex;
    justify-content: center;
    padding: var(--space-2) 0;
  }

  .radar {
    display: block;
  }

  .grid-ring {
    fill: none;
    stroke: var(--border-primary);
    stroke-width: 0.7;
  }

  .axis-line {
    stroke: var(--border-primary);
    stroke-width: 0.6;
  }

  .data-poly {
    fill: var(--accent-primary);
    fill-opacity: 0.14;
    stroke: var(--accent-primary);
    stroke-width: 1.5;
    stroke-linejoin: round;
  }

  .data-point {
    fill: var(--accent-primary);
    stroke: var(--bg-surface);
    stroke-width: 1;
  }

  .axis-label {
    font-family: var(--font-mono);
    font-size: 9px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    fill: var(--text-secondary);
  }

  .radar-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 200px;
    gap: var(--space-2);
    text-align: center;
  }

  .radar-empty-sub {
    font-family: var(--font-serif);
    font-size: var(--fs-micro);
    color: var(--text-tertiary);
  }
</style>
