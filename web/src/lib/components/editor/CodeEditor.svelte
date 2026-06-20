<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import type { RemoteCursor, RemoteSelection } from '$lib/stores/collab';
  import { theme } from '$lib/stores/theme';

  interface Props {
    code: string;
    language?: string;
    readonly?: boolean;
    height?: string;
    selectedLines?: number[];
    onLineToggle?: (line: number) => void;
    remoteCursors?: Map<string, RemoteCursor>;
    remoteSelections?: Map<string, RemoteSelection>;
    onCursorMove?: (line: number, column: number) => void;
  }

  let {
    code,
    language = 'go',
    readonly = true,
    height = '100%',
    selectedLines = [],
    onLineToggle,
    remoteCursors,
    remoteSelections,
    onCursorMove,
  }: Props = $props();

  let container: HTMLDivElement;
  let editor: any;
  let monaco: any;
  let decorationIds: string[] = [];
  let remoteDecorationIds: string[] = [];
  let resizeObserver: ResizeObserver | null = null;
  let cursorWidgets: Map<string, any> = new Map();
  let cursorPositionDisposable: any = null;

  // Map our language slugs to Monaco language IDs
  const languageMap: Record<string, string> = {
    go: 'go',
    rust: 'rust',
    nodejs: 'javascript',
    javascript: 'javascript',
    csharp: 'csharp',
    c: 'c',
    cpp: 'cpp',
    assembly: 'plaintext',
    perl: 'perl',
    cobol: 'plaintext',
    fortran: 'plaintext',
    flutter: 'dart',
    python: 'python',
    ruby: 'ruby',
    java: 'java',
  };

  function defineVulnArenaTheme(m: any) {
    m.editor.defineTheme('vulnarena-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        { token: 'comment', foreground: '6b6560', fontStyle: 'italic' },
        { token: 'keyword', foreground: '8b9dc3' },
        { token: 'string', foreground: '7c9f6b' },
        { token: 'number', foreground: 'd4a574' },
        { token: 'type', foreground: 'a78bba' },
        { token: 'function', foreground: 'f5f0eb' },
        { token: 'variable', foreground: 'e8e0d8' },
        { token: 'operator', foreground: 'a8a29e' },
        { token: 'delimiter', foreground: '78716c' },
        { token: 'identifier', foreground: 'e8e0d8' },
      ],
      colors: {
        'editor.background': '#1e1e1e',
        'editor.foreground': '#f5f0eb',
        'editor.lineHighlightBackground': '#2a2a2a40',
        'editor.selectionBackground': '#3a3a5a80',
        'editor.inactiveSelectionBackground': '#3a3a5a40',
        'editorLineNumber.foreground': '#5a5a5a',
        'editorLineNumber.activeForeground': '#d4a574',
        'editorCursor.foreground': '#d4a574',
        'editor.selectionHighlightBackground': '#d4a57415',
        'editorIndentGuide.background': '#333333',
        'editorIndentGuide.activeBackground': '#444444',
        'editorGutter.background': '#1a1a1a',
        'scrollbar.shadow': '#00000000',
        'scrollbarSlider.background': '#44444450',
        'scrollbarSlider.hoverBackground': '#44444480',
        'scrollbarSlider.activeBackground': '#444444a0',
        'editorOverviewRuler.border': '#1e1e1e',
        'minimap.background': '#1a1a1a',
      },
    });

    // Light variant — Monaco themes are registered imperatively in JS (they
    // cannot read CSS custom properties), so the warm light palette is mirrored
    // here key-for-key and swapped reactively on theme change.
    m.editor.defineTheme('vulnarena-light', {
      base: 'vs',
      inherit: true,
      rules: [
        { token: 'comment', foreground: '9c9690', fontStyle: 'italic' },
        { token: 'keyword', foreground: '6b7fa3' },
        { token: 'string', foreground: '6b8f5b' },
        { token: 'number', foreground: 'b8845a' },
        { token: 'type', foreground: '8b6fa8' },
        { token: 'function', foreground: '2c2723' },
        { token: 'variable', foreground: '3a342e' },
        { token: 'operator', foreground: '6b6560' },
        { token: 'delimiter', foreground: '9c9690' },
        { token: 'identifier', foreground: '2c2723' },
      ],
      colors: {
        'editor.background': '#faf8f5',
        'editor.foreground': '#1a1a1a',
        'editor.lineHighlightBackground': '#00000008',
        'editor.selectionBackground': '#e3ddd4',
        'editor.inactiveSelectionBackground': '#e8e3de80',
        'editorLineNumber.foreground': '#c2bab0',
        'editorLineNumber.activeForeground': '#b8845a',
        'editorCursor.foreground': '#b8845a',
        'editor.selectionHighlightBackground': '#b8845a18',
        'editorIndentGuide.background': '#e8e3de',
        'editorIndentGuide.activeBackground': '#cdc6be',
        'editorGutter.background': '#f2eeea',
        'scrollbar.shadow': '#00000000',
        'scrollbarSlider.background': '#c9c1b650',
        'scrollbarSlider.hoverBackground': '#c9c1b680',
        'scrollbarSlider.activeBackground': '#c9c1b6a0',
        'editorOverviewRuler.border': '#faf8f5',
        'minimap.background': '#f2eeea',
      },
    });
  }

  function updateDecorations() {
    if (!editor || !monaco) return;
    const newDecorations = selectedLines.map((line) => ({
      range: new monaco.Range(line, 1, line, 1),
      options: {
        isWholeLine: true,
        className: 'vuln-line-highlight',
        glyphMarginClassName: 'vuln-glyph-margin',
        overviewRuler: {
          color: '#c9726b',
          position: monaco.editor.OverviewRulerLane.Full,
        },
      },
    }));
    decorationIds = editor.deltaDecorations(decorationIds, newDecorations);
  }

  // ─── Remote cursor widgets ───

  function updateRemoteCursors() {
    if (!editor || !monaco || !remoteCursors) return;

    // Remove stale widgets
    for (const [userId, widget] of cursorWidgets) {
      if (!remoteCursors.has(userId)) {
        editor.removeContentWidget(widget);
        cursorWidgets.delete(userId);
      }
    }

    // Add/update widgets
    for (const [userId, cursor] of remoteCursors) {
      const existingWidget = cursorWidgets.get(userId);
      if (existingWidget) {
        editor.removeContentWidget(existingWidget);
      }

      const domNode = document.createElement('div');
      domNode.className = 'remote-cursor-widget';
      domNode.style.borderLeftColor = cursor.color;

      const label = document.createElement('div');
      label.className = 'remote-cursor-label';
      label.textContent = cursor.username;
      label.style.backgroundColor = cursor.color;
      domNode.appendChild(label);

      const widget = {
        getId: () => `remote-cursor-${userId}`,
        getDomNode: () => domNode,
        getPosition: () => ({
          position: { lineNumber: cursor.line, column: cursor.column },
          preference: [monaco.editor.ContentWidgetPositionPreference.EXACT],
        }),
      };

      editor.addContentWidget(widget);
      cursorWidgets.set(userId, widget);
    }
  }

  // ─── Remote line selections ───

  function updateRemoteSelections() {
    if (!editor || !monaco || !remoteSelections) return;

    const newDecorations: any[] = [];

    for (const [, sel] of remoteSelections) {
      for (const line of sel.lines) {
        newDecorations.push({
          range: new monaco.Range(line, 1, line, 1),
          options: {
            isWholeLine: true,
            className: `remote-selection-highlight`,
            before: {
              content: ' ',
              inlineClassName: 'remote-selection-marker',
              inlineClassNameAffectsLetterSpacing: false,
            },
            overviewRuler: {
              color: sel.color,
              position: monaco.editor.OverviewRulerLane.Center,
            },
          },
        });
      }
    }

    remoteDecorationIds = editor.deltaDecorations(remoteDecorationIds, newDecorations);
  }

  onMount(async () => {
    monaco = await import('monaco-editor');

    defineVulnArenaTheme(monaco);

    editor = monaco.editor.create(container, {
      value: code,
      language: languageMap[language] || 'plaintext',
      theme: $theme === 'light' ? 'vulnarena-light' : 'vulnarena-dark',
      readOnly: readonly,
      minimap: { enabled: true, scale: 1, showSlider: 'mouseover' },
      fontSize: 13.5,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
      fontLigatures: true,
      lineNumbers: 'on',
      renderLineHighlight: 'line',
      scrollBeyondLastLine: false,
      wordWrap: 'off',
      padding: { top: 16, bottom: 16 },
      smoothScrolling: true,
      cursorBlinking: 'smooth',
      cursorSmoothCaretAnimation: 'on',
      bracketPairColorization: { enabled: true },
      guides: { bracketPairs: true, indentation: true },
      overviewRulerBorder: false,
      hideCursorInOverviewRuler: true,
      contextmenu: false,
      domReadOnly: readonly,
      glyphMargin: !!onLineToggle,
    });

    // Line selection via glyph margin click
    if (onLineToggle) {
      editor.onMouseDown((e: any) => {
        if (
          e.target.type === monaco.editor.MouseTargetType.GUTTER_GLYPH_MARGIN ||
          e.target.type === monaco.editor.MouseTargetType.GUTTER_LINE_NUMBERS
        ) {
          const line = e.target.position?.lineNumber;
          if (line) onLineToggle(line);
        }
      });
    }

    // Track cursor position for co-op broadcast
    if (onCursorMove) {
      cursorPositionDisposable = editor.onDidChangeCursorPosition((e: any) => {
        if (onCursorMove) {
          onCursorMove(e.position.lineNumber, e.position.column);
        }
      });
    }

    updateDecorations();

    resizeObserver = new ResizeObserver(() => {
      editor?.layout();
    });
    resizeObserver.observe(container);
  });

  onDestroy(() => {
    // Clean up remote cursor widgets
    for (const [, widget] of cursorWidgets) {
      try { editor?.removeContentWidget(widget); } catch {}
    }
    cursorWidgets.clear();
    cursorPositionDisposable?.dispose();
    resizeObserver?.disconnect();
    editor?.dispose();
  });

  // Update code when prop changes
  $effect(() => {
    if (editor && code !== editor.getValue()) {
      editor.setValue(code);
    }
  });

  // Update language when prop changes
  $effect(() => {
    if (editor && monaco) {
      const model = editor.getModel();
      if (model) {
        monaco.editor.setModelLanguage(model, languageMap[language] || 'plaintext');
      }
    }
  });

  // Swap the Monaco theme when the app theme changes so the code surface
  // follows light/dark. setTheme is the only reactive lever Monaco exposes.
  $effect(() => {
    const t = $theme;
    if (editor && monaco) {
      monaco.editor.setTheme(t === 'light' ? 'vulnarena-light' : 'vulnarena-dark');
    }
  });

  // Update decorations when selectedLines changes
  $effect(() => {
    if (selectedLines) {
      updateDecorations();
    }
  });

  // Update remote cursors when prop changes
  $effect(() => {
    if (remoteCursors) {
      updateRemoteCursors();
    }
  });

  // Update remote selections when prop changes
  $effect(() => {
    if (remoteSelections) {
      updateRemoteSelections();
    }
  });
</script>

<div class="editor-wrapper" style="height: {height}">
  <div class="editor-container" bind:this={container}></div>
</div>

<style>
  .editor-wrapper {
    position: relative;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    overflow: hidden;
    background: #1e1e1e;
  }

  .editor-container {
    width: 100%;
    height: 100%;
  }

  :global(.vuln-line-highlight) {
    background: color-mix(in srgb, var(--accent-red) 12%, transparent) !important;
    border-left: 3px solid var(--accent-red) !important;
  }

  :global(.vuln-glyph-margin) {
    background: var(--accent-red);
    border-radius: 50%;
    width: 8px !important;
    height: 8px !important;
    margin-top: 6px;
    margin-left: 4px;
  }

  /* Remote cursor widget */
  :global(.remote-cursor-widget) {
    width: 2px;
    height: 18px;
    border-left: 2px solid;
    position: relative;
    pointer-events: none;
    z-index: 10;
  }

  :global(.remote-cursor-label) {
    position: absolute;
    bottom: 100%;
    left: -1px;
    padding: 1px 5px;
    font-family: var(--font-mono), 'JetBrains Mono', monospace;
    font-size: 9px;
    font-weight: 600;
    color: #fff;
    border-radius: 3px 3px 3px 0;
    white-space: nowrap;
    letter-spacing: 0.03em;
    line-height: 1.3;
    opacity: 0.9;
    pointer-events: none;
  }

  /* Remote selection highlight */
  :global(.remote-selection-highlight) {
    background: color-mix(in srgb, var(--accent-blue) 12%, transparent) !important;
    border-left: 2px solid var(--accent-blue) !important;
  }
</style>
