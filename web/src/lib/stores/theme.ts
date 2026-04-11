import { writable } from 'svelte/store';

type Theme = 'dark' | 'light';

function createThemeStore() {
  const { subscribe, set } = writable<Theme>('dark');

  return {
    subscribe,

    initialize() {
      if (typeof window === 'undefined') return;

      const saved = localStorage.getItem('theme') as Theme | null;
      const theme = saved || 'dark';
      set(theme);
      document.documentElement.setAttribute('data-theme', theme);
    },

    toggle() {
      let current: Theme = 'dark';
      subscribe((v) => (current = v))();

      const next: Theme = current === 'dark' ? 'light' : 'dark';
      set(next);
      localStorage.setItem('theme', next);
      document.documentElement.setAttribute('data-theme', next);
    },

    setTheme(theme: Theme) {
      set(theme);
      localStorage.setItem('theme', theme);
      document.documentElement.setAttribute('data-theme', theme);
    },
  };
}

export const theme = createThemeStore();
