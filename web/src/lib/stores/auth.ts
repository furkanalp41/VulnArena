import { writable, derived } from 'svelte/store';

interface User {
  id: string;
  email: string;
  username: string;
  display_name: string;
  avatar_url: string;
  role: string;
  created_at: string;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  loading: boolean;
}

const initialState: AuthState = {
  user: null,
  accessToken: null,
  refreshToken: null,
  loading: true,
};

function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>(initialState);

  return {
    subscribe,

    initialize() {
      if (typeof window === 'undefined') return;

      const accessToken = localStorage.getItem('access_token');
      const refreshToken = localStorage.getItem('refresh_token');
      const userStr = localStorage.getItem('user');

      if (accessToken && userStr) {
        try {
          const user = JSON.parse(userStr);
          set({ user, accessToken, refreshToken, loading: false });
        } catch {
          this.clear();
        }
      } else {
        update((s) => ({ ...s, loading: false }));
      }
    },

    setAuth(user: User, tokens: { access_token: string; refresh_token: string }) {
      localStorage.setItem('access_token', tokens.access_token);
      localStorage.setItem('refresh_token', tokens.refresh_token);
      localStorage.setItem('user', JSON.stringify(user));

      set({
        user,
        accessToken: tokens.access_token,
        refreshToken: tokens.refresh_token,
        loading: false,
      });
    },

    updateTokens(tokens: { access_token: string; refresh_token: string }) {
      localStorage.setItem('access_token', tokens.access_token);
      localStorage.setItem('refresh_token', tokens.refresh_token);

      update((s) => ({
        ...s,
        accessToken: tokens.access_token,
        refreshToken: tokens.refresh_token,
      }));
    },

    clear() {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('user');
      set({ user: null, accessToken: null, refreshToken: null, loading: false });
    },
  };
}

export const auth = createAuthStore();
export const isAuthenticated = derived(auth, ($auth) => !!$auth.user);
export const currentUser = derived(auth, ($auth) => $auth.user);
