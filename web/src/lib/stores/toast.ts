import { writable } from 'svelte/store';

export interface Toast {
  id: string;
  type: string;
  user: string;
  challenge?: string;
  achievement?: string;
  timestamp: number;
}

export const toasts = writable<Toast[]>([]);

let counter = 0;

export function addToast(event: { type: string; user: string; challenge?: string; achievement?: string }) {
  const id = `toast-${++counter}-${Date.now()}`;
  const toast: Toast = {
    id,
    ...event,
    timestamp: Date.now(),
  };

  toasts.update((prev) => [toast, ...prev].slice(0, 5));

  // Auto-remove after 6 seconds
  setTimeout(() => {
    removeToast(id);
  }, 6000);
}

export function removeToast(id: string) {
  toasts.update((prev) => prev.filter((t) => t.id !== id));
}
