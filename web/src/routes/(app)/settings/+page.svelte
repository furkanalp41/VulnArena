<script lang="ts">
  import { onMount } from 'svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Terminal from '$lib/components/ui/Terminal.svelte';
  import { generateApiKey, revokeApiKey, getApiKeyInfo } from '$lib/api/settings';
  import { addToast } from '$lib/stores/toast';
  import type { ApiKeyInfo } from '$lib/api/settings';

  let keyInfo: ApiKeyInfo | null = $state(null);
  let newKey: string | null = $state(null);
  let loading = $state(true);
  let generating = $state(false);
  let revoking = $state(false);
  let showRevokeConfirm = $state(false);
  let showRegenerateConfirm = $state(false);
  let copied = $state(false);

  onMount(async () => {
    await loadKeyInfo();
  });

  async function loadKeyInfo() {
    loading = true;
    try {
      keyInfo = await getApiKeyInfo();
    } catch {
      keyInfo = null;
    } finally {
      loading = false;
    }
  }

  async function handleGenerate() {
    generating = true;
    showRegenerateConfirm = false;
    try {
      const res = await generateApiKey();
      newKey = res.api_key;
      await loadKeyInfo();
      addToast({ type: 'success', user: 'API key generated successfully' });
    } catch {
      addToast({ type: 'error', user: 'Failed to generate API key' });
    } finally {
      generating = false;
    }
  }

  async function handleRevoke() {
    revoking = true;
    showRevokeConfirm = false;
    try {
      await revokeApiKey();
      keyInfo = null;
      newKey = null;
      addToast({ type: 'success', user: 'API key revoked' });
    } catch {
      addToast({ type: 'error', user: 'Failed to revoke API key' });
    } finally {
      revoking = false;
    }
  }

  async function copyKey() {
    if (!newKey) return;
    await navigator.clipboard.writeText(newKey);
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }
</script>

<div class="settings-page">
  <div class="page-header">
    <h1 class="page-title">Settings</h1>
    <p class="page-subtitle">Developer & API Configuration</p>
  </div>

  <section class="section">
    <div class="section-header">
      <h2 class="section-title">API Key</h2>
      <span class="section-badge">Developer</span>
    </div>

    <Card variant="bordered">
      {#if loading}
        <div class="loading-state">
          <span class="loading-text">Loading API key status...</span>
        </div>
      {:else if keyInfo}
        <div class="key-status">
          <div class="status-indicator active">
            <span class="status-dot"></span>
            <span>Active</span>
          </div>
          <div class="key-meta">
            <div class="meta-item">
              <span class="meta-label">Key ending in</span>
              <span class="meta-value font-mono">...{keyInfo.hint}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Created</span>
              <span class="meta-value">{formatDate(keyInfo.created_at)}</span>
            </div>
          </div>
          <div class="key-actions">
            {#if showRegenerateConfirm}
              <div class="confirm-inline">
                <span class="confirm-text">This will invalidate your current key. Continue?</span>
                <div class="confirm-buttons">
                  <Button variant="danger" size="sm" onclick={handleGenerate} loading={generating}>Confirm</Button>
                  <Button variant="ghost" size="sm" onclick={() => (showRegenerateConfirm = false)}>Cancel</Button>
                </div>
              </div>
            {:else if showRevokeConfirm}
              <div class="confirm-inline">
                <span class="confirm-text">Revoke this key? CLI access will stop working.</span>
                <div class="confirm-buttons">
                  <Button variant="danger" size="sm" onclick={handleRevoke} loading={revoking}>Revoke</Button>
                  <Button variant="ghost" size="sm" onclick={() => (showRevokeConfirm = false)}>Cancel</Button>
                </div>
              </div>
            {:else}
              <Button variant="secondary" size="sm" onclick={() => (showRegenerateConfirm = true)}>Regenerate</Button>
              <Button variant="ghost" size="sm" onclick={() => (showRevokeConfirm = true)}>Revoke</Button>
            {/if}
          </div>
        </div>
      {:else}
        <div class="key-empty">
          <p class="empty-text">No API key generated</p>
          <p class="empty-hint">Generate a key to use VulnArena from your terminal via the CLI tool.</p>
          <Button variant="primary" onclick={handleGenerate} loading={generating}>Generate API Key</Button>
        </div>
      {/if}
    </Card>

    {#if newKey}
      <div class="new-key-banner">
        <div class="banner-header">
          <span class="banner-icon">!</span>
          <span class="banner-title">API key generated</span>
        </div>
        <p class="banner-warning">Copy this key now. It will not be shown again.</p>
        <div class="key-display">
          <code class="key-value font-mono">{newKey}</code>
          <button class="copy-btn" onclick={copyKey}>
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
      </div>
    {/if}
  </section>

  <section class="section">
    <div class="section-header">
      <h2 class="section-title">CLI usage</h2>
      <span class="section-badge">Terminal</span>
    </div>

    <Terminal
      title="VULNARENA CLI"
      animate={false}
      lines={[
        '# Authenticate with your API key',
        '$ vulnarena auth login <your-api-key>',
        '',
        '# List available challenges',
        '$ vulnarena arena list',
        '',
        '# Submit a solution',
        '$ vulnarena arena submit <challenge-id> -m "explanation"',
        '',
        '# Build from source',
        '$ go build -o vulnarena ./cmd/cli',
      ]}
    />
  </section>
</div>

<style>
  .settings-page {
    max-width: 800px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-8);
  }

  .page-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .page-title {
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .page-subtitle {
    font-size: 0.875rem;
    color: var(--text-tertiary);
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .section-title {
    font-family: var(--font-serif);
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .section-badge {
    font-family: var(--font-sans);
    font-size: 0.625rem;
    color: var(--text-tertiary);
    border: 1px solid var(--border-primary);
    padding: 2px var(--space-2);
    border-radius: var(--radius-sm);
  }

  .loading-state {
    padding: var(--space-4);
    text-align: center;
  }

  .loading-text {
    color: var(--text-tertiary);
    font-size: 0.8125rem;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .key-status {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-family: var(--font-sans);
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .status-indicator.active {
    color: var(--accent-green);
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--accent-green);
    box-shadow: 0 0 8px var(--accent-green);
  }

  .key-meta {
    display: flex;
    gap: var(--space-6);
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .meta-label {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
  }

  .meta-value {
    font-size: 0.875rem;
    color: var(--text-primary);
  }

  .key-actions {
    display: flex;
    gap: var(--space-2);
    align-items: flex-start;
  }

  .confirm-inline {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .confirm-text {
    font-size: 0.8125rem;
    color: var(--accent-orange);
  }

  .confirm-buttons {
    display: flex;
    gap: var(--space-2);
  }

  .key-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4) 0;
    text-align: center;
  }

  .empty-text {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .empty-hint {
    font-size: 0.8125rem;
    color: var(--text-tertiary);
    max-width: 400px;
  }

  .new-key-banner {
    background: rgba(0, 255, 136, 0.04);
    border: 1px solid var(--accent-green);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .banner-header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .banner-icon {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--accent-green);
    color: var(--text-inverse);
    font-size: 0.75rem;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .banner-title {
    font-family: var(--font-sans);
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--accent-green);
  }

  .banner-warning {
    font-family: var(--font-sans);
    font-size: 0.75rem;
    color: var(--accent-orange);
  }

  .key-display {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: var(--bg-primary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    overflow-x: auto;
  }

  .key-value {
    flex: 1;
    font-size: 0.8125rem;
    color: var(--accent-green);
    word-break: break-all;
    user-select: all;
  }

  .copy-btn {
    flex-shrink: 0;
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    padding: var(--space-1) var(--space-3);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .copy-btn:hover {
    border-color: var(--accent-green);
    color: var(--accent-green);
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
</style>
