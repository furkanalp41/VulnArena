<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    lines: string[];
    title?: string;
    animate?: boolean;
  }

  let { lines = [], title = 'ANALYSIS OUTPUT', animate = true }: Props = $props();

  let visibleCount = $state(0);
  let terminalEl: HTMLDivElement;

  $effect(() => {
    if (!animate) {
      visibleCount = lines.length;
      return;
    }

    // Reset and animate when new lines arrive
    visibleCount = 0;
    let i = 0;
    const interval = setInterval(() => {
      if (i < lines.length) {
        visibleCount = i + 1;
        i++;
        // Auto-scroll to bottom
        if (terminalEl) {
          terminalEl.scrollTop = terminalEl.scrollHeight;
        }
      } else {
        clearInterval(interval);
      }
    }, 60);

    return () => clearInterval(interval);
  });
</script>

<div class="terminal">
  <div class="terminal-header">
    <div class="terminal-dots">
      <span class="dot dot-red"></span>
      <span class="dot dot-yellow"></span>
      <span class="dot dot-green"></span>
    </div>
    <span class="terminal-title font-mono">{title}</span>
    <div class="terminal-spacer"></div>
  </div>
  <div class="terminal-body" bind:this={terminalEl}>
    {#each lines.slice(0, visibleCount) as line, i}
      <div class="terminal-line" class:highlight={line.includes('[+]') || line.includes('PASSED')} class:error={line.includes('INSUFFICIENT') || line.includes('HINT:')}>
        <span class="line-content font-mono">{line}</span>
      </div>
    {/each}
    {#if visibleCount < lines.length}
      <div class="cursor-line font-mono">
        <span class="cursor">_</span>
      </div>
    {/if}
  </div>
</div>

<style>
  .terminal {
    background: var(--bg-primary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .terminal-header {
    display: flex;
    align-items: center;
    padding: var(--space-2) var(--space-3);
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    gap: var(--space-3);
  }

  .terminal-dots {
    display: flex;
    gap: 6px;
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }

  .dot-red { background: #ff5f57; }
  .dot-yellow { background: #febc2e; }
  .dot-green { background: #28c840; }

  .terminal-title {
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    letter-spacing: 0.08em;
  }

  .terminal-spacer {
    flex: 1;
  }

  .terminal-body {
    padding: var(--space-3);
    overflow-y: auto;
    max-height: 300px;
    flex: 1;
  }

  .terminal-line {
    padding: 1px 0;
    line-height: 1.5;
  }

  .line-content {
    font-size: 0.75rem;
    color: var(--text-secondary);
    white-space: pre-wrap;
    word-break: break-word;
  }

  .highlight .line-content {
    color: var(--accent-green);
  }

  .error .line-content {
    color: var(--accent-yellow);
  }

  .cursor-line {
    padding: 1px 0;
  }

  .cursor {
    font-size: 0.75rem;
    color: var(--accent-green);
    animation: blink 1s step-end infinite;
  }

  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0; }
  }
</style>
