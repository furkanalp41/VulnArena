<script lang="ts">
  import { toasts, removeToast, type Toast } from '$lib/stores/toast';

  function toastMessage(t: Toast): string {
    switch (t.type) {
      case 'FIRST_BLOOD':
        return `FIRST BLOOD! ${t.user} pwned "${t.challenge}"`;
      case 'ACHIEVEMENT':
        return `${t.user} unlocked "${t.achievement}"`;
      default:
        return `${t.type}: ${t.user}`;
    }
  }

  function toastBorderColor(type: string): string {
    switch (type) {
      case 'FIRST_BLOOD':
        return '#c9726b';
      case 'ACHIEVEMENT':
        return '#d4a574';
      default:
        return 'var(--accent-green)';
    }
  }

  function toastLabel(type: string): string {
    switch (type) {
      case 'FIRST_BLOOD':
        return 'FIRST BLOOD';
      case 'ACHIEVEMENT':
        return 'ACHIEVEMENT';
      default:
        return 'EVENT';
    }
  }
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as toast (toast.id)}
      <button
        class="toast-card"
        style:--toast-color={toastBorderColor(toast.type)}
        onclick={() => removeToast(toast.id)}
      >
        <span class="toast-label">{toastLabel(toast.type)}</span>
        <span class="toast-msg">{toastMessage(toast)}</span>
      </button>
    {/each}
  </div>
{/if}

<style>
  .toast-container {
    position: fixed;
    top: 68px;
    right: 16px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 380px;
    pointer-events: none;
  }

  .toast-card {
    pointer-events: all;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 10px 14px;
    background: var(--bg-primary);
    border: 1px solid var(--toast-color);
    border-left: 3px solid var(--toast-color);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    cursor: pointer;
    animation: toast-slide-in 0.3s ease-out;
    text-align: left;
    width: 100%;
    font: inherit;
    color: inherit;
  }

  .toast-label {
    font-size: 0.6rem;
    letter-spacing: 0.15em;
    color: var(--toast-color);
    font-weight: 700;
  }

  .toast-msg {
    font-size: 0.75rem;
    color: var(--text-primary);
    line-height: 1.4;
  }

  @keyframes toast-slide-in {
    from {
      transform: translateX(100%);
      opacity: 0;
    }
    to {
      transform: translateX(0);
      opacity: 1;
    }
  }
</style>
