<script lang="ts">
  interface Props {
    level: number;
    size?: 'sm' | 'md';
  }

  let { level, size = 'md' }: Props = $props();

  const labels: Record<number, string> = {
    1: 'Trivial', 2: 'Easy', 3: 'Moderate', 4: 'Moderate',
    5: 'Challenging', 6: 'Hard', 7: 'Hard',
    8: 'Expert', 9: 'Elite', 10: 'Legendary',
  };

  function getDiffColor(d: number): string {
    if (d <= 2) return 'var(--diff-1)';
    if (d <= 4) return 'var(--diff-3)';
    if (d <= 6) return 'var(--diff-5)';
    if (d <= 8) return 'var(--diff-7)';
    return 'var(--diff-10)';
  }
</script>

<span
  class="badge badge-{size}"
  style="--badge-color: {getDiffColor(level)}"
>
  Level {level} &middot; {labels[level] || 'Unknown'}
</span>

<style>
  .badge {
    font-family: var(--font-sans);
    display: inline-flex;
    align-items: center;
    border: 1px solid var(--badge-color);
    color: var(--badge-color);
    background: color-mix(in srgb, var(--badge-color) 10%, transparent);
    border-radius: var(--radius-md);
    letter-spacing: 0.02em;
    font-weight: 500;
    white-space: nowrap;
  }

  .badge-sm {
    font-size: 0.625rem;
    padding: 1px 6px;
  }

  .badge-md {
    font-size: 0.6875rem;
    padding: 2px 8px;
  }
</style>
