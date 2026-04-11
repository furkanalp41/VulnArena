import { auth } from '$lib/stores/auth';
import { goto } from '$app/navigation';

const API_BASE = 'http://localhost:8080/api/v1';

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

let currentAccessToken: string | null = null;
let currentRefreshToken: string | null = null;

auth.subscribe((state) => {
  currentAccessToken = state.accessToken;
  currentRefreshToken = state.refreshToken;
});

async function refreshTokens(): Promise<boolean> {
  if (!currentRefreshToken) return false;

  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: currentRefreshToken }),
    });

    if (!res.ok) return false;

    const tokens = await res.json();
    auth.updateTokens(tokens);
    return true;
  } catch {
    return false;
  }
}

export async function api<T = unknown>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const url = `${API_BASE}${endpoint}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (currentAccessToken) {
    headers['Authorization'] = `Bearer ${currentAccessToken}`;
  }

  const config: RequestInit = {
    ...options,
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
  };

  let res = await fetch(url, config);

  // Auto-refresh on 401
  if (res.status === 401 && currentRefreshToken) {
    const refreshed = await refreshTokens();
    if (refreshed) {
      headers['Authorization'] = `Bearer ${currentAccessToken}`;
      config.headers = headers;
      res = await fetch(url, config);
    } else {
      auth.clear();
      goto('/login');
      throw new Error('Session expired');
    }
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Unknown error' }));
    throw new ApiError(res.status, error.error || 'Request failed');
  }

  return res.json();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
