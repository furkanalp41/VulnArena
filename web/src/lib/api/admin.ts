import { api } from './client';
import type { CommunityChallenge } from './community';

export interface PlatformStats {
  total_users: number;
  total_challenges: number;
  total_submissions: number;
  total_lessons: number;
  total_solves: number;
  active_today: number;
}

export interface CreateChallengeInput {
  title: string;
  description: string;
  difficulty: number;
  language_slug: string;
  vuln_category_slug: string;
  vulnerable_code: string;
  target_vulnerability: string;
  conceptual_fix: string;
  hints: string[];
  points: number;
  is_published: boolean;
}

export interface UpdateChallengeInput extends CreateChallengeInput {}

export interface CreateLessonInput {
  title: string;
  category: string;
  description: string;
  content: string;
  difficulty: number;
  read_time_min: number;
  tags: string[];
  is_published: boolean;
}

export async function getPlatformStats(): Promise<PlatformStats> {
  return api<PlatformStats>('/admin/stats');
}

export async function createChallenge(input: CreateChallengeInput) {
  return api('/admin/challenges', {
    method: 'POST',
    body: input,
  });
}

export async function updateChallenge(id: string, input: UpdateChallengeInput) {
  return api(`/admin/challenges/${id}`, {
    method: 'PUT',
    body: input,
  });
}

export async function createLesson(input: CreateLessonInput) {
  return api('/admin/lessons', {
    method: 'POST',
    body: input,
  });
}

// Community Forge admin endpoints

export async function listCommunityQueue(
  status: string = 'pending',
  page: number = 1
): Promise<{ challenges: CommunityChallenge[]; total: number; page: number }> {
  return api<{ challenges: CommunityChallenge[]; total: number; page: number }>(
    `/admin/community/queue?status=${status}&page=${page}`
  );
}

export async function getCommunitySubmission(id: string): Promise<CommunityChallenge> {
  return api<CommunityChallenge>(`/admin/community/challenges/${id}`);
}

export async function reviewCommunityChallenge(
  id: string,
  action: 'approve' | 'reject',
  notes: string = ''
): Promise<void> {
  await api(`/admin/community/challenges/${id}/review`, {
    method: 'POST',
    body: { action, notes },
  });
}

export async function publishCommunityChallenge(id: string): Promise<void> {
  await api(`/admin/community/challenges/${id}/publish`, {
    method: 'POST',
    body: {},
  });
}
