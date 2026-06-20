<script lang="ts">
  import { onMount } from 'svelte';
  import { Marked } from 'marked';

  interface Props {
    content: string;
  }

  let { content }: Props = $props();
  let renderedHtml = $state('');
  let containerEl: HTMLDivElement;

  const marked = new Marked();

  // Custom renderer for the cyber aesthetic
  // Uses `any` for token params to avoid type drift with marked v18+ token-based API
  const renderer = {
    heading(token: any) {
      const tag = `h${token.depth}`;
      const cls = token.depth <= 2 ? 'md-heading-major' : 'md-heading-minor';
      return `<${tag} class="${cls}">${token.text}</${tag}>`;
    },
    blockquote(token: any) {
      return `<blockquote class="md-callout">${token.text ?? ''}</blockquote>`;
    },
    code(token: any) {
      const text = token.text ?? '';
      const lang = token.lang ?? 'text';
      const escaped = text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
      return `<div class="md-code-block"><div class="md-code-header"><span class="md-code-lang">${lang}</span></div><pre class="md-pre"><code class="language-${lang}">${escaped}</code></pre></div>`;
    },
    codespan(token: any) {
      return `<code class="md-inline-code">${token.text ?? ''}</code>`;
    },
    hr() {
      return '<hr class="md-divider" />';
    },
    list(token: any) {
      const tag = token.ordered ? 'ol' : 'ul';
      const body = token.items?.map((item: any) => `<li>${item.text ?? ''}</li>`).join('') ?? token.body ?? '';
      return `<${tag} class="md-list">${body}</${tag}>`;
    },
    strong(token: any) {
      return `<strong class="md-bold">${token.text ?? ''}</strong>`;
    },
    link(token: any) {
      return `<a href="${token.href ?? ''}" class="md-link" target="_blank" rel="noopener">${token.text ?? ''}</a>`;
    },
  };

  marked.use({ renderer: renderer as any });

  $effect(() => {
    if (content) {
      renderedHtml = marked.parse(content) as string;
    }
  });
</script>

<div class="markdown-body" bind:this={containerEl}>
  {@html renderedHtml}
</div>

<style>
  /* Editorial / print-classic prose — serif headings, mono small-caps
     section labels, hairline rules, a ~66ch measure, sand-ink accent. */
  .markdown-body {
    font-family: var(--font-sans);
    color: var(--text-secondary);
    line-height: 1.7;
    font-size: var(--fs-body);
    max-width: var(--measure);
  }

  /* H1/H2 — serif display over a hairline */
  .markdown-body :global(.md-heading-major) {
    font-family: var(--font-serif);
    font-weight: 600;
    letter-spacing: -0.015em;
    line-height: 1.15;
    color: var(--text-primary);
    margin: var(--space-8) 0 var(--space-4);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-primary);
  }

  .markdown-body :global(h1.md-heading-major) {
    font-size: var(--fs-h1);
    letter-spacing: -0.02em;
    margin-top: var(--space-3);
  }

  .markdown-body :global(h2.md-heading-major) {
    font-size: var(--fs-h2);
  }

  /* H3/H4 — mono small-caps section labels over a hairline */
  .markdown-body :global(.md-heading-minor) {
    font-family: var(--font-mono);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.13em;
    color: var(--text-tertiary);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-primary);
    margin: var(--space-6) 0 var(--space-4);
  }

  .markdown-body :global(h3.md-heading-minor) {
    font-size: var(--fs-label);
  }

  .markdown-body :global(h4.md-heading-minor) {
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
  }

  /* Paragraphs — generous rhythm on the reading measure */
  .markdown-body :global(p) {
    margin: var(--space-4) 0;
    color: var(--text-secondary);
    line-height: 1.7;
  }

  /* Callout / blockquote — hairline rule, no fill, sand accent edge */
  .markdown-body :global(.md-callout) {
    border-left: 2px solid var(--accent-primary);
    padding: var(--space-1) 0 var(--space-1) var(--space-4);
    margin: var(--space-5) 0;
    color: var(--text-secondary);
    font-size: var(--fs-body);
  }

  .markdown-body :global(.md-callout p) {
    margin: var(--space-2) 0;
    color: var(--text-primary);
  }

  .markdown-body :global(.md-callout strong) {
    color: var(--accent-primary);
  }

  /* Code blocks — editor surface, hairline border, no shadow */
  .markdown-body :global(.md-code-block) {
    margin: var(--space-4) 0;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    overflow: hidden;
    background: var(--editor-bg);
  }

  .markdown-body :global(.md-code-header) {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-2) var(--space-4);
    border-bottom: 1px solid var(--border-primary);
  }

  .markdown-body :global(.md-code-lang) {
    font-family: var(--font-mono);
    font-size: var(--fs-eyebrow);
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .markdown-body :global(.md-pre) {
    margin: 0;
    padding: var(--space-4);
    overflow-x: auto;
  }

  .markdown-body :global(.md-pre code) {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    line-height: 1.6;
    color: var(--text-primary);
    background: none;
    padding: 0;
  }

  /* Inline code — quiet mono on the editor surface, hairline border */
  .markdown-body :global(.md-inline-code) {
    font-family: var(--font-mono);
    font-size: 0.875em;
    background: var(--editor-bg);
    color: var(--text-primary);
    padding: 0.1em 0.35em;
    border-radius: var(--radius-input);
    border: 1px solid var(--border-primary);
  }

  /* Honest status hues for inline status spans, if authored */
  .markdown-body :global(.md-inline-code.status-success) {
    color: var(--diff-1);
  }

  .markdown-body :global(.md-inline-code.status-error) {
    color: var(--accent-red);
  }

  /* Divider — hairline rule */
  .markdown-body :global(.md-divider) {
    border: none;
    border-top: 1px solid var(--border-primary);
    margin: var(--space-8) 0;
  }

  /* Lists */
  .markdown-body :global(.md-list) {
    margin: var(--space-4) 0;
    padding-left: var(--space-6);
  }

  .markdown-body :global(.md-list li) {
    margin-bottom: var(--space-2);
    color: var(--text-secondary);
    font-size: var(--fs-body);
    line-height: 1.7;
  }

  /* Bold — promote to primary ink */
  .markdown-body :global(.md-bold) {
    color: var(--text-primary);
    font-weight: 600;
  }

  /* Links — sand ink with a faint underline that deepens on hover */
  .markdown-body :global(.md-link) {
    color: var(--accent-primary);
    text-decoration: none;
    border-bottom: 1px solid color-mix(in srgb, var(--accent-primary) 40%, transparent);
    transition: border-color var(--transition-fast);
  }

  .markdown-body :global(.md-link:hover) {
    border-bottom-color: var(--accent-primary);
  }
</style>
