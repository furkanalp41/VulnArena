import { api } from './client';

export interface CommunityChallenge {
  id: string;
  author_id: string;
  title: string;
  description: string;
  difficulty: number;
  language_slug: string;
  vuln_category_slug: string;
  vulnerable_code: string;
  target_vulnerability: string;
  conceptual_fix: string;
  vulnerable_lines: string;
  hints: string[];
  points: number;
  status: 'pending' | 'approved' | 'rejected' | 'published';
  reviewer_id?: string;
  reviewer_notes?: string;
  challenge_id?: string;
  created_at: string;
  updated_at: string;
  author_username?: string;
}

export interface CommunitySubmitInput {
  title: string;
  description: string;
  difficulty: number;
  language_slug: string;
  vuln_category_slug: string;
  vulnerable_code: string;
  target_vulnerability: string;
  conceptual_fix: string;
  vulnerable_lines: string;
  hints: string[];
  points: number;
}

export function submitCommunityChallenge(input: CommunitySubmitInput): Promise<CommunityChallenge> {
  return api<CommunityChallenge>('/community/challenges', { method: 'POST', body: input });
}

export function listMyCommunitySubmissions(): Promise<{ challenges: CommunityChallenge[]; total: number }> {
  return api<{ challenges: CommunityChallenge[]; total: number }>('/community/challenges');
}

export function getCommunityChallenge(id: string): Promise<CommunityChallenge> {
  return api<CommunityChallenge>(`/community/challenges/${id}`);
}

export function updateCommunityChallenge(id: string, input: CommunitySubmitInput): Promise<CommunityChallenge> {
  return api<CommunityChallenge>(`/community/challenges/${id}`, { method: 'PUT', body: input });
}

export function deleteCommunityChallenge(id: string): Promise<void> {
  return api<void>(`/community/challenges/${id}`, { method: 'DELETE' });
}
