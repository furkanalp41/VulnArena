<script lang="ts">
  import { createChallenge, type CreateChallengeInput } from '$lib/api/admin';
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  let form: CreateChallengeInput = $state({
    title: '',
    description: '',
    difficulty: 5,
    language_slug: '',
    vuln_category_slug: '',
    vulnerable_code: '',
    target_vulnerability: '',
    conceptual_fix: '',
    hints: [],
    points: 100,
    is_published: true,
  });

  let hintInput = $state('');
  let submitting = $state(false);
  let success = $state('');
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
    success = '';

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
      await createChallenge(form);
      success = `Challenge "${form.title}" deployed successfully.`;
      form = {
        title: '', description: '', difficulty: 5,
        language_slug: '', vuln_category_slug: '',
        vulnerable_code: '', target_vulnerability: '', conceptual_fix: '',
        hints: [], points: 100, is_published: true,
      };
    } catch (e: any) {
      error = e?.message || 'Failed to create challenge.';
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Challenge Forge | Admin | VulnArena</title>
</svelte:head>

<div class="forge">
  <header class="forge-header">
    <h1 class="forge-title font-mono">
      <span class="c2-bracket">[</span>CHALLENGE FORGE<span class="c2-bracket">]</span>
    </h1>
    <p class="forge-sub font-mono">Deploy a new vulnerable code lab into the Arena</p>
  </header>

  {#if success}
    <div class="alert alert-success font-mono">[OK] {success}</div>
  {/if}
  {#if error}
    <div class="alert alert-error font-mono">[ERR] {error}</div>
  {/if}

  <form class="forge-form" onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
    <!-- Metadata Row -->
    <div class="form-row-2">
      <div class="field">
        <label class="field-label font-mono" for="title">TITLE</label>
        <input id="title" type="text" class="field-input" bind:value={form.title} placeholder="SQL Injection in User Login" />
      </div>
      <div class="field">
        <label class="field-label font-mono" for="points">XP POINTS</label>
        <input id="points" type="number" class="field-input" bind:value={form.points} min="25" max="1000" step="25" />
      </div>
    </div>

    <div class="field">
      <label class="field-label font-mono" for="description">DESCRIPTION</label>
      <textarea id="description" class="field-textarea" rows="3" bind:value={form.description} placeholder="Analyze the following code snippet and identify the vulnerability..."></textarea>
    </div>

    <!-- Config Row -->
    <div class="form-row-3">
      <div class="field">
        <label class="field-label font-mono" for="language">LANGUAGE</label>
        <select id="language" class="field-select" bind:value={form.language_slug}>
          <option value="">Select...</option>
          {#each languages as lang}
            <option value={lang.slug}>{lang.name}</option>
          {/each}
        </select>
      </div>
      <div class="field">
        <label class="field-label font-mono" for="vuln_cat">VULNERABILITY TYPE</label>
        <select id="vuln_cat" class="field-select" bind:value={form.vuln_category_slug}>
          <option value="">Select...</option>
          {#each vulnCategories as cat}
            <option value={cat.slug}>{cat.name}</option>
          {/each}
        </select>
      </div>
      <div class="field">
        <label class="field-label font-mono" for="difficulty">DIFFICULTY (1-10)</label>
        <input id="difficulty" type="range" class="field-range" min="1" max="10" bind:value={form.difficulty} />
        <span class="range-value font-mono">{form.difficulty}</span>
      </div>
    </div>

    <!-- Code Sections -->
    <div class="field">
      <label class="field-label font-mono" for="vuln_code">VULNERABLE CODE</label>
      <textarea id="vuln_code" class="field-textarea code-area" rows="12" bind:value={form.vulnerable_code} placeholder="Paste the vulnerable code snippet here..." spellcheck="false"></textarea>
    </div>

    <div class="form-row-2">
      <div class="field">
        <label class="field-label font-mono" for="target_vuln">TARGET VULNERABILITY (Server-only)</label>
        <textarea id="target_vuln" class="field-textarea" rows="4" bind:value={form.target_vulnerability} placeholder="The SQL query is built via string concatenation, allowing injection..."></textarea>
      </div>
      <div class="field">
        <label class="field-label font-mono" for="fix">CONCEPTUAL FIX (Server-only)</label>
        <textarea id="fix" class="field-textarea" rows="4" bind:value={form.conceptual_fix} placeholder="Use parameterized queries or prepared statements..."></textarea>
      </div>
    </div>

    <!-- Hints -->
    <div class="field">
      <label class="field-label font-mono">HINTS</label>
      <div class="hints-input-row">
        <input type="text" class="field-input" bind:value={hintInput} placeholder="Add a hint..." onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addHint(); } }} />
        <button type="button" class="hint-add-btn font-mono" onclick={addHint}>+ADD</button>
      </div>
      {#if form.hints.length > 0}
        <div class="hints-list">
          {#each form.hints as hint, i}
            <div class="hint-item font-mono">
              <span class="hint-num">#{i + 1}</span>
              <span class="hint-text">{hint}</span>
              <button type="button" class="hint-remove" onclick={() => removeHint(i)}>x</button>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Publish toggle -->
    <div class="field">
      <label class="toggle-row">
        <input type="checkbox" bind:checked={form.is_published} />
        <span class="font-mono toggle-label">PUBLISH IMMEDIATELY</span>
      </label>
    </div>

    <div class="form-actions">
      <Button variant="primary" size="md" disabled={submitting}>
        {submitting ? 'DEPLOYING...' : 'DEPLOY CHALLENGE'}
      </Button>
    </div>
  </form>
</div>

<style>
  .forge {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .forge-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .forge-title {
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    color: var(--text-primary);
  }

  .c2-bracket { color: #ff6432; }

  .forge-sub {
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
  .forge-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .form-row-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
  }

  .form-row-3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
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

  .code-area {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    line-height: 1.5;
    tab-size: 4;
  }

  .field-select {
    cursor: pointer;
  }

  .field-range {
    accent-color: #ff6432;
    width: 100%;
    margin-top: var(--space-1);
  }

  .range-value {
    font-size: 1rem;
    font-weight: 700;
    color: #ff6432;
    text-align: center;
  }

  /* Hints */
  .hints-input-row {
    display: flex;
    gap: var(--space-2);
  }

  .hints-input-row .field-input {
    flex: 1;
  }

  .hint-add-btn {
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

  .hint-add-btn:hover {
    background: rgba(255, 100, 50, 0.2);
  }

  .hints-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    margin-top: var(--space-2);
  }

  .hint-item {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-2);
    background: var(--bg-secondary);
    border-radius: var(--radius-sm);
    font-size: 0.8rem;
  }

  .hint-num {
    color: #ff6432;
    font-size: 0.65rem;
    flex-shrink: 0;
  }

  .hint-text {
    flex: 1;
    color: var(--text-secondary);
  }

  .hint-remove {
    background: none;
    border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    font-size: 0.75rem;
    padding: 2px 6px;
  }

  .hint-remove:hover {
    color: #ff4444;
  }

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

  /* Responsive */
  @media (max-width: 768px) {
    .form-row-2, .form-row-3 {
      grid-template-columns: 1fr;
    }
  }
</style>
