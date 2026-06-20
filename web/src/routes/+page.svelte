<script lang="ts">
  import Button from '$lib/components/ui/Button.svelte';
  import { isAuthenticated } from '$lib/stores/auth';

  const languages = ['Go', 'Rust', 'Node.js', 'C#', 'C', 'C++', 'Assembly', 'Perl', 'COBOL', 'Fortran', 'Flutter'];
</script>

<div class="landing shell">
  <section class="hero">
    <span class="eyebrow">Secure-code-review platform</span>
    <h1 class="hero-title">
      Find the <span class="ink">vulnerability</span>.<br />
      Write the <span class="ink">fix</span>.
    </h1>
    <p class="hero-sub">
      A white-box training ground: read real vulnerable code in eleven languages,
      mark the lines that bite, and explain the flaw in your own words for semantic assessment.
    </p>
    <div class="hero-cta">
      {#if $isAuthenticated}
        <a href="/arena"><Button variant="primary" size="lg">Enter the Arena</Button></a>
        <a href="/academy"><Button variant="ghost" size="lg">Browse the Academy</Button></a>
      {:else}
        <a href="/register"><Button variant="primary" size="lg">Start an audit</Button></a>
        <a href="/login"><Button variant="ghost" size="lg">Sign in</Button></a>
      {/if}
    </div>
  </section>

  <div class="features">
    <article class="feature">
      <div class="numeral" aria-hidden="true">I</div>
      <h3>The Arena</h3>
      <p>Real CVEs and planted bugs, set in readable code. Flag the lines, then defend your reasoning in prose.</p>
    </article>
    <article class="feature">
      <div class="numeral" aria-hidden="true">II</div>
      <h3>The Academy</h3>
      <p>Long-form write-ups that turn each class of flaw into something you can recognise on sight — from OWASP Top 10 to COBOL mainframes and Fortran HPC kernels.</p>
    </article>
    <article class="feature">
      <div class="numeral" aria-hidden="true">III</div>
      <h3>Semantic Assessment</h3>
      <p>Explain the vulnerability in your own words. Claude grades meaning, not keywords — partial credit for partial insight.</p>
    </article>
  </div>

  <div class="langs">
    <span class="smallcaps">Eleven languages, one discipline</span>
    <div class="lang-list">
      {#each languages as lang}
        <span>{lang}</span>
      {/each}
    </div>
  </div>

  <footer class="colophon">
    <span>VulnArena — a journal of secure code review</span>
    <span>Set in Source Serif 4 · Inter · JetBrains Mono</span>
  </footer>
</div>

<style>
  .landing {
    padding: var(--space-12) 0 var(--space-8);
  }

  .hero {
    padding: var(--space-8) 0 var(--space-10);
    max-width: 760px;
  }

  .hero-title {
    font-family: var(--font-serif);
    font-size: var(--fs-display);
    font-weight: 700;
    line-height: 1.06;
    letter-spacing: -0.02em;
    color: var(--text-primary);
    margin: var(--space-4) 0 var(--space-5);
    text-wrap: balance;
  }

  .hero-title .ink {
    color: var(--accent-primary);
  }

  .hero-sub {
    font-size: var(--fs-lead);
    color: var(--text-secondary);
    line-height: 1.5;
    max-width: 56ch;
  }

  .hero-cta {
    display: flex;
    gap: var(--space-3);
    margin-top: var(--space-6);
    flex-wrap: wrap;
  }

  .hero-cta a {
    text-decoration: none;
  }

  /* Three features divided by hairline rules, not boxes. */
  .features {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0;
    border-top: 1px solid var(--border-primary);
  }

  .feature {
    padding: var(--space-6) var(--space-5) var(--space-6) 0;
    border-right: 1px solid var(--border-primary);
  }

  .feature:last-child {
    border-right: 0;
    padding-right: 0;
  }

  .feature:not(:first-child) {
    padding-left: var(--space-5);
  }

  .numeral {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-input);
    font-family: var(--font-serif);
    font-size: 0.95rem;
    color: var(--accent-primary);
    letter-spacing: 0.04em;
    margin-bottom: var(--space-4);
  }

  .feature h3 {
    font-family: var(--font-serif);
    font-size: var(--fs-h4);
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    margin-bottom: var(--space-2);
  }

  .feature p {
    color: var(--text-secondary);
    font-size: var(--fs-body);
    line-height: 1.6;
  }

  .langs {
    padding: var(--space-6) 0;
    border-top: 1px solid var(--border-primary);
    margin-top: var(--space-8);
  }

  .lang-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0;
    margin-top: var(--space-3);
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
  }

  .lang-list span {
    color: var(--text-secondary);
    letter-spacing: 0.02em;
  }

  .lang-list span:not(:last-child)::after {
    content: '·';
    color: var(--text-tertiary);
    margin: 0 0.7em;
  }

  .colophon {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: var(--space-3);
    border-top: 1px solid var(--border-primary);
    margin-top: var(--space-6);
    padding-top: var(--space-5);
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
  }

  @media (max-width: 880px) {
    .features {
      grid-template-columns: 1fr;
    }

    .feature {
      border-right: 0;
      border-bottom: 1px solid var(--border-primary);
      padding: var(--space-5) 0;
    }

    .feature:not(:first-child) {
      padding-left: 0;
    }

    .hero-title {
      font-size: 2.4rem;
    }
  }
</style>
