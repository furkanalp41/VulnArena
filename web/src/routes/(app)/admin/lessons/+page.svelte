<script lang="ts">
  import { createLesson, type CreateLessonInput } from '$lib/api/admin';
  import Button from '$lib/components/ui/Button.svelte';
  import { marked } from 'marked';

  let form: CreateLessonInput = $state({
    title: '',
    category: '',
    description: '',
    content: '',
    difficulty: 3,
    read_time_min: 0,
    tags: [],
    is_published: true,
  });

  let tagInput = $state('');
  let submitting = $state(false);
  let success = $state('');
  let error = $state('');
  let showPreview = $state(false);

  const categories = [
    'Web Security',
    'Network Security',
    'Application Security',
    'Cryptography',
    'Reverse Engineering',
    'Forensics',
    'OWASP Top 10',
    'Secure Coding',
    'Cloud Security',
    'Mobile Security',
  ];

  function addTag() {
    const tag = tagInput.trim().toLowerCase();
    if (tag && !form.tags.includes(tag)) {
      form.tags = [...form.tags, tag];
      tagInput = '';
    }
  }

  function removeTag(index: number) {
    form.tags = form.tags.filter((_, i) => i !== index);
  }

  function getPreviewHtml(): string {
    try {
      return marked(form.content) as string;
    } catch {
      return '<p style="color:#ff4444;">[Markdown parse error]</p>';
    }
  }

  async function handleSubmit() {
    error = '';
    success = '';

    if (!form.title || !form.content || !form.category) {
      error = 'Title, category, and content are required.';
      return;
    }
    if (form.difficulty < 1 || form.difficulty > 10) {
      error = 'Difficulty must be between 1 and 10.';
      return;
    }

    submitting = true;
    try {
      await createLesson(form);
      success = `Report "${form.title}" published successfully.`;
      form = {
        title: '', category: '', description: '', content: '',
        difficulty: 3, read_time_min: 0, tags: [], is_published: true,
      };
      showPreview = false;
    } catch (e: any) {
      error = e?.message || 'Failed to publish lesson.';
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Academy Publisher | Admin | VulnArena</title>
</svelte:head>

<div class="publisher">
  <header class="pub-header">
    <h1 class="pub-title font-mono">
      <span class="c2-bracket">[</span>ACADEMY PUBLISHER<span class="c2-bracket">]</span>
    </h1>
    <p class="pub-sub font-mono">Compose and publish classified threat intelligence reports</p>
  </header>

  {#if success}
    <div class="alert alert-success font-mono">[OK] {success}</div>
  {/if}
  {#if error}
    <div class="alert alert-error font-mono">[ERR] {error}</div>
  {/if}

  <form class="pub-form" onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
    <!-- Metadata -->
    <div class="form-row-2">
      <div class="field">
        <label class="field-label font-mono" for="title">REPORT TITLE</label>
        <input id="title" type="text" class="field-input" bind:value={form.title} placeholder="Understanding SQL Injection Attacks" />
      </div>
      <div class="field">
        <label class="field-label font-mono" for="category">CLASSIFICATION</label>
        <select id="category" class="field-select" bind:value={form.category}>
          <option value="">Select category...</option>
          {#each categories as cat}
            <option value={cat}>{cat}</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="field">
      <label class="field-label font-mono" for="description">BRIEF</label>
      <input id="description" type="text" class="field-input" bind:value={form.description} placeholder="A comprehensive analysis of SQL injection attack vectors and defenses..." />
    </div>

    <div class="form-row-2">
      <div class="field">
        <label class="field-label font-mono" for="difficulty">DIFFICULTY (1-10)</label>
        <div class="range-row">
          <input id="difficulty" type="range" class="field-range" min="1" max="10" bind:value={form.difficulty} />
          <span class="range-value font-mono">{form.difficulty}</span>
        </div>
      </div>
      <div class="field">
        <label class="field-label font-mono" for="read_time">READ TIME (min, 0=auto)</label>
        <input id="read_time" type="number" class="field-input" bind:value={form.read_time_min} min="0" max="120" />
      </div>
    </div>

    <!-- Tags -->
    <div class="field">
      <label class="field-label font-mono">TAGS</label>
      <div class="tags-input-row">
        <input type="text" class="field-input" bind:value={tagInput} placeholder="Add a tag..." onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addTag(); } }} />
        <button type="button" class="tag-add-btn font-mono" onclick={addTag}>+ADD</button>
      </div>
      {#if form.tags.length > 0}
        <div class="tags-list">
          {#each form.tags as tag, i}
            <span class="tag-chip font-mono">
              {tag}
              <button type="button" class="tag-remove" onclick={() => removeTag(i)}>x</button>
            </span>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Markdown Editor + Preview -->
    <div class="field">
      <div class="editor-header">
        <label class="field-label font-mono">REPORT CONTENT (MARKDOWN)</label>
        <button type="button" class="preview-toggle font-mono" onclick={() => showPreview = !showPreview}>
          {showPreview ? 'EDIT' : 'PREVIEW'}
        </button>
      </div>

      {#if showPreview}
        <div class="markdown-preview">
          {@html getPreviewHtml()}
        </div>
      {:else}
        <textarea
          class="field-textarea content-area"
          rows="20"
          bind:value={form.content}
          placeholder={"# Report Title\n\n## Overview\n\nWrite your markdown content here...\n\n```python\n# Vulnerable code example\nquery = 'SELECT * FROM users WHERE id = ' + user_input\n```"}
          spellcheck="false"
        ></textarea>
      {/if}
    </div>

    <!-- Publish -->
    <div class="field">
      <label class="toggle-row">
        <input type="checkbox" bind:checked={form.is_published} />
        <span class="font-mono toggle-label">DECLASSIFY AND PUBLISH IMMEDIATELY</span>
      </label>
    </div>

    <div class="form-actions">
      <Button variant="primary" size="md" disabled={submitting}>
        {submitting ? 'PUBLISHING...' : 'PUBLISH REPORT'}
      </Button>
    </div>
  </form>
</div>

<style>
  .publisher {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .pub-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .pub-title {
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    color: var(--text-primary);
  }

  .c2-bracket { color: #ff6432; }

  .pub-sub {
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    letter-spacing: 0.1em;
  }

  /* Alerts */
  .alert {
    padding: var(--space-3);
    border-radius: var(--radius-md);
    font-size: 0.8rem;
    letter-spacing: 0.04em;
  }

  .alert-success {
    background: var(--accent-green-glow);
    border: 1px solid color-mix(in srgb, var(--accent-green) 30%, transparent);
    color: var(--accent-green);
  }

  .alert-error {
    background: var(--accent-red-glow);
    border: 1px solid color-mix(in srgb, var(--accent-red) 30%, transparent);
    color: var(--accent-red);
  }

  /* Form */
  .pub-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .form-row-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .field-label {
    font-size: 0.65rem;
    font-weight: 600;
    color: #ff6432;
    letter-spacing: 0.12em;
  }

  .field-input,
  .field-textarea,
  .field-select {
    background: var(--bg-primary);
    border: 1px solid rgba(255, 100, 50, 0.15);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: 0.85rem;
    padding: var(--space-2) var(--space-3);
    transition: border-color var(--transition-fast);
  }

  .field-input:focus,
  .field-textarea:focus,
  .field-select:focus {
    outline: none;
    border-color: #ff6432;
  }

  .field-textarea {
    resize: vertical;
    min-height: 60px;
  }

  .content-area {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    line-height: 1.6;
    tab-size: 4;
  }

  .field-select {
    cursor: pointer;
  }

  .range-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .field-range {
    accent-color: #ff6432;
    flex: 1;
  }

  .range-value {
    font-size: 1rem;
    font-weight: 700;
    color: #ff6432;
    min-width: 20px;
    text-align: center;
  }

  /* Tags */
  .tags-input-row {
    display: flex;
    gap: var(--space-2);
  }

  .tags-input-row .field-input {
    flex: 1;
  }

  .tag-add-btn {
    background: rgba(255, 100, 50, 0.1);
    border: 1px solid rgba(255, 100, 50, 0.3);
    border-radius: var(--radius-sm);
    color: #ff6432;
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    padding: var(--space-1) var(--space-3);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .tag-add-btn:hover {
    background: rgba(255, 100, 50, 0.2);
  }

  .tags-list {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin-top: var(--space-2);
  }

  .tag-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    background: rgba(255, 100, 50, 0.08);
    border: 1px solid rgba(255, 100, 50, 0.2);
    border-radius: var(--radius-sm);
    font-size: 0.7rem;
    color: var(--text-secondary);
  }

  .tag-remove {
    background: none;
    border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    font-size: 0.7rem;
    padding: 0 2px;
  }

  .tag-remove:hover {
    color: #ff4444;
  }

  /* Editor + Preview */
  .editor-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .preview-toggle {
    background: rgba(255, 100, 50, 0.08);
    border: 1px solid rgba(255, 100, 50, 0.25);
    border-radius: var(--radius-sm);
    color: #ff6432;
    font-size: 0.65rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    padding: 3px 10px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .preview-toggle:hover {
    background: rgba(255, 100, 50, 0.15);
  }

  .markdown-preview {
    background: var(--bg-primary);
    border: 1px solid rgba(255, 100, 50, 0.15);
    border-radius: var(--radius-sm);
    padding: var(--space-4);
    min-height: 400px;
    color: var(--text-primary);
    font-size: 0.9rem;
    line-height: 1.7;
    overflow-y: auto;
  }

  .markdown-preview :global(h1) { font-size: 1.4rem; margin-bottom: 0.5em; color: var(--text-primary); }
  .markdown-preview :global(h2) { font-size: 1.2rem; margin-top: 1em; margin-bottom: 0.4em; color: var(--text-primary); }
  .markdown-preview :global(h3) { font-size: 1rem; margin-top: 0.8em; color: var(--text-primary); }
  .markdown-preview :global(code) { font-family: var(--font-mono); background: var(--bg-secondary); padding: 2px 6px; border-radius: 3px; font-size: 0.85em; }
  .markdown-preview :global(pre) { background: var(--bg-secondary); padding: 1rem; border-radius: var(--radius-md); overflow-x: auto; margin: 1em 0; }
  .markdown-preview :global(pre code) { background: none; padding: 0; }
  .markdown-preview :global(blockquote) { border-left: 3px solid #ff6432; padding-left: 1rem; color: var(--text-secondary); margin: 1em 0; }
  .markdown-preview :global(a) { color: #ff6432; }
  .markdown-preview :global(ul), .markdown-preview :global(ol) { padding-left: 1.5rem; }

  /* Toggle */
  .toggle-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    cursor: pointer;
  }

  .toggle-label {
    font-size: 0.7rem;
    color: var(--text-secondary);
    letter-spacing: 0.06em;
  }

  /* Actions */
  .form-actions {
    display: flex;
    justify-content: flex-end;
    padding-top: var(--space-2);
  }

  @media (max-width: 768px) {
    .form-row-2 {
      grid-template-columns: 1fr;
    }
  }
</style>
