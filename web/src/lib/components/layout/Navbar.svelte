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
    <a href="/" class="logo">VulnArena<span class="logo-dot">.</span></a>

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
      <button
        class="theme-toggle"
        onclick={() => theme.toggle()}
        aria-label="Toggle colour theme"
        title="Toggle colour theme"
      >
        <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
          <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
        </svg>
        <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
          <circle cx="12" cy="12" r="4.2" />
          <path d="M12 2v2.5M12 19.5V22M2 12h2.5M19.5 12H22M4.6 4.6l1.8 1.8M17.6 17.6l1.8 1.8M19.4 4.6l-1.8 1.8M6.4 17.6l-1.8 1.8" />
        </svg>
      </button>

      {#if $isAuthenticated}
        <div class="user-menu">
          <span class="username">{$currentUser?.username}</span>
          <button class="nav-btn" onclick={handleLogout}>Logout</button>
        </div>

        <details class="nav-menu">
          <summary class="menu-trigger" aria-label="Open navigation menu">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
          </summary>
          <div class="menu-pop">
            <a href="/dashboard">Dashboard</a>
            <a href="/arena">Arena</a>
            <a href="/academy">Academy</a>
            <a href="/teams">Squads</a>
            <a href="/leaderboard">Leaderboard</a>
            <a href="/forge" class="nav-link-forge">Forge</a>
            <a href="/settings">Settings</a>
            {#if $currentUser?.role === 'admin'}
              <a href="/admin" class="nav-link-admin">Admin</a>
            {/if}
          </div>
        </details>
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
    background: color-mix(in srgb, var(--bg-primary) 94%, transparent);
    border-bottom: 1px solid var(--border-primary);
    backdrop-filter: saturate(120%) blur(6px);
  }

  .navbar-inner {
    max-width: var(--shell);
    margin: 0 auto;
    padding: 0 var(--space-6);
    height: 56px;
    display: flex;
    align-items: center;
    gap: var(--space-5);
  }

  .logo {
    font-family: var(--font-serif);
    font-weight: 600;
    font-size: 1.15rem;
    font-variant-caps: all-small-caps;
    letter-spacing: 0.08em;
    color: var(--text-primary);
    text-decoration: none;
    white-space: nowrap;
  }

  .logo-dot {
    color: var(--accent-primary);
  }

  .nav-links {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex: 1;
    margin-left: var(--space-3);
  }

  .nav-link {
    font-family: var(--font-sans);
    font-size: var(--fs-micro);
    font-weight: 500;
    color: var(--text-secondary);
    padding: 0.35rem 0.55rem;
    border-radius: 5px;
    transition: color var(--transition-fast);
    text-decoration: none;
  }

  .nav-link:hover {
    color: var(--text-primary);
  }

  .nav-link-forge {
    color: var(--accent-blue);
  }

  .nav-link-forge:hover {
    color: var(--accent-blue-dim);
  }

  .nav-link-admin {
    color: var(--accent-red);
  }

  .nav-link-admin:hover {
    color: var(--accent-red-dim);
  }

  .nav-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-left: auto;
  }

  .theme-toggle {
    display: grid;
    place-items: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    color: var(--text-secondary);
    cursor: pointer;
    transition:
      color var(--transition-fast),
      border-color var(--transition-fast);
  }

  .theme-toggle:hover {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
  }

  .theme-toggle svg {
    width: 16px;
    height: 16px;
  }

  /* Sun/moon swap driven by the global data-theme attribute on <html>. */
  :global([data-theme='dark']) .icon-sun {
    display: none;
  }

  :global([data-theme='light']) .icon-moon {
    display: none;
  }

  .user-menu {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .username {
    font-family: var(--font-mono);
    font-size: var(--fs-micro);
    color: var(--accent-primary);
    letter-spacing: 0.02em;
  }

  .nav-btn {
    font-family: var(--font-sans);
    font-size: var(--fs-micro);
    font-weight: 500;
    padding: var(--space-1) var(--space-3);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    text-decoration: none;
    transition:
      color var(--transition-fast),
      border-color var(--transition-fast),
      background var(--transition-fast);
  }

  .nav-btn:hover {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
  }

  .nav-btn-primary {
    background: var(--accent-primary);
    color: var(--text-inverse);
    border-color: var(--accent-primary);
  }

  .nav-btn-primary:hover {
    background: var(--accent-primary-dim);
    border-color: var(--accent-primary-dim);
    color: var(--text-inverse);
  }

  /* Responsive collapse menu — hidden until the link row stops fitting. */
  .nav-menu {
    display: none;
    position: relative;
  }

  .nav-menu summary {
    list-style: none;
    cursor: pointer;
  }

  .nav-menu summary::-webkit-details-marker {
    display: none;
  }

  .menu-trigger {
    display: grid;
    place-items: center;
    width: 32px;
    height: 32px;
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-input);
    color: var(--text-secondary);
  }

  .menu-trigger svg {
    width: 16px;
    height: 16px;
  }

  .menu-pop {
    position: absolute;
    right: 0;
    top: 42px;
    min-width: 184px;
    display: flex;
    flex-direction: column;
    padding: var(--space-2);
    background: var(--bg-elevated);
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow-lg);
    z-index: var(--z-dropdown);
  }

  .menu-pop a {
    padding: 0.5rem 0.7rem;
    border-radius: 6px;
    font-size: var(--fs-micro);
    color: var(--text-secondary);
    text-decoration: none;
  }

  .menu-pop a:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  @media (max-width: 880px) {
    .nav-links,
    .user-menu .username {
      display: none;
    }

    .nav-menu {
      display: block;
    }
  }
</style>
