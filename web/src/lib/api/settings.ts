import { api } from './client';

export interface ApiKeyInfo {
  hint: string;
  created_at: string;
}

export interface GenerateKeyResponse {
  api_key: string;
  message: string;
}

export async function generateApiKey(): Promise<GenerateKeyResponse> {
  return api<GenerateKeyResponse>('/user/api-key', { method: 'POST' });
}

export async function revokeApiKey(): Promise<{ message: string }> {
  return api('/user/api-key', { method: 'DELETE' });
}

export async function getApiKeyInfo(): Promise<ApiKeyInfo> {
  return api<ApiKeyInfo>('/user/api-key');
}
