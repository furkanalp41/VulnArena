<script lang="ts">
  import type { ActivityEntry } from '$lib/api/dashboard';

  interface Props {
    activities: ActivityEntry[];
  }

  let { activities }: Props = $props();

  function timeAgo(dateStr: string): string {
    const now = Date.now();
    const then = new Date(dateStr).getTime();
    const diff = Math.floor((now - then) / 1000);

    if (diff < 60) return 'just now';
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
    return new Date(dateStr).toLocaleDateString();
  }

  function typeIcon(type: ActivityEntry['type']): string {
    switch (type) {
      case 'challenge_solved': return '[+]';
      case 'challenge_attempted': return '[~]';
      case 'lesson_completed': return '[>]';
      default: return '[*]';
    }
  }

  function typeColor(type: ActivityEntry['type']): string {
    switch (type) {
      case 'challenge_solved': return 'var(--accent-primary)';
      case 'challenge_attempted': return 'var(--diff-5)';
      case 'lesson_completed': return 'var(--accent-blue)';
      default: return 'var(--text-tertiary)';
    }
  }
</script>

<div class="activity-feed">
  {#if activities.length === 0}
    <div class="feed-empty">
      <div class="rule"></div>
      <p class="feed-empty-line">
        <span class="prompt">$</span> log --tail
      </p>
      <p class="feed-empty-msg">No activity yet. Start solving challenges.</p>
    </div>
  {:else}
    <ul class="feed">
      {#each activities as entry}
        <li>
          <span class="mark" style:color={typeColor(entry.type)}>{typeIcon(entry.type)}</span>
          <span class="subj">
            <b>{entry.title}</b>
            <span class="meta">
              {#if entry.type !== 'lesson_completed'}
                <span class="score tnum" style:color={typeColor(entry.type)}>{entry.score}%</span>
                <span class="sep">·</span>
              {/if}
              <span class="points tnum">+{entry.points} XP</span>
            </span>
          </span>
          <span class="ts tnum">{timeAgo(entry.occurred_at)}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .activity-feed {
    width: 100%;
  }

  .feed {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .feed li {
    display: grid;
    grid-template-columns: 1.4rem 1fr auto;
    gap: var(--space-3);
    align-items: baseline;
    padding: var(--space-3) 0;
    border-bottom: 1px solid var(--border-primary);
    font-size: var(--fs-micro);
  }

  .feed li:last-child {
    border-bottom: 0;
  }

  .feed .mark {
    font-family: var(--font-mono);
    color: var(--text-tertiary);
  }

  .subj {
    color: var(--text-primary);
    min-width: 0;
  }

  .subj b {
    font-family: var(--font-serif);
    font-weight: 600;
  }

  .meta {
    display: inline;
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
    white-space: nowrap;
  }

  .meta .score {
    font-weight: 600;
  }

  .meta .points {
    color: var(--text-tertiary);
  }

  .meta .sep {
    opacity: 0.4;
    margin: 0 0.15em;
  }

  .ts {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
    text-align: right;
    white-space: nowrap;
    min-width: 5.5rem;
  }

  .feed-empty {
    padding: var(--space-3) 0;
  }

  .feed-empty .rule {
    height: 1px;
    background: var(--border-primary);
    border: 0;
    margin: 0 0 var(--space-3);
  }

  .feed-empty-line {
    margin: 0;
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    color: var(--text-secondary);
  }

  .feed-empty-line .prompt {
    color: var(--accent-primary);
  }

  .feed-empty-msg {
    margin: var(--space-2) 0 0;
    font-size: var(--fs-micro);
    color: var(--text-tertiary);
  }
</style>
