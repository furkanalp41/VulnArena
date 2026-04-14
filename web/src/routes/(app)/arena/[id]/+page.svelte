<script lang="ts">
  import { page } from '$app/stores';
  import { onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { getChallenge, submitAnswer, revealSolution, type Challenge, type UserProgress, type EvaluationFeedback, type RevealResult } from '$lib/api/arena';
  import { ApiError } from '$lib/api/client';
  import CodeEditor from '$lib/components/editor/CodeEditor.svelte';
  import Terminal from '$lib/components/ui/Terminal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import DifficultyBadge from '$lib/components/ui/DifficultyBadge.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import { auth } from '$lib/stores/auth';
  import { connectCollab, disconnectCollab } from '$lib/stores/websocket';
  import {
    remoteCursors as remoteCursorsStore,
    remoteSelections as remoteSelectionsStore,
    roomMembers as roomMembersStore,
    joinAuditRoom,
    leaveAuditRoom,
    broadcastCursor,
    broadcastLineSelect,
    startCollabListener,
    stopCollabListener,
    type RemoteCursor,
    type RemoteSelection,
    type RoomMember,
  } from '$lib/stores/collab';
  import { getMyTeam, type TeamWithMembers } from '$lib/api/teams';

  let challenge = $state<Challenge | null>(null);
  let progress = $state<UserProgress | null>(null);
  let loading = $state(true);
  let error = $state('');

  // Submission state
  let answerText = $state('');
  let targetLines = $state<number[]>([]);
  let lineInput = $state('');
  let submitting = $state(false);
  let submitError = $state('');
  let feedback = $state<EvaluationFeedback | null>(null);
  let terminalLines = $state<string[]>([]);
  let showTerminal = $state(false);
  let startTime = $state(Date.now());

  // Hints state
  let hintsRevealed = $state(0);

  // Reveal solution state
  let showRevealConfirm = $state(false);
  let solutionRevealed = $state(false);
  let solutionData = $state<RevealResult['solution'] | null>(null);
  let revealing = $state(false);

  // Panel state
  let activeTab = $state<'brief' | 'submit' | 'results'>('brief');

  // Co-op state
  let myTeam = $state<TeamWithMembers | null>(null);
  let coopActive = $state(false);
  let coopCursors = $state<Map<string, RemoteCursor>>(new Map());
  let coopSelections = $state<Map<string, RemoteSelection>>(new Map());
  let coopMembers = $state<RoomMember[]>([]);
  let unsubCursors: (() => void) | null = null;
  let unsubSelections: (() => void) | null = null;
  let unsubMembers: (() => void) | null = null;

  const challengeId = $derived($page.params.id ?? '');

  onMount(async () => {
    try {
      const res = await getChallenge(challengeId);
      challenge = res.challenge;
      progress = res.progress;
      startTime = Date.now();
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        error = 'Challenge not found';
      } else {
        error = 'Failed to load challenge';
      }
    } finally {
      loading = false;
    }

    // Set up co-op if user is authenticated and in a team
    const authState = get(auth);
    if (authState.accessToken && authState.user) {
      try {
        myTeam = await getMyTeam();
        if (myTeam && myTeam.team) {
          // Connect to collab WebSocket
          connectCollab(authState.accessToken, authState.user.username, authState.user.display_name);
          startCollabListener();

          // Give the WS a moment to connect, then join the room
          setTimeout(() => {
            joinAuditRoom(challengeId, myTeam!.team.id);
            coopActive = true;
          }, 500);

          // Subscribe to collab stores
          unsubCursors = remoteCursorsStore.subscribe((v) => { coopCursors = v; });
          unsubSelections = remoteSelectionsStore.subscribe((v) => { coopSelections = v; });
          unsubMembers = roomMembersStore.subscribe((v) => { coopMembers = v; });
        }
      } catch {
        // No team or fetch failed — just continue without co-op
      }
    }
  });

  onDestroy(() => {
    if (coopActive) {
      leaveAuditRoom();
      stopCollabListener();
      disconnectCollab();
    }
    unsubCursors?.();
    unsubSelections?.();
    unsubMembers?.();
  });

  function handleLineToggle(line: number) {
    const wasSelected = targetLines.includes(line);
    if (wasSelected) {
      targetLines = targetLines.filter(l => l !== line);
    } else {
      targetLines = [...targetLines, line].sort((a, b) => a - b);
    }
    // Broadcast line selection to squad
    if (coopActive) {
      broadcastLineSelect(line, !wasSelected);
    }
  }

  function handleCursorMove(line: number, column: number) {
    if (coopActive) {
      broadcastCursor(line, column);
    }
  }

  function removeLine(line: number) {
    targetLines = targetLines.filter(l => l !== line);
  }

  function addLineFromInput() {
    // Strip brackets and other non-numeric delimiters, then parse
    const cleaned = lineInput.replace(/[\[\](){}]/g, '');
    const maxLine = challenge?.line_count ?? Infinity;
    const nums = cleaned
      .split(/[,\s]+/)
      .map(s => parseInt(s.trim(), 10))
      .filter(n => !isNaN(n) && n > 0 && n <= maxLine);
    const unique = [...new Set([...targetLines, ...nums])].sort((a, b) => a - b);
    targetLines = unique;
    lineInput = '';
  }

  function handleLineInputKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      addLineFromInput();
    }
  }

  async function handleSubmit() {
    if (!answerText.trim() || answerText.trim().length < 10) {
      submitError = 'Your analysis must be at least 10 characters.';
      return;
    }

    submitting = true;
    submitError = '';
    feedback = null;
    showTerminal = true;
    activeTab = 'results';
    terminalLines = [
      '> Establishing secure connection to SAST engine...',
      `> Transmitting code audit (${targetLines.length} target lines)...`,
    ];

    try {
      const timeSpent = Math.floor((Date.now() - startTime) / 1000);
      const result = await submitAnswer(
        challengeId,
        answerText.trim(),
        targetLines.length > 0 ? targetLines : undefined,
        timeSpent
      );

      feedback = result.feedback;
      progress = result.progress;
      terminalLines = result.feedback.terminal_log;
    } catch (e) {
      if (e instanceof ApiError) {
        submitError = e.message;
        terminalLines = [
          ...terminalLines,
          `> ERROR: ${e.message}`,
          '> Submission rejected.',
        ];
      } else {
        submitError = 'Evaluation failed. Please try again.';
      }
    } finally {
      submitting = false;
    }
  }

  function revealHint() {
    if (challenge && hintsRevealed < challenge.hints.length) {
      hintsRevealed++;
    }
  }

  async function handleRevealSolution() {
    revealing = true;
    try {
      const result = await revealSolution(challengeId);
      solutionData = result.solution;
      solutionRevealed = true;
      progress = result.progress;
      showRevealConfirm = false;
    } catch (e) {
      submitError = e instanceof ApiError ? e.message : 'Failed to reveal solution.';
    } finally {
      revealing = false;
    }
  }

  function clearSelection() {
    targetLines = [];
  }
</script>

{#if loading}
  <div class="loading-screen">
    <span class="loading-text">Loading challenge...</span>
  </div>
{:else if error}
  <div class="error-screen">
    <span class="error-icon">X</span>
    <p>{error}</p>
    <a href="/arena"><Button variant="ghost">Back to Arena</Button></a>
  </div>
{:else if challenge}
  <div class="audit-workspace">
    <!-- Top Bar -->
    <header class="audit-header">
      <div class="header-left">
        <a href="/arena" class="back-link">&larr; Arena</a>
        <h1 class="audit-title">{challenge.title}</h1>
        <div class="audit-meta">
          <DifficultyBadge level={challenge.difficulty} size="sm" />
          <span class="meta-tag">{challenge.language.name}</span>
          <span class="meta-tag">{challenge.vuln_category.name}</span>
          <span class="meta-tag">{challenge.points} pts</span>
          <span class="meta-tag">{challenge.line_count} lines</span>
          {#if challenge.cve_reference}
            <span class="cve-tag">{challenge.cve_reference}</span>
          {/if}
          {#if progress?.status === 'solved'}
            <span class="solved-tag">Solved</span>
          {/if}
        </div>
      </div>
      <div class="header-right">
        {#if coopActive && coopMembers.length > 0}
          <div class="squad-indicator">
            <span class="squad-icon">&gt;&gt;</span>
            Squad linked ({coopMembers.length + 1})
            <div class="squad-avatars">
              {#each coopMembers as member}
                <span class="squad-member" style="border-color: {member.color}" title={member.displayName}>
                  {member.username.slice(0, 2).toUpperCase()}
                </span>
              {/each}
            </div>
          </div>
        {:else if coopActive}
          <div class="squad-indicator solo">
            <span class="squad-icon">&gt;&gt;</span>
            Squad linked (solo)
          </div>
        {/if}
        {#if targetLines.length > 0}
          <div class="line-counter">
            <span class="line-counter-icon">!</span>
            {targetLines.length} line{targetLines.length !== 1 ? 's' : ''} flagged
          </div>
        {/if}
      </div>
    </header>

    <!-- Main Content: Code Editor (left) + Panel (right) -->
    <div class="audit-body">
      <!-- Code Editor - takes majority of space -->
      <div class="code-pane">
        <div class="code-header">
          <span class="code-label">Source Code Audit</span>
          <span class="code-hint">Click line numbers to flag vulnerable lines</span>
        </div>
        <div class="code-editor-area">
          <CodeEditor
            code={challenge.vulnerable_code}
            language={challenge.language.slug}
            readonly={true}
            height="100%"
            selectedLines={targetLines}
            onLineToggle={handleLineToggle}
            remoteCursors={coopActive ? coopCursors : undefined}
            remoteSelections={coopActive ? coopSelections : undefined}
            onCursorMove={coopActive ? handleCursorMove : undefined}
          />
        </div>
      </div>

      <!-- Right Panel - tabbed -->
      <div class="panel-pane">
        <div class="panel-tabs">
          <button
            class="panel-tab"
            class:active={activeTab === 'brief'}
            onclick={() => activeTab = 'brief'}
          >Briefing</button>
          <button
            class="panel-tab"
            class:active={activeTab === 'submit'}
            onclick={() => activeTab = 'submit'}
          >Submit</button>
          <button
            class="panel-tab"
            class:active={activeTab === 'results'}
            onclick={() => activeTab = 'results'}
            disabled={!showTerminal && !feedback}
          >Results</button>
        </div>

        <div class="panel-content">
          <!-- BRIEFING TAB -->
          {#if activeTab === 'brief'}
            <div class="tab-scroll">
              <div class="briefing-section">
                <h3 class="section-label">Mission Briefing</h3>
                <div class="briefing-text">
                  {#each challenge.description.split('\n') as line}
                    {#if line.trim()}
                      <p>{line}</p>
                    {/if}
                  {/each}
                </div>
              </div>

              {#if challenge.cve_reference}
                <div class="cve-section">
                  <h3 class="section-label">CVE Reference</h3>
                  <p class="cve-ref font-mono">{challenge.cve_reference}</p>
                  <p class="cve-note">This challenge is inspired by a real-world vulnerability. Identify the flaw pattern in the source code.</p>
                </div>
              {/if}

              {#if challenge.hints.length > 0}
                <div class="hints-section">
                  <div class="hints-header">
                    <span class="section-label">Hints</span>
                    {#if hintsRevealed < challenge.hints.length}
                      <button class="hint-btn" onclick={revealHint}>
                        Reveal ({hintsRevealed}/{challenge.hints.length})
                      </button>
                    {/if}
                  </div>
                  {#each challenge.hints.slice(0, hintsRevealed) as hint, i}
                    <div class="hint-card">
                      <span class="hint-num">#{i + 1}</span>
                      <p>{hint}</p>
                    </div>
                  {/each}
                </div>
              {/if}

              <!-- Reveal Solution -->
              {#if solutionRevealed && solutionData}
                <div class="solution-section">
                  <h3 class="section-label">Solution (Revealed)</h3>
                  <div class="solution-card">
                    <div class="solution-block">
                      <span class="solution-label">Vulnerability</span>
                      <p>{solutionData.target_vulnerability}</p>
                    </div>
                    <div class="solution-block">
                      <span class="solution-label">Conceptual Fix</span>
                      <p>{solutionData.conceptual_fix}</p>
                    </div>
                    {#if solutionData.vulnerable_lines?.length > 0}
                      <div class="solution-block">
                        <span class="solution-label">Vulnerable Lines</span>
                        <div class="solution-lines font-mono">
                          {solutionData.vulnerable_lines.join(', ')}
                        </div>
                      </div>
                    {/if}
                    <p class="solution-warning">Points for this challenge have been set to 0.</p>
                  </div>
                </div>
              {:else if !solutionRevealed}
                <div class="reveal-section">
                  <Button variant="danger" size="sm" onclick={() => showRevealConfirm = true}>
                    Reveal Solution
                  </Button>
                  <span class="reveal-note">Warning: this sets your score to 0 for this challenge.</span>
                </div>
              {/if}

              <div class="how-to-section">
                <h3 class="section-label">How to Audit</h3>
                <ol class="how-to-list">
                  <li>Review the source code in the editor</li>
                  <li>Click line numbers to flag vulnerable lines</li>
                  <li>Switch to SUBMIT tab</li>
                  <li>Describe the vulnerability and propose a fix</li>
                  <li>Submit for SAST analysis scoring</li>
                </ol>
              </div>
            </div>

          <!-- SUBMIT TAB -->
          {:else if activeTab === 'submit'}
            <div class="tab-scroll">
              <!-- Target Lines Display -->
              <div class="lines-section">
                <h3 class="section-label">
                  Flagged Lines
                  {#if targetLines.length > 0}
                    <button class="clear-btn" onclick={clearSelection}>Clear all</button>
                  {/if}
                </h3>
                {#if targetLines.length > 0}
                  <div class="line-chips">
                    {#each targetLines as line}
                      <button class="line-chip font-mono" onclick={() => removeLine(line)}>
                        L{line} <span class="chip-x">x</span>
                      </button>
                    {/each}
                  </div>
                {:else}
                  <p class="lines-empty">No lines flagged. Click line numbers in the editor or enter them below.</p>
                {/if}

                <div class="line-input-row">
                  <input
                    type="text"
                    class="line-input font-mono"
                    placeholder="Add lines: 42, 105, 230"
                    bind:value={lineInput}
                    onkeydown={handleLineInputKeydown}
                  />
                  <Button variant="ghost" size="sm" onclick={addLineFromInput} disabled={!lineInput.trim()}>
                    ADD
                  </Button>
                </div>
              </div>

              <!-- Analysis Text -->
              <div class="analysis-section">
                <h3 class="section-label">Vulnerability Analysis</h3>
                <p class="analysis-hint">Identify the vulnerability class, explain the attack vector, and propose a concrete fix.</p>
                <textarea
                  class="analysis-input font-mono"
                  placeholder="1. Vulnerability: The code is vulnerable to [type] because...&#10;2. Attack vector: An attacker could exploit this by...&#10;3. Fix: The remediation should include..."
                  bind:value={answerText}
                  rows="12"
                  disabled={submitting}
                ></textarea>
              </div>

              {#if submitError}
                <p class="submit-error">{submitError}</p>
              {/if}

              <div class="submit-actions">
                <Button
                  variant="primary"
                  size="lg"
                  onclick={handleSubmit}
                  loading={submitting}
                  disabled={submitting || !answerText.trim()}
                >
                  {submitting ? 'Analyzing...' : 'Submit Audit'}
                </Button>

                {#if progress}
                  <span class="attempt-info font-mono">
                    Attempts: {progress.attempt_count} | Best: {progress.best_score.toFixed(1)}%
                  </span>
                {/if}
              </div>
            </div>

          <!-- RESULTS TAB -->
          {:else if activeTab === 'results'}
            <div class="tab-scroll">
              {#if feedback}
                <div class="score-section">
                  <Card variant={feedback.passed ? 'default' : 'bordered'}>
                    <div class="score-content" class:passed={feedback.passed}>
                      <div class="score-header">
                        <span class="score-status">
                          {feedback.passed ? 'Audit accepted' : 'Audit insufficient'}
                        </span>
                        <span class="score-value font-mono">{feedback.overall_score.toFixed(1)}%</span>
                      </div>
                      <div class="score-breakdown">
                        <div class="score-row">
                          <span class="score-label">Vulnerability ID</span>
                          <div class="score-bar">
                            <div class="score-fill" style="width: {feedback.vuln_score}%; background: {feedback.vuln_identified ? 'var(--accent-green)' : 'var(--accent-red)'}"></div>
                          </div>
                          <span class="score-num font-mono">{feedback.vuln_score.toFixed(1)}</span>
                        </div>
                        <div class="score-row">
                          <span class="score-label">Remediation</span>
                          <div class="score-bar">
                            <div class="score-fill" style="width: {feedback.fix_score}%; background: {feedback.fix_identified ? 'var(--accent-green)' : 'var(--accent-red)'}"></div>
                          </div>
                          <span class="score-num font-mono">{feedback.fix_score.toFixed(1)}</span>
                        </div>
                        <div class="score-row">
                          <span class="score-label">Line Accuracy</span>
                          <div class="score-bar">
                            <div class="score-fill" style="width: {feedback.line_accuracy}%; background: {feedback.line_accuracy >= 50 ? 'var(--accent-green)' : feedback.line_accuracy > 0 ? 'var(--accent-yellow)' : 'var(--accent-red)'}"></div>
                          </div>
                          <span class="score-num font-mono">{feedback.line_accuracy.toFixed(1)}</span>
                        </div>
                      </div>

                      {#if feedback.matched_vuln_terms && feedback.matched_vuln_terms.length > 0}
                        <div class="matched-terms">
                          <span class="terms-label">Matched concepts:</span>
                          <div class="terms-list">
                            {#each feedback.matched_vuln_terms as term}
                              <span class="term-chip">{term}</span>
                            {/each}
                            {#each feedback.matched_fix_terms ?? [] as term}
                              <span class="term-chip fix">{term}</span>
                            {/each}
                          </div>
                        </div>
                      {/if}
                    </div>
                  </Card>
                </div>
              {/if}

              {#if showTerminal}
                <div class="terminal-area">
                  <Terminal lines={terminalLines} title="SAST ANALYSIS ENGINE" animate={true} />
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>

    <!-- Reveal Confirmation Modal -->
    {#if showRevealConfirm}
      <div class="modal-overlay" onclick={() => showRevealConfirm = false}>
        <div class="modal-box" onclick={(e) => e.stopPropagation()}>
          <h3 class="modal-title">Reveal Solution?</h3>
          <p class="modal-text">
            This will show the full solution including the vulnerability description,
            correct line numbers, and conceptual fix.
          </p>
          <p class="modal-warning">
            Your score for this challenge will be permanently set to 0.
            You will not earn leaderboard points for this challenge.
          </p>
          <div class="modal-actions">
            <Button variant="ghost" size="sm" onclick={() => showRevealConfirm = false} disabled={revealing}>
              Cancel
            </Button>
            <Button variant="danger" size="sm" onclick={handleRevealSolution} loading={revealing} disabled={revealing}>
              {revealing ? 'Revealing...' : 'Confirm Reveal'}
            </Button>
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  /* Loading / Error */
  .loading-screen, .error-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    min-height: 50vh;
  }

  .loading-text {
    color: var(--text-tertiary);
    font-size: 0.875rem;
    letter-spacing: 0.08em;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .error-icon {
    font-size: 2rem;
    color: var(--accent-red);
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px solid var(--accent-red);
    border-radius: 50%;
  }

  /* Workspace layout */
  .audit-workspace {
    display: flex;
    flex-direction: column;
    height: calc(100vh - 80px);
    gap: var(--space-3);
  }

  /* Header */
  .audit-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    flex-shrink: 0;
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-primary);
  }

  .back-link {
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    text-decoration: none;
    letter-spacing: 0.04em;
    transition: color var(--transition-fast);
  }

  .back-link:hover {
    color: var(--accent-green);
  }

  .audit-title {
    font-family: var(--font-serif);
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
    margin-top: 2px;
  }

  .audit-meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-1);
    flex-wrap: wrap;
  }

  .meta-tag {
    font-size: 0.5625rem;
    color: var(--text-tertiary);
    letter-spacing: 0.05em;
    padding: 1px 6px;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
  }

  .cve-tag {
    font-size: 0.5625rem;
    color: var(--accent-yellow);
    letter-spacing: 0.05em;
    padding: 1px 6px;
    border: 1px solid var(--accent-yellow);
    border-radius: var(--radius-sm);
    background: rgba(251, 191, 36, 0.08);
  }

  .solved-tag {
    font-size: 0.5625rem;
    color: var(--accent-green);
    letter-spacing: 0.08em;
    padding: 1px 6px;
    border: 1px solid var(--accent-green);
    border-radius: var(--radius-sm);
    background: var(--accent-green-glow);
  }

  .header-right {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  .squad-indicator {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.6875rem;
    color: var(--accent-blue);
    letter-spacing: 0.02em;
    padding: var(--space-1) var(--space-3);
    border: 1px solid var(--accent-blue-glow);
    border-radius: var(--radius-sm);
    background: var(--accent-blue-glow);
  }

  .squad-indicator.solo {
    opacity: 0.6;
  }

  .squad-icon {
    font-weight: 700;
  }

  .squad-avatars {
    display: flex;
    gap: 4px;
    margin-left: var(--space-1);
  }

  .squad-member {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 2px solid;
    background: var(--bg-tertiary);
    font-size: 0.5rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: 0;
  }

  .line-counter {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.75rem;
    color: var(--accent-red);
    letter-spacing: 0.04em;
    padding: var(--space-1) var(--space-3);
    border: 1px solid var(--accent-red);
    border-radius: var(--radius-sm);
    background: var(--accent-red-glow);
    flex-shrink: 0;
  }

  .line-counter-icon {
    width: 16px;
    height: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--accent-red);
    color: var(--bg-primary);
    border-radius: 50%;
    font-size: 0.625rem;
    font-weight: 700;
  }

  /* Body: code + panel */
  .audit-body {
    display: grid;
    grid-template-columns: 1fr 420px;
    gap: var(--space-3);
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  /* Code pane */
  .code-pane {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .code-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: var(--space-2);
    flex-shrink: 0;
  }

  .code-label {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    color: var(--text-secondary);
  }

  .code-hint {
    font-size: 0.625rem;
    color: var(--text-tertiary);
    letter-spacing: 0.03em;
  }

  .code-editor-area {
    flex: 1;
    min-height: 0;
  }

  /* Panel pane */
  .panel-pane {
    display: flex;
    flex-direction: column;
    min-height: 0;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    background: var(--bg-secondary);
  }

  .panel-tabs {
    display: flex;
    border-bottom: 1px solid var(--border-primary);
    flex-shrink: 0;
  }

  .panel-tab {
    flex: 1;
    padding: var(--space-2) var(--space-3);
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    color: var(--text-tertiary);
    background: none;
    border: none;
    cursor: pointer;
    transition: all var(--transition-fast);
    border-bottom: 2px solid transparent;
  }

  .panel-tab:hover:not(:disabled) {
    color: var(--text-secondary);
    background: var(--bg-tertiary);
  }

  .panel-tab.active {
    color: var(--accent-green);
    border-bottom-color: var(--accent-green);
  }

  .panel-tab:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .panel-content {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .tab-scroll {
    height: 100%;
    overflow-y: auto;
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  /* Briefing */
  .section-label {
    font-family: var(--font-serif);
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: var(--space-2);
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .briefing-text p {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.65;
    margin-bottom: var(--space-2);
  }

  /* CVE section */
  .cve-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .cve-ref {
    font-size: 0.875rem;
    color: var(--accent-yellow);
    letter-spacing: 0.02em;
  }

  .cve-note {
    font-size: 0.75rem;
    color: var(--text-tertiary);
    line-height: 1.5;
  }

  /* How to audit */
  .how-to-list {
    list-style: none;
    counter-reset: steps;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .how-to-list li {
    counter-increment: steps;
    font-size: 0.8125rem;
    color: var(--text-secondary);
    padding-left: var(--space-4);
    position: relative;
    line-height: 1.5;
  }

  .how-to-list li::before {
    content: counter(steps);
    position: absolute;
    left: 0;
    color: var(--accent-green);
    font-family: var(--font-mono);
    font-size: 0.6875rem;
    font-weight: 600;
  }

  /* Hints */
  .hints-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .hints-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .hint-btn {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--accent-yellow);
    background: none;
    border: 1px solid var(--accent-yellow);
    border-radius: var(--radius-sm);
    padding: 1px 6px;
    cursor: pointer;
    letter-spacing: 0.04em;
    transition: all var(--transition-fast);
  }

  .hint-btn:hover {
    background: var(--accent-green-glow);
  }

  .hint-card {
    display: flex;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    border-left: 2px solid var(--accent-yellow);
  }

  .hint-num {
    font-size: 0.625rem;
    color: var(--accent-yellow);
    font-weight: 600;
    flex-shrink: 0;
  }

  .hint-card p {
    font-size: 0.75rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  /* Flagged Lines */
  .lines-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .clear-btn {
    font-family: var(--font-sans);
    font-size: 0.5625rem;
    color: var(--accent-red);
    background: none;
    border: none;
    cursor: pointer;
    letter-spacing: 0.04em;
    margin-left: auto;
  }

  .clear-btn:hover {
    text-decoration: underline;
  }

  .line-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }

  .line-chip {
    display: flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-mono);
    font-size: 0.6875rem;
    padding: 2px 8px;
    background: var(--accent-red-glow);
    border: 1px solid var(--accent-red-glow);
    border-radius: var(--radius-sm);
    color: var(--accent-red);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .line-chip:hover {
    background: rgba(201, 114, 107, 0.2);
    border-color: var(--accent-red);
  }

  .chip-x {
    font-size: 0.5rem;
    opacity: 0.6;
  }

  .lines-empty {
    font-size: 0.75rem;
    color: var(--text-tertiary);
    padding: var(--space-2) 0;
  }

  .line-input-row {
    display: flex;
    gap: var(--space-2);
    align-items: center;
  }

  .line-input {
    flex: 1;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    padding: var(--space-2);
    background: var(--bg-input);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    outline: none;
  }

  .line-input::placeholder {
    color: var(--text-tertiary);
  }

  .line-input:focus {
    border-color: var(--accent-green);
  }

  /* Analysis */
  .analysis-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .analysis-hint {
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    line-height: 1.4;
  }

  .analysis-input {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    padding: var(--space-3);
    background: var(--bg-input);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    resize: vertical;
    outline: none;
    line-height: 1.6;
    width: 100%;
    transition: border-color var(--transition-fast);
  }

  .analysis-input::placeholder {
    color: var(--text-tertiary);
  }

  .analysis-input:focus {
    border-color: var(--accent-green);
    box-shadow: 0 0 0 2px var(--accent-green-glow);
  }

  .analysis-input:disabled {
    opacity: 0.5;
  }

  .submit-error {
    font-size: 0.75rem;
    color: var(--accent-red);
  }

  .submit-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .attempt-info {
    font-size: 0.625rem;
    color: var(--text-tertiary);
    letter-spacing: 0.03em;
  }

  /* Scores */
  .score-section {
    margin: 0;
  }

  .score-content {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .score-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .score-status {
    font-size: 0.6875rem;
    letter-spacing: 0.08em;
    color: var(--accent-red);
    font-weight: 600;
  }

  .score-content.passed .score-status {
    color: var(--accent-green);
  }

  .score-value {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .score-breakdown {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .score-row {
    display: grid;
    grid-template-columns: 110px 1fr 40px;
    gap: var(--space-2);
    align-items: center;
  }

  .score-label {
    font-size: 0.6875rem;
    color: var(--text-secondary);
  }

  .score-bar {
    height: 5px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    overflow: hidden;
  }

  .score-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.8s ease;
  }

  .score-num {
    font-size: 0.6875rem;
    color: var(--text-secondary);
    text-align: right;
  }

  /* Matched terms */
  .matched-terms {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    margin-top: var(--space-1);
  }

  .terms-label {
    font-size: 0.5625rem;
    color: var(--text-tertiary);
    letter-spacing: 0.06em;
  }

  .terms-list {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .term-chip {
    font-size: 0.5625rem;
    padding: 1px 5px;
    border-radius: var(--radius-sm);
    background: var(--accent-green-glow);
    border: 1px solid var(--border-accent);
    color: var(--accent-green);
  }

  .term-chip.fix {
    background: var(--accent-blue-glow);
    border-color: var(--accent-blue-glow);
    color: var(--accent-blue);
  }

  /* Reveal Solution */
  .reveal-section {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-primary);
  }

  .reveal-note {
    font-size: 0.625rem;
    color: var(--text-tertiary);
    line-height: 1.4;
  }

  .solution-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .solution-card {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-3);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    border-left: 3px solid var(--accent-red);
  }

  .solution-block {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .solution-label {
    font-size: 0.625rem;
    font-weight: 600;
    color: var(--accent-red);
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .solution-card p {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.6;
  }

  .solution-lines {
    font-size: 0.8125rem;
    color: var(--accent-yellow);
    letter-spacing: 0.02em;
  }

  .solution-warning {
    font-size: 0.6875rem;
    color: var(--accent-red);
    font-style: italic;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-box {
    background: var(--bg-secondary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    padding: var(--space-6);
    max-width: 420px;
    width: 90%;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .modal-title {
    font-family: var(--font-serif);
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .modal-text {
    font-size: 0.8125rem;
    color: var(--text-secondary);
    line-height: 1.6;
  }

  .modal-warning {
    font-size: 0.8125rem;
    color: var(--accent-red);
    line-height: 1.5;
    padding: var(--space-2) var(--space-3);
    background: var(--accent-red-glow);
    border-radius: var(--radius-sm);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-top: var(--space-2);
  }

  /* Terminal */
  .terminal-area {
    max-height: 350px;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }

  /* Responsive */
  @media (max-width: 1024px) {
    .audit-body {
      grid-template-columns: 1fr;
      overflow-y: auto;
    }

    .code-pane {
      min-height: 400px;
    }

    .panel-pane {
      min-height: 500px;
    }

    .audit-workspace {
      height: auto;
    }
  }
</style>
