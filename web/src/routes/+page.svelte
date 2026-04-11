<script lang="ts">
  import Button from '$lib/components/ui/Button.svelte';
  import { isAuthenticated } from '$lib/stores/auth';
</script>

<div class="landing">
  <section class="hero">
    <div class="hero-badge">Cybersecurity training platform</div>
    <h1 class="hero-title">
      Find the <span class="accent">vulnerability</span>.<br />
      Write the <span class="accent">fix</span>.
    </h1>
    <p class="hero-subtitle">
      Interactive code challenges across 10+ languages. From basic injection flaws
      to advanced memory corruption in legacy systems. AI-powered assessment that
      understands your reasoning.
    </p>
    <div class="hero-actions">
      {#if $isAuthenticated}
        <a href="/arena">
          <Button variant="primary" size="lg">Enter the Arena</Button>
        </a>
        <a href="/academy">
          <Button variant="secondary" size="lg">Start Learning</Button>
        </a>
      {:else}
        <a href="/register">
          <Button variant="primary" size="lg">Get Started</Button>
        </a>
        <a href="/login">
          <Button variant="ghost" size="lg">Sign In</Button>
        </a>
      {/if}
    </div>
  </section>

  <section class="features">
    <div class="feature-card">
      <div class="feature-icon">&gt;_</div>
      <h3 class="feature-title">The Arena</h3>
      <p class="feature-desc">
        Hunt vulnerabilities in real-world code. SQLi, RCE, memory corruption,
        race conditions. Difficulty 1 through 10.
      </p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">[]</div>
      <h3 class="feature-title">The Academy</h3>
      <p class="feature-desc">
        Deep-dive breakdowns of how vulnerabilities manifest at the function level.
        From OWASP Top 10 to COBOL mainframes.
      </p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">{'{}'}</div>
      <h3 class="feature-title">Semantic Assessment</h3>
      <p class="feature-desc">
        No exact-match grading. Our NLP engine understands your reasoning.
        Explain the vuln in your own words.
      </p>
    </div>
  </section>

  <section class="languages">
    <p class="lang-label">Supported languages</p>
    <div class="lang-grid">
      {#each ['Go', 'Rust', 'Node.js', 'C#', 'C', 'C++', 'Assembly', 'Perl', 'COBOL', 'Flutter'] as lang}
        <span class="lang-tag">{lang}</span>
      {/each}
    </div>
  </section>
</div>

<style>
  .landing {
    display: flex;
    flex-direction: column;
    gap: var(--space-16);
    padding: var(--space-12) 0;
  }

  .hero {
    text-align: center;
    max-width: 720px;
    margin: 0 auto;
  }

  .hero-badge {
    display: inline-block;
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-secondary);
    border: 1px solid var(--border-primary);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    margin-bottom: var(--space-6);
  }

  .hero-title {
    font-family: var(--font-serif);
    font-size: clamp(2rem, 5vw, 3.5rem);
    font-weight: 700;
    line-height: 1.15;
    color: var(--text-primary);
    margin-bottom: var(--space-5);
  }

  .accent {
    color: var(--accent-primary);
  }

  .hero-subtitle {
    font-size: 1.0625rem;
    color: var(--text-secondary);
    line-height: 1.7;
    margin-bottom: var(--space-8);
  }

  .hero-actions {
    display: flex;
    gap: var(--space-4);
    justify-content: center;
    flex-wrap: wrap;
  }

  .hero-actions a {
    text-decoration: none;
  }

  .features {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: var(--space-6);
  }

  .feature-card {
    background: var(--bg-surface);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-lg);
    padding: var(--space-8);
    transition: all var(--transition-base);
  }

  .feature-card:hover {
    border-color: var(--border-secondary);
    transform: translateY(-2px);
  }

  .feature-icon {
    font-size: 1.5rem;
    color: var(--text-secondary);
    margin-bottom: var(--space-4);
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
    border-radius: var(--radius-md);
  }

  .feature-title {
    font-family: var(--font-serif);
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: var(--space-2);
  }

  .feature-desc {
    font-size: 0.875rem;
    color: var(--text-secondary);
    line-height: 1.6;
  }

  .languages {
    text-align: center;
  }

  .lang-label {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    margin-bottom: var(--space-4);
  }

  .lang-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    justify-content: center;
  }

  .lang-tag {
    font-size: 0.75rem;
    color: var(--text-secondary);
    border: 1px solid var(--border-primary);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .lang-tag:hover {
    color: var(--text-primary);
    border-color: var(--border-secondary);
  }
</style>
