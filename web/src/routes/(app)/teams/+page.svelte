<script lang="ts">
  import { onMount } from 'svelte';
  import { listTeams, createTeam, type Team, type CreateTeamInput } from '$lib/api/teams';
  import Card from '$lib/components/ui/Card.svelte';

  let teams: Team[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  let showForm = $state(false);
  let formName = $state('');
  let formTag = $state('');
  let formDesc = $state('');
  let creating = $state(false);
  let createError = $state('');

  onMount(async () => {
    try {
      teams = await listTeams();
    } catch (e: any) {
      error = e?.message || 'Failed to load teams';
    } finally {
      loading = false;
    }
  });

  async function handleCreate() {
    createError = '';
    creating = true;
    try {
      const input: CreateTeamInput = {
        name: formName,
        tag: formTag.toUpperCase(),
        description: formDesc,
      };
      await createTeam(input);
      // Reload teams list
      teams = await listTeams();
      showForm = false;
      formName = '';
      formTag = '';
      formDesc = '';
    } catch (e: any) {
      createError = e?.message || 'Failed to create team';
    } finally {
      creating = false;
    }
  }
</script>

<svelte:head>
  <title>Squads | VulnArena</title>
</svelte:head>

{#if loading}
  <div class="loading-state">
    <div class="loading-pulse"></div>
    <span class="loading-text">Loading teams...</span>
  </div>
{:else if error}
  <div class="error-state">
    <span>{error}</span>
  </div>
{:else}
  <div class="squads">
    <header class="squads-header">
      <div>
        <h1 class="squads-title">Teams</h1>
        <p class="squads-sub">Form a team and compete together.</p>
      </div>
      <button class="create-btn" onclick={() => (showForm = !showForm)}>
        {showForm ? 'Cancel' : '+ Create team'}
      </button>
    </header>

    {#if showForm}
      <Card variant="elevated" padding="lg">
        <div class="create-form">
          <div class="form-row">
            <label class="form-label">Team name</label>
            <input class="form-input" bind:value={formName} placeholder="Shadow Collective" maxlength="100" />
          </div>
          <div class="form-row">
            <label class="form-label">Tag (2-4 chars)</label>
            <input
              class="form-input tag-input"
              bind:value={formTag}
              placeholder="R00T"
              maxlength="4"
              style="text-transform: uppercase"
            />
          </div>
          <div class="form-row">
            <label class="form-label">Description</label>
            <textarea class="form-input" bind:value={formDesc} placeholder="Team bio..." rows="3"></textarea>
          </div>
          {#if createError}
            <div class="form-error">{createError}</div>
          {/if}
          <button class="submit-btn" onclick={handleCreate} disabled={creating || !formName || !formTag}>
            {creating ? 'Creating...' : 'Create team'}
          </button>
        </div>
      </Card>
    {/if}

    {#if teams.length === 0}
      <Card variant="bordered">
        <div class="empty-state">
          No teams yet. Be the first to form one.
        </div>
      </Card>
    {:else}
      <div class="team-grid">
        {#each teams as team}
          <a href="/teams/{team.tag}" class="team-card">
            <div class="team-tag">{team.tag}</div>
            <div class="team-name">{team.name}</div>
            {#if team.description}
              <div class="team-desc">{team.description}</div>
            {/if}
          </a>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .squads {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .loading-state, .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    min-height: 50vh;
  }

  .loading-pulse {
    width: 40px;
    height: 40px;
    border: 2px solid var(--border-secondary);
    border-radius: 50%;
    animation: pulse 1.2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { transform: scale(0.8); opacity: 0.3; }
    50% { transform: scale(1.1); opacity: 1; }
  }

  .loading-text {
    font-family: var(--font-sans);
    font-size: 0.7rem;
    color: var(--text-tertiary);
  }

  .error-state {
    color: var(--accent-red);
    font-size: 0.85rem;
  }

  .squads-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-4);
  }

  .squads-title {
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .squads-sub {
    font-family: var(--font-sans);
    font-size: 0.6875rem;
    color: var(--text-tertiary);
    margin-top: 4px;
  }

  .create-btn {
    font-family: var(--font-sans);
    font-size: 0.75rem;
    font-weight: 600;
    padding: var(--space-2) var(--space-4);
    background: var(--accent-primary);
    color: var(--text-inverse);
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
    white-space: nowrap;
  }

  .create-btn:hover {
    opacity: 0.9;
  }

  /* Create form */
  .create-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .form-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .form-label {
    font-family: var(--font-sans);
    font-size: 0.65rem;
    color: var(--text-tertiary);
  }

  .form-input {
    font-family: var(--font-sans);
    background: var(--bg-primary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    padding: var(--space-2) var(--space-3);
    color: var(--text-primary);
    font-size: 0.85rem;
    transition: border-color var(--transition-fast);
  }

  .form-input:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  .tag-input {
    max-width: 120px;
  }

  .form-error {
    font-size: 0.75rem;
    color: var(--accent-red);
  }

  .submit-btn {
    align-self: flex-start;
    font-family: var(--font-sans);
    font-size: 0.75rem;
    font-weight: 600;
    padding: var(--space-2) var(--space-5);
    background: var(--accent-primary);
    color: var(--text-inverse);
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .submit-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .submit-btn:not(:disabled):hover {
    opacity: 0.9;
  }

  /* Teams grid */
  .team-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--space-4);
  }

  .team-card {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-4);
    background: var(--bg-secondary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    text-decoration: none;
    color: inherit;
    transition: all 0.2s ease;
  }

  .team-card:hover {
    border-color: var(--border-secondary);
    transform: translateY(-2px);
  }

  .team-tag {
    font-family: var(--font-sans);
    font-size: 0.7rem;
    color: var(--text-secondary);
    font-weight: 700;
  }

  .team-name {
    font-family: var(--font-sans);
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .team-desc {
    font-size: 0.8rem;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .empty-state {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4);
    font-size: 0.85rem;
    color: var(--text-secondary);
  }
</style>
