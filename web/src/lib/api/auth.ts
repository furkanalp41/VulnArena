import { api } from './client';

interface AuthResponse {
  user: {
    id: string;
    email: string;
    username: string;
    display_name: string;
    avatar_url: string;
    role: string;
    created_at: string;
  };
  tokens: {
    access_token: string;
    refresh_token: string;
  };
}

export function register(email: string, username: string, password: string) {
  return api<AuthResponse>('/auth/register', {
    method: 'POST',
    body: { email, username, password },
  });
}

export function login(email: string, password: string) {
  return api<AuthResponse>('/auth/login', {
    method: 'POST',
    body: { email, password },
  });
}

export function logout(refreshToken: string) {
  return api('/auth/logout', {
    method: 'POST',
    body: { refresh_token: refreshToken },
  });
}
