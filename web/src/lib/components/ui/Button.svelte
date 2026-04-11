<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    size?: 'sm' | 'md' | 'lg';
    disabled?: boolean;
    loading?: boolean;
    type?: 'button' | 'submit';
    onclick?: () => void;
    children: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    disabled = false,
    loading = false,
    type = 'button',
    onclick,
    children,
  }: Props = $props();
</script>

<button
  class="btn btn-{variant} btn-{size}"
  {type}
  disabled={disabled || loading}
  {onclick}
>
  {#if loading}
    <span class="spinner"></span>
  {/if}
  {@render children()}
</button>

<style>
  .btn {
    font-family: var(--font-sans);
    font-weight: 500;
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    transition: all var(--transition-fast);
    position: relative;
    overflow: hidden;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Sizes */
  .btn-sm {
    padding: var(--space-1) var(--space-3);
    font-size: 0.75rem;
  }

  .btn-md {
    padding: var(--space-2) var(--space-5);
    font-size: 0.8125rem;
  }

  .btn-lg {
    padding: var(--space-3) var(--space-8);
    font-size: 0.875rem;
  }

  /* Variants */
  .btn-primary {
    background: var(--accent-green);
    color: var(--text-inverse);
    border-color: var(--accent-green);
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-green-dim);
    box-shadow: var(--shadow-glow-green);
  }

  .btn-secondary {
    background: transparent;
    color: var(--accent-green);
    border-color: var(--accent-green);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--accent-green-glow);
  }

  .btn-ghost {
    background: transparent;
    color: var(--text-secondary);
    border-color: var(--border-primary);
  }

  .btn-ghost:hover:not(:disabled) {
    color: var(--text-primary);
    border-color: var(--border-secondary);
    background: var(--bg-hover);
  }

  .btn-danger {
    background: var(--accent-red);
    color: #fff;
    border-color: var(--accent-red);
  }

  .btn-danger:hover:not(:disabled) {
    background: var(--accent-red-dim);
    box-shadow: 0 2px 8px rgba(201, 114, 107, 0.15);
  }

  /* Spinner */
  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
