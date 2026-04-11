<script lang="ts">
  import { goto } from '$app/navigation';
  import { submitCommunityChallenge, type CommunitySubmitInput } from '$lib/api/community';
  import CodeEditor from '$lib/components/editor/CodeEditor.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';

  let form: CommunitySubmitInput = $state({
    title: '',
    description: '',
    difficulty: 5,
    language_slug: '',
    vuln_category_slug: '',
    vulnerable_code: '',
    target_vulnerability: '',
    conceptual_fix: '',
    vulnerable_lines: '',
    hints: [],
    points: 100,
  });

  let hintInput = $state('');
  let submitting = $state(false);
  let error = $state('');

  const languages = [
    { slug: 'go', name: 'Go' },
    { slug: 'javascript', name: 'Node.js' },
    { slug: 'python', name: 'Python' },
    { slug: 'java', name: 'Java' },
    { slug: 'csharp', name: 'C#' },
    { slug: 'rust', name: 'Rust' },
    { slug: 'c', name: 'C' },
    { slug: 'cpp', name: 'C++' },
    { slug: 'ruby', name: 'Ruby' },
  ];

  const vulnCategories = [
    { slug: 'injection', name: 'Injection' },
    { slug: 'broken-auth', name: 'Broken Authentication' },
    { slug: 'xss', name: 'Cross-Site Scripting (XSS)' },
    { slug: 'insecure-deser', name: 'Insecure Deserialization' },
    { slug: 'broken-access', name: 'Broken Access Control' },
    { slug: 'security-misconfig', name: 'Security Misconfiguration' },
    { slug: 'crypto-failures', name: 'Cryptographic Failures' },
    { slug: 'ssrf', name: 'SSRF' },
    { slug: 'cmd-injection', name: 'Command Injection' },
    { slug: 'memory-corruption', name: 'Memory Corruption' },
    { slug: 'race-condition', name: 'Race Condition' },
    { slug: 'rce', name: 'Remote Code Execution' },
  ];

  function addHint() {
    if (hintInput.trim()) {
      form.hints = [...form.hints, hintInput.trim()];
      hintInput = '';
    }
  }

  function removeHint(index: number) {
    form.hints = form.hints.filter((_, i) => i !== index);
  }

  async function handleSubmit() {
    error = '';

    if (!form.title || !form.vulnerable_code || !form.target_vulnerability) {
      error = 'Title, vulnerable code, and target vulnerability are required.';
      return;
    }
    if (!form.language_slug || !form.vuln_category_slug) {
      error = 'Language and vulnerability category are required.';
      return;
    }

    submitting = true;
    try {
      await submitCommunityChallenge(form);
      goto('/forge');
    } catch (e: any) {
      error = e.message || 'Failed to submit challenge';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="forge-new">
  <div class="page-header">
    <a href="/forge" class="back-link">&larr; Forge</a>
    <h1 class="page-title">New submission</h1>
    <p class="page-subtitle">Submit a vulnerable code snippet for community review.</p>
  </div>

  <form class="form-layout" onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
    <div class="form-grid">
      <!-- Left: Code & Details -->
      <div class="form-left">
        <Card>
          <div class="card-inner">
            <label class="form-label">Title</label>
            <input
              type="text"
              class="form-input"
              placeholder="e.g., Unsafe SQL Query Builder"
              bind:value={form.title}
            />

            <label class="form-label">Description</label>
            <textarea
              class="form-textarea"
              placeholder="Describe the scenario and what makes this code vulnerable..."
              bind:value={form.description}
              rows="3"
            ></textarea>

            <div class="form-row">
              <div class="form-field">
                <label class="form-label">Language</label>
                <select class="form-select" bind:value={form.language_slug}>
                  <option value="">Select...</option>
                  {#each languages as lang}
                    <option value={lang.slug}>{lang.name}</option>
                  {/each}
                </select>
              </div>
              <div class="form-field">
                <label class="form-label">Vulnerability type</label>
                <select class="form-select" bind:value={form.vuln_category_slug}>
                  <option value="">Select...</option>
                  {#each vulnCategories as cat}
                    <option value={cat.slug}>{cat.name}</option>
                  {/each}
                </select>
              </div>
            </div>

            <div class="form-row">
              <div class="form-field">
                <label class="form-label">Difficulty (1-10)</label>
                <input
                  type="range"
                  min="1"
                  max="10"
                  bind:value={form.difficulty}
                  class="form-range"
                />
                <span class="range-val">{form.difficulty}</span>
              </div>
              <div class="form-field">
                <label class="form-label">Points</label>
                <input
                  type="number"
                  class="form-input"
                  min="50"
                  max="1000"
                  step="25"
                  bind:value={form.points}
                />
              </div>
            </div>

            <label class="form-label">Vulnerable code</label>
            <textarea
              class="form-textarea code-textarea font-mono"
              placeholder="Paste your vulnerable code snippet here..."
              bind:value={form.vulnerable_code}
              rows="14"
            ></textarea>

            <label class="form-label">Vulnerable lines (comma-separated)</label>
            <input
              type="text"
              class="form-input"
              placeholder="e.g., 12, 15, 23-25"
              bind:value={form.vulnerable_lines}
            />
          </div>
        </Card>
      </div>

      <!-- Right: Solution & Hints -->
      <div class="form-right">
        <Card>
          <div class="card-inner">
            <label class="form-label">Target vulnerability</label>
            <textarea
              class="form-textarea"
              placeholder="Describe the vulnerability: what it is, how it can be exploited..."
              bind:value={form.target_vulnerability}
              rows="5"
            ></textarea>

            <label class="form-label">Conceptual fix</label>
            <textarea
              class="form-textarea"
              placeholder="Describe how to fix the vulnerability..."
              bind:value={form.conceptual_fix}
              rows="5"
            ></textarea>

            <label class="form-label">Hints</label>
            <div class="hints-area">
              {#each form.hints as hint, i}
                <div class="hint-row">
                  <span class="hint-num">#{i + 1}</span>
                  <span class="hint-text">{hint}</span>
                  <button type="button" class="hint-remove" onclick={() => removeHint(i)}>x</button>
                </div>
              {/each}
              <div class="hint-input-row">
                <input
                  type="text"
                  class="form-input"
                  placeholder="Add a hint..."
                  bind:value={hintInput}
                  onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addHint(); } }}
                />
                <Button variant="ghost" size="sm" onclick={addHint} disabled={!hintInput.trim()}>Add</Button>
              </div>
            </div>
          </div>
        </Card>

        {#if form.vulnerable_code && form.language_slug}
          <Card>
            <div class="card-inner">
              <label class="form-label">Preview</label>
              <div class="preview-editor">
                <CodeEditor
                  code={form.vulnerable_code}
                  language={form.language_slug}
                  readonly={true}
                  height="250px"
                />
              </div>
            </div>
          </Card>
        {/if}

        {#if error}
          <p class="form-error">{error}</p>
        {/if}

        <div class="form-actions">
          <Button variant="primary" size="lg" onclick={handleSubmit} loading={submitting} disabled={submitting}>
            Submit for review
          </Button>
          <a href="/forge"><Button variant="ghost">Cancel</Button></a>
        </div>
      </div>
    </div>
  </form>
</div>

<style>
  .forge-new {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .page-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .back-link {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    text-decoration: none;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .page-title {
    font-family: var(--font-serif);
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .page-subtitle {
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
  }

  .form-left, .form-right {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .card-inner {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .form-label {
    font-family: var(--font-sans);
    font-size: 0.625rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .form-input, .form-textarea, .form-select {
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    padding: var(--space-2) var(--space-3);
    background: var(--bg-input);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    outline: none;
    width: 100%;
    transition: border-color var(--transition-fast);
  }

  .form-input:focus, .form-textarea:focus, .form-select:focus {
    border-color: var(--accent-primary);
  }

  .form-textarea {
    resize: vertical;
    line-height: 1.5;
  }

  .code-textarea {
    font-size: 0.75rem;
    line-height: 1.6;
    tab-size: 4;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .form-range {
    width: 100%;
    accent-color: var(--accent-primary);
  }

  .range-val {
    font-family: var(--font-sans);
    font-size: 0.75rem;
    color: var(--text-secondary);
    text-align: center;
  }

  .hints-area {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .hint-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-2);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }

  .hint-num {
    font-size: 0.625rem;
    color: var(--accent-yellow);
    font-weight: 600;
    flex-shrink: 0;
  }

  .hint-text {
    flex: 1;
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .hint-remove {
    background: none;
    border: none;
    color: var(--accent-red);
    cursor: pointer;
    font-size: 0.75rem;
    padding: 0 4px;
  }

  .hint-input-row {
    display: flex;
    gap: var(--space-2);
  }

  .preview-editor {
    border-radius: var(--radius-sm);
    overflow: hidden;
  }

  .form-error {
    font-size: 0.8125rem;
    color: var(--accent-red);
  }

  .form-actions {
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }

  @media (max-width: 1024px) {
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
