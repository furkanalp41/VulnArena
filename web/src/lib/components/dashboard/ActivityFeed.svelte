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
      case 'challenge_solved': return 'var(--accent-green)';
      case 'challenge_attempted': return 'var(--difficulty-5)';
      case 'lesson_completed': return 'var(--accent-blue)';
      default: return 'var(--text-tertiary)';
    }
  }
</script>

<div class="activity-feed">
  {#if activities.length === 0}
    <div class="feed-empty">
      <span class="feed-empty-icon">$</span>
      <span>No activity yet. Start solving challenges.</span>
    </div>
  {:else}
    <ul class="feed-list">
      {#each activities as entry}
        <li class="feed-item">
          <span class="feed-icon" style:color={typeColor(entry.type)}>{typeIcon(entry.type)}</span>
          <div class="feed-content">
            <span class="feed-title">{entry.title}</span>
            <div class="feed-meta">
              {#if entry.type !== 'lesson_completed'}
                <span class="feed-score" style:color={typeColor(entry.type)}>
                  {entry.score}%
                </span>
                <span class="feed-sep">·</span>
              {/if}
              <span class="feed-points">+{entry.points} XP</span>
              <span class="feed-sep">·</span>
              <span class="feed-time">{timeAgo(entry.occurred_at)}</span>
            </div>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .activity-feed {
    width: 100%;
  }

  .feed-empty {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1.5rem;
    font-size: 0.8rem;
    color: var(--text-tertiary);
  }

  .feed-empty-icon {
    color: var(--accent-green);
  }

  .feed-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
  }

  .feed-item {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 0.65rem 0.75rem;
    border-bottom: 1px solid var(--border-primary);
    transition: background 0.15s ease;
  }

  .feed-item:last-child {
    border-bottom: none;
  }

  .feed-item:hover {
    background: var(--bg-hover);
  }

  .feed-icon {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    font-weight: 700;
    flex-shrink: 0;
    padding-top: 0.1rem;
  }

  .feed-content {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    min-width: 0;
  }

  .feed-title {
    font-size: 0.8rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .feed-meta {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.7rem;
    color: var(--text-tertiary);
  }

  .feed-score {
    font-weight: 600;
  }

  .feed-points {
    color: var(--text-tertiary);
  }

  .feed-sep {
    opacity: 0.4;
  }

  .feed-time {
    opacity: 0.6;
  }
</style>
