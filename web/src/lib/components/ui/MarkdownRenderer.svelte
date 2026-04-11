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
  .markdown-body {
    font-family: var(--font-sans);
    color: var(--text-primary);
    line-height: 1.75;
    font-size: 0.9375rem;
  }

  /* Headings */
  .markdown-body :global(.md-heading-major) {
    font-family: var(--font-serif);
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--text-primary);
    margin-top: 2.5rem;
    margin-bottom: 1rem;
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--border-primary);
  }

  .markdown-body :global(h1.md-heading-major) {
    font-size: 1.625rem;
    color: var(--text-primary);
    border-bottom-color: var(--border-secondary);
  }

  .markdown-body :global(h2.md-heading-major) {
    font-size: 1.25rem;
  }

  .markdown-body :global(.md-heading-minor) {
    font-family: var(--font-serif);
    font-weight: 600;
    color: var(--text-primary);
    margin-top: 1.75rem;
    margin-bottom: 0.75rem;
  }

  .markdown-body :global(h3.md-heading-minor) {
    font-size: 1.0625rem;
  }

  .markdown-body :global(h4.md-heading-minor) {
    font-size: 0.9375rem;
    color: var(--text-secondary);
  }

  /* Paragraphs */
  .markdown-body :global(p) {
    margin-bottom: 1rem;
    color: var(--text-secondary);
  }

  /* Callout / blockquote */
  .markdown-body :global(.md-callout) {
    border-left: 3px solid var(--accent-yellow);
    background: var(--bg-tertiary);
    padding: 1rem 1.25rem;
    margin: 1.5rem 0;
    border-radius: 0 var(--radius-md) var(--radius-md) 0;
    font-size: 0.875rem;
  }

  .markdown-body :global(.md-callout p) {
    margin-bottom: 0;
    color: var(--text-primary);
    font-weight: 500;
  }

  /* Code blocks */
  .markdown-body :global(.md-code-block) {
    margin: 1.25rem 0;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    overflow: hidden;
    background: var(--editor-bg);
  }

  .markdown-body :global(.md-code-header) {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.375rem 1rem;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
  }

  .markdown-body :global(.md-code-lang) {
    font-family: var(--font-mono);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .markdown-body :global(.md-pre) {
    margin: 0;
    padding: 1rem;
    overflow-x: auto;
  }

  .markdown-body :global(.md-pre code) {
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    line-height: 1.6;
    color: var(--text-primary);
    background: none;
    padding: 0;
  }

  /* Inline code */
  .markdown-body :global(.md-inline-code) {
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    background: var(--bg-tertiary);
    color: var(--accent-green);
    padding: 0.125rem 0.375rem;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-primary);
  }

  /* Divider */
  .markdown-body :global(.md-divider) {
    border: none;
    border-top: 1px solid var(--border-primary);
    margin: 2rem 0;
  }

  /* Lists */
  .markdown-body :global(.md-list) {
    margin: 0.75rem 0;
    padding-left: 1.5rem;
  }

  .markdown-body :global(.md-list li) {
    margin-bottom: 0.375rem;
    color: var(--text-secondary);
    font-size: 0.9375rem;
  }

  /* Bold */
  .markdown-body :global(.md-bold) {
    color: var(--text-primary);
    font-weight: 600;
  }

  /* Links */
  .markdown-body :global(.md-link) {
    color: var(--accent-blue);
    text-decoration: none;
    border-bottom: 1px dashed var(--accent-blue-dim);
    transition: all var(--transition-fast);
  }

  .markdown-body :global(.md-link:hover) {
    color: var(--accent-blue-dim);
    border-bottom-style: solid;
  }

  /* Strong emphasis in first child (for the classification header) */
  .markdown-body :global(.md-callout strong) {
    color: var(--accent-yellow);
  }
</style>
