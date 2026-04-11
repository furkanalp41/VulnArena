<script lang="ts">
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth';
  import { register } from '$lib/api/auth';
  import { ApiError } from '$lib/api/client';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Card from '$lib/components/ui/Card.svelte';

  let email = $state('');
  let username = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    error = '';

    if (password !== confirmPassword) {
      error = 'Passwords do not match';
      return;
    }

    if (password.length < 8) {
      error = 'Password must be at least 8 characters';
      return;
    }

    if (username.length < 3) {
      error = 'Username must be at least 3 characters';
      return;
    }

    loading = true;

    try {
      const res = await register(email, username, password);
      auth.setAuth(res.user, res.tokens);
      goto('/dashboard');
    } catch (err) {
      if (err instanceof ApiError) {
        error = err.message;
      } else {
        error = 'An unexpected error occurred';
      }
    } finally {
      loading = false;
    }
  }
</script>

<div class="auth-page">
  <div class="auth-container">
    <div class="auth-header">
      <h1 class="auth-title font-mono">
        <span class="bracket">[</span>REGISTER<span class="bracket">]</span>
      </h1>
      <p class="auth-subtitle">Create your operator account</p>
    </div>

    <Card variant="elevated" padding="lg">
      <form onsubmit={handleSubmit} class="auth-form">
        {#if error}
          <div class="error-banner font-mono">{error}</div>
        {/if}

        <Input
          type="email"
          name="email"
          label="Email"
          placeholder="operator@vulnarena.io"
          bind:value={email}
          required
        />

        <Input
          type="text"
          name="username"
          label="Username"
          placeholder="Choose your callsign"
          bind:value={username}
          required
        />

        <Input
          type="password"
          name="password"
          label="Password"
          placeholder="Min. 8 characters"
          bind:value={password}
          required
        />

        <Input
          type="password"
          name="confirmPassword"
          label="Confirm Password"
          placeholder="Re-enter password"
          bind:value={confirmPassword}
          required
        />

        <Button type="submit" variant="primary" size="lg" {loading}>
          {loading ? 'Creating Account...' : 'Create Account'}
        </Button>
      </form>
    </Card>

    <p class="auth-footer">
      Already registered? <a href="/login">Sign in</a>
    </p>
  </div>
</div>

<style>
  .auth-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 200px);
  }

  .auth-container {
    width: 100%;
    max-width: 420px;
  }

  .auth-header {
    text-align: center;
    margin-bottom: var(--space-6);
  }

  .auth-title {
    font-size: 1.5rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: var(--text-primary);
  }

  .bracket {
    color: var(--accent-green);
  }

  .auth-subtitle {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-top: var(--space-2);
  }

  .auth-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .error-banner {
    font-size: 0.8125rem;
    color: var(--accent-red);
    background: var(--accent-red-glow);
    border: 1px solid var(--accent-red);
    border-radius: var(--radius-md);
    padding: var(--space-3) var(--space-4);
  }

  .auth-footer {
    text-align: center;
    margin-top: var(--space-5);
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .auth-footer a {
    color: var(--accent-green);
    font-weight: 500;
  }

  .auth-footer a:hover {
    text-decoration: underline;
  }
</style>
