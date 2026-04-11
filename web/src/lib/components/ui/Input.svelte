<script lang="ts">
  interface Props {
    type?: string;
    placeholder?: string;
    value?: string;
    label?: string;
    error?: string;
    disabled?: boolean;
    name?: string;
    required?: boolean;
    oninput?: (e: Event) => void;
  }

  let {
    type = 'text',
    placeholder = '',
    value = $bindable(''),
    label = '',
    error = '',
    disabled = false,
    name = '',
    required = false,
    oninput,
  }: Props = $props();
</script>

<div class="input-group" class:has-error={!!error}>
  {#if label}
    <label class="input-label" for={name}>
      {label}
      {#if required}<span class="required">*</span>{/if}
    </label>
  {/if}
  <input
    class="input-field"
    {type}
    {placeholder}
    bind:value
    {disabled}
    {name}
    id={name}
    {required}
    {oninput}
  />
  {#if error}
    <span class="input-error">{error}</span>
  {/if}
</div>

<style>
  .input-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .input-label {
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .required {
    color: var(--accent-red);
  }

  .input-field {
    font-family: var(--font-sans);
    font-size: 0.875rem;
    padding: var(--space-3) var(--space-4);
    background: var(--bg-input);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    outline: none;
    transition: all var(--transition-fast);
    width: 100%;
  }

  .input-field::placeholder {
    color: var(--text-tertiary);
  }

  .input-field:focus {
    border-color: var(--accent-green);
    box-shadow: 0 0 0 2px var(--accent-green-glow);
  }

  .input-field:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .has-error .input-field {
    border-color: var(--accent-red);
  }

  .has-error .input-field:focus {
    box-shadow: 0 0 0 2px var(--accent-red-glow);
  }

  .input-error {
    font-size: 0.75rem;
    color: var(--accent-red);
    font-family: var(--font-sans);
  }
</style>
