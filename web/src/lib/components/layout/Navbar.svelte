<script lang="ts">
  import { auth, isAuthenticated, currentUser } from '$lib/stores/auth';
  import { theme } from '$lib/stores/theme';
  import { logout } from '$lib/api/auth';
  import { goto } from '$app/navigation';

  async function handleLogout() {
    let refreshToken: string | null = null;
    auth.subscribe((s) => (refreshToken = s.refreshToken))();

    if (refreshToken) {
      try {
        await logout(refreshToken);
      } catch {
        // Proceed with local logout anyway
      }
    }
    auth.clear();
    goto('/login');
  }
</script>

<nav class="navbar">
  <div class="navbar-inner">
    <a href="/" class="logo">
      <span class="logo-text">Vuln</span><span class="logo-accent">Arena</span>
    </a>

    <div class="nav-links">
      {#if $isAuthenticated}
        <a href="/dashboard" class="nav-link">Dashboard</a>
        <a href="/arena" class="nav-link">Arena</a>
        <a href="/academy" class="nav-link">Academy</a>
        <a href="/teams" class="nav-link">Squads</a>
        <a href="/leaderboard" class="nav-link">Leaderboard</a>
        <a href="/forge" class="nav-link nav-link-forge">Forge</a>
        <a href="/settings" class="nav-link">Settings</a>
        {#if $currentUser?.role === 'admin'}
          <a href="/admin" class="nav-link nav-link-admin">Admin</a>
        {/if}
      {/if}
    </div>

    <div class="nav-actions">
      <button class="theme-toggle" onclick={() => theme.toggle()} title="Toggle theme">
        <span class="toggle-icon">
          {#if true}
            <!-- Sun/Moon icon via CSS -->
          {/if}
        </span>
        <span class="toggle-label">Theme</span>
      </button>

      {#if $isAuthenticated}
        <div class="user-menu">
          <span class="username">{$currentUser?.username}</span>
          <button class="nav-btn" onclick={handleLogout}>Logout</button>
        </div>
      {:else}
        <a href="/login" class="nav-btn">Login</a>
        <a href="/register" class="nav-btn nav-btn-primary">Register</a>
      {/if}
    </div>
  </div>
</nav>

<style>
  .navbar {
    position: sticky;
    top: 0;
    z-index: var(--z-sticky);
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    backdrop-filter: blur(12px);
  }

  .navbar-inner {
    max-width: 1400px;
    margin: 0 auto;
    padding: 0 var(--space-6);
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .logo {
    font-family: var(--font-serif);
    font-size: 1.25rem;
    font-weight: 600;
    text-decoration: none;
    display: flex;
    align-items: center;
  }

  .logo-text {
    color: var(--text-primary);
  }

  .logo-accent {
    color: var(--accent-green);
  }

  .nav-links {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .nav-link {
    font-family: var(--font-sans);
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
    text-decoration: none;
  }

  .nav-link:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .nav-link-forge {
    color: var(--accent-blue);
  }

  .nav-link-forge:hover {
    color: var(--accent-blue-dim);
    background: var(--accent-blue-glow);
  }

  .nav-link-admin {
    color: var(--accent-red);
  }

  .nav-link-admin:hover {
    color: var(--accent-red-dim);
    background: var(--accent-red-glow);
  }

  .nav-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .theme-toggle {
    background: none;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    cursor: pointer;
    padding: var(--space-1) var(--space-2);
    font-size: 0.6875rem;
    display: flex;
    align-items: center;
    gap: var(--space-1);
    transition: all var(--transition-fast);
  }

  .theme-toggle:hover {
    border-color: var(--border-secondary);
    color: var(--text-primary);
  }

  .toggle-label {
    font-family: var(--font-sans);
  }

  .toggle-icon {
    display: none;
  }

  .user-menu {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .username {
    font-size: 0.8125rem;
    color: var(--accent-green);
  }

  .nav-btn {
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    font-weight: 500;
    padding: var(--space-1) var(--space-3);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    text-decoration: none;
    transition: all var(--transition-fast);
  }

  .nav-btn:hover {
    border-color: var(--accent-green);
    color: var(--accent-green);
  }

  .nav-btn-primary {
    background: var(--accent-green);
    color: var(--text-inverse);
    border-color: var(--accent-green);
  }

  .nav-btn-primary:hover {
    background: var(--accent-green-dim);
    box-shadow: var(--shadow-glow-green);
  }
</style>
