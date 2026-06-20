<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'default' | 'elevated' | 'bordered';
    padding?: 'sm' | 'md' | 'lg';
    children: Snippet;
  }

  let {
    variant = 'default',
    padding = 'md',
    children,
  }: Props = $props();
</script>

<div class="card card-{variant} pad-{padding}">
  {@render children()}
</div>

<style>
  .card {
    border-radius: var(--radius-card);
    transition: border-color var(--transition-base);
  }

  .card-default {
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
  }

  /* Canonical elevation/hover defined once, here — a quiet border deepen,
     no translateY lift. */
  .card-default:hover,
  .card-bordered:hover {
    border-color: var(--border-secondary);
  }

  .card-elevated {
    background: var(--bg-elevated);
    border: 1px solid var(--border-primary);
    box-shadow: var(--shadow-md);
  }

  .card-bordered {
    background: transparent;
    border: 1px solid var(--border-secondary);
  }

  .pad-sm {
    padding: var(--space-4);
  }

  .pad-md {
    padding: var(--space-6);
  }

  .pad-lg {
    padding: var(--space-8);
  }
</style>
